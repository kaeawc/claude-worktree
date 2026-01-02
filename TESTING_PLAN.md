# Comprehensive Testing Plan - Issue #88

This document outlines the testing strategy to achieve feature parity between the Bash (`aw.sh`) and Go (`auto-worktree`) implementations.

## Testing Pyramid

```
                    /\
                   /  \
                  / E2E \          - Full workflow tests
                 /--------\        - Cross-platform verification
                /          \
               /     INT    \      - Integration tests
              /--------------\    - Command chains
             /                \
            /       UNIT        \  - Function tests
           /__________________\   - Mock dependencies
```

## Test Coverage by Category

### 1. Unit Tests (Foundation Layer)

#### 1.1 Core Git Operations (internal/git/)

**Files to enhance:**
- `branch_test.go` - Branch name validation and generation
- `repository_test.go` - Repository detection
- `worktree_test.go` - Worktree operations
- `config_test.go` - Configuration management
- `hooks_test.go` - Hook execution
- `names_test.go` - Branch name sanitization

**Test Coverage:**
```
branch_test.go:
  ✅ Branch validation (existing)
  ✅ Branch name sanitization (existing)
  [ ] Duplicate branch detection
  [ ] Special character handling
  [ ] Unicode character handling
  [ ] Max length enforcement

repository_test.go:
  ✅ Remote URL parsing (existing)
  [ ] Repository info detection
  [ ] Main/master branch detection
  [ ] Detached HEAD handling

worktree_test.go:
  ✅ Worktree listing (existing)
  [ ] Worktree creation
  [ ] Worktree deletion
  [ ] Worktree path validation
  [ ] Pruning orphaned worktrees

config_test.go:
  ✅ Config read/write (existing)
  [ ] Provider-specific settings
  [ ] Default values
  [ ] Invalid config handling

hooks_test.go:
  ✅ Hook execution (existing)
  [ ] Hook discovery
  [ ] Hook error handling
  [ ] Custom hook support

names_test.go:
  ✅ Name generation (existing)
  ✅ Sanitization rules (existing)
  [ ] Issue title formatting
  [ ] PR title formatting
```

#### 1.2 UI Components (internal/ui/)

**Files to enhance:**
- `provider_menu_test.go` - Provider selection UI
- `theme_test.go` - Theme and colors
- New: `menu_test.go` - Main menu interactions
- New: `filter_list_test.go` - Interactive filtering

**Test Coverage:**
```
menu_test.go (new):
  [ ] Menu item display
  [ ] Menu selection
  [ ] Menu navigation (up/down)
  [ ] Menu routing to commands

provider_menu_test.go:
  ✅ Menu initialization (existing)
  [ ] Provider selection
  [ ] AI tool selection
  [ ] Reset functionality

filter_list_test.go (new):
  [ ] List rendering
  [ ] Text filtering
  [ ] Item selection
  [ ] Keyboard navigation
  [ ] Exit handling
```

#### 1.3 GitHub Integration (internal/github/)

**Files to test:**
- `client_test.go` - CLI client wrapper
- `issue_test.go` - Issue operations
- `pr_test.go` - PR operations
- `repository_test.go` - Repository detection

**Test Coverage:**
```
client_test.go:
  ✅ Client initialization (existing)
  ✅ Auth check (existing)
  [ ] CLI availability check
  [ ] Command execution
  [ ] Error handling

issue_test.go:
  ✅ Branch name generation (existing)
  ✅ Title sanitization (existing)
  [ ] Issue listing
  [ ] Issue details fetching
  [ ] Closed issue detection
  [ ] Merged PR detection
  [ ] Pagination handling

pr_test.go:
  ✅ Title sanitization (existing)
  [ ] PR listing
  [ ] PR details fetching
  [ ] Merged PR detection
```

#### 1.4 Provider Interfaces (new)

**New files:**
- `internal/providers/interface.go` - Provider interface definitions
- `internal/providers/github.go` - GitHub implementation (move from github/)
- `internal/providers/github_test.go` - GitHub provider tests
- `internal/providers/stubs/` - Provider stub implementations

