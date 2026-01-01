// Package main provides the auto-worktree CLI tool for managing git worktrees.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("auto-worktree version 0.1.0-dev")

		return
	}

	fmt.Println("auto-worktree - Git worktree management tool")
	fmt.Println("This is a development build.")
	os.Exit(0)
}
