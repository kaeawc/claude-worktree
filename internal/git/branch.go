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

// IsBranchMergedInto checks if a branch has been merged into another branch
// This is used to verify that a branch's changes are fully incorporated into the target branch
func IsBranchMergedInto(repoPath, branchName, targetBranch string) (bool, error) {
	// Use git branch --merged to check if the branch has been merged
	// This command lists all branches that have been merged into the current branch
	cmd := exec.Command("git", "branch", "--merged", targetBranch, "--list", branchName)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, it might be because the branch doesn't exist
		// or there's another error - we'll consider it not merged
		return false, nil
	}

	// If output contains the branch name, it has been merged
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return false, nil
	}

	// The output will be the branch name with optional leading spaces and asterisk
	// e.g., "  branch-name" or "* branch-name"
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == branchName {
			return true, nil
		}
	}

	return false, nil
}

// GetMergeBase returns the merge base (common ancestor) between two branches
func GetMergeBase(repoPath, branch1, branch2 string) (string, error) {
	cmd := exec.Command("git", "merge-base", branch1, branch2)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get merge base: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
