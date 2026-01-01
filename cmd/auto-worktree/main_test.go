package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") == "1" {
		os.Args = []string{"auto-worktree", "version"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestVersionCommand")
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
