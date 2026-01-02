package git

import (
	"fmt"
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
	// executor handles git command execution
	executor GitExecutor
	// filesystem handles filesystem operations
	filesystem FileSystem
}

// NewRepository creates a Repository instance from the current working directory
func NewRepository() (*Repository, error) {
	return NewRepositoryFromPath(".")
}

// NewRepositoryFromPath creates a Repository instance from the specified path
func NewRepositoryFromPath(path string) (*Repository, error) {
	executor := NewGitExecutor()
	filesystem := NewFileSystem()
	return NewRepositoryFromPathWithDeps(path, executor, filesystem)
}

// NewRepositoryWithDeps creates a Repository instance with provided dependencies
// This is useful for testing where you can inject mock/fake implementations
func NewRepositoryWithDeps(executor GitExecutor, filesystem FileSystem) (*Repository, error) {
	return NewRepositoryFromPathWithDeps(".", executor, filesystem)
}

// NewRepositoryFromPathWithDeps creates a Repository instance from the specified path with dependencies
func NewRepositoryFromPathWithDeps(path string, executor GitExecutor, filesystem FileSystem) (*Repository, error) {
	// Check if we're in a git repository
	if !isGitRepository(path, executor) {
		return nil, fmt.Errorf("not a git repository (or any of the parent directories): %s", path)
	}

	// Get the repository root
	rootPath, err := getRepositoryRoot(path, executor)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get the source folder name
	sourceFolder := filesystem.Base(rootPath)

	// Construct worktree base path: ~/worktrees/<repo-name>
	homeDir, err := filesystem.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	worktreeBase := filesystem.Join(homeDir, "worktrees", sourceFolder)

	return &Repository{
		RootPath:     rootPath,
		WorktreeBase: worktreeBase,
		SourceFolder: sourceFolder,
		Config:       NewConfig(rootPath),
		executor:     executor,
		filesystem:   filesystem,
	}, nil
}

// IsGitRepository checks if the given path is within a git repository
func IsGitRepository(path string) bool {
	executor := NewGitExecutor()
	return isGitRepository(path, executor)
}

// isGitRepository checks if the given path is within a git repository using provided executor
func isGitRepository(path string, executor GitExecutor) bool {
	_, err := executor.ExecuteInDir(path, "rev-parse", "--git-dir")
	return err == nil
}

// GetRepositoryRoot returns the absolute path to the git repository root
func GetRepositoryRoot(path string) (string, error) {
	executor := NewGitExecutor()
	return getRepositoryRoot(path, executor)
}

// getRepositoryRoot returns the absolute path to the git repository root using provided executor
func getRepositoryRoot(path string, executor GitExecutor) (string, error) {
	output, err := executor.ExecuteInDir(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}
	return output, nil
}

// GetDefaultBranch returns the default branch name (main, master, etc.)
func (r *Repository) GetDefaultBranch() (string, error) {
	// Try to get from remote HEAD
	if output, err := r.executor.ExecuteInDir(r.RootPath, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		// Output format: refs/remotes/origin/main
		parts := strings.Split(output, "/")
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
	_, err := r.executor.ExecuteInDir(r.RootPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	return err == nil
}

// remoteBranchExists checks if a remote branch exists
func (r *Repository) remoteBranchExists(refName string) bool {
	_, err := r.executor.ExecuteInDir(r.RootPath, "show-ref", "--verify", "--quiet", "refs/remotes/"+refName)
	return err == nil
}

// GetCurrentBranch returns the current branch name, or empty string if in detached HEAD
func (r *Repository) GetCurrentBranch() (string, error) {
	output, err := r.executor.ExecuteInDir(r.RootPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	// If in detached HEAD state, git returns "HEAD"
	if output == "HEAD" {
		return "", nil
	}
	return output, nil
}

// CreateBranch creates a new branch from the specified base branch
func (r *Repository) CreateBranch(branchName, baseBranch string) error {
	if _, err := r.executor.ExecuteInDir(r.RootPath, "branch", branchName, baseBranch); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}
	return nil
}

// DeleteBranch deletes a branch (force delete)
func (r *Repository) DeleteBranch(branchName string) error {
	if _, err := r.executor.ExecuteInDir(r.RootPath, "branch", "-D", branchName); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
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

// FilterOutMainBranch removes the main/root repository from a list of worktrees
// The main repository is identified by its path being equal to the repository root path
func (r *Repository) FilterOutMainBranch(worktrees []*Worktree) []*Worktree {
	var filtered []*Worktree
	for _, wt := range worktrees {
		// Skip the main worktree (the repository root)
		if wt.Path == r.RootPath {
			continue
		}
		filtered = append(filtered, wt)
	}
	return filtered
}

// ListWorktreesWithMergeStatusExcludingMain returns all worktrees enriched with merge status,
// excluding the main/root repository
func (r *Repository) ListWorktreesWithMergeStatusExcludingMain() ([]*Worktree, error) {
	worktrees, err := r.ListWorktreesWithMergeStatus()
	if err != nil {
		return nil, err
	}
	return r.FilterOutMainBranch(worktrees), nil
}

// GetCleanupCandidates returns worktrees that should be cleaned up
// Returns merged worktrees first, then stale worktrees
func (r *Repository) GetCleanupCandidates() ([]*Worktree, error) {
	worktrees, err := r.ListWorktreesWithMergeStatus()
	if err != nil {
		return nil, err
	}

	// Filter out main branch
	worktrees = r.FilterOutMainBranch(worktrees)

	var merged []*Worktree
	var stale []*Worktree

	for _, wt := range worktrees {
		if wt.IsMerged() {
			merged = append(merged, wt)
		} else if wt.IsStale() {
			stale = append(stale, wt)
		}
	}

	// Return merged first, then stale
	return append(merged, stale...), nil
}

// StartupCleanupCandidates represents cleanup results categorized by type
type StartupCleanupCandidates struct {
	Orphaned []*Worktree
	Merged   []*Worktree
}

// GetStartupCleanupCandidates returns worktrees that need cleanup at startup
// Orphaned worktrees are automatically cleaned, merged ones are interactive
func (r *Repository) GetStartupCleanupCandidates() (*StartupCleanupCandidates, error) {
	worktrees, err := r.ListWorktreesWithMergeStatus()
	if err != nil {
		return nil, err
	}

	// Filter out main branch
	worktrees = r.FilterOutMainBranch(worktrees)

	candidates := &StartupCleanupCandidates{
		Orphaned: []*Worktree{},
		Merged:   []*Worktree{},
	}

	for _, wt := range worktrees {
		if wt.IsOrphaned() {
			candidates.Orphaned = append(candidates.Orphaned, wt)
		} else if wt.IsMerged() {
			candidates.Merged = append(candidates.Merged, wt)
		}
	}

	return candidates, nil
}
