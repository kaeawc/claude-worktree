// Package stubs provides stub implementations of providers for testing.
package stubs

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/providers"
)

// StubProvider is a stub implementation of the Provider interface for testing.
type StubProvider struct {
	// Name of the provider (e.g., "GitHub", "GitLab")
	ProviderName string
	// Type of the provider (e.g., "github", "gitlab")
	ProviderTypeValue string
	// Issues stored by ID
	Issues map[string]*providers.Issue
	// PRs stored by ID
	PullRequests map[string]*providers.PullRequest
	// Errors to return for specific operations
	Errors map[string]error
	// Method call tracking for assertions
	Calls []MethodCall
	// Config for the provider
	Config *providers.Config
}

// MethodCall tracks a method call for assertion purposes.
type MethodCall struct {
	Method string
	Args   interface{}
}

// NewStubProvider creates a new stub provider with no data.
func NewStubProvider(name, providerType string) *StubProvider {
	return &StubProvider{
		ProviderName:      name,
		ProviderTypeValue: providerType,
		Issues:            make(map[string]*providers.Issue),
		PullRequests:      make(map[string]*providers.PullRequest),
		Errors:            make(map[string]error),
		Calls:             []MethodCall{},
		Config:            &providers.Config{},
	}
}

// NewGitHubStub creates a GitHub stub provider with sample data.
func NewGitHubStub() *StubProvider {
	stub := NewStubProvider("GitHub", "github")

	// Add sample issues
	stub.AddIssue(&providers.Issue{
		ID:        "1",
		Number:    123,
		Title:     "Fix authentication bug",
		Body:      "Users are unable to log in with GitHub",
		URL:       "https://github.com/owner/repo/issues/123",
		State:     "OPEN",
		Labels:    []string{"bug", "authentication"},
		Author:    "alice",
		CreatedAt: "2025-01-01T10:00:00Z",
		UpdatedAt: "2025-01-02T12:00:00Z",
		Assignee:  "bob",
		IsClosed:  false,
	})

	stub.AddIssue(&providers.Issue{
		ID:        "2",
		Number:    124,
		Title:     "Add dark mode support",
		Body:      "Implement dark mode theme for the application",
		URL:       "https://github.com/owner/repo/issues/124",
		State:     "OPEN",
		Labels:    []string{"enhancement", "ui"},
		Author:    "bob",
		CreatedAt: "2025-01-01T11:00:00Z",
		UpdatedAt: "2025-01-02T11:00:00Z",
		Assignee:  "",
		IsClosed:  false,
	})

	stub.AddIssue(&providers.Issue{
		ID:        "3",
		Number:    125,
		Title:     "Refactor database queries",
		Body:      "Optimize slow queries in the database layer",
		URL:       "https://github.com/owner/repo/issues/125",
		State:     "CLOSED",
		Labels:    []string{"performance"},
		Author:    "charlie",
		CreatedAt: "2024-12-15T10:00:00Z",
		UpdatedAt: "2025-01-02T10:00:00Z",
		Assignee:  "alice",
		IsClosed:  true,
	})

	// Add sample PRs
	stub.AddPullRequest(&providers.PullRequest{
		ID:                 "1",
		Number:             456,
		Title:              "Fix: Handle nil pointer in parser",
		Body:               "This PR fixes a nil pointer dereference in the parser",
		URL:                "https://github.com/owner/repo/pull/456",
		State:              "OPEN",
		HeadBranch:         "fix/nil-pointer",
		BaseBranch:         "main",
		Labels:             []string{"bugfix"},
		Author:             "alice",
		CreatedAt:          "2025-01-02T09:00:00Z",
		UpdatedAt:          "2025-01-02T14:00:00Z",
		IsMerged:           false,
		IsClosed:           false,
		ReviewersRequested: []string{"bob", "charlie"},
		Approvals:          []string{},
	})

	stub.AddPullRequest(&providers.PullRequest{
		ID:                 "2",
		Number:             457,
		Title:              "Feature: Add caching layer",
		Body:               "Implements Redis-based caching",
		URL:                "https://github.com/owner/repo/pull/457",
		State:              "MERGED",
		HeadBranch:         "feature/caching",
		BaseBranch:         "main",
		Labels:             []string{"feature"},
		Author:             "bob",
		CreatedAt:          "2025-01-01T09:00:00Z",
		UpdatedAt:          "2025-01-02T13:00:00Z",
		IsMerged:           true,
		IsClosed:           false,
		ReviewersRequested: []string{},
		Approvals:          []string{"alice", "charlie"},
	})

	return stub
}

