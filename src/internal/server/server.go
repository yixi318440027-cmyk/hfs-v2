package server

import (
	"net/http"

	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/config"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	cfg    *config.Config
	router http.Handler
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfg *config.Config) *Server {
	s := &Server{cfg: cfg}
	s.router = s.setupRoutes()
	return s
}

// Handler returns the HTTP handler (router) for the server.
func (s *Server) Handler() http.Handler {
	return s.router
}
