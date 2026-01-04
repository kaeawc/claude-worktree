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
	"github.com/kaeawc/auto-worktree/internal/linear"
	"github.com/kaeawc/auto-worktree/internal/providers"
	"github.com/kaeawc/auto-worktree/internal/providers/stubs"
)

const (
	providerGitHub = "github"
	providerGitLab = "gitlab"
	providerJira   = "jira"
	providerLinear = "linear"
)

// GetProviderForRepository returns the appropriate provider for the given repository
// based on configuration or auto-detection
func GetProviderForRepository(repo *git.Repository) (providers.Provider, error) {
	cfg := git.NewConfig(repo.RootPath)

	providerType := cfg.GetIssueProvider()

	switch providerType {
	case providerGitHub:
		return newGitHubProvider(repo)
	case providerGitLab:
		return newGitLabProvider(repo)
	case providerJira:
		return newJIRAProvider()
	case providerLinear:
		return newLinearProvider(repo)
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
	installInfo := GitHubInstallInfo()

	if !github.IsInstalled(executor) {
		return nil, errors.New(installInfo.FormatNotInstalledError())
	}

	if err := github.IsAuthenticated(executor); err != nil {
		return nil, errors.New(installInfo.FormatNotAuthenticatedError())
	}

	client, err := github.NewClient(repo.RootPath)
	if err != nil {
		return nil, handleGitHubClientError(err)
	}

	return newGitHubProviderFromClient(client), nil
}

// handleGitHubClientError converts GitHub client errors to user-friendly messages
func handleGitHubClientError(err error) error {
	installInfo := GitHubInstallInfo()

	if errors.Is(err, github.ErrGHNotInstalled) {
		return errors.New(installInfo.FormatNotInstalledError())
	}

	if errors.Is(err, github.ErrGHNotAuthenticated) {
		return errors.New(installInfo.FormatNotAuthenticatedError())
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

func (g *githubProviderShim) ListIssues(_ context.Context, limit int) ([]providers.Issue, error) {
	issues, err := g.client.ListOpenIssues(limit)
	if err != nil {
		return nil, err
	}

	result := make([]providers.Issue, 0, len(issues))

	for i := range issues {
		labelNames := make([]string, len(issues[i].Labels))
		for j, label := range issues[i].Labels {
			labelNames[j] = label.Name
		}

		result = append(result, providers.Issue{
			ID:     fmt.Sprintf("%d", issues[i].Number),
			Number: issues[i].Number,
			Title:  issues[i].Title,
			Body:   issues[i].Body,
			URL:    issues[i].URL,
			State:  issues[i].State,
			Labels: labelNames,
		})
	}

	return result, nil
}

func (g *githubProviderShim) GetIssue(_ context.Context, id string) (*providers.Issue, error) {
	// Parse issue number from ID
	var issueNum int
	_, _ = fmt.Sscanf(id, "%d", &issueNum) //nolint:gosec,errcheck

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

func (g *githubProviderShim) IsIssueClosed(_ context.Context, id string) (bool, error) {
	var issueNum int
	_, _ = fmt.Sscanf(id, "%d", &issueNum) //nolint:gosec,errcheck

	return g.client.IsIssueMerged(issueNum)
}

func (g *githubProviderShim) ListPullRequests(_ context.Context, _ int) ([]providers.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (g *githubProviderShim) GetPullRequest(_ context.Context, _ string) (*providers.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (g *githubProviderShim) IsPullRequestMerged(_ context.Context, _ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (g *githubProviderShim) CreateIssue(_ context.Context, title, body string) (*providers.Issue, error) {
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

func (g *githubProviderShim) CreatePullRequest(_ context.Context, _, _, _, _ string) (*providers.PullRequest, error) {
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
	installInfo := GitLabInstallInfo()

	if !gitlab.IsInstalled(executor) {
		return nil, errors.New(installInfo.FormatNotInstalledError())
	}

	if err := gitlab.IsAuthenticated(executor); err != nil {
		return nil, errors.New(installInfo.FormatNotAuthenticatedError())
	}

	client, err := gitlab.NewClient(repo.RootPath)
	if err != nil {
		return nil, handleGitLabClientError(err)
	}

	return newGitLabProviderFromClient(client), nil
}

// handleGitLabClientError converts GitLab client errors to user-friendly messages
func handleGitLabClientError(err error) error {
	installInfo := GitLabInstallInfo()

	if errors.Is(err, gitlab.ErrGlabNotInstalled) {
		return errors.New(installInfo.FormatNotInstalledError())
	}

	if errors.Is(err, gitlab.ErrGlabNotAuthenticated) {
		return errors.New(installInfo.FormatNotAuthenticatedError())
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

func (g *gitlabProviderShim) ListIssues(_ context.Context, limit int) ([]providers.Issue, error) {
	issues, err := g.client.ListOpenIssues(limit)
	if err != nil {
		return nil, err
	}

	result := make([]providers.Issue, 0, len(issues))

	for i := range issues {
		result = append(result, providers.Issue{
			ID:     fmt.Sprintf("%d", issues[i].IID),
			Number: issues[i].IID,
			Title:  issues[i].Title,
			Body:   issues[i].Description,
			URL:    issues[i].WebURL,
			State:  issues[i].State,
			Labels: issues[i].Labels,
		})
	}

	return result, nil
}

func (g *gitlabProviderShim) GetIssue(_ context.Context, id string) (*providers.Issue, error) {
	var issueID int
	_, _ = fmt.Sscanf(id, "%d", &issueID) //nolint:gosec,errcheck

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

func (g *gitlabProviderShim) IsIssueClosed(_ context.Context, id string) (bool, error) {
	var issueID int
	_, _ = fmt.Sscanf(id, "%d", &issueID) //nolint:gosec,errcheck

	return g.client.IsIssueClosed(issueID)
}

func (g *gitlabProviderShim) ListPullRequests(_ context.Context, _ int) ([]providers.PullRequest, error) {
	return nil, errors.New("use GetMergeRequests instead")
}

func (g *gitlabProviderShim) GetPullRequest(_ context.Context, _ string) (*providers.PullRequest, error) {
	return nil, errors.New("use GetMergeRequest instead")
}

func (g *gitlabProviderShim) IsPullRequestMerged(_ context.Context, _ string) (bool, error) {
	return false, errors.New("use IsMergeRequestMerged instead")
}

func (g *gitlabProviderShim) CreateIssue(_ context.Context, title, body string) (*providers.Issue, error) {
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

func (g *gitlabProviderShim) CreatePullRequest(_ context.Context, _, _, _, _ string) (*providers.PullRequest, error) {
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
	installInfo := JIRAInstallInfo()

	if !jira.IsInstalled() {
		return nil, errors.New(installInfo.FormatNotInstalledError())
	}

	if err := jira.IsConfigured(); err != nil {
		return nil, errors.New(installInfo.FormatNotAuthenticatedError())
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

// newLinearProvider creates a Linear provider
func newLinearProvider(repo *git.Repository) (providers.Provider, error) {
	executor := linear.NewExecutor()
	installInfo := LinearInstallInfo()

	if !linear.IsInstalled(executor) {
		return nil, errors.New(installInfo.FormatNotInstalledError())
	}

	if err := linear.IsAuthenticated(executor); err != nil {
		return nil, errors.New(installInfo.FormatNotAuthenticatedError())
	}

	cfg := git.NewConfig(repo.RootPath)

	client, err := linear.NewClientWithExecutor(repo.RootPath, cfg, executor)
	if err != nil {
		return nil, handleLinearClientError(err)
	}

	return newLinearProviderFromClient(client), nil
}

// handleLinearClientError converts Linear client errors to user-friendly messages
func handleLinearClientError(err error) error {
	installInfo := LinearInstallInfo()

	if errors.Is(err, linear.ErrLinearNotInstalled) {
		return errors.New(installInfo.FormatNotInstalledError())
	}

	if errors.Is(err, linear.ErrLinearNotAuthenticated) {
		return errors.New(installInfo.FormatNotAuthenticatedError())
	}

	if errors.Is(err, linear.ErrNoTeamConfigured) {
		return errors.New("no Linear team configured. Run: auto-worktree settings and set linear-team")
	}

	return fmt.Errorf("failed to initialize Linear client: %w", err)
}

// newLinearProviderFromClient creates a provider wrapper around Linear client
func newLinearProviderFromClient(client *linear.Client) providers.Provider {
	return &linearProviderShim{client: client}
}

// linearProviderShim adapts the Linear client to the providers.Provider interface
type linearProviderShim struct {
	client *linear.Client
}

func (l *linearProviderShim) ListIssues(_ context.Context, limit int) ([]providers.Issue, error) {
	issues, err := l.client.ListOpenIssues(limit)
	if err != nil {
		return nil, err
	}

	result := make([]providers.Issue, 0, len(issues))

	for i := range issues {
		result = append(result, providers.Issue{
			ID:     issues[i].Identifier,
			Number: issues[i].Number,
			Title:  issues[i].Title,
			Body:   issues[i].Description,
			URL:    issues[i].URL,
			State:  issues[i].State.Type,
			Labels: extractLinearLabels(issues[i].Labels),
		})
	}

	return result, nil
}

func (l *linearProviderShim) GetIssue(_ context.Context, id string) (*providers.Issue, error) {
	issue, err := l.client.GetIssue(id)
	if err != nil {
		return nil, err
	}

	return &providers.Issue{
		ID:     issue.Identifier,
		Number: issue.Number,
		Title:  issue.Title,
		Body:   issue.Description,
		URL:    issue.URL,
		State:  issue.State.Type,
		Labels: extractLinearLabels(issue.Labels),
	}, nil
}

func (l *linearProviderShim) IsIssueClosed(_ context.Context, id string) (bool, error) {
	issue, err := l.client.GetIssue(id)
	if err != nil {
		return false, err
	}

	// Check if issue is in a completed or canceled state
	stateType := issue.State.Type

	return stateType == "completed" || stateType == "canceled", nil
}

func (l *linearProviderShim) ListPullRequests(_ context.Context, _ int) ([]providers.PullRequest, error) {
	return nil, errors.New("linear does not have pull requests")
}

func (l *linearProviderShim) GetPullRequest(_ context.Context, _ string) (*providers.PullRequest, error) {
	return nil, errors.New("linear does not have pull requests")
}

func (l *linearProviderShim) IsPullRequestMerged(_ context.Context, _ string) (bool, error) {
	return false, errors.New("linear does not have pull requests")
}

func (l *linearProviderShim) CreateIssue(_ context.Context, _, _ string) (*providers.Issue, error) {
	return nil, errors.New("creating issues via CLI not yet implemented for Linear")
}

func (l *linearProviderShim) CreatePullRequest(_ context.Context, _, _, _, _ string) (*providers.PullRequest, error) {
	return nil, errors.New("linear does not have pull requests")
}

func (l *linearProviderShim) GetBranchNameSuffix(issue *providers.Issue) string {
	// Linear issues use identifier like "ENG-123"
	return issue.ID
}

func (l *linearProviderShim) SanitizeBranchName(title string) string {
	return git.SanitizeBranchName(title)
}

func (l *linearProviderShim) Name() string {
	return "Linear"
}

func (l *linearProviderShim) ProviderType() string {
	return providerLinear
}

// extractLinearLabels extracts label names from Linear labels
func extractLinearLabels(labels []linear.Label) []string {
	result := make([]string, len(labels))
	for i, label := range labels {
		result[i] = label.Name
	}

	return result
}

// GetTestProvider returns a stub provider for testing
func GetTestProvider(providerType string) providers.Provider {
	switch providerType {
	case providerGitHub:
		return stubs.NewGitHubStub()
	case providerJira:
		return stubs.NewJIRAStub()
	case providerGitLab:
		return stubs.NewGitLabStub()
	case providerLinear:
		return stubs.NewLinearStub()
	default:
		return stubs.NewGitHubStub()
	}
}