// NewGitLabStub creates a GitLab stub provider with sample data.
func NewGitLabStub() *StubProvider {
	stub := NewStubProvider("GitLab", "gitlab")

	stub.AddIssue(&providers.Issue{
		ID:        "456",
		Number:    456,
		Title:     "Fix: API timeout issue",
		Body:      "API requests are timing out after 30 seconds",
		URL:       "https://gitlab.com/group/project/-/issues/456",
		State:     "OPENED",
		Labels:    []string{"bug"},
		Author:    "alice",
		CreatedAt: "2025-01-01T10:00:00Z",
		UpdatedAt: "2025-01-02T12:00:00Z",
		IsClosed:  false,
	})

	return stub
}

// NewJIRAStub creates a JIRA stub provider with sample data.
func NewJIRAStub() *StubProvider {
	stub := NewStubProvider("JIRA", "jira")

	stub.AddIssue(&providers.Issue{
		ID:        "PROJ-789",
		Key:       "PROJ-789",
		Title:     "Document new API endpoints",
		Body:      "Add documentation for the new REST API endpoints",
		URL:       "https://jira.example.com/browse/PROJ-789",
		State:     "OPEN",
		Labels:    []string{"documentation"},
		Author:    "bob",
		CreatedAt: "2025-01-01T10:00:00Z",
		UpdatedAt: "2025-01-02T12:00:00Z",
		IsClosed:  false,
	})

	return stub
}

// NewLinearStub creates a Linear stub provider with sample data.
func NewLinearStub() *StubProvider {
	stub := NewStubProvider("Linear", "linear")

	stub.AddIssue(&providers.Issue{
		ID:        "ENG-123",
		Key:       "ENG-123",
		Title:     "Improve test coverage",
		Body:      "Increase test coverage to 80%+",
		URL:       "https://linear.app/team/issue/ENG-123",
		State:     "OPEN",
		Labels:    []string{"testing"},
		Author:    "alice",
		CreatedAt: "2025-01-01T10:00:00Z",
		UpdatedAt: "2025-01-02T12:00:00Z",
		IsClosed:  false,
	})

	return stub
}

// AddIssue adds an issue to the stub provider.
func (s *StubProvider) AddIssue(issue *providers.Issue) {
	if s.Issues == nil {
		s.Issues = make(map[string]*providers.Issue)
	}

	s.Issues[issue.ID] = issue
}

// AddPullRequest adds a PR to the stub provider.
func (s *StubProvider) AddPullRequest(pr *providers.PullRequest) {
	if s.PullRequests == nil {
		s.PullRequests = make(map[string]*providers.PullRequest)
	}

	s.PullRequests[pr.ID] = pr
}

// SetError configures an error for a specific method.
func (s *StubProvider) SetError(method string, err error) {
	if s.Errors == nil {
		s.Errors = make(map[string]error)
	}

	s.Errors[method] = err
}

// ListIssues returns all issues (or error if configured).
func (s *StubProvider) ListIssues(_ context.Context, limit int) ([]providers.Issue, error) { //nolint:dupl
	s.recordCall("ListIssues", limit)

	if err, ok := s.Errors["ListIssues"]; ok {
		return nil, err
	}

	issues := make([]providers.Issue, 0, len(s.Issues))
	for _, issue := range s.Issues {
		issues = append(issues, *issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].ID < issues[j].ID
	})

	if limit > 0 && len(issues) > limit {
		issues = issues[:limit]
	}

	return issues, nil
}

// GetIssue returns a specific issue by ID.
func (s *StubProvider) GetIssue(_ context.Context, id string) (*providers.Issue, error) {
	s.recordCall("GetIssue", id)

	if err, ok := s.Errors["GetIssue"]; ok {
		return nil, err
	}

	issue, ok := s.Issues[id]
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", id)
	}

	return issue, nil
}

// IsIssueClosed returns true if an issue is closed.
func (s *StubProvider) IsIssueClosed(_ context.Context, id string) (bool, error) {
	s.recordCall("IsIssueClosed", id)

	if err, ok := s.Errors["IsIssueClosed"]; ok {
		return false, err
	}

	issue, ok := s.Issues[id]
	if !ok {
		return false, fmt.Errorf("issue not found: %s", id)
	}

	return issue.IsClosed, nil
}

// ListPullRequests returns all pull requests.
func (s *StubProvider) ListPullRequests(_ context.Context, limit int) ([]providers.PullRequest, error) { //nolint:dupl
	s.recordCall("ListPullRequests", limit)

	if err, ok := s.Errors["ListPullRequests"]; ok {
		return nil, err
	}

	prs := make([]providers.PullRequest, 0, len(s.PullRequests))
	for _, pr := range s.PullRequests {
		prs = append(prs, *pr)
	}

	sort.Slice(prs, func(i, j int) bool {
		return prs[i].ID < prs[j].ID
	})

	if limit > 0 && len(prs) > limit {
		prs = prs[:limit]
	}

	return prs, nil
}

