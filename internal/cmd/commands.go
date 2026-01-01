// Package cmd provides command implementations for the auto-worktree CLI.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaeawc/auto-worktree/internal/git"
	"github.com/kaeawc/auto-worktree/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// RunInteractiveMenu displays the main interactive menu.
func RunInteractiveMenu() error {
	items := []ui.MenuItem{
		ui.NewMenuItem("New Worktree", "Create a new worktree with a new branch", "new"),
		ui.NewMenuItem("Resume Worktree", "Resume working on the last worktree", "resume"),
		ui.NewMenuItem("Work on Issue", "Create worktree for a GitHub/GitLab/JIRA issue", "issue"),
		ui.NewMenuItem("Create Issue", "Create a new issue and start working on it", "create"),
		ui.NewMenuItem("Review PR", "Review a pull request in a new worktree", "pr"),
		ui.NewMenuItem("List Worktrees", "Show all existing worktrees", "list"),
		ui.NewMenuItem("Cleanup Worktrees", "Interactive cleanup of merged/stale worktrees", "cleanup"),
		ui.NewMenuItem("Settings", "Configure per-repository settings", "settings"),
	}

	menu := ui.NewMenu("auto-worktree", items)
	p := tea.NewProgram(menu)

	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run menu: %w", err)
	}

	finalModel, ok := m.(ui.MenuModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	choice := finalModel.Choice()

	if choice == "" {
		return nil
	}

	// Route to the appropriate command handler
	return routeMenuChoice(choice)
}

func routeMenuChoice(choice string) error {
	switch choice {
	case "new":
		return RunNew()
	case "resume":
		return RunResume()
	case "issue":
		return RunIssue("")
	case "create":
		return RunCreate()
	case "pr":
		return RunPR("")
	case "list":
		return RunList()
	case "cleanup":
		return RunCleanup()
	case "settings":
		return RunSettings()
	default:
		return fmt.Errorf("unknown command: %s", choice)
	}
}

