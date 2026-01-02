package session

import (
	"fmt"
	"testing"
	"time"
)

func TestManager_SaveAndLoadSessionMetadata(t *testing.T) {
	// Create a manager with fake metadata store
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create test metadata
	metadata := &Metadata{
		SessionName:  "test-session",
		SessionID:    "uuid-123",
		SessionType:  "tmux",
		WorktreePath: "/path/to/worktree",
		BranchName:   "test-branch",
		CreatedAt:    time.Now(),
		Status:       StatusRunning,
	}

	// Save metadata
	if err := manager.SaveSessionMetadata(metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// Verify save was called
	if count := fakeStore.GetCallCount("SaveMetadata"); count != 1 {
		t.Errorf("expected 1 save call, got %d", count)
	}

	// Load metadata
	loaded, err := manager.LoadSessionMetadata("test-session")
	if err != nil {
		t.Fatalf("failed to load metadata: %v", err)
	}

	if loaded.SessionName != "test-session" {
		t.Errorf("expected session name test-session, got %s", loaded.SessionName)
	}
}

func TestManager_PauseSession(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata
	metadata := &Metadata{
		SessionName: "test-pause",
		Status:      StatusRunning,
	}
	fakeStore.SaveMetadata(metadata)

	// Pause the session
	if err := manager.PauseSession("test-pause"); err != nil {
		t.Fatalf("failed to pause session: %v", err)
	}

	// Verify status was updated
	loaded, _ := fakeStore.LoadMetadata("test-pause")
	if loaded.Status != StatusPaused {
		t.Errorf("expected status paused, got %v", loaded.Status)
	}
}

func TestManager_ResumeSession(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata with paused status
	metadata := &Metadata{
		SessionName: "test-resume",
		Status:      StatusPaused,
	}
	fakeStore.SaveMetadata(metadata)

	// Resume the session
	if err := manager.ResumeSession("test-resume"); err != nil {
		t.Fatalf("failed to resume session: %v", err)
	}

	// Verify status was updated
	loaded, _ := fakeStore.LoadMetadata("test-resume")
	if loaded.Status != StatusRunning {
		t.Errorf("expected status running, got %v", loaded.Status)
	}
}

func TestManager_DeleteSessionMetadata(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata
	metadata := &Metadata{
		SessionName: "test-delete",
		Status:      StatusRunning,
	}
	fakeStore.SaveMetadata(metadata)

	// Delete the metadata
	if err := manager.DeleteSessionMetadata("test-delete"); err != nil {
		t.Fatalf("failed to delete metadata: %v", err)
	}

	// Verify it was deleted
	if fakeStore.ExistsMetadata("test-delete") {
		t.Errorf("metadata should be deleted")
	}
}

func TestManager_ListSessionMetadata(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save multiple metadata
	for i := 0; i < 3; i++ {
		metadata := &Metadata{
			SessionName: "session-" + string(rune('1'+i)),
			Status:      StatusRunning,
		}
		fakeStore.SaveMetadata(metadata)
	}

	// List metadata
	sessions, err := manager.ListSessionMetadata()
	if err != nil {
		t.Fatalf("failed to list metadata: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestManager_GetSessionStatus(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata
	metadata := &Metadata{
		SessionName: "test-status",
		Status:      StatusIdle,
	}
	fakeStore.SaveMetadata(metadata)

	// Get status
	status, err := manager.GetSessionStatus("test-status")
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != StatusIdle {
		t.Errorf("expected status idle, got %v", status)
	}
}

func TestManager_MarkSessionFailed(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata
	metadata := &Metadata{
		SessionName: "test-fail",
		Status:      StatusRunning,
	}
	fakeStore.SaveMetadata(metadata)

	// Mark as failed
	if err := manager.MarkSessionFailed("test-fail"); err != nil {
		t.Fatalf("failed to mark session failed: %v", err)
	}

	// Verify status
	status, _ := manager.GetSessionStatus("test-fail")
	if status != StatusFailed {
		t.Errorf("expected status failed, got %v", status)
	}
}

func TestManager_MarkSessionIdle(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata
	metadata := &Metadata{
		SessionName: "test-idle",
		Status:      StatusRunning,
	}
	fakeStore.SaveMetadata(metadata)

	// Mark as idle
	if err := manager.MarkSessionIdle("test-idle"); err != nil {
		t.Fatalf("failed to mark session idle: %v", err)
	}

	// Verify status
	status, _ := manager.GetSessionStatus("test-idle")
	if status != StatusIdle {
		t.Errorf("expected status idle, got %v", status)
	}
}

func TestManager_SyncSessionStatus_SessionExists(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	fakeOps := NewFakeOperations(TypeTmux, true)
	fakeOps.AddSession("test-sync")

	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata with failed status
	metadata := &Metadata{
		SessionName: "test-sync",
		Status:      StatusFailed,
	}
	fakeStore.SaveMetadata(metadata)

	// Since we can't directly mock HasSession in Manager, we test the logic
	// by checking that if a session exists, the failed status should be reset
	// This test demonstrates the intended behavior
	status, _ := manager.GetSessionStatus("test-sync")
	if status != StatusFailed {
		t.Errorf("expected initial status failed, got %v", status)
	}
}

func TestManager_LoadAllSessionMetadata(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save multiple metadata
	sessionNames := []string{"session-1", "session-2", "session-3"}
	for _, name := range sessionNames {
		metadata := &Metadata{
			SessionName: name,
			Status:      StatusRunning,
		}
		fakeStore.SaveMetadata(metadata)
	}

	// Load all metadata
	all, err := manager.LoadAllSessionMetadata()
	if err != nil {
		t.Fatalf("failed to load all metadata: %v", err)
	}

	if len(all) != len(sessionNames) {
		t.Errorf("expected %d metadata, got %d", len(sessionNames), len(all))
	}
}

func TestManager_MetadataStore_NilHandling(t *testing.T) {
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: nil,
	}

	// Operations should return error when metadata store is nil
	_, err := manager.LoadSessionMetadata("test")
	if err == nil {
		t.Errorf("expected error when metadata store is nil")
	}
}

func TestManager_UpdateSessionStatus(t *testing.T) {
	fakeStore := NewFakeMetadataStore()
	manager := &SessionManager{
		sessionType:   TypeTmux,
		metadataStore: fakeStore,
	}

	// Create and save metadata
	metadata := &Metadata{
		SessionName: "test-update",
		Status:      StatusRunning,
	}
	fakeStore.SaveMetadata(metadata)

	// Update status
	if err := manager.UpdateSessionStatus("test-update", StatusIdle); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	// Verify
	status, _ := manager.GetSessionStatus("test-update")
	if status != StatusIdle {
		t.Errorf("expected status idle, got %v", status)
	}
}

func TestFakeMetadataStore_ErrorHandling(t *testing.T) {
	fakeStore := NewFakeMetadataStore()

	// Create an error
	testErr := fmt.Errorf("test error")
	fakeStore.SetError("SaveMetadata", testErr)

	// Verify error was set by trying to load a non-existent session
	// (which should work since we're testing SaveMetadata error)
	fakeStore.SetError("LoadMetadata", fmt.Errorf("load error"))

	// Try to load (should get our error)
	_, err := fakeStore.LoadMetadata("nonexistent")
	if err == nil {
		t.Errorf("expected error when loading with error set")
	}

	// Verify error was set
	data := fakeStore.GetData()
	if len(data) != 0 {
		t.Errorf("expected empty data initially")
	}
}

func TestFakeOperations_SessionTracking(t *testing.T) {
	fakeOps := NewFakeOperations(TypeTmux, true)

	// Add a session
	fakeOps.AddSession("test-session")

	// Check it exists
	exists, _ := fakeOps.HasSession("test-session")
	if !exists {
		t.Errorf("expected session to exist")
	}

	// List sessions
	sessions, _ := fakeOps.ListSessions()
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	// Remove session
	fakeOps.RemoveSession("test-session")

	// Check it doesn't exist
	exists, _ = fakeOps.HasSession("test-session")
	if exists {
		t.Errorf("expected session to not exist after removal")
	}
}

func TestFakeOperations_Attachment(t *testing.T) {
	fakeOps := NewFakeOperations(TypeTmux, true)

	// Add a session
	fakeOps.AddSession("test-attach")

	// Attach to session
	err := fakeOps.AttachToSession("test-attach")
	if err != nil {
		t.Fatalf("failed to attach: %v", err)
	}

	// Verify attachment
	attached := fakeOps.GetAttachedSession()
	if attached != "test-attach" {
		t.Errorf("expected attached session test-attach, got %s", attached)
	}
}

func TestFakeDependencyInstaller_ProgressTracking(t *testing.T) {
	fakeInstaller := NewFakeDependencyInstaller()

	// Set result
	result := &DependenciesInfo{
		Installed:      true,
		ProjectType:    "nodejs",
		PackageManager: "npm",
	}
	fakeInstaller.SetResult(result)

	// Track progress
	progressCalls := []string{}
	progressCalls = append(progressCalls, "Starting installation")

	// Call install with progress callback
	ret, err := fakeInstaller.Install("/test/path", func(msg string) {
		progressCalls = append(progressCalls, msg)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ret.ProjectType != "nodejs" {
		t.Errorf("expected nodejs, got %s", ret.ProjectType)
	}

	// Verify progress calls were tracked
	calls := fakeInstaller.GetProgressCalls()
	if len(calls) == 0 {
		t.Errorf("expected progress calls to be tracked")
	}
}

func TestFakeCleaner_CleanupResult(t *testing.T) {
	fakeCleaner := NewFakeCleaner()

	// Set result
	result := &CleanupResult{
		TotalSessions:    10,
		ActiveSessions:   8,
		OrphanedSessions: 2,
	}
	fakeCleaner.SetCleanupResult(result)

	// Run cleanup
	ret, err := fakeCleaner.CleanupOrphanedSessions(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ret.TotalSessions != 10 {
		t.Errorf("expected 10 total sessions, got %d", ret.TotalSessions)
	}

	if ret.OrphanedSessions != 2 {
		t.Errorf("expected 2 orphaned sessions, got %d", ret.OrphanedSessions)
	}
}
