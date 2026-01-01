package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListWorktrees(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// List worktrees (should have at least the main worktree)
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	if len(worktrees) == 0 {
		t.Errorf("ListWorktrees() returned no worktrees, expected at least 1")
	}

	// Verify the main worktree
	mainWorktree := worktrees[0]
	// Compare resolved paths to handle symlinks (macOS /var vs /private/var)
	expectedPath, _ := filepath.EvalSymlinks(tmpDir)
	actualPath, _ := filepath.EvalSymlinks(mainWorktree.Path)
	if actualPath != expectedPath {
		t.Errorf("Main worktree path = %v, want %v", mainWorktree.Path, tmpDir)
	}

	if mainWorktree.Branch == "" {
		t.Errorf("Main worktree should have a branch name")
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Get current branch
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Create a new worktree with a new branch
	worktreePath := filepath.Join(tmpDir, "..", "test-worktree")
	testBranch := "test-worktree-branch"

	err = repo.CreateWorktreeWithNewBranch(worktreePath, testBranch, currentBranch)
	if err != nil {
		t.Fatalf("CreateWorktreeWithNewBranch() error = %v", err)
	}
	defer os.RemoveAll(worktreePath)

	// Verify worktree was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree directory was not created at %s", worktreePath)
	}

	// List worktrees and verify the new one exists
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	found := false
	expectedPath, _ := filepath.EvalSymlinks(worktreePath)
	for _, wt := range worktrees {
		actualPath, _ := filepath.EvalSymlinks(wt.Path)
		if actualPath == expectedPath && wt.Branch == testBranch {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Created worktree not found in list (path: %s, branch: %s)", worktreePath, testBranch)
	}

	// Remove the worktree
	if err := repo.RemoveWorktree(worktreePath); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}

	// Verify worktree was removed
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Errorf("Worktree directory still exists after removal at %s", worktreePath)
	}

	// Clean up the branch
	repo.DeleteBranch(testBranch)
}

func TestCreateWorktreeWithExistingBranch(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Get current branch
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Create a branch first
	testBranch := "existing-branch"
	if err := repo.CreateBranch(testBranch, currentBranch); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	defer repo.DeleteBranch(testBranch)

	// Create a worktree for the existing branch
	worktreePath := filepath.Join(tmpDir, "..", "test-existing-worktree")
	err = repo.CreateWorktree(worktreePath, testBranch)
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}
	defer repo.RemoveWorktree(worktreePath)
	defer os.RemoveAll(worktreePath)

	// Verify worktree was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree directory was not created at %s", worktreePath)
	}
}

func TestGetWorktreeForBranch(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Get current branch
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Should find the main worktree
	wt, err := repo.GetWorktreeForBranch(currentBranch)
	if err != nil {
		t.Fatalf("GetWorktreeForBranch() error = %v", err)
	}

	if wt == nil {
		t.Errorf("Expected to find worktree for branch %q", currentBranch)
	} else if wt.Branch != currentBranch {
		t.Errorf("Found worktree with branch %q, want %q", wt.Branch, currentBranch)
	}

	// Should not find worktree for non-existent branch
	wt, err = repo.GetWorktreeForBranch("nonexistent-branch")
	if err != nil {
		t.Fatalf("GetWorktreeForBranch() error = %v", err)
	}

	if wt != nil {
		t.Errorf("Expected nil worktree for non-existent branch, got %v", wt)
	}
}

func TestParseWorktreeList(t *testing.T) {
	porcelainOutput := `worktree /home/user/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /home/user/worktrees/repo/feature
HEAD abcdef1234567890abcdef1234567890abcdef12
branch refs/heads/feature

worktree /home/user/worktrees/repo/detached
HEAD 9876543210fedcba9876543210fedcba98765432
detached
`

	worktrees, err := parseWorktreeList(porcelainOutput)
	if err != nil {
		t.Fatalf("parseWorktreeList() error = %v", err)
	}

	if len(worktrees) != 3 {
		t.Fatalf("parseWorktreeList() returned %d worktrees, want 3", len(worktrees))
	}

	// Check first worktree (main)
	if worktrees[0].Path != "/home/user/repo" {
		t.Errorf("worktrees[0].Path = %v, want /home/user/repo", worktrees[0].Path)
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("worktrees[0].Branch = %v, want main", worktrees[0].Branch)
	}
	if worktrees[0].HEAD != "1234567890abcdef1234567890abcdef12345678" {
		t.Errorf("worktrees[0].HEAD = %v", worktrees[0].HEAD)
	}
	if worktrees[0].IsDetached {
		t.Errorf("worktrees[0].IsDetached should be false")
	}

	// Check second worktree (feature branch)
	if worktrees[1].Path != "/home/user/worktrees/repo/feature" {
		t.Errorf("worktrees[1].Path = %v", worktrees[1].Path)
	}
	if worktrees[1].Branch != "feature" {
		t.Errorf("worktrees[1].Branch = %v, want feature", worktrees[1].Branch)
	}

	// Check third worktree (detached)
	if worktrees[2].Path != "/home/user/worktrees/repo/detached" {
		t.Errorf("worktrees[2].Path = %v", worktrees[2].Path)
	}
	if !worktrees[2].IsDetached {
		t.Errorf("worktrees[2].IsDetached should be true")
	}
}

func TestWorktreeAge(t *testing.T) {
	wt := &Worktree{
		LastCommitTime: time.Now().Add(-24 * time.Hour),
	}

	age := wt.Age()
	if age < 23*time.Hour || age > 25*time.Hour {
		t.Errorf("Age() = %v, expected approximately 24 hours", age)
	}
}

func TestGetLastCommitTimestamp(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	timestamp, err := getLastCommitTimestamp(tmpDir)
	if err != nil {
		t.Fatalf("getLastCommitTimestamp() error = %v", err)
	}

	// Should be a recent timestamp (within the last minute)
	if time.Since(timestamp) > time.Minute {
		t.Errorf("Last commit timestamp is too old: %v", timestamp)
	}
}

func TestGetUnpushedCommitCount(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Since there's no upstream, this should count total commits
	count, err := getUnpushedCommitCount(tmpDir, currentBranch)
	if err != nil {
		t.Fatalf("getUnpushedCommitCount() error = %v", err)
	}

	// Should have at least 1 commit (the initial commit)
	if count < 1 {
		t.Errorf("getUnpushedCommitCount() = %d, expected at least 1", count)
	}
}

func TestPruneWorktrees(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Get current branch
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Create a worktree
	worktreePath := filepath.Join(tmpDir, "..", "prune-test-worktree")
	testBranch := "prune-test-branch"

	err = repo.CreateWorktreeWithNewBranch(worktreePath, testBranch, currentBranch)
	if err != nil {
		t.Fatalf("CreateWorktreeWithNewBranch() error = %v", err)
	}

	// Manually delete the worktree directory (simulating orphaned worktree)
	os.RemoveAll(worktreePath)

	// Prune should clean up the orphaned worktree
	if err := repo.PruneWorktrees(); err != nil {
		t.Fatalf("PruneWorktrees() error = %v", err)
	}

	// Clean up the branch
	repo.DeleteBranch(testBranch)
}

func TestGetLastModificationTime(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// The repo has at least a README.md file
	modTime := getLastModificationTime(tmpDir)

	// Should be a recent timestamp (within the last minute)
	if time.Since(modTime) > time.Minute {
		t.Errorf("Last modification time is too old: %v", modTime)
	}

	// Should not be zero
	if modTime.IsZero() {
		t.Errorf("Last modification time should not be zero")
	}
}

func TestGetLastModificationTimeEmptyDir(t *testing.T) {
	// Create an empty temporary directory
	tmpDir, err := os.MkdirTemp("", "empty-dir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modTime := getLastModificationTime(tmpDir)

	// Should return current time for empty directory
	if time.Since(modTime) > time.Second {
		t.Errorf("Last modification time for empty dir should be recent: %v", modTime)
	}
}
