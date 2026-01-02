package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CleanupResult contains information about cleanup operations
type CleanupResult struct {
	TotalSessions    int
	ActiveSessions   int
	OrphanedSessions int
	IdleSessions     int
	FailedSessions   int
	RemovedMetadata  []string
	Errors           []error
}

// CleanupOptions controls cleanup behavior
type CleanupOptions struct {
	// RemoveOrphanedMetadata removes metadata files for non-existent sessions
	RemoveOrphanedMetadata bool

	// MarkIdleAsIdle marks sessions idle after threshold without removing them
	MarkIdleAsIdle bool

	// IdleThresholdMinutes is the duration (in minutes) before marking as idle
	IdleThresholdMinutes int

	// DryRun performs cleanup checks without modifying files
	DryRun bool

	// OnProgress is called with progress messages
	OnProgress func(string)
}

// DefaultCleanupOptions returns default cleanup options
func DefaultCleanupOptions() *CleanupOptions {
	return &CleanupOptions{
		RemoveOrphanedMetadata: true,
		MarkIdleAsIdle:         true,
		IdleThresholdMinutes:   120,
		DryRun:                 false,
		OnProgress: func(string) {
			// No-op by default
		},
	}
}

// isMetadataFile checks if a directory entry is a metadata JSON file
func isMetadataFile(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}

	return filepath.Ext(entry.Name()) == ".json"
}

// extractSessionName extracts session name from a metadata filename
func extractSessionName(filename string) string {
	if !strings.HasSuffix(filename, ".json") {
		return ""
	}

	return filename[:len(filename)-5]
}

// sessionClassification holds the result of classifying a session
type sessionClassification struct {
	exists   bool
	isIdle   bool
	isFailed bool
	checkErr error
}

// classifySession determines the current state of a session
func (m *SessionManager) classifySession(metadata *Metadata) sessionClassification {
	exists, err := m.HasSession(metadata.SessionName)

	return sessionClassification{
		exists:   exists,
		isIdle:   metadata.Status == StatusIdle,
		isFailed: metadata.Status == StatusFailed,
		checkErr: err,
	}
}

// processOrphanedSession handles cleanup for a single orphaned session
func (m *SessionManager) processOrphanedSession(metadata *Metadata, opts *CleanupOptions) error {
	opts.OnProgress(fmt.Sprintf("Found orphaned session: %s", metadata.SessionName))

	if !opts.RemoveOrphanedMetadata || opts.DryRun {
		return nil
	}

	if err := m.DeleteSessionMetadata(metadata.SessionName); err != nil {
		return err
	}

	opts.OnProgress(fmt.Sprintf("Removed metadata for orphaned session: %s", metadata.SessionName))

	return nil
}

// isSessionIdle checks if a session is idle based on its last access time
func isSessionIdle(metadata *Metadata, idleThresholdMinutes int) bool {
	idleDuration := time.Since(metadata.LastAccessedAt)
	idleThreshold := time.Duration(idleThresholdMinutes) * time.Minute

	return idleDuration > idleThreshold
}

// processActiveSession handles idle detection for an active session
func (m *SessionManager) processActiveSession(metadata *Metadata, opts *CleanupOptions) error {
	// Early return if idle marking is disabled or already idle/failed
	if !opts.MarkIdleAsIdle || metadata.Status == StatusIdle || metadata.Status == StatusFailed {
		return nil
	}

	if !isSessionIdle(metadata, opts.IdleThresholdMinutes) {
		return nil
	}

	idleDuration := time.Since(metadata.LastAccessedAt)
	opts.OnProgress(fmt.Sprintf("Found idle session: %s (inactive for %v)", metadata.SessionName, idleDuration))

	if opts.DryRun {
		return nil
	}

	if err := m.MarkSessionIdle(metadata.SessionName); err != nil {
		return err
	}

	opts.OnProgress(fmt.Sprintf("Marked as idle: %s", metadata.SessionName))

	return nil
}

// processMetadataFile processes a single metadata file for cleanup
func (m *SessionManager) processMetadataFile(entry os.DirEntry, sessionDir string, opts *CleanupOptions, fs FileSystem) error {
	sessionName := extractSessionName(entry.Name())
	if sessionName == "" {
		return nil
	}

	exists, err := m.HasSession(sessionName)
	if err != nil {
		// Skip on error - don't delete metadata if we can't verify
		return nil
	}

	if exists {
		return nil
	}

	opts.OnProgress(fmt.Sprintf("Removing orphaned metadata file: %s.json", sessionName))

	if opts.DryRun {
		return nil
	}

	path := fs.Join(sessionDir, entry.Name())
	if err := fs.Remove(path); err != nil {
		opts.OnProgress(fmt.Sprintf("Failed to remove %s: %v", entry.Name(), err))
		return err
	}

	return nil
}

// processSingleMetadata processes a single metadata entry and updates results
func (m *SessionManager) processSingleMetadata(metadata *Metadata, opts *CleanupOptions, result *CleanupResult) {
	classification := m.classifySession(metadata)

	if classification.checkErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to check session %s: %w", metadata.SessionName, classification.checkErr))
		return
	}

	if !classification.exists {
		result.OrphanedSessions++
		if err := m.processOrphanedSession(metadata, opts); err != nil {
			result.Errors = append(result.Errors, err)
			return
		}

		if !opts.DryRun && opts.RemoveOrphanedMetadata {
			result.RemovedMetadata = append(result.RemovedMetadata, metadata.SessionName)
		}

		return
	}

	result.ActiveSessions++
	if err := m.processActiveSession(metadata, opts); err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	if classification.isIdle {
		result.IdleSessions++
	}

	if classification.isFailed {
		result.FailedSessions++
	}
}

// CleanupOrphanedSessions cleans up metadata for sessions that no longer exist
func (m *SessionManager) CleanupOrphanedSessions(opts *CleanupOptions) (*CleanupResult, error) {
	if opts == nil {
		opts = DefaultCleanupOptions()
	}

	result := &CleanupResult{
		RemovedMetadata: []string{},
		Errors:          []error{},
	}

	if opts.OnProgress == nil {
		opts.OnProgress = func(string) {}
	}

	allMetadata, err := m.LoadAllSessionMetadata()
	if err != nil {
		return result, fmt.Errorf("failed to load session metadata: %w", err)
	}

	result.TotalSessions = len(allMetadata)

	for _, metadata := range allMetadata {
		m.processSingleMetadata(metadata, opts, result)
	}

	return result, nil
}

// cleanupOrphanedMetadataFilesWithFS is the testable implementation using an injected FileSystem
func (m *SessionManager) cleanupOrphanedMetadataFilesWithFS(opts *CleanupOptions, fs FileSystem) error {
	if opts == nil {
		opts = DefaultCleanupOptions()
	}

	if opts.OnProgress == nil {
		opts.OnProgress = func(string) {}
	}

	sessionDir, err := GetSessionDir()
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	for _, entry := range entries {
		if !isMetadataFile(entry) {
			continue
		}

		_ = m.processMetadataFile(entry, sessionDir, opts, fs) //nolint:errcheck // Errors logged via OnProgress
		// Errors are logged via OnProgress but cleanup continues
	}

	return nil
}

// CleanupOrphanedMetadataFiles removes orphaned metadata files from disk
// This is useful if the metadata directory somehow contains files without corresponding sessions
func (m *SessionManager) CleanupOrphanedMetadataFiles(opts *CleanupOptions) error {
	return m.cleanupOrphanedMetadataFilesWithFS(opts, newRealFileSystem())
}
