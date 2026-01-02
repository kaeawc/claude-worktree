package git

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// FakeHookExecutor is a fake implementation for testing
type FakeHookExecutor struct {
	// ExecutedHooks records all executed hooks
	ExecutedHooks []ExecutedHook
	// Errors maps hook paths to errors
	Errors map[string]error
	// IsExecutableFunc allows customizing IsExecutable behavior
	IsExecutableFunc func(path string) bool
}

// ExecutedHook records details about an executed hook
type ExecutedHook struct {
	Path       string
	Params     []string
	Env        []string
	WorkingDir string
	Output     string
}

// NewFakeHookExecutor creates a new fake hook executor
func NewFakeHookExecutor() *FakeHookExecutor {
	return &FakeHookExecutor{
		ExecutedHooks: []ExecutedHook{},
		Errors:        make(map[string]error),
	}
}

// Execute records the hook execution
func (e *FakeHookExecutor) Execute(hookPath string, params []string, env []string, workingDir string, output io.Writer) error {
	// Write fake output
	outputStr := "hook output from " + hookPath
	output.Write([]byte(outputStr))

	// Record execution
	e.ExecutedHooks = append(e.ExecutedHooks, ExecutedHook{
		Path:       hookPath,
		Params:     params,
		Env:        env,
		WorkingDir: workingDir,
		Output:     outputStr,
	})

	// Check for configured error
	if err, ok := e.Errors[hookPath]; ok {
		return err
	}

	return nil
}

// IsExecutable checks if a hook is executable
func (e *FakeHookExecutor) IsExecutable(path string) bool {
	if e.IsExecutableFunc != nil {
		return e.IsExecutableFunc(path)
	}
	return false
}

// SetError configures an error for a specific hook path
func (e *FakeHookExecutor) SetError(hookPath string, err error) {
	e.Errors[hookPath] = err
}

