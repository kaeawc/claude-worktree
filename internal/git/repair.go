package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepairActionType identifies the type of repair action
type RepairActionType int

const (
	RepairRemoveStaleLock RepairActionType = iota
	RepairPruneOrphan
	RepairWorktreeLink
	RepairRebuildIndex
)

func (t RepairActionType) String() string {
	switch t {
	case RepairRemoveStaleLock:
		return "Remove Stale Lock"
	case RepairPruneOrphan:
		return "Prune Orphaned Worktree"
	case RepairWorktreeLink:
		return "Repair Worktree Link"
	case RepairRebuildIndex:
		return "Rebuild Git Index"
	default:
		return "Unknown"
	}
}

// RepairAction represents a single repair operation
type RepairAction struct {
	Type        RepairActionType
	WorktreePath string
	Description string
	Target      string // The specific file/path being repaired
	Safe        bool   // Whether this is a safe operation that doesn't need confirmation
}

// RepairResult contains the result of a repair operation
type RepairResult struct {
	Action  RepairAction
	Success bool
	Error   error
	Message string
}

// GetRepairActions analyzes health check results and returns recommended repair actions
func (r *Repository) GetRepairActions(results []*HealthCheckResult) []RepairAction {
	var actions []RepairAction

	for _, result := range results {
		for _, issue := range result.Issues {
			if !issue.Repairable {
				continue
			}

			// Determine the repair action based on the issue
			switch issue.Category {
			case "Lock Files":
				if strings.Contains(issue.Description, "Stale lock file") {
					// Extract lock file path from description
					parts := strings.Split(issue.Description, ": ")
					if len(parts) >= 2 {
						lockPath := strings.Split(parts[1], " (age:")[0]
						actions = append(actions, RepairAction{
							Type:         RepairRemoveStaleLock,
							WorktreePath: result.WorktreePath,
							Description:  fmt.Sprintf("Remove stale lock file: %s", lockPath),
							Target:       lockPath,
							Safe:         true,
						})
					}
				}

			case "Orphaned Worktrees":
				if strings.Contains(issue.Description, "Orphaned worktree metadata") {
					actions = append(actions, RepairAction{
						Type:         RepairPruneOrphan,
						WorktreePath: result.WorktreePath,
						Description:  "Prune orphaned worktree metadata",
						Target:       result.WorktreePath,
						Safe:         true,
					})
				}

			case "Git Metadata":
				if strings.Contains(issue.RepairHint, "git worktree repair") {
					actions = append(actions, RepairAction{
						Type:         RepairWorktreeLink,
						WorktreePath: result.WorktreePath,
						Description:  "Repair worktree link",
						Target:       result.WorktreePath,
						Safe:         true,
					})
				}

			case "Git Operations":
				if strings.Contains(issue.RepairHint, "index rebuild") {
					actions = append(actions, RepairAction{
						Type:         RepairRebuildIndex,
						WorktreePath: result.WorktreePath,
						Description:  "Rebuild corrupted git index",
						Target:       filepath.Join(result.WorktreePath, ".git", "index"),
						Safe:         false, // Index rebuild requires confirmation
					})
				}

			case "Directory":
				if strings.Contains(issue.RepairHint, "pruned") {
					actions = append(actions, RepairAction{
						Type:         RepairPruneOrphan,
						WorktreePath: result.WorktreePath,
						Description:  "Prune worktree with missing directory",
						Target:       result.WorktreePath,
						Safe:         true,
					})
				}
			}
		}
	}

	return actions
}

// PerformRepair executes a single repair action
func (r *Repository) PerformRepair(action RepairAction) RepairResult {
	result := RepairResult{
		Action:  action,
		Success: false,
	}

	switch action.Type {
	case RepairRemoveStaleLock:
		result.Error = r.performRemoveStaleLock(action)
		if result.Error == nil {
			result.Success = true
			result.Message = fmt.Sprintf("Successfully removed stale lock file: %s", action.Target)
		} else {
			result.Message = fmt.Sprintf("Failed to remove lock file: %v", result.Error)
		}

	case RepairPruneOrphan:
		result.Error = r.performPruneOrphan(action)
		if result.Error == nil {
			result.Success = true
			result.Message = "Successfully pruned orphaned worktree metadata"
		} else {
			result.Message = fmt.Sprintf("Failed to prune orphaned worktree: %v", result.Error)
		}

	case RepairWorktreeLink:
		result.Error = r.performRepairWorktreeLink(action)
		if result.Error == nil {
			result.Success = true
			result.Message = "Successfully repaired worktree link"
		} else {
			result.Message = fmt.Sprintf("Failed to repair worktree link: %v", result.Error)
		}

	case RepairRebuildIndex:
		result.Error = r.performRebuildIndex(action)
		if result.Error == nil {
			result.Success = true
			result.Message = "Successfully rebuilt git index"
		} else {
			result.Message = fmt.Sprintf("Failed to rebuild index: %v", result.Error)
		}

	default:
		result.Error = fmt.Errorf("unknown repair action type: %v", action.Type)
		result.Message = result.Error.Error()
	}

	return result
}

