package vfs

import (
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
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
	MIME    string    `json:"mime,omitempty"`
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
}

// NewVFS creates a new VFS instance with the given roots.
func NewVFS(roots []config.VFSRoot) *VFS {
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

// ListDir lists the contents of a directory at the given VFS path.
func (v *VFS) ListDir(vfsPath string) (*DirEntry, error) {
	localPath, root, err := v.GetFilePath(vfsPath)
	if err != nil {
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

	// Ensure parent is a directory
	info, err := os.Stat(parentPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", vfsPath)
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

// splitPath splits a VFS path into segments, removing empty parts.
func splitPath(p string) []string {
	p = filepath.ToSlash(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// detectMIME returns the MIME type for a file based on its extension.
func detectMIME(name string) string {
	m := mime.TypeByExtension(filepath.Ext(name))
	if m == "" {
		return "application/octet-stream"
	}
	return m
}
