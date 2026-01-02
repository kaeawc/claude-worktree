// Package ai resolves which AI tool to use based on configuration and availability
package ai

import (
	"fmt"
	"os/exec"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// Tool represents an AI coding assistant tool
type Tool struct {
	Name    string
	Command []string
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
				Name:    "Claude Code",
				Command: []string{"claude", "--dangerously-skip-permissions"},
			}
		}
	case "codex":
		if commandExists("codex") {
			return &Tool{
				Name:    "Codex",
				Command: []string{"codex", "--yolo"},
			}
		}
	case "gemini":
		if commandExists("gemini") {
			return &Tool{
				Name:    "Gemini CLI",
				Command: []string{"gemini", "--yolo"},
			}
		}
	case "jules":
		if commandExists("jules") {
			return &Tool{
				Name:    "Google Jules CLI",
				Command: []string{"jules"},
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

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
