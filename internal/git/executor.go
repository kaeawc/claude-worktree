package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitExecutor defines the interface for executing git commands
type GitExecutor interface {
	// Execute runs a git command and returns the output
	Execute(args ...string) (string, error)
	// ExecuteInDir runs a git command in a specific directory
	ExecuteInDir(dir string, args ...string) (string, error)
}

// RealGitExecutor executes actual git commands via exec.Command
type RealGitExecutor struct{}

// NewGitExecutor creates a new real git executor for production use
func NewGitExecutor() GitExecutor {
	return &RealGitExecutor{}
}

// Execute runs a git command and returns the output
func (e *RealGitExecutor) Execute(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ExecuteInDir runs a git command in a specific directory
func (e *RealGitExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// FakeGitExecutor is a fake implementation for testing
type FakeGitExecutor struct {
	// Commands records all executed commands for verification
	Commands [][]string
	// Responses maps command strings to their responses
	Responses map[string]string
	// Errors maps command strings to errors
	Errors map[string]error
	// DefaultResponse is returned when no specific response is configured
	DefaultResponse string
}

// NewFakeGitExecutor creates a new fake git executor for testing
func NewFakeGitExecutor() *FakeGitExecutor {
	return &FakeGitExecutor{
		Commands:  [][]string{},
		Responses: make(map[string]string),
		Errors:    make(map[string]error),
	}
}

// Execute records the command and returns a configured response
func (e *FakeGitExecutor) Execute(args ...string) (string, error) {
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
func (e *FakeGitExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
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
func (e *FakeGitExecutor) SetResponse(command string, response string) {
	e.Responses[command] = response
}

// SetError configures an error for a specific command
func (e *FakeGitExecutor) SetError(command string, err error) {
	e.Errors[command] = err
}

// GetCommandCount returns the number of commands executed
func (e *FakeGitExecutor) GetCommandCount() int {
	return len(e.Commands)
}

// GetLastCommand returns the last executed command, or nil if none
func (e *FakeGitExecutor) GetLastCommand() []string {
	if len(e.Commands) == 0 {
		return nil
	}
	return e.Commands[len(e.Commands)-1]
}

// Reset clears all recorded commands and responses
func (e *FakeGitExecutor) Reset() {
	e.Commands = [][]string{}
	e.Responses = make(map[string]string)
	e.Errors = make(map[string]error)
	e.DefaultResponse = ""
}