**Test Coverage:**
```
Provider Interface Tests:
  [ ] List issues operation
  [ ] List PRs operation
  [ ] Create issue operation
  [ ] Get issue details
  [ ] Closed/merged status checks

GitHub Provider Tests:
  [ ] Issue listing with filtering
  [ ] PR listing with filtering
  [ ] Branch name generation
  [ ] Error handling (not authenticated)

Stub Implementations:
  [ ] In-memory issue storage
  [ ] In-memory PR storage
  [ ] Configurable responses
  [ ] Error simulation
```

#### 1.5 Session Management (internal/session/)

**Files to enhance:**
- `manager_test.go` - Session tracking
- `metadata_test.go` - Metadata storage

**Test Coverage:**
```
manager_test.go:
  ✅ Session creation (existing)
  ✅ Session metadata (existing)
  [ ] Last worktree tracking
  [ ] Session cleanup
  [ ] Tmux integration

metadata_test.go:
  ✅ Metadata structure (existing)
  [ ] Metadata persistence
  [ ] Metadata version compatibility
```

### 2. Integration Tests (Command Level)

**Location:** `internal/cmd/commands_test.go` (enhance existing)

#### 2.1 Core Command Tests

```
TestRunNew():
  [ ] Interactive branch name entry
  [ ] Default random name generation
  [ ] Branch validation
  [ ] Worktree creation
  [ ] Hook execution
  [ ] Error handling (branch exists)
  [ ] Terminal output verification

TestRunList():
  [ ] Empty repository (no worktrees)
  [ ] Multiple worktrees display
  [ ] Current worktree indicator
  [ ] Sorted output
  [ ] Timestamp display

TestRunResume():
  [ ] Last worktree recovery
  [ ] Directory navigation
  [ ] Tmux session detection
  [ ] No previous worktree error
  [ ] Main branch filtering

TestRunIssue():
  [ ] Interactive issue selection
  [ ] Direct issue number parsing
  [ ] GitHub provider flow
  [ ] Closed issue detection
  [ ] Worktree creation
  [ ] Existing worktree resume offer

TestRunCleanup():
  [ ] Worktree list display
  [ ] Merged branch detection
  [ ] Stale worktree detection
  [ ] Deletion confirmation
  [ ] Actual deletion execution

TestRunSettings():
  [ ] Provider selection
  [ ] Settings display
  [ ] Settings persistence
  [ ] Settings reset

TestRunRemove():
  [ ] CLI argument parsing
  [ ] Confirmation dialog
  [ ] Actual removal
  [ ] Branch cleanup

TestRunPrune():
  [ ] Orphaned worktree detection
  [ ] Silent execution
  [ ] Exit status
```

### 3. End-to-End Tests

**Location:** `tests/e2e/` (new directory structure)

#### 3.1 Workflow Tests

```
tests/e2e/workflows_test.go:
  [ ] Workflow: new → list → resume
  [ ] Workflow: issue → work → cleanup
  [ ] Workflow: pr → review → cleanup
  [ ] Workflow: new → settings → new (with config)
  [ ] Workflow: multiple worktrees lifecycle
  [ ] Workflow: interactive menu navigation
```

#### 3.2 Provider Workflow Tests

```
tests/e2e/github_workflow_test.go:
  [ ] GitHub issue workflow (select → create → cleanup)
  [ ] GitHub PR workflow (select → review → cleanup)
  [ ] GitHub provider switching
  [ ] GitHub configuration persistence

tests/e2e/gitlab_workflow_test.go (when implemented):
  [ ] GitLab issue workflow
  [ ] GitLab MR workflow
  [ ] GitLab configuration

tests/e2e/jira_workflow_test.go (when implemented):
  [ ] JIRA issue workflow
  [ ] JIRA project selection

tests/e2e/linear_workflow_test.go (when implemented):
  [ ] Linear issue workflow
  [ ] Linear team selection
```

### 4. Edge Case Tests

**Location:** `tests/edge_cases/` (new)

#### 4.1 Branch Name Edge Cases

