package github

import (
	"testing"
)

func TestIssueSanitizedTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "Simple title",
			title: "Fix bug in authentication",
			want:  "fix-bug-in-authentication",
		},
		{
			name:  "Title with special characters",
			title: "Fix: Critical Bug in Auth!!",
			want:  "fix-critical-bug-in-auth",
		},
		{
			name:  "Long title (over 40 chars)",
			title: "This is a very long title that exceeds forty characters and should be truncated",
			want:  "this-is-a-very-long-title-that-exceeds-f",
		},
		{
			name:  "Title with numbers",
			title: "Add feature #123",
			want:  "add-feature-123",
		},
		{
			name:  "Title with multiple spaces",
			title: "Fix   multiple   spaces",
			want:  "fix-multiple-spaces",
		},
		{
			name:  "Title with leading/trailing spaces",
			title: "  trim spaces  ",
			want:  "trim-spaces",
		},
		{
			name:  "Title with underscores",
			title: "fix_underscore_naming",
			want:  "fix-underscore-naming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &Issue{Title: tt.title}
			got := issue.SanitizedTitle()

			if got != tt.want {
				t.Errorf("SanitizedTitle() = %v, want %v", got, tt.want)
			}

			// Verify it's not longer than 40 characters
			if len(got) > 40 {
				t.Errorf("SanitizedTitle() length = %d, want <= 40", len(got))
			}
		})
	}
}

func TestIssueBranchName(t *testing.T) {
	tests := []struct {
		name   string
		number int
		title  string
		want   string
	}{
		{
			name:   "Simple issue",
			number: 123,
			title:  "Fix login bug",
			want:   "work/123-fix-login-bug",
		},
		{
			name:   "Issue with special characters",
			number: 456,
			title:  "Add: New Feature!",
			want:   "work/456-add-new-feature",
		},
		{
			name:   "Long title",
			number: 789,
			title:  "This is a very long issue title that should be truncated properly",
			want:   "work/789-this-is-a-very-long-issue-title-that-sho",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &Issue{
				Number: tt.number,
				Title:  tt.title,
			}
			got := issue.BranchName()

			if got != tt.want {
				t.Errorf("BranchName() = %v, want %v", got, tt.want)
			}

			// Verify format is work/<number>-<sanitized>
			if len(got) < 6 || got[:5] != "work/" {
				t.Errorf("BranchName() should start with 'work/', got %v", got)
			}
		})
	}
}

