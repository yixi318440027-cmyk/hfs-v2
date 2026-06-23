package server

import (
	"net/http"

	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/auth"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/config"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/db"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/vfs"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	cfg    *config.Config
	vfs    *vfs.VFS
	db     *db.DB
	auth   *auth.AuthService
	router http.Handler
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
		cfg:  cfg,
		vfs:  vfs.NewVFS(cfg.VFS.Roots),
		db:   database,
		auth: authSvc,
	}
	s.router = s.setupRoutes()
	return s
}

// Handler returns the HTTP handler (router) for the server.
func (s *Server) Handler() http.Handler {
	return s.router
}