// GetPullRequest returns a specific PR by ID.
func (s *StubProvider) GetPullRequest(_ context.Context, id string) (*providers.PullRequest, error) {
	s.recordCall("GetPullRequest", id)

	if err, ok := s.Errors["GetPullRequest"]; ok {
		return nil, err
	}

	pr, ok := s.PullRequests[id]
	if !ok {
		return nil, fmt.Errorf("pull request not found: %s", id)
	}

	return pr, nil
}

// IsPullRequestMerged returns true if a PR is merged.
func (s *StubProvider) IsPullRequestMerged(_ context.Context, id string) (bool, error) {
	s.recordCall("IsPullRequestMerged", id)

	if err, ok := s.Errors["IsPullRequestMerged"]; ok {
		return false, err
	}

	pr, ok := s.PullRequests[id]
	if !ok {
		return false, fmt.Errorf("pull request not found: %s", id)
	}

	return pr.IsMerged, nil
}

// CreateIssue creates a new issue.
func (s *StubProvider) CreateIssue(_ context.Context, title, body string) (*providers.Issue, error) {
	s.recordCall("CreateIssue", map[string]string{"title": title, "body": body})

	if err, ok := s.Errors["CreateIssue"]; ok {
		return nil, err
	}

	newID := fmt.Sprintf("%d", len(s.Issues)+1)
	issue := &providers.Issue{
		ID:        newID,
		Number:    len(s.Issues) + 1,
		Title:     title,
		Body:      body,
		State:     "OPEN",
		IsClosed:  false,
		CreatedAt: "2025-01-02T15:00:00Z",
		UpdatedAt: "2025-01-02T15:00:00Z",
	}

	s.AddIssue(issue)

	return issue, nil
}

// CreatePullRequest creates a new PR.
func (s *StubProvider) CreatePullRequest(_ context.Context, title, body, baseBranch, headBranch string) (*providers.PullRequest, error) {
	s.recordCall("CreatePullRequest", map[string]string{
		"title":      title,
		"baseBranch": baseBranch,
		"headBranch": headBranch,
	})

	if err, ok := s.Errors["CreatePullRequest"]; ok {
		return nil, err
	}

	newID := fmt.Sprintf("%d", len(s.PullRequests)+1)
	pr := &providers.PullRequest{
		ID:         newID,
		Number:     len(s.PullRequests) + 1,
		Title:      title,
		Body:       body,
		State:      "OPEN",
		HeadBranch: headBranch,
		BaseBranch: baseBranch,
		IsMerged:   false,
		IsClosed:   false,
		CreatedAt:  "2025-01-02T15:00:00Z",
		UpdatedAt:  "2025-01-02T15:00:00Z",
	}

	s.AddPullRequest(pr)

	return pr, nil
}

// GetBranchNameSuffix returns the suffix for branch names.
func (s *StubProvider) GetBranchNameSuffix(issue *providers.Issue) string {
	if issue.Key != "" {
		return strings.ToLower(issue.Key)
	}

	return fmt.Sprintf("%d", issue.Number)
}

// SanitizeBranchName sanitizes a title for use in a branch name.
func (s *StubProvider) SanitizeBranchName(title string) string {
	s.recordCall("SanitizeBranchName", title)

	result := strings.ToLower(title)
	result = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}

		if r == ' ' || r == '_' {
			return '-'
		}

		return -1
	}, result)

	result = strings.Trim(result, "-")
	if len(result) > 40 {
		result = result[:40]
	}

	return result
}

// Name returns the provider name.
func (s *StubProvider) Name() string {
	s.recordCall("Name", nil)
	return s.ProviderName
}

// ProviderType returns the provider type.
func (s *StubProvider) ProviderType() string {
	s.recordCall("ProviderType", nil)
	return s.ProviderTypeValue
}

// recordCall records a method call for assertion purposes.
func (s *StubProvider) recordCall(method string, args interface{}) {
	s.Calls = append(s.Calls, MethodCall{
		Method: method,
		Args:   args,
	})
}

// GetCallCount returns the number of times a method was called.
func (s *StubProvider) GetCallCount(method string) int {
	count := 0

	for _, call := range s.Calls {
		if call.Method == method {
			count++
		}
	}

	return count
}

// Reset clears all data and call history.
func (s *StubProvider) Reset() {
	s.Issues = make(map[string]*providers.Issue)
	s.PullRequests = make(map[string]*providers.PullRequest)
	s.Errors = make(map[string]error)
	s.Calls = []MethodCall{}
}
