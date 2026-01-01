package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
}

// ListWorktrees returns all worktrees for the repository
func (r *Repository) ListWorktrees() ([]*Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.RootPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(string(output))
}

// parseWorktreeList parses the output of 'git worktree list --porcelain'
func parseWorktreeList(output string) ([]*Worktree, error) {
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
			current = &Worktree{Path: value}
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
		if err := enrichWorktree(wt); err != nil {
			// Log error but don't fail - continue with partial data
			continue
		}
	}

	return worktrees, scanner.Err()
}

// enrichWorktree adds additional information to the worktree
func enrichWorktree(wt *Worktree) error {
	// Get last commit timestamp
	timestamp, err := getLastCommitTimestamp(wt.Path)
	if err == nil {
		wt.LastCommitTime = timestamp
	} else {
		// Fallback to file modification time if no commits
		wt.LastCommitTime = getLastModificationTime(wt.Path)
	}

	// Get unpushed commit count
	if !wt.IsDetached && wt.Branch != "" {
		count, err := getUnpushedCommitCount(wt.Path, wt.Branch)
		if err == nil {
			wt.UnpushedCount = count
		}
	}

	return nil
}

// getLastCommitTimestamp returns the timestamp of the last commit in the worktree
func getLastCommitTimestamp(path string) (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// getLastModificationTime returns the most recent file modification time
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
func getUnpushedCommitCount(path, branch string) (int, error) {
	// First, try to get the upstream branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmd.Dir = path
	_, err := cmd.Output()

	var output []byte
	if err != nil {
		// No upstream branch configured, count all commits
		cmd = exec.Command("git", "rev-list", "--count", "HEAD")
		cmd.Dir = path
		output, err = cmd.Output()
		if err != nil {
			return 0, err
		}
	} else {
		// Count commits ahead of upstream
		cmd = exec.Command("git", "rev-list", "--count", "@{u}..HEAD")
		cmd.Dir = path
		output, err = cmd.Output()
		if err != nil {
			return 0, err
		}
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Age returns the duration since the last commit
func (w *Worktree) Age() time.Duration {
	return time.Since(w.LastCommitTime)
}

// CreateWorktree creates a new worktree with an existing branch
func (r *Repository) CreateWorktree(path, branchName string) error {
	cmd := exec.Command("git", "worktree", "add", path, branchName)
	cmd.Dir = r.RootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// CreateWorktreeWithNewBranch creates a new worktree with a new branch
func (r *Repository) CreateWorktreeWithNewBranch(path, branchName, baseBranch string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, path, baseBranch)
	cmd.Dir = r.RootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree with new branch: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// RemoveWorktree removes a worktree (force removal)
func (r *Repository) RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	cmd.Dir = r.RootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// PruneWorktrees removes worktree information for deleted directories
func (r *Repository) PruneWorktrees() error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = r.RootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w\nOutput: %s", err, string(output))
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
