package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tmpDir
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create an initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to commit: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestIsGitRepository(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "valid git repository",
			path:     tmpDir,
			expected: true,
		},
		{
			name:     "non-existent path",
			path:     "/nonexistent/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitRepository(tt.path)
			if result != tt.expected {
				t.Errorf("IsGitRepository(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetRepositoryRoot(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		expectedDir string
	}{
		{
			name:        "from repository root",
			path:        tmpDir,
			wantErr:     false,
			expectedDir: tmpDir,
		},
		{
			name:        "from subdirectory",
			path:        subDir,
			wantErr:     false,
			expectedDir: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := GetRepositoryRoot(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepositoryRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare resolved paths to handle symlinks (macOS /var vs /private/var)
				expected, _ := filepath.EvalSymlinks(tt.expectedDir)
				actual, _ := filepath.EvalSymlinks(root)
				if actual != expected {
					t.Errorf("GetRepositoryRoot() = %v, want %v", root, tt.expectedDir)
				}
			}
		})
	}
}

func TestNewRepositoryFromPath(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Compare resolved paths to handle symlinks (macOS /var vs /private/var)
	expectedRoot, _ := filepath.EvalSymlinks(tmpDir)
	actualRoot, _ := filepath.EvalSymlinks(repo.RootPath)
	if actualRoot != expectedRoot {
		t.Errorf("RootPath = %v, want %v", repo.RootPath, tmpDir)
	}

	expectedFolder := filepath.Base(tmpDir)
	if repo.SourceFolder != expectedFolder {
		t.Errorf("SourceFolder = %v, want %v", repo.SourceFolder, expectedFolder)
	}

	homeDir, _ := os.UserHomeDir()
	expectedBase := filepath.Join(homeDir, "worktrees", expectedFolder)
	if repo.WorktreeBase != expectedBase {
		t.Errorf("WorktreeBase = %v, want %v", repo.WorktreeBase, expectedBase)
	}
}

func TestBranchExists(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Create a test branch
	cmd := exec.Command("git", "branch", "test-branch")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create test branch: %v", err)
	}

	tests := []struct {
		name       string
		branchName string
		expected   bool
	}{
		{
			name:       "existing branch",
			branchName: "test-branch",
			expected:   true,
		},
		{
			name:       "non-existent branch",
			branchName: "nonexistent-branch",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.BranchExists(tt.branchName)
			if result != tt.expected {
				t.Errorf("BranchExists(%q) = %v, want %v", tt.branchName, result, tt.expected)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	// Get the current branch (likely 'main' or 'master')
	branch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Should be a non-empty branch name
	if branch == "" {
		t.Errorf("GetCurrentBranch() returned empty string, expected branch name")
	}
}

func TestCreateAndDeleteBranch(t *testing.T) {
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

	// Create a new branch
	testBranch := "test-new-branch"
	if err := repo.CreateBranch(testBranch, currentBranch); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Verify branch exists
	if !repo.BranchExists(testBranch) {
		t.Errorf("Branch %q should exist after creation", testBranch)
	}

	// Delete the branch
	if err := repo.DeleteBranch(testBranch); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	// Verify branch no longer exists
	if repo.BranchExists(testBranch) {
		t.Errorf("Branch %q should not exist after deletion", testBranch)
	}
}

func TestGetDefaultBranch(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := NewRepositoryFromPath(tmpDir)
	if err != nil {
		t.Fatalf("NewRepositoryFromPath() error = %v", err)
	}

	defaultBranch, err := repo.GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	// Should return either 'main' or 'master' (or whatever the current branch is)
	if defaultBranch == "" {
		t.Errorf("GetDefaultBranch() returned empty string")
	}
}
