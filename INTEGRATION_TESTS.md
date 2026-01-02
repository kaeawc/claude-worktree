# Integration Test Implementation Guide

This guide provides detailed instructions for implementing integration tests for each command.

## Test Infrastructure

### Test Repository Setup

Each integration test should create a temporary git repository to work with:

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRepo represents a temporary git repository for testing
type TestRepo struct {
	Dir     string
	GitDir  string
	TempDir string
}

// NewTestRepo creates a temporary git repository for testing
func NewTestRepo(t *testing.T) *TestRepo {
	tempDir, err := os.MkdirTemp("", "auto-worktree-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repoDir := filepath.Join(tempDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git (required for commits)
	cmds := []*exec.Cmd{
		{Cmd: "git", Args: []string{"config", "user.email", "test@example.com"}},
		{Cmd: "git", Args: []string{"config", "user.name", "Test User"}},
	}

	for _, c := range cmds {
		cmd := exec.Command(c.Cmd, c.Args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to configure git: %v", err)
		}
	}

	return &TestRepo{
		Dir:     repoDir,
		GitDir:  filepath.Join(repoDir, ".git"),
		TempDir: tempDir,
	}
}

// Cleanup removes the temporary repository
func (tr *TestRepo) Cleanup() error {
	return os.RemoveAll(tr.TempDir)
}

// CreateBranch creates a new branch in the repository
func (tr *TestRepo) CreateBranch(t *testing.T, branchName string) {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = tr.Dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
}

// CreateWorktree creates a new worktree
func (tr *TestRepo) CreateWorktree(t *testing.T, path, branch string) {
	cmd := exec.Command("git", "worktree", "add", path, "-b", branch)
	cmd.Dir = tr.Dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}
}

// ListWorktrees lists all worktrees
func (tr *TestRepo) ListWorktrees(t *testing.T) []string {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = tr.Dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}
	// Parse output...
	return []string{}
}

