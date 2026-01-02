package provider

import (
	"errors"
	"fmt"
)

// Provider type constants
const (
	ProviderTypeGitHubIssue = "github-issue"
	ProviderTypeGitHubPR    = "github-pr"
	ProviderTypeGitLabMR    = "gitlab-mr"
	ProviderTypeJira        = "jira"
	ProviderTypeLinear      = "linear"
)

// Branch prefixes
const (
	BranchPrefixWork  = "work/"
	BranchPrefixPR    = "pr/"
	BranchPrefixMR    = "mr/"
	BranchPrefixIssue = "issue/"
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
	// Try simple numeric patterns first (GitHub/GitLab)
	if id, found := extractNumericID(branchName, BranchPrefixWork, 5); found {
		return ProviderTypeGitHubIssue, id, true
	}

	if id, found := extractNumericID(branchName, BranchPrefixPR, 3); found {
		return ProviderTypeGitHubPR, id, true
	}

	if id, found := extractNumericID(branchName, BranchPrefixMR, 3); found {
		return ProviderTypeGitLabMR, id, true
	}

	// Try JIRA/Linear patterns (PROJ-123 format)
	if id, found := extractProjectID(branchName, BranchPrefixIssue, 6); found {
		return ProviderTypeJira, id, true
	}

	if id, found := extractProjectID(branchName, BranchPrefixWork, 5); found {
		// Could be JIRA or Linear - for now assume JIRA if uppercase
		return ProviderTypeJira, id, true
	}

	return "", "", false
}

// extractNumericID extracts a numeric ID after a prefix
func extractNumericID(branchName, prefix string, prefixLen int) (string, bool) {
	if len(branchName) <= prefixLen || branchName[:prefixLen] != prefix {
		return "", false
	}

	var num string
	for i := prefixLen; i < len(branchName) && branchName[i] >= '0' && branchName[i] <= '9'; i++ {
		num += string(branchName[i])
	}

	return num, num != ""
}

// extractProjectID extracts a project-based ID (e.g., PROJ-123) after a prefix
func extractProjectID(branchName, prefix string, prefixLen int) (string, bool) {
	if len(branchName) <= prefixLen {
		return "", false
	}

	// Handle variable length prefixes
	actualPrefixLen := len(prefix)
	if len(branchName) <= actualPrefixLen || branchName[:actualPrefixLen] != prefix {
		return "", false
	}

	var projectID string
	inNumber := false

	for i := actualPrefixLen; i < len(branchName); i++ {
		ch := branchName[i]
		if ch >= 'A' && ch <= 'Z' {
			projectID += string(ch)
		} else if ch == '-' {
			if len(projectID) > 0 {
				projectID += string(ch)
				inNumber = true
			} else {
				break
			}
		} else if ch >= '0' && ch <= '9' && inNumber {
			projectID += string(ch)
		} else {
			break
		}
	}

	return projectID, len(projectID) > 0 && inNumber
}

// DetectProvider determines which provider to use based on configuration and branch name
func DetectProvider(branchName, configuredProvider string) Provider {
	// If a specific provider is configured, use it
	switch configuredProvider {
	case "github":
		return &GitHubProvider{}
	case "gitlab":
		return &GitLabProvider{}
	case ProviderTypeJira:
		return &JiraProvider{}
	case ProviderTypeLinear:
		return &LinearProvider{}
	}

	// Otherwise, try to detect from branch name
	providerType, _, found := ParseBranchName(branchName)
	if !found {
		return nil
	}

	switch providerType {
	case ProviderTypeGitHubIssue, ProviderTypeGitHubPR:
		return &GitHubProvider{}
	case ProviderTypeGitLabMR:
		return &GitLabProvider{}
	case ProviderTypeJira:
		return &JiraProvider{}
	case ProviderTypeLinear:
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