func TestHookManager_FindHookDirectories(t *testing.T) {
	tests := []struct {
		name            string
		configResponses map[string]string
		gitResponses    map[string]string
		expectedDirs    []string
	}{
		{
			name: "standard hooks directory",
			configResponses: map[string]string{
				"config --local --get core.hooksPath":  "",
				"config --global --get core.hooksPath": "",
			},
			gitResponses: map[string]string{
				"rev-parse --git-common-dir": ".git",
			},
			expectedDirs: []string{
				"/test/repo/.git/hooks",
			},
		},
		{
			name: "custom hooks path configured (relative)",
			configResponses: map[string]string{
				"config --local --get core.hooksPath":  ".githooks",
				"config --global --get core.hooksPath": "",
			},
			gitResponses: map[string]string{
				"rev-parse --git-common-dir": ".git",
			},
			expectedDirs: []string{
				"/test/repo/.githooks",
				"/test/repo/.git/hooks",
			},
		},
		{
			name: "custom hooks path configured (absolute)",
			configResponses: map[string]string{
				"config --local --get core.hooksPath": func() string {
					if filepath.Separator == '\\' {
						// Windows needs drive letter for absolute paths
						return "C:\\custom\\hooks"
					}
					return "/custom/hooks"
				}(),
				"config --global --get core.hooksPath": "",
			},
			gitResponses: map[string]string{
				"rev-parse --git-common-dir": ".git",
			},
			expectedDirs: []string{
				func() string {
					if filepath.Separator == '\\' {
						return "C:\\custom\\hooks"
					}
					return "/custom/hooks"
				}(),
				"/test/repo/.git/hooks",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake git executor
			fakeGit := NewFakeGitExecutor()
			for cmd, resp := range tt.gitResponses {
				fakeGit.SetResponse(cmd, resp)
			}
			for cmd, resp := range tt.configResponses {
				fakeGit.SetResponse(cmd, resp)
			}

			// Setup config
			config := NewConfigWithExecutor("/test/repo", fakeGit)

			// Setup fake hook executor
			fakeHook := NewFakeHookExecutor()

			// Create hook manager
			output := &bytes.Buffer{}
			hm := NewHookManager("/test/repo", config, fakeGit, fakeHook, output)

			// Find hook directories
			dirs, err := hm.findHookDirectories()
			if err != nil {
				t.Fatalf("findHookDirectories() error = %v", err)
			}

			// Verify the expected directories are found
			// Note: .husky directory may not exist in test environment, so we only check
			// that custom paths and standard hooks are properly discovered
			if len(dirs) < 1 {
				t.Errorf("Expected at least 1 directory, got %d", len(dirs))
			}

			// Check that expected directories are in the result (allowing for .husky to be absent)
			for _, expected := range tt.expectedDirs {
				// Skip checking .husky since it may not exist in test environment
				if strings.Contains(expected, ".husky") {
					continue
				}
				found := false
				// Normalize paths for cross-platform comparison
				expectedNorm := filepath.FromSlash(expected)
				for _, dir := range dirs {
					if dir == expectedNorm {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected directory %s not found in result: %v", expectedNorm, dirs)
				}
			}
		})
	}
}

func TestHookManager_ExecuteWorktreeHooks(t *testing.T) {
	tests := []struct {
		name            string
		runHooks        bool
		failOnError     bool
		customHooks     string
		executableHooks []string
		hookErrors      map[string]error
		expectError     bool
		expectedHooks   []string
	}{
		{
			name:          "hooks disabled",
			runHooks:      false,
			failOnError:   false,
			customHooks:   "",
			expectedHooks: []string{},
		},
		{
			name:        "execute post-checkout and post-worktree",
			runHooks:    true,
			failOnError: false,
			customHooks: "",
			executableHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
				"/test/repo/.git/hooks/post-worktree",
			},
			expectedHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
				"/test/repo/.git/hooks/post-worktree",
			},
		},
		{
			name:        "execute custom hooks",
			runHooks:    true,
			failOnError: false,
			customHooks: "post-merge post-commit",
			executableHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
				"/test/repo/.git/hooks/post-worktree",
				"/test/repo/.git/hooks/post-merge",
				"/test/repo/.git/hooks/post-commit",
			},
			expectedHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
				"/test/repo/.git/hooks/post-worktree",
				"/test/repo/.git/hooks/post-merge",
				"/test/repo/.git/hooks/post-commit",
			},
		},
		{
			name:        "hook failure with fail-on-error disabled",
			runHooks:    true,
			failOnError: false,
			customHooks: "",
			executableHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
			},
			hookErrors: map[string]error{
				"/test/repo/.git/hooks/post-checkout": errors.New("hook failed"),
			},
			expectError: false,
			expectedHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
			},
		},
		{
			name:        "hook failure with fail-on-error enabled",
			runHooks:    true,
			failOnError: true,
			customHooks: "",
			executableHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
			},
			hookErrors: map[string]error{
				"/test/repo/.git/hooks/post-checkout": errors.New("hook failed"),
			},
			expectError: true,
			expectedHooks: []string{
				"/test/repo/.git/hooks/post-checkout",
			},
		},
		{
			name:        "comma-separated custom hooks",
			runHooks:    true,
			failOnError: false,
			customHooks: "post-merge,post-commit",
			executableHooks: []string{
				"/test/repo/.git/hooks/post-merge",
				"/test/repo/.git/hooks/post-commit",
			},
			expectedHooks: []string{
				"/test/repo/.git/hooks/post-merge",
				"/test/repo/.git/hooks/post-commit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake git executor
			fakeGit := NewFakeGitExecutor()
			fakeGit.SetResponse("config --local --get --bool auto-worktree.run-hooks", func() string {
				if tt.runHooks {
					return "true"
				}
				return ""
			}())
			fakeGit.SetResponse("config --global --get --bool auto-worktree.run-hooks", func() string {
				if tt.runHooks {
					return "true"
				}
				return ""
			}())
			fakeGit.SetResponse("config --local --get --bool auto-worktree.fail-on-hook-error", func() string {
				if tt.failOnError {
					return "true"
				}
				return "false"
			}())
			fakeGit.SetResponse("config --global --get --bool auto-worktree.fail-on-hook-error", "false")
			fakeGit.SetResponse("config --local --get auto-worktree.custom-hooks", tt.customHooks)
			fakeGit.SetResponse("config --global --get auto-worktree.custom-hooks", "")
			fakeGit.SetResponse("rev-parse --git-common-dir", ".git")
			fakeGit.SetResponse("rev-parse HEAD", "abc123def456")

			// Setup config
			config := NewConfigWithExecutor("/test/repo", fakeGit)

			// Setup fake hook executor
			fakeHook := NewFakeHookExecutor()
			fakeHook.IsExecutableFunc = func(path string) bool {
				for _, hookPath := range tt.executableHooks {
					// Normalize hook path for cross-platform comparison
					normalizedHook := filepath.FromSlash(hookPath)

					// Check exact match or with Windows extensions
					if path == normalizedHook ||
						path == normalizedHook+".bat" ||
						path == normalizedHook+".cmd" ||
						path == normalizedHook+".exe" ||
						path == normalizedHook+".ps1" {
						return true
					}
				}
				return false
			}

			// Configure hook errors (normalize paths and add extension variants for Windows)
			for hookPath, err := range tt.hookErrors {
				normalizedPath := filepath.FromSlash(hookPath)
				// Set error for base path and all Windows extension variants
				fakeHook.SetError(normalizedPath, err)
				fakeHook.SetError(normalizedPath+".bat", err)
				fakeHook.SetError(normalizedPath+".cmd", err)
				fakeHook.SetError(normalizedPath+".exe", err)
				fakeHook.SetError(normalizedPath+".ps1", err)
			}

			// Create hook manager
			output := &bytes.Buffer{}
			hm := NewHookManager("/test/repo", config, fakeGit, fakeHook, output)

			// Execute worktree hooks
			err := hm.ExecuteWorktreeHooks("/test/repo/worktrees/test-branch")

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify executed hooks
			if len(fakeHook.ExecutedHooks) != len(tt.expectedHooks) {
				t.Errorf("Expected %d hooks executed, got %d", len(tt.expectedHooks), len(fakeHook.ExecutedHooks))
			}

			for i, expectedPath := range tt.expectedHooks {
				if i >= len(fakeHook.ExecutedHooks) {
					break
				}
				// Normalize expected path for cross-platform comparison
				expectedNorm := filepath.FromSlash(expectedPath)
				actualPath := fakeHook.ExecutedHooks[i].Path

				// Check if path matches (with or without Windows extensions)
				pathMatches := actualPath == expectedNorm ||
					actualPath == expectedNorm+".bat" ||
					actualPath == expectedNorm+".cmd" ||
					actualPath == expectedNorm+".exe" ||
					actualPath == expectedNorm+".ps1"

				if !pathMatches {
					t.Errorf("Hook %d: expected path %s, got %s", i, expectedNorm, actualPath)
				}
			}

			// Verify post-checkout hook has correct parameters
			if tt.runHooks && len(fakeHook.ExecutedHooks) > 0 {
				firstHook := fakeHook.ExecutedHooks[0]
				// Check for post-checkout in path (handles both Unix and Windows with extensions)
				if strings.Contains(firstHook.Path, "post-checkout") {
					expectedParams := []string{
						"0000000000000000000000000000000000000000",
						"abc123def456",
						"1",
					}
					if len(firstHook.Params) != len(expectedParams) {
						t.Errorf("Expected %d params, got %d", len(expectedParams), len(firstHook.Params))
					}
					for i, expected := range expectedParams {
						if i >= len(firstHook.Params) {
							break
						}
						if firstHook.Params[i] != expected {
							t.Errorf("Param %d: expected %s, got %s", i, expected, firstHook.Params[i])
						}
					}
				}
			}
		})
	}
}

