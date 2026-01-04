// Package ai resolves which AI tool to use based on configuration and availability
package ai

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// Tool represents an AI coding assistant tool
type Tool struct {
	Name          string   // Display name (e.g., "Claude Code")
	ConfigKey     string   // Config value (e.g., "claude")
	Command       []string // Command to start fresh session
	ResumeCommand []string // Command to resume existing session
}

// InstallInstructions contains installation information for an AI tool
type InstallInstructions struct {
	Name    string
	Methods []string
	InfoURL string
}

// Resolver resolves which AI tool to use based on configuration
type Resolver struct {
	config *git.Config
}

// NewResolver creates a new AI tool resolver
func NewResolver(config *git.Config) *Resolver {
	return &Resolver{config: config}
}

// Resolve determines which AI tool to use
// It checks:
// 1. Saved configuration (auto-worktree.ai-tool)
// 2. Available tools in PATH
// 3. Returns error if no tool is available or configured to skip
func (r *Resolver) Resolve() (*Tool, error) {
	// Check saved preference
	savedTool := r.config.GetAITool()

	if savedTool == "skip" {
		return nil, fmt.Errorf("AI tool disabled (auto-worktree.ai-tool=skip)")
	}

	// If a tool is configured, try to use it
	if savedTool != "" {
		if tool := r.getTool(savedTool); tool != nil {
			return tool, nil
		}
		// Configured tool not found, fall through to auto-detect
	}

	// Auto-detect available tools (in preference order)
	toolPreferences := []string{"claude", "codex", "gemini", "jules"}
	for _, name := range toolPreferences {
		if tool := r.getTool(name); tool != nil {
			return tool, nil
		}
	}

	return nil, fmt.Errorf("no AI tool found (install claude, codex, gemini, or jules)")
}

// getTool returns a Tool if the specified tool is available
func (r *Resolver) getTool(name string) *Tool {
	switch name {
	case "claude":
		if commandExists("claude") {
			return &Tool{
				Name:          "Claude Code",
				ConfigKey:     "claude",
				Command:       []string{"claude", "--dangerously-skip-permissions"},
				ResumeCommand: []string{"claude", "--dangerously-skip-permissions", "--continue"},
			}
		}
	case "codex":
		if commandExists("codex") {
			return &Tool{
				Name:          "Codex",
				ConfigKey:     "codex",
				Command:       []string{"codex", "--yolo"},
				ResumeCommand: []string{"codex", "resume", "--last"},
			}
		}
	case "gemini":
		if commandExists("gemini") {
			return &Tool{
				Name:          "Gemini CLI",
				ConfigKey:     "gemini",
				Command:       []string{"gemini", "--yolo"},
				ResumeCommand: []string{"gemini", "--resume"},
			}
		}
	case "jules":
		if commandExists("jules") {
			return &Tool{
				Name:          "Google Jules CLI",
				ConfigKey:     "jules",
				Command:       []string{"jules"},
				ResumeCommand: []string{"jules"}, // Jules has no special resume flag
			}
		}
	}

	return nil
}

// ListAvailable returns all available AI tools
func (r *Resolver) ListAvailable() []Tool {
	var tools []Tool

	for _, name := range []string{"claude", "codex", "gemini", "jules"} {
		if tool := r.getTool(name); tool != nil {
			tools = append(tools, *tool)
		}
	}

	return tools
}

// CommandWithContext returns the command to run with an initial context/prompt.
// The context is passed as a positional argument to the AI tool.
func (t *Tool) CommandWithContext(context string) []string {
	if context == "" {
		return t.Command
	}

	// Append context as positional argument
	cmd := make([]string, len(t.Command), len(t.Command)+1)
	copy(cmd, t.Command)

	return append(cmd, context)
}

// ResumeCommandWithContext returns the resume command with optional context.
func (t *Tool) ResumeCommandWithContext(context string) []string {
	if context == "" {
		return t.ResumeCommand
	}

	cmd := make([]string, len(t.ResumeCommand), len(t.ResumeCommand)+1)
	copy(cmd, t.ResumeCommand)

	return append(cmd, context)
}

// HasExistingSession checks if there's an existing AI session in the given directory
// that can be resumed. This checks for tool-specific session markers.
func HasExistingSession(worktreePath string) bool {
	// Check for Claude Code session markers
	claudeDir := filepath.Join(worktreePath, ".claude")
	if _, err := os.Stat(claudeDir); err == nil {
		return true
	}

	claudeJSON := filepath.Join(worktreePath, ".claude.json")
	if _, err := os.Stat(claudeJSON); err == nil {
		return true
	}

	// Other tools may have their own session markers
	// Add checks here as needed for codex, gemini, jules

	return false
}

// GetInstallInstructions returns installation instructions for all supported AI tools
func GetInstallInstructions() []InstallInstructions {
	return []InstallInstructions{
		{
			Name: "Claude Code (Anthropic)",
			Methods: []string{
				"macOS:   brew install claude",
				"npm:     npm install -g @anthropic-ai/claude-code",
			},
			InfoURL: "https://github.com/anthropics/claude-code",
		},
		{
			Name: "Codex CLI (OpenAI)",
			Methods: []string{
				"npm:     npm install -g @openai/codex-cli",
			},
			InfoURL: "https://github.com/openai/codex",
		},
		{
			Name: "Gemini CLI (Google)",
			Methods: []string{
				"npm:     npm install -g @google/gemini-cli",
			},
			InfoURL: "https://github.com/google-gemini/gemini-cli",
		},
		{
			Name: "Google Jules CLI (Google)",
			Methods: []string{
				"npm:     npm install -g @google/jules",
			},
			InfoURL: "https://jules.google/docs",
		},
	}
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