// PerformRepairs executes multiple repair actions
func (r *Repository) PerformRepairs(actions []RepairAction) ([]RepairResult, error) {
	var results []RepairResult

	for _, action := range actions {
		result := r.PerformRepair(action)
		results = append(results, result)

		// Stop on critical failures
		if !result.Success && !action.Safe {
			return results, fmt.Errorf("repair failed: %v", result.Error)
		}
	}

	return results, nil
}

// performRemoveStaleLock removes a stale lock file
func (r *Repository) performRemoveStaleLock(action RepairAction) error {
	lockPath := action.Target

	// Verify the lock file still exists
	if _, err := r.filesystem.Stat(lockPath); os.IsNotExist(err) {
		// Already removed, not an error
		return nil
	}

	// Re-verify it's still stale before removing
	lockFiles, err := DetectLockFiles(filepath.Dir(lockPath))
	if err != nil {
		return fmt.Errorf("failed to detect lock files: %w", err)
	}

	staleLocks := GetStaleLockFiles(lockFiles)
	isStale := false
	for _, lock := range staleLocks {
		if lock.Path == lockPath {
			isStale = true
			break
		}
	}

	if !isStale {
		return fmt.Errorf("lock file is no longer stale, refusing to remove: %s", lockPath)
	}

	// Find the actual lock file struct
	var lockFileToRemove *LockFile
	for _, lock := range staleLocks {
		if lock.Path == lockPath {
			lockFileToRemove = &lock
			break
		}
	}

	if lockFileToRemove == nil {
		return fmt.Errorf("lock file not found in stale list: %s", lockPath)
	}

	// Remove the stale lock file
	if err := RemoveLockFile(*lockFileToRemove); err != nil {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

// performPruneOrphan prunes orphaned worktree metadata
func (r *Repository) performPruneOrphan(action RepairAction) error {
	// Run git worktree prune
	_, err := r.executor.Execute("worktree", "prune", "-v")
	if err != nil {
		return fmt.Errorf("git worktree prune failed: %w", err)
	}

	return nil
}

// performRepairWorktreeLink repairs worktree links
func (r *Repository) performRepairWorktreeLink(action RepairAction) error {
	// Run git worktree repair
	_, err := r.executor.Execute("worktree", "repair")
	if err != nil {
		return fmt.Errorf("git worktree repair failed: %w", err)
	}

	return nil
}

// performRebuildIndex rebuilds a corrupted git index
func (r *Repository) performRebuildIndex(action RepairAction) error {
	worktreePath := action.WorktreePath

	// First, try to backup the existing index
	indexPath := filepath.Join(worktreePath, ".git", "index")
	backupPath := indexPath + ".backup"

	// For linked worktrees, the index path is in the worktrees directory
	if worktreePath != r.RootPath {
		gitFilePath := filepath.Join(worktreePath, ".git")
		content, err := r.filesystem.ReadFile(gitFilePath)
		if err == nil {
			gitdirLine := strings.TrimSpace(string(content))
			if strings.HasPrefix(gitdirLine, "gitdir: ") {
				gitDir := strings.TrimPrefix(gitdirLine, "gitdir: ")
				indexPath = filepath.Join(gitDir, "index")
				backupPath = indexPath + ".backup"
			}
		}
	}

	// Backup the corrupted index by reading and writing
	if indexContent, err := r.filesystem.ReadFile(indexPath); err == nil {
		if err := r.filesystem.WriteFile(backupPath, indexContent, 0644); err != nil {
			return fmt.Errorf("failed to backup index: %w", err)
		}
	}

	// Remove the corrupted index
	if err := r.filesystem.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove corrupted index: %w", err)
	}

	// Reset the index from HEAD
	_, err := r.executor.ExecuteInDir(worktreePath, "read-tree", "HEAD")
	if err != nil {
		// Try to restore the backup
		if backupContent, readErr := r.filesystem.ReadFile(backupPath); readErr == nil {
			r.filesystem.WriteFile(indexPath, backupContent, 0644)
		}
		return fmt.Errorf("failed to rebuild index: %w", err)
	}

	// If successful, remove the backup
	r.filesystem.Remove(backupPath)

	return nil
}

// GetSafeRepairActions returns only actions that are safe to run without confirmation
func GetSafeRepairActions(actions []RepairAction) []RepairAction {
	var safe []RepairAction
	for _, action := range actions {
		if action.Safe {
			safe = append(safe, action)
		}
	}
	return safe
}

// GetUnsafeRepairActions returns actions that require user confirmation
func GetUnsafeRepairActions(actions []RepairAction) []RepairAction {
	var unsafe []RepairAction
	for _, action := range actions {
		if !action.Safe {
			unsafe = append(unsafe, action)
		}
	}
	return unsafe
}
