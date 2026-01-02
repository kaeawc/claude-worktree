package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FakeMetadataStore is a fake implementation of MetadataStore for testing
type FakeMetadataStore struct {
	mu          sync.RWMutex
	data        map[string]*Metadata
	saveCount   int
	loadCount   int
	deleteCount int
	errors      map[string]error
}

// NewFakeMetadataStore creates a new fake metadata store
func NewFakeMetadataStore() *FakeMetadataStore {
	return &FakeMetadataStore{
		data:   make(map[string]*Metadata),
		errors: make(map[string]error),
	}
}

// SaveMetadata saves metadata to memory
func (f *FakeMetadataStore) SaveMetadata(metadata *Metadata) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.saveCount++

	if err, ok := f.errors["SaveMetadata"]; ok {
		return err
	}

	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	metadata.LastAccessedAt = time.Now()
	f.data[metadata.SessionName] = metadata

	return nil
}

// LoadMetadata loads metadata from memory
func (f *FakeMetadataStore) LoadMetadata(sessionName string) (*Metadata, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.loadCount++

	if err, ok := f.errors["LoadMetadata"]; ok {
		return nil, err
	}

	metadata, ok := f.data[sessionName]
	if !ok {
		return nil, fmt.Errorf("metadata not found: %s", sessionName)
	}

	return metadata, nil
}

// DeleteMetadata deletes metadata from memory
func (f *FakeMetadataStore) DeleteMetadata(sessionName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.deleteCount++

	if err, ok := f.errors["DeleteMetadata"]; ok {
		return err
	}

	delete(f.data, sessionName)

	return nil
}

// ListMetadata returns all session names
func (f *FakeMetadataStore) ListMetadata() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors["ListMetadata"]; ok {
		return nil, err
	}

	sessions := make([]string, 0, len(f.data))
	for name := range f.data {
		sessions = append(sessions, name)
	}

	return sessions, nil
}

// LoadAllMetadata loads all metadata
func (f *FakeMetadataStore) LoadAllMetadata() ([]*Metadata, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors["LoadAllMetadata"]; ok {
		return nil, err
	}

	metadata := make([]*Metadata, 0, len(f.data))
	for _, m := range f.data {
		metadata = append(metadata, m)
	}

	return metadata, nil
}

// ExistsMetadata checks if metadata exists
func (f *FakeMetadataStore) ExistsMetadata(sessionName string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, ok := f.data[sessionName]

	return ok
}

// UpdateStatus updates the status
func (f *FakeMetadataStore) UpdateStatus(sessionName string, status Status) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.errors["UpdateStatus"]; ok {
		return err
	}

	metadata, ok := f.data[sessionName]
	if !ok {
		return fmt.Errorf("metadata not found: %s", sessionName)
	}

	metadata.Status = status

	return nil
}

// GetCallCount returns the number of save calls
func (f *FakeMetadataStore) GetCallCount(method string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	switch method {
	case "SaveMetadata":
		return f.saveCount
	case "LoadMetadata":
		return f.loadCount
	case "DeleteMetadata":
		return f.deleteCount
	default:
		return 0
	}
}

// SetError sets an error to be returned by a method
func (f *FakeMetadataStore) SetError(method string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.errors[method] = err
}

// GetData returns the internal data for inspection
func (f *FakeMetadataStore) GetData() map[string]*Metadata {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Return a copy to prevent external modification
	dataCopy := make(map[string]*Metadata, len(f.data))
	for k, v := range f.data {
		dataCopy[k] = v
	}

	return dataCopy
}

// FakeOperations is a fake implementation of SessionOperations
type FakeOperations struct {
	mu              sync.RWMutex
	activeSessions  map[string]bool
	attachedSession string
	attachErrors    map[string]error
	sessionType     Type
	isAvailable     bool
	killCount       int
}

// NewFakeOperations creates a new fake session operations
func NewFakeOperations(sessionType Type, available bool) *FakeOperations {
	return &FakeOperations{
		activeSessions: make(map[string]bool),
		attachErrors:   make(map[string]error),
		sessionType:    sessionType,
		isAvailable:    available,
	}
}

// HasSession checks if session exists
func (f *FakeOperations) HasSession(name string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.activeSessions[name], nil
}

// ListSessions lists all sessions
func (f *FakeOperations) ListSessions() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	sessions := make([]string, 0, len(f.activeSessions))

	for name, active := range f.activeSessions {
		if active {
			sessions = append(sessions, name)
		}
	}

	return sessions, nil
}

// KillSession kills a session
func (f *FakeOperations) KillSession(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.killCount++
	f.activeSessions[name] = false

	return nil
}

// AttachToSession attaches to a session
func (f *FakeOperations) AttachToSession(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.attachErrors[name]; ok {
		return err
	}

	f.attachedSession = name

	return nil
}

// SessionType returns the session type
func (f *FakeOperations) SessionType() Type {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.sessionType
}

// IsAvailable returns whether a session manager is available
func (f *FakeOperations) IsAvailable() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.isAvailable
}

// AddSession adds an active session for testing
func (f *FakeOperations) AddSession(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.activeSessions[name] = true
}

// RemoveSession removes a session for testing
func (f *FakeOperations) RemoveSession(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.activeSessions[name] = false
}

