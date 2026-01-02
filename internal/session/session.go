// Package session manages terminal multiplexer sessions for worktrees
package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Type represents the type of terminal multiplexer
type Type string

// Terminal multiplexer types
const (
	TypeTmux   Type = "tmux"
	TypeScreen Type = "screen"
	TypeNone   Type = "none"
)

// Session represents a terminal multiplexer session
type Session struct {
	Name      string
	Type      Type
	Directory string
}

// Manager handles terminal multiplexer sessions
type Manager struct {
	sessionType Type
}

// NewManager creates a new session manager
// It detects which multiplexer is available (tmux preferred, screen fallback)
func NewManager() *Manager {
	if commandExists("tmux") {
		return &Manager{sessionType: TypeTmux}
	}

	if commandExists("screen") {
		return &Manager{sessionType: TypeScreen}
	}

	return &Manager{sessionType: TypeNone}
}

// SessionType returns the session type this manager uses
func (m *Manager) SessionType() Type {
	return m.sessionType
}

// IsAvailable returns true if a session manager is available
func (m *Manager) IsAvailable() bool {
	return m.sessionType != TypeNone
}

// CreateSession creates a new detached session running the specified command
func (m *Manager) CreateSession(name, workingDir string, command []string) error {
	if !m.IsAvailable() {
		return fmt.Errorf("no terminal multiplexer available (install tmux or screen)")
	}

	switch m.sessionType {
	case TypeTmux:
		return m.createTmuxSession(name, workingDir, command)
	case TypeScreen:
		return m.createScreenSession(name, workingDir, command)
	default:
		return fmt.Errorf("unsupported session type: %s", m.sessionType)
	}
}

// createTmuxSession creates a detached tmux session
func (m *Manager) createTmuxSession(name, workingDir string, command []string) error {
	args := []string{
		"new-session",
		"-d",       // Detached
		"-s", name, // Session name
		"-c", workingDir, // Working directory
	}
	args = append(args, command...)

	cmd := exec.CommandContext(context.Background(), "tmux", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	return nil
}

// createScreenSession creates a detached screen session
func (m *Manager) createScreenSession(name, workingDir string, command []string) error {
	// screen doesn't support -c flag for working directory,
	// so we wrap the command in a shell that changes directory first
	shellCmd := fmt.Sprintf("cd %s && %s",
		escapeShellArg(workingDir),
		strings.Join(escapeShellArgs(command), " "))

	cmd := exec.CommandContext(context.Background(), "screen", "-dmS", name, "bash", "-c", shellCmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create screen session: %w", err)
	}

	return nil
}

// HasSession checks if a session with the given name exists
func (m *Manager) HasSession(name string) (bool, error) {
	if !m.IsAvailable() {
		return false, nil
	}

	switch m.sessionType {
	case TypeTmux:
		cmd := exec.CommandContext(context.Background(), "tmux", "has-session", "-t", name)
		return cmd.Run() == nil, nil
	case TypeScreen:
		// List sessions and check if name exists
		cmd := exec.CommandContext(context.Background(), "screen", "-ls")
		output, err := cmd.Output()

		if err != nil {
			// screen -ls returns exit code 1 if no sessions exist
			if len(output) > 0 {
				return strings.Contains(string(output), name), nil
			}

			return false, nil
		}

		return strings.Contains(string(output), name), nil
	default:
		return false, nil
	}
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() ([]string, error) {
	if !m.IsAvailable() {
		return []string{}, nil
	}

	switch m.sessionType {
	case TypeTmux:
		return m.listTmuxSessions()
	case TypeScreen:
		return m.listScreenSessions()
	default:
		return []string{}, nil
	}
}

// listTmuxSessions lists all tmux sessions
func (m *Manager) listTmuxSessions() ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()

	if err != nil {
		// No sessions exist
		return []string{}, nil
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")

	return sessions, nil
}

// listScreenSessions lists all screen sessions
func (m *Manager) listScreenSessions() ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "screen", "-ls")
	output, err := cmd.Output()

	if err != nil && len(output) == 0 {
		// No sessions exist
		return []string{}, nil
	}

	// Parse screen -ls output
	// Format: "12345.session-name	(Detached)"
	var sessions []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "(Detached)") || strings.Contains(line, "(Attached)") {
			// Extract session name
			parts := strings.Fields(line)
			if len(parts) > 0 {
				// Remove PID prefix (12345.session-name -> session-name)
				sessionFull := parts[0]
				if idx := strings.Index(sessionFull, "."); idx != -1 {
					sessions = append(sessions, sessionFull[idx+1:])
				}
			}
		}
	}

	return sessions, nil
}

