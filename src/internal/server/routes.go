package server

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/vfs"
	"gopkg.in/yaml.v3"
)

//go:embed webdist/*
var embeddedSPA embed.FS

// spaFS is the sub-filesystem pointing to webdist/ (stripped prefix).
var spaFS fs.FS

func init() {
	var err error
	spaFS, err = fs.Sub(embeddedSPA, "webdist")
	if err != nil {
		panic("failed to create spa sub-filesystem: " + err.Error())
	}
}

func init() {
	// Register WebDAV HTTP methods that chi doesn't support by default.
	chi.RegisterMethod("PROPFIND")
	chi.RegisterMethod("PROPPATCH")
	chi.RegisterMethod("MKCOL")
	chi.RegisterMethod("COPY")
	chi.RegisterMethod("MOVE")
	chi.RegisterMethod("LOCK")
	chi.RegisterMethod("UNLOCK")
}

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

	// Auth API routes (no middleware — public).
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)
	})

	// Public VFS API routes (no auth — only exposes Public=true roots).
	r.Route("/api/public/files", func(r chi.Router) {
		r.Get("/roots", s.handlePublicRoots)
		r.Get("/list", s.handlePublicListDir)
		r.Get("/download", s.handlePublicDownload)
	})

	// VFS API routes (protected by auth middleware).
	r.Route("/api/files", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/", s.handleListDir)
		r.Get("/download", s.handleDownload)
		r.Get("/download-zip", s.handleDownloadZip)
		r.Delete("/", s.handleDelete)
		r.Put("/rename", s.handleRename)
		r.Put("/comment", s.handleUpdateComment)
		r.Post("/mkdir", s.handleMkdir)
		r.Post("/upload", s.handleUpload)
		r.Post("/batch-delete", s.handleBatchDelete)
	})

	// WebDAV routes (protected by auth middleware).
	// Non-standard HTTP methods (PROPFIND etc.) are registered via chi.RegisterMethod in init().
	r.Route("/dav", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.HandleFunc("/*", s.handleWebDAV)
	})

	// Admin API routes (protected by auth + admin middleware).
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.adminMiddleware)
		r.Get("/users", s.handleAdminGetUsers)
		r.Post("/users", s.handleAdminCreateUser)
		r.Delete("/users", s.handleAdminDeleteUser)
		r.Get("/logs", s.handleAdminGetLogs)
		r.Get("/config", s.handleAdminGetConfig)
		r.Put("/config", s.handleAdminUpdateConfig)
		r.Get("/download-counts", s.handleAdminGetDownloadCounts)
		r.Get("/connections", s.handleAdminGetConnections)
		r.Get("/disk-usage", s.handleAdminGetDiskUsage)
	})

	// Serve frontend SPA (static files + fallback to index.html).
	s.serveSPA(r)

	return r
}

// serveSPA mounts the Vue 3 SPA static files (embedded into the binary) and configures the history fallback.
func (s *Server) serveSPA(r chi.Router) {
	// Always use the embedded filesystem — the frontend is compiled into the binary.
	fileSystem := http.FS(spaFS)

	fsHandler := http.FileServer(fileSystem)

	// Serve static assets (JS, CSS, images, fonts) at root level.
	r.Get("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		fsHandler.ServeHTTP(w, r)
	})
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		fsHandler.ServeHTTP(w, r)
	})

	// SPA fallback: all other non-API routes serve index.html.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		f, err := fileSystem.Open("index.html")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"ok":    "false",
				"error": "not found",
			})
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, f)
	})
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
	vfsPath := r.URL.Query().Get("path")
	if vfsPath == "" {
		jsonError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	localPath, _, err := s.vfs.GetFilePath(vfsPath)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	// Increment download count.
	s.vfs.IncrementDownload(vfsPath)

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

	// Use path.Join for forward-slash paths in API responses (cross-platform consistency)
	jsonOK(w, map[string]string{"created": path.Join(req.Path, req.DirName)})
}

// UploadedFile represents a successfully uploaded file in the response.
type UploadedFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// UploadError represents a failed upload in the response.
type UploadError struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// handleUpload handles POST /api/files/upload (multipart/form-data).
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 32MB.
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	dirPath := r.FormValue("path")
	if dirPath == "" {
		dirPath = "/Files"
	}

	var uploaded []UploadedFile
	var uploadErrors []UploadError

	// Handle multiple files under the "files" field.
	fhs := r.MultipartForm.File["files"]
	for _, fh := range fhs {
		// Sanitize filename to prevent path traversal.
		safeName := filepath.Base(fh.Filename)

		file, err := fh.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, UploadError{Name: fh.Filename, Error: err.Error()})
			continue
		}
		defer file.Close()

		vfsPath := path.Join(dirPath, safeName)
		if err := s.vfs.UploadFile(vfsPath, file); err != nil {
			uploadErrors = append(uploadErrors, UploadError{Name: fh.Filename, Error: err.Error()})
			continue
		}

		// Record upload metadata.
		username, _ := r.Context().Value("username").(string)
		s.vfs.SetFileMeta(vfsPath, username)

		uploaded = append(uploaded, UploadedFile{
			Name: safeName,
			Size: fh.Size,
			Path: vfsPath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"uploaded": uploaded,
			"errors":   uploadErrors,
		},
	})
}

