package gitlab

import (
	"testing"
)

func TestIssueSanitizedTitle(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{
			title:    "Fix authentication bug",
			expected: "fix-authentication-bug",
		},
		{
			title:    "Add dark mode to settings page",
			expected: "add-dark-mode-to-settings-page",
		},
		{
			title:    "VERY LONG TITLE THAT EXCEEDS FORTY CHARACTERS IN LENGTH",
			expected: "very-long-title-that-exceeds-forty-chara",
		},
		{
			title:    "Issue with Special!@#$%Characters",
			expected: "issue-with-special-characters",
		},
	}

	for _, tt := range tests {
		issue := &Issue{
			IID:   123,
			Title: tt.title,
		}
		result := issue.SanitizedTitle()
		if result != tt.expected {
			t.Errorf("SanitizedTitle(%q): expected %q, got %q", tt.title, tt.expected, result)
		}
	}
}

func TestIssueBranchName(t *testing.T) {
	issue := &Issue{
		IID:   123,
		Title: "Fix authentication bug",
	}

	expected := "work/123-fix-authentication-bug"
	result := issue.BranchName()
	if result != expected {
		t.Errorf("BranchName(): expected %q, got %q", expected, result)
	}
}

func TestIssueFormatForDisplay(t *testing.T) {
	tests := []struct {
		issue    *Issue
		expected string
	}{
		{
			issue: &Issue{
				IID:    123,
				Title:  "Fix bug",
				Labels: []string{},
			},
			expected: "#123 | Fix bug",
		},
		{
			issue: &Issue{
				IID:    456,
				Title:  "Add feature",
				Labels: []string{"enhancement", "priority::high"},
			},
			expected: "#456 | Add feature | enhancement priority::high",
		},
	}

	for _, tt := range tests {
		result := tt.issue.FormatForDisplay()
		if result != tt.expected {
			t.Errorf("FormatForDisplay(): expected %q, got %q", tt.expected, result)
		}
	}
}

func TestListOpenIssues(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	issueListJSON := `[
  {
    "iid": 123,
    "title": "Fix authentication bug",
    "description": "Users can't log in",
    "state": "opened",
    "labels": ["bug"],
    "web_url": "https://gitlab.com/owner/project/-/issues/123",
    "author": {"username": "alice", "name": "Alice Smith"},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T14:30:00Z"
  }
]`

	// Match the exact command being executed
	fake.SetResponse("-R owner/project issue list --state opened --per-page 25 --json", issueListJSON)

	client := &Client{
		Owner:    "owner",
		Project:  "project",
		Host:     "gitlab.com",
		executor: fake,
	}

	issues, err := client.ListOpenIssues(25)
	if err != nil {
		t.Fatalf("ListOpenIssues failed: %v", err)
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
		return
	}

	if issues[0].IID != 123 {
		t.Errorf("expected IID 123, got %d", issues[0].IID)
	}
	if issues[0].Title != "Fix authentication bug" {
		t.Errorf("expected title 'Fix authentication bug', got %q", issues[0].Title)
	}
}

func TestGetIssue(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	issueJSON := `{
    "iid": 123,
    "title": "Fix bug",
    "description": "This is a bug",
    "state": "opened",
    "labels": ["bug"],
    "web_url": "https://gitlab.com/owner/project/-/issues/123",
    "author": {"username": "bob", "name": "Bob Jones"},
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-16T14:30:00Z"
  }`

	fake.SetResponse("-R owner/project issue view 123 --json", issueJSON)

	client := &Client{
		Owner:    "owner",
		Project:  "project",
		Host:     "gitlab.com",
		executor: fake,
	}

	issue, err := client.GetIssue(123)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}

	if issue.IID != 123 {
		t.Errorf("expected IID 123, got %d", issue.IID)
	}
	if issue.Title != "Fix bug" {
		t.Errorf("expected title 'Fix bug', got %q", issue.Title)
	}
}

func TestIsIssueClosed(t *testing.T) {
	fake := NewFakeGitLabExecutor()

	// Test open issue
	openIssueJSON := `{
    "iid": 123,
    "title": "Open issue",
    "state": "opened",
    "labels": [],
    "web_url": "https://gitlab.com/owner/project/-/issues/123",
    "author": {"username": "alice", "name": "Alice"}
  }`

	fake.SetResponse("-R owner/project issue view 123 --json", openIssueJSON)

	client := &Client{
		Owner:    "owner",
		Project:  "project",
		Host:     "gitlab.com",
		executor: fake,
	}

	closed, err := client.IsIssueClosed(123)
	if err != nil {
		t.Fatalf("IsIssueClosed failed: %v", err)
	}
	if closed {
		t.Error("expected open issue to return false for IsIssueClosed")
	}

	// Test closed issue
	fake.Reset()
	closedIssueJSON := `{
    "iid": 456,
    "title": "Closed issue",
    "state": "closed",
    "labels": [],
    "web_url": "https://gitlab.com/owner/project/-/issues/456",
    "author": {"username": "bob", "name": "Bob"}
  }`

	fake.SetResponse("-R owner/project issue view 456 --json", closedIssueJSON)

	closed, err = client.IsIssueClosed(456)
	if err != nil {
		t.Fatalf("IsIssueClosed failed: %v", err)
	}
	if !closed {
		t.Error("expected closed issue to return true for IsIssueClosed")
	}
}
