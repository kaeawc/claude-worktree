package git

import (
	"testing"
)

func TestRepairActionType_String(t *testing.T) {
	tests := []struct {
		actionType RepairActionType
		expected   string
	}{
		{RepairRemoveStaleLock, "Remove Stale Lock"},
		{RepairPruneOrphan, "Prune Orphaned Worktree"},
		{RepairWorktreeLink, "Repair Worktree Link"},
		{RepairRebuildIndex, "Rebuild Git Index"},
		{RepairActionType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.actionType.String(); got != tt.expected {
				t.Errorf("RepairActionType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSafeRepairActions(t *testing.T) {
	actions := []RepairAction{
		{Type: RepairRemoveStaleLock, Safe: true},
		{Type: RepairRebuildIndex, Safe: false},
		{Type: RepairPruneOrphan, Safe: true},
		{Type: RepairWorktreeLink, Safe: true},
	}

	safe := GetSafeRepairActions(actions)

	if len(safe) != 3 {
		t.Errorf("GetSafeRepairActions() returned %d actions, want 3", len(safe))
	}

	for _, action := range safe {
		if !action.Safe {
			t.Errorf("GetSafeRepairActions() returned unsafe action: %v", action)
		}
	}
}

func TestGetUnsafeRepairActions(t *testing.T) {
	actions := []RepairAction{
		{Type: RepairRemoveStaleLock, Safe: true},
		{Type: RepairRebuildIndex, Safe: false},
		{Type: RepairPruneOrphan, Safe: true},
	}

	unsafe := GetUnsafeRepairActions(actions)

	if len(unsafe) != 1 {
		t.Errorf("GetUnsafeRepairActions() returned %d actions, want 1", len(unsafe))
	}

	for _, action := range unsafe {
		if action.Safe {
			t.Errorf("GetUnsafeRepairActions() returned safe action: %v", action)
		}
	}
}

func TestGetRepairActions_LockFiles(t *testing.T) {
	executor := NewFakeGitExecutor()
	fs := NewFakeFileSystem()
	repo, err := NewRepositoryFromPathWithDeps("/fake/repo", executor, fs)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	results := []*HealthCheckResult{
		{
			WorktreePath: "/fake/wt",
			Issues: []HealthCheckIssue{
				{
					Category:    "Lock Files",
					Description: "Stale lock file found: /fake/wt/.git/index.lock (age: 1h)",
					Repairable:  true,
				},
			},
		},
	}

	actions := repo.GetRepairActions(results)

	if len(actions) != 1 {
		t.Errorf("GetRepairActions() returned %d actions, want 1", len(actions))
	}

	if actions[0].Type != RepairRemoveStaleLock {
		t.Errorf("GetRepairActions() returned action type %v, want %v", actions[0].Type, RepairRemoveStaleLock)
	}
}

func TestGetRepairActions_OrphanedWorktree(t *testing.T) {
	executor := NewFakeGitExecutor()
	fs := NewFakeFileSystem()
	repo, err := NewRepositoryFromPathWithDeps("/fake/repo", executor, fs)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	results := []*HealthCheckResult{
		{
			WorktreePath: "/fake/wt",
			Issues: []HealthCheckIssue{
				{
					Category:    "Orphaned Worktrees",
					Description: "Orphaned worktree metadata found: /fake/wt (directory missing)",
					Repairable:  true,
					RepairHint:  "Can be pruned with 'git worktree prune'",
				},
			},
		},
	}

	actions := repo.GetRepairActions(results)

	if len(actions) != 1 {
		t.Errorf("GetRepairActions() returned %d actions, want 1", len(actions))
	}

	if actions[0].Type != RepairPruneOrphan {
		t.Errorf("GetRepairActions() returned action type %v, want %v", actions[0].Type, RepairPruneOrphan)
	}
}

func TestGetRepairActions_GitMetadata(t *testing.T) {
	executor := NewFakeGitExecutor()
	fs := NewFakeFileSystem()
	repo, err := NewRepositoryFromPathWithDeps("/fake/repo", executor, fs)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	results := []*HealthCheckResult{
		{
			WorktreePath: "/fake/wt",
			Issues: []HealthCheckIssue{
				{
					Category:    "Git Metadata",
					Description: ".git file missing",
					Repairable:  true,
					RepairHint:  "Can be repaired with 'git worktree repair'",
				},
			},
		},
	}

	actions := repo.GetRepairActions(results)

	if len(actions) != 1 {
		t.Errorf("GetRepairActions() returned %d actions, want 1", len(actions))
	}

	if actions[0].Type != RepairWorktreeLink {
		t.Errorf("GetRepairActions() returned action type %v, want %v", actions[0].Type, RepairWorktreeLink)
	}
}

func TestGetRepairActions_IndexCorruption(t *testing.T) {
	executor := NewFakeGitExecutor()
	fs := NewFakeFileSystem()
	repo, err := NewRepositoryFromPathWithDeps("/fake/repo", executor, fs)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	results := []*HealthCheckResult{
		{
			WorktreePath: "/fake/wt",
			Issues: []HealthCheckIssue{
				{
					Category:    "Git Operations",
					Description: "Git status failed",
					Repairable:  true,
					RepairHint:  "May indicate index corruption; can attempt index rebuild",
				},
			},
		},
	}

	actions := repo.GetRepairActions(results)

	if len(actions) != 1 {
		t.Errorf("GetRepairActions() returned %d actions, want 1", len(actions))
	}

	if actions[0].Type != RepairRebuildIndex {
		t.Errorf("GetRepairActions() returned action type %v, want %v", actions[0].Type, RepairRebuildIndex)
	}

	if actions[0].Safe {
		t.Error("Index rebuild should not be marked as safe")
	}
}

func TestGetRepairActions_NonRepairableIgnored(t *testing.T) {
	executor := NewFakeGitExecutor()
	fs := NewFakeFileSystem()
	repo, err := NewRepositoryFromPathWithDeps("/fake/repo", executor, fs)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	results := []*HealthCheckResult{
		{
			WorktreePath: "/fake/wt",
			Issues: []HealthCheckIssue{
				{
					Category:    "Something",
					Description: "Some issue",
					Repairable:  false,
				},
				{
					Category:    "Lock Files",
					Description: "Stale lock file found: /fake/lock (age: 1h)",
					Repairable:  true,
				},
			},
		},
	}

	actions := repo.GetRepairActions(results)

	if len(actions) != 1 {
		t.Errorf("GetRepairActions() returned %d actions, want 1 (non-repairable should be ignored)", len(actions))
	}

	if actions[0].Type != RepairRemoveStaleLock {
		t.Errorf("GetRepairActions() returned action type %v, want %v", actions[0].Type, RepairRemoveStaleLock)
	}
}