// handleBatchDelete handles POST /api/files/batch-delete with JSON body {"paths": [...]}.
func (s *Server) handleBatchDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Paths) == 0 {
		jsonError(w, http.StatusBadRequest, "paths must be a non-empty array")
		return
	}

	type DeleteResult struct {
		Path  string `json:"path"`
		Error string `json:"error,omitempty"`
	}

	var results []DeleteResult
	for _, p := range req.Paths {
		if err := s.vfs.DeletePath(p); err != nil {
			results = append(results, DeleteResult{Path: p, Error: err.Error()})
		} else {
			results = append(results, DeleteResult{Path: p})
		}
	}

	jsonOK(w, results)
}

// handleDownloadZip handles GET /api/files/download-zip?paths=...&paths=...
func (s *Server) handleDownloadZip(w http.ResponseWriter, r *http.Request) {
	paths := r.URL.Query()["paths"]
	if len(paths) == 0 {
		jsonError(w, http.StatusBadRequest, "at least one paths parameter is required")
		return
	}

	// Build zip in memory.
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, vfsPath := range paths {
		localPath, _, err := s.vfs.GetFilePath(vfsPath)
		if err != nil {
			continue // skip paths that cannot be resolved
		}

		info, err := os.Stat(localPath)
		if err != nil || info.IsDir() {
			continue // skip directories and inaccessible files
		}

		entryName := filepath.Base(localPath)

		// Check for duplicate entry names.
		// zip.Writer doesn't error on duplicate, but we avoid by appending suffix.
		fw, err := zipWriter.Create(entryName)
		if err != nil {
			continue
		}

		file, err := os.Open(localPath)
		if err != nil {
			continue
		}
		io.Copy(fw, file)
		file.Close()
	}

	if err := zipWriter.Close(); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create zip")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"download.zip\"")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}

// handleAdminGetUsers handles GET /api/admin/users.
func (s *Server) handleAdminGetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Conn().Query("SELECT id, username, role, enabled, created_at FROM users")
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query users")
		return
	}
	defer rows.Close()

	type UserInfo struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		Enabled   int    `json:"enabled"`
		CreatedAt string `json:"created_at"`
	}

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Enabled, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	if users == nil {
		users = []UserInfo{}
	}

	jsonOK(w, users)
}

// handleAdminCreateUser handles POST /api/admin/users.
func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		jsonError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	hash, err := s.auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	_, err = s.db.Conn().Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		req.Username, hash, req.Role,
	)
	if err != nil {
		jsonError(w, http.StatusConflict, "username already exists")
		return
	}

	jsonOK(w, map[string]string{"created": req.Username})
}

// handleAdminDeleteUser handles DELETE /api/admin/users?id=123.
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		jsonError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Get the username of the user to delete.
	var targetUsername string
	err = s.db.Conn().QueryRow("SELECT username FROM users WHERE id = ?", id).Scan(&targetUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to query user")
		return
	}

	// Cannot delete self.
	currentUsername, _ := r.Context().Value("username").(string)
	if targetUsername == currentUsername {
		jsonError(w, http.StatusForbidden, "cannot delete yourself")
		return
	}

	// Cannot delete the last admin.
	var adminCount int
	s.db.Conn().QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' AND enabled = 1").Scan(&adminCount)
	if adminCount <= 1 {
		// Check if the target is the last admin.
		var targetRole string
		s.db.Conn().QueryRow("SELECT role FROM users WHERE id = ?", id).Scan(&targetRole)
		if targetRole == "admin" {
			jsonError(w, http.StatusForbidden, "cannot delete the last admin user")
			return
		}
	}

	_, err = s.db.Conn().Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	jsonOK(w, map[string]string{"deleted": targetUsername})
}

// handleAdminGetLogs handles GET /api/admin/logs.
func (s *Server) handleAdminGetLogs(w http.ResponseWriter, r *http.Request) {
	logPath := filepath.Join(s.cfg.DataDir, "access.log")

	// Read last N lines (max 500).
	limitStr := r.URL.Query().Get("limit")
	limit := 200
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		jsonOK(w, []string{})
		return
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Reverse so newest first.
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	if len(lines) > limit {
		lines = lines[:limit]
	}
	if lines == nil {
		lines = []string{}
	}

	jsonOK(w, lines)
}

