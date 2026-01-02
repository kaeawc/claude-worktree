package gitlab

import (
	"errors"
	"testing"
)

func TestNewFakeGitLabExecutor(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	if fake == nil {
		t.Fatal("NewFakeGitLabExecutor returned nil")
	}
	if fake.GetCommandCount() != 0 {
		t.Errorf("expected 0 commands, got %d", fake.GetCommandCount())
	}
}

func TestFakeExecute(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("version", "glab version 1.0.0")

	output, err := fake.Execute("version")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "glab version 1.0.0" {
		t.Errorf("expected 'glab version 1.0.0', got '%s'", output)
	}
	if fake.GetCommandCount() != 1 {
		t.Errorf("expected 1 command, got %d", fake.GetCommandCount())
	}
}

func TestFakeExecuteInDir(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("auth status", "user authenticated")

	output, err := fake.ExecuteInDir("/some/dir", "auth", "status")
	if err != nil {
		t.Fatalf("ExecuteInDir failed: %v", err)
	}
	if output != "user authenticated" {
		t.Errorf("expected 'user authenticated', got '%s'", output)
	}
	if fake.GetCommandCount() != 1 {
		t.Errorf("expected 1 command, got %d", fake.GetCommandCount())
	}
}

func TestFakeExecuteError(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	testErr := errors.New("command failed")
	fake.SetError("issue view 123", testErr)

	output, err := fake.Execute("issue", "view", "123")
	if err != testErr {
		t.Errorf("expected error '%v', got '%v'", testErr, err)
	}
	if output != "" {
		t.Errorf("expected empty output, got '%s'", output)
	}
}

func TestFakeGetLastCommand(t *testing.T) {
	fake := NewFakeGitLabExecutor()

	// No commands yet
	if cmd := fake.GetLastCommand(); cmd != nil {
		t.Error("expected nil for last command, got non-nil")
	}

	// Execute some commands
	fake.Execute("issue", "list")
	fake.Execute("mr", "view", "123")

	// Check last command
	last := fake.GetLastCommand()
	if len(last) != 3 {
		t.Errorf("expected 3 args in last command, got %d", len(last))
	}
	if last[0] != "mr" || last[1] != "view" || last[2] != "123" {
		t.Errorf("unexpected last command: %v", last)
	}
}

func TestFakeReset(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("version", "1.0.0")
	fake.Execute("version")

	if fake.GetCommandCount() != 1 {
		t.Errorf("expected 1 command before reset, got %d", fake.GetCommandCount())
	}

	fake.Reset()

	if fake.GetCommandCount() != 0 {
		t.Errorf("expected 0 commands after reset, got %d", fake.GetCommandCount())
	}

	// Verify responses were cleared
	fake.Execute("version")
	if fake.GetCommandCount() != 1 {
		t.Errorf("expected 1 command after reset and new execute, got %d", fake.GetCommandCount())
	}
}

func TestFakeDefaultResponse(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.DefaultResponse = "default response"

	output, err := fake.Execute("unknown", "command")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if output != "default response" {
		t.Errorf("expected 'default response', got '%s'", output)
	}
}

func TestFakeMultipleCommands(t *testing.T) {
	fake := NewFakeGitLabExecutor()

	fake.Execute("issue", "list")
	fake.Execute("issue", "view", "123")
	fake.Execute("mr", "list")

	if fake.GetCommandCount() != 3 {
		t.Errorf("expected 3 commands, got %d", fake.GetCommandCount())
	}
}
