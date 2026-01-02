package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Worktree represents a git worktree
type Worktree struct {
	// Path is the absolute path to the worktree
	Path string
	// Branch is the branch name, empty if detached HEAD
	Branch string
	// HEAD is the commit SHA
	HEAD string
	// IsDetached indicates if the worktree is in detached HEAD state
	IsDetached bool
	// LastCommitTime is the timestamp of the last commit
	LastCommitTime time.Time
	// UnpushedCount is the number of unpushed commits
	UnpushedCount int
	// IsBranchMerged indicates if the branch has been merged into the default branch
	IsBranchMerged bool
	// IssueStatus holds the status from external providers (GitHub, JIRA, etc.)
	IssueStatus *IssueStatus
	// executor is the git command executor for this worktree
	executor GitExecutor
	// TODO: Add FileSystem field once the FileSystem interface is created
	// filesystem FileSystem
}

// IssueStatus represents the status of an issue or PR from external providers
type IssueStatus struct {
	// Provider is the name of the provider (github, gitlab, jira, linear)
	Provider string
	// ID is the issue/PR identifier
	ID string
	// IsClosed indicates if the issue/PR is closed
	IsClosed bool
	// IsCompleted indicates if the issue/PR is merged/resolved/completed
	IsCompleted bool
	// Title is the issue/PR title (optional)
	Title string
}

// ListWorktrees returns all worktrees for the repository
func (r *Repository) ListWorktrees() ([]*Worktree, error) {
	output, err := r.executor.ExecuteInDir(r.RootPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(output, r.executor)
}

// parseWorktreeList parses the output of 'git worktree list --porcelain'
func parseWorktreeList(output string, executor GitExecutor) ([]*Worktree, error) {
	var worktrees []*Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line separates worktrees
			if current != nil {
				worktrees = append(worktrees, current)
				current = nil
			}
			continue
		}

		// Parse fields
		parts := strings.SplitN(line, " ", 2)
		field := parts[0]

		// Handle detached field (which has no value)
		if field == "detached" {
			if current != nil {
				current.IsDetached = true
			}
			continue
		}

		// All other fields require a value
		if len(parts) < 2 {
			continue
		}

		value := parts[1]

		switch field {
		case "worktree":
			current = &Worktree{Path: value, executor: executor}
		case "HEAD":
			if current != nil {
				current.HEAD = value
			}
		case "branch":
			if current != nil {
				// Format: refs/heads/branch-name
				branchParts := strings.Split(value, "/")
				if len(branchParts) >= 3 {
					current.Branch = strings.Join(branchParts[2:], "/")
				}
			}
		}
	}

	// Add the last worktree if exists
	if current != nil {
		worktrees = append(worktrees, current)
	}

	// Enrich worktrees with additional information
	for _, wt := range worktrees {
		if err := enrichWorktree(wt, executor); err != nil {
			// Log error but don't fail - continue with partial data
			continue
		}
	}

	return worktrees, scanner.Err()
}

// enrichWorktree adds additional information to the worktree
func enrichWorktree(wt *Worktree, executor GitExecutor) error {
	// Get last commit timestamp
	timestamp, err := getLastCommitTimestamp(wt.Path, executor)
	if err == nil {
		wt.LastCommitTime = timestamp
	} else {
		// Fallback to file modification time if no commits
		wt.LastCommitTime = getLastModificationTime(wt.Path)
	}

	// Get unpushed commit count
	if !wt.IsDetached && wt.Branch != "" {
		count, err := getUnpushedCommitCount(wt.Path, wt.Branch, executor)
		if err == nil {
			wt.UnpushedCount = count
		}
	}

	return nil
}

