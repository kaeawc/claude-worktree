package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrGlabNotInstalled is returned when glab CLI is not installed
	ErrGlabNotInstalled = errors.New("glab CLI not installed")
	// ErrGlabNotAuthenticated is returned when glab CLI is not authenticated
	ErrGlabNotAuthenticated = errors.New("glab CLI not authenticated")
)

// Client provides GitLab operations via glab CLI
type Client struct {
	// Owner is the group/owner (may include nested groups)
	Owner string
	// Project is the project name
	Project string
	// Host is the GitLab host (gitlab.com or self-hosted)
	Host string
	// executor handles glab CLI command execution
	executor GitLabExecutor
}

// NewClient creates a GitLab client, auto-detecting project from git remote
// Returns error if glab CLI not installed or not authenticated
func NewClient(gitRoot string) (*Client, error) {
	executor := NewGitLabExecutor()
	return NewClientWithExecutor(gitRoot, executor)
}

// NewClientWithExecutor creates a GitLab client with a custom executor (for testing)
func NewClientWithExecutor(gitRoot string, executor GitLabExecutor) (*Client, error) {
	// Check if glab CLI is installed
	if !IsInstalled(executor) {
		return nil, ErrGlabNotInstalled
	}

	// Check if glab is authenticated
	if err := IsAuthenticated(executor); err != nil {
		return nil, err
	}

	// Auto-detect repository
	info, err := DetectRepository(gitRoot)
	if err != nil {
		return nil, err
	}

	return &Client{
		Owner:    info.Owner,
		Project:  info.Project,
		Host:     info.Host,
		executor: executor,
	}, nil
}

// NewClientWithProject creates a client with explicit owner/project/host
func NewClientWithProject(owner, project, host string) (*Client, error) {
	executor := NewGitLabExecutor()
	return NewClientWithProjectAndExecutor(owner, project, host, executor)
}

// NewClientWithProjectAndExecutor creates a client with explicit params and custom executor (for testing)
func NewClientWithProjectAndExecutor(owner, project, host string, executor GitLabExecutor) (*Client, error) {
	// Check if glab CLI is installed
	if !IsInstalled(executor) {
		return nil, ErrGlabNotInstalled
	}

	// Check if glab is authenticated
	if err := IsAuthenticated(executor); err != nil {
		return nil, err
	}

	return &Client{
		Owner:    owner,
		Project:  project,
		Host:     host,
		executor: executor,
	}, nil
}

// IsInstalled checks if glab CLI is installed
func IsInstalled(executor GitLabExecutor) bool {
	_, err := executor.Execute("--version")
	return err == nil
}

// IsAuthenticated checks if glab CLI is authenticated
func IsAuthenticated(executor GitLabExecutor) error {
	_, err := executor.Execute("auth", "status")
	if err != nil {
		return ErrGlabNotAuthenticated
	}
	return nil
}

// execGlab executes a glab CLI command and returns output
func (c *Client) execGlab(args ...string) ([]byte, error) {
	output, err := c.executor.Execute(args...)
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

// execGlabInRepo executes a glab CLI command with repo context
func (c *Client) execGlabInRepo(args ...string) ([]byte, error) {
	// Build repo context: owner/project or owner/subgroup/project
	repoPath := fmt.Sprintf("%s/%s", c.Owner, c.Project)

	// Prepend host and repo flags to args
	fullArgs := []string{}

	// Add host flag for self-hosted instances
	if c.Host != "gitlab.com" {
		fullArgs = append(fullArgs, "--host", c.Host)
	}

	// Add repo context
	fullArgs = append(fullArgs, "-R", repoPath)

	// Add original args
	fullArgs = append(fullArgs, args...)

	return c.execGlab(fullArgs...)
}

// CreateIssue creates a new issue with the given title and body
// Returns the created issue with its IID and URL
func (c *Client) CreateIssue(title, body string) (*Issue, error) {
	if title == "" {
		return nil, fmt.Errorf("issue title cannot be empty")
	}

	args := []string{"issue", "create", "--title", title}
	if body != "" {
		args = append(args, "--description", body)
	}
	args = append(args, "--json")

	output, err := c.execGlabInRepo(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse created issue: %w", err)
	}

	return &issue, nil
}
