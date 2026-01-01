package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") == "1" {
		// Run with help argument to avoid interactive menu in tests
		os.Args = []string{"auto-worktree", "help"}
		main()
		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestMain")
	cmd.Env = append(os.Environ(), "GO_TEST_PROCESS=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process exited with error: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "auto-worktree") {
		t.Errorf("Expected output to contain 'auto-worktree', got: %s", outputStr)
	}
}

func TestVersionCommand(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") == "1" {
		os.Args = []string{"auto-worktree", "version"}
		main()
		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestVersionCommand")
	cmd.Env = append(os.Environ(), "GO_TEST_PROCESS=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process exited with error: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "version") {
		t.Errorf("Expected version output, got: %s", outputStr)
	}
}
