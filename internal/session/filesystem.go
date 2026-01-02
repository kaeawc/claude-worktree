package session

import (
	"os"
	"path/filepath"
)

// realFileSystem implements FileSystem using real filesystem operations
type realFileSystem struct{}

// newRealFileSystem creates a new real file system instance
func newRealFileSystem() FileSystem {
	return &realFileSystem{}
}

// ReadDir reads the named directory
func (r *realFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

// Remove removes the named file or empty directory
func (r *realFileSystem) Remove(name string) error {
	return os.Remove(name)
}

// Join joins path elements into a single path
func (r *realFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}
