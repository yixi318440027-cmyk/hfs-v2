package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/auth"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/config"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/db"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/permission"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/vfs"
)

// ConnInfo tracks a single active connection.
type ConnInfo struct {
	IP        string    `json:"ip"`
	Username  string    `json:"username"`
	Path      string    `json:"path"`
	Method    string    `json:"method"`
	Connected time.Time `json:"connected"`
}

// Server wraps the HTTP server and its dependencies.
type Server struct {
	cfg         *config.Config
	vfs         *vfs.VFS
	db          *db.DB
	auth        *auth.AuthService
	perm        *permission.Engine
	router      http.Handler
	connMu      sync.RWMutex
	connections map[string]*ConnInfo // keyed by IP
	accessLogMu sync.Mutex
}

// NewServer creates a new Server with the given configuration.
// It initializes the database, auth service, and default admin account.
func NewServer(cfg *config.Config) *Server {
	// Initialize database.
	database, err := db.Init(cfg.DataDir)
	if err != nil {
		// Panic is acceptable here: the server cannot function without a database.
		panic("failed to initialize database: " + err.Error())
	}

	// Initialize auth service.
	authSvc := auth.NewAuthService(database, cfg.JWTSecret)

	// Set up default admin account.
	if err := authSvc.SetupDefaultAdmin(cfg.AdminUser, cfg.AdminPass); err != nil {
		database.Close()
		panic("failed to setup default admin: " + err.Error())
	}

	s := &Server{
		cfg:         cfg,
		vfs:         vfs.NewVFS(cfg.VFS.Roots),
		db:          database,
		auth:        authSvc,
		perm:        permission.NewEngine(database.Conn()),
		connections: make(map[string]*ConnInfo),
	}
	// Wire database to VFS for metadata queries.
	s.vfs.SetDB(database.Conn())
	s.router = s.setupRoutes()
	return s
}

// Handler returns the HTTP handler (router) for the server.
func (s *Server) Handler() http.Handler {
	// Wrap with connection tracking and access logging middleware.
	return s.connTrackMiddleware(s.accessLogMiddleware(s.router))
}

// connTrackMiddleware records active connections per IP.
func (s *Server) connTrackMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		username, _ := r.Context().Value("username").(string)

		info := &ConnInfo{
			IP:        ip,
			Username:  username,
			Path:      r.URL.Path,
			Method:    r.Method,
			Connected: time.Now(),
		}

		s.connMu.Lock()
		s.connections[ip] = info
		s.connMu.Unlock()

		next.ServeHTTP(w, r)

		s.connMu.Lock()
		delete(s.connections, ip)
		s.connMu.Unlock()
	})
}

// accessLogMiddleware writes access log entries to access.log.
func (s *Server) accessLogMiddleware(next http.Handler) http.Handler {
	logDir := s.cfg.DataDir
	logPath := filepath.Join(logDir, "access.log")

	// Ensure directory exists.
	os.MkdirAll(logDir, 0755)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(start)

		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}

		// Build log line: time method path ip elapsed user-agent
		line := time.Now().Format("2006-01-02 15:04:05") + " " +
			r.Method + " " +
			r.URL.RequestURI() + " " +
			ip + " " +
			elapsed.String() + " " +
			r.UserAgent()

		s.accessLogMu.Lock()
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(line + "\n")
			f.Close()
		} else {
			log.Printf("access log write error: %v", err)
		}
		s.accessLogMu.Unlock()
	})
}