// GetAttachedSession returns the last attached session
func (f *FakeOperations) GetAttachedSession() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.attachedSession
}

// GetKillCount returns the number of kill operations
func (f *FakeOperations) GetKillCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.killCount
}

// SetAttachError sets an error for AttachToSession
func (f *FakeOperations) SetAttachError(sessionName string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.attachErrors[sessionName] = err
}

// FakeDependencyInstaller is a fake implementation of DependencyInstaller
type FakeDependencyInstaller struct {
	mu            sync.RWMutex
	result        *DependenciesInfo
	error         error
	installPath   string
	progressCalls []string
}

// NewFakeDependencyInstaller creates a new fake installer
func NewFakeDependencyInstaller() *FakeDependencyInstaller {
	return &FakeDependencyInstaller{
		progressCalls: []string{},
	}
}

// Install installs dependencies
func (f *FakeDependencyInstaller) Install(worktreePath string, onProgress func(string)) (*DependenciesInfo, error) {
	f.mu.Lock()
	f.installPath = worktreePath
	f.mu.Unlock()

	if onProgress != nil {
		onProgress("Installing dependencies")
		f.mu.Lock()
		f.progressCalls = append(f.progressCalls, "Installing dependencies")
		f.mu.Unlock()
	}

	if f.error != nil {
		return nil, f.error
	}

	return f.result, nil
}

// SetResult sets the result to return
func (f *FakeDependencyInstaller) SetResult(result *DependenciesInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.result = result
}

// SetError sets the error to return
func (f *FakeDependencyInstaller) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.error = err
}

// GetInstallPath returns the path that was passed to Install
func (f *FakeDependencyInstaller) GetInstallPath() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.installPath
}

// GetProgressCalls returns all progress messages
func (f *FakeDependencyInstaller) GetProgressCalls() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Return a copy
	calls := make([]string, len(f.progressCalls))
	copy(calls, f.progressCalls)

	return calls
}

// FakeCleaner is a fake implementation of SessionCleaner
type FakeCleaner struct {
	mu                    sync.RWMutex
	cleanupResult         *CleanupResult
	cleanupError          error
	cleanupFilesError     error
	cleanupCalledWithOpts *CleanupOptions
}

// NewFakeCleaner creates a new fake cleaner
func NewFakeCleaner() *FakeCleaner {
	return &FakeCleaner{
		cleanupResult: &CleanupResult{},
	}
}

// CleanupOrphanedSessions cleans up orphaned sessions
func (f *FakeCleaner) CleanupOrphanedSessions(opts *CleanupOptions) (*CleanupResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if opts != nil {
		f.cleanupCalledWithOpts = opts
	}

	if f.cleanupError != nil {
		return nil, f.cleanupError
	}

	return f.cleanupResult, nil
}

// CleanupOrphanedMetadataFiles cleans up orphaned metadata files
func (f *FakeCleaner) CleanupOrphanedMetadataFiles(_ *CleanupOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.cleanupFilesError
}

// SetCleanupResult sets the result to return
func (f *FakeCleaner) SetCleanupResult(result *CleanupResult) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cleanupResult = result
}

// SetCleanupError sets the error to return
func (f *FakeCleaner) SetCleanupError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cleanupError = err
}

// GetCleanupCalledWithOpts returns the options passed to cleanup
func (f *FakeCleaner) GetCleanupCalledWithOpts() *CleanupOptions {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.cleanupCalledWithOpts
}

// FakeFileSystem is a fake implementation of FileSystem for testing
type FakeFileSystem struct {
	mu           sync.RWMutex
	files        map[string]bool  // path -> exists
	removeErrors map[string]error // path -> error to return
	removeCount  int
}

// NewFakeFileSystem creates a new fake file system instance
func NewFakeFileSystem() *FakeFileSystem {
	return &FakeFileSystem{
		files:        make(map[string]bool),
		removeErrors: make(map[string]error),
	}
}

// ReadDir returns mock directory entries
func (f *FakeFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var entries []os.DirEntry

	for path := range f.files {
		if filepath.Dir(path) == name && f.files[path] {
			entries = append(entries, &fakeDirEntry{
				name: filepath.Base(path),
			})
		}
	}

	return entries, nil
}

// Remove removes a file from the fake filesystem
func (f *FakeFileSystem) Remove(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.removeCount++

	if err, ok := f.removeErrors[name]; ok {
		return err
	}

	delete(f.files, name)

	return nil
}

// Join joins path elements
func (f *FakeFileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// AddFile adds a mock file to the fake filesystem
func (f *FakeFileSystem) AddFile(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = true
}

// SetRemoveError sets an error to return for a specific file removal
func (f *FakeFileSystem) SetRemoveError(path string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeErrors[path] = err
}

// GetRemoveCount returns the number of times Remove was called
func (f *FakeFileSystem) GetRemoveCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.removeCount
}

// fakeDirEntry implements os.DirEntry for testing
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f *fakeDirEntry) Name() string               { return f.name }
func (f *fakeDirEntry) IsDir() bool                { return f.isDir }
func (f *fakeDirEntry) Type() os.FileMode          { return 0 }
func (f *fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }
