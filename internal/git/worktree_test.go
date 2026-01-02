package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestListWorktrees(t *testing.T) {
	fake := NewFakeGitExecutor()

	// Configure fake response for 'git worktree list --porcelain'
	fake.SetResponse("worktree list --porcelain", `worktree /home/user/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

`)

	// Configure fake response for 'git log -1 --format=%ct' (last commit timestamp)
	fake.SetResponse("log -1 --format=%ct", "1609459200")

	// Configure fake response for upstream branch check (no upstream)
	fake.SetError("rev-parse --abbrev-ref --symbolic-full-name @{u}", &exec.ExitError{})

	// Configure fake response for commit count (no upstream)
	fake.SetResponse("rev-list --count HEAD", "5")

	repo := &Repository{
		RootPath: "/home/user/repo",
		executor: fake,
	}

	// List worktrees
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	if len(worktrees) != 1 {
		t.Errorf("ListWorktrees() returned %d worktrees, expected 1", len(worktrees))
	}

	// Verify the main worktree
	mainWorktree := worktrees[0]
	if mainWorktree.Path != "/home/user/repo" {
		t.Errorf("Main worktree path = %v, want /home/user/repo", mainWorktree.Path)
	}

	if mainWorktree.Branch != "main" {
		t.Errorf("Main worktree branch = %v, want main", mainWorktree.Branch)
	}

	// Verify git commands were executed
	if len(fake.Commands) == 0 {
		t.Errorf("Expected git commands to be executed, but got none")
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	fake := NewFakeGitExecutor()

	// Configure fake response for creating worktree
	fake.SetResponse("worktree add -b test-branch /home/user/worktrees/test-worktree main", "")

	// Configure fake response for listing worktrees (after creation)
	fake.SetResponse("worktree list --porcelain", `worktree /home/user/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /home/user/worktrees/test-worktree
HEAD abcdef1234567890abcdef1234567890abcdef12
branch refs/heads/test-branch

`)

	// Configure fake response for removing worktree
	fake.SetResponse("worktree remove --force /home/user/worktrees/test-worktree", "")

	// Configure fake responses for enrichment
	fake.SetResponse("log -1 --format=%ct", "1609459200")
	fake.SetError("rev-parse --abbrev-ref --symbolic-full-name @{u}", &exec.ExitError{})
	fake.SetResponse("rev-list --count HEAD", "5")

	repo := &Repository{
		RootPath: "/home/user/repo",
		executor: fake,
	}

	// Create a new worktree with a new branch
	err := repo.CreateWorktreeWithNewBranch("/home/user/worktrees/test-worktree", "test-branch", "main")
	if err != nil {
		t.Fatalf("CreateWorktreeWithNewBranch() error = %v", err)
	}

	// Verify the command was executed
	found := false
	for _, cmd := range fake.Commands {
		if len(cmd) > 1 && cmd[1] == "worktree" && cmd[2] == "add" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'git worktree add' command to be executed")
	}

	// Remove the worktree
	if err := repo.RemoveWorktree("/home/user/worktrees/test-worktree"); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}

	// Verify the remove command was executed
	found = false
	for _, cmd := range fake.Commands {
		if len(cmd) > 1 && cmd[1] == "worktree" && cmd[2] == "remove" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'git worktree remove' command to be executed")
	}
}

func TestCreateWorktreeWithExistingBranch(t *testing.T) {
	fake := NewFakeGitExecutor()

	// Configure fake response for creating worktree with existing branch
	fake.SetResponse("worktree add /home/user/worktrees/existing-worktree existing-branch", "")

	repo := &Repository{
		RootPath: "/home/user/repo",
		executor: fake,
	}

	// Create a worktree for the existing branch
	err := repo.CreateWorktree("/home/user/worktrees/existing-worktree", "existing-branch")
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	// Verify the command was executed
	found := false
	for _, cmd := range fake.Commands {
		if len(cmd) > 3 && cmd[1] == "worktree" && cmd[2] == "add" && cmd[3] == "/home/user/worktrees/existing-worktree" && cmd[4] == "existing-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'git worktree add /home/user/worktrees/existing-worktree existing-branch' command to be executed")
	}
}

