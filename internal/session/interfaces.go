package session

import "os"

// Testing Guide for Session Package
//
// This package uses interface-based design to enable easy testing with fake
// implementations. All major components have corresponding interfaces and fake
// implementations to support unit testing without external dependencies.
//
// ## Available Interfaces
//
// 1. MetadataStore - Persistence of session metadata
// 2. SessionOperations - Tmux/screen session operations
// 3. SessionMetadataManager - High-level session state management
// 4. SessionManager - Combined interface for all session operations
// 5. DependencyInstaller - Project dependency installation
// 6. SessionCleaner - Session cleanup and orphan detection
//
// ## Fake Implementations
//
// For each interface, a fake implementation is provided in fakes.go:
// - FakeMetadataStore
// - FakeSessionOperations
// - FakeDependencyInstaller
// - FakeSessionCleaner
//
// ## Using Fakes in Tests
//
// Example:
//
//   func TestMyFunction(t *testing.T) {
//       fakeStore := NewFakeMetadataStore()
//       manager := &Manager{
//           sessionType: TypeTmux,
//           metadataStore: fakeStore,
//       }
//
//       // Use manager with fake store - no filesystem access!
//       manager.SaveSessionMetadata(&Metadata{...})
//
//       // Inspect fake for verification
//       if count := fakeStore.GetCallCount("SaveMetadata"); count != 1 {
//           t.Errorf("expected 1 save, got %d", count)
//       }
//   }
//
// ## Benefits of This Approach
//
// - Fast: No filesystem or subprocess calls needed
// - Isolated: Tests don't depend on external state
// - Deterministic: No timing issues or race conditions
// - Thread-safe: All fakes use proper synchronization
// - Observable: Can verify calls and inspect internal state
// - Testable Error Conditions: Can simulate failures easily
//

// MetadataStore defines the interface for session metadata persistence
type MetadataStore interface {
	// SaveMetadata persists session metadata to storage
	SaveMetadata(metadata *Metadata) error

	// LoadMetadata loads metadata for a session
	LoadMetadata(sessionName string) (*Metadata, error)

	// DeleteMetadata removes metadata for a session
	DeleteMetadata(sessionName string) error

	// ListMetadata returns all session metadata names
	ListMetadata() ([]string, error)

	// LoadAllMetadata loads all session metadata
	LoadAllMetadata() ([]*Metadata, error)

	// ExistsMetadata checks if metadata exists for a session
	ExistsMetadata(sessionName string) bool

	// UpdateStatus updates only the status of a session
	UpdateStatus(sessionName string, status Status) error
}

// Operations defines the interface for session operations
type Operations interface {
	// HasSession checks if a session exists
	HasSession(name string) (bool, error)

	// ListSessions returns all active sessions
	ListSessions() ([]string, error)

	// KillSession terminates a session
	KillSession(name string) error

	// AttachToSession opens a terminal window attached to the session
	AttachToSession(name string) error

	// SessionType returns the multiplexer type (tmux, screen, none)
	SessionType() Type

	// IsAvailable returns true if a session manager is available
	IsAvailable() bool
}

// MetadataManager defines the interface for session metadata management
type MetadataManager interface {
	// SaveSessionMetadata saves metadata for a session
	SaveSessionMetadata(metadata *Metadata) error

	// LoadSessionMetadata loads metadata for a session
	LoadSessionMetadata(sessionName string) (*Metadata, error)

	// DeleteSessionMetadata removes metadata for a session
	DeleteSessionMetadata(sessionName string) error

	// ListSessionMetadata returns all session metadata
	ListSessionMetadata() ([]string, error)

	// LoadAllSessionMetadata loads all session metadata
	LoadAllSessionMetadata() ([]*Metadata, error)

	// UpdateSessionStatus updates the status of a session
	UpdateSessionStatus(sessionName string, status Status) error

	// PauseSession marks a session as paused
	PauseSession(sessionName string) error

	// ResumeSession marks a session as running
	ResumeSession(sessionName string) error

	// GetSessionStatus returns the current status
	GetSessionStatus(sessionName string) (Status, error)

	// MarkSessionFailed marks a session as failed
	MarkSessionFailed(sessionName string) error

	// MarkSessionIdle marks a session as idle
	MarkSessionIdle(sessionName string) error

	// SyncSessionStatus synchronizes session metadata with actual state
	SyncSessionStatus(sessionName string) error
}

// Manager combines all session operations
type Manager interface {
	Operations
	MetadataManager

	// CreateSession creates a new detached session with optional command
	CreateSession(name, workingDir string, command []string) error
}

// DependencyInstaller defines the interface for installing project dependencies
type DependencyInstaller interface {
	// Install installs dependencies and returns the metadata
	Install(worktreePath string, onProgress func(string)) (*DependenciesInfo, error)
}

// Cleaner defines the interface for cleaning up sessions
type Cleaner interface {
	// CleanupOrphanedSessions cleans up metadata for sessions that no longer exist
	CleanupOrphanedSessions(opts *CleanupOptions) (*CleanupResult, error)

	// CleanupOrphanedMetadataFiles removes orphaned metadata files
	CleanupOrphanedMetadataFiles(opts *CleanupOptions) error
}

// FileSystem abstracts filesystem operations for testing
type FileSystem interface {
	// ReadDir reads the named directory
	ReadDir(name string) ([]os.DirEntry, error)

	// Remove removes the named file or empty directory
	Remove(name string) error

	// Join joins path elements into a single path
	Join(elem ...string) string
}