// KillSession terminates a session
func (m *Manager) KillSession(name string) error {
	if !m.IsAvailable() {
		return fmt.Errorf("no terminal multiplexer available")
	}

	switch m.sessionType {
	case TypeTmux:
		cmd := exec.CommandContext(context.Background(), "tmux", "kill-session", "-t", name)
		return cmd.Run()
	case TypeScreen:
		// screen requires the full session name with PID prefix
		// We need to find it first
		cmd := exec.CommandContext(context.Background(), "screen", "-ls")
		output, err := cmd.Output()

		if err != nil {
			return fmt.Errorf("failed to list screen sessions: %w", err)
		}

		lines := strings.Split(string(output), "\n")

		for _, line := range lines {
			if strings.Contains(line, name) {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					sessionFull := parts[0]
					killCmd := exec.CommandContext(context.Background(), "screen", "-S", sessionFull, "-X", "quit")

					return killCmd.Run()
				}
			}
		}

		return fmt.Errorf("session not found: %s", name)
	default:
		return fmt.Errorf("unsupported session type: %s", m.sessionType)
	}
}

// AttachToSession opens a new terminal window attached to the session
func (m *Manager) AttachToSession(name string) error {
	if !m.IsAvailable() {
		return fmt.Errorf("no terminal multiplexer available")
	}

	// Check if session exists
	exists, err := m.HasSession(name)
	if err != nil {
		return fmt.Errorf("failed to check session: %w", err)
	}

	if !exists {
		return fmt.Errorf("session not found: %s", name)
	}

	// Build attach command
	var attachCmd string

	switch m.sessionType {
	case TypeTmux:
		attachCmd = fmt.Sprintf("tmux attach -t %s", name)
	case TypeScreen:
		attachCmd = fmt.Sprintf("screen -r %s", name)
	default:
		return fmt.Errorf("unsupported session type: %s", m.sessionType)
	}

	// Detect terminal and open new window
	return openTerminalWindow(attachCmd)
}

// openTerminalWindow opens a new terminal window running the specified command
func openTerminalWindow(command string) error {
	termProgram := os.Getenv("TERM_PROGRAM")

	switch termProgram {
	case "iTerm.app":
		return openITermWindow(command)
	case "Apple_Terminal":
		return openTerminalAppWindow(command)
	default:
		// Try iTerm first, fall back to Terminal.app
		if err := openITermWindow(command); err != nil {
			return openTerminalAppWindow(command)
		}

		return nil
	}
}

// openITermWindow opens a new iTerm2 window
func openITermWindow(command string) error {
	script := fmt.Sprintf(`
		tell application "iTerm"
			create window with default profile
			tell current session of current window
				write text "%s"
			end tell
		end tell
	`, escapeAppleScript(command))

	cmd := exec.CommandContext(context.Background(), "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open iTerm window: %w", err)
	}

	return nil
}

// openTerminalAppWindow opens a new Terminal.app window
func openTerminalAppWindow(command string) error {
	script := fmt.Sprintf(`
		tell application "Terminal"
			do script "%s"
			activate
		end tell
	`, escapeAppleScript(command))

	cmd := exec.CommandContext(context.Background(), "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open Terminal window: %w", err)
	}

	return nil
}

// GenerateSessionName creates a session name from a branch name
func GenerateSessionName(branchName string) string {
	// Remove work/ prefix if present
	name := strings.TrimPrefix(branchName, "work/")

	// Sanitize: replace slashes and spaces with hyphens
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")

	return "auto-worktree-" + name
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// escapeShellArg escapes a single shell argument
func escapeShellArg(arg string) string {
	// Simple escaping: wrap in single quotes and escape existing single quotes
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

// escapeShellArgs escapes multiple shell arguments
func escapeShellArgs(args []string) []string {
	escaped := make([]string, len(args))

	for i, arg := range args {
		escaped[i] = escapeShellArg(arg)
	}

	return escaped
}

// escapeAppleScript escapes a string for use in AppleScript
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")

	return s
}