func TestGetWorktreeForBranch(t *testing.T) {
	fake := NewFakeGitExecutor()

	// Configure fake response for 'git worktree list --porcelain'
	fake.SetResponse("worktree list --porcelain", `worktree /home/user/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /home/user/worktrees/feature
HEAD abcdef1234567890abcdef1234567890abcdef12
branch refs/heads/feature-branch

`)

	// Configure fake responses for enrichment
	fake.SetResponse("log -1 --format=%ct", "1609459200")
	fake.SetError("rev-parse --abbrev-ref --symbolic-full-name @{u}", &exec.ExitError{})
	fake.SetResponse("rev-list --count HEAD", "5")

	repo := &Repository{
		RootPath: "/home/user/repo",
		executor: fake,
	}

	// Should find the main worktree
	wt, err := repo.GetWorktreeForBranch("main")
	if err != nil {
		t.Fatalf("GetWorktreeForBranch() error = %v", err)
	}

	if wt == nil {
		t.Errorf("Expected to find worktree for branch %q", "main")
	} else if wt.Branch != "main" {
		t.Errorf("Found worktree with branch %q, want %q", wt.Branch, "main")
	}

	// Should find the feature worktree
	fake.Reset()
	fake.SetResponse("worktree list --porcelain", `worktree /home/user/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /home/user/worktrees/feature
HEAD abcdef1234567890abcdef1234567890abcdef12
branch refs/heads/feature-branch

`)
	fake.SetResponse("log -1 --format=%ct", "1609459200")
	fake.SetError("rev-parse --abbrev-ref --symbolic-full-name @{u}", &exec.ExitError{})
	fake.SetResponse("rev-list --count HEAD", "5")

	wt, err = repo.GetWorktreeForBranch("feature-branch")
	if err != nil {
		t.Fatalf("GetWorktreeForBranch() error = %v", err)
	}

	if wt == nil {
		t.Errorf("Expected to find worktree for branch %q", "feature-branch")
	} else if wt.Branch != "feature-branch" {
		t.Errorf("Found worktree with branch %q, want %q", wt.Branch, "feature-branch")
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
	fake := NewFakeGitExecutor()

	// Configure fake responses for enrichment
	fake.SetResponse("log -1 --format=%ct", "1609459200")
	fake.SetError("rev-parse --abbrev-ref --symbolic-full-name @{u}", &exec.ExitError{})
	fake.SetResponse("rev-list --count HEAD", "5")

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

	worktrees, err := parseWorktreeList(porcelainOutput, fake)
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
	fake := NewFakeGitExecutor()

	// Configure fake response for 'git log -1 --format=%ct'
	// This is a Unix timestamp for 2021-01-01 00:00:00 UTC
	fake.SetResponse("log -1 --format=%ct", "1609459200")

	timestamp, err := getLastCommitTimestamp("/home/user/repo", fake)
	if err != nil {
		t.Fatalf("getLastCommitTimestamp() error = %v", err)
	}

	// Verify the timestamp matches what we configured
	expectedTime := time.Unix(1609459200, 0)
	if !timestamp.Equal(expectedTime) {
		t.Errorf("Last commit timestamp = %v, want %v", timestamp, expectedTime)
	}

	// Verify the command was executed
	found := false
	for _, cmd := range fake.Commands {
		if len(cmd) > 3 && cmd[1] == "log" && cmd[2] == "-1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'git log -1 --format=%%ct' command to be executed")
	}
}

func TestGetUnpushedCommitCount(t *testing.T) {
	t.Run("no upstream branch", func(t *testing.T) {
		fake := NewFakeGitExecutor()

		// Configure fake error for upstream branch check (no upstream)
		fake.SetError("rev-parse --abbrev-ref --symbolic-full-name @{u}", &exec.ExitError{})

		// Configure fake response for commit count (no upstream)
		fake.SetResponse("rev-list --count HEAD", "7")

		count, err := getUnpushedCommitCount("/home/user/repo", "main", fake)
		if err != nil {
			t.Fatalf("getUnpushedCommitCount() error = %v", err)
		}

		if count != 7 {
			t.Errorf("getUnpushedCommitCount() = %d, expected 7", count)
		}

		// Verify commands were executed
		if len(fake.Commands) < 2 {
			t.Errorf("Expected at least 2 commands to be executed, got %d", len(fake.Commands))
		}
	})

	t.Run("with upstream branch", func(t *testing.T) {
		fake := NewFakeGitExecutor()

		// Configure fake response for upstream branch check
		fake.SetResponse("rev-parse --abbrev-ref --symbolic-full-name @{u}", "origin/main")

		// Configure fake response for commits ahead of upstream
		fake.SetResponse("rev-list --count @{u}..HEAD", "3")

		count, err := getUnpushedCommitCount("/home/user/repo", "main", fake)
		if err != nil {
			t.Fatalf("getUnpushedCommitCount() error = %v", err)
		}

		if count != 3 {
			t.Errorf("getUnpushedCommitCount() = %d, expected 3", count)
		}

		// Verify commands were executed
		if len(fake.Commands) < 2 {
			t.Errorf("Expected at least 2 commands to be executed, got %d", len(fake.Commands))
		}
	})
}

func TestPruneWorktrees(t *testing.T) {
	fake := NewFakeGitExecutor()

	// Configure fake response for prune command
	fake.SetResponse("worktree prune", "")

	repo := &Repository{
		RootPath: "/home/user/repo",
		executor: fake,
	}

	// Prune should clean up the orphaned worktree
	if err := repo.PruneWorktrees(); err != nil {
		t.Fatalf("PruneWorktrees() error = %v", err)
	}

	// Verify the prune command was executed
	found := false
	for _, cmd := range fake.Commands {
		if len(cmd) > 2 && cmd[1] == "worktree" && cmd[2] == "prune" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'git worktree prune' command to be executed")
	}
}

func TestGetLastModificationTime(t *testing.T) {
	// Create a temporary directory with a file
	tmpDir, err := os.MkdirTemp("", "mod-time-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

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
