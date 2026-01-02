package gitlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// MergeRequest represents a GitLab merge request
type MergeRequest struct {
	IID            int      `json:"iid"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	State          string   `json:"state"`        // "opened", "closed", "merged"
	MergeStatus    string   `json:"merge_status"` // "can_be_merged", "cannot_be_merged", etc.
	Author         Author   `json:"author"`
	SourceBranch   string   `json:"source_branch"`
	TargetBranch   string   `json:"target_branch"`
	Labels         []string `json:"labels"`
	WebURL         string   `json:"web_url"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	WorkInProgress bool     `json:"work_in_progress"` // GitLab's draft equivalent
	ChangesCount   string   `json:"changes_count"`
	UserNotesCount int      `json:"user_notes_count"`
}

// ListOpenMRs fetches open merge requests (up to limit)
// Uses: glab mr list --state opened --per-page <limit> --json
func (c *Client) ListOpenMRs(limit int) ([]MergeRequest, error) {
	output, err := c.execGlabInRepo("mr", "list",
		"--state", "opened",
		"--per-page", strconv.Itoa(limit),
		"--json")
	if err != nil {
		return nil, fmt.Errorf("failed to list merge requests: %w", err)
	}

	var mrs []MergeRequest
	if err := json.Unmarshal(output, &mrs); err != nil {
		return nil, fmt.Errorf("failed to parse merge requests: %w", err)
	}

	return mrs, nil
}

// GetMR fetches a specific merge request by IID
// Uses: glab mr view <iid> --json
func (c *Client) GetMR(iid int) (*MergeRequest, error) {
	output, err := c.execGlabInRepo("mr", "view", strconv.Itoa(iid), "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request !%d: %w", iid, err)
	}

	var mr MergeRequest
	if err := json.Unmarshal(output, &mr); err != nil {
		return nil, fmt.Errorf("failed to parse merge request: %w", err)
	}

	return &mr, nil
}

// IsMRMerged checks if a merge request is merged
func (c *Client) IsMRMerged(iid int) (bool, error) {
	mr, err := c.GetMR(iid)
	if err != nil {
		return false, err
	}
	return mr.State == "merged", nil
}

// HasMergeConflicts checks if MR has merge conflicts
func (c *Client) HasMergeConflicts(iid int) (bool, error) {
	mr, err := c.GetMR(iid)
	if err != nil {
		return false, err
	}
	return mr.MergeStatus != "can_be_merged" && mr.MergeStatus != "can_be_merged_automerge", nil
}

// GetMRDiff fetches the diff for a merge request
// Uses: glab mr diff <iid>
func (c *Client) GetMRDiff(iid int) (string, error) {
	output, err := c.execGlabInRepo("mr", "diff", strconv.Itoa(iid))
	if err != nil {
		return "", fmt.Errorf("failed to get merge request diff: %w", err)
	}
	return string(output), nil
}

// SanitizedTitle returns sanitized title suitable for branch names
func (mr *MergeRequest) SanitizedTitle() string {
	title := mr.Title

	// Truncate to 40 characters (before sanitization)
	if len(title) > 40 {
		title = title[:40]
	}

	// Use git.SanitizeBranchName for consistent sanitization
	return git.SanitizeBranchName(title)
}

// FormatForDisplay formats MR for display in lists
// Format: !<iid> | <title> | @<author> | [label1] [label2]
func (mr *MergeRequest) FormatForDisplay() string {
	author := mr.Author.Username
	labels := ""
	if len(mr.Labels) > 0 {
		labels = " | " + strings.Join(mr.Labels, " ")
	}
	return fmt.Sprintf("!%d | %s | @%s%s", mr.IID, mr.Title, author, labels)
}

// BranchName generates the branch name for this MR
// Format: mr/<iid>-<sanitized-title>
func (mr *MergeRequest) BranchName() string {
	return fmt.Sprintf("mr/%d-%s", mr.IID, mr.SanitizedTitle())
}
