package server

import (
	"os"
	"path/filepath"
)

// DiskUsageInfo holds disk space information for a path.
type DiskUsageInfo struct {
	MountPoint string `json:"mountPoint"`
	Total      uint64 `json:"total"`
	Free       uint64 `json:"free"`
	Used       uint64 `json:"used"`
	Label      string `json:"label"`
}

// getDiskUsage returns disk space info for the given directory path.
func getDiskUsage(dirPath string) (uint64, uint64, error) {
	// Walk upward from dirPath to find the root of the filesystem.
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return 0, 0, err
	}

	// Use os.Stat and a platform-specific helper.
	return diskUsage(absPath)
}

// GetDisksForVFS returns disk usage for all unique VFS root mount points.
func (s *Server) GetDisksForVFS() []DiskUsageInfo {
	var disks []DiskUsageInfo
	seen := make(map[string]bool)

	for _, root := range s.cfg.VFS.Roots {
		absPath, err := filepath.Abs(root.Path)
		if err != nil {
			continue
		}

		// Find the actual mount point / drive root.
		mountPoint := findMountPoint(absPath)
		if seen[mountPoint] {
			continue
		}
		seen[mountPoint] = true

		total, free, err := getDiskUsage(mountPoint)
		if err != nil {
			continue
		}

		used := total - free
		disks = append(disks, DiskUsageInfo{
			MountPoint: mountPoint,
			Total:      total,
			Free:       free,
			Used:       used,
			Label:      root.Name,
		})
	}

	if disks == nil {
		disks = []DiskUsageInfo{}
	}
	return disks
}

// findMountPoint walks up the path to find the filesystem root (drive root on Windows).
func findMountPoint(absPath string) string {
	for {
		parent := filepath.Dir(absPath)
		if parent == absPath {
			return absPath
		}
		absPath = parent
	}
}

// isMountPoint checks if a path is a filesystem root.
// On Windows this is a drive root like C:\, on Unix it's /.
func isMountPoint(path string) bool {
	parent := filepath.Dir(path)
	return parent == path
}

// dirSize calculates total size of a directory recursively (fallback method).
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
