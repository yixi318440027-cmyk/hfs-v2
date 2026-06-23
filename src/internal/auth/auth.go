package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// Claims represents the JWT claims for an authenticated user.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// challengeEntry stores a challenge and its expiration.
type challengeEntry struct {
	challenge string
	expiresAt time.Time
}

// AuthService handles authentication and authorization.
type AuthService struct {
	db        *db.DB
	jwtSecret []byte
	challenges map[string]challengeEntry
	mu        sync.Mutex
}

// NewAuthService creates a new AuthService with the given database and JWT secret.
func NewAuthService(database *db.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:         database,
		jwtSecret:  []byte(jwtSecret),
		challenges: make(map[string]challengeEntry),
	}
}

// HashPassword returns a bcrypt hash of the password (cost=12).
func (a *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword compares a bcrypt hashed password with a plaintext candidate.
func (a *AuthService) VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a JWT token for the given user, valid for 24 hours.
func (a *AuthService) GenerateToken(username, role string) (string, error) {
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "hfs-v2",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateToken parses and validates a JWT token string, returning the claims.
func (a *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// Authenticate verifies username/password against the database and returns a JWT token.
func (a *AuthService) Authenticate(username, password string) (string, error) {
	var passwordHash, role string
	var enabled int

	err := a.db.Conn().QueryRow(
		"SELECT password_hash, role, enabled FROM users WHERE username = ?",
		username,
	).Scan(&passwordHash, &role, &enabled)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid username or password")
		}
		return "", err
	}

	if enabled == 0 {
		return "", errors.New("account is disabled")
	}

	if !a.VerifyPassword(passwordHash, password) {
		return "", errors.New("invalid username or password")
	}

	return a.GenerateToken(username, role)
}

// GenerateChallenge creates a random 32-byte challenge (hex-encoded) that expires in 5 minutes.
// The challenge is stored in memory and consumed on first use.
func (a *AuthService) GenerateChallenge() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	challenge := hex.EncodeToString(b)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Clean expired challenges.
	now := time.Now()
	for k, v := range a.challenges {
		if now.After(v.expiresAt) {
			delete(a.challenges, k)
		}
	}

	a.challenges[challenge] = challengeEntry{
		challenge: challenge,
		expiresAt: now.Add(5 * time.Minute),
	}

	return challenge, nil
}

// ConsumeChallenge validates and consumes a challenge. Returns true if valid and not yet used.
func (a *AuthService) ConsumeChallenge(challenge string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, exists := a.challenges[challenge]
	if !exists {
		return false
	}

	if time.Now().After(entry.expiresAt) {
		delete(a.challenges, challenge)
		return false
	}

	// One-time use — remove after consumption.
	delete(a.challenges, challenge)
	return true
}

// SetupDefaultAdmin ensures the default admin user exists with the configured password.
// It updates the admin password hash if the account exists but has an empty hash.
func (a *AuthService) SetupDefaultAdmin(adminUser, adminPass string) error {
	hash, err := a.HashPassword(adminPass)
	if err != nil {
		return err
	}

	_, err = a.db.Conn().Exec(`
		INSERT INTO users (username, password_hash, role)
		VALUES (?, ?, 'admin')
		ON CONFLICT(username) DO UPDATE SET password_hash = excluded.password_hash
		WHERE users.password_hash = '';
	`, adminUser, hash)

	return err
}
