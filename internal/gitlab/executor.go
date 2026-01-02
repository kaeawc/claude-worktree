package gitlab

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitLabExecutor defines the interface for executing glab CLI commands
type GitLabExecutor interface {
	// Execute runs a glab command and returns the output
	Execute(args ...string) (string, error)
	// ExecuteInDir runs a glab command in a specific directory
	ExecuteInDir(dir string, args ...string) (string, error)
}

// RealGitLabExecutor executes actual glab commands via exec.Command
type RealGitLabExecutor struct{}

// NewGitLabExecutor creates a new real GitLab executor for production use
func NewGitLabExecutor() GitLabExecutor {
	return &RealGitLabExecutor{}
}

// Execute runs a glab command and returns the output
func (e *RealGitLabExecutor) Execute(args ...string) (string, error) {
	cmd := exec.Command("glab", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("glab %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ExecuteInDir runs a glab command in a specific directory
func (e *RealGitLabExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("glab", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("glab %s failed in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// FakeGitLabExecutor is a fake implementation for testing
type FakeGitLabExecutor struct {
	// Commands records all executed commands for verification
	Commands [][]string
	// Responses maps command strings to their responses
	Responses map[string]string
	// Errors maps command strings to errors
	Errors map[string]error
	// DefaultResponse is returned when no specific response is configured
	DefaultResponse string
}

// NewFakeGitLabExecutor creates a new fake GitLab executor for testing
func NewFakeGitLabExecutor() *FakeGitLabExecutor {
	return &FakeGitLabExecutor{
		Commands:  [][]string{},
		Responses: make(map[string]string),
		Errors:    make(map[string]error),
	}
}

// Execute records the command and returns a configured response
func (e *FakeGitLabExecutor) Execute(args ...string) (string, error) {
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
func (e *FakeGitLabExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
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
func (e *FakeGitLabExecutor) SetResponse(command string, response string) {
	e.Responses[command] = response
}

// SetError configures an error for a specific command
func (e *FakeGitLabExecutor) SetError(command string, err error) {
	e.Errors[command] = err
}

// GetCommandCount returns the number of commands executed
func (e *FakeGitLabExecutor) GetCommandCount() int {
	return len(e.Commands)
}

// GetLastCommand returns the last executed command, or nil if none
func (e *FakeGitLabExecutor) GetLastCommand() []string {
	if len(e.Commands) == 0 {
		return nil
	}
	return e.Commands[len(e.Commands)-1]
}

// Reset clears all recorded commands and responses
func (e *FakeGitLabExecutor) Reset() {
	e.Commands = [][]string{}
	e.Responses = make(map[string]string)
	e.Errors = make(map[string]error)
	e.DefaultResponse = ""
}
