package session

import (
	"os"
	"path/filepath"
)

// homeDir returns the current user's home directory, or "" if unknown.
func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

// pathSize returns the total size in bytes of a file, or of a directory's
// contents recursively. Missing paths contribute 0.
func pathSize(path string) int64 {
	var total int64
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if !fi.IsDir() {
		return fi.Size()
	}
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}

// removeAll removes every given path (file or directory tree), ignoring
// paths that no longer exist. It returns the first real error encountered,
// but keeps trying the remaining paths so a delete is as complete as
// possible even if one path fails.
func removeAll(paths []string) error {
	var firstErr error
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.RemoveAll(p); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
