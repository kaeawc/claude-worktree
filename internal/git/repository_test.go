package git

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestIsGitRepository(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		setupCmd string
		cmdErr   error
		expected bool
	}{
		{
			name:     "valid git repository",
			path:     "/test/repo",
			setupCmd: "rev-parse --git-dir",
			cmdErr:   nil,
			expected: true,
		},
		{
			name:     "non-existent path",
			path:     "/nonexistent/path",
			setupCmd: "rev-parse --git-dir",
			cmdErr:   errors.New("not a git repository"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeExec := NewFakeGitExecutor()
			if tt.cmdErr != nil {
				fakeExec.SetError(tt.setupCmd, tt.cmdErr)
			} else {
				fakeExec.SetResponse(tt.setupCmd, ".git")
			}

			result := isGitRepository(tt.path, fakeExec)
			if result != tt.expected {
				t.Errorf("isGitRepository(%q) = %v, want %v", tt.path, result, tt.expected)
			}

			// Verify the command was executed
			if len(fakeExec.Commands) != 1 {
				t.Errorf("Expected 1 command, got %d", len(fakeExec.Commands))
			}
			if len(fakeExec.Commands) > 0 {
				cmd := fakeExec.Commands[0]
				// Check that the command includes the directory context
				if len(cmd) < 3 || cmd[0] != "[in:"+tt.path+"]" {
					t.Errorf("Expected command with dir context, got %v", cmd)
				}
			}
		})
	}
}

func TestGetRepositoryRoot(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		rootPath    string
		cmdErr      error
		wantErr     bool
		expectedDir string
	}{
		{
			name:        "from repository root",
			path:        "/test/repo",
			rootPath:    "/test/repo",
			cmdErr:      nil,
			wantErr:     false,
			expectedDir: "/test/repo",
		},
		{
			name:        "from subdirectory",
			path:        "/test/repo/subdir",
			rootPath:    "/test/repo",
			cmdErr:      nil,
			wantErr:     false,
			expectedDir: "/test/repo",
		},
		{
			name:    "not a git repository",
			path:    "/not/a/repo",
			cmdErr:  errors.New("not a git repository"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeExec := NewFakeGitExecutor()
			if tt.cmdErr != nil {
				fakeExec.SetError("rev-parse --show-toplevel", tt.cmdErr)
			} else {
				fakeExec.SetResponse("rev-parse --show-toplevel", tt.rootPath)
			}

			root, err := getRepositoryRoot(tt.path, fakeExec)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRepositoryRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if root != tt.expectedDir {
					t.Errorf("getRepositoryRoot() = %v, want %v", root, tt.expectedDir)
				}
			}

			// Verify the command was executed
			if len(fakeExec.Commands) != 1 {
				t.Errorf("Expected 1 command, got %d", len(fakeExec.Commands))
			}
		})
	}
}

func TestNewRepositoryFromPath(t *testing.T) {
	fakeExec := NewFakeGitExecutor()
	fakeFS := NewFakeFileSystem()

	// Setup fake git responses
	fakeExec.SetResponse("rev-parse --git-dir", ".git")
	fakeExec.SetResponse("rev-parse --show-toplevel", "/test/repo")

	// Setup fake filesystem
	fakeFS.HomeDir = "/home/testuser"
	fakeFS.Dirs["/test/repo"] = true

	repo, err := NewRepositoryFromPathWithDeps("/test/repo", fakeExec, fakeFS)
	if err != nil {
		t.Fatalf("NewRepositoryFromPathWithDeps() error = %v", err)
	}

	if repo.RootPath != "/test/repo" {
		t.Errorf("RootPath = %v, want %v", repo.RootPath, "/test/repo")
	}

	expectedFolder := "repo"
	if repo.SourceFolder != expectedFolder {
		t.Errorf("SourceFolder = %v, want %v", repo.SourceFolder, expectedFolder)
	}

	expectedBase := filepath.Join("/home/testuser", "worktrees", expectedFolder)
	if repo.WorktreeBase != expectedBase {
		t.Errorf("WorktreeBase = %v, want %v", repo.WorktreeBase, expectedBase)
	}

	// Verify commands executed
	if len(fakeExec.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d: %v", len(fakeExec.Commands), fakeExec.Commands)
	}

	// Verify filesystem operations
	foundHomeDir := false
	for _, op := range fakeFS.OperationLog {
		if op == "UserHomeDir()" {
			foundHomeDir = true
			break
		}
	}
	if !foundHomeDir {
		t.Errorf("Expected UserHomeDir() to be called")
	}
}