// getLastCommitTimestamp returns the timestamp of the last commit in the worktree
func getLastCommitTimestamp(path string, executor GitExecutor) (time.Time, error) {
	output, err := executor.ExecuteInDir(path, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}, err
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// getLastModificationTime returns the most recent file modification time
// TODO: Refactor to accept FileSystem parameter once the FileSystem interface is created
// Will need to replace: filepath.Walk, filepath.Rel, filepath.SkipDir, os.PathSeparator, os.FileInfo
func getLastModificationTime(path string) time.Time {
	var latestTime time.Time

	// Walk up to 3 levels deep, excluding .git directory
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Limit depth to 3 levels
		relPath, _ := filepath.Rel(path, p)
		if strings.Count(relPath, string(os.PathSeparator)) > 3 {
			return filepath.SkipDir
		}

		if !info.IsDir() && info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}

		return nil
	})

	if latestTime.IsZero() {
		return time.Now()
	}

	return latestTime
}

// getUnpushedCommitCount returns the number of unpushed commits
func getUnpushedCommitCount(path, branch string, executor GitExecutor) (int, error) {
	// First, try to get the upstream branch
	_, err := executor.ExecuteInDir(path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")

	var output string
	if err != nil {
		// No upstream branch configured, count all commits
		output, err = executor.ExecuteInDir(path, "rev-list", "--count", "HEAD")
		if err != nil {
			return 0, err
		}
	} else {
		// Count commits ahead of upstream
		output, err = executor.ExecuteInDir(path, "rev-list", "--count", "@{u}..HEAD")
		if err != nil {
			return 0, err
		}
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Age returns the duration since the last commit
func (w *Worktree) Age() time.Duration {
	return time.Since(w.LastCommitTime)
}

// IsStale returns true if the worktree is older than 4 days
func (w *Worktree) IsStale() bool {
	return w.Age() > 4*24*time.Hour
}

// IsMerged returns true if both the branch is merged AND the issue/PR is completed
func (w *Worktree) IsMerged() bool {
	// A worktree is considered merged if both:
	// 1. The git branch has been merged into the default branch
	// 2. The associated issue/PR is completed (if we have that info)

	// If we don't have issue status, just check git merge status
	if w.IssueStatus == nil {
		return w.IsBranchMerged
	}

	// Both must be true for full merge confirmation
	return w.IsBranchMerged && w.IssueStatus.IsCompleted
}

// ShouldCleanup returns true if the worktree is a candidate for cleanup
// Either it's merged or it's stale
func (w *Worktree) ShouldCleanup() bool {
	return w.IsMerged() || w.IsStale()
}

// CleanupReason returns a string describing why this worktree should be cleaned up
func (w *Worktree) CleanupReason() string {
	if w.IsMerged() {
		if w.IssueStatus != nil {
			return fmt.Sprintf("merged (#%s)", w.IssueStatus.ID)
		}
		return "merged"
	}
	if w.IsStale() {
		days := int(w.Age().Hours() / 24)
		return fmt.Sprintf("stale (%d days old)", days)
	}
	return ""
}

// CreateWorktree creates a new worktree with an existing branch
func (r *Repository) CreateWorktree(path, branchName string) error {
	_, err := r.executor.ExecuteInDir(r.RootPath, "worktree", "add", path, branchName)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	return nil
}

// CreateWorktreeWithNewBranch creates a new worktree with a new branch
func (r *Repository) CreateWorktreeWithNewBranch(path, branchName, baseBranch string) error {
	_, err := r.executor.ExecuteInDir(r.RootPath, "worktree", "add", "-b", branchName, path, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to create worktree with new branch: %w", err)
	}
	return nil
}

// RemoveWorktree removes a worktree (force removal)
func (r *Repository) RemoveWorktree(path string) error {
	_, err := r.executor.ExecuteInDir(r.RootPath, "worktree", "remove", "--force", path)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}

// PruneWorktrees removes worktree information for deleted directories
func (r *Repository) PruneWorktrees() error {
	_, err := r.executor.ExecuteInDir(r.RootPath, "worktree", "prune")
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}
	return nil
}

// GetWorktreeForBranch returns the worktree for a specific branch, or nil if none exists
func (r *Repository) GetWorktreeForBranch(branchName string) (*Worktree, error) {
	worktrees, err := r.ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branchName {
			return wt, nil
		}
	}

	return nil, nil
}
