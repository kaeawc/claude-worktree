package provider

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitLabProvider implements Provider for GitLab
type GitLabProvider struct{}

// Name returns the provider name
func (g *GitLabProvider) Name() string {
	return "gitlab"
}

// IsAvailable checks if glab CLI is installed
func (g *GitLabProvider) IsAvailable() bool {
	cmd := exec.Command("glab", "--version")
	err := cmd.Run()
	return err == nil
}

// GetIssueStatus checks if a GitLab issue is closed
func (g *GitLabProvider) GetIssueStatus(issueID string) (bool, bool, error) {
	if !g.IsAvailable() {
		return false, false, ErrProviderNotAvailable
	}

	// Use glab issue view with JSON output
	cmd := exec.Command("glab", "issue", "view", issueID, "--json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") ||
				strings.Contains(string(exitErr.Stderr), "Could not find") {
				return false, false, ErrNotFound
			}
		}
		return false, false, fmt.Errorf("failed to get issue status: %w", err)
	}

	var result struct {
		State string `json:"state"`
		Title string `json:"title"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, false, fmt.Errorf("failed to parse issue status: %w", err)
	}

	isClosed := result.State == "closed"
	// For GitLab issues, we consider closed as completed
	isCompleted := isClosed

	return isClosed, isCompleted, nil
}

// GetPRStatus checks if a GitLab MR is merged
func (g *GitLabProvider) GetPRStatus(mrID string) (bool, error) {
	if !g.IsAvailable() {
		return false, ErrProviderNotAvailable
	}

	// Use glab mr view with JSON output
	cmd := exec.Command("glab", "mr", "view", mrID, "--json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") ||
				strings.Contains(string(exitErr.Stderr), "Could not find") {
				return false, ErrNotFound
			}
		}
		return false, fmt.Errorf("failed to get MR status: %w", err)
	}

	var result struct {
		State string `json:"state"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("failed to parse MR status: %w", err)
	}

	// MR is merged if state is "merged"
	isMerged := result.State == "merged"

	return isMerged, nil
}

// GetStatusForBranch attempts to determine the status based on branch name
func (g *GitLabProvider) GetStatusForBranch(branchName string) (*IssueStatus, error) {
	providerType, id, found := ParseBranchName(branchName)
	if !found {
		return nil, fmt.Errorf("could not parse branch name: %s", branchName)
	}

	status := &IssueStatus{
		ID:       id,
		Provider: "gitlab",
	}

	switch providerType {
	case "gitlab-mr":
		isMerged, err := g.GetPRStatus(id)
		if err != nil {
			return nil, err
		}
		status.IsClosed = isMerged
		status.IsCompleted = isMerged

	default:
		return nil, fmt.Errorf("unsupported branch type: %s", providerType)
	}

	return status, nil
}
