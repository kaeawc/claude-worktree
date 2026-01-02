package gitlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// Issue represents a GitLab issue
type Issue struct {
	IID         int      `json:"iid"` // Issue IID (internal ID, scoped to project)
	Title       string   `json:"title"`
	Description string   `json:"description"` // GitLab uses "description" not "body"
	State       string   `json:"state"`       // "opened" or "closed"
	Labels      []string `json:"labels"`
	WebURL      string   `json:"web_url"`
	Author      Author   `json:"author"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// Author represents a GitLab user
type Author struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}

// ListOpenIssues fetches open issues (up to limit)
// Uses: glab issue list --state opened --per-page <limit> --json
func (c *Client) ListOpenIssues(limit int) ([]Issue, error) {
	output, err := c.execGlabInRepo("issue", "list",
		"--state", "opened",
		"--per-page", strconv.Itoa(limit),
		"--json")
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// GetIssue fetches a specific issue by IID
// Uses: glab issue view <iid> --json
func (c *Client) GetIssue(iid int) (*Issue, error) {
	output, err := c.execGlabInRepo("issue", "view", strconv.Itoa(iid), "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to get issue #%d: %w", iid, err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	return &issue, nil
}

// IsIssueClosed checks if an issue is closed
func (c *Client) IsIssueClosed(iid int) (bool, error) {
	issue, err := c.GetIssue(iid)
	if err != nil {
		return false, err
	}
	return issue.State == "closed", nil
}

// SanitizedTitle returns sanitized title suitable for branch names
func (i *Issue) SanitizedTitle() string {
	title := i.Title

	// Truncate to 40 characters (before sanitization)
	if len(title) > 40 {
		title = title[:40]
	}

	// Use git.SanitizeBranchName for consistent sanitization
	return git.SanitizeBranchName(title)
}

// FormatForDisplay formats issue for display in lists
// Format: #<iid> | <title> | [label1] [label2]
func (i *Issue) FormatForDisplay() string {
	labels := ""
	if len(i.Labels) > 0 {
		labels = " | " + strings.Join(i.Labels, " ")
	}
	return fmt.Sprintf("#%d | %s%s", i.IID, i.Title, labels)
}

// BranchName generates the branch name for this issue
// Format: work/<iid>-<sanitized-title>
func (i *Issue) BranchName() string {
	return fmt.Sprintf("work/%d-%s", i.IID, i.SanitizedTitle())
}
