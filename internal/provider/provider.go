package provider

import (
	"errors"
	"fmt"
)

// Provider represents an external issue tracking system
type Provider interface {
	// GetIssueStatus checks if an issue is closed/completed
	// Returns: isClosed, isCompleted (merged/resolved), error
	GetIssueStatus(issueID string) (bool, bool, error)

	// GetPRStatus checks if a pull/merge request is merged
	// Returns: isMerged, error
	GetPRStatus(prID string) (bool, error)

	// Name returns the provider name
	Name() string

	// IsAvailable checks if the CLI tool is installed and configured
	IsAvailable() bool
}

// IssueStatus represents the status of an issue or PR
type IssueStatus struct {
	ID          string
	IsClosed    bool
	IsCompleted bool // Merged for PRs, Completed/Resolved for issues
	Title       string
	Provider    string
}

// ErrProviderNotAvailable is returned when the provider CLI is not available
var ErrProviderNotAvailable = errors.New("provider CLI tool not available")

// ErrNotFound is returned when an issue or PR is not found
var ErrNotFound = errors.New("issue or PR not found")

// ParseBranchName attempts to extract issue/PR information from a branch name
// Supported formats:
//   - work/123-description (GitHub issue)
//   - pr/456-description (GitHub PR)
//   - issue/PROJ-123-description (JIRA)
//   - mr/789-description (GitLab MR)
func ParseBranchName(branchName string) (providerType, id string, found bool) {
	// We'll implement more sophisticated parsing later
	// For now, return basic patterns

	// GitHub issue: work/123-*
	if len(branchName) > 5 && branchName[:5] == "work/" {
		// Extract number after work/
		var num string
		for i := 5; i < len(branchName) && branchName[i] >= '0' && branchName[i] <= '9'; i++ {
			num += string(branchName[i])
		}
		if num != "" {
			return "github-issue", num, true
		}
	}

	// GitHub PR: pr/456-*
	if len(branchName) > 3 && branchName[:3] == "pr/" {
		var num string
		for i := 3; i < len(branchName) && branchName[i] >= '0' && branchName[i] <= '9'; i++ {
			num += string(branchName[i])
		}
		if num != "" {
			return "github-pr", num, true
		}
	}

	// GitLab MR: mr/789-*
	if len(branchName) > 3 && branchName[:3] == "mr/" {
		var num string
		for i := 3; i < len(branchName) && branchName[i] >= '0' && branchName[i] <= '9'; i++ {
			num += string(branchName[i])
		}
		if num != "" {
			return "gitlab-mr", num, true
		}
	}

	// JIRA: issue/PROJ-123-* or work/PROJ-123-*
	if len(branchName) > 6 && (branchName[:6] == "issue/" || branchName[:5] == "work/") {
		start := 6
		if branchName[:5] == "work/" {
			start = 5
		}

		// Look for PROJ-123 pattern
		var jiraID string
		inNumber := false
		for i := start; i < len(branchName); i++ {
			ch := branchName[i]
			if ch >= 'A' && ch <= 'Z' {
				jiraID += string(ch)
			} else if ch == '-' {
				if len(jiraID) > 0 {
					jiraID += string(ch)
					inNumber = true
				} else {
					break
				}
			} else if ch >= '0' && ch <= '9' && inNumber {
				jiraID += string(ch)
			} else {
				break
			}
		}

		if len(jiraID) > 0 && inNumber {
			return "jira", jiraID, true
		}
	}

	// Linear: work/TEAM-123-* pattern
	// Linear IDs are typically like "ENG-123"
	if len(branchName) > 5 && branchName[:5] == "work/" {
		var linearID string
		inNumber := false
		for i := 5; i < len(branchName); i++ {
			ch := branchName[i]
			if ch >= 'A' && ch <= 'Z' {
				linearID += string(ch)
			} else if ch == '-' {
				if len(linearID) > 0 {
					linearID += string(ch)
					inNumber = true
				} else {
					break
				}
			} else if ch >= '0' && ch <= '9' && inNumber {
				linearID += string(ch)
			} else {
				break
			}
		}

		if len(linearID) > 0 && inNumber {
			return "linear", linearID, true
		}
	}

	return "", "", false
}

// DetectProvider determines which provider to use based on configuration and branch name
func DetectProvider(branchName, configuredProvider string) Provider {
	// If a specific provider is configured, use it
	switch configuredProvider {
	case "github":
		return &GitHubProvider{}
	case "gitlab":
		return &GitLabProvider{}
	case "jira":
		return &JiraProvider{}
	case "linear":
		return &LinearProvider{}
	}

	// Otherwise, try to detect from branch name
	providerType, _, found := ParseBranchName(branchName)
	if !found {
		return nil
	}

	switch providerType {
	case "github-issue", "github-pr":
		return &GitHubProvider{}
	case "gitlab-mr":
		return &GitLabProvider{}
	case "jira":
		return &JiraProvider{}
	case "linear":
		return &LinearProvider{}
	}

	return nil
}

// FormatStatusString creates a display string for issue/PR status
func FormatStatusString(status *IssueStatus) string {
	if status == nil {
		return ""
	}

	if status.IsCompleted {
		return fmt.Sprintf("[merged #%s]", status.ID)
	}

	if status.IsClosed {
		return fmt.Sprintf("[closed #%s]", status.ID)
	}

	return ""
}
