package git

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetCurrentBranchInWorktree(t *testing.T) {
	fake := NewFakeGitExecutor()
	worktreePath := "/fake/worktree"

	// Configure fake to return a branch name
	fake.SetResponse("rev-parse --abbrev-ref HEAD", "feature-branch")

	branch, err := getCurrentBranchInWorktree(worktreePath, fake)
	if err != nil {
		t.Fatalf("GetCurrentBranchInWorktree() error = %v", err)
	}

	if branch != "feature-branch" {
		t.Errorf("GetCurrentBranchInWorktree() = %q, want %q", branch, "feature-branch")
	}

	// Verify the command was executed
	if len(fake.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(fake.Commands))
	}

	expectedCmd := []string{"[in:" + worktreePath + "]", "rev-parse", "--abbrev-ref", "HEAD"}
	if !equalSlices(fake.Commands[0], expectedCmd) {
		t.Errorf("Command = %v, want %v", fake.Commands[0], expectedCmd)
	}
}

func TestGetCurrentBranchInWorktreeDetached(t *testing.T) {
	fake := NewFakeGitExecutor()
	worktreePath := "/fake/worktree"

	// Configure fake to return "HEAD" indicating detached state
	fake.SetResponse("rev-parse --abbrev-ref HEAD", "HEAD")

	branch, err := getCurrentBranchInWorktree(worktreePath, fake)
	if err != nil {
		t.Fatalf("GetCurrentBranchInWorktree() error = %v", err)
	}

	if branch != "" {
		t.Errorf("GetCurrentBranchInWorktree() = %q, expected empty string for detached HEAD", branch)
	}
}

func TestGetUpstreamBranch(t *testing.T) {
	fake := NewFakeGitExecutor()
	worktreePath := "/fake/worktree"
	branchName := "feature-branch"

	// Configure fake to return error (no upstream configured)
	fake.SetError("rev-parse --abbrev-ref --symbolic-full-name "+branchName+"@{u}", fmt.Errorf("no upstream"))

	_, err := getUpstreamBranch(worktreePath, branchName, fake)
	if err == nil {
		t.Errorf("GetUpstreamBranch() should fail without upstream, but succeeded")
	}

	// Error message should indicate no upstream
	if !strings.Contains(err.Error(), "no upstream") {
		t.Errorf("GetUpstreamBranch() error = %v, want error about no upstream", err)
	}

	// Verify the command was executed
	if len(fake.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(fake.Commands))
	}
}

