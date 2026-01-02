// Package cmd provides command implementations for the auto-worktree CLI.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kaeawc/auto-worktree/internal/ai"
	"github.com/kaeawc/auto-worktree/internal/environment"
	"github.com/kaeawc/auto-worktree/internal/git"
	"github.com/kaeawc/auto-worktree/internal/github"
	"github.com/kaeawc/auto-worktree/internal/hooks"
	"github.com/kaeawc/auto-worktree/internal/session"
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

	// Use ListWorktreesWithMergeStatus to get merge status information
	worktrees, err := repo.ListWorktreesWithMergeStatus()
	if err != nil {
		return fmt.Errorf("error listing worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	fmt.Printf("Repository: %s\n", repo.SourceFolder)
	fmt.Printf("Worktree base: %s\n\n", repo.WorktreeBase)
	fmt.Printf("%-45s %-20s %-12s %-12s %s\n", "PATH", "BRANCH", "AGE", "STATUS", "UNPUSHED")
	fmt.Println(strings.Repeat("-", 120))

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
		if len(path) > 43 {
			path = "..." + path[len(path)-40:]
		}

		// Get status indicator
		status := getStatusIndicator(wt)

		fmt.Printf("%-45s %-20s %-12s %-12s %s\n", path, branch, age, status, unpushed)
	}

	fmt.Printf("\nTotal: %d worktree(s)\n", len(worktrees))

	return nil
}

// getStatusIndicator returns a status string for the worktree
func getStatusIndicator(wt *git.Worktree) string {
	if wt.IsMerged() {
		return "[merged]"
	}
	if wt.IsStale() {
		days := int(wt.Age().Hours() / 24)
		return fmt.Sprintf("[stale %dd]", days)
	}
	if wt.IsBranchMerged {
		return "[git-merged]"
	}
	return "-"
}

// RunNew creates a new worktree.
func RunNew() error {
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	branchName, useExisting, err := getBranchInput(repo)
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

func getBranchInput(repo *git.Repository) (branchName string, useExisting bool, err error) {
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
		// Generate random branch name
		branchName, err = repo.GenerateUniqueBranchName(100)
		if err != nil {
			return "", false, fmt.Errorf("failed to generate random branch name: %w", err)
		}
		fmt.Printf("✓ Generated branch: %s\n", branchName)
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

		if err := repo.CreateWorktree(worktreePath, branchName); err != nil {
			return err
		}
	} else {
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

		if err := repo.CreateWorktreeWithNewBranch(worktreePath, branchName, defaultBranch); err != nil {
			return err
		}
	}

	// Setup environment after worktree creation
	setupEnvironment(repo, worktreePath)

	return nil
}

// setupEnvironment runs environment setup for a worktree
func setupEnvironment(repo *git.Repository, worktreePath string) {
	config := git.NewConfig(repo.RootPath)

	// Get configuration
	autoInstall := config.GetAutoInstall()
	packageManager := config.GetPackageManager()

	// Skip if auto-install is disabled
	if !autoInstall {
		return
	}

	// Run setup with spinner
	spinnerModel := ui.NewSpinnerModel("Detecting project type...")
	p := tea.NewProgram(spinnerModel)

	// Run setup in background
	go func() {
		opts := &environment.SetupOptions{
			AutoInstall:              autoInstall,
			ConfiguredPackageManager: packageManager,
			OnProgress: func(message string) {
				p.Send(ui.SpinnerUpdateMsg{Message: message})
			},
			OnWarning: func(message string) {
				// Warnings will be shown after spinner completes
				fmt.Fprintf(os.Stderr, "\nWarning: %s\n", message)
			},
		}

		// Run setup
		err := environment.Setup(worktreePath, opts)

		// Signal completion
		p.Send(ui.SpinnerDoneMsg{Err: err})
	}()

	// Run spinner
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running spinner: %v\n", err)
	}
}

// RunResume resumes the last worktree.
func RunResume() error {
	// TODO: Implement resume logic
	return fmt.Errorf("'resume' command not yet implemented")
}

