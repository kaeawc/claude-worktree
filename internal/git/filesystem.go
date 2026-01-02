package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileSystem defines the interface for file system operations
type FileSystem interface {
	// MkdirAll creates a directory path
	MkdirAll(path string, perm os.FileMode) error
	// Remove removes a file or directory
	Remove(path string) error
	// RemoveAll removes a path and any children it contains
	RemoveAll(path string) error
	// ReadFile reads the entire file
	ReadFile(path string) ([]byte, error)
	// WriteFile writes data to a file
	WriteFile(path string, data []byte, perm os.FileMode) error
	// Stat returns file info
	Stat(path string) (os.FileInfo, error)
	// UserHomeDir returns the current user's home directory
	UserHomeDir() (string, error)
	// Walk walks the file tree
	Walk(root string, fn filepath.WalkFunc) error
	// Exists checks if a path exists
	Exists(path string) bool
	// Base returns the last element of path
	Base(path string) string
	// Join joins path elements
	Join(elem ...string) string
}

// RealFileSystem implements FileSystem using actual os/filepath functions
type RealFileSystem struct{}

// NewFileSystem creates a new real file system for production use
func NewFileSystem() FileSystem {
	return &RealFileSystem{}
}

// MkdirAll creates a directory path
func (f *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove removes a file or directory
func (f *RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll removes a path and any children it contains
func (f *RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// ReadFile reads the entire file
func (f *RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file
func (f *RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat returns file info
func (f *RealFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// UserHomeDir returns the current user's home directory
func (f *RealFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// Walk walks the file tree
func (f *RealFileSystem) Walk(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}

// Exists checks if a path exists
func (f *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FakeFileSystem is a fake implementation for testing with in-memory storage
type FakeFileSystem struct {
	// Files stores file contents keyed by path
	Files map[string][]byte
	// Dirs stores directories
	Dirs map[string]bool
	// Permissions stores file permissions
	Permissions map[string]os.FileMode
	// ModTimes stores modification times
	ModTimes map[string]time.Time
	// HomeDir is the configured home directory
	HomeDir string
	// Errors maps paths to errors for simulating failures
	Errors map[string]error
	// OperationLog records all operations for verification
	OperationLog []string
}

// NewFakeFileSystem creates a new fake file system for testing
func NewFakeFileSystem() *FakeFileSystem {
	return &FakeFileSystem{
		Files:        make(map[string][]byte),
		Dirs:         make(map[string]bool),
		Permissions:  make(map[string]os.FileMode),
		ModTimes:     make(map[string]time.Time),
		HomeDir:      "/home/testuser",
		Errors:       make(map[string]error),
		OperationLog: []string{},
	}
}

// MkdirAll creates a directory path in memory
func (f *FakeFileSystem) MkdirAll(path string, perm os.FileMode) error {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("MkdirAll(%s, %v)", path, perm))

	if err, ok := f.Errors[path]; ok {
		return err
	}

	// Create all parent directories
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if current == "" {
			current = string(filepath.Separator) + part
		} else {
			current = filepath.Join(current, part)
		}
		f.Dirs[current] = true
		f.Permissions[current] = perm
		f.ModTimes[current] = time.Now()
	}

	return nil
}

// Remove removes a file or directory from memory
func (f *FakeFileSystem) Remove(path string) error {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("Remove(%s)", path))

	if err, ok := f.Errors[path]; ok {
		return err
	}

	if _, exists := f.Files[path]; exists {
		delete(f.Files, path)
		delete(f.Permissions, path)
		delete(f.ModTimes, path)
		return nil
	}

	if _, exists := f.Dirs[path]; exists {
		delete(f.Dirs, path)
		delete(f.Permissions, path)
		delete(f.ModTimes, path)
		return nil
	}

	return fmt.Errorf("remove %s: no such file or directory", path)
}

// RemoveAll removes a path and all children from memory
func (f *FakeFileSystem) RemoveAll(path string) error {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("RemoveAll(%s)", path))

	if err, ok := f.Errors[path]; ok {
		return err
	}

	// Remove the path itself
	delete(f.Files, path)
	delete(f.Dirs, path)
	delete(f.Permissions, path)
	delete(f.ModTimes, path)

	// Remove all children
	prefix := path + string(filepath.Separator)
	for p := range f.Files {
		if strings.HasPrefix(p, prefix) {
			delete(f.Files, p)
			delete(f.Permissions, p)
			delete(f.ModTimes, p)
		}
	}
	for p := range f.Dirs {
		if strings.HasPrefix(p, prefix) {
			delete(f.Dirs, p)
			delete(f.Permissions, p)
			delete(f.ModTimes, p)
		}
	}

	return nil
}

// ReadFile reads a file from memory
func (f *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("ReadFile(%s)", path))

	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	if data, exists := f.Files[path]; exists {
		return data, nil
	}

	return nil, fmt.Errorf("open %s: no such file or directory", path)
}

// WriteFile writes a file to memory
func (f *FakeFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("WriteFile(%s, %d bytes, %v)", path, len(data), perm))

	if err, ok := f.Errors[path]; ok {
		return err
	}

	f.Files[path] = data
	f.Permissions[path] = perm
	f.ModTimes[path] = time.Now()
	return nil
}

// Stat returns file info from memory
func (f *FakeFileSystem) Stat(path string) (os.FileInfo, error) {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("Stat(%s)", path))

	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	if data, exists := f.Files[path]; exists {
		return &fakeFileInfo{
			name:    filepath.Base(path),
			size:    int64(len(data)),
			mode:    f.Permissions[path],
			modTime: f.ModTimes[path],
			isDir:   false,
		}, nil
	}

	if _, exists := f.Dirs[path]; exists {
		return &fakeFileInfo{
			name:    filepath.Base(path),
			size:    0,
			mode:    f.Permissions[path],
			modTime: f.ModTimes[path],
			isDir:   true,
		}, nil
	}

	return nil, fmt.Errorf("stat %s: no such file or directory", path)
}

// UserHomeDir returns the configured home directory
func (f *FakeFileSystem) UserHomeDir() (string, error) {
	f.OperationLog = append(f.OperationLog, "UserHomeDir()")

	if err, ok := f.Errors["UserHomeDir"]; ok {
		return "", err
	}

	return f.HomeDir, nil
}

// Walk walks the in-memory file tree
func (f *FakeFileSystem) Walk(root string, fn filepath.WalkFunc) error {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("Walk(%s)", root))

	if err, ok := f.Errors[root]; ok {
		return err
	}

	// Visit root
	info, err := f.Stat(root)
	if err != nil {
		return fn(root, nil, err)
	}
	if err := fn(root, info, nil); err != nil {
		return err
	}

	// Visit children
	prefix := root + string(filepath.Separator)
	visited := make(map[string]bool)

	// Visit directories
	for dir := range f.Dirs {
		if strings.HasPrefix(dir, prefix) && dir != root {
			if !visited[dir] {
				info, err := f.Stat(dir)
				if err := fn(dir, info, err); err != nil {
					return err
				}
				visited[dir] = true
			}
		}
	}

	// Visit files
	for file := range f.Files {
		if strings.HasPrefix(file, prefix) && file != root {
			if !visited[file] {
				info, err := f.Stat(file)
				if err := fn(file, info, err); err != nil {
					return err
				}
				visited[file] = true
			}
		}
	}

	return nil
}

