package github

import (
	"testing"
)

func TestPullRequestSanitizedTitle(t *testing.T) {
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
			pr := &PullRequest{Title: tt.title}
			got := pr.SanitizedTitle()

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

func TestPullRequestBranchName(t *testing.T) {
	tests := []struct {
		name   string
		number int
		title  string
		want   string
	}{
		{
			name:   "Simple PR",
			number: 123,
			title:  "Fix login bug",
			want:   "pr/123-fix-login-bug",
		},
		{
			name:   "PR with special characters",
			number: 456,
			title:  "Add: New Feature!",
			want:   "pr/456-add-new-feature",
		},
		{
			name:   "Long title",
			number: 789,
			title:  "This is a very long PR title that should be truncated properly",
			want:   "pr/789-this-is-a-very-long-pr-title-that-should",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PullRequest{
				Number: tt.number,
				Title:  tt.title,
			}
			got := pr.BranchName()

			if got != tt.want {
				t.Errorf("BranchName() = %v, want %v", got, tt.want)
			}

			// Verify format is pr/<number>-<sanitized>
			if len(got) < 4 || got[:3] != "pr/" {
				t.Errorf("BranchName() should start with 'pr/', got %v", got)
			}
		})
	}
}

func TestPullRequestFormatForDisplay(t *testing.T) {
	tests := []struct {
		name string
		pr   PullRequest
		want string
	}{
		{
			name: "PR without labels",
			pr: PullRequest{
				Number:    123,
				Title:     "Fix bug",
				Author:    Author{Login: "octocat"},
				Additions: 10,
				Deletions: 5,
			},
			want: "#123 | Fix bug | @octocat | +10 -5",
		},
		{
			name: "PR with single label",
			pr: PullRequest{
				Number:    456,
				Title:     "Add feature",
				Author:    Author{Login: "developer"},
				Additions: 100,
				Deletions: 20,
				Labels: []Label{
					{Name: "enhancement"},
				},
			},
			want: "#456 | Add feature | @developer | +100 -20 | [enhancement]",
		},
		{
			name: "PR with multiple labels",
			pr: PullRequest{
				Number:    789,
				Title:     "Critical bug",
				Author:    Author{Login: "contributor"},
				Additions: 50,
				Deletions: 30,
				Labels: []Label{
					{Name: "bug"},
					{Name: "critical"},
					{Name: "security"},
				},
			},
			want: "#789 | Critical bug | @contributor | +50 -30 | [bug] [critical] [security]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.FormatForDisplay()

			if got != tt.want {
				t.Errorf("FormatForDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListOpenPRs(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		setupFake func() *FakeGitHubExecutor
		wantCount int
		wantErr   bool
	}{
		{
			name:  "List 10 PRs successfully",
			limit: 10,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr list --limit 10 --state open --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `[
					{
						"number":123,
						"title":"Fix bug",
						"body":"Fix description",
						"state":"OPEN",
						"author":{"login":"octocat","name":"Octocat","is_bot":false},
						"headRefName":"fix-bug",
						"baseRefName":"main",
						"labels":[],
						"url":"https://github.com/testowner/testrepo/pull/123",
						"isDraft":false,
						"reviewRequests":[],
						"additions":10,
						"deletions":5,
						"changedFiles":2,
						"statusCheckRollup":[]
					},
					{
						"number":124,
						"title":"Add feature",
						"body":"Feature description",
						"state":"OPEN",
						"author":{"login":"developer","name":"Dev","is_bot":false},
						"headRefName":"feature",
						"baseRefName":"main",
						"labels":[{"name":"enhancement","color":"00ff00"}],
						"url":"https://github.com/testowner/testrepo/pull/124",
						"isDraft":true,
						"reviewRequests":[{"login":"reviewer"}],
						"additions":100,
						"deletions":20,
						"changedFiles":10,
						"statusCheckRollup":[{"__typename":"CheckRun","name":"CI","status":"COMPLETED","conclusion":"SUCCESS"}]
					}
				]`)
				return fake
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "Empty PR list",
			limit: 5,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr list --limit 5 --state open --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `[]`)
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

			prs, err := client.ListOpenPRs(tt.limit)

			if tt.wantErr {
				if err == nil {
					t.Error("ListOpenPRs() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ListOpenPRs() unexpected error: %v", err)
				return
			}

			if len(prs) != tt.wantCount {
				t.Errorf("ListOpenPRs() returned %d PRs, want %d", len(prs), tt.wantCount)
			}

			// Verify basic structure of returned PRs
			for _, pr := range prs {
				if pr.Number == 0 {
					t.Error("PR number should not be 0")
				}
				if pr.Title == "" {
					t.Error("PR title should not be empty")
				}
				if pr.URL == "" {
					t.Error("PR URL should not be empty")
				}
				if pr.Author.Login == "" {
					t.Error("PR author login should not be empty")
				}
			}
		})
	}
}

func TestGetPR(t *testing.T) {
	tests := []struct {
		name       string
		prNum      int
		setupFake  func() *FakeGitHubExecutor
		wantNumber int
		wantTitle  string
		wantState  string
		wantErr    bool
	}{
		{
			name:  "Get PR successfully",
			prNum: 123,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 123 --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `{
					"number":123,
					"title":"Fix authentication bug",
					"body":"This is the bug fix",
					"state":"OPEN",
					"author":{"login":"octocat","name":"Octocat","is_bot":false},
					"headRefName":"fix-auth",
					"baseRefName":"main",
					"labels":[{"name":"bug","color":"ff0000"}],
					"url":"https://github.com/testowner/testrepo/pull/123",
					"isDraft":false,
					"reviewRequests":[],
					"additions":50,
					"deletions":10,
					"changedFiles":3,
					"statusCheckRollup":[{"__typename":"CheckRun","name":"CI","status":"COMPLETED","conclusion":"SUCCESS"}]
				}`)
				return fake
			},
			wantNumber: 123,
			wantTitle:  "Fix authentication bug",
			wantState:  "OPEN",
			wantErr:    false,
		},
		{
			name:  "Get merged PR",
			prNum: 456,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 456 --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `{
					"number":456,
					"title":"Add new feature",
					"body":"Feature description",
					"state":"MERGED",
					"author":{"login":"developer","name":"Dev","is_bot":false},
					"headRefName":"feature",
					"baseRefName":"main",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/pull/456",
					"isDraft":false,
					"reviewRequests":[],
					"additions":200,
					"deletions":50,
					"changedFiles":10,
					"statusCheckRollup":[]
				}`)
				return fake
			},
			wantNumber: 456,
			wantTitle:  "Add new feature",
			wantState:  "MERGED",
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

			pr, err := client.GetPR(tt.prNum)

			if tt.wantErr {
				if err == nil {
					t.Error("GetPR() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetPR() unexpected error: %v", err)
				return
			}

			if pr.Number != tt.wantNumber {
				t.Errorf("GetPR() Number = %d, want %d", pr.Number, tt.wantNumber)
			}

			if pr.Title != tt.wantTitle {
				t.Errorf("GetPR() Title = %s, want %s", pr.Title, tt.wantTitle)
			}

			if pr.State != tt.wantState {
				t.Errorf("GetPR() State = %s, want %s", pr.State, tt.wantState)
			}

			if pr.URL == "" {
				t.Error("GetPR() URL should not be empty")
			}
		})
	}
}

