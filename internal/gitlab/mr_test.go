package gitlab

import (
	"testing"
)

func TestMRSanitizedTitle(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{
			title:    "Update dependencies",
			expected: "update-dependencies",
		},
		{
			title:    "Refactor API client with special chars!",
			expected: "refactor-api-client-with-special-chars",
		},
		{
			title:    "Very Long Title That Goes Beyond The Forty Character Limit",
			expected: "very-long-title-that-goes-beyond-the-for",
		},
	}

	for _, tt := range tests {
		mr := &MergeRequest{
			IID:   456,
			Title: tt.title,
		}
		result := mr.SanitizedTitle()
		if result != tt.expected {
			t.Errorf("SanitizedTitle(%q): expected %q, got %q", tt.title, tt.expected, result)
		}
	}
}

func TestMRBranchName(t *testing.T) {
	mr := &MergeRequest{
		IID:   456,
		Title: "Update dependencies",
	}

	expected := "mr/456-update-dependencies"
	result := mr.BranchName()
	if result != expected {
		t.Errorf("BranchName(): expected %q, got %q", expected, result)
	}
}

func TestMRFormatForDisplay(t *testing.T) {
	tests := []struct {
		mr       *MergeRequest
		expected string
	}{
		{
			mr: &MergeRequest{
				IID:   789,
				Title: "Update docs",
				Author: Author{
					Username: "alice",
					Name:     "Alice Smith",
				},
				Labels: []string{},
			},
			expected: "!789 | Update docs | @alice",
		},
		{
			mr: &MergeRequest{
				IID:   101,
				Title: "Add feature",
				Author: Author{
					Username: "bob",
					Name:     "Bob Jones",
				},
				Labels: []string{"enhancement", "priority::high"},
			},
			expected: "!101 | Add feature | @bob | enhancement priority::high",
		},
	}

	for _, tt := range tests {
		result := tt.mr.FormatForDisplay()
		if result != tt.expected {
			t.Errorf("FormatForDisplay(): expected %q, got %q", tt.expected, result)
		}
	}
}

func TestListOpenMRs(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	mrListJSON := `[
  {
    "iid": 456,
    "title": "Update dependencies",
    "description": "Bump all packages to latest versions",
    "state": "opened",
    "merge_status": "can_be_merged",
    "author": {"username": "alice", "name": "Alice Smith"},
    "source_branch": "update-deps",
    "target_branch": "main",
    "labels": ["maintenance"],
    "web_url": "https://gitlab.com/owner/project/-/merge_requests/456",
    "created_at": "2024-01-17T09:00:00Z",
    "updated_at": "2024-01-17T15:45:00Z",
    "work_in_progress": false,
    "changes_count": "42",
    "user_notes_count": 3
  }
]`

	fake.SetResponse("-R owner/project mr list --state opened --per-page 25 --json", mrListJSON)

	client := &Client{
		Owner:    "owner",
		Project:  "project",
		Host:     "gitlab.com",
		executor: fake,
	}

	mrs, err := client.ListOpenMRs(25)
	if err != nil {
		t.Fatalf("ListOpenMRs failed: %v", err)
	}

	if len(mrs) != 1 {
		t.Errorf("expected 1 MR, got %d", len(mrs))
		return
	}

	if mrs[0].IID != 456 {
		t.Errorf("expected IID 456, got %d", mrs[0].IID)
	}
	if mrs[0].Title != "Update dependencies" {
		t.Errorf("expected title 'Update dependencies', got %q", mrs[0].Title)
	}
}

