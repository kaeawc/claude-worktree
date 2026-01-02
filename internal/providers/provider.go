// Package providers defines interfaces for different issue tracking and PR management providers.
package providers

import "context"

// Provider defines the interface for issue tracking and PR management providers.
// Implementations should support GitHub, GitLab, JIRA, and Linear.
type Provider interface {
	// ListIssues returns all open issues.
	// Limit controls how many issues to fetch (0 means default limit).
	ListIssues(ctx context.Context, limit int) ([]Issue, error)

	// GetIssue returns details for a specific issue by ID or key.
	GetIssue(ctx context.Context, id string) (*Issue, error)

	// IsIssueClosed returns true if an issue is closed.
	IsIssueClosed(ctx context.Context, id string) (bool, error)

	// ListPullRequests returns all open pull requests.
	// Limit controls how many PRs to fetch (0 means default limit).
	ListPullRequests(ctx context.Context, limit int) ([]PullRequest, error)

	// GetPullRequest returns details for a specific PR by ID or number.
	GetPullRequest(ctx context.Context, id string) (*PullRequest, error)

	// IsPullRequestMerged returns true if a PR is merged.
	IsPullRequestMerged(ctx context.Context, id string) (bool, error)

	// CreateIssue creates a new issue with the given details.
	CreateIssue(ctx context.Context, title, body string) (*Issue, error)

	// CreatePullRequest creates a new pull request.
	CreatePullRequest(ctx context.Context, title, body, baseBranch, headBranch string) (*PullRequest, error)

	// GetBranchNameSuffix returns the suffix to append to branch names
	// (e.g., "123" for issue 123 in GitHub, "PROJ-456" for JIRA)
	GetBranchNameSuffix(issue *Issue) string

	// SanitizeBranchName sanitizes a title for use in a branch name
	// (e.g., "Fix bug" -> "fix-bug")
	SanitizeBranchName(title string) string

	// Name returns the provider name (e.g., "GitHub", "GitLab")
	Name() string

	// ProviderType returns the provider type for configuration
	ProviderType() string
}

// Issue represents an issue in a provider.
type Issue struct {
	// ID is the unique identifier (number for GitHub, key for JIRA, etc.)
	ID string
	// Number is the issue number (GitHub specific)
	Number int
	// Key is the issue key (JIRA specific, e.g., "PROJ-123")
	Key string
	// Title is the issue title
	Title string
	// Body is the issue description/body
	Body string
	// URL is the issue URL
	URL string
	// State is the issue state ("OPEN", "CLOSED", etc.)
	State string
	// Labels are the issue labels
	Labels []string
	// Author is the person who created the issue
	Author string
	// CreatedAt is the creation timestamp
	CreatedAt string
	// UpdatedAt is the last update timestamp
	UpdatedAt string
	// Assignee is the person assigned to the issue (if any)
	Assignee string
	// IsClosed is true if the issue is closed
	IsClosed bool
}

// PullRequest represents a pull request in a provider.
type PullRequest struct {
	// ID is the unique identifier (number for GitHub, MR ID for GitLab, etc.)
	ID string
	// Number is the PR number (GitHub specific)
	Number int
	// Title is the PR title
	Title string
	// Body is the PR description/body
	Body string
	// URL is the PR URL
	URL string
	// State is the PR state ("OPEN", "CLOSED", "MERGED", etc.)
	State string
	// HeadBranch is the branch being merged (source branch)
	HeadBranch string
	// BaseBranch is the branch being merged into (target branch)
	BaseBranch string
	// Labels are the PR labels
	Labels []string
	// Author is the person who created the PR
	Author string
	// CreatedAt is the creation timestamp
	CreatedAt string
	// UpdatedAt is the last update timestamp
	UpdatedAt string
	// IsMerged is true if the PR is merged
	IsMerged bool
	// IsClosed is true if the PR is closed (but not merged)
	IsClosed bool
	// ReviewersRequested are the reviewers who need to approve
	ReviewersRequested []string
	// Approvals are the reviewers who have approved
	Approvals []string
}

// Config contains provider-specific configuration.
type Config struct {
	// Provider type (github, gitlab, jira, linear)
	Provider string
	// Owner/Organization (GitHub specific)
	Owner string
	// Repo name (GitHub specific)
	Repo string
	// Server URL (GitLab, JIRA self-hosted)
	ServerURL string
	// Project key or ID (GitLab, JIRA)
	ProjectKey string
	// Project ID (GitLab)
	ProjectID string
	// Team (Linear specific)
	Team string
	// APIKey for authentication (if needed)
	APIKey string
}
