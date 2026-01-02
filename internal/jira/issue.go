package jira

import (
	"fmt"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// Issue represents a JIRA issue
type Issue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Resolution struct {
			Name string `json:"name"`
		} `json:"resolution"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Creator struct {
			DisplayName string `json:"displayName"`
		} `json:"creator"`
		Created string   `json:"created"`
		Updated string   `json:"updated"`
		Labels  []string `json:"labels"`
		URL     string   `json:"url"`
	} `json:"fields"`
}

// ID returns the issue ID (key) for compatibility with providers.Issue
func (i *Issue) ID() string {
	return i.Key
}

// Title returns the issue summary
func (i *Issue) Title() string {
	return i.Fields.Summary
}

// Body returns the issue description
func (i *Issue) Body() string {
	return i.Fields.Description
}

// Status returns the issue status
func (i *Issue) Status() string {
	return i.Fields.Status.Name
}

// SanitizedTitle returns sanitized title suitable for branch names
func (i *Issue) SanitizedTitle() string {
	title := i.Fields.Summary

	// Lowercase
	title = strings.ToLower(title)

	// Truncate to 40 characters
	if len(title) > 40 {
		title = title[:40]
	}

	// Use git.SanitizeBranchName for consistent sanitization
	return git.SanitizeBranchName(title)
}

// FormatForDisplay formats issue for display in lists
// Format: PROJ-123 | <title> | [label1] [label2]
func (i *Issue) FormatForDisplay() string {
	var parts []string

	// Add key and title
	parts = append(parts, fmt.Sprintf("%s | %s", i.Key, i.Fields.Summary))

	// Add labels if present
	if len(i.Fields.Labels) > 0 {
		labelNames := make([]string, len(i.Fields.Labels))
		for idx, label := range i.Fields.Labels {
			labelNames[idx] = fmt.Sprintf("[%s]", label)
		}
		parts = append(parts, "|", strings.Join(labelNames, " "))
	}

	return strings.Join(parts, " ")
}

// BranchName generates the branch name for this issue
// Format: work/PROJ-123-<sanitized-title>
func (i *Issue) BranchName() string {
	return fmt.Sprintf("work/%s-%s", i.Key, i.SanitizedTitle())
}

// IsClosed checks if the issue is resolved/done
func (i *Issue) IsClosed() bool {
	status := i.Fields.Status.Name
	resolution := i.Fields.Resolution.Name

	return isResolvedStatus(status) || isResolvedResolution(resolution)
}

// isResolvedStatus checks if a status string indicates resolution
func isResolvedStatus(status string) bool {
	resolvedStatuses := []string{"done", "resolved", "closed"}
	for _, s := range resolvedStatuses {
		if strings.EqualFold(status, s) {
			return true
		}
	}
	return false
}

// isResolvedResolution checks if a resolution string indicates completion
func isResolvedResolution(resolution string) bool {
	resolvedResolutions := []string{"done", "resolved", "fixed"}
	for _, r := range resolvedResolutions {
		if strings.EqualFold(resolution, r) {
			return true
		}
	}
	return false
}
