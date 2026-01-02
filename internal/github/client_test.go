package github

import (
	"testing"
)

func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name          string
		setupFake     func() *FakeGitHubExecutor
		wantInstalled bool
	}{
		{
			name: "gh is installed",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				return fake
			},
			wantInstalled: true,
		},
		{
			name: "gh is not installed",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetError("--version", ErrGHNotInstalled)
				return fake
			},
			wantInstalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			installed := IsInstalled(fake)

			if installed != tt.wantInstalled {
				t.Errorf("IsInstalled() = %v, want %v", installed, tt.wantInstalled)
			}

			// Verify the command was executed
			if len(fake.Commands) != 1 {
				t.Errorf("Expected 1 command, got %d", len(fake.Commands))
			}
			if len(fake.Commands) > 0 {
				cmd := fake.Commands[0]
				if len(cmd) != 1 || cmd[0] != "--version" {
					t.Errorf("Expected command [--version], got %v", cmd)
				}
			}
		})
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name      string
		setupFake func() *FakeGitHubExecutor
		wantErr   error
	}{
		{
			name: "gh is authenticated",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("auth status", "Logged in to github.com")
				return fake
			},
			wantErr: nil,
		},
		{
			name: "gh is not authenticated",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetError("auth status", ErrGHNotAuthenticated)
				return fake
			},
			wantErr: ErrGHNotAuthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			err := IsAuthenticated(fake)

			if err != tt.wantErr {
				t.Errorf("IsAuthenticated() error = %v, want %v", err, tt.wantErr)
			}

			// Verify the command was executed
			if len(fake.Commands) != 1 {
				t.Errorf("Expected 1 command, got %d", len(fake.Commands))
			}
			if len(fake.Commands) > 0 {
				cmd := fake.Commands[0]
				if len(cmd) != 2 || cmd[0] != "auth" || cmd[1] != "status" {
					t.Errorf("Expected command [auth status], got %v", cmd)
				}
			}
		})
	}
}

func TestNewClientWithRepoAndExecutor(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		repo      string
		setupFake func() *FakeGitHubExecutor
		wantErr   bool
	}{
		{
			name:  "Valid owner and repo",
			owner: "testowner",
			repo:  "testrepo",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				return fake
			},
			wantErr: false,
		},
		{
			name:  "gh not installed",
			owner: "testowner",
			repo:  "testrepo",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetError("--version", ErrGHNotInstalled)
				return fake
			},
			wantErr: true,
		},
		{
			name:  "gh not authenticated",
			owner: "testowner",
			repo:  "testrepo",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetError("auth status", ErrGHNotAuthenticated)
				return fake
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			client, err := NewClientWithRepoAndExecutor(tt.owner, tt.repo, fake)

			if tt.wantErr {
				if err == nil {
					t.Error("NewClientWithRepoAndExecutor() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewClientWithRepoAndExecutor() unexpected error: %v", err)
				return
			}

			if client.Owner != tt.owner {
				t.Errorf("NewClientWithRepoAndExecutor() Owner = %v, want %v", client.Owner, tt.owner)
			}

			if client.Repo != tt.repo {
				t.Errorf("NewClientWithRepoAndExecutor() Repo = %v, want %v", client.Repo, tt.repo)
			}

			// Verify commands were executed
			if len(fake.Commands) < 2 {
				t.Errorf("Expected at least 2 commands (version and auth), got %d", len(fake.Commands))
			}
		})
	}
}

func TestClientExecGH(t *testing.T) {
	tests := []struct {
		name      string
		setupFake func() *FakeGitHubExecutor
		args      []string
		wantErr   bool
	}{
		{
			name: "Successful command execution",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetResponse("-R testowner/testrepo issue list --limit 1 --json number", `[{"number":123}]`)
				return fake
			},
			args:    []string{"issue", "list", "--limit", "1", "--json", "number"},
			wantErr: false,
		},
		{
			name: "Command execution fails",
			setupFake: func() *FakeGitHubExecutor {
				fake := NewFakeGitHubExecutor()
				fake.SetResponse("--version", "gh version 2.0.0")
				fake.SetResponse("auth status", "Logged in to github.com")
				fake.SetError("-R testowner/testrepo issue view 999 --json number", ErrGHNotInstalled)
				return fake
			},
			args:    []string{"issue", "view", "999", "--json", "number"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := tt.setupFake()
			client, err := NewClientWithRepoAndExecutor("testowner", "testrepo", fake)
			if err != nil {
				t.Fatalf("NewClientWithRepoAndExecutor() error = %v", err)
			}

			_, err = client.execGHInRepo(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Error("execGHInRepo() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("execGHInRepo() unexpected error: %v", err)
				}
			}

			// Verify that commands were recorded
			// Should have: --version, auth status, and the actual command
			if len(fake.Commands) < 3 {
				t.Errorf("Expected at least 3 commands, got %d", len(fake.Commands))
			}
		})
	}
}
