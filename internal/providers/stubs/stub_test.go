package stubs

import (
	"context"
	"testing"

	"github.com/kaeawc/auto-worktree/internal/providers"
)

func TestNewStubProvider(t *testing.T) {
	stub := NewStubProvider("TestProvider", "test")

	if stub.ProviderName != "TestProvider" {
		t.Errorf("ProviderName = %q, want %q", stub.ProviderName, "TestProvider")
	}

	if stub.ProviderTypeValue != "test" {
		t.Errorf("ProviderTypeValue = %q, want %q", stub.ProviderTypeValue, "test")
	}

	if stub.Name() != "TestProvider" {
		t.Errorf("Name() = %q, want %q", stub.Name(), "TestProvider")
	}

	if stub.ProviderType() != "test" {
		t.Errorf("ProviderType() = %q, want %q", stub.ProviderType(), "test")
	}
}

func TestStubProvider_AddAndListIssues(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	// Add issues
	stub.AddIssue(&providers.Issue{
		ID:       "1",
		Number:   1,
		Title:    "Issue 1",
		IsClosed: false,
	})
	stub.AddIssue(&providers.Issue{
		ID:       "2",
		Number:   2,
		Title:    "Issue 2",
		IsClosed: true,
	})

	// List issues
	issues, err := stub.ListIssues(ctx, 0)
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("ListIssues() returned %d issues, want 2", len(issues))
	}

	// Check limit
	issues, err = stub.ListIssues(ctx, 1)
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}

	if len(issues) != 1 {
		t.Errorf("ListIssues(limit=1) returned %d issues, want 1", len(issues))
	}
}

func TestStubProvider_GetIssue(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	issue := &providers.Issue{
		ID:       "1",
		Number:   1,
		Title:    "Test Issue",
		IsClosed: false,
	}
	stub.AddIssue(issue)

	// Get existing issue
	retrieved, err := stub.GetIssue(ctx, "1")
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}

	if retrieved.ID != issue.ID {
		t.Errorf("GetIssue() ID = %q, want %q", retrieved.ID, issue.ID)
	}

	// Get non-existent issue
	_, err = stub.GetIssue(ctx, "999")
	if err == nil {
		t.Errorf("GetIssue(non-existent) error = nil, want error")
	}
}

func TestStubProvider_IsIssueClosed(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	stub.AddIssue(&providers.Issue{
		ID:       "1",
		Number:   1,
		Title:    "Open Issue",
		IsClosed: false,
	})
	stub.AddIssue(&providers.Issue{
		ID:       "2",
		Number:   2,
		Title:    "Closed Issue",
		IsClosed: true,
	})

	// Check open issue
	closed, err := stub.IsIssueClosed(ctx, "1")
	if err != nil {
		t.Fatalf("IsIssueClosed() error = %v", err)
	}
	if closed {
		t.Errorf("IsIssueClosed(1) = true, want false")
	}

	// Check closed issue
	closed, err = stub.IsIssueClosed(ctx, "2")
	if err != nil {
		t.Fatalf("IsIssueClosed() error = %v", err)
	}
	if !closed {
		t.Errorf("IsIssueClosed(2) = false, want true")
	}
}

func TestStubProvider_CreateIssue(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	issue, err := stub.CreateIssue(ctx, "New Issue", "Description")
	if err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}

	if issue.Title != "New Issue" {
		t.Errorf("CreateIssue() Title = %q, want %q", issue.Title, "New Issue")
	}

	if issue.Body != "Description" {
		t.Errorf("CreateIssue() Body = %q, want %q", issue.Body, "Description")
	}

	if issue.State != "OPEN" {
		t.Errorf("CreateIssue() State = %q, want %q", issue.State, "OPEN")
	}

	if issue.IsClosed {
		t.Errorf("CreateIssue() IsClosed = true, want false")
	}
}

func TestStubProvider_ListPullRequests(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	// Add PRs
	stub.AddPullRequest(&providers.PullRequest{
		ID:       "1",
		Number:   1,
		Title:    "PR 1",
		IsMerged: false,
		IsClosed: false,
	})
	stub.AddPullRequest(&providers.PullRequest{
		ID:       "2",
		Number:   2,
		Title:    "PR 2",
		IsMerged: true,
		IsClosed: false,
	})

	// List PRs
	prs, err := stub.ListPullRequests(ctx, 0)
	if err != nil {
		t.Fatalf("ListPullRequests() error = %v", err)
	}

	if len(prs) != 2 {
		t.Errorf("ListPullRequests() returned %d PRs, want 2", len(prs))
	}
}

func TestStubProvider_IsPullRequestMerged(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	stub.AddPullRequest(&providers.PullRequest{
		ID:       "1",
		Number:   1,
		Title:    "Open PR",
		IsMerged: false,
		IsClosed: false,
	})
	stub.AddPullRequest(&providers.PullRequest{
		ID:       "2",
		Number:   2,
		Title:    "Merged PR",
		IsMerged: true,
		IsClosed: false,
	})

	// Check open PR
	merged, err := stub.IsPullRequestMerged(ctx, "1")
	if err != nil {
		t.Fatalf("IsPullRequestMerged() error = %v", err)
	}
	if merged {
		t.Errorf("IsPullRequestMerged(1) = true, want false")
	}

	// Check merged PR
	merged, err = stub.IsPullRequestMerged(ctx, "2")
	if err != nil {
		t.Fatalf("IsPullRequestMerged() error = %v", err)
	}
	if !merged {
		t.Errorf("IsPullRequestMerged(2) = false, want true")
	}
}

