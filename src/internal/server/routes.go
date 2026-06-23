package server

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// jsonOK writes a successful JSON response.
func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"data": data,
	})
}

// jsonError writes an error JSON response with the given status code.
func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"ok":    "false",
		"error": msg,
	})
}

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

	// Auth API routes (no middleware — public).
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)
	})

	// VFS API routes (protected by auth middleware).
	r.Route("/api/files", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/", s.handleListDir)
		r.Get("/download", s.handleDownload)
		r.Delete("/", s.handleDelete)
		r.Put("/rename", s.handleRename)
		r.Post("/mkdir", s.handleMkdir)
	})

	return r
}

// handleLogin handles POST /api/auth/login.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		jsonError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	token, err := s.auth.Authenticate(req.Username, req.Password)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Fetch user info for the response.
	var role string
	err = s.db.Conn().QueryRow("SELECT role FROM users WHERE username = ?", req.Username).Scan(&role)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to retrieve user info")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":    true,
		"token": token,
		"user": map[string]string{
			"username": req.Username,
			"role":     role,
		},
	})
}

// handleLogout handles POST /api/auth/logout.
// In this stateless JWT implementation, logout is handled client-side by discarding the token.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"message": "logged out",
	})
}

// handleListDir handles GET /api/files?path=/Files/subdir
func (s *Server) handleListDir(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/Files"
	}

	entry, err := s.vfs.ListDir(path)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	jsonOK(w, entry)
}

// handleDownload handles GET /api/files/download?path=/Files/report.pdf
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	localPath, _, err := s.vfs.GetFilePath(path)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	filename := filepath.Base(localPath)
	mimeType := detectMIME(filename)

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, localPath)
}

// handleDelete handles DELETE /api/files?path=/Files/old.txt
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	if err := s.vfs.DeletePath(path); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonOK(w, map[string]string{"deleted": path})
}

// handleRename handles PUT /api/files/rename with JSON body {path, newName}
func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		NewName string `json:"newName"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Path == "" || req.NewName == "" {
		jsonError(w, http.StatusBadRequest, "path and newName are required")
		return
	}

	if err := s.vfs.RenamePath(req.Path, req.NewName); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonOK(w, map[string]string{"renamed": req.Path, "to": req.NewName})
}

// handleMkdir handles POST /api/files/mkdir with JSON body {path, dirName}
func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		DirName string `json:"dirName"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Path == "" || req.DirName == "" {
		jsonError(w, http.StatusBadRequest, "path and dirName are required")
		return
	}

	if err := s.vfs.CreateDir(req.Path, req.DirName); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonOK(w, map[string]string{"created": filepath.Join(req.Path, req.DirName)})
}

// detectMIME returns the MIME type for a file based on its extension.
func detectMIME(name string) string {
	m := mime.TypeByExtension(filepath.Ext(name))
	if m == "" {
		return "application/octet-stream"
	}
	return m
}
