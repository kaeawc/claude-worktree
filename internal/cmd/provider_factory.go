// Package cmd provides command-line interface handlers for auto-worktree operations.
package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/kaeawc/auto-worktree/internal/git"
	"github.com/kaeawc/auto-worktree/internal/github"
	"github.com/kaeawc/auto-worktree/internal/gitlab"
	"github.com/kaeawc/auto-worktree/internal/jira"
	"github.com/kaeawc/auto-worktree/internal/providers"
	"github.com/kaeawc/auto-worktree/internal/providers/stubs"
)

// GetProviderForRepository returns the appropriate provider for the given repository
// based on configuration or auto-detection
func GetProviderForRepository(repo *git.Repository) (providers.Provider, error) {
	cfg := git.NewConfig(repo.RootPath)

	providerType := cfg.GetIssueProvider()

	switch providerType {
	case "github":
		return newGitHubProvider(repo)
	case "gitlab":
		return newGitLabProvider(repo)
	case "jira":
		return newJIRAProvider()
	case "linear":
		return nil, errors.New("linear provider not yet implemented")
	case "":
		// Try to auto-detect from the repo
		return autoDetectProvider(repo)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// newGitHubProvider creates a GitHub provider
func newGitHubProvider(repo *git.Repository) (providers.Provider, error) {
	executor := github.NewGitHubExecutor()
	if !github.IsInstalled(executor) {
		return nil, errors.New("gh CLI is not installed. Install with: brew install gh")
	}

	if err := github.IsAuthenticated(executor); err != nil {
		return nil, errors.New("gh CLI is not authenticated. Run: gh auth login")
	}

	client, err := github.NewClient(repo.RootPath)
	if err != nil {
		return nil, handleGitHubClientError(err)
	}

	return newGitHubProviderFromClient(client), nil
}

// handleGitHubClientError converts GitHub client errors to user-friendly messages
func handleGitHubClientError(err error) error {
	if errors.Is(err, github.ErrGHNotInstalled) {
		return errors.New("gh CLI is not installed. Install with: brew install gh")
	}
	if errors.Is(err, github.ErrGHNotAuthenticated) {
		return errors.New("gh CLI is not authenticated. Run: gh auth login")
	}
	return fmt.Errorf("failed to initialize GitHub client: %w", err)
}

// newGitHubProviderFromClient creates a provider wrapper around GitHub client
// This is a temporary shim until GitHub provider is migrated to the new interface
func newGitHubProviderFromClient(client *github.Client) providers.Provider {
	return &githubProviderShim{client: client}
}

// githubProviderShim adapts the GitHub client to the providers.Provider interface
type githubProviderShim struct {
	client *github.Client
}

func (g *githubProviderShim) ListIssues(ctx context.Context, limit int) ([]providers.Issue, error) {
	issues, err := g.client.ListOpenIssues(limit)
	if err != nil {
		return nil, err
	}

	var result []providers.Issue
	for _, issue := range issues {
		labelNames := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labelNames[i] = label.Name
		}

		result = append(result, providers.Issue{
			ID:     fmt.Sprintf("%d", issue.Number),
			Number: issue.Number,
			Title:  issue.Title,
			Body:   issue.Body,
			URL:    issue.URL,
			State:  issue.State,
			Labels: labelNames,
		})
	}

	return result, nil
}

func (g *githubProviderShim) GetIssue(ctx context.Context, id string) (*providers.Issue, error) {
	// Parse issue number from ID
	var issueNum int
	//nolint:errcheck
	fmt.Sscanf(id, "%d", &issueNum)

	issue, err := g.client.GetIssue(issueNum)
	if err != nil {
		return nil, err
	}

	labelNames := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labelNames[i] = label.Name
	}

	return &providers.Issue{
		ID:     fmt.Sprintf("%d", issue.Number),
		Number: issue.Number,
		Title:  issue.Title,
		Body:   issue.Body,
		URL:    issue.URL,
		State:  issue.State,
		Labels: labelNames,
	}, nil
}

