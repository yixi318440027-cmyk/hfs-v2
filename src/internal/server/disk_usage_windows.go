//go:build windows

package server

import (
	"golang.org/x/sys/windows"
)

func diskUsage(path string) (uint64, uint64, error) {
	var freeBytes, totalBytes, availBytes uint64
	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(path),
		&availBytes,
		&totalBytes,
		&freeBytes,
	)
	if err != nil {
		return 0, 0, err
	}
	return totalBytes, freeBytes, nil
}
