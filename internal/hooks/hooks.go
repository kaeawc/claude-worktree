// Package hooks executes git hooks in worktrees
package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// Runner executes git hooks in a worktree
type Runner struct {
	worktreePath string
	config       *git.Config
	executor     git.GitExecutor
}

// NewRunner creates a new hook runner
func NewRunner(worktreePath string, config *git.Config) *Runner {
	return &Runner{
		worktreePath: worktreePath,
		config:       config,
		executor:     git.NewGitExecutor(),
	}
}

// Run executes post-worktree hooks
// Note: post-checkout is automatically run by git during worktree creation
func (r *Runner) Run() error {
	// Check if hooks are enabled (default: true)
	if !r.config.GetRunHooks() {
		return nil
	}

	// Get failure handling preference (default: false = warn only)
	failOnError := r.config.GetFailOnHookError()

	// Find hook directories
	hookPaths := r.findHookPaths()

	if len(hookPaths) == 0 {
		// No hooks found, skip silently
		return nil
	}

	// Get list of hooks to run
	hooksToRun := r.getHooksToRun()

	// Execute each hook in order
	return r.executeHooks(hooksToRun, hookPaths, failOnError)
}

// getHooksToRun returns the list of hooks to execute
func (r *Runner) getHooksToRun() []string {
	// Hooks to run (post-checkout is already run by git automatically)
	hooksToRun := []string{"post-clone", "post-worktree"}

	// Add custom hooks from config
	customHooks := r.config.GetWithDefault(git.ConfigCustomHooks, "", git.ConfigScopeAuto)
	if customHooks != "" {
		hooksToRun = append(hooksToRun, r.parseCustomHooks(customHooks)...)
	}

	return hooksToRun
}

// parseCustomHooks parses the custom hooks configuration string
func (r *Runner) parseCustomHooks(customHooks string) []string {
	var parsed []string

	// Split by comma or space
	for _, hook := range strings.FieldsFunc(customHooks, func(c rune) bool {
		return c == ',' || c == ' '
	}) {
		hook = strings.TrimSpace(hook)
		if hook != "" {
			parsed = append(parsed, hook)
		}
	}

	return parsed
}

// executeHooks runs each hook across all hook directories
func (r *Runner) executeHooks(hooksToRun []string, hookPaths []string, failOnError bool) error {
	for _, hookName := range hooksToRun {
		if err := r.executeHookInPaths(hookName, hookPaths, failOnError); err != nil {
			return err
		}
	}

	return nil
}

// executeHookInPaths tries to execute a hook in each hook directory
func (r *Runner) executeHookInPaths(hookName string, hookPaths []string, failOnError bool) error {
	for _, hookDir := range hookPaths {
		hookPath := filepath.Join(hookDir, hookName)

		result := r.executeHook(hookPath)

		switch result {
		case hookSuccess:
			return nil // Don't run same hook from other directories
		case hookFailed:
			return r.handleHookFailure(hookName, failOnError)
		}
		// hookNotFound: try next directory
	}

	return nil
}

// handleHookFailure handles a failed hook based on configuration
func (r *Runner) handleHookFailure(hookName string, failOnError bool) error {
	fmt.Printf("\n✗ Hook %s failed\n", hookName)

	if failOnError {
		fmt.Println("To continue despite hook failures, run:")
		fmt.Println("  git config auto-worktree.fail-on-hook-error false")

		return fmt.Errorf("hook %s failed", hookName)
	}

	fmt.Println("⚠ Continuing despite hook failure (auto-worktree.fail-on-hook-error=false)")
	fmt.Println("  To fail on hook errors, run: git config auto-worktree.fail-on-hook-error true")

	return nil
}

// hookResult represents the result of hook execution
type hookResult int

const (
	hookNotFound hookResult = iota
	hookSuccess
	hookFailed
)

// executeHook executes a single hook if it exists and is executable
func (r *Runner) executeHook(hookPath string) hookResult {
	// Check if hook exists and is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		return hookNotFound
	}

	if !isExecutable(info) {
		return hookNotFound
	}

	hookName := filepath.Base(hookPath)
	fmt.Printf("\nRunning git hook: %s\n", hookName)

	// Prepare hook parameters (standard git hook format for post-checkout)
	// <prev-head> <new-head> <branch-flag>
	prevHead := "0000000000000000000000000000000000000000" // All zeros for new branch
	newHead := r.getCurrentHead()
	branchFlag := "1" // 1 = branch checkout

	// Execute hook
	cmd := exec.CommandContext(context.Background(), hookPath, prevHead, newHead, branchFlag)
	cmd.Dir = r.worktreePath
	cmd.Env = r.prepareHookEnvironment()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return hookFailed
	}

	fmt.Printf("✓ Hook %s completed successfully\n", hookName)

	return hookSuccess
}

// findHookPaths finds all possible hook directories
func (r *Runner) findHookPaths() []string {
	var hookPaths []string

	// 1. Check custom git config core.hooksPath
	customPath, err := r.executor.ExecuteInDir(r.worktreePath, "config", "core.hooksPath")
	if err == nil && customPath != "" {
		// Handle both absolute and relative paths
		if filepath.IsAbs(customPath) {
			hookPaths = append(hookPaths, customPath)
		} else {
			hookPaths = append(hookPaths, filepath.Join(r.worktreePath, customPath))
		}
	}

	// 2. Check .husky directory (popular Node.js hook manager)
	huskyPath := filepath.Join(r.worktreePath, ".husky")
	if dirExists(huskyPath) {
		hookPaths = append(hookPaths, huskyPath)
	}

	// 3. Standard .git/hooks directory (use --git-common-dir for worktrees)
	commonDir, err := r.executor.ExecuteInDir(r.worktreePath, "rev-parse", "--git-common-dir")
	if err == nil && commonDir != "" {
		hooksDir := filepath.Join(commonDir, "hooks")
		if dirExists(hooksDir) {
			hookPaths = append(hookPaths, hooksDir)
		}
	}

	return hookPaths
}

// prepareHookEnvironment prepares the environment for hook execution
func (r *Runner) prepareHookEnvironment() []string {
	env := os.Environ()

	// Ensure PATH includes common directories
	currentPath := os.Getenv("PATH")
	additionalPaths := "/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

	// Combine current PATH with additional paths
	newPath := currentPath + ":" + additionalPaths

	// Replace PATH in environment
	var newEnv []string
	pathSet := false

	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			newEnv = append(newEnv, "PATH="+newPath)
			pathSet = true
		} else {
			newEnv = append(newEnv, e)
		}
	}

	// If PATH wasn't in the environment, add it
	if !pathSet {
		newEnv = append(newEnv, "PATH="+newPath)
	}

	return newEnv
}

// getCurrentHead gets the current HEAD commit
func (r *Runner) getCurrentHead() string {
	head, err := r.executor.ExecuteInDir(r.worktreePath, "rev-parse", "HEAD")
	if err != nil {
		return "HEAD"
	}

	return head
}

// isExecutable checks if a file is executable
func isExecutable(info os.FileInfo) bool {
	return info.Mode()&0111 != 0
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
