// Package main provides the auto-worktree CLI tool for managing git worktrees.
package main

import (
	"fmt"
	"os"

	"github.com/kaeawc/auto-worktree/internal/cmd"
)

const version = "0.1.0-dev"

func main() {
	// If no arguments, show interactive menu
	if len(os.Args) < 2 {
		if err := cmd.RunInteractiveMenu(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		return
	}

	if err := runCommand(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(command string) error {
	switch command {
	case "version", "--version", "-v":
		fmt.Printf("auto-worktree version %s\n", version)
		return nil

	case "help", "--help", "-h":
		showHelp()
		return nil

	case "list", "ls":
		return cmd.RunList()

	case "new", "create":
		return cmd.RunNew()

	case "resume":
		return cmd.RunResume()

	case "issue":
		return runIssueCommand()

	case "pr":
		return runPRCommand()

	case "cleanup":
		return cmd.RunCleanup()

	case "settings":
		return cmd.RunSettings()

	case "remove", "rm":
		return runRemoveCommand()

	case "prune":
		return cmd.RunPrune()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		showHelp()
		os.Exit(1)

		return nil
	}
}

func runIssueCommand() error {
	issueID := ""
	if len(os.Args) > 2 {
		issueID = os.Args[2]
	}

	return cmd.RunIssue(issueID)
}

func runPRCommand() error {
	prNum := ""
	if len(os.Args) > 2 {
		prNum = os.Args[2]
	}

	return cmd.RunPR(prNum)
}

func runRemoveCommand() error {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: worktree path required\n")
		fmt.Fprintf(os.Stderr, "Usage: auto-worktree remove <path>\n")
		os.Exit(1)
	}

	return cmd.RunRemove(os.Args[2])
}

func showHelp() {
	help := `auto-worktree - Git worktree management tool

USAGE:
    auto-worktree [command] [arguments]
    aw [command] [arguments]              # Shorter alias

COMMANDS:
    (no command)          Show interactive menu
    new [branch]          Create new worktree
    resume                Resume last worktree
    issue [id]            Work on an issue (GitHub, GitLab, JIRA, or Linear)
    create                Create a new issue and start working on it
    pr [num]              Review a pull request
    list, ls              List all worktrees with status
    cleanup               Interactive cleanup of merged/stale worktrees
    settings              Configure per-repository settings
    remove <path>         Remove a worktree
    prune                 Prune orphaned worktrees
    version               Show version information
    help                  Show this help message

EXAMPLES:
    # Show interactive menu
    auto-worktree

    # Create a new worktree
    auto-worktree new feature/new-feature

    # Work on a GitHub issue
    auto-worktree issue 42

    # Review a pull request
    auto-worktree pr 123

    # List all worktrees
    auto-worktree list

    # Resume last worktree
    auto-worktree resume

    # Interactive cleanup
    auto-worktree cleanup

    # Configure settings
    auto-worktree settings

    # Remove a worktree
    auto-worktree remove ~/worktrees/my-repo/feature-branch

    # Clean up orphaned worktrees
    auto-worktree prune

For more information, visit: https://github.com/kaeawc/auto-worktree
`
	fmt.Print(help)
}