// Exists checks if a path exists in memory
func (f *FakeFileSystem) Exists(path string) bool {
	f.OperationLog = append(f.OperationLog, fmt.Sprintf("Exists(%s)", path))

	if _, ok := f.Files[path]; ok {
		return true
	}
	if _, ok := f.Dirs[path]; ok {
		return true
	}
	return false
}

// Base returns the last element of path
func (f *RealFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Join joins path elements
func (f *RealFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Base returns the last element of path
func (f *FakeFileSystem) Base(path string) string {
	return filepath.Base(path)
}

// Join joins path elements
func (f *FakeFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// SetError configures an error for a specific path or operation
func (f *FakeFileSystem) SetError(path string, err error) {
	f.Errors[path] = err
}

// GetOperationCount returns the number of operations performed
func (f *FakeFileSystem) GetOperationCount() int {
	return len(f.OperationLog)
}

// GetLastOperation returns the last operation, or empty string if none
func (f *FakeFileSystem) GetLastOperation() string {
	if len(f.OperationLog) == 0 {
		return ""
	}
	return f.OperationLog[len(f.OperationLog)-1]
}

// Reset clears all state
func (f *FakeFileSystem) Reset() {
	f.Files = make(map[string][]byte)
	f.Dirs = make(map[string]bool)
	f.Permissions = make(map[string]os.FileMode)
	f.ModTimes = make(map[string]time.Time)
	f.Errors = make(map[string]error)
	f.OperationLog = []string{}
}

// fakeFileInfo implements os.FileInfo for testing
type fakeFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return f.size }
func (f *fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f *fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f *fakeFileInfo) IsDir() bool        { return f.isDir }
func (f *fakeFileInfo) Sys() interface{}   { return nil }