func TestGetUpstreamBranchSuccess(t *testing.T) {
	fake := NewFakeGitExecutor()
	worktreePath := "/fake/worktree"
	branchName := "feature-branch"

	// Configure fake to return upstream branch
	fake.SetResponse("rev-parse --abbrev-ref --symbolic-full-name "+branchName+"@{u}", "origin/feature-branch")

	upstream, err := getUpstreamBranch(worktreePath, branchName, fake)
	if err != nil {
		t.Fatalf("GetUpstreamBranch() error = %v", err)
	}

	if upstream != "origin/feature-branch" {
		t.Errorf("GetUpstreamBranch() = %q, want %q", upstream, "origin/feature-branch")
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
	fake := NewFakeGitExecutor()
	fakeFS := NewFakeFileSystem()

	// Setup fake filesystem for repository
	repoPath := "/fake/repo"
	fakeFS.Dirs[repoPath] = true
	fakeFS.HomeDir = "/home/testuser"

	// Setup fake executor responses
	fake.SetResponse("rev-parse --git-dir", ".git")
	fake.SetResponse("rev-parse --show-toplevel", repoPath)
	fake.SetResponse("rev-parse --abbrev-ref HEAD", "main")

	repo, err := NewRepositoryFromPathWithDeps(repoPath, fake, fakeFS)
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

	if sanitizedName != "feature-proj-123-fix-bug" {
		t.Errorf("SanitizeBranchName() = %q, want %q", sanitizedName, "feature-proj-123-fix-bug")
	}

	// Configure fake to succeed on branch creation
	fake.SetResponse("branch "+sanitizedName+" "+currentBranch, "")

	if err := repo.CreateBranch(sanitizedName, currentBranch); err != nil {
		t.Fatalf("CreateBranch(%q) error = %v", sanitizedName, err)
	}

	// Verify the branch creation command was executed
	foundBranchCmd := false
	for _, cmd := range fake.Commands {
		cmdStr := strings.Join(cmd, " ")
		if strings.Contains(cmdStr, "branch "+sanitizedName) {
			foundBranchCmd = true
			break
		}
	}
	if !foundBranchCmd {
		t.Errorf("Expected branch creation command to be executed")
	}
}

func TestIsBranchMergedInto(t *testing.T) {
	fake := NewFakeGitExecutor()
	repoPath := "/fake/repo"

	tests := []struct {
		name         string
		branchName   string
		targetBranch string
		gitOutput    string
		gitError     error
		wantMerged   bool
	}{
		{
			name:         "branch is merged",
			branchName:   "feature-branch",
			targetBranch: "main",
			gitOutput:    "  feature-branch",
			gitError:     nil,
			wantMerged:   true,
		},
		{
			name:         "branch is not merged - empty output",
			branchName:   "feature-branch",
			targetBranch: "main",
			gitOutput:    "",
			gitError:     nil,
			wantMerged:   false,
		},
		{
			name:         "branch is not merged - different branch",
			branchName:   "feature-branch",
			targetBranch: "main",
			gitOutput:    "  other-branch",
			gitError:     nil,
			wantMerged:   false,
		},
		{
			name:         "git command fails",
			branchName:   "feature-branch",
			targetBranch: "main",
			gitOutput:    "",
			gitError:     fmt.Errorf("branch does not exist"),
			wantMerged:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.Reset()

			cmdKey := "branch --merged " + tt.targetBranch + " --list " + tt.branchName
			if tt.gitError != nil {
				fake.SetError(cmdKey, tt.gitError)
			} else {
				fake.SetResponse(cmdKey, tt.gitOutput)
			}

			merged, err := isBranchMergedInto(repoPath, tt.branchName, tt.targetBranch, fake)
			if err != nil {
				t.Fatalf("isBranchMergedInto() error = %v", err)
			}

			if merged != tt.wantMerged {
				t.Errorf("isBranchMergedInto() = %v, want %v", merged, tt.wantMerged)
			}

			// Verify the command was executed
			if len(fake.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(fake.Commands))
			}
		})
	}
}

func TestGetMergeBase(t *testing.T) {
	fake := NewFakeGitExecutor()
	repoPath := "/fake/repo"

	tests := []struct {
		name      string
		branch1   string
		branch2   string
		gitOutput string
		gitError  error
		wantHash  string
		wantErr   bool
	}{
		{
			name:      "successful merge base",
			branch1:   "feature-branch",
			branch2:   "main",
			gitOutput: "abc123def456",
			gitError:  nil,
			wantHash:  "abc123def456",
			wantErr:   false,
		},
		{
			name:      "git command fails",
			branch1:   "feature-branch",
			branch2:   "main",
			gitOutput: "",
			gitError:  fmt.Errorf("no merge base"),
			wantHash:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.Reset()

			cmdKey := "merge-base " + tt.branch1 + " " + tt.branch2
			if tt.gitError != nil {
				fake.SetError(cmdKey, tt.gitError)
			} else {
				fake.SetResponse(cmdKey, tt.gitOutput)
			}

			hash, err := getMergeBase(repoPath, tt.branch1, tt.branch2, fake)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getMergeBase() error = %v, wantErr %v", err, tt.wantErr)
			}

			if hash != tt.wantHash {
				t.Errorf("getMergeBase() = %q, want %q", hash, tt.wantHash)
			}

			// Verify the command was executed
			if len(fake.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(fake.Commands))
			}
		})
	}
}

// Helper function to compare slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
