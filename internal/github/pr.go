package github

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number            int             `json:"number"`
	Title             string          `json:"title"`
	Body              string          `json:"body"`
	State             string          `json:"state"` // "OPEN", "CLOSED", "MERGED"
	Author            Author          `json:"author"`
	HeadRefName       string          `json:"headRefName"`
	BaseRefName       string          `json:"baseRefName"`
	Labels            []Label         `json:"labels"`
	URL               string          `json:"url"`
	IsDraft           bool            `json:"isDraft"`
	ReviewRequests    []ReviewRequest `json:"reviewRequests"`
	Additions         int             `json:"additions"`
	Deletions         int             `json:"deletions"`
	ChangedFiles      int             `json:"changedFiles"`
	StatusCheckRollup []StatusCheck   `json:"statusCheckRollup"`
}

// Author represents a GitHub user
type Author struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	IsBot bool   `json:"is_bot"`
}

// ReviewRequest represents a review request
type ReviewRequest struct {
	Login string `json:"login"`
}

// StatusCheck represents a CI check result
type StatusCheck struct {
	TypeName   string `json:"__typename"`
	Name       string `json:"name"`
	Status     string `json:"status"`     // "COMPLETED", "IN_PROGRESS", etc.
	Conclusion string `json:"conclusion"` // "SUCCESS", "FAILURE", "NEUTRAL", etc.
}

// ListOpenPRs fetches open pull requests (up to limit)
// Uses: gh pr list --limit <limit> --state open --json <fields>
func (c *Client) ListOpenPRs(limit int) ([]PullRequest, error) {
	fields := "number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup"
	output, err := c.execGHInRepo("pr", "list",
		"--limit", strconv.Itoa(limit),
		"--state", "open",
		"--json", fields)
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	var prs []PullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PRs: %w", err)
	}

	return prs, nil
}

// GetPR fetches a specific pull request by number
// Uses: gh pr view <number> --json <fields>
func (c *Client) GetPR(number int) (*PullRequest, error) {
	fields := "number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup"
	output, err := c.execGHInRepo("pr", "view", strconv.Itoa(number),
		"--json", fields)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %w", number, err)
	}

	var pr PullRequest
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	return &pr, nil
}

// IsPRMerged checks if a pull request is merged
func (c *Client) IsPRMerged(number int) (bool, error) {
	pr, err := c.GetPR(number)
	if err != nil {
		return false, err
	}

	return pr.State == "MERGED", nil
}

// SanitizedTitle returns sanitized title suitable for branch names (max 40 chars)
func (pr *PullRequest) SanitizedTitle() string {
	title := pr.Title

	// Lowercase
	title = strings.ToLower(title)

	// Truncate to 40 characters
	if len(title) > 40 {
		title = title[:40]
	}

	// Use git.SanitizeBranchName for consistent sanitization
	return git.SanitizeBranchName(title)
}

// FormatForDisplay formats PR for display in lists
// Format: #<number> | <title> | @<author> | +<additions> -<deletions> | [label1] [label2]
func (pr *PullRequest) FormatForDisplay() string {
	var parts []string

	// Add number, title, and author
	parts = append(parts, fmt.Sprintf("#%d | %s | @%s", pr.Number, pr.Title, pr.Author.Login))

	// Add change stats
	parts = append(parts, fmt.Sprintf("| +%d -%d", pr.Additions, pr.Deletions))

	// Add labels if present
	if len(pr.Labels) > 0 {
		labelNames := make([]string, len(pr.Labels))
		for idx, label := range pr.Labels {
			labelNames[idx] = fmt.Sprintf("[%s]", label.Name)
		}
		parts = append(parts, "|", strings.Join(labelNames, " "))
	}

	return strings.Join(parts, " ")
}

// BranchName generates the branch name for this PR
// Format: pr/<number>-<sanitized-title>
func (pr *PullRequest) BranchName() string {
	return fmt.Sprintf("pr/%d-%s", pr.Number, pr.SanitizedTitle())
}

// HasMergeConflicts checks if PR has merge conflicts with base branch
// Uses: gh pr view <number> --json mergeable
func (c *Client) HasMergeConflicts(number int) (bool, error) {
	output, err := c.execGHInRepo("pr", "view", strconv.Itoa(number),
		"--json", "mergeable")
	if err != nil {
		return false, fmt.Errorf("failed to check PR #%d mergeable status: %w", number, err)
	}

	var result struct {
		Mergeable string `json:"mergeable"` // "MERGEABLE", "CONFLICTING", "UNKNOWN"
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("failed to parse mergeable status: %w", err)
	}

	return result.Mergeable == "CONFLICTING", nil
}

// GetPRDiff fetches the diff for a pull request
// Uses: gh pr diff <number>
func (c *Client) GetPRDiff(number int) (string, error) {
	output, err := c.execGHInRepo("pr", "diff", strconv.Itoa(number))
	if err != nil {
		return "", fmt.Errorf("failed to get PR #%d diff: %w", number, err)
	}

	return string(output), nil
}

// AllChecksPass returns true if all status checks have passed
func (pr *PullRequest) AllChecksPass() bool {
	if len(pr.StatusCheckRollup) == 0 {
		return true // No checks configured
	}

	for _, check := range pr.StatusCheckRollup {
		if check.Status != "COMPLETED" || check.Conclusion != "SUCCESS" {
			return false
		}
	}

	return true
}

// ChangeSize returns a categorical size based on lines changed
func (pr *PullRequest) ChangeSize() string {
	total := pr.Additions + pr.Deletions
	switch {
	case total < 50:
		return "XS"
	case total < 200:
		return "S"
	case total < 500:
		return "M"
	case total < 1000:
		return "L"
	default:
		return "XL"
	}
}

// IsRequestedReviewer checks if a given username is a requested reviewer
func (pr *PullRequest) IsRequestedReviewer(username string) bool {
	for _, req := range pr.ReviewRequests {
		if req.Login == username {
			return true
		}
	}
	return false
}
