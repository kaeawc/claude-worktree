package provider

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const (
	jiraStatusDone     = "done"
	jiraStatusResolved = "resolved"
	jiraStatusClosed   = "closed"
	jiraStatusFixed    = "fixed"
)

// JiraProvider implements Provider for JIRA
type JiraProvider struct{}

// Name returns the provider name
func (j *JiraProvider) Name() string {
	return "jira"
}

// IsAvailable checks if jira CLI is installed
func (j *JiraProvider) IsAvailable() bool {
	cmd := exec.Command("jira", "version")
	err := cmd.Run()
	return err == nil
}

// GetIssueStatus checks if a JIRA issue is resolved/done
func (j *JiraProvider) GetIssueStatus(issueID string) (bool, bool, error) {
	if !j.IsAvailable() {
		return false, false, ErrProviderNotAvailable
	}

	// Use jira view with JSON output
	// Note: The exact CLI command may vary depending on the jira CLI tool used
	// This assumes go-jira (https://github.com/go-jira/jira)
	cmd := exec.Command("jira", "view", issueID, "--template", "json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") ||
				strings.Contains(string(exitErr.Stderr), "does not exist") {
				return false, false, ErrNotFound
			}
		}

		return false, false, fmt.Errorf("failed to get JIRA issue status: %w", err)
	}

	var result struct {
		Fields struct {
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Resolution struct {
				Name string `json:"name"`
			} `json:"resolution"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, false, fmt.Errorf("failed to parse JIRA issue status: %w", err)
	}

	statusName := strings.ToLower(result.Fields.Status.Name)
	resolutionName := strings.ToLower(result.Fields.Resolution.Name)

	// Issue is closed if status is Done, Resolved, Closed, etc.
	isClosed := statusName == jiraStatusDone ||
		statusName == jiraStatusResolved ||
		statusName == jiraStatusClosed ||
		resolutionName == jiraStatusDone ||
		resolutionName == jiraStatusResolved ||
		resolutionName == jiraStatusFixed

	// Issue is completed if it's closed with a positive resolution
	isCompleted := isClosed && (resolutionName == jiraStatusDone ||
		resolutionName == jiraStatusResolved ||
		resolutionName == jiraStatusFixed)

	return isClosed, isCompleted, nil
}

// GetPRStatus is not applicable for JIRA (no PR concept)
func (j *JiraProvider) GetPRStatus(prID string) (bool, error) {
	return false, fmt.Errorf("jira does not have pull requests")
}

// GetStatusForBranch attempts to determine the status based on branch name
func (j *JiraProvider) GetStatusForBranch(branchName string) (*IssueStatus, error) {
	providerType, id, found := ParseBranchName(branchName)
	if !found {
		return nil, fmt.Errorf("could not parse branch name: %s", branchName)
	}

	if providerType != "jira" {
		return nil, fmt.Errorf("unsupported branch type: %s", providerType)
	}

	status := &IssueStatus{
		ID:       id,
		Provider: "jira",
	}

	isClosed, isCompleted, err := j.GetIssueStatus(id)
	if err != nil {
		return nil, err
	}

	status.IsClosed = isClosed
	status.IsCompleted = isCompleted

	return status, nil
}
