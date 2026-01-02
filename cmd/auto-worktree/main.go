// Package main provides the auto-worktree CLI tool for managing git worktrees.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/kaeawc/auto-worktree/internal/cmd"
)

const version = "0.1.0-dev"

func main() {
	// Prune orphaned worktrees on startup (silently)
	_ = pruneOrphanedWorktrees() //nolint:errcheck

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

// pruneOrphanedWorktrees silently prunes orphaned worktrees using git
func pruneOrphanedWorktrees() error {
	// Run git worktree prune in the current directory
	// This cleans up stale worktree metadata for deleted directories
	cmd := exec.CommandContext(context.Background(), "git", "worktree", "prune")
	// Suppress output - we don't want to show this to the user
	return cmd.Run()
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
		return runSettingsCommand()

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

func runSettingsCommand() error {
	// If no subcommand, run interactive mode
	if len(os.Args) < 3 {
		return cmd.RunSettings()
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "set":
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "Error: key and value required\n")
			fmt.Fprintf(os.Stderr, "Usage: auto-worktree settings set <key> <value> [--global]\n")
			os.Exit(1)
		}

		key := os.Args[3]
		value := os.Args[4]
		scope := "local"

		if len(os.Args) > 5 && os.Args[5] == "--global" {
			scope = "global"
		}

		return cmd.RunSettingsSet(key, value, scope)

	case "get":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Error: key required\n")
			fmt.Fprintf(os.Stderr, "Usage: auto-worktree settings get <key>\n")
			os.Exit(1)
		}

		key := os.Args[3]

		return cmd.RunSettingsGet(key)

	case "list":
		return cmd.RunSettingsList()

	case "reset":
		scope := "local"

		if len(os.Args) > 3 && os.Args[3] == "--global" {
			scope = "global"
		}

		return cmd.RunSettingsReset(scope)

	default:
		fmt.Fprintf(os.Stderr, "Unknown settings subcommand: %s\n\n", subcommand)
		fmt.Fprintf(os.Stderr, "Available subcommands:\n")
		fmt.Fprintf(os.Stderr, "  set <key> <value> [--global]  Set a configuration value\n")
		fmt.Fprintf(os.Stderr, "  get <key>                      Get a configuration value\n")
		fmt.Fprintf(os.Stderr, "  list                           List all configuration values\n")
		fmt.Fprintf(os.Stderr, "  reset [--global]               Reset all settings to defaults\n")
		os.Exit(1)

		return nil
	}
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
