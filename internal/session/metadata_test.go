package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataStore_SaveAndLoadMetadata(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create metadata store
	store, err := NewMetadataStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// Create test metadata
	now := time.Now()
	metadata := &Metadata{
		SessionName:    "test-session",
		SessionID:      "uuid-123",
		SessionType:    "tmux",
		WorktreePath:   "/path/to/worktree",
		BranchName:     "test-branch",
		CreatedAt:      now,
		LastAccessedAt: now,
		Status:         StatusRunning,
		WindowCount:    2,
		PaneCount:      3,
		RootProcessPid: 12345,
		Dependencies: DependenciesInfo{
			Installed:      true,
			ProjectType:    "nodejs",
			PackageManager: "npm",
		},
		CustomMetadata: map[string]interface{}{
			"issueId": "gh-42",
		},
	}

	// Save metadata
	if err := store.SaveMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Load metadata
	loaded, loadErr := store.LoadMetadata("test-session")
	if loadErr != nil {
		t.Fatalf("failed to load metadata: %v", loadErr)
	}

	// Verify metadata
	if loaded.SessionName != metadata.SessionName {
		t.Errorf("expected session name %q, got %q", metadata.SessionName, loaded.SessionName)
	}
	if loaded.BranchName != metadata.BranchName {
		t.Errorf("expected branch name %q, got %q", metadata.BranchName, loaded.BranchName)
	}
	if loaded.Status != StatusRunning {
		t.Errorf("expected status running, got %v", loaded.Status)
	}
	if loaded.WindowCount != 2 {
		t.Errorf("expected 2 windows, got %d", loaded.WindowCount)
	}
}

func TestMetadataStore_DeleteMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Save metadata
	metadata := &Metadata{
		SessionName: "test-delete",
		SessionType: "tmux",
		Status:      StatusRunning,
	}
	store.SaveMetadata(metadata)

	// Delete metadata
	if err := store.DeleteMetadata("test-delete"); err != nil {
		t.Fatalf("failed to delete metadata: %v", err)
	}

	// Verify it's deleted
	if store.ExistsMetadata("test-delete") {
		t.Errorf("metadata should be deleted but still exists")
	}
}

func TestMetadataStore_ListMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Save multiple metadata
	sessionNames := []string{"session1", "session2", "session3"}
	for _, name := range sessionNames {
		metadata := &Metadata{
			SessionName: name,
			SessionType: "tmux",
			Status:      StatusRunning,
		}
		store.SaveMetadata(metadata)
	}

	// List metadata
	sessions, err := store.ListMetadata()
	if err != nil {
		t.Fatalf("failed to list metadata: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestMetadataStore_UpdateStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Save initial metadata
	metadata := &Metadata{
		SessionName: "test-status",
		SessionType: "tmux",
		Status:      StatusRunning,
	}
	store.SaveMetadata(metadata)

	// Update status
	if err := store.UpdateStatus("test-status", StatusPaused); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	// Verify status was updated
	loaded, _ := store.LoadMetadata("test-status")
	if loaded.Status != StatusPaused {
		t.Errorf("expected status paused, got %v", loaded.Status)
	}
}

func TestMetadataStore_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Save metadata multiple times to test atomicity
	for i := 0; i < 10; i++ {
		metadata := &Metadata{
			SessionName: "atomic-test",
			SessionType: "tmux",
			Status:      StatusRunning,
			WindowCount: i,
		}
		if err := store.SaveMetadata(metadata); err != nil {
			t.Fatalf("failed to save on iteration %d: %v", i, err)
		}
	}

	// Verify final state
	loaded, _ := store.LoadMetadata("atomic-test")
	if loaded.WindowCount != 9 {
		t.Errorf("expected final window count 9, got %d", loaded.WindowCount)
	}

	// Verify no temporary files were left
	files, _ := os.ReadDir(tmpDir)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".tmp" {
			t.Errorf("found leftover temporary file: %s", f.Name())
		}
	}
}

func TestMetadataStore_ExistsMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Verify non-existent metadata
	if store.ExistsMetadata("nonexistent") {
		t.Errorf("metadata should not exist")
	}

	// Save metadata
	metadata := &Metadata{
		SessionName: "exists-test",
		SessionType: "tmux",
		Status:      StatusRunning,
	}
	store.SaveMetadata(metadata)

	// Verify it exists
	if !store.ExistsMetadata("exists-test") {
		t.Errorf("metadata should exist")
	}
}

func TestMetadataStore_LastAccessedAtUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	beforeSave := time.Now()

	metadata := &Metadata{
		SessionName:    "access-test",
		SessionType:    "tmux",
		Status:         StatusRunning,
		CreatedAt:      beforeSave.Add(-1 * time.Hour),
		LastAccessedAt: beforeSave.Add(-1 * time.Hour),
	}
	store.SaveMetadata(metadata)

	// Load and verify LastAccessedAt was updated
	loaded, _ := store.LoadMetadata("access-test")
	if loaded.LastAccessedAt.Before(beforeSave) {
		t.Errorf("LastAccessedAt should have been updated to current time")
	}
}

func TestMetadataStore_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Write corrupted JSON to a metadata file
	metadataPath := filepath.Join(tmpDir, "corrupt-session.json")
	os.WriteFile(metadataPath, []byte("{invalid json"), 0o600)

	// Try to load corrupted metadata
	_, err := store.LoadMetadata("corrupt-session")
	if err == nil {
		t.Errorf("expected error loading corrupted metadata")
	}
}

func TestMetadataStore_LoadAllMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewMetadataStore(tmpDir)

	// Save multiple metadata
	sessionNames := []string{"session-1", "session-2", "session-3", "session-4", "session-5"}
	for _, name := range sessionNames {
		metadata := &Metadata{
			SessionName: name,
			SessionType: "tmux",
			Status:      StatusRunning,
		}
		store.SaveMetadata(metadata)
	}

	// Load all metadata
	allMetadata, err := store.LoadAllMetadata()
	if err != nil {
		t.Fatalf("failed to load all metadata: %v", err)
	}

	if len(allMetadata) != len(sessionNames) {
		t.Errorf("expected %d metadata, got %d", len(sessionNames), len(allMetadata))
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusRunning, "running"},
		{StatusPaused, "paused"},
		{StatusIdle, "idle"},
		{StatusFailed, "failed"},
		{StatusUnknown, "unknown"},
	}

	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("expected %q, got %q", test.expected, string(test.status))
		}
	}
}

func TestDependenciesInfo(t *testing.T) {
	now := time.Now()
	deps := DependenciesInfo{
		Installed:      true,
		ProjectType:    "nodejs",
		PackageManager: "npm",
		InstalledAt:    &now,
	}

	if !deps.Installed {
		t.Errorf("expected installed to be true")
	}
	if deps.ProjectType != "nodejs" {
		t.Errorf("expected project type nodejs, got %q", deps.ProjectType)
	}
	if deps.PackageManager != "npm" {
		t.Errorf("expected npm, got %q", deps.PackageManager)
	}
	if deps.InstalledAt == nil {
		t.Errorf("expected InstalledAt to be set")
	}
}