func (g *githubProviderShim) IsIssueClosed(ctx context.Context, id string) (bool, error) {
	var issueNum int
	//nolint:errcheck
	fmt.Sscanf(id, "%d", &issueNum)
	return g.client.IsIssueMerged(issueNum)
}

func (g *githubProviderShim) ListPullRequests(ctx context.Context, limit int) ([]providers.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (g *githubProviderShim) GetPullRequest(ctx context.Context, id string) (*providers.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (g *githubProviderShim) IsPullRequestMerged(ctx context.Context, id string) (bool, error) {
	return false, errors.New("not implemented")
}

func (g *githubProviderShim) CreateIssue(ctx context.Context, title, body string) (*providers.Issue, error) {
	issue, err := g.client.CreateIssue(title, body)
	if err != nil {
		return nil, err
	}

	return &providers.Issue{
		ID:    fmt.Sprintf("%d", issue.Number),
		Title: issue.Title,
		Body:  issue.Body,
		URL:   issue.URL,
	}, nil
}

func (g *githubProviderShim) CreatePullRequest(ctx context.Context, title, body, baseBranch, headBranch string) (*providers.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (g *githubProviderShim) GetBranchNameSuffix(issue *providers.Issue) string {
	return fmt.Sprintf("%d", issue.Number)
}

func (g *githubProviderShim) SanitizeBranchName(title string) string {
	// Use git.SanitizeBranchName
	return git.SanitizeBranchName(title)
}

func (g *githubProviderShim) Name() string {
	return "GitHub"
}

func (g *githubProviderShim) ProviderType() string {
	return "github"
}

// newGitLabProvider creates a GitLab provider
func newGitLabProvider(repo *git.Repository) (providers.Provider, error) {
	executor := gitlab.NewGitLabExecutor()
	if !gitlab.IsInstalled(executor) {
		return nil, errors.New("glab CLI is not installed. Install with: brew install glab")
	}

	if err := gitlab.IsAuthenticated(executor); err != nil {
		return nil, errors.New("glab CLI is not authenticated. Run: glab auth login")
	}

	client, err := gitlab.NewClient(repo.RootPath)
	if err != nil {
		return nil, handleGitLabClientError(err)
	}

	return newGitLabProviderFromClient(client), nil
}

// handleGitLabClientError converts GitLab client errors to user-friendly messages
func handleGitLabClientError(err error) error {
	if errors.Is(err, gitlab.ErrGlabNotInstalled) {
		return errors.New("glab CLI is not installed. Install with: brew install glab")
	}
	if errors.Is(err, gitlab.ErrGlabNotAuthenticated) {
		return errors.New("glab CLI is not authenticated. Run: glab auth login")
	}
	return fmt.Errorf("failed to initialize GitLab client: %w", err)
}

// newGitLabProviderFromClient creates a provider wrapper around GitLab client
func newGitLabProviderFromClient(client *gitlab.Client) providers.Provider {
	return &gitlabProviderShim{client: client}
}

// gitlabProviderShim adapts the GitLab client to the providers.Provider interface
type gitlabProviderShim struct {
	client *gitlab.Client
}

func (g *gitlabProviderShim) ListIssues(ctx context.Context, limit int) ([]providers.Issue, error) {
	issues, err := g.client.ListOpenIssues(limit)
	if err != nil {
		return nil, err
	}

	var result []providers.Issue
	for _, issue := range issues {
		result = append(result, providers.Issue{
			ID:     fmt.Sprintf("%d", issue.IID),
			Number: issue.IID,
			Title:  issue.Title,
			Body:   issue.Description,
			URL:    issue.WebURL,
			State:  issue.State,
			Labels: issue.Labels,
		})
	}

	return result, nil
}

func (g *gitlabProviderShim) GetIssue(ctx context.Context, id string) (*providers.Issue, error) {
	var issueID int
	//nolint:errcheck
	fmt.Sscanf(id, "%d", &issueID)

	issue, err := g.client.GetIssue(issueID)
	if err != nil {
		return nil, err
	}

	return &providers.Issue{
		ID:     fmt.Sprintf("%d", issue.IID),
		Number: issue.IID,
		Title:  issue.Title,
		Body:   issue.Description,
		URL:    issue.WebURL,
		State:  issue.State,
		Labels: issue.Labels,
	}, nil
}

func (g *gitlabProviderShim) IsIssueClosed(ctx context.Context, id string) (bool, error) {
	var issueID int
	//nolint:errcheck
	fmt.Sscanf(id, "%d", &issueID)
	return g.client.IsIssueClosed(issueID)
}

func (g *gitlabProviderShim) ListPullRequests(ctx context.Context, limit int) ([]providers.PullRequest, error) {
	return nil, errors.New("use GetMergeRequests instead")
}

func (g *gitlabProviderShim) GetPullRequest(ctx context.Context, id string) (*providers.PullRequest, error) {
	return nil, errors.New("use GetMergeRequest instead")
}

func (g *gitlabProviderShim) IsPullRequestMerged(ctx context.Context, id string) (bool, error) {
	return false, errors.New("use IsMergeRequestMerged instead")
}

func (g *gitlabProviderShim) CreateIssue(ctx context.Context, title, body string) (*providers.Issue, error) {
	issue, err := g.client.CreateIssue(title, body)
	if err != nil {
		return nil, err
	}

	return &providers.Issue{
		ID:    fmt.Sprintf("%d", issue.IID),
		Title: issue.Title,
		Body:  issue.Description,
		URL:   issue.WebURL,
	}, nil
}

func (g *gitlabProviderShim) CreatePullRequest(ctx context.Context, title, body, baseBranch, headBranch string) (*providers.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (g *gitlabProviderShim) GetBranchNameSuffix(issue *providers.Issue) string {
	return fmt.Sprintf("%d", issue.Number)
}

func (g *gitlabProviderShim) SanitizeBranchName(title string) string {
	return git.SanitizeBranchName(title)
}

func (g *gitlabProviderShim) Name() string {
	return "GitLab"
}

func (g *gitlabProviderShim) ProviderType() string {
	return "gitlab"
}

// newJIRAProvider creates a JIRA provider
func newJIRAProvider() (providers.Provider, error) {
	if !jira.IsInstalled() {
		return nil, fmt.Errorf("jira CLI is not installed. Install with:\n" +
			"  brew install ankitpokhrel/jira-cli/jira-cli\n" +
			"  or see https://github.com/ankitpokhrel/jira-cli#installation\n" +
			"After installation, run: jira init")
	}

	if err := jira.IsConfigured(); err != nil {
		return nil, fmt.Errorf("jira CLI is not configured. Run: jira init")
	}

	// Get repository for configuration
	repo, err := git.NewRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Get configuration
	cfg := git.NewConfig(repo.RootPath)

	server := cfg.GetJiraServer()
	project := cfg.GetJiraProject()

	// Create provider
	provider, err := jira.NewProvider(server, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create JIRA provider: %w", err)
	}

	return provider, nil
}

// autoDetectProvider attempts to detect the provider based on repository type
func autoDetectProvider(repo *git.Repository) (providers.Provider, error) {
	// Try GitHub first (most common)
	executor := github.NewGitHubExecutor()
	if github.IsInstalled(executor) {
		if client, err := github.NewClient(repo.RootPath); err == nil {
			return newGitHubProviderFromClient(client), nil
		}
	}

	// Try GitLab
	glabExecutor := gitlab.NewGitLabExecutor()
	if gitlab.IsInstalled(glabExecutor) {
		if client, err := gitlab.NewClient(repo.RootPath); err == nil {
			return newGitLabProviderFromClient(client), nil
		}
	}

	// Try JIRA
	if jira.IsInstalled() {
		if provider, err := newJIRAProvider(); err == nil {
			return provider, nil
		}
	}

	return nil, errors.New("could not detect any configured issue provider")
}

// GetTestProvider returns a stub provider for testing
func GetTestProvider(providerType string) providers.Provider {
	switch providerType {
	case "github":
		return stubs.NewGitHubStub()
	case "jira":
		return stubs.NewJIRAStub()
	case "gitlab":
		return stubs.NewGitLabStub()
	case "linear":
		return stubs.NewLinearStub()
	default:
		return stubs.NewGitHubStub()
	}
}