func TestIssueFormatForDisplay(t *testing.T) {
	tests := []struct {
		name  string
		issue Issue
		want  string
	}{
		{
			name: "Issue without labels",
			issue: Issue{
				Number: 123,
				Title:  "Fix bug",
			},
			want: "#123 | Fix bug",
		},
		{
			name: "Issue with single label",
			issue: Issue{
				Number: 456,
				Title:  "Add feature",
				Labels: []Label{
					{Name: "enhancement"},
				},
			},
			want: "#456 | Add feature | [enhancement]",
		},
		{
			name: "Issue with multiple labels",
			issue: Issue{
				Number: 789,
				Title:  "Critical bug",
				Labels: []Label{
					{Name: "bug"},
					{Name: "critical"},
					{Name: "security"},
				},
			},
			want: "#789 | Critical bug | [bug] [critical] [security]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.issue.FormatForDisplay()

			if got != tt.want {
				t.Errorf("FormatForDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListOpenIssues(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		setupFake func() *FakeGitHubExecutor
		wantCount int
		wantErr   bool
	}{
		{
			name:  "List 10 issues successfully",
			limit: 10,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue list --limit 10 --state open --json number,title,labels,url", `[
					{"number":123,"title":"Fix bug","labels":[],"url":"https://github.com/testowner/testrepo/issues/123"},
					{"number":124,"title":"Add feature","labels":[{"name":"enhancement","color":"00ff00"}],"url":"https://github.com/testowner/testrepo/issues/124"}
				]`)
				return fake
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "Empty issue list",
			limit: 5,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue list --limit 5 --state open --json number,title,labels,url", `[]`)
				return fake
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			client, err := NewClientWithRepoAndExecutor("testowner", "testrepo", fake)
			if err != nil {
				t.Fatalf("NewClientWithRepoAndExecutor() error = %v", err)
			}

			issues, err := client.ListOpenIssues(tt.limit)

			if tt.wantErr {
				if err == nil {
					t.Error("ListOpenIssues() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ListOpenIssues() unexpected error: %v", err)
				return
			}

			if len(issues) != tt.wantCount {
				t.Errorf("ListOpenIssues() returned %d issues, want %d", len(issues), tt.wantCount)
			}

			// Verify basic structure of returned issues
			for _, issue := range issues {
				if issue.Number == 0 {
					t.Error("Issue number should not be 0")
				}
				if issue.Title == "" {
					t.Error("Issue title should not be empty")
				}
				if issue.URL == "" {
					t.Error("Issue URL should not be empty")
				}
			}

			// Verify the correct command was executed
			expectedCmd := []string{"-R", "testowner/testrepo", "issue", "list", "--limit", "10", "--state", "open", "--json", "number,title,labels,url"}
			if tt.limit == 5 {
				expectedCmd = []string{"-R", "testowner/testrepo", "issue", "list", "--limit", "5", "--state", "open", "--json", "number,title,labels,url"}
			}
			lastCmd := fake.GetLastCommand()
			if len(lastCmd) != len(expectedCmd) {
				t.Errorf("Expected command %v, got %v", expectedCmd, lastCmd)
			}
		})
	}
}

func TestGetIssue(t *testing.T) {
	tests := []struct {
		name       string
		issueNum   int
		setupFake  func() *FakeGitHubExecutor
		wantNumber int
		wantTitle  string
		wantState  string
		wantErr    bool
	}{
		{
			name:     "Get issue successfully",
			issueNum: 123,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue view 123 --json number,title,body,state,stateReason,labels,url", `{
					"number":123,
					"title":"Fix authentication bug",
					"body":"This is the bug description",
					"state":"OPEN",
					"stateReason":"",
					"labels":[{"name":"bug","color":"ff0000"}],
					"url":"https://github.com/testowner/testrepo/issues/123"
				}`)
				return fake
			},
			wantNumber: 123,
			wantTitle:  "Fix authentication bug",
			wantState:  "OPEN",
			wantErr:    false,
		},
		{
			name:     "Get closed issue",
			issueNum: 456,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue view 456 --json number,title,body,state,stateReason,labels,url", `{
					"number":456,
					"title":"Add new feature",
					"body":"Feature description",
					"state":"CLOSED",
					"stateReason":"COMPLETED",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/issues/456"
				}`)
				return fake
			},
			wantNumber: 456,
			wantTitle:  "Add new feature",
			wantState:  "CLOSED",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			client, err := NewClientWithRepoAndExecutor("testowner", "testrepo", fake)
			if err != nil {
				t.Fatalf("NewClientWithRepoAndExecutor() error = %v", err)
			}

			issue, err := client.GetIssue(tt.issueNum)

			if tt.wantErr {
				if err == nil {
					t.Error("GetIssue() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetIssue() unexpected error: %v", err)
				return
			}

			if issue.Number != tt.wantNumber {
				t.Errorf("GetIssue() Number = %d, want %d", issue.Number, tt.wantNumber)
			}

			if issue.Title != tt.wantTitle {
				t.Errorf("GetIssue() Title = %s, want %s", issue.Title, tt.wantTitle)
			}

			if issue.State != tt.wantState {
				t.Errorf("GetIssue() State = %s, want %s", issue.State, tt.wantState)
			}

			if issue.URL == "" {
				t.Error("GetIssue() URL should not be empty")
			}

			// Verify the correct command was executed
			lastCmd := fake.GetLastCommand()
			expectedCmdStart := []string{"-R", "testowner/testrepo", "issue", "view"}
			for i, expected := range expectedCmdStart {
				if i >= len(lastCmd) || lastCmd[i] != expected {
					t.Errorf("Expected command to start with %v, got %v", expectedCmdStart, lastCmd)
					break
				}
			}
		})
	}
}

func TestIsIssueMerged(t *testing.T) {
	tests := []struct {
		name       string
		issueNum   int
		setupFake  func() *FakeGitHubExecutor
		wantMerged bool
		wantErr    bool
	}{
		{
			name:     "Issue is merged",
			issueNum: 123,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue view 123 --json number,title,body,state,stateReason,labels,url", `{
					"number":123,
					"title":"Fix bug",
					"body":"",
					"state":"CLOSED",
					"stateReason":"COMPLETED",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/issues/123"
				}`)
				fake.SetResponse("-R testowner/testrepo pr list --state merged --search closes #123 OR fixes #123 OR resolves #123 --json number --jq length", "1")
				return fake
			},
			wantMerged: true,
			wantErr:    false,
		},
		{
			name:     "Issue is closed but not merged",
			issueNum: 456,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue view 456 --json number,title,body,state,stateReason,labels,url", `{
					"number":456,
					"title":"Won't fix",
					"body":"",
					"state":"CLOSED",
					"stateReason":"NOT_PLANNED",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/issues/456"
				}`)
				fake.SetResponse("-R testowner/testrepo pr list --state merged --search closes #456 OR fixes #456 OR resolves #456 --json number --jq length", "0")
				return fake
			},
			wantMerged: false,
			wantErr:    false,
		},
		{
			name:     "Issue is still open",
			issueNum: 789,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue view 789 --json number,title,body,state,stateReason,labels,url", `{
					"number":789,
					"title":"In progress",
					"body":"",
					"state":"OPEN",
					"stateReason":"",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/issues/789"
				}`)
				return fake
			},
			wantMerged: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			client, err := NewClientWithRepoAndExecutor("testowner", "testrepo", fake)
			if err != nil {
				t.Fatalf("NewClientWithRepoAndExecutor() error = %v", err)
			}

			merged, err := client.IsIssueMerged(tt.issueNum)

			if tt.wantErr {
				if err == nil {
					t.Error("IsIssueMerged() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("IsIssueMerged() unexpected error: %v", err)
				return
			}

			if merged != tt.wantMerged {
				t.Errorf("IsIssueMerged() = %v, want %v", merged, tt.wantMerged)
			}

			// Verify commands were executed
			// Should have at minimum: --version, auth status, issue view
			if len(fake.Commands) < 3 {
				t.Errorf("Expected at least 3 commands, got %d", len(fake.Commands))
			}
		})
	}
}
