package git

import (
	"fmt"
	"strings"
)

// GetCurrentBranchInWorktree returns the current branch name in a specific worktree
func GetCurrentBranchInWorktree(worktreePath string) (string, error) {
	executor := NewGitExecutor()
	return getCurrentBranchInWorktree(worktreePath, executor)
}

// getCurrentBranchInWorktree returns the current branch name in a specific worktree using provided executor
func getCurrentBranchInWorktree(worktreePath string, executor GitExecutor) (string, error) {
	output, err := executor.ExecuteInDir(worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	// If in detached HEAD state, git returns "HEAD"
	if output == "HEAD" {
		return "", nil
	}
	return output, nil
}

// GetUpstreamBranch returns the upstream branch for a given branch in a worktree
func GetUpstreamBranch(worktreePath, branchName string) (string, error) {
	executor := NewGitExecutor()
	return getUpstreamBranch(worktreePath, branchName, executor)
}

// getUpstreamBranch returns the upstream branch for a given branch in a worktree using provided executor
func getUpstreamBranch(worktreePath, branchName string, executor GitExecutor) (string, error) {
	output, err := executor.ExecuteInDir(worktreePath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", branchName+"@{u}")
	if err != nil {
		return "", fmt.Errorf("no upstream branch configured")
	}
	return output, nil
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
	executor := NewGitExecutor()
	return isBranchMergedInto(repoPath, branchName, targetBranch, executor)
}

// isBranchMergedInto checks if a branch has been merged into another branch using provided executor
func isBranchMergedInto(repoPath, branchName, targetBranch string, executor GitExecutor) (bool, error) {
	// Use git branch --merged to check if the branch has been merged
	// This command lists all branches that have been merged into the current branch
	output, err := executor.ExecuteInDir(repoPath, "branch", "--merged", targetBranch, "--list", branchName)
	if err != nil {
		// If the command fails, it might be because the branch doesn't exist
		// or there's another error - we'll consider it not merged
		return false, nil
	}

	// If output contains the branch name, it has been merged
	if output == "" {
		return false, nil
	}

	// The output will be the branch name with optional leading spaces and asterisk
	// e.g., "  branch-name" or "* branch-name"
	lines := strings.Split(output, "\n")
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
	executor := NewGitExecutor()
	return getMergeBase(repoPath, branch1, branch2, executor)
}

// getMergeBase returns the merge base (common ancestor) between two branches using provided executor
func getMergeBase(repoPath, branch1, branch2 string, executor GitExecutor) (string, error) {
	output, err := executor.ExecuteInDir(repoPath, "merge-base", branch1, branch2)
	if err != nil {
		return "", fmt.Errorf("failed to get merge base: %w", err)
	}
	return output, nil
}
