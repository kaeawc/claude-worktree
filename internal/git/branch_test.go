package git

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCurrentBranchInWorktree(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	branch, err := GetCurrentBranchInWorktree(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentBranchInWorktree() error = %v", err)
	}

	if branch == "" {
		t.Errorf("GetCurrentBranchInWorktree() returned empty string")
	}
}

func TestGetCurrentBranchInWorktreeDetached(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a detached HEAD state
	cmd := exec.Command("git", "checkout", "--detach", "HEAD")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create detached HEAD: %v", err)
	}

	branch, err := GetCurrentBranchInWorktree(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentBranchInWorktree() error = %v", err)
	}

	if branch != "" {
		t.Errorf("GetCurrentBranchInWorktree() = %q, expected empty string for detached HEAD", branch)
	}
}

func TestGetUpstreamBranch(t *testing.T) {
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

	// Without a remote, this should fail
	_, err = GetUpstreamBranch(tmpDir, currentBranch)
	if err == nil {
		t.Errorf("GetUpstreamBranch() should fail without upstream, but succeeded")
	}

	// Error message should indicate no upstream
	if !strings.Contains(err.Error(), "no upstream") {
		t.Errorf("GetUpstreamBranch() error = %v, want error about no upstream", err)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "feature",
			expected: "feature",
		},
		{
			name:     "uppercase to lowercase",
			input:    "FEATURE",
			expected: "feature",
		},
		{
			name:     "mixed case with special chars",
			input:    "Feature/PROJ-123-Fix_Bug",
			expected: "feature-proj-123-fix-bug",
		},
		{
			name:     "with slashes and underscores",
			input:    "work/123-My-New-Feature!",
			expected: "work-123-my-new-feature",
		},
		{
			name:     "multiple consecutive dashes",
			input:    "feature---name",
			expected: "feature-name",
		},
		{
			name:     "leading and trailing dashes",
			input:    "-feature-name-",
			expected: "feature-name",
		},
		{
			name:     "special characters",
			input:    "fix/bug#123@user",
			expected: "fix-bug-123-user",
		},
		{
			name:     "spaces",
			input:    "my feature branch",
			expected: "my-feature-branch",
		},
		{
			name:     "unicode and special chars",
			input:    "feature-émoji-☺️-test",
			expected: "feature-moji-test",
		},
		{
			name:     "numbers only",
			input:    "123",
			expected: "123",
		},
		{
			name:     "alphanumeric",
			input:    "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "complex real-world example",
			input:    "Work/JIRA-1234_Fix_Critical_Bug!!!",
			expected: "work-jira-1234-fix-critical-bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeBranchNameIntegration(t *testing.T) {
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

	// Test creating a branch with a sanitized name
	unsanitizedName := "Feature/PROJ-123_Fix Bug!"
	sanitizedName := SanitizeBranchName(unsanitizedName)

	if err := repo.CreateBranch(sanitizedName, currentBranch); err != nil {
		t.Fatalf("CreateBranch(%q) error = %v", sanitizedName, err)
	}
	defer repo.DeleteBranch(sanitizedName)

	// Verify the branch was created
	if !repo.BranchExists(sanitizedName) {
		t.Errorf("Branch %q should exist after creation", sanitizedName)
	}

	// Verify we can create a worktree with the sanitized name
	worktreePath := filepath.Join(tmpDir, "..", "sanitized-test-worktree")
	err = repo.CreateWorktree(worktreePath, sanitizedName)
	if err != nil {
		t.Fatalf("CreateWorktree() with sanitized branch error = %v", err)
	}
	defer repo.RemoveWorktree(worktreePath)
}