func TestIsPRMerged(t *testing.T) {
	tests := []struct {
		name       string
		prNum      int
		setupFake  func() *FakeGitHubExecutor
		wantMerged bool
		wantErr    bool
	}{
		{
			name:  "PR is merged",
			prNum: 123,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 123 --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `{
					"number":123,
					"title":"Fix bug",
					"body":"",
					"state":"MERGED",
					"author":{"login":"octocat","name":"Octocat","is_bot":false},
					"headRefName":"fix-bug",
					"baseRefName":"main",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/pull/123",
					"isDraft":false,
					"reviewRequests":[],
					"additions":10,
					"deletions":5,
					"changedFiles":2,
					"statusCheckRollup":[]
				}`)
				return fake
			},
			wantMerged: true,
			wantErr:    false,
		},
		{
			name:  "PR is closed but not merged",
			prNum: 456,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 456 --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `{
					"number":456,
					"title":"Won't merge",
					"body":"",
					"state":"CLOSED",
					"author":{"login":"octocat","name":"Octocat","is_bot":false},
					"headRefName":"feature",
					"baseRefName":"main",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/pull/456",
					"isDraft":false,
					"reviewRequests":[],
					"additions":20,
					"deletions":10,
					"changedFiles":5,
					"statusCheckRollup":[]
				}`)
				return fake
			},
			wantMerged: false,
			wantErr:    false,
		},
		{
			name:  "PR is still open",
			prNum: 789,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 789 --json number,title,body,state,author,headRefName,baseRefName,labels,url,isDraft,reviewRequests,additions,deletions,changedFiles,statusCheckRollup", `{
					"number":789,
					"title":"In progress",
					"body":"",
					"state":"OPEN",
					"author":{"login":"octocat","name":"Octocat","is_bot":false},
					"headRefName":"wip",
					"baseRefName":"main",
					"labels":[],
					"url":"https://github.com/testowner/testrepo/pull/789",
					"isDraft":true,
					"reviewRequests":[],
					"additions":30,
					"deletions":15,
					"changedFiles":7,
					"statusCheckRollup":[]
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

			merged, err := client.IsPRMerged(tt.prNum)

			if tt.wantErr {
				if err == nil {
					t.Error("IsPRMerged() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("IsPRMerged() unexpected error: %v", err)
				return
			}

			if merged != tt.wantMerged {
				t.Errorf("IsPRMerged() = %v, want %v", merged, tt.wantMerged)
			}
		})
	}
}