func TestGetMR(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	mrJSON := `{
    "iid": 456,
    "title": "Update deps",
    "description": "Update packages",
    "state": "opened",
    "merge_status": "can_be_merged",
    "author": {"username": "bob", "name": "Bob Jones"},
    "source_branch": "feature-branch",
    "target_branch": "main",
    "labels": ["feature"],
    "web_url": "https://gitlab.com/owner/project/-/merge_requests/456",
    "created_at": "2024-01-17T09:00:00Z",
    "updated_at": "2024-01-17T15:45:00Z",
    "work_in_progress": false,
    "changes_count": "10",
    "user_notes_count": 2
  }`

	fake.SetResponse("-R owner/project mr view 456 --json", mrJSON)

	client := &Client{
		Owner:    "owner",
		Project:  "project",
		Host:     "gitlab.com",
		executor: fake,
	}

	mr, err := client.GetMR(456)
	if err != nil {
		t.Fatalf("GetMR failed: %v", err)
	}

	if mr.IID != 456 {
		t.Errorf("expected IID 456, got %d", mr.IID)
	}
	if mr.Title != "Update deps" {
		t.Errorf("expected title 'Update deps', got %q", mr.Title)
	}
}

func TestIsMRMerged(t *testing.T) {
	fake := NewFakeGitLabExecutor()

	// Test open MR
	openMRJSON := `{
    "iid": 123,
    "title": "Open MR",
    "state": "opened",
    "merge_status": "can_be_merged",
    "author": {"username": "alice", "name": "Alice"},
    "source_branch": "feature",
    "target_branch": "main",
    "labels": [],
    "web_url": "https://gitlab.com/owner/project/-/merge_requests/123"
  }`

	fake.SetResponse("-R owner/project mr view 123 --json", openMRJSON)

	client := &Client{
		Owner:    "owner",
		Project:  "project",
		Host:     "gitlab.com",
		executor: fake,
	}

	merged, err := client.IsMRMerged(123)
	if err != nil {
		t.Fatalf("IsMRMerged failed: %v", err)
	}
	if merged {
		t.Error("expected open MR to return false for IsMRMerged")
	}

	// Test merged MR
	fake.Reset()
	mergedMRJSON := `{
    "iid": 456,
    "title": "Merged MR",
    "state": "merged",
    "merge_status": "can_be_merged",
    "author": {"username": "bob", "name": "Bob"},
    "source_branch": "feature",
    "target_branch": "main",
    "labels": [],
    "web_url": "https://gitlab.com/owner/project/-/merge_requests/456"
  }`

	fake.SetResponse("-R owner/project mr view 456 --json", mergedMRJSON)

	merged, err = client.IsMRMerged(456)
	if err != nil {
		t.Fatalf("IsMRMerged failed: %v", err)
	}
	if !merged {
		t.Error("expected merged MR to return true for IsMRMerged")
	}
}

func TestHasMergeConflicts(t *testing.T) {
	fake := NewFakeGitLabExecutor()

	tests := []struct {
		mergeStatus    string
		hasConflicts   bool
		description    string
	}{
		{"can_be_merged", false, "can be merged"},
		{"can_be_merged_automerge", false, "can be merged with automerge"},
		{"cannot_be_merged", true, "cannot be merged"},
		{"cannot_be_merged_rebase", true, "cannot be merged - requires rebase"},
	}

	for _, tt := range tests {
		mrJSON := `{
    "iid": 789,
    "title": "Test MR",
    "state": "opened",
    "merge_status": "` + tt.mergeStatus + `",
    "author": {"username": "alice", "name": "Alice"},
    "source_branch": "feature",
    "target_branch": "main",
    "labels": [],
    "web_url": "https://gitlab.com/owner/project/-/merge_requests/789"
  }`

		fake.Reset()
		fake.SetResponse("-R owner/project mr view 789 --json", mrJSON)

		client := &Client{
			Owner:    "owner",
			Project:  "project",
			Host:     "gitlab.com",
			executor: fake,
		}

		hasConflicts, err := client.HasMergeConflicts(789)
		if err != nil {
			t.Fatalf("HasMergeConflicts failed for %s: %v", tt.description, err)
		}
		if hasConflicts != tt.hasConflicts {
			t.Errorf("HasMergeConflicts for %s: expected %v, got %v", tt.description, tt.hasConflicts, hasConflicts)
		}
	}
}