func TestConfig_GetCustomHooks(t *testing.T) {
	tests := []struct {
		name          string
		configValue   string
		expectedHooks []string
	}{
		{
			name:          "no custom hooks",
			configValue:   "",
			expectedHooks: []string{},
		},
		{
			name:          "space-separated hooks",
			configValue:   "post-merge post-commit",
			expectedHooks: []string{"post-merge", "post-commit"},
		},
		{
			name:          "comma-separated hooks",
			configValue:   "post-merge,post-commit",
			expectedHooks: []string{"post-merge", "post-commit"},
		},
		{
			name:          "mixed separators",
			configValue:   "post-merge, post-commit post-rewrite",
			expectedHooks: []string{"post-merge", "post-commit", "post-rewrite"},
		},
		{
			name:          "extra whitespace",
			configValue:   "  post-merge  ,  post-commit  ",
			expectedHooks: []string{"post-merge", "post-commit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake git executor
			fakeGit := NewFakeGitExecutor()
			fakeGit.SetResponse("config --local --get auto-worktree.custom-hooks", tt.configValue)
			fakeGit.SetResponse("config --global --get auto-worktree.custom-hooks", "")

			// Setup config
			config := NewConfigWithExecutor("/test/repo", fakeGit)

			// Get custom hooks
			hooks := config.GetCustomHooks()

			// Verify
			if len(hooks) != len(tt.expectedHooks) {
				t.Errorf("Expected %d hooks, got %d", len(tt.expectedHooks), len(hooks))
			}

			for i, expected := range tt.expectedHooks {
				if i >= len(hooks) {
					break
				}
				if hooks[i] != expected {
					t.Errorf("Hook %d: expected %s, got %s", i, expected, hooks[i])
				}
			}
		})
	}
}