// GetBranches returns all branches
func (tr *TestRepo) GetBranches(t *testing.T) []string {
	cmd := exec.Command("git", "branch")
	cmd.Dir = tr.Dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get branches: %v", err)
	}
	// Parse output...
	return []string{}
}
```

## Command Integration Tests

### 1. RunNew Integration Tests

**File:** `internal/cmd/commands_test.go` (enhance)

```go
func TestRunNew(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Test 1: Basic new worktree creation
	t.Run("basic worktree creation", func(t *testing.T) {
		// Simulate user input
		// Call RunNew() in the test repo
		// Verify:
		// - Worktree was created
		// - Branch was created
		// - Branch name is valid
		// - Hook was executed (if configured)
	})

	// Test 2: Default random name generation
	t.Run("default random name", func(t *testing.T) {
		// Call RunNew() without branch name input
		// Verify:
		// - Random name was generated
		// - Name follows valid format
		// - Worktree created with random name
	})

	// Test 3: Branch name validation
	t.Run("branch name validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			branchName  string
			shouldFail  bool
		}{
			{"valid name", "feature/my-feature", false},
			{"name with spaces", "feature with spaces", true},
			{"name with special chars", "feature!@#$", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test with tc.branchName
				// Verify expectation: tc.shouldFail
			})
		}
	})

	// Test 4: Duplicate branch detection
	t.Run("duplicate branch detection", func(t *testing.T) {
		// Create a branch
		// Try to create same branch again
		// Verify error is returned
	})

	// Test 5: Hook execution
	t.Run("post-create hook execution", func(t *testing.T) {
		// Create hook script
		// Enable hooks in config
		// Create worktree
		// Verify hook was executed
	})

	// Test 6: Hook failure handling
	t.Run("hook failure handling", func(t *testing.T) {
		// Create failing hook
		// Create worktree
		// Verify behavior based on fail-on-hook-error config
	})
}
```

### 2. RunList Integration Tests

**File:** `internal/cmd/commands_test.go` (enhance)

```go
func TestRunList(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("list empty repository", func(t *testing.T) {
		// Call RunList()
		// Verify empty or minimal output
	})

	t.Run("list multiple worktrees", func(t *testing.T) {
		// Create 3 worktrees
		// Call RunList()
		// Verify all appear in output
		// Verify current worktree is marked
	})

	t.Run("current worktree indicator", func(t *testing.T) {
		// Create 2 worktrees
		// List should show current one with indicator
	})

	t.Run("detached head handling", func(t *testing.T) {
		// Create worktree in detached state
		// Verify it's listed and marked as detached
	})

	t.Run("sorted output", func(t *testing.T) {
		// Create worktrees in random order
		// Verify output is sorted consistently
	})
}
```

### 3. RunIssue Integration Tests

**File:** `internal/cmd/commands_test.go` (enhance)

```go
func TestRunIssue(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Use stub providers for testing
	githubStub := stubs.NewGitHubStub()

	t.Run("direct issue mode", func(t *testing.T) {
		// Test: aw issue 123
		// Verify:
		// - Issue is fetched
		// - Worktree is created with correct branch name
		// - Branch name format: work/123-fix-authentication-bug
	})

	t.Run("closed issue detection", func(t *testing.T) {
		// Test issue that is already closed
		// Verify user is warned
	})

	t.Run("existing worktree resume", func(t *testing.T) {
		// Create worktree for issue
		// Try to work on same issue again
		// Verify offer to resume is shown
	})

	t.Run("branch name generation", func(t *testing.T) {
		testCases := []struct {
			title         string
			expectedSuffix string
		}{
			{"Fix authentication bug", "fix-authentication-bug"},
			{"Add dark mode support", "add-dark-mode-support"},
			{"Special chars: !@#$", "special-chars"},
		}

		for _, tc := range testCases {
			// Verify branch name is generated correctly
		}
	})

	t.Run("provider switching", func(t *testing.T) {
		// Test GitHub provider
		// Switch to GitLab (when implemented)
		// Verify provider is switched
	})
}
```

### 4. RunCleanup Integration Tests

**File:** `internal/cmd/commands_test.go` (enhance)

```go
func TestRunCleanup(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("cleanup merged worktree", func(t *testing.T) {
		// Create worktree
		// Mark branch as merged (simulate)
		// Run cleanup
		// Verify worktree is deleted
		// Verify branch is deleted
	})

	t.Run("cleanup stale worktree", func(t *testing.T) {
		// Create old worktree (modify timestamp)
		// Run cleanup
		// Verify it's offered for cleanup
	})

	t.Run("cleanup confirmation", func(t *testing.T) {
		// Create worktree
		// Run cleanup with confirmation
		// Verify user can decline
		// Verify actual deletion on confirm
	})

	t.Run("cleanup with pruning", func(t *testing.T) {
		// Create orphaned worktree metadata
		// Run cleanup
		// Verify pruning is done
	})
}
```

### 5. RunSettings Integration Tests

**File:** `internal/cmd/commands_test.go` (enhance)

```go
func TestRunSettings(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("set issue provider", func(t *testing.T) {
		// Select GitHub provider
		// Verify git config is updated
		// Verify next issue command uses GitHub
	})

	t.Run("get settings", func(t *testing.T) {
		// Set some config
		// Get settings
		// Verify all values are shown
	})

	t.Run("reset settings", func(t *testing.T) {
		// Set multiple config values
		// Reset settings
		// Verify all are cleared
	})

	t.Run("provider-specific settings", func(t *testing.T) {
		// Test JIRA: set server URL and project
		// Test GitLab: set server URL and project
		// Verify each provider's specific settings
	})
}
```

## Provider Integration Tests

### GitHub Provider Tests

**File:** `internal/github/client_integration_test.go` (new)

```go
func TestGitHubClientIntegration(t *testing.T) {
	// Skip if gh CLI not available
	if !isGHAvailable() {
		t.Skip("gh CLI not available")
	}

	// Create stub to avoid real API calls
	stub := stubs.NewGitHubStub()

	t.Run("list issues", func(t *testing.T) {
		// Create client with stub
		// List issues
		// Verify correct number returned
	})

	t.Run("get issue details", func(t *testing.T) {
		// Get specific issue
		// Verify all fields are populated
	})

	t.Run("closed issue detection", func(t *testing.T) {
		// Test closed issue in stub
		// Verify IsIssueClosed returns true
	})

	t.Run("branch name generation", func(t *testing.T) {
		// Test various titles
		// Verify sanitization rules applied
	})
}
```

## E2E Workflow Tests

### Complete Workflow Tests

**File:** `tests/e2e/workflows_test.go` (new)

```go
func TestWorkflow_NewToCleanup(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Workflow: new → list → cleanup

	// 1. Create new worktree
	// Verify created

	// 2. List worktrees
	// Verify it appears in list

	// 3. Cleanup worktree
	// Verify it's removed

	// Verify directory is cleaned up
}

func TestWorkflow_IssueToCleanup(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Workflow: issue selection → work on issue → cleanup

	// 1. Select GitHub issue
	// 2. Verify worktree created
	// 3. Simulate work (create file, commit)
	// 4. Cleanup
	// 5. Verify all cleaned up
}

