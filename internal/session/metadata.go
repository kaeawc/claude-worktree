// Package session manages terminal multiplexer sessions for worktrees
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Status represents the state of a session
type Status string

// Session status constants
const (
	StatusRunning        Status = "running"
	StatusPaused         Status = "paused"
	StatusIdle           Status = "idle"
	StatusNeedsAttention Status = "needs_attention"
	StatusFailed         Status = "failed"
	StatusUnknown        Status = "unknown"
)

// Metadata represents persistent session metadata
type Metadata struct {
	SessionName    string                 `json:"sessionName"`
	SessionID      string                 `json:"sessionId"`
	SessionType    string                 `json:"sessionType"`
	WorktreePath   string                 `json:"worktreePath"`
	BranchName     string                 `json:"branchName"`
	CreatedAt      time.Time              `json:"createdAt"`
	LastAccessedAt time.Time              `json:"lastAccessedAt"`
	Status         Status                 `json:"status"`
	WindowCount    int                    `json:"windowCount"`
	PaneCount      int                    `json:"paneCount"`
	RootProcessPid int                    `json:"rootProcessPid"`
	Dependencies   DependenciesInfo       `json:"dependencies"`
	CustomMetadata map[string]interface{} `json:"customMetadata,omitempty"`
}

// DependenciesInfo tracks dependency installation state
type DependenciesInfo struct {
	Installed      bool       `json:"installed"`
	ProjectType    string     `json:"projectType"`
	PackageManager string     `json:"packageManager"`
	InstalledAt    *time.Time `json:"installedAt,omitempty"`
}

// FileMetadataStore handles reading/writing session metadata to the file system
type FileMetadataStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewMetadataStore creates a new file-based metadata store
func NewMetadataStore(baseDir string) (*FileMetadataStore, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &FileMetadataStore{
		baseDir: baseDir,
	}, nil
}

// GetSessionDir returns the directory where session metadata is stored
func GetSessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	sessionsDir := filepath.Join(home, ".auto-worktree", "sessions")

	return sessionsDir, nil
}

// metadataPath returns the path for a session's metadata file
func (s *FileMetadataStore) metadataPath(sessionName string) string {
	return filepath.Join(s.baseDir, sessionName+".json")
}

// SaveMetadata persists session metadata to disk
func (s *FileMetadataStore) SaveMetadata(metadata *Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if metadata.SessionName == "" {
		return fmt.Errorf("session name is required")
	}

	// Update LastAccessedAt to current time
	metadata.LastAccessedAt = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	path := s.metadataPath(metadata.SessionName)

	// Write to temporary file first for atomicity
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Rename to final path
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // Cleanup attempt on failure
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// LoadMetadata loads session metadata from disk
func (s *FileMetadataStore) LoadMetadata(sessionName string) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.metadataPath(sessionName)
	data, err := os.ReadFile(path) //nolint:gosec // G304: Path is derived from sessionName parameter

	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found for session: %s", sessionName)
		}

		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// DeleteMetadata removes session metadata from disk
func (s *FileMetadataStore) DeleteMetadata(sessionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.metadataPath(sessionName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

// ListMetadata returns all session metadata files
func (s *FileMetadataStore) ListMetadata() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)

	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, fmt.Errorf("failed to list metadata: %w", err)
	}

	var sessions []string

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessionName := entry.Name()[:len(entry.Name())-5] // Remove .json
			sessions = append(sessions, sessionName)
		}
	}

	return sessions, nil
}

// LoadAllMetadata loads all session metadata files
func (s *FileMetadataStore) LoadAllMetadata() ([]*Metadata, error) {
	sessionNames, err := s.ListMetadata()
	if err != nil {
		return nil, err
	}

	metadataList := make([]*Metadata, 0, len(sessionNames))

	for _, sessionName := range sessionNames {
		metadata, err := s.LoadMetadata(sessionName)

		if err != nil {
			// Skip corrupted metadata files
			continue
		}

		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

// ExistsMetadata checks if metadata exists for a session
func (s *FileMetadataStore) ExistsMetadata(sessionName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.metadataPath(sessionName)
	_, err := os.Stat(path)

	return err == nil
}

// UpdateStatus updates only the status of a session
func (s *FileMetadataStore) UpdateStatus(sessionName string, status Status) error {
	metadata, err := s.LoadMetadata(sessionName)
	if err != nil {
		return err
	}

	metadata.Status = status

	return s.SaveMetadata(metadata)
}
