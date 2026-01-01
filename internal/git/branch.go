package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetCurrentBranchInWorktree returns the current branch name in a specific worktree
func GetCurrentBranchInWorktree(worktreePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(output))
	// If in detached HEAD state, git returns "HEAD"
	if branch == "HEAD" {
		return "", nil
	}
	return branch, nil
}

// GetUpstreamBranch returns the upstream branch for a given branch in a worktree
func GetUpstreamBranch(worktreePath, branchName string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", branchName+"@{u}")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no upstream branch configured")
	}
	return strings.TrimSpace(string(output)), nil
}

// SanitizeBranchName converts a string into a valid git branch name
// Follows the logic from _aw_sanitize_branch_name in aw.sh
func SanitizeBranchName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace non-alphanumeric characters with dashes
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	name = result.String()

	// Collapse multiple dashes into single dash
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Remove leading and trailing dashes
	name = strings.Trim(name, "-")

	return name
}
