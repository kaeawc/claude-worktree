package provider

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const (
	linearStatusCompleted = "completed"
	linearStatusDone      = "done"
	linearStatusCanceled  = "canceled"
)

// LinearProvider implements Provider for Linear
type LinearProvider struct{}

// Name returns the provider name
func (l *LinearProvider) Name() string {
	return "linear"
}

// IsAvailable checks if linear CLI is installed
func (l *LinearProvider) IsAvailable() bool {
	cmd := exec.Command("linear", "--version")
	err := cmd.Run()
	return err == nil
}

// GetIssueStatus checks if a Linear issue is completed
func (l *LinearProvider) GetIssueStatus(issueID string) (bool, bool, error) {
	if !l.IsAvailable() {
		return false, false, ErrProviderNotAvailable
	}

	// Use linear issue view with JSON output
	cmd := exec.Command("linear", "issue", issueID, "--json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") ||
				strings.Contains(string(exitErr.Stderr), "Could not find") {
				return false, false, ErrNotFound
			}
		}
		return false, false, fmt.Errorf("failed to get Linear issue status: %w", err)
	}

	var result struct {
		State struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"state"`
		CompletedAt string `json:"completedAt"`
		CanceledAt  string `json:"canceledAt"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, false, fmt.Errorf("failed to parse Linear issue status: %w", err)
	}

	// Linear issues can be in various states
	// Type "completed" means the issue is done
	// Type "canceled" means it was closed without completion
	stateName := strings.ToLower(result.State.Name)
	stateType := strings.ToLower(result.State.Type)

	isClosed := stateType == linearStatusCompleted ||
		stateType == linearStatusCanceled ||
		stateName == linearStatusDone ||
		stateName == linearStatusCompleted ||
		stateName == linearStatusCanceled ||
		result.CompletedAt != "" ||
		result.CanceledAt != ""

	// Issue is completed if it's marked as completed (not just canceled)
	isCompleted := stateType == linearStatusCompleted ||
		stateName == linearStatusDone ||
		stateName == linearStatusCompleted ||
		result.CompletedAt != ""

	return isClosed, isCompleted, nil
}

// GetPRStatus is not applicable for Linear (no PR concept)
func (l *LinearProvider) GetPRStatus(prID string) (bool, error) {
	return false, fmt.Errorf("linear does not have pull requests")
}

// GetStatusForBranch attempts to determine the status based on branch name
func (l *LinearProvider) GetStatusForBranch(branchName string) (*IssueStatus, error) {
	providerType, id, found := ParseBranchName(branchName)
	if !found {
		return nil, fmt.Errorf("could not parse branch name: %s", branchName)
	}

	if providerType != "linear" {
		return nil, fmt.Errorf("unsupported branch type: %s", providerType)
	}

	status := &IssueStatus{
		ID:       id,
		Provider: "linear",
	}

	isClosed, isCompleted, err := l.GetIssueStatus(id)
	if err != nil {
		return nil, err
	}

	status.IsClosed = isClosed
	status.IsCompleted = isCompleted

	return status, nil
}