func TestStubProvider_SanitizeBranchName(t *testing.T) {
	stub := NewStubProvider("Test", "test")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Fix authentication bug",
			expected: "fix-authentication-bug",
		},
		{
			name:     "title with special chars",
			input:    "Fix: Bug in Auth!!",
			expected: "fix-bug-in-auth",
		},
		{
			name:     "long title",
			input:    "This is a very long title that exceeds the forty character limit and should be truncated",
			expected: "this-is-a-very-long-title-that-exceeds-t",
		},
		{
			name:     "title with underscores",
			input:    "fix_authentication_bug",
			expected: "fix-authentication-bug",
		},
		{
			name:     "leading/trailing spaces",
			input:    "  trim spaces  ",
			expected: "trim-spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stub.SanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStubProvider_GetBranchNameSuffix(t *testing.T) {
	stub := NewStubProvider("Test", "test")

	// Test with number
	issue := &providers.Issue{
		ID:     "1",
		Number: 123,
		Title:  "Test Issue",
	}
	suffix := stub.GetBranchNameSuffix(issue)
	if suffix != "123" {
		t.Errorf("GetBranchNameSuffix() = %q, want %q", suffix, "123")
	}

	// Test with key (JIRA)
	issue2 := &providers.Issue{
		ID:     "PROJ-123",
		Key:    "PROJ-123",
		Number: 0,
		Title:  "Test Issue",
	}
	suffix2 := stub.GetBranchNameSuffix(issue2)
	if suffix2 != "proj-123" {
		t.Errorf("GetBranchNameSuffix() = %q, want %q", suffix2, "proj-123")
	}
}

func TestStubProvider_SetError(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	stub.SetError("ListIssues", nil)

	_, err := stub.ListIssues(ctx, 0)
	if err != nil {
		t.Fatalf("ListIssues() error = %v (expected nil after SetError with nil)", err)
	}
}

func TestStubProvider_CallTracking(t *testing.T) {
	stub := NewStubProvider("Test", "test")
	ctx := context.Background()

	stub.AddIssue(&providers.Issue{ID: "1", Title: "Test"})

	// Call some methods
	stub.ListIssues(ctx, 0)
	stub.GetIssue(ctx, "1")
	stub.Name()
	stub.ListIssues(ctx, 0)

	// Check call counts
	if count := stub.GetCallCount("ListIssues"); count != 2 {
		t.Errorf("GetCallCount(ListIssues) = %d, want 2", count)
	}

	if count := stub.GetCallCount("GetIssue"); count != 1 {
		t.Errorf("GetCallCount(GetIssue) = %d, want 1", count)
	}

	if count := stub.GetCallCount("Name"); count != 1 {
		t.Errorf("GetCallCount(Name) = %d, want 1", count)
	}
}

func TestStubProvider_Reset(t *testing.T) {
	stub := NewStubProvider("Test", "test")

	stub.AddIssue(&providers.Issue{ID: "1", Title: "Test"})
	stub.AddPullRequest(&providers.PullRequest{ID: "1", Title: "Test"})

	if len(stub.Issues) != 1 || len(stub.PullRequests) != 1 {
		t.Fatalf("Issues and PRs not added correctly")
	}

	stub.Reset()

	if len(stub.Issues) != 0 || len(stub.PullRequests) != 0 || len(stub.Calls) != 0 {
		t.Errorf("Reset() did not clear data")
	}
}

func TestPreBuiltStubs(t *testing.T) {
	tests := []struct {
		name           string
		stubFactory    func() *StubProvider
		expectedName   string
		expectedType   string
		expectedIssues int
	}{
		{
			name:           "GitHub stub",
			stubFactory:    NewGitHubStub,
			expectedName:   "GitHub",
			expectedType:   "github",
			expectedIssues: 3,
		},
		{
			name:           "GitLab stub",
			stubFactory:    NewGitLabStub,
			expectedName:   "GitLab",
			expectedType:   "gitlab",
			expectedIssues: 1,
		},
		{
			name:           "JIRA stub",
			stubFactory:    NewJIRAStub,
			expectedName:   "JIRA",
			expectedType:   "jira",
			expectedIssues: 1,
		},
		{
			name:           "Linear stub",
			stubFactory:    NewLinearStub,
			expectedName:   "Linear",
			expectedType:   "linear",
			expectedIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stubFactory()

			if stub.Name() != tt.expectedName {
				t.Errorf("Name() = %q, want %q", stub.Name(), tt.expectedName)
			}

			if stub.ProviderType() != tt.expectedType {
				t.Errorf("ProviderType() = %q, want %q", stub.ProviderType(), tt.expectedType)
			}

			issues, err := stub.ListIssues(context.Background(), 0)
			if err != nil {
				t.Fatalf("ListIssues() error = %v", err)
			}

			if len(issues) != tt.expectedIssues {
				t.Errorf("ListIssues() returned %d issues, want %d", len(issues), tt.expectedIssues)
			}
		})
	}
}
