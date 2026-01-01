package git_test

import (
	"fmt"
	"log"
	"os"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// ExampleConfig_basic demonstrates basic configuration operations
func ExampleConfig_basic() {
	// Create a repository instance (assumes current directory is a git repo)
	repo, err := git.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// Set configuration values
	repo.Config.Set(git.ConfigIssueProvider, "github", git.ConfigScopeLocal)
	repo.Config.Set(git.ConfigAITool, "claude", git.ConfigScopeLocal)
	repo.Config.SetBool(git.ConfigIssueAutoselect, true, git.ConfigScopeLocal)

	// Get configuration values
	provider := repo.Config.GetIssueProvider()
	aiTool := repo.Config.GetAITool()
	autoselect := repo.Config.GetIssueAutoselect()

	fmt.Printf("Provider: %s\n", provider)
	fmt.Printf("AI Tool: %s\n", aiTool)
	fmt.Printf("Autoselect: %v\n", autoselect)

	// Clean up
	repo.Config.UnsetAll(git.ConfigScopeLocal)
}

// ExampleConfig_validation demonstrates configuration validation
func ExampleConfig_validation() {
	repo, err := git.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// Validate before setting
	if err := repo.Config.Validate(git.ConfigIssueProvider, "github"); err != nil {
		fmt.Printf("Invalid: %v\n", err)
	} else {
		fmt.Println("Valid provider")
	}

	// Using SetValidated ensures validation happens automatically
	if err := repo.Config.SetValidated(git.ConfigIssueProvider, "invalid", git.ConfigScopeLocal); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Output:
	// Valid provider
	// Error: invalid issue provider: invalid (must be one of: github, gitlab, jira, linear)
}

// ExampleConfig_scopePriority demonstrates local/global scope priority
func ExampleConfig_scopePriority() {
	repo, err := git.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// Set global default
	repo.Config.Set(git.ConfigAITool, "codex", git.ConfigScopeGlobal)

	// Set local override
	repo.Config.Set(git.ConfigAITool, "claude", git.ConfigScopeLocal)

	// ConfigScopeAuto checks local first, then falls back to global
	tool, _ := repo.Config.Get(git.ConfigAITool, git.ConfigScopeAuto)
	fmt.Printf("Tool (auto): %s\n", tool) // Will use local value

	// Get only local
	localTool, _ := repo.Config.Get(git.ConfigAITool, git.ConfigScopeLocal)
	fmt.Printf("Tool (local): %s\n", localTool)

	// Get only global
	globalTool, _ := repo.Config.Get(git.ConfigAITool, git.ConfigScopeGlobal)
	fmt.Printf("Tool (global): %s\n", globalTool)

	// Clean up
	repo.Config.Unset(git.ConfigAITool, git.ConfigScopeLocal)
	repo.Config.Unset(git.ConfigAITool, git.ConfigScopeGlobal)
}

// ExampleConfig_providerSpecific demonstrates provider-specific configuration
func ExampleConfig_providerSpecific() {
	repo, err := git.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// Set provider
	repo.Config.Set(git.ConfigIssueProvider, "jira", git.ConfigScopeLocal)

	// Configure JIRA-specific settings
	repo.Config.Set(git.ConfigJiraServer, "https://jira.example.com", git.ConfigScopeLocal)
	repo.Config.Set(git.ConfigJiraProject, "PROJ", git.ConfigScopeLocal)

	// Retrieve settings
	jiraServer, _ := repo.Config.Get(git.ConfigJiraServer, git.ConfigScopeAuto)
	jiraProject, _ := repo.Config.Get(git.ConfigJiraProject, git.ConfigScopeAuto)

	fmt.Printf("JIRA Server: %s\n", jiraServer)
	fmt.Printf("JIRA Project: %s\n", jiraProject)

	// Clean up
	repo.Config.UnsetAll(git.ConfigScopeLocal)
}

// ExampleConfig_defaults demonstrates using default values
func ExampleConfig_defaults() {
	repo, err := git.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// GetWithDefault returns default if key not set
	provider := repo.Config.GetWithDefault(git.ConfigIssueProvider, "github", git.ConfigScopeAuto)
	fmt.Printf("Provider (with default): %s\n", provider)

	// GetBoolWithDefault for boolean configs
	runHooks := repo.Config.GetBoolWithDefault(git.ConfigRunHooks, true, git.ConfigScopeAuto)
	fmt.Printf("Run Hooks (default): %v\n", runHooks)

	// These helper methods have built-in defaults
	fmt.Printf("GetRunHooks(): %v\n", repo.Config.GetRunHooks())             // defaults to true
	fmt.Printf("GetFailOnHookError(): %v\n", repo.Config.GetFailOnHookError()) // defaults to false
}

// ExampleConfig_directGitCommands shows compatibility with git config commands
func ExampleConfig_directGitCommands() {
	// Values set via the Config API can be read with git config
	repo, err := git.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// Set via Config API
	repo.Config.Set(git.ConfigIssueProvider, "github", git.ConfigScopeLocal)

	// Can be read directly with git command:
	// $ git config --get auto-worktree.issue-provider
	// github

	// Similarly, values set via git command can be read via Config API
	// $ git config --local auto-worktree.ai-tool claude

	aiTool := repo.Config.GetAITool()
	fmt.Printf("AI Tool: %s\n", aiTool)

	repo.Config.UnsetAll(git.ConfigScopeLocal)
}

func init() {
	// Suppress output for examples that don't have explicit Output comments
	// This is needed because these examples interact with the actual git config
	// and the output would vary based on the test environment
	os.Setenv("GO_TEST_EXAMPLE", "1")
}