```
tests/edge_cases/branch_names_test.go:
  [ ] Branch with special characters: !@#$%^&*()
  [ ] Branch with Unicode: café, 日本語
  [ ] Branch with consecutive spaces/dashes
  [ ] Branch with leading/trailing spaces
  [ ] Very long branch names (>200 chars)
  [ ] Empty branch name
  [ ] Branch with newlines
  [ ] Branch with paths: feature/sub/path
```

#### 4.2 Configuration Edge Cases

```
tests/edge_cases/config_test.go:
  [ ] Missing git config
  [ ] Invalid provider values
  [ ] Missing dependency (gh CLI)
  [ ] Invalid JIRA server URL
  [ ] Self-signed certificates (GitLab)
```

#### 4.3 Network Edge Cases

```
tests/edge_cases/network_test.go:
  [ ] Provider API timeout
  [ ] Rate limiting (429 response)
  [ ] Connection refused
  [ ] DNS resolution failure
  [ ] Authentication failure
  [ ] Repository not found
  [ ] No network connectivity
```

#### 4.4 State Edge Cases

```
tests/edge_cases/state_test.go:
  [ ] Orphaned git worktree metadata
  [ ] Stale tmux session
  [ ] Missing worktree directory
  [ ] Detached HEAD state
  [ ] Dirty working tree
  [ ] Concurrent worktree operations
```

### 5. Cross-Platform Tests

**Location:** `tests/cross_platform/` (new)

```
tests/cross_platform/darwin_test.go:
  [ ] macOS-specific paths (~/Library)
  [ ] macOS git hook execution
  [ ] macOS tmux integration
  [ ] macOS terminal detection

tests/cross_platform/linux_test.go:
  [ ] Linux-specific paths (~/.config)
  [ ] Linux git hook execution
  [ ] Linux tmux integration
  [ ] Linux terminal detection

tests/cross_platform/windows_test.go (if applicable):
  [ ] Windows path handling (backslashes)
  [ ] Windows git bash integration
  [ ] Windows line endings (CRLF)
  [ ] Windows terminal detection
```

### 6. Performance Benchmarks

**Location:** `tests/benchmarks/` (new)

```
tests/benchmarks/operations_test.go:
  [ ] Benchmark: ListWorktrees() with 100 worktrees
  [ ] Benchmark: CreateWorktree()
  [ ] Benchmark: BranchNameSanitization()
  [ ] Benchmark: GitConfig read/write
  [ ] Benchmark: Hook execution

tests/benchmarks/providers_test.go:
  [ ] Benchmark: List issues (GitHub)
  [ ] Benchmark: Issue details fetch
  [ ] Benchmark: Interactive filtering (1000 items)
  [ ] Benchmark: Memory usage (long session)
```

## Test Fixture and Utilities

### Test Fixtures

**Location:** `tests/fixtures/` (new)

```
fixtures/
├── git_repo/              # Sample git repository
├── issues.json            # Sample GitHub issues
├── prs.json               # Sample GitHub PRs
├── gitlab_issues.json     # Sample GitLab issues
└── jira_issues.json       # Sample JIRA issues
```

### Test Utilities

**Location:** `tests/testutil/` (new)

```
testutil/
├── git.go                 # Helper functions for git operations
├── providers.go           # Provider stub factory
├── fixtures.go            # Fixture loading utilities
├── temp_dir.go            # Temporary directory management
└── mock.go                # Mock implementations
```

## Provider Stub Implementations

### Stub Factory

**Location:** `internal/providers/stubs/factory.go`

```go
// Factory for creating stub providers with configurable responses
type StubProviderFactory struct {
    Issues map[int]Issue
    PRs    map[int]PR
    Errors map[string]error
}

func (f *StubProviderFactory) NewGitHub() Provider {
    return &StubGitHub{
        issues: f.Issues,
        prs:    f.PRs,
        errors: f.Errors,
    }
}
```

### Stub GitHub Provider

**Location:** `internal/providers/stubs/github.go`