func TestBranchExists(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		cmdErr     error
		expected   bool
	}{
		{
			name:       "existing branch",
			branchName: "test-branch",
			cmdErr:     nil,
			expected:   true,
		},
		{
			name:       "non-existent branch",
			branchName: "nonexistent-branch",
			cmdErr:     errors.New("not found"),
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeExec := NewFakeGitExecutor()
			fakeFS := NewFakeFileSystem()

			// Setup repository creation
			fakeExec.SetResponse("rev-parse --git-dir", ".git")
			fakeExec.SetResponse("rev-parse --show-toplevel", "/test/repo")
			fakeFS.HomeDir = "/home/testuser"

			repo, err := NewRepositoryFromPathWithDeps("/test/repo", fakeExec, fakeFS)
			if err != nil {
				t.Fatalf("NewRepositoryFromPathWithDeps() error = %v", err)
			}

			// Setup branch check response
			expectedCmd := "show-ref --verify --quiet refs/heads/" + tt.branchName
			if tt.cmdErr != nil {
				fakeExec.SetError(expectedCmd, tt.cmdErr)
			} else {
				fakeExec.SetResponse(expectedCmd, "abc123 refs/heads/"+tt.branchName)
			}

			result := repo.BranchExists(tt.branchName)
			if result != tt.expected {
				t.Errorf("BranchExists(%q) = %v, want %v", tt.branchName, result, tt.expected)
			}

			// Verify the branch check command was executed
			foundBranchCheck := false
			for _, cmd := range fakeExec.Commands {
				if len(cmd) >= 5 && cmd[1] == "show-ref" {
					foundBranchCheck = true
					break
				}
			}
			if !foundBranchCheck {
				t.Errorf("Expected show-ref command to be executed")
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tests := []struct {
		name         string
		branchOutput string
		cmdErr       error
		wantErr      bool
		expected     string
	}{
		{
			name:         "on main branch",
			branchOutput: "main",
			cmdErr:       nil,
			wantErr:      false,
			expected:     "main",
		},
		{
			name:         "on feature branch",
			branchOutput: "feature/test",
			cmdErr:       nil,
			wantErr:      false,
			expected:     "feature/test",
		},
		{
			name:         "detached HEAD",
			branchOutput: "HEAD",
			cmdErr:       nil,
			wantErr:      false,
			expected:     "",
		},
		{
			name:     "error getting branch",
			cmdErr:   errors.New("git error"),
			wantErr:  true,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeExec := NewFakeGitExecutor()
			fakeFS := NewFakeFileSystem()

			// Setup repository creation
			fakeExec.SetResponse("rev-parse --git-dir", ".git")
			fakeExec.SetResponse("rev-parse --show-toplevel", "/test/repo")
			fakeFS.HomeDir = "/home/testuser"

			repo, err := NewRepositoryFromPathWithDeps("/test/repo", fakeExec, fakeFS)
			if err != nil {
				t.Fatalf("NewRepositoryFromPathWithDeps() error = %v", err)
			}

			// Setup current branch response
			if tt.cmdErr != nil {
				fakeExec.SetError("rev-parse --abbrev-ref HEAD", tt.cmdErr)
			} else {
				fakeExec.SetResponse("rev-parse --abbrev-ref HEAD", tt.branchOutput)
			}

			branch, err := repo.GetCurrentBranch()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if branch != tt.expected {
				t.Errorf("GetCurrentBranch() = %v, want %v", branch, tt.expected)
			}

			// Verify the command was executed
			foundBranchCmd := false
			for _, cmd := range fakeExec.Commands {
				if len(cmd) >= 4 && cmd[1] == "rev-parse" && cmd[2] == "--abbrev-ref" {
					foundBranchCmd = true
					break
				}
			}
			if !foundBranchCmd {
				t.Errorf("Expected rev-parse --abbrev-ref HEAD command to be executed")
			}
		})
	}
}

func TestCreateAndDeleteBranch(t *testing.T) {
	fakeExec := NewFakeGitExecutor()
	fakeFS := NewFakeFileSystem()

	// Setup repository creation
	fakeExec.SetResponse("rev-parse --git-dir", ".git")
	fakeExec.SetResponse("rev-parse --show-toplevel", "/test/repo")
	fakeFS.HomeDir = "/home/testuser"

	repo, err := NewRepositoryFromPathWithDeps("/test/repo", fakeExec, fakeFS)
	if err != nil {
		t.Fatalf("NewRepositoryFromPathWithDeps() error = %v", err)
	}

	// Setup responses for current branch
	fakeExec.SetResponse("rev-parse --abbrev-ref HEAD", "main")

	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// Create a new branch
	testBranch := "test-new-branch"
	fakeExec.SetResponse("branch "+testBranch+" "+currentBranch, "")

	if err := repo.CreateBranch(testBranch, currentBranch); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Verify branch creation command was executed
	foundCreateCmd := false
	for _, cmd := range fakeExec.Commands {
		if len(cmd) >= 4 && cmd[1] == "branch" && cmd[2] == testBranch {
			foundCreateCmd = true
			break
		}
	}
	if !foundCreateCmd {
		t.Errorf("Expected branch create command to be executed")
	}

	// Setup branch exists check
	fakeExec.SetResponse("show-ref --verify --quiet refs/heads/"+testBranch, "abc123 refs/heads/"+testBranch)

	// Verify branch exists
	if !repo.BranchExists(testBranch) {
		t.Errorf("Branch %q should exist after creation", testBranch)
	}

	// Delete the branch
	fakeExec.SetResponse("branch -D "+testBranch, "")

	if err := repo.DeleteBranch(testBranch); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	// Verify branch delete command was executed
	foundDeleteCmd := false
	for _, cmd := range fakeExec.Commands {
		if len(cmd) >= 4 && cmd[1] == "branch" && cmd[2] == "-D" && cmd[3] == testBranch {
			foundDeleteCmd = true
			break
		}
	}
	if !foundDeleteCmd {
		t.Errorf("Expected branch delete command to be executed")
	}

	// Setup branch not exists check
	fakeExec.SetError("show-ref --verify --quiet refs/heads/"+testBranch, errors.New("not found"))

	// Verify branch no longer exists
	if repo.BranchExists(testBranch) {
		t.Errorf("Branch %q should not exist after deletion", testBranch)
	}
}

func TestGetDefaultBranch(t *testing.T) {
	tests := []struct {
		name              string
		symbolicRefOutput string
		symbolicRefError  error
		mainExists        bool
		masterExists      bool
		remoteMainExists  bool
		expected          string
		wantErr           bool
	}{
		{
			name:              "default from remote HEAD",
			symbolicRefOutput: "refs/remotes/origin/main",
			symbolicRefError:  nil,
			expected:          "main",
			wantErr:           false,
		},
		{
			name:             "local main branch",
			symbolicRefError: errors.New("no remote"),
			mainExists:       true,
			expected:         "main",
			wantErr:          false,
		},
		{
			name:             "local master branch",
			symbolicRefError: errors.New("no remote"),
			masterExists:     true,
			expected:         "master",
			wantErr:          false,
		},
		{
			name:             "remote main branch",
			symbolicRefError: errors.New("no remote"),
			mainExists:       false,
			masterExists:     false,
			remoteMainExists: true,
			expected:         "main",
			wantErr:          false,
		},
		{
			name:             "no default branch found",
			symbolicRefError: errors.New("no remote"),
			mainExists:       false,
			masterExists:     false,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeExec := NewFakeGitExecutor()
			fakeFS := NewFakeFileSystem()

			// Setup repository creation
			fakeExec.SetResponse("rev-parse --git-dir", ".git")
			fakeExec.SetResponse("rev-parse --show-toplevel", "/test/repo")
			fakeFS.HomeDir = "/home/testuser"

			repo, err := NewRepositoryFromPathWithDeps("/test/repo", fakeExec, fakeFS)
			if err != nil {
				t.Fatalf("NewRepositoryFromPathWithDeps() error = %v", err)
			}

			// Setup symbolic-ref response
			if tt.symbolicRefError != nil {
				fakeExec.SetError("symbolic-ref refs/remotes/origin/HEAD", tt.symbolicRefError)
			} else {
				fakeExec.SetResponse("symbolic-ref refs/remotes/origin/HEAD", tt.symbolicRefOutput)
			}

			// Setup branch exists checks
			if tt.mainExists {
				fakeExec.SetResponse("show-ref --verify --quiet refs/heads/main", "abc123 refs/heads/main")
			} else {
				fakeExec.SetError("show-ref --verify --quiet refs/heads/main", errors.New("not found"))
			}

			if tt.masterExists {
				fakeExec.SetResponse("show-ref --verify --quiet refs/heads/master", "abc123 refs/heads/master")
			} else {
				fakeExec.SetError("show-ref --verify --quiet refs/heads/master", errors.New("not found"))
			}

			if tt.remoteMainExists {
				fakeExec.SetResponse("show-ref --verify --quiet refs/remotes/origin/main", "abc123 refs/remotes/origin/main")
			} else {
				fakeExec.SetError("show-ref --verify --quiet refs/remotes/origin/main", errors.New("not found"))
			}

			fakeExec.SetError("show-ref --verify --quiet refs/remotes/origin/master", errors.New("not found"))

			defaultBranch, err := repo.GetDefaultBranch()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && defaultBranch != tt.expected {
				t.Errorf("GetDefaultBranch() = %v, want %v", defaultBranch, tt.expected)
			}

			// Verify symbolic-ref command was executed
			foundSymbolicRef := false
			for _, cmd := range fakeExec.Commands {
				if len(cmd) >= 3 && cmd[1] == "symbolic-ref" {
					foundSymbolicRef = true
					break
				}
			}
			if !foundSymbolicRef {
				t.Errorf("Expected symbolic-ref command to be executed")
			}
		})
	}
}