// handleUpdateComment handles PUT /api/files/comment with JSON body {path, comment}.
func (s *Server) handleUpdateComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Comment string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		jsonError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := s.vfs.UpdateFileComment(req.Path, req.Comment); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonOK(w, map[string]string{"path": req.Path, "comment": req.Comment})
}

// handleAdminGetDownloadCounts handles GET /api/admin/download-counts.
func (s *Server) handleAdminGetDownloadCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := s.vfs.GetDownloadCounts()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, counts)
}

// handleAdminGetConnections handles GET /api/admin/connections.
func (s *Server) handleAdminGetConnections(w http.ResponseWriter, r *http.Request) {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	var list []*ConnInfo
	for _, c := range s.connections {
		list = append(list, c)
	}
	if list == nil {
		list = []*ConnInfo{}
	}
	jsonOK(w, list)
}

// handleAdminGetDiskUsage handles GET /api/admin/disk-usage.
func (s *Server) handleAdminGetDiskUsage(w http.ResponseWriter, r *http.Request) {
	disks := s.GetDisksForVFS()
	jsonOK(w, disks)
}

// handleAdminGetConfig handles GET /api/admin/config.
func (s *Server) handleAdminGetConfig(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, s.cfg)
}

// handleAdminUpdateConfig handles PUT /api/admin/config.
func (s *Server) handleAdminUpdateConfig(w http.ResponseWriter, r *http.Request) {
	// We decode into a temporary Config to validate YAML, then marshal back.
	var newCfg map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Update config fields that are present in the request.
	if v, ok := newCfg["port"]; ok {
		if str, ok := v.(string); ok {
			s.cfg.Port = str
		}
	}
	if v, ok := newCfg["data_dir"]; ok {
		if str, ok := v.(string); ok {
			s.cfg.DataDir = str
		}
	}
	if v, ok := newCfg["vfs"]; ok {
		// Re-marshal the vfs portion to apply it.
		data, _ := json.Marshal(v)
		json.Unmarshal(data, &s.cfg.VFS)
	}

	// Save config to YAML file.
	configPath := "config.yaml"
	if s.cfg.DataDir != "" {
		configPath = filepath.Join(s.cfg.DataDir, "config.yaml")
	}

	yamlData, err := yaml.Marshal(s.cfg)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to marshal config")
		return
	}

	if err := os.MkdirAll(s.cfg.DataDir, 0755); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create data directory")
		return
	}

	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to write config file")
		return
	}

	jsonOK(w, s.cfg)
}

// --- Public VFS handlers (no auth) ---

// handlePublicRoots handles GET /api/public/files/roots — returns only Public VFS roots.
func (s *Server) handlePublicRoots(w http.ResponseWriter, r *http.Request) {
	roots := s.vfs.PublicRoots()
	// Never return local filesystem paths to clients.
	type safeRoot struct {
		Name string `json:"name"`
	}
	out := make([]safeRoot, len(roots))
	for i, rt := range roots {
		out[i] = safeRoot{Name: rt.Name}
	}
	jsonOK(w, out)
}

// handlePublicListDir handles GET /api/public/files/list?path=/Files/subdir
func (s *Server) handlePublicListDir(w http.ResponseWriter, r *http.Request) {
	vfsPath := r.URL.Query().Get("path")
	if vfsPath == "" {
		// Default to first public root
		roots := s.vfs.PublicRoots()
		if len(roots) == 0 {
			jsonOK(w, &vfs.DirEntry{Path: "/", Files: []vfs.FileInfo{}, Total: 0})
			return
		}
		vfsPath = "/" + roots[0].Name
	}

	entry, err := s.vfs.ListDirPublic(vfsPath)
	if err != nil {
		jsonError(w, http.StatusForbidden, err.Error())
		return
	}
	jsonOK(w, entry)
}

// handlePublicDownload handles GET /api/public/files/download?path=/Files/report.pdf
func (s *Server) handlePublicDownload(w http.ResponseWriter, r *http.Request) {
	vfsPath := r.URL.Query().Get("path")
	if vfsPath == "" {
		jsonError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	localPath, root, err := s.vfs.GetFilePath(vfsPath)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	if !root.Public {
		jsonError(w, http.StatusForbidden, "access denied")
		return
	}

	// Increment download count.
	s.vfs.IncrementDownload(vfsPath)

	filename := filepath.Base(localPath)
	mimeType := detectMIME(filename)

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, localPath)
}

// detectMIME returns the MIME type for a file based on its extension.
func detectMIME(name string) string {
	m := mime.TypeByExtension(filepath.Ext(name))
	if m == "" {
		return "application/octet-stream"
	}
	return m
}