func TestHasMergeConflicts(t *testing.T) {
	tests := []struct {
		name         string
		prNum        int
		setupFake    func() *FakeGitHubExecutor
		wantConflict bool
		wantErr      bool
	}{
		{
			name:  "PR has merge conflicts",
			prNum: 123,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 123 --json mergeable", `{"mergeable":"CONFLICTING"}`)
				return fake
			},
			wantConflict: true,
			wantErr:      false,
		},
		{
			name:  "PR is mergeable",
			prNum: 456,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 456 --json mergeable", `{"mergeable":"MERGEABLE"}`)
				return fake
			},
			wantConflict: false,
			wantErr:      false,
		},
		{
			name:  "PR mergeable status unknown",
			prNum: 789,
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo pr view 789 --json mergeable", `{"mergeable":"UNKNOWN"}`)
				return fake
			},
			wantConflict: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			client, err := NewClientWithRepoAndExecutor("testowner", "testrepo", fake)
			if err != nil {
				t.Fatalf("NewClientWithRepoAndExecutor() error = %v", err)
			}

			hasConflict, err := client.HasMergeConflicts(tt.prNum)

			if tt.wantErr {
				if err == nil {
					t.Error("HasMergeConflicts() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("HasMergeConflicts() unexpected error: %v", err)
				return
			}

			if hasConflict != tt.wantConflict {
				t.Errorf("HasMergeConflicts() = %v, want %v", hasConflict, tt.wantConflict)
			}
		})
	}
}

func TestAllChecksPass(t *testing.T) {
	tests := []struct {
		name     string
		pr       PullRequest
		wantPass bool
	}{
		{
			name: "All checks pass",
			pr: PullRequest{
				StatusCheckRollup: []StatusCheck{
					{Name: "CI", Status: "COMPLETED", Conclusion: "SUCCESS"},
					{Name: "Tests", Status: "COMPLETED", Conclusion: "SUCCESS"},
				},
			},
			wantPass: true,
		},
		{
			name: "One check fails",
			pr: PullRequest{
				StatusCheckRollup: []StatusCheck{
					{Name: "CI", Status: "COMPLETED", Conclusion: "SUCCESS"},
					{Name: "Tests", Status: "COMPLETED", Conclusion: "FAILURE"},
				},
			},
			wantPass: false,
		},
		{
			name: "One check in progress",
			pr: PullRequest{
				StatusCheckRollup: []StatusCheck{
					{Name: "CI", Status: "COMPLETED", Conclusion: "SUCCESS"},
					{Name: "Tests", Status: "IN_PROGRESS", Conclusion: ""},
				},
			},
			wantPass: false,
		},
		{
			name: "No checks configured",
			pr: PullRequest{
				StatusCheckRollup: []StatusCheck{},
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.AllChecksPass()

			if got != tt.wantPass {
				t.Errorf("AllChecksPass() = %v, want %v", got, tt.wantPass)
			}
		})
	}
}

func TestChangeSize(t *testing.T) {
	tests := []struct {
		name      string
		additions int
		deletions int
		want      string
	}{
		{
			name:      "Extra small (< 50 lines)",
			additions: 20,
			deletions: 10,
			want:      "XS",
		},
		{
			name:      "Small (50-199 lines)",
			additions: 80,
			deletions: 40,
			want:      "S",
		},
		{
			name:      "Medium (200-499 lines)",
			additions: 200,
			deletions: 100,
			want:      "M",
		},
		{
			name:      "Large (500-999 lines)",
			additions: 500,
			deletions: 200,
			want:      "L",
		},
		{
			name:      "Extra large (>= 1000 lines)",
			additions: 800,
			deletions: 400,
			want:      "XL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PullRequest{
				Additions: tt.additions,
				Deletions: tt.deletions,
			}
			got := pr.ChangeSize()

			if got != tt.want {
				t.Errorf("ChangeSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRequestedReviewer(t *testing.T) {
	tests := []struct {
		name     string
		pr       PullRequest
		username string
		want     bool
	}{
		{
			name: "User is requested reviewer",
			pr: PullRequest{
				ReviewRequests: []ReviewRequest{
					{Login: "reviewer1"},
					{Login: "reviewer2"},
				},
			},
			username: "reviewer1",
			want:     true,
		},
		{
			name: "User is not requested reviewer",
			pr: PullRequest{
				ReviewRequests: []ReviewRequest{
					{Login: "reviewer1"},
					{Login: "reviewer2"},
				},
			},
			username: "reviewer3",
			want:     false,
		},
		{
			name: "No reviewers requested",
			pr: PullRequest{
				ReviewRequests: []ReviewRequest{},
			},
			username: "reviewer1",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.IsRequestedReviewer(tt.username)

			if got != tt.want {
				t.Errorf("IsRequestedReviewer() = %v, want %v", got, tt.want)
			}
		})
	}
}
