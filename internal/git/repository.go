package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repository represents a Git repository
type Repository struct {
	// RootPath is the absolute path to the git repository root
	RootPath string
	// WorktreeBase is the base directory for all worktrees (e.g., ~/worktrees/repo-name)
	WorktreeBase string
	// SourceFolder is the name of the repository directory
	SourceFolder string
	// Config provides access to git configuration for this repository
	Config *Config
}

// NewRepository creates a Repository instance from the current working directory
func NewRepository() (*Repository, error) {
	return NewRepositoryFromPath(".")
}

// NewRepositoryFromPath creates a Repository instance from the specified path
func NewRepositoryFromPath(path string) (*Repository, error) {
	// Check if we're in a git repository
	if !IsGitRepository(path) {
		return nil, fmt.Errorf("not a git repository (or any of the parent directories): %s", path)
	}

	// Get the repository root
	rootPath, err := GetRepositoryRoot(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get the source folder name
	sourceFolder := filepath.Base(rootPath)

	// Construct worktree base path: ~/worktrees/<repo-name>
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	worktreeBase := filepath.Join(homeDir, "worktrees", sourceFolder)

	return &Repository{
		RootPath:     rootPath,
		WorktreeBase: worktreeBase,
		SourceFolder: sourceFolder,
		Config:       NewConfig(rootPath),
	}, nil
}

// IsGitRepository checks if the given path is within a git repository
func IsGitRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// GetRepositoryRoot returns the absolute path to the git repository root
func GetRepositoryRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch returns the default branch name (main, master, etc.)
func (r *Repository) GetDefaultBranch() (string, error) {
	// Try to get from remote HEAD
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = r.RootPath
	if output, err := cmd.Output(); err == nil {
		// Output format: refs/remotes/origin/main
		ref := strings.TrimSpace(string(output))
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Try common default branches in order
	defaultBranches := []string{"main", "master"}
	for _, branch := range defaultBranches {
		// Check local branch first
		if r.BranchExists(branch) {
			return branch, nil
		}
		// Check remote branch
		if r.remoteBranchExists("origin/" + branch) {
			return branch, nil
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

// BranchExists checks if a local branch exists
func (r *Repository) BranchExists(branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = r.RootPath
	return cmd.Run() == nil
}

// remoteBranchExists checks if a remote branch exists
func (r *Repository) remoteBranchExists(refName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/"+refName)
	cmd.Dir = r.RootPath
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current branch name, or empty string if in detached HEAD
func (r *Repository) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = r.RootPath
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

// CreateBranch creates a new branch from the specified base branch
func (r *Repository) CreateBranch(branchName, baseBranch string) error {
	cmd := exec.Command("git", "branch", branchName, baseBranch)
	cmd.Dir = r.RootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// DeleteBranch deletes a branch (force delete)
func (r *Repository) DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = r.RootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// EnrichWorktreeWithMergeStatus adds merge status information to a worktree
// This checks both git merge status and external provider status
func (r *Repository) EnrichWorktreeWithMergeStatus(wt *Worktree) error {
	// Skip if no branch (detached HEAD)
	if wt.Branch == "" || wt.IsDetached {
		return nil
	}

	// Get the default branch
	defaultBranch, err := r.GetDefaultBranch()
	if err != nil {
		// If we can't get the default branch, continue without merge check
		return nil
	}

	// Check if branch is merged into default branch
	isMerged, err := IsBranchMergedInto(r.RootPath, wt.Branch, defaultBranch)
	if err == nil {
		wt.IsBranchMerged = isMerged
	}

	return nil
}

// ListWorktreesWithMergeStatus returns all worktrees enriched with merge status
func (r *Repository) ListWorktreesWithMergeStatus() ([]*Worktree, error) {
	worktrees, err := r.ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if err := r.EnrichWorktreeWithMergeStatus(wt); err != nil {
			// Continue on error, just skip enrichment for this worktree
			continue
		}
	}

	return worktrees, nil
}

// GetCleanupCandidates returns worktrees that should be cleaned up
// Returns merged worktrees first, then stale worktrees
func (r *Repository) GetCleanupCandidates() ([]*Worktree, error) {
	worktrees, err := r.ListWorktreesWithMergeStatus()
	if err != nil {
		return nil, err
	}

	var merged []*Worktree
	var stale []*Worktree

	for _, wt := range worktrees {
		// Skip the main worktree (the repository root)
		if wt.Path == r.RootPath {
			continue
		}

		if wt.IsMerged() {
			merged = append(merged, wt)
		} else if wt.IsStale() {
			stale = append(stale, wt)
		}
	}

	// Return merged first, then stale
	return append(merged, stale...), nil
}
