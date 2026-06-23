package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// setupRoutes configures the chi router with all routes.
func (s *Server) setupRoutes() http.Handler {
	r := chi.NewRouter()

	// Health check / version endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": "0.1.0-dev",
			"message": "Hello, hfs-v2",
		})
	})

	return r
}
