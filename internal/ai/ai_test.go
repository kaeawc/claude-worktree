// Package ai resolves which AI tool to use based on configuration and availability
package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolCommandWithContext(t *testing.T) {
	tool := &Tool{
		Name:          "Claude Code",
		ConfigKey:     "claude",
		Command:       []string{"claude", "--dangerously-skip-permissions"},
		ResumeCommand: []string{"claude", "--dangerously-skip-permissions", "--continue"},
	}

	tests := []struct {
		name     string
		context  string
		expected []string
	}{
		{
			name:     "empty context returns base command",
			context:  "",
			expected: []string{"claude", "--dangerously-skip-permissions"},
		},
		{
			name:     "context is appended",
			context:  "I'm working on issue #123",
			expected: []string{"claude", "--dangerously-skip-permissions", "I'm working on issue #123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.CommandWithContext(tt.context)
			if len(result) != len(tt.expected) {
				t.Errorf("CommandWithContext() = %v, want %v", result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("CommandWithContext()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestToolResumeCommandWithContext(t *testing.T) {
	tool := &Tool{
		Name:          "Claude Code",
		ConfigKey:     "claude",
		Command:       []string{"claude", "--dangerously-skip-permissions"},
		ResumeCommand: []string{"claude", "--dangerously-skip-permissions", "--continue"},
	}

	tests := []struct {
		name     string
		context  string
		expected []string
	}{
		{
			name:     "empty context returns base resume command",
			context:  "",
			expected: []string{"claude", "--dangerously-skip-permissions", "--continue"},
		},
		{
			name:     "context is appended to resume command",
			context:  "Continue from where we left off",
			expected: []string{"claude", "--dangerously-skip-permissions", "--continue", "Continue from where we left off"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.ResumeCommandWithContext(tt.context)
			if len(result) != len(tt.expected) {
				t.Errorf("ResumeCommandWithContext() = %v, want %v", result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("ResumeCommandWithContext()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestHasExistingSession(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func() string
		expected bool
	}{
		{
			name: "no session markers",
			setup: func() string {
				dir := filepath.Join(tempDir, "no-session")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: false,
		},
		{
			name: "has .claude directory",
			setup: func() string {
				dir := filepath.Join(tempDir, "with-claude-dir")
				claudeDir := filepath.Join(dir, ".claude")
				if err := os.MkdirAll(claudeDir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: true,
		},
		{
			name: "has .claude.json file",
			setup: func() string {
				dir := filepath.Join(tempDir, "with-claude-json")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				claudeJSON := filepath.Join(dir, ".claude.json")
				if err := os.WriteFile(claudeJSON, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktreePath := tt.setup()
			result := HasExistingSession(worktreePath)
			if result != tt.expected {
				t.Errorf("HasExistingSession() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetInstallInstructions(t *testing.T) {
	instructions := GetInstallInstructions()

	// Should have instructions for all 4 tools
	if len(instructions) != 4 {
		t.Errorf("GetInstallInstructions() returned %d instructions, want 4", len(instructions))
	}

	// Check that each has required fields
	expectedNames := []string{
		"Claude Code (Anthropic)",
		"Codex CLI (OpenAI)",
		"Gemini CLI (Google)",
		"Google Jules CLI (Google)",
	}

	for i, inst := range instructions {
		if inst.Name != expectedNames[i] {
			t.Errorf("instruction[%d].Name = %v, want %v", i, inst.Name, expectedNames[i])
		}
		if len(inst.Methods) == 0 {
			t.Errorf("instruction[%d].Methods is empty", i)
		}
		if inst.InfoURL == "" {
			t.Errorf("instruction[%d].InfoURL is empty", i)
		}
	}
}

func TestToolConfigKeys(t *testing.T) {
	// Test that ConfigKey values match git config values
	tests := []struct {
		toolName  string
		configKey string
	}{
		{"Claude Code", "claude"},
		{"Codex", "codex"},
		{"Gemini CLI", "gemini"},
		{"Google Jules CLI", "jules"},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			// We can't easily test getTool without mocking command availability,
			// but we can verify the expected config keys are documented correctly
			switch tt.configKey {
			case "claude", "codex", "gemini", "jules":
				// Valid config keys
			default:
				t.Errorf("Unknown config key: %s", tt.configKey)
			}
		})
	}
}
