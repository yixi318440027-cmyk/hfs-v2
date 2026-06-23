package vfs

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yixi318440027-cmyk/hfs-v2/src/internal/config"
)

// FileInfo represents a file or directory entry in the VFS.
type FileInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	SizeHuman  string    `json:"sizeHuman,omitempty"`
	ModTime    time.Time `json:"modTime"`
	IsDir      bool      `json:"isDir"`
	MIME       string    `json:"mime,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	UploadedBy string    `json:"uploadedBy,omitempty"`
	CreatedAt  string    `json:"createdAt,omitempty"`
	Downloads  int64     `json:"downloads,omitempty"`
}

// DirEntry represents the contents of a directory in the VFS.
type DirEntry struct {
	Path  string     `json:"path"`
	Name  string     `json:"name"`
	Files []FileInfo `json:"files"`
	Total int        `json:"total"`
}

// VFS represents the virtual file system.
type VFS struct {
	roots []config.VFSRoot
	db    *sql.DB
}

// SetDB sets the database connection for file metadata queries.
func (v *VFS) SetDB(db *sql.DB) {
	v.db = db
}

// PublicRoots returns all VFSRoot entries that are marked as public.
func (v *VFS) PublicRoots() []config.VFSRoot {
	var out []config.VFSRoot
	for _, r := range v.roots {
		if r.Public {
			out = append(out, r)
		}
	}
	return out
}

// NewVFS creates a new VFS instance with the given roots.
// It ensures all root directories exist on disk.
func NewVFS(roots []config.VFSRoot) *VFS {
	for _, root := range roots {
		if err := os.MkdirAll(root.Path, 0755); err != nil {
			// Log a warning but don't fail — the directory might be created later.
			fmt.Fprintf(os.Stderr, "vfs: failed to create root %q: %v\n", root.Path, err)
		}
	}
	return &VFS{roots: roots}
}

// ResolveRoot finds a VFSRoot by its display name.
func (v *VFS) ResolveRoot(name string) (*config.VFSRoot, error) {
	for i := range v.roots {
		if v.roots[i].Name == name {
			return &v.roots[i], nil
		}
	}
	return nil, fmt.Errorf("root %q not found", name)
}

// GetFilePath converts a VFS path (e.g. "/Files/subdir/file.txt") to a local
// filesystem path. It returns the local path, the matching VFSRoot, and any error.
func (v *VFS) GetFilePath(vfsPath string) (string, *config.VFSRoot, error) {
	parts := splitPath(vfsPath)
	if len(parts) == 0 {
		return "", nil, errors.New("invalid vfs path: empty")
	}

	root, err := v.ResolveRoot(parts[0])
	if err != nil {
		return "", nil, err
	}

	relPath := filepath.Join(parts[1:]...)
	clean := filepath.Clean(relPath)

	// Prevent path traversal
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", nil, errors.New("path traversal detected")
	}

	localPath := filepath.Join(root.Path, clean)

	// Verify the resolved path is still under root.Path
	absLocal, err := filepath.Abs(localPath)
	if err != nil {
		return "", nil, err
	}
	absRoot, err := filepath.Abs(root.Path)
	if err != nil {
		return "", nil, err
	}
	if !strings.HasPrefix(absLocal, absRoot+string(filepath.Separator)) && absLocal != absRoot {
		return "", nil, errors.New("path traversal detected")
	}

	return localPath, root, nil
}

// enrichFileMeta fills in Comment, UploadedBy, CreatedAt, Downloads, and SizeHuman
// from the database for each file in the list.
func (v *VFS) enrichFileMeta(vfsPath string, files []FileInfo) {
	if v.db == nil {
		// No DB set; still fill SizeHuman
		for i := range files {
			files[i].SizeHuman = FormatSize(files[i].Size)
		}
		return
	}

	// Batch load download counts for all files in this directory.
	for i := range files {
		fp := vfsPath + "/" + files[i].Name
		var count int64
		v.db.QueryRow("SELECT count FROM download_counts WHERE vfs_path = ?", fp).Scan(&count)
		files[i].Downloads = count
		files[i].SizeHuman = FormatSize(files[i].Size)

		// Load metadata.
		var comment, uploadedBy, createdAt string
		err := v.db.QueryRow("SELECT comment, uploaded_by, created_at FROM files_meta WHERE vfs_path = ?", fp).Scan(&comment, &uploadedBy, &createdAt)
		if err == nil {
			files[i].Comment = comment
			files[i].UploadedBy = uploadedBy
			files[i].CreatedAt = createdAt
		}
	}
}

// ListDir lists the contents of a directory at the given VFS path.
func (v *VFS) ListDir(vfsPath string) (*DirEntry, error) {
	localPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
		return nil, err
	}

	// Ensure the directory exists (create if it's a VFS root).
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return nil, err
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", vfsPath)
	}

	entries, err := os.ReadDir(localPath)
	if err != nil {
		return nil, err
	}

	files := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			continue
		}
		f := FileInfo{
			Name:    e.Name(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
			IsDir:   e.IsDir(),
		}
		if !e.IsDir() {
			f.MIME = detectMIME(e.Name())
		}
		files = append(files, f)
	}

	// Sort: directories first, then by name alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Build the display path relative to the root
	relDisplay := strings.TrimPrefix(vfsPath, "/"+root.Name)
	if relDisplay == "" {
		relDisplay = "/"
	}

	// Enrich with metadata from database.
	v.enrichFileMeta(vfsPath, files)

	return &DirEntry{
		Path:  relDisplay,
		Name:  root.Name,
		Files: files,
		Total: len(files),
	}, nil
}

// TreePath represents a single directory path entry for the path picker.
type TreePath struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	IsRoot bool   `json:"isRoot"`
}

// CollectTreePaths recursively walks all VFS roots and returns every directory
// path that exists on disk. This is used by the permissions page to let admins
// pick a path from the real file tree instead of typing it manually.
func (v *VFS) CollectTreePaths() []TreePath {
	var result []TreePath

	for _, root := range v.roots {
		rootVfsPath := "/" + root.Name
		result = append(result, TreePath{Path: rootVfsPath, Name: root.Name, IsRoot: true})

		// Walk the root directory recursively.
		_ = filepath.Walk(root.Path, func(localPath string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() || localPath == root.Path {
				return nil
			}

			// Compute the VFS path from the local path.
			rel, err := filepath.Rel(root.Path, localPath)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			vfsPath := rootVfsPath + "/" + rel
			result = append(result, TreePath{Path: vfsPath, Name: info.Name(), IsRoot: false})

			return nil
		})
	}

	return result
}

// ListDirPublic lists a directory under a Public VFS root.
// Returns an error if the root is not public.
func (v *VFS) ListDirPublic(vfsPath string) (*DirEntry, error) {
	localPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
		return nil, err
	}
	if !root.Public {
		return nil, errors.New("access denied: root is not public")
	}
	// Reuse ListDir logic but pass through the already-resolved info.
	// Ensure the directory exists.
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return nil, err
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", vfsPath)
	}
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return nil, err
	}
	files := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			continue
		}
		f := FileInfo{
			Name:    e.Name(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
			IsDir:   e.IsDir(),
		}
		if !e.IsDir() {
			f.MIME = detectMIME(e.Name())
		}
		files = append(files, f)
	}
	// Sort: directories first, then by name alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	relDisplay := strings.TrimPrefix(vfsPath, "/"+root.Name)
	if relDisplay == "" {
		relDisplay = "/"
	}

	// Enrich with metadata from database (public-only: no uploadedBy).
	v.enrichFileMeta(vfsPath, files)
	// Strip uploadedBy for public view.
	for i := range files {
		files[i].UploadedBy = ""
	}

	return &DirEntry{
		Path:  relDisplay,
		Name:  root.Name,
		Files: files,
		Total: len(files),
	}, nil
}

// DeletePath deletes a file or empty directory at the given VFS path.
func (v *VFS) DeletePath(vfsPath string) error {
	localPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
		return err
	}
	if root.ReadOnly {
		return errors.New("root is read-only")
	}
	return os.Remove(localPath)
}

// RenamePath renames a file or directory at the given VFS path.
func (v *VFS) RenamePath(vfsPath, newName string) error {
	localPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
		return err
	}
	if root.ReadOnly {
		return errors.New("root is read-only")
	}

	dir := filepath.Dir(localPath)
	newPath := filepath.Join(dir, newName)

	// Validate the new path is still under root
	absNew, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(root.Path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absNew, absRoot+string(filepath.Separator)) && absNew != absRoot {
		return errors.New("path traversal detected")
	}

	return os.Rename(localPath, newPath)
}

// CreateDir creates a new directory at the given VFS path.
func (v *VFS) CreateDir(vfsPath, dirName string) error {
	parentPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
		return err
	}
	if root.ReadOnly {
		return errors.New("root is read-only")
	}

	// Ensure parent exists.
	if err := os.MkdirAll(parentPath, 0755); err != nil {
		return err
	}

	newPath := filepath.Join(parentPath, dirName)
	return os.Mkdir(newPath, 0755)
}

// UploadFile writes the contents of r to the given VFS path.
// vfsPath is the full VFS path including filename, e.g. "/Files/subdir/new.txt".
// Parent directories are created automatically if they do not exist.
func (v *VFS) UploadFile(vfsPath string, reader io.Reader) error {
	localPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
		return err
	}

	if root.ReadOnly {
		return errors.New("root is read-only")
	}

	// Ensure parent directory exists.
	parentDir := filepath.Dir(localPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	// Create the file.
	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy contents.
	if _, err := io.Copy(file, reader); err != nil {
		return err
	}

	return nil
}

// FormatSize returns a human-readable file size string (1024-based).
func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	size := float64(bytes)
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i > 0 {
		return fmt.Sprintf("%.1f %s", size, units[i])
	}
	return fmt.Sprintf("%.0f %s", size, units[i])
}

// splitPath splits a VFS path into segments, removing empty parts.
func splitPath(p string) []string {
	p = filepath.ToSlash(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// IncrementDownload increments the download count for a VFS path.
func (v *VFS) IncrementDownload(vfsPath string) {
	if v.db == nil {
		return
	}
	v.db.Exec(`
		INSERT INTO download_counts (vfs_path, count, last_download_at) VALUES (?, 1, datetime('now'))
		ON CONFLICT(vfs_path) DO UPDATE SET count = count + 1, last_download_at = datetime('now')
	`, vfsPath)
}

// SetFileMeta sets metadata for a file at the given VFS path.
func (v *VFS) SetFileMeta(vfsPath, uploadedBy string) {
	if v.db == nil {
		return
	}
	v.db.Exec(`
		INSERT INTO files_meta (vfs_path, uploaded_by, created_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(vfs_path) DO UPDATE SET uploaded_by = excluded.uploaded_by
	`, vfsPath, uploadedBy)
}

// UpdateFileComment updates the comment for a VFS path.
func (v *VFS) UpdateFileComment(vfsPath, comment string) error {
	if v.db == nil {
		return errors.New("database not available")
	}
	_, err := v.db.Exec(`
		INSERT INTO files_meta (vfs_path, comment) VALUES (?, ?)
		ON CONFLICT(vfs_path) DO UPDATE SET comment = excluded.comment
	`, vfsPath, comment)
	return err
}

// GetDownloadCounts returns all download counts (admin use).
func (v *VFS) GetDownloadCounts() ([]map[string]interface{}, error) {
	if v.db == nil {
		return nil, errors.New("database not available")
	}
	rows, err := v.db.Query("SELECT vfs_path, count, last_download_at FROM download_counts ORDER BY count DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var path string
		var count int64
		var lastAt string
		if err := rows.Scan(&path, &count, &lastAt); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"path":            path,
			"count":           count,
			"lastDownloadAt":  lastAt,
		})
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	return results, nil
}

// detectMIME returns the MIME type for a file based on its extension.
func detectMIME(name string) string {
	m := mime.TypeByExtension(filepath.Ext(name))
	if m == "" {
		return "application/octet-stream"
	}
	return m
}