func TestHookManager_WorkingDirectory(t *testing.T) {
	// Setup fake executors
	fakeGit := NewFakeGitExecutor()
	fakeGit.SetResponse("config --local --get --bool auto-worktree.run-hooks", "true")
	fakeGit.SetResponse("config --global --get --bool auto-worktree.run-hooks", "true")
	fakeGit.SetResponse("config --local --get --bool auto-worktree.fail-on-hook-error", "false")
	fakeGit.SetResponse("config --global --get --bool auto-worktree.fail-on-hook-error", "false")
	fakeGit.SetResponse("config --local --get auto-worktree.custom-hooks", "")
	fakeGit.SetResponse("config --global --get auto-worktree.custom-hooks", "")
	fakeGit.SetResponse("rev-parse --git-common-dir", ".git")
	fakeGit.SetResponse("rev-parse HEAD", "abc123")

	config := NewConfigWithExecutor("/test/repo", fakeGit)

	fakeHook := NewFakeHookExecutor()
	fakeHook.IsExecutableFunc = func(path string) bool {
		// Normalize path for cross-platform comparison
		expectedPath := filepath.FromSlash("/test/repo/.git/hooks/post-checkout")
		return path == expectedPath ||
			path == expectedPath+".bat" ||
			path == expectedPath+".cmd" ||
			path == expectedPath+".exe" ||
			path == expectedPath+".ps1"
	}

	output := &bytes.Buffer{}
	hm := NewHookManager("/test/repo", config, fakeGit, fakeHook, output)

	// Execute hooks with a specific worktree path
	worktreePath := "/test/repo/worktrees/test-branch"
	err := hm.ExecuteWorktreeHooks(worktreePath)
	if err != nil {
		t.Fatalf("ExecuteWorktreeHooks() error = %v", err)
	}

	// Verify hooks were executed
	if len(fakeHook.ExecutedHooks) == 0 {
		t.Fatal("No hooks executed")
	}

	// Verify working directory is set to the worktree path
	for i, hook := range fakeHook.ExecutedHooks {
		if hook.WorkingDir != worktreePath {
			t.Errorf("Hook %d: expected working directory %s, got %s", i, worktreePath, hook.WorkingDir)
		}
	}
}

func TestHookManager_PathEnvironment(t *testing.T) {
	// Setup fake executors
	fakeGit := NewFakeGitExecutor()
	fakeGit.SetResponse("config --local --get --bool auto-worktree.run-hooks", "true")
	fakeGit.SetResponse("config --global --get --bool auto-worktree.run-hooks", "true")
	fakeGit.SetResponse("config --local --get --bool auto-worktree.fail-on-hook-error", "false")
	fakeGit.SetResponse("config --global --get --bool auto-worktree.fail-on-hook-error", "false")
	fakeGit.SetResponse("config --local --get auto-worktree.custom-hooks", "")
	fakeGit.SetResponse("config --global --get auto-worktree.custom-hooks", "")
	fakeGit.SetResponse("rev-parse --git-common-dir", ".git")
	fakeGit.SetResponse("rev-parse HEAD", "abc123")

	config := NewConfigWithExecutor("/test/repo", fakeGit)

	fakeHook := NewFakeHookExecutor()
	fakeHook.IsExecutableFunc = func(path string) bool {
		// Normalize path for cross-platform comparison
		expectedPath := filepath.FromSlash("/test/repo/.git/hooks/post-checkout")
		// Check exact match or with Windows extensions
		return path == expectedPath ||
			path == expectedPath+".bat" ||
			path == expectedPath+".cmd" ||
			path == expectedPath+".exe" ||
			path == expectedPath+".ps1"
	}

	output := &bytes.Buffer{}
	hm := NewHookManager("/test/repo", config, fakeGit, fakeHook, output)

	// Execute hooks
	err := hm.ExecuteWorktreeHooks("/test/repo/worktrees/test")
	if err != nil {
		t.Fatalf("ExecuteWorktreeHooks() error = %v", err)
	}

	// Verify PATH environment contains Homebrew directories
	if len(fakeHook.ExecutedHooks) == 0 {
		t.Fatal("No hooks executed")
	}

	env := fakeHook.ExecutedHooks[0].Env
	pathFound := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			// Verify Homebrew paths are included
			if !strings.Contains(e, "/opt/homebrew/bin") {
				t.Errorf("PATH missing /opt/homebrew/bin: %s", e)
			}
			if !strings.Contains(e, "/usr/local/bin") {
				t.Errorf("PATH missing /usr/local/bin: %s", e)
			}
		}
	}

	if !pathFound {
		t.Error("PATH environment variable not found")
	}
}
