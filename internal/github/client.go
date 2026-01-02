package github

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrGHNotInstalled is returned when gh CLI is not installed
	ErrGHNotInstalled = errors.New("gh CLI not installed")
	// ErrGHNotAuthenticated is returned when gh CLI is not authenticated
	ErrGHNotAuthenticated = errors.New("gh CLI not authenticated")
)

// Client provides GitHub operations via gh CLI
type Client struct {
	// Owner is the repository owner (org or user)
	Owner string
	// Repo is the repository name
	Repo string
	// executor handles gh CLI command execution
	executor GitHubExecutor
}

// NewClient creates a GitHub client, auto-detecting repo from git remote
// Returns error if gh CLI not installed or not authenticated
func NewClient(gitRoot string) (*Client, error) {
	executor := NewGitHubExecutor()
	return NewClientWithExecutor(gitRoot, executor)
}

// NewClientWithExecutor creates a GitHub client with a custom executor (for testing)
func NewClientWithExecutor(gitRoot string, executor GitHubExecutor) (*Client, error) {
	// Check if gh CLI is installed
	if !IsInstalled(executor) {
		return nil, ErrGHNotInstalled
	}

	// Check if gh is authenticated
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
		Repo:     info.Name,
		executor: executor,
	}, nil
}

// NewClientWithRepo creates a client with explicit owner/repo
func NewClientWithRepo(owner, repo string) (*Client, error) {
	executor := NewGitHubExecutor()
	return NewClientWithRepoAndExecutor(owner, repo, executor)
}

// NewClientWithRepoAndExecutor creates a client with explicit owner/repo and custom executor (for testing)
func NewClientWithRepoAndExecutor(owner, repo string, executor GitHubExecutor) (*Client, error) {
	// Check if gh CLI is installed
	if !IsInstalled(executor) {
		return nil, ErrGHNotInstalled
	}

	// Check if gh is authenticated
	if err := IsAuthenticated(executor); err != nil {
		return nil, err
	}

	return &Client{
		Owner:    owner,
		Repo:     repo,
		executor: executor,
	}, nil
}

// IsInstalled checks if gh CLI is installed
func IsInstalled(executor GitHubExecutor) bool {
	_, err := executor.Execute("--version")
	return err == nil
}

// IsAuthenticated checks if gh CLI is authenticated
func IsAuthenticated(executor GitHubExecutor) error {
	_, err := executor.Execute("auth", "status")
	if err != nil {
		return ErrGHNotAuthenticated
	}
	return nil
}

// execGH executes a gh CLI command and returns output
func (c *Client) execGH(args ...string) ([]byte, error) {
	output, err := c.executor.Execute(args...)
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

// execGHInRepo executes a gh CLI command with repo context
func (c *Client) execGHInRepo(args ...string) ([]byte, error) {
	// Prepend repo flag to args
	fullArgs := append([]string{"-R", fmt.Sprintf("%s/%s", c.Owner, c.Repo)}, args...)
	return c.execGH(fullArgs...)
}

// CreateIssue creates a new issue with the given title and body
// Returns the created issue with its number and URL
func (c *Client) CreateIssue(title, body string) (*Issue, error) {
	if title == "" {
		return nil, fmt.Errorf("issue title cannot be empty")
	}

	args := []string{"issue", "create", "--title", title}
	if body != "" {
		args = append(args, "--body", body)
	}
	args = append(args, "--json", "number,title,body,url")

	output, err := c.execGHInRepo(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse created issue: %w", err)
	}

	return &issue, nil
}
