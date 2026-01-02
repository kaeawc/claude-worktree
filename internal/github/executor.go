package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitHubExecutor defines the interface for executing gh CLI commands
type GitHubExecutor interface {
	// Execute runs a gh command and returns the output
	Execute(args ...string) (string, error)
	// ExecuteInDir runs a gh command in a specific directory
	ExecuteInDir(dir string, args ...string) (string, error)
}

// RealGitHubExecutor executes actual gh commands via exec.Command
type RealGitHubExecutor struct{}

// NewGitHubExecutor creates a new real GitHub executor for production use
func NewGitHubExecutor() GitHubExecutor {
	return &RealGitHubExecutor{}
}

// Execute runs a gh command and returns the output
func (e *RealGitHubExecutor) Execute(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ExecuteInDir runs a gh command in a specific directory
func (e *RealGitHubExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh %s failed in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// FakeGitHubExecutor is a fake implementation for testing
type FakeGitHubExecutor struct {
	// Commands records all executed commands for verification
	Commands [][]string
	// Responses maps command strings to their responses
	Responses map[string]string
	// Errors maps command strings to errors
	Errors map[string]error
	// DefaultResponse is returned when no specific response is configured
	DefaultResponse string
}

// NewFakeGitHubExecutor creates a new fake GitHub executor for testing
func NewFakeGitHubExecutor() *FakeGitHubExecutor {
	return &FakeGitHubExecutor{
		Commands:  [][]string{},
		Responses: make(map[string]string),
		Errors:    make(map[string]error),
	}
}

// Execute records the command and returns a configured response
func (e *FakeGitHubExecutor) Execute(args ...string) (string, error) {
	e.Commands = append(e.Commands, args)
	key := strings.Join(args, " ")

	if err, ok := e.Errors[key]; ok {
		return "", err
	}

	if resp, ok := e.Responses[key]; ok {
		return resp, nil
	}

	return e.DefaultResponse, nil
}

// ExecuteInDir records the command and returns a configured response
func (e *FakeGitHubExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
	// Record with directory context
	cmdWithDir := append([]string{"[in:" + dir + "]"}, args...)
	e.Commands = append(e.Commands, cmdWithDir)

	key := strings.Join(args, " ")

	if err, ok := e.Errors[key]; ok {
		return "", err
	}

	if resp, ok := e.Responses[key]; ok {
		return resp, nil
	}

	return e.DefaultResponse, nil
}

// SetResponse configures a response for a specific command
func (e *FakeGitHubExecutor) SetResponse(command string, response string) {
	e.Responses[command] = response
}

// SetError configures an error for a specific command
func (e *FakeGitHubExecutor) SetError(command string, err error) {
	e.Errors[command] = err
}

// GetCommandCount returns the number of commands executed
func (e *FakeGitHubExecutor) GetCommandCount() int {
	return len(e.Commands)
}

// GetLastCommand returns the last executed command, or nil if none
func (e *FakeGitHubExecutor) GetLastCommand() []string {
	if len(e.Commands) == 0 {
		return nil
	}
	return e.Commands[len(e.Commands)-1]
}

// Reset clears all recorded commands and responses
func (e *FakeGitHubExecutor) Reset() {
	e.Commands = [][]string{}
	e.Responses = make(map[string]string)
	e.Errors = make(map[string]error)
	e.DefaultResponse = ""
}
