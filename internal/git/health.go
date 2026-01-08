package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HealthCheckSeverity indicates the severity level of a health check issue
type HealthCheckSeverity int

const (
	SeverityOK HealthCheckSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s HealthCheckSeverity) String() string {
	switch s {
	case SeverityOK:
		return "OK"
	case SeverityWarning:
		return "Warning"
	case SeverityError:
		return "Error"
	case SeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// HealthCheckIssue represents a single health check finding
type HealthCheckIssue struct {
	Severity    HealthCheckSeverity
	Category    string
	Description string
	Repairable  bool
	RepairHint  string
}

// HealthCheckResult contains the results of a health check
type HealthCheckResult struct {
	WorktreePath string
	CheckTime    time.Time
	Issues       []HealthCheckIssue
	Healthy      bool
}

// GetMaxSeverity returns the highest severity found in the issues
func (r *HealthCheckResult) GetMaxSeverity() HealthCheckSeverity {
	maxSeverity := SeverityOK
	for _, issue := range r.Issues {
		if issue.Severity > maxSeverity {
			maxSeverity = issue.Severity
		}
	}
	return maxSeverity
}

// GetRepairableIssues returns only issues that can be automatically repaired
func (r *HealthCheckResult) GetRepairableIssues() []HealthCheckIssue {
	var repairable []HealthCheckIssue
	for _, issue := range r.Issues {
		if issue.Repairable {
			repairable = append(repairable, issue)
		}
	}
	return repairable
}

// PerformHealthCheck runs a comprehensive health check on a specific worktree
func (r *Repository) PerformHealthCheck(worktreePath string) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		WorktreePath: worktreePath,
		CheckTime:    time.Now(),
		Issues:       []HealthCheckIssue{},
		Healthy:      true,
	}

	// Check if this is the main worktree
	isMainWorktree := worktreePath == r.RootPath

	// 1. Check if directory exists and is accessible
	if err := r.checkDirectoryExists(worktreePath, result); err != nil {
		// If directory doesn't exist, this is critical and we can't continue
		result.Healthy = false
		return result, nil
	}

	// 2. Check .git file/directory integrity
	if !isMainWorktree {
		r.checkGitFileIntegrity(worktreePath, result)
	}

	// 3. Check for stale lock files
	r.checkStaleLockFiles(worktreePath, result)

	// 4. Check branch refs are accessible
	r.checkBranchRefs(worktreePath, result)

	// 5. Verify git commands execute successfully
	r.checkGitCommandExecution(worktreePath, result)

	// 6. Check for orphaned worktrees (only for main repo)
	if isMainWorktree {
		r.checkOrphanedWorktrees(result)
	}

	// Determine overall health
	result.Healthy = result.GetMaxSeverity() < SeverityError

	return result, nil
}

// PerformHealthCheckAll runs health checks on all worktrees
func (r *Repository) PerformHealthCheckAll() ([]*HealthCheckResult, error) {
	var results []*HealthCheckResult

	// First check the main repository
	mainResult, err := r.PerformHealthCheck(r.RootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check main repository: %w", err)
	}
	results = append(results, mainResult)

	// Then check all worktrees
	worktrees, err := r.ListWorktrees()
	if err != nil {
		return results, fmt.Errorf("failed to list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		// Skip main worktree as we already checked it
		if wt.Path == r.RootPath {
			continue
		}

		wtResult, err := r.PerformHealthCheck(wt.Path)
		if err != nil {
			// Don't fail completely, just record the error
			results = append(results, &HealthCheckResult{
				WorktreePath: wt.Path,
				CheckTime:    time.Now(),
				Issues: []HealthCheckIssue{
					{
						Severity:    SeverityCritical,
						Category:    "Health Check",
						Description: fmt.Sprintf("Failed to perform health check: %v", err),
						Repairable:  false,
					},
				},
				Healthy: false,
			})
			continue
		}
		results = append(results, wtResult)
	}

	return results, nil
}

// checkDirectoryExists verifies the worktree directory exists and is accessible
func (r *Repository) checkDirectoryExists(path string, result *HealthCheckResult) error {
	info, err := r.filesystem.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Issues = append(result.Issues, HealthCheckIssue{
				Severity:    SeverityCritical,
				Category:    "Directory",
				Description: fmt.Sprintf("Worktree directory does not exist: %s", path),
				Repairable:  true,
				RepairHint:  "Can be pruned from worktree list",
			})
			return err
		}
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Directory",
			Description: fmt.Sprintf("Cannot access worktree directory: %v", err),
			Repairable:  false,
		})
		return err
	}

	if !info.IsDir() {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityCritical,
			Category:    "Directory",
			Description: fmt.Sprintf("Worktree path is not a directory: %s", path),
			Repairable:  false,
		})
		return fmt.Errorf("not a directory")
	}

	// Check if directory is readable and writable
	testFile := filepath.Join(path, ".git-health-check-test")
	if err := r.filesystem.WriteFile(testFile, []byte("test"), 0644); err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Permissions",
			Description: fmt.Sprintf("Directory is not writable: %v", err),
			Repairable:  false,
		})
	} else {
		r.filesystem.Remove(testFile)
	}

	return nil
}

// checkGitFileIntegrity verifies the .git file in linked worktrees
func (r *Repository) checkGitFileIntegrity(path string, result *HealthCheckResult) {
	gitPath := filepath.Join(path, ".git")

	info, err := r.filesystem.Stat(gitPath)
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityCritical,
			Category:    "Git Metadata",
			Description: fmt.Sprintf(".git file/directory missing: %v", err),
			Repairable:  true,
			RepairHint:  "Can be repaired with 'git worktree repair'",
		})
		return
	}

	// For linked worktrees, .git should be a file, not a directory
	if info.IsDir() {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityWarning,
			Category:    "Git Metadata",
			Description: ".git is a directory (expected a file for linked worktree)",
			Repairable:  false,
		})
		return
	}

	// Read the .git file and verify it points to a valid location
	content, err := r.filesystem.ReadFile(gitPath)
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Git Metadata",
			Description: fmt.Sprintf("Cannot read .git file: %v", err),
			Repairable:  true,
			RepairHint:  "Can be repaired with 'git worktree repair'",
		})
		return
	}

	// .git file should contain "gitdir: /path/to/repo/.git/worktrees/<name>"
	gitdirLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(gitdirLine, "gitdir: ") {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Git Metadata",
			Description: fmt.Sprintf("Invalid .git file format: %s", gitdirLine),
			Repairable:  true,
			RepairHint:  "Can be repaired with 'git worktree repair'",
		})
		return
	}

	// Verify the gitdir path exists
	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")
	if _, err := r.filesystem.Stat(gitdir); err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Git Metadata",
			Description: fmt.Sprintf("Git directory referenced by .git file does not exist: %s", gitdir),
			Repairable:  true,
			RepairHint:  "Can be repaired with 'git worktree repair'",
		})
	}
}

// checkStaleLockFiles looks for stale git lock files
func (r *Repository) checkStaleLockFiles(path string, result *HealthCheckResult) {
	gitDir := filepath.Join(path, ".git")

	// For linked worktrees, read the actual git directory location
	if path != r.RootPath {
		content, err := r.filesystem.ReadFile(gitDir)
		if err == nil {
			gitdirLine := strings.TrimSpace(string(content))
			if strings.HasPrefix(gitdirLine, "gitdir: ") {
				gitDir = strings.TrimPrefix(gitdirLine, "gitdir: ")
			}
		}
	}

	lockFiles, err := DetectLockFiles(gitDir)
	if err != nil {
		// Don't fail the health check if we can't detect lock files
		// This might happen if git directory is inaccessible
		return
	}

	staleLocks := GetStaleLockFiles(lockFiles)

	for _, lock := range staleLocks {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityWarning,
			Category:    "Lock Files",
			Description: fmt.Sprintf("Stale lock file found: %s (age: %s)", lock.Path, lock.Age),
			Repairable:  true,
			RepairHint:  "Can be safely removed",
		})
	}

	// Report active lock files as informational
	activeLocks := make([]LockFile, 0)
	for _, lock := range lockFiles {
		isStale := false
		for _, stale := range staleLocks {
			if stale.Path == lock.Path {
				isStale = true
				break
			}
		}
		if !isStale {
			activeLocks = append(activeLocks, lock)
		}
	}

	if len(activeLocks) > 0 {
		for _, lock := range activeLocks {
			result.Issues = append(result.Issues, HealthCheckIssue{
				Severity:    SeverityOK,
				Category:    "Lock Files",
				Description: fmt.Sprintf("Active lock file (PID %d): %s", lock.ProcessID, lock.Path),
				Repairable:  false,
			})
		}
	}
}

// checkBranchRefs verifies branch references are accessible
func (r *Repository) checkBranchRefs(path string, result *HealthCheckResult) {
	// Get the current branch
	branch, err := r.executor.ExecuteInDir(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Branch Refs",
			Description: fmt.Sprintf("Cannot determine current branch: %v", err),
			Repairable:  false,
		})
		return
	}

	branch = strings.TrimSpace(branch)

	// Check if we're in detached HEAD state
	if branch == "HEAD" {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityOK,
			Category:    "Branch Refs",
			Description: "Worktree is in detached HEAD state",
			Repairable:  false,
		})
		return
	}

	// Verify the branch ref exists
	_, err = r.executor.ExecuteInDir(path, "rev-parse", "--verify", branch)
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Branch Refs",
			Description: fmt.Sprintf("Branch reference is invalid: %s", branch),
			Repairable:  false,
		})
		return
	}

	// Check if the branch has an upstream
	_, err = r.executor.ExecuteInDir(path, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil {
		// Not having an upstream is just informational, not an error
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityOK,
			Category:    "Branch Refs",
			Description: fmt.Sprintf("Branch '%s' has no upstream configured", branch),
			Repairable:  false,
		})
	}
}

// checkGitCommandExecution verifies basic git commands work
func (r *Repository) checkGitCommandExecution(path string, result *HealthCheckResult) {
	// Try a simple git status command
	_, err := r.executor.ExecuteInDir(path, "status", "--porcelain")
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Git Operations",
			Description: fmt.Sprintf("Git status command failed: %v", err),
			Repairable:  true,
			RepairHint:  "May indicate index corruption; can attempt index rebuild",
		})
		return
	}

	// Try to verify the repository
	_, err = r.executor.ExecuteInDir(path, "rev-parse", "--git-dir")
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityError,
			Category:    "Git Operations",
			Description: fmt.Sprintf("Cannot determine git directory: %v", err),
			Repairable:  false,
		})
	}
}

// checkOrphanedWorktrees looks for worktree metadata without corresponding directories
func (r *Repository) checkOrphanedWorktrees(result *HealthCheckResult) {
	worktrees, err := r.ListWorktrees()
	if err != nil {
		result.Issues = append(result.Issues, HealthCheckIssue{
			Severity:    SeverityWarning,
			Category:    "Orphaned Worktrees",
			Description: fmt.Sprintf("Failed to list worktrees: %v", err),
			Repairable:  false,
		})
		return
	}

	orphanCount := 0
	for _, wt := range worktrees {
		if wt.Path == r.RootPath {
			continue // Skip main worktree
		}

		if _, err := r.filesystem.Stat(wt.Path); os.IsNotExist(err) {
			orphanCount++
			result.Issues = append(result.Issues, HealthCheckIssue{
				Severity:    SeverityWarning,
				Category:    "Orphaned Worktrees",
				Description: fmt.Sprintf("Orphaned worktree metadata found: %s (directory missing)", wt.Path),
				Repairable:  true,
				RepairHint:  "Can be pruned with 'git worktree prune'",
			})
		}
	}
}
