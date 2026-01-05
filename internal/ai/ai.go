// Package ai resolves which AI tool to use based on configuration and availability
package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// AI tool config keys
const (
	toolClaude = "claude"
	toolCodex  = "codex"
	toolGemini = "gemini"
	toolJules  = "jules"
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
	case toolClaude:
		if commandExists(toolClaude) {
			return &Tool{
				Name:          "Claude Code",
				ConfigKey:     toolClaude,
				Command:       []string{toolClaude, "--dangerously-skip-permissions"},
				ResumeCommand: []string{toolClaude, "--dangerously-skip-permissions", "--continue"},
			}
		}
	case toolCodex:
		if commandExists(toolCodex) {
			return &Tool{
				Name:          "Codex",
				ConfigKey:     toolCodex,
				Command:       []string{toolCodex, "--yolo"},
				ResumeCommand: []string{toolCodex, "resume", "--last"},
			}
		}
	case toolGemini:
		if commandExists(toolGemini) {
			return &Tool{
				Name:          "Gemini CLI",
				ConfigKey:     toolGemini,
				Command:       []string{toolGemini, "--yolo"},
				ResumeCommand: []string{toolGemini, "--resume"},
			}
		}
	case toolJules:
		if commandExists(toolJules) {
			return &Tool{
				Name:          "Google Jules CLI",
				ConfigKey:     toolJules,
				Command:       []string{toolJules},
				ResumeCommand: []string{toolJules}, // Jules has no special resume flag
			}
		}
	}

	return nil
}

// ListAvailable returns all available AI tools
func (r *Resolver) ListAvailable() []Tool {
	var tools []Tool

	for _, name := range []string{toolClaude, toolCodex, toolGemini, toolJules} {
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
	if hasCodexSession(worktreePath) {
		return true
	}

	return false
}

type codexSessionMeta struct {
	Type    string `json:"type"`
	Payload struct {
		Cwd string `json:"cwd"`
	} `json:"payload"`
}

// errCodexSessionFound is a sentinel error used to short-circuit the walk.
var errCodexSessionFound = errors.New("codex session found")

func hasCodexSession(worktreePath string) bool {
	sessionsDir := getCodexSessionsDir()
	if sessionsDir == "" {
		return false
	}

	if _, err := os.Stat(sessionsDir); err != nil {
		return false
	}

	err := filepath.WalkDir(sessionsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			return nil
		}

		if checkCodexSessionFile(path, worktreePath) {
			return errCodexSessionFound
		}

		return nil
	})

	return errors.Is(err, errCodexSessionFound)
}

func getCodexSessionsDir() string {
	codexHome := os.Getenv("CODEX_HOME")

	if codexHome == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return ""
		}

		codexHome = filepath.Join(homeDir, ".codex")
	}

	return filepath.Join(codexHome, "sessions")
}

func checkCodexSessionFile(path, worktreePath string) bool {
	file, err := os.Open(path) //nolint:gosec // path comes from filepath.WalkDir
	if err != nil {
		return false
	}

	defer file.Close() //nolint:errcheck // read-only file, error on close is not actionable

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineCount := 0

	for scanner.Scan() {
		lineCount++

		if lineCount > 200 {
			break
		}

		line := scanner.Text()

		if !strings.Contains(line, "session_meta") {
			continue
		}

		var meta codexSessionMeta
		if err := json.Unmarshal([]byte(line), &meta); err != nil {
			continue
		}

		if meta.Type == "session_meta" && meta.Payload.Cwd == worktreePath {
			return true
		}
	}

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

// ExecutePrompt executes a one-shot prompt with the AI tool and returns the output.
// This is used for non-interactive tasks like auto-selecting issues/PRs.
// Returns the raw output from the AI tool.
func (t *Tool) ExecutePrompt(prompt string) (string, error) {
	// Build tool-specific command for one-shot prompt execution
	ctx := context.Background()
	var cmd *exec.Cmd

	switch t.ConfigKey {
	case toolClaude:
		// Claude uses --print flag for non-interactive output
		cmd = exec.CommandContext(ctx, toolClaude, "--print")
	case toolGemini:
		// Gemini uses --yolo flag to auto-approve actions
		cmd = exec.CommandContext(ctx, toolGemini, "--yolo")
	case toolCodex:
		// Codex needs testing - using similar pattern to gemini
		cmd = exec.CommandContext(ctx, toolCodex, "--yolo")
	case toolJules:
		// Jules doesn't support stdin piping for one-shot prompts
		return "", fmt.Errorf("jules does not support one-shot prompt execution")
	default:
		return "", fmt.Errorf("unsupported AI tool for prompt execution: %s", t.ConfigKey)
	}

	// Set up stdin with the prompt
	cmd.Stdin = strings.NewReader(prompt)

	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		// Include stderr in error message for debugging
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("AI tool execution failed: %w\nstderr: %s", err, stderrStr)
		}

		return "", fmt.Errorf("AI tool execution failed: %w", err)
	}

	return stdout.String(), nil
}

// ParseNumericIDs extracts numeric IDs from AI output.
// Used for GitHub issue/PR numbers (e.g., "42", "123").
func ParseNumericIDs(output string, limit int) []string {
	var ids []string
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Match lines that contain only digits
		if len(line) > 0 && isNumeric(line) {
			ids = append(ids, line)
			if len(ids) >= limit {
				break
			}
		}
	}

	return ids
}

// ParseLinearIDs extracts Linear-style IDs from AI output.
// Used for Linear issue IDs (e.g., "TEAM-42", "PROJ-123").
func ParseLinearIDs(output string, limit int) []string {
	var ids []string
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Match lines with Linear ID format: UPPERCASE-NUMBER
		if len(line) > 0 && isLinearID(line) {
			ids = append(ids, line)
			if len(ids) >= limit {
				break
			}
		}
	}

	return ids
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// isLinearID checks if a string matches Linear ID format (e.g., "TEAM-42")
func isLinearID(s string) bool {
	if len(s) < 3 {
		return false
	}

	// Find the hyphen
	hyphenIdx := strings.IndexByte(s, '-')
	if hyphenIdx <= 0 || hyphenIdx >= len(s)-1 {
		return false
	}

	// Check prefix is uppercase alphanumeric with at least one letter
	prefix := s[:hyphenIdx]
	hasLetter := false

	for _, c := range prefix {
		switch {
		case c >= 'A' && c <= 'Z':
			hasLetter = true
		case c >= '0' && c <= '9':
			continue
		default:
			return false
		}
	}

	if !hasLetter {
		return false
	}

	// Check suffix is numeric
	suffix := s[hyphenIdx+1:]

	return isNumeric(suffix)
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