// RunList lists all worktrees.
func RunList() error {
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	worktrees, err := repo.ListWorktrees()
	if err != nil {
		return fmt.Errorf("error listing worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	fmt.Printf("Repository: %s\n", repo.SourceFolder)
	fmt.Printf("Worktree base: %s\n\n", repo.WorktreeBase)
	fmt.Printf("%-50s %-25s %-15s %s\n", "PATH", "BRANCH", "AGE", "UNPUSHED")
	fmt.Println(strings.Repeat("-", 110))

	for _, wt := range worktrees {
		path := wt.Path
		branch := wt.Branch

		if branch == "" {
			branch = fmt.Sprintf("(detached @ %s)", wt.HEAD[:7])
		}

		age := formatAge(wt.Age())
		unpushed := ""

		if wt.UnpushedCount > 0 {
			unpushed = fmt.Sprintf("%d commits", wt.UnpushedCount)
		} else if !wt.IsDetached {
			unpushed = "up to date"
		}

		// Truncate path if too long
		if len(path) > 48 {
			path = "..." + path[len(path)-45:]
		}

		fmt.Printf("%-50s %-25s %-15s %s\n", path, branch, age, unpushed)
	}

	fmt.Printf("\nTotal: %d worktree(s)\n", len(worktrees))

	return nil
}

// RunNew creates a new worktree.
func RunNew() error {
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	branchName, useExisting, err := getBranchInput()
	if err != nil {
		return err
	}

	// Sanitize branch name
	sanitizedName := git.SanitizeBranchName(branchName)

	// Check if worktree already exists for this branch
	if err := checkExistingWorktree(repo, branchName); err != nil {
		return err
	}

	// Construct worktree path
	worktreePath := filepath.Join(repo.WorktreeBase, sanitizedName)

	if err := createWorktree(repo, worktreePath, branchName, useExisting); err != nil {
		return err
	}

	fmt.Printf("✓ Worktree created at: %s\n", worktreePath)
	fmt.Printf("\nTo start working:\n")
	fmt.Printf("  cd %s\n", worktreePath)

	return nil
}

func getBranchInput() (branchName string, useExisting bool, err error) {
	if len(os.Args) > 2 {
		// Command line argument provided
		arg := os.Args[2]
		if arg == "--existing" {
			if len(os.Args) < 4 {
				return "", false, fmt.Errorf("branch name required after --existing")
			}

			return os.Args[3], true, nil
		}

		return arg, false, nil
	}

	// Interactive mode
	input := ui.NewInput("Enter branch name:", "feature/my-feature or leave empty for random name")
	p := tea.NewProgram(input)

	m, err := p.Run()
	if err != nil {
		return "", false, fmt.Errorf("failed to get input: %w", err)
	}

	finalModel, ok := m.(ui.InputModel)
	if !ok {
		return "", false, fmt.Errorf("unexpected model type")
	}

	if finalModel.Err() != nil {
		return "", false, finalModel.Err()
	}

	branchName = finalModel.Value()
	if branchName == "" {
		// TODO: Generate random branch name
		return "", false, fmt.Errorf("random branch names not yet implemented - please provide a branch name")
	}

	return branchName, false, nil
}

func checkExistingWorktree(repo *git.Repository, branchName string) error {
	existingWt, err := repo.GetWorktreeForBranch(branchName)
	if err != nil {
		return fmt.Errorf("error checking for existing worktree: %w", err)
	}

	if existingWt != nil {
		return fmt.Errorf("worktree already exists for branch %s at %s", branchName, existingWt.Path)
	}

	return nil
}

func createWorktree(repo *git.Repository, worktreePath, branchName string, useExisting bool) error {
	if useExisting {
		// Check if branch exists
		if !repo.BranchExists(branchName) {
			return fmt.Errorf("branch %s does not exist", branchName)
		}

		fmt.Printf("Creating worktree for existing branch: %s\n", branchName)

		return repo.CreateWorktree(worktreePath, branchName)
	}

	// Check if branch already exists
	if repo.BranchExists(branchName) {
		return fmt.Errorf("branch %s already exists. Use --existing flag to create worktree for it", branchName)
	}

	// Get default branch as base
	defaultBranch, err := repo.GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("error getting default branch: %w", err)
	}

	fmt.Printf("Creating worktree with new branch: %s (from %s)\n", branchName, defaultBranch)

	return repo.CreateWorktreeWithNewBranch(worktreePath, branchName, defaultBranch)
}

// RunResume resumes the last worktree.
func RunResume() error {
	// TODO: Implement resume logic
	return fmt.Errorf("'resume' command not yet implemented")
}

// RunIssue works on an issue.
func RunIssue(_ string) error {
	// TODO: Implement issue workflow
	return fmt.Errorf("'issue' command not yet implemented")
}

// RunCreate creates a new issue.
func RunCreate() error {
	// TODO: Implement issue creation
	return fmt.Errorf("'create' command not yet implemented")
}

// RunPR reviews a pull request.
func RunPR(_ string) error {
	// TODO: Implement PR review workflow
	return fmt.Errorf("'pr' command not yet implemented")
}

// RunCleanup performs interactive cleanup.
func RunCleanup() error {
	// TODO: Implement cleanup workflow
	return fmt.Errorf("'cleanup' command not yet implemented")
}

// RunSettings shows settings menu.
func RunSettings() error {
	// TODO: Implement settings menu
	return fmt.Errorf("'settings' command not yet implemented")
}

// RunRemove removes a worktree.
func RunRemove(path string) error {
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr == nil {
			path = filepath.Join(homeDir, path[1:])
		}
	}

	// Make absolute path
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("error resolving path: %w", err)
		}
	}

	fmt.Printf("Removing worktree: %s\n", path)

	err = repo.RemoveWorktree(path)
	if err != nil {
		return fmt.Errorf("error removing worktree: %w", err)
	}

	fmt.Printf("✓ Worktree removed\n")

	return nil
}

// RunPrune prunes orphaned worktrees.
func RunPrune() error {
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	fmt.Println("Pruning orphaned worktrees...")

	err = repo.PruneWorktrees()
	if err != nil {
		return fmt.Errorf("error pruning worktrees: %w", err)
	}

	fmt.Println("✓ Pruned orphaned worktrees")

	return nil
}

// formatAge formats a duration into a human-readable string.
func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}
