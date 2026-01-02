package provider

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubProvider implements Provider for GitHub
type GitHubProvider struct{}

// Name returns the provider name
func (g *GitHubProvider) Name() string {
	return "github"
}

// IsAvailable checks if gh CLI is installed
func (g *GitHubProvider) IsAvailable() bool {
	cmd := exec.Command("gh", "--version")
	err := cmd.Run()
	return err == nil
}

// GetIssueStatus checks if a GitHub issue is closed/completed
func (g *GitHubProvider) GetIssueStatus(issueID string) (bool, bool, error) {
	if !g.IsAvailable() {
		return false, false, ErrProviderNotAvailable
	}

	// Use gh issue view with JSON output
	cmd := exec.Command("gh", "issue", "view", issueID, "--json", "state,stateReason,title")
	output, err := cmd.Output()
	if err != nil {
		// Check if issue not found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") ||
				strings.Contains(string(exitErr.Stderr), "Could not resolve") {
				return false, false, ErrNotFound
			}
		}
		return false, false, fmt.Errorf("failed to get issue status: %w", err)
	}

	var result struct {
		State       string `json:"state"`
		StateReason string `json:"stateReason"`
		Title       string `json:"title"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, false, fmt.Errorf("failed to parse issue status: %w", err)
	}

	isClosed := result.State == "CLOSED"
	// Issue is "completed" if it's closed with state reason "COMPLETED"
	// This usually means a PR was merged that closed the issue
	isCompleted := isClosed && result.StateReason == "COMPLETED"

	return isClosed, isCompleted, nil
}

// GetPRStatus checks if a GitHub PR is merged
func (g *GitHubProvider) GetPRStatus(prID string) (bool, error) {
	if !g.IsAvailable() {
		return false, ErrProviderNotAvailable
	}

	// Use gh pr view with JSON output
	cmd := exec.Command("gh", "pr", "view", prID, "--json", "state,merged")
	output, err := cmd.Output()
	if err != nil {
		// Check if PR not found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") ||
				strings.Contains(string(exitErr.Stderr), "Could not resolve") {
				return false, ErrNotFound
			}
		}
		return false, fmt.Errorf("failed to get PR status: %w", err)
	}

	var result struct {
		State  string `json:"state"`
		Merged bool   `json:"merged"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("failed to parse PR status: %w", err)
	}

	return result.Merged, nil
}

// GetStatusForBranch attempts to determine the status based on branch name
func (g *GitHubProvider) GetStatusForBranch(branchName string) (*IssueStatus, error) {
	providerType, id, found := ParseBranchName(branchName)
	if !found {
		return nil, fmt.Errorf("could not parse branch name: %s", branchName)
	}

	status := &IssueStatus{
		ID:       id,
		Provider: "github",
	}

	switch providerType {
	case "github-pr":
		isMerged, err := g.GetPRStatus(id)
		if err != nil {
			return nil, err
		}
		status.IsClosed = isMerged
		status.IsCompleted = isMerged

	case "github-issue":
		isClosed, isCompleted, err := g.GetIssueStatus(id)
		if err != nil {
			return nil, err
		}
		status.IsClosed = isClosed
		status.IsCompleted = isCompleted

	default:
		return nil, fmt.Errorf("unsupported branch type: %s", providerType)
	}

	return status, nil
}