func TestWorkflow_MultipleWorktrees(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create 3 worktrees
	// List them
	// Resume different ones
	// Cleanup some
	// Verify state is consistent
}
```

## Edge Case Tests

### Branch Name Edge Cases

**File:** `tests/edge_cases/branch_names_test.go` (new)

```go
func TestBranchNameEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		shouldErr bool
	}{
		// Unicode
		{"unicode characters", "café", "caf", false},
		{"chinese characters", "修复漏洞", "", true},

		// Length
		{"very long name", strings.Repeat("a", 500), strings.Repeat("a", 40), false},
		{"empty string", "", "", true},

		// Special chars
		{"paths", "feature/sub/path", "feature-sub-path", false},
		{"dots", "v1.0.0", "v1-0-0", false},

		// Spaces
		{"multiple spaces", "fix   bug", "fix-bug", false},
		{"tabs", "fix\tbug", "fix-bug", false},
		{"newlines", "fix\nbug", "fix-bug", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test sanitization
			// Verify expected result or error
		})
	}
}
```

### Configuration Edge Cases

**File:** `tests/edge_cases/config_test.go` (new)

```go
func TestConfigEdgeCases(t *testing.T) {
	t.Run("missing git config", func(t *testing.T) {
		// Config not set
		// Operation should use defaults
	})

	t.Run("invalid provider", func(t *testing.T) {
		// Set invalid provider value
		// Verify error or fallback
	})

	t.Run("missing gh CLI", func(t *testing.T) {
		// Temporarily hide gh CLI
		// Verify proper error message
	})

	t.Run("authentication failure", func(t *testing.T) {
		// Test with invalid auth
		// Verify error handling
	})
}
```

## Performance Benchmarks

**File:** `tests/benchmarks/operations_test.go` (new)

```go
func BenchmarkListWorktrees(b *testing.B) {
	repo := NewTestRepo(&testing.T{})
	defer repo.Cleanup()

	// Create 100 worktrees
	for i := 0; i < 100; i++ {
		// Create worktree
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run ListWorktrees
	}
}

func BenchmarkBranchNameSanitization(b *testing.B) {
	titles := []string{
		"Fix authentication bug",
		"Add dark mode support",
		"Refactor database queries",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, title := range titles {
			// Sanitize title
		}
	}
}
```

## Test Utilities

Create helpers to make tests easier to write:

**File:** `tests/testutil/git_helpers.go` (new)

```go
// GitHelper provides helper functions for git operations in tests
type GitHelper struct {
	repoDir string
	t       *testing.T
}

// NewGitHelper creates a new git helper
func NewGitHelper(repoDir string, t *testing.T) *GitHelper {
	return &GitHelper{repoDir: repoDir, t: t}
}

// CreateBranch creates a branch
func (gh *GitHelper) CreateBranch(name string) {
	cmd := exec.Command("git", "checkout", "-b", name)
	cmd.Dir = gh.repoDir
	if err := cmd.Run(); err != nil {
		gh.t.Fatalf("Failed to create branch: %v", err)
	}
}

// CreateWorktree creates a worktree
func (gh *GitHelper) CreateWorktree(path, branch string) {
	cmd := exec.Command("git", "worktree", "add", path, "-b", branch)
	cmd.Dir = gh.repoDir
	if err := cmd.Run(); err != nil {
		gh.t.Fatalf("Failed to create worktree: %v", err)
	}
}

// ListWorktrees lists all worktrees
func (gh *GitHelper) ListWorktrees() []string {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = gh.repoDir
	output, err := cmd.Output()
	if err != nil {
		gh.t.Fatalf("Failed to list worktrees: %v", err)
	}

	var worktrees []string
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			worktrees = append(worktrees, parts[0])
		}
	}
	return worktrees
}

// GetBranches gets all branches
func (gh *GitHelper) GetBranches() []string {
	cmd := exec.Command("git", "branch", "--list")
	cmd.Dir = gh.repoDir
	output, err := cmd.Output()
	if err != nil {
		gh.t.Fatalf("Failed to get branches: %v", err)
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branches = append(branches, strings.TrimPrefix(line, "* "))
	}
	return branches
}
```

## Test Execution

Run tests by category:

```bash
# Unit tests
go test ./internal/... -v

# Unit tests with coverage
go test ./internal/... -v -cover

# Integration tests
go test ./internal/cmd/... -v

# E2E tests
go test ./tests/e2e/... -v

# Edge case tests
go test ./tests/edge_cases/... -v

# Benchmarks
go test ./tests/benchmarks/... -bench=. -benchmem

# All tests
go test ./... -v

# With coverage report
go test ./... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Continuous Integration

Tests should run on:
- macOS (via GitHub Actions)
- Linux (via GitHub Actions)
- Windows (via GitHub Actions) if applicable

Skip tests that require:
- Actual GitHub authentication
- System-specific features (tmux, shell paths)
- External services (with proper mock alternatives)

## Test Data

Keep test data organized:

```
tests/
├── fixtures/
│   ├── github_issues.json
│   ├── gitlab_issues.json
│   ├── jira_issues.json
│   └── linear_issues.json
├── testutil/
│   └── helpers.go
├── e2e/
│   └── workflows_test.go
├── edge_cases/
│   └── *_test.go
└── benchmarks/
    └── operations_test.go
```

## Success Criteria

- All integration tests pass on all platforms
- All command paths are tested
- Edge cases are covered
- Benchmarks provide baseline performance
- Test execution time < 5 minutes
- Code coverage > 80%