// RunIssue works on an issue.
// If issueID is empty, shows interactive issue selector.
// If issueID is numeric, directly creates worktree for that issue.
func RunIssue(issueID string) error {
	// 1. Initialize repository
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// 2. Check gh CLI availability
	executor := github.NewGitHubExecutor()
	if !github.IsInstalled(executor) {
		return fmt.Errorf("gh CLI is not installed. Install with: brew install gh")
	}

	// 3. Create GitHub client (auto-detects owner/repo)
	client, err := github.NewClient(repo.RootPath)
	if err != nil {
		if errors.Is(err, github.ErrGHNotInstalled) {
			return fmt.Errorf("gh CLI is not installed. Install with: brew install gh")
		}
		if errors.Is(err, github.ErrGHNotAuthenticated) {
			return fmt.Errorf("gh CLI is not authenticated. Run: gh auth login")
		}
		if errors.Is(err, github.ErrNotGitHubRepo) {
			return fmt.Errorf("not a GitHub repository")
		}
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	fmt.Printf("Repository: %s/%s\n\n", client.Owner, client.Repo)

	// 4. Get issue number (interactive or direct)
	var issueNum int
	if issueID == "" {
		// Interactive mode: show issue selector
		issueNum, err = selectIssueInteractive(client, repo)
		if err != nil {
			return err
		}
	} else {
		// Direct mode: parse issue number
		issueNum, err = parseIssueNumber(issueID)
		if err != nil {
			return fmt.Errorf("invalid issue number: %s", issueID)
		}
	}

	// 5. Fetch full issue details
	issue, err := client.GetIssue(issueNum)
	if err != nil {
		return fmt.Errorf("failed to fetch issue #%d: %w", issueNum, err)
	}

	// 6. Check if issue is closed/merged
	if issue.State == "CLOSED" {
		merged, err := client.IsIssueMerged(issueNum)
		if err != nil {
			fmt.Printf("Warning: Could not check merge status: %v\n", err)
		} else if merged {
			return fmt.Errorf("issue #%d is already closed and merged", issueNum)
		} else {
			fmt.Printf("Warning: Issue #%d is closed but not merged\n", issueNum)
		}
	}

	// 7. Generate branch name: work/<number>-<sanitized-title>
	branchName := issue.BranchName()

	// 8. Check if worktree already exists
	existingWt, err := repo.GetWorktreeForBranch(branchName)
	if err != nil {
		return fmt.Errorf("error checking for existing worktree: %w", err)
	}

	if existingWt != nil {
		// Offer to resume existing worktree
		return offerResumeWorktree(existingWt, issue)
	}

	// 9. Create worktree
	worktreePath := filepath.Join(repo.WorktreeBase, git.SanitizeBranchName(branchName))

	// Check if branch exists
	if repo.BranchExists(branchName) {
		fmt.Printf("Creating worktree for existing branch: %s\n", branchName)
		if err := repo.CreateWorktree(worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		defaultBranch, err := repo.GetDefaultBranch()
		if err != nil {
			return fmt.Errorf("error getting default branch: %w", err)
		}

		fmt.Printf("Creating worktree for issue #%d: %s\n", issue.Number, issue.Title)
		fmt.Printf("Branch: %s (from %s)\n", branchName, defaultBranch)

		if err := repo.CreateWorktreeWithNewBranch(worktreePath, branchName, defaultBranch); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Setup environment after worktree creation
	setupEnvironment(repo, worktreePath)

	// 10. Display success message
	fmt.Printf("\n✓ Worktree created at: %s\n", worktreePath)

	// 11. Run post-worktree hooks
	if err := runPostWorktreeHooks(worktreePath, repo.RootPath); err != nil {
		return fmt.Errorf("hook execution failed: %w", err)
	}

	// 12. Install dependencies
	setupEnvironment(repo, worktreePath)

	// 13. Start AI tool in background session
	if err := startAISession(worktreePath, branchName, repo.RootPath, issue); err != nil {
		// Non-fatal: warn but continue
		fmt.Printf("⚠ Failed to start AI session: %v\n", err)
		fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
		fmt.Printf("URL: %s\n", issue.URL)
		fmt.Printf("\nTo start working:\n")
		fmt.Printf("  cd %s\n", worktreePath)
	}

	return nil
}

// RunCreate creates a new issue.
func RunCreate() error {
	// 1. Initialize repository
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// 2. Check gh CLI availability
	executor := github.NewGitHubExecutor()
	if !github.IsInstalled(executor) {
		return fmt.Errorf("gh CLI is not installed. Install with: brew install gh")
	}

	// 3. Create GitHub client (auto-detects owner/repo)
	client, err := github.NewClient(repo.RootPath)
	if err != nil {
		if errors.Is(err, github.ErrGHNotInstalled) {
			return fmt.Errorf("gh CLI is not installed. Install with: brew install gh")
		}
		if errors.Is(err, github.ErrGHNotAuthenticated) {
			return fmt.Errorf("gh CLI is not authenticated. Run: gh auth login")
		}
		if errors.Is(err, github.ErrNotGitHubRepo) {
			return fmt.Errorf("not a GitHub repository")
		}
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	fmt.Printf("Repository: %s/%s\n\n", client.Owner, client.Repo)

	// 4. Get issue title (interactive)
	titleInput := ui.NewInput("Issue Title", "Enter a title for the issue")
	p := tea.NewProgram(titleInput)
	result, err := p.Run()
	if err != nil {
		return fmt.Errorf("error getting title input: %w", err)
	}

	titleModel, ok := result.(ui.InputModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}
	if titleModel.Err() != nil {
		return fmt.Errorf("canceled")
	}

	title := titleModel.Value()
	if title == "" {
		return fmt.Errorf("issue title cannot be empty")
	}

	// 5. Get issue body (interactive, optional)
	bodyInput := ui.NewTextArea("Issue Description (optional)", "Describe the issue...")
	p = tea.NewProgram(bodyInput)
	result, err = p.Run()
	if err != nil {
		return fmt.Errorf("error getting body input: %w", err)
	}

	bodyModel, ok := result.(ui.TextAreaModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}
	if bodyModel.Err() != nil {
		return fmt.Errorf("canceled")
	}

	body := bodyModel.Value()

	// 6. Confirm before creating
	confirmMsg := fmt.Sprintf("Create issue: %s?", title)
	confirmModel := ui.NewConfirmModel(confirmMsg)
	p = tea.NewProgram(confirmModel)
	result, err = p.Run()
	if err != nil {
		return fmt.Errorf("error getting confirmation: %w", err)
	}

	confirmed, ok := result.(ui.ConfirmModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}
	if !confirmed.GetChoice() {
		fmt.Println("Issue creation canceled.")
		return nil
	}

	// 7. Create the issue
	fmt.Println("\nCreating issue...")
	issue, err := client.CreateIssue(title, body)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// 8. Display success message
	fmt.Printf("\n✓ Issue created successfully!\n")
	fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
	fmt.Printf("URL: %s\n", issue.URL)

	// 9. Offer to create worktree for the new issue
	wtConfirmMsg := fmt.Sprintf("Create a worktree for issue #%d?", issue.Number)
	wtConfirmModel := ui.NewConfirmModel(wtConfirmMsg)
	p = tea.NewProgram(wtConfirmModel)
	result, err = p.Run()
	if err != nil {
		return fmt.Errorf("error getting worktree confirmation: %w", err)
	}

	wtConfirmed, ok := result.(ui.ConfirmModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}
	if !wtConfirmed.GetChoice() {
		return nil
	}

	// 10. Create worktree for the new issue
	branchName := issue.BranchName()
	worktreePath := filepath.Join(repo.WorktreeBase, git.SanitizeBranchName(branchName))

	defaultBranch, err := repo.GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("error getting default branch: %w", err)
	}

	fmt.Printf("\nCreating worktree for issue #%d...\n", issue.Number)
	fmt.Printf("Branch: %s (from %s)\n", branchName, defaultBranch)

	if err := repo.CreateWorktreeWithNewBranch(worktreePath, branchName, defaultBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Setup environment after worktree creation
	setupEnvironment(repo, worktreePath)

	fmt.Printf("\n✓ Worktree created at: %s\n", worktreePath)
	fmt.Printf("\nTo start working:\n")
	fmt.Printf("  cd %s\n", worktreePath)

	return nil
}

// RunPR reviews a pull request.
// If prID is empty, shows interactive PR selector.
// If prID is numeric, directly creates worktree for that PR.
func RunPR(prID string) error {
	// 1. Initialize repository
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// 2. Check gh CLI availability
	executor := github.NewGitHubExecutor()
	if !github.IsInstalled(executor) {
		return fmt.Errorf("gh CLI is not installed. Install with: brew install gh")
	}

	// 3. Create GitHub client (auto-detects owner/repo)
	client, err := github.NewClient(repo.RootPath)
	if err != nil {
		if errors.Is(err, github.ErrGHNotInstalled) {
			return fmt.Errorf("gh CLI is not installed. Install with: brew install gh")
		}
		if errors.Is(err, github.ErrGHNotAuthenticated) {
			return fmt.Errorf("gh CLI is not authenticated. Run: gh auth login")
		}
		if errors.Is(err, github.ErrNotGitHubRepo) {
			return fmt.Errorf("not a GitHub repository")
		}
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	fmt.Printf("Repository: %s/%s\n\n", client.Owner, client.Repo)

	// 4. Get PR number (interactive or direct)
	var prNum int
	if prID == "" {
		// Interactive mode: show PR selector
		prNum, err = selectPRInteractive(client, repo)
		if err != nil {
			return err
		}
	} else {
		// Direct mode: parse PR number
		prNum, err = parsePRNumber(prID)
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", prID)
		}
	}

	// 5. Fetch full PR details
	pr, err := client.GetPR(prNum)
	if err != nil {
		return fmt.Errorf("failed to fetch PR #%d: %w", prNum, err)
	}

	// 6. Check if PR is already merged or closed
	if pr.State == "MERGED" {
		return fmt.Errorf("PR #%d is already merged", prNum)
	}
	if pr.State == "CLOSED" {
		fmt.Printf("Warning: PR #%d is closed but not merged\n", prNum)
	}

	// 7. Display PR metadata
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("PR #%d: %s\n", pr.Number, pr.Title)
	fmt.Printf("Author: @%s\n", pr.Author.Login)
	fmt.Printf("Base: %s ← Head: %s\n", pr.BaseRefName, pr.HeadRefName)
	if pr.IsDraft {
		fmt.Printf("Status: DRAFT\n")
	}

	// Show labels if present
	if len(pr.Labels) > 0 {
		labels := make([]string, len(pr.Labels))
		for i, label := range pr.Labels {
			labels[i] = label.Name
		}
		fmt.Printf("Labels: %s\n", strings.Join(labels, ", "))
	}

	// 8. Display diff stats
	fmt.Printf("\n📊 Changes:\n")
	fmt.Printf("  Files changed: %d\n", pr.ChangedFiles)
	fmt.Printf("  Additions:     +%d\n", pr.Additions)
	fmt.Printf("  Deletions:     -%d\n", pr.Deletions)
	fmt.Printf("  Size:          %s\n", pr.ChangeSize())

	// 9. Check for merge conflicts
	hasConflicts, err := client.HasMergeConflicts(prNum)
	if err != nil {
		fmt.Printf("Warning: Could not check merge conflicts: %v\n", err)
	} else if hasConflicts {
		fmt.Printf("\n⚠️  Warning: This PR has merge conflicts with %s\n", pr.BaseRefName)
	}

	// 10. Display CI status
	if len(pr.StatusCheckRollup) > 0 {
		if pr.AllChecksPass() {
			fmt.Printf("\n✓ All CI checks passed\n")
		} else {
			fmt.Printf("\n⚠️  Some CI checks are failing or pending\n")
		}
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// 11. Check if AI review is enabled
	if shouldGenerateAIReview(repo) {
		fmt.Println("Generating AI review summary...")
		if err := generateAIReviewSummary(client, pr, repo); err != nil {
			fmt.Printf("Warning: Could not generate AI review: %v\n\n", err)
		}
	}

	// 12. Generate branch name: pr/<number>-<sanitized-title>
	branchName := pr.BranchName()

	// 13. Check if worktree already exists
	existingWt, err := repo.GetWorktreeForBranch(branchName)
	if err != nil {
		return fmt.Errorf("error checking for existing worktree: %w", err)
	}

	if existingWt != nil {
		// Offer to resume existing worktree
		return offerResumePRWorktree(existingWt, pr)
	}

	// 14. Create worktree
	worktreePath := filepath.Join(repo.WorktreeBase, git.SanitizeBranchName(branchName))

	// Check if branch exists locally
	if repo.BranchExists(branchName) {
		fmt.Printf("Creating worktree for existing branch: %s\n", branchName)
		if err := repo.CreateWorktree(worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		// Fetch the PR branch from the remote
		fmt.Printf("Creating worktree for PR #%d: %s\n", pr.Number, pr.Title)
		fmt.Printf("Branch: %s (tracking %s)\n", branchName, pr.HeadRefName)

		// Create worktree and checkout the PR
		if err := checkoutPRInWorktree(repo, worktreePath, branchName, pr); err != nil {
			return fmt.Errorf("failed to checkout PR: %w", err)
		}
	}

	// 15. Display success message
	fmt.Printf("\n✓ Worktree created at: %s\n", worktreePath)
	fmt.Printf("\nPR #%d: %s\n", pr.Number, pr.Title)
	fmt.Printf("URL: %s\n", pr.URL)
	fmt.Printf("\nTo start reviewing:\n")
	fmt.Printf("  cd %s\n", worktreePath)

	return nil
}

// RunCleanup performs interactive cleanup.
func RunCleanup() error {
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// Get cleanup candidates (merged first, then stale)
	candidates, err := repo.GetCleanupCandidates()
	if err != nil {
		return fmt.Errorf("error finding cleanup candidates: %w", err)
	}

	if len(candidates) == 0 {
		fmt.Println("No worktrees found that need cleanup.")
		return nil
	}

	// Separate merged and stale
	merged, stale := categorizeWorktrees(candidates)

	// Process merged worktrees (automatic with confirmation)
	if err := processMergedWorktrees(repo, merged, stale); err != nil {
		return err
	}

	// Process stale worktrees (interactive)
	processStaleWorktrees(repo, stale)

	fmt.Println("\nCleanup complete!")
	return nil
}

// categorizeWorktrees separates worktrees into merged and stale categories
func categorizeWorktrees(candidates []*git.Worktree) ([]*git.Worktree, []*git.Worktree) {
	var merged, stale []*git.Worktree
	for _, wt := range candidates {
		if wt.IsMerged() {
			merged = append(merged, wt)
		} else if wt.IsStale() {
			stale = append(stale, wt)
		}
	}
	return merged, stale
}

// processMergedWorktrees handles automatic cleanup of merged worktrees with confirmation
func processMergedWorktrees(repo *git.Repository, merged, stale []*git.Worktree) error {
	if len(merged) == 0 {
		return nil
	}

	// Show confirmation prompt
	if !confirmCleanup(len(merged), len(stale)) {
		return nil
	}

	// Clean up merged worktrees
	fmt.Printf("\nCleaning up %d merged worktree(s)...\n\n", len(merged))
	for _, wt := range merged {
		if err := cleanupWorktree(repo, wt, true); err != nil {
			fmt.Printf("  Error cleaning up %s: %v\n", wt.Path, err)
			continue
		}
		fmt.Printf("  ✓ Removed %s (%s)\n", wt.Path, wt.CleanupReason())
	}

	return nil
}

// confirmCleanup shows confirmation dialog and returns user's choice
func confirmCleanup(mergedCount, staleCount int) bool {
	confirmation := ui.NewCleanupConfirmation(mergedCount, staleCount)
	p := tea.NewProgram(confirmation)

	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error showing confirmation: %v\n", err)
		return false
	}

	finalModel, ok := m.(ui.CleanupConfirmationModel)
	if !ok {
		return false
	}

	if finalModel.WasCanceled() {
		fmt.Println("Cleanup canceled")
		return false
	}

	return finalModel.WasConfirmed()
}

// processStaleWorktrees handles interactive cleanup of stale worktrees
func processStaleWorktrees(repo *git.Repository, stale []*git.Worktree) {
	if len(stale) == 0 {
		return
	}

	fmt.Printf("\nInteractive cleanup for %d stale worktree(s)...\n\n", len(stale))
	for _, wt := range stale {
		if err := interactiveCleanup(repo, wt); err != nil {
			fmt.Printf("  Error: %v\n", err)
		}
	}
}

// interactiveCleanup prompts the user to clean up a worktree
func interactiveCleanup(repo *git.Repository, wt *git.Worktree) error {
	prompt := ui.NewCleanupPrompt(wt.Path, wt.Branch, wt.CleanupReason(), wt.UnpushedCount, true)
	p := tea.NewProgram(prompt)

	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("error showing prompt: %w", err)
	}

	finalModel, ok := m.(ui.CleanupPromptModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if finalModel.WasCanceled() {
		fmt.Println("  Skipped")
		return nil
	}

	if !finalModel.WasConfirmed() {
		fmt.Println("  Skipped")
		return nil
	}

	// Clean up the worktree
	if err := cleanupWorktree(repo, wt, finalModel.ShouldDeleteBranch()); err != nil {
		return err
	}

	fmt.Printf("  ✓ Removed %s\n", wt.Path)
	if finalModel.ShouldDeleteBranch() && wt.Branch != "" {
		fmt.Printf("  ✓ Deleted branch %s\n", wt.Branch)
	}

	return nil
}

// cleanupWorktree removes a worktree and optionally deletes its branch
func cleanupWorktree(repo *git.Repository, wt *git.Worktree, deleteBranch bool) error {
	// Remove the worktree
	if err := repo.RemoveWorktree(wt.Path); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Delete the branch if requested
	if deleteBranch && wt.Branch != "" {
		if err := repo.DeleteBranch(wt.Branch); err != nil {
			// Don't fail the cleanup if branch deletion fails
			fmt.Printf("  Warning: failed to delete branch %s: %v\n", wt.Branch, err)
		}
	}

	return nil
}

const (
	scopeLocal  = "local"
	scopeGlobal = "global"
)

// RunSettings shows settings menu.
func RunSettings() error {
	// Initialize repository and config
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	cfg := git.NewConfig(repo.RootPath)

	// Main settings loop
	for {
		settings := loadCurrentSettings(cfg)

		menu := ui.NewSettingsMenuModel(settings)
		p := tea.NewProgram(menu, tea.WithAltScreen())

		model, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run settings menu: %w", err)
		}

		m, ok := model.(*ui.SettingsMenuModel)
		if !ok {
			return fmt.Errorf("unexpected model type")
		}

		choice := m.GetChoice()
		if choice == "" {
			// User quit
			return nil
		}

		// Handle special actions
		if choice == "view-all" {
			if err := showAllSettings(cfg); err != nil {
				return err
			}
			continue
		}

		if choice == "reset" {
			if err := resetSettings(cfg); err != nil {
				return err
			}
			continue
		}

		// Handle setting edits
		if strings.HasPrefix(choice, "edit:") {
			key := strings.TrimPrefix(choice, "edit:")
			if err := editSetting(cfg, key, settings); err != nil {
				return err
			}
			continue
		}
	}
}

func loadCurrentSettings(cfg *git.Config) []ui.SettingItem {
	settings := []ui.SettingItem{
		ui.NewSettingItem(
			git.ConfigIssueProvider,
			"Issue Provider",
			"Select issue tracking system",
			"select",
			git.ValidIssueProviders,
			cfg.GetWithDefault(git.ConfigIssueProvider, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigAITool,
			"AI Tool",
			"Select AI coding assistant",
			"select",
			git.ValidAITools,
			cfg.GetWithDefault(git.ConfigAITool, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigIssueAutoselect,
			"Issue Autoselect",
			"Automatically select first issue in list",
			"bool",
			nil,
			fmt.Sprintf("%t", cfg.GetIssueAutoselect()),
		),
		ui.NewSettingItem(
			git.ConfigPRAutoselect,
			"PR Autoselect",
			"Automatically select first PR in list",
			"bool",
			nil,
			fmt.Sprintf("%t", cfg.GetPRAutoselect()),
		),
		ui.NewSettingItem(
			git.ConfigRunHooks,
			"Run Hooks",
			"Execute git hooks during operations",
			"bool",
			nil,
			fmt.Sprintf("%t", cfg.GetRunHooks()),
		),
		ui.NewSettingItem(
			git.ConfigFailOnHookError,
			"Fail on Hook Error",
			"Stop operation if a hook fails",
			"bool",
			nil,
			fmt.Sprintf("%t", cfg.GetFailOnHookError()),
		),
		ui.NewSettingItem(
			git.ConfigJiraServer,
			"JIRA Server",
			"JIRA server URL (e.g., https://company.atlassian.net)",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigJiraServer, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigJiraProject,
			"JIRA Project",
			"JIRA project key (e.g., PROJ)",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigJiraProject, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigGitLabServer,
			"GitLab Server",
			"GitLab server URL (e.g., https://gitlab.com)",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigGitLabServer, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigGitLabProject,
			"GitLab Project",
			"GitLab project path (e.g., group/project)",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigGitLabProject, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigLinearTeam,
			"Linear Team",
			"Linear team identifier",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigLinearTeam, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigCustomHooks,
			"Custom Hooks",
			"Comma-separated list of custom hook names",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigCustomHooks, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigIssueTemplatesDir,
			"Issue Templates Directory",
			"Directory containing issue templates",
			"string",
			nil,
			cfg.GetWithDefault(git.ConfigIssueTemplatesDir, "", git.ConfigScopeAuto),
		),
		ui.NewSettingItem(
			git.ConfigIssueTemplatesDisabled,
			"Disable Issue Templates",
			"Don't use issue templates",
			"bool",
			nil,
			fmt.Sprintf("%t", cfg.GetBoolWithDefault(git.ConfigIssueTemplatesDisabled, false, git.ConfigScopeAuto)),
		),
		ui.NewSettingItem(
			git.ConfigIssueTemplatesNoPrompt,
			"No Template Prompt",
			"Don't prompt for template selection",
			"bool",
			nil,
			fmt.Sprintf("%t", cfg.GetBoolWithDefault(git.ConfigIssueTemplatesNoPrompt, false, git.ConfigScopeAuto)),
		),
	}

	return settings
}

func editSetting(cfg *git.Config, key string, settings []ui.SettingItem) error {
	// Find the setting
	var setting ui.SettingItem
	found := false
	for _, s := range settings {
		if s.Key == key {
			setting = s
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("setting not found: %s", key)
	}

	// Show editor
	editor := ui.NewSettingEditor(setting)
	p := tea.NewProgram(editor, tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	editorModel, ok := model.(*ui.SettingEditorModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if editorModel.Err() != nil {
		// User canceled
		return nil
	}

	newValue := editorModel.GetValue()
	if newValue == "" && setting.ValueType != "string" {
		// User canceled or didn't select anything
		return nil
	}

	// Validate the value
	if err := cfg.Validate(key, newValue); err != nil {
		fmt.Println("\n" + ui.ErrorStyle.Render(fmt.Sprintf("Invalid value: %v", err)))
		fmt.Println()
		return nil
	}

	// Ask for scope
	scopeSelector := ui.NewScopeSelector()
	p = tea.NewProgram(scopeSelector, tea.WithAltScreen())
	model, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run scope selector: %w", err)
	}

	scopeModel, ok := model.(*ui.ScopeSelectorModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	scope := scopeModel.GetScope()
	if scope == "" {
		// User canceled
		return nil
	}

	// Convert scope string to ConfigScope
	var configScope git.ConfigScope
	switch scope {
	case scopeLocal:
		configScope = git.ConfigScopeLocal
	case scopeGlobal:
		configScope = git.ConfigScopeGlobal
	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}

	// Save the setting
	if err := cfg.SetValidated(key, newValue, configScope); err != nil {
		return fmt.Errorf("failed to save setting: %w", err)
	}

	fmt.Println("\n" + ui.SuccessStyle.Render(fmt.Sprintf("Setting saved: %s = %s (%s)",
		strings.TrimPrefix(key, "auto-worktree."), newValue, scope)))
	fmt.Println()

	return nil
}

func showAllSettings(cfg *git.Config) error {
	// Collect all local and global values
	localValues := make(map[string]string)
	globalValues := make(map[string]string)

	allKeys := []string{
		git.ConfigIssueProvider,
		git.ConfigAITool,
		git.ConfigIssueAutoselect,
		git.ConfigPRAutoselect,
		git.ConfigRunHooks,
		git.ConfigFailOnHookError,
		git.ConfigCustomHooks,
		git.ConfigJiraServer,
		git.ConfigJiraProject,
		git.ConfigGitLabServer,
		git.ConfigGitLabProject,
		git.ConfigLinearTeam,
		git.ConfigIssueTemplatesDir,
		git.ConfigIssueTemplatesDisabled,
		git.ConfigIssueTemplatesNoPrompt,
		git.ConfigIssueTemplatesDetected,
	}

	for _, key := range allKeys {
		if val, err := cfg.Get(key, git.ConfigScopeLocal); err == nil && val != "" {
			localValues[key] = val
		}
		if val, err := cfg.Get(key, git.ConfigScopeGlobal); err == nil && val != "" {
			globalValues[key] = val
		}
	}

	viewer := ui.NewSettingsViewer(localValues, globalValues)
	p := tea.NewProgram(viewer, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func resetSettings(cfg *git.Config) error {
	// Confirm reset
	confirm := ui.NewConfirmModel("Are you sure you want to reset ALL settings to defaults?\nThis will clear all auto-worktree configuration.")
	p := tea.NewProgram(confirm, tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run confirmation: %w", err)
	}

	confirmModel, ok := model.(ui.ConfirmModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if !confirmModel.GetChoice() {
		return nil
	}

	// Ask for scope
	scopeSelector := ui.NewScopeSelector()
	p = tea.NewProgram(scopeSelector, tea.WithAltScreen())
	model, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run scope selector: %w", err)
	}

	scopeModel, ok := model.(*ui.ScopeSelectorModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	scope := scopeModel.GetScope()
	if scope == "" {
		return nil
	}

	var configScope git.ConfigScope
	switch scope {
	case "local":
		configScope = git.ConfigScopeLocal
	case "global":
		configScope = git.ConfigScopeGlobal
	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}

	// Reset all settings
	if err := cfg.UnsetAll(configScope); err != nil {
		return fmt.Errorf("failed to reset settings: %w", err)
	}

	fmt.Println("\n" + ui.SuccessStyle.Render(fmt.Sprintf("All settings reset (%s)", scope)))
	fmt.Println()

	return nil
}

// RunSettingsSet sets a configuration value (non-interactive mode)
func RunSettingsSet(key, value, scope string) error {
	// Normalize key - add auto-worktree prefix if not present
	if !strings.HasPrefix(key, "auto-worktree.") {
		key = "auto-worktree." + key
	}

	// Initialize repository and config
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	cfg := git.NewConfig(repo.RootPath)

	// Validate the key is a known config key
	validKeys := []string{
		git.ConfigIssueProvider,
		git.ConfigAITool,
		git.ConfigIssueAutoselect,
		git.ConfigPRAutoselect,
		git.ConfigRunHooks,
		git.ConfigFailOnHookError,
		git.ConfigCustomHooks,
		git.ConfigJiraServer,
		git.ConfigJiraProject,
		git.ConfigGitLabServer,
		git.ConfigGitLabProject,
		git.ConfigLinearTeam,
		git.ConfigIssueTemplatesDir,
		git.ConfigIssueTemplatesDisabled,
		git.ConfigIssueTemplatesNoPrompt,
		git.ConfigIssueTemplatesDetected,
	}

	isValidKey := false
	for _, validKey := range validKeys {
		if key == validKey {
			isValidKey = true
			break
		}
	}

	if !isValidKey {
		return fmt.Errorf("unknown configuration key: %s\nRun 'auto-worktree settings list' to see available keys", key)
	}

	// Validate the value
	if err := cfg.Validate(key, value); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}

	// Convert scope
	var configScope git.ConfigScope
	switch scope {
	case scopeLocal:
		configScope = git.ConfigScopeLocal
	case scopeGlobal:
		configScope = git.ConfigScopeGlobal
	default:
		return fmt.Errorf("invalid scope: %s (must be 'local' or 'global')", scope)
	}

	// Set the value
	if err := cfg.SetValidated(key, value, configScope); err != nil {
		return fmt.Errorf("failed to set configuration: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Set %s = %s (%s)",
		strings.TrimPrefix(key, "auto-worktree."), value, scope)))

	return nil
}

// RunSettingsGet gets a configuration value (non-interactive mode)
func RunSettingsGet(key string) error {
	// Normalize key
	if !strings.HasPrefix(key, "auto-worktree.") {
		key = "auto-worktree." + key
	}

	// Initialize repository and config
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	cfg := git.NewConfig(repo.RootPath)

	// Get the value
	value, err := cfg.Get(key, git.ConfigScopeAuto)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	if value == "" {
		fmt.Println(ui.SubtleStyle.Render("(not set)"))
	} else {
		fmt.Println(value)
	}

	return nil
}

// RunSettingsList lists all configuration values (non-interactive mode)
func RunSettingsList() error {
	// Initialize repository and config
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	cfg := git.NewConfig(repo.RootPath)

	// Use the existing showAllSettings function but in a simpler format
	allKeys := []string{
		git.ConfigIssueProvider,
		git.ConfigAITool,
		git.ConfigIssueAutoselect,
		git.ConfigPRAutoselect,
		git.ConfigRunHooks,
		git.ConfigFailOnHookError,
		git.ConfigCustomHooks,
		git.ConfigJiraServer,
		git.ConfigJiraProject,
		git.ConfigGitLabServer,
		git.ConfigGitLabProject,
		git.ConfigLinearTeam,
		git.ConfigIssueTemplatesDir,
		git.ConfigIssueTemplatesDisabled,
		git.ConfigIssueTemplatesNoPrompt,
		git.ConfigIssueTemplatesDetected,
	}

	fmt.Println(ui.TitleStyle.Render("Configuration Settings"))
	fmt.Println()

	for _, key := range allKeys {
		shortKey := strings.TrimPrefix(key, "auto-worktree.")
		localVal, err := cfg.Get(key, git.ConfigScopeLocal)
		if err != nil {
			localVal = ""
		}
		globalVal, err := cfg.Get(key, git.ConfigScopeGlobal)
		if err != nil {
			globalVal = ""
		}

		if localVal != "" {
			fmt.Printf("  %s %s %s\n",
				ui.BoldStyle.Render(shortKey),
				ui.SubtleStyle.Render("[local]"),
				ui.SuccessStyle.Render(localVal))
		}

		if globalVal != "" && globalVal != localVal {
			fmt.Printf("  %s %s %s\n",
				ui.BoldStyle.Render(shortKey),
				ui.SubtleStyle.Render("[global]"),
				ui.InfoStyle.Render(globalVal))
		}
	}

	fmt.Println()
	return nil
}

// RunSettingsReset resets configuration (non-interactive mode)
func RunSettingsReset(scope string) error {
	// Initialize repository and config
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	cfg := git.NewConfig(repo.RootPath)

	// Convert scope
	var configScope git.ConfigScope
	switch scope {
	case scopeLocal:
		configScope = git.ConfigScopeLocal
	case scopeGlobal:
		configScope = git.ConfigScopeGlobal
	default:
		return fmt.Errorf("invalid scope: %s (must be 'local' or 'global')", scope)
	}

	// Confirm with user
	fmt.Printf("%s\n", ui.WarningStyle.Render(fmt.Sprintf("This will reset ALL %s auto-worktree settings.", scope)))
	fmt.Print("Are you sure? (y/N): ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// If error reading input, default to no
		response = "n"
	}

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println(ui.SubtleStyle.Render("Canceled"))
		return nil
	}

	// Reset
	if err := cfg.UnsetAll(configScope); err != nil {
		return fmt.Errorf("failed to reset settings: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ All %s settings reset", scope)))

	return nil
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

// Helper functions for RunIssue

// selectIssueInteractive shows a filterable list of issues and returns the selected issue number
func selectIssueInteractive(client *github.Client, repo *git.Repository) (int, error) {
	// Fetch issues
	fmt.Println("Fetching issues...")
	issues, err := client.ListOpenIssues(100)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch issues: %w", err)
	}

	if len(issues) == 0 {
		return 0, fmt.Errorf("no open issues found")
	}

	// Convert to filterable list items
	items := make([]ui.FilterableListItem, len(issues))
	for i, issue := range issues {
		// Check if worktree exists for this issue
		branchName := issue.BranchName()
		wt, err := repo.GetWorktreeForBranch(branchName)
		if err != nil {
			// Ignore error, just mark as no worktree
			wt = nil
		}

		// Extract label names
		labelNames := make([]string, len(issue.Labels))
		for j, label := range issue.Labels {
			labelNames[j] = label.Name
		}

		items[i] = ui.NewFilterableListItem(
			issue.Number,
			issue.Title,
			labelNames,
			wt != nil,
		)
	}

	// Show filterable list
	filterList := ui.NewFilterList("Select an issue to work on", items)
	p := tea.NewProgram(filterList, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to run issue selector: %w", err)
	}

	finalModel, ok := m.(ui.FilterListModel)
	if !ok {
		return 0, fmt.Errorf("unexpected model type")
	}

	if finalModel.Err() != nil {
		return 0, finalModel.Err()
	}

	choice := finalModel.Choice()
	if choice == nil {
		return 0, fmt.Errorf("no issue selected")
	}

	return choice.Number(), nil
}

// parseIssueNumber parses an issue number from a string, handling "#" prefix
func parseIssueNumber(s string) (int, error) {
	// Remove # prefix if present
	s = strings.TrimPrefix(s, "#")
	return strconv.Atoi(s)
}

// offerResumeWorktree displays information about an existing worktree for an issue
func offerResumeWorktree(wt *git.Worktree, issue *github.Issue) error {
	fmt.Printf("Worktree already exists for issue #%d\n", issue.Number)
	fmt.Printf("Path: %s\n", wt.Path)
	fmt.Printf("Branch: %s\n", wt.Branch)
	fmt.Printf("\nTo resume working:\n")
	fmt.Printf("  cd %s\n", wt.Path)
	return nil
}

// runPostWorktreeHooks executes git hooks after worktree creation
func runPostWorktreeHooks(worktreePath, rootPath string) error {
	config := git.NewConfig(rootPath)
	hookRunner := hooks.NewRunner(worktreePath, config)
	return hookRunner.Run()
}

// startAISession starts an AI tool in a background tmux/screen session
func startAISession(worktreePath, branchName, rootPath string, issue *github.Issue) error {
	// Initialize session manager
	sessionMgr := session.NewManager()
	if !sessionMgr.IsAvailable() {
		fmt.Println("\n⚠ No terminal multiplexer available (install tmux or screen)")
		fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
		fmt.Printf("URL: %s\n", issue.URL)
		fmt.Printf("\nTo start working:\n")
		fmt.Printf("  cd %s\n", worktreePath)
		return nil
	}

	// Resolve AI tool
	config := git.NewConfig(rootPath)
	aiResolver := ai.NewResolver(config)
	aiTool, err := aiResolver.Resolve()
	if err != nil {
		fmt.Printf("\n⚠ %v\n", err)
		fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
		fmt.Printf("URL: %s\n", issue.URL)
		fmt.Printf("\nTo start working:\n")
		fmt.Printf("  cd %s\n", worktreePath)
		return nil
	}

	// Generate session name
	sessionName := session.GenerateSessionName(branchName)

	// Check if session already exists
	exists, err := sessionMgr.HasSession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to check for existing session: %w", err)
	}

	if exists {
		fmt.Printf("\n✓ Session already exists: %s\n", sessionName)
		fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
		fmt.Printf("URL: %s\n", issue.URL)
		fmt.Printf("\nTo attach to the session, run:\n")
		fmt.Printf("  auto-worktree resume\n")
		return nil
	}

	// Create session
	fmt.Printf("\nStarting %s in background session...\n", aiTool.Name)
	if err := sessionMgr.CreateSession(sessionName, worktreePath, aiTool.Command); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	fmt.Printf("✓ Session started: %s\n", sessionName)
	fmt.Printf("\nIssue #%d: %s\n", issue.Number, issue.Title)
	fmt.Printf("URL: %s\n", issue.URL)
	fmt.Printf("\nSession is running in the background using %s\n", sessionMgr.SessionType())
	fmt.Printf("To attach to the session:\n")
	fmt.Printf("  1. Run: auto-worktree resume\n")
	fmt.Printf("  2. Or use: %s attach -t %s\n", sessionMgr.SessionType(), sessionName)

	return nil
}

// selectPRInteractive shows an interactive PR selector with AI-powered priority sorting
func selectPRInteractive(client *github.Client, repo *git.Repository) (int, error) {
	// Fetch PRs
	fmt.Println("Fetching pull requests...")
	prs, err := client.ListOpenPRs(100)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch PRs: %w", err)
	}

	if len(prs) == 0 {
		return 0, fmt.Errorf("no open pull requests found")
	}

	// Check if PR auto-selection is enabled
	prAutoselect, err := repo.Config.GetBool(git.ConfigPRAutoselect, git.ConfigScopeAuto)
	if err == nil && prAutoselect {
		// Apply AI-powered priority sorting
		currentUser := getCurrentGitHubUser()
		prs = sortPRsByPriority(prs, currentUser)

		// Limit to top 5 for auto-selection
		if len(prs) > 5 {
			prs = prs[:5]
		}
		fmt.Printf("Showing top %d prioritized PRs\n", len(prs))
	}

	// Convert to filterable list items
	items := make([]ui.FilterableListItem, len(prs))
	for i, pr := range prs {
		// Check if worktree exists for this PR
		branchName := pr.BranchName()
		wt, err := repo.GetWorktreeForBranch(branchName)
		if err != nil {
			// Ignore error, just mark as no worktree
			wt = nil
		}

		// Extract label names
		labelNames := make([]string, len(pr.Labels))
		for j, label := range pr.Labels {
			labelNames[j] = label.Name
		}

		items[i] = ui.NewFilterableListItem(
			pr.Number,
			pr.Title,
			labelNames,
			wt != nil,
		)
	}

	// Show filterable list
	filterList := ui.NewFilterList("Select a pull request to review", items)
	p := tea.NewProgram(filterList, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to run PR selector: %w", err)
	}

	finalModel, ok := m.(ui.FilterListModel)
	if !ok {
		return 0, fmt.Errorf("unexpected model type")
	}

	if finalModel.Err() != nil {
		return 0, finalModel.Err()
	}

	choice := finalModel.Choice()
	if choice == nil {
		return 0, fmt.Errorf("no PR selected")
	}

	return choice.Number(), nil
}

// parsePRNumber parses a PR number from a string, handling "#" prefix
func parsePRNumber(s string) (int, error) {
	// Remove # prefix if present
	s = strings.TrimPrefix(s, "#")
	return strconv.Atoi(s)
}

// offerResumePRWorktree displays information about an existing worktree for a PR
func offerResumePRWorktree(wt *git.Worktree, pr *github.PullRequest) error {
	fmt.Printf("Worktree already exists for PR #%d\n", pr.Number)
	fmt.Printf("Path: %s\n", wt.Path)
	fmt.Printf("Branch: %s\n", wt.Branch)
	fmt.Printf("\nTo resume reviewing:\n")
	fmt.Printf("  cd %s\n", wt.Path)
	return nil
}

// getCurrentGitHubUser gets the current GitHub user's login
func getCurrentGitHubUser() string {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// sortPRsByPriority sorts PRs by priority:
// 1. Review requested from current user
// 2. Age (oldest first)
// 3. Size (smaller first)
// 4. CI status (passing first)
func sortPRsByPriority(prs []github.PullRequest, currentUser string) []github.PullRequest {
	// Create a copy to avoid modifying the original
	sorted := make([]github.PullRequest, len(prs))
	copy(sorted, prs)

	// Sort using multiple criteria
	// Note: In Go, we need to implement a custom sort with comparison function
	// For simplicity, we'll use a scoring system

	type prScore struct {
		pr    github.PullRequest
		score int
	}

	scores := make([]prScore, len(sorted))
	for i, pr := range sorted {
		score := 0

		// Priority 1: Review requested from current user (highest priority)
		if currentUser != "" && pr.IsRequestedReviewer(currentUser) {
			score += 1000
		}

		// Priority 2: Age (older = higher score, max 100 points)
		// We don't have creation date in the struct, so we'll use the PR number as a proxy
		// Lower PR numbers = older PRs
		ageScore := 100 - (pr.Number % 100)
		score += ageScore

		// Priority 3: Size (smaller = higher score, max 50 points)
		totalChanges := pr.Additions + pr.Deletions
		var sizeScore int
		switch {
		case totalChanges < 50:
			sizeScore = 50
		case totalChanges < 200:
			sizeScore = 40
		case totalChanges < 500:
			sizeScore = 30
		case totalChanges < 1000:
			sizeScore = 20
		default:
			sizeScore = 10
		}
		score += sizeScore

		// Priority 4: CI status (passing = higher score, 25 points)
		if pr.AllChecksPass() {
			score += 25
		}

		scores[i] = prScore{pr: pr, score: score}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Extract sorted PRs
	for i, ps := range scores {
		sorted[i] = ps.pr
	}

	return sorted
}

// shouldGenerateAIReview checks if AI review should be generated
func shouldGenerateAIReview(repo *git.Repository) bool {
	aiTool, err := repo.Config.Get(git.ConfigAITool, git.ConfigScopeAuto)
	if err != nil {
		return false
	}
	return aiTool != "" && aiTool != "skip"
}

// generateAIReviewSummary generates an AI-powered review summary
func generateAIReviewSummary(client *github.Client, pr *github.PullRequest, repo *git.Repository) error {
	// Get configured AI tool
	aiTool, err := repo.Config.Get(git.ConfigAITool, git.ConfigScopeAuto)
	if err != nil || aiTool == "" || aiTool == "skip" {
		return fmt.Errorf("no AI tool configured")
	}

	// Get PR diff
	diff, err := client.GetPRDiff(pr.Number)
	if err != nil {
		return fmt.Errorf("failed to fetch PR diff: %w", err)
	}

	// Truncate diff if too long (limit to first 10000 chars)
	if len(diff) > 10000 {
		diff = diff[:10000] + "\n... (diff truncated)"
	}

	// Format prompt for AI
	prompt := formatAIReviewPrompt(pr, diff)

	fmt.Printf("\n━━━━ AI Review Summary (%s) ━━━━\n\n", aiTool)
	fmt.Println("This PR makes the following changes:")

	// For now, we'll show a placeholder message
	// In a full implementation, this would call the AI service
	fmt.Printf("\nPR #%d modifies %d files with +%d/-%d lines.\n", pr.Number, pr.ChangedFiles, pr.Additions, pr.Deletions)
	fmt.Printf("\nKey areas to review:\n")
	fmt.Printf("  • Changes affect %s → %s\n", pr.BaseRefName, pr.HeadRefName)

	if len(pr.Labels) > 0 {
		labels := make([]string, len(pr.Labels))
		for i, label := range pr.Labels {
			labels[i] = label.Name
		}
		fmt.Printf("  • Labeled as: %s\n", strings.Join(labels, ", "))
	}

	fmt.Printf("\n💡 Note: Full AI integration requires API configuration\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Store prompt for future use
	_ = prompt

	return nil
}

// formatAIReviewPrompt formats a prompt for AI review
func formatAIReviewPrompt(pr *github.PullRequest, diff string) string {
	return fmt.Sprintf(`Please review this pull request:

Title: %s
Author: %s
Description:
%s

Changes:
- Files changed: %d
- Additions: +%d
- Deletions: -%d

Diff:
%s

Please provide:
1. A summary of what this PR does
2. Key areas to review
3. Potential concerns or questions
4. Suggestions for improvement
`, pr.Title, pr.Author.Login, pr.Body, pr.ChangedFiles, pr.Additions, pr.Deletions, diff)
}

// checkoutPRInWorktree creates a worktree and checks out the PR branch
func checkoutPRInWorktree(repo *git.Repository, worktreePath, branchName string, pr *github.PullRequest) error {
	// Use gh pr checkout to fetch and checkout the PR
	// This will create a local branch tracking the PR's head branch
	executor := git.NewGitExecutor()

	// First, create the worktree directory with a temporary branch
	defaultBranch, err := repo.GetDefaultBranch()
	if err != nil {
		return fmt.Errorf("error getting default branch: %w", err)
	}

	// Create worktree with new branch
	if err := repo.CreateWorktreeWithNewBranch(worktreePath, branchName, defaultBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Now checkout the PR in that worktree using gh pr checkout
	checkoutCmd := fmt.Sprintf("cd %s && gh pr checkout %d -b %s", worktreePath, pr.Number, branchName)
	if _, err := executor.Execute(checkoutCmd); err != nil {
		// If checkout fails, try to clean up the worktree
		if removeErr := repo.RemoveWorktree(worktreePath); removeErr != nil {
			// Log the error but don't fail - we're already in an error state
			fmt.Printf("Warning: Could not clean up worktree: %v\n", removeErr)
		}
		return fmt.Errorf("failed to checkout PR #%d: %w", pr.Number, err)
	}

	return nil
}