```go
type StubGitHub struct {
    issues map[int]Issue
    prs    map[int]PR
    errors map[string]error
    calls  []MethodCall  // Track method calls for assertions
}

func (s *StubGitHub) ListIssues(ctx context.Context) ([]Issue, error) {
    s.calls = append(s.calls, MethodCall{Method: "ListIssues"})
    if err, ok := s.errors["ListIssues"]; ok {
        return nil, err
    }
    // Return issues from map
}

func (s *StubGitHub) GetMethodCallCount(method string) int {
    count := 0
    for _, call := range s.calls {
        if call.Method == method {
            count++
        }
    }
    return count
}
```

## Test Data Management

### Test Repositories

Create temporary git repositories for testing:

```go
type TestRepo struct {
    Dir       string
    GitDir    string
    WorktreeDir string
}

func NewTestRepo(t *testing.T) *TestRepo {
    // Create temp directory
    // Initialize git repo
    // Create sample branches
    // Return TestRepo
}
```

### Test Provider Data

Use JSON fixtures for consistent test data:

```go
func LoadGitHubIssues(t *testing.T) []Issue {
    data, err := ioutil.ReadFile("fixtures/issues.json")
    if err != nil {
        t.Fatalf("Failed to load fixtures: %v", err)
    }
    var issues []Issue
    json.Unmarshal(data, &issues)
    return issues
}
```

## Test Execution Strategy

### Running Tests by Category

```bash
# Unit tests only
go test ./internal/... -v

# Unit tests for specific package
go test ./internal/github/... -v

# Integration tests
go test ./internal/cmd/... -v

# E2E tests
go test ./tests/e2e/... -v

# All tests
go test ./... -v

# Tests with coverage
go test ./... -v -cover

# Tests with coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### CI/CD Integration

```yaml
# .github/workflows/test.yml structure
- name: Unit Tests
  run: go test ./internal/... -v

- name: Integration Tests
  run: go test ./internal/cmd/... -v

- name: E2E Tests (macOS)
  run: go test ./tests/e2e/... -v
  if: runner.os == 'macos'

- name: E2E Tests (Linux)
  run: go test ./tests/e2e/... -v
  if: runner.os == 'linux'

- name: Coverage Report
  run: |
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
```

## Success Metrics

### Coverage Goals
- Unit tests: 80%+ coverage
- Integration tests: All command paths covered
- E2E tests: All major workflows tested
- Edge cases: All known edge cases tested

### Quality Metrics
- No flaky tests (all tests deterministic)
- All tests pass on macOS, Linux, and Windows
- Performance benchmarks establish baselines
- User acceptance test sign-off

### Test Run Times
- Unit tests: < 5 seconds
- Integration tests: < 30 seconds
- E2E tests: < 2 minutes
- All tests: < 5 minutes

## Known Testing Challenges

### 1. Interactive UI Testing
- Bubbletea components need special handling
- Mock terminal input/output
- Test fixtures for UI state

### 2. Git Operations
- Requires real git repository
- Clean up after tests
- Handle OS-specific behavior

### 3. Provider Integration
- Can't rely on real APIs (auth, rate limiting)
- Stub implementations must be realistic
- Test error scenarios

### 4. Tmux Integration
- Requires tmux to be installed
- Skip tests if tmux unavailable
- Session cleanup between tests

### 5. Cross-Platform
- Path handling differences
- Line ending differences
- Terminal detection differences
- Skip platform-specific tests on other platforms

## Rollout Plan

### Week 1: Foundation
- Create test infrastructure (fixtures, utilities)
- Create provider stubs
- Enhance existing unit tests
- Target: 60% unit test coverage

### Week 2: Integration
- Create command integration tests
- Test each command thoroughly
- Target: 80% command coverage

### Week 3: E2E and Edge Cases
- Create E2E workflow tests
- Create edge case tests
- Target: All workflows tested

### Week 4: Cross-Platform and Polish
- Create cross-platform tests
- Create performance benchmarks
- Create UAT guide
- Final validation

## Acceptance Criteria

✅ All unit tests passing
✅ All integration tests passing
✅ All E2E tests passing
✅ 80%+ code coverage
✅ Cross-platform compatibility verified
✅ Performance benchmarks established
✅ No flaky tests
✅ User acceptance test guide created
