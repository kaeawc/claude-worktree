package git

import (
	"fmt"
	"strconv"
	"strings"
)

// Configuration key constants
const (
	// Issue provider configuration
	ConfigIssueProvider = "auto-worktree.issue-provider"

	// AI tool configuration
	ConfigAITool          = "auto-worktree.ai-tool"
	ConfigIssueAutoselect = "auto-worktree.issue-autoselect"
	ConfigPRAutoselect    = "auto-worktree.pr-autoselect"

	// JIRA provider configuration
	ConfigJiraServer  = "auto-worktree.jira-server"
	ConfigJiraProject = "auto-worktree.jira-project"

	// GitLab provider configuration
	ConfigGitLabServer  = "auto-worktree.gitlab-server"
	ConfigGitLabProject = "auto-worktree.gitlab-project"

	// Linear provider configuration
	ConfigLinearTeam = "auto-worktree.linear-team"

	// Hook configuration
	ConfigRunHooks        = "auto-worktree.run-hooks"
	ConfigFailOnHookError = "auto-worktree.fail-on-hook-error"
	ConfigCustomHooks     = "auto-worktree.custom-hooks"

	// Issue template configuration
	ConfigIssueTemplatesDir      = "auto-worktree.issue-templates-dir"
	ConfigIssueTemplatesDisabled = "auto-worktree.issue-templates-disabled"
	ConfigIssueTemplatesNoPrompt = "auto-worktree.issue-templates-no-prompt"
	ConfigIssueTemplatesDetected = "auto-worktree.issue-templates-detected"

	// Environment setup configuration
	ConfigAutoInstall    = "auto-worktree.auto-install"
	ConfigPackageManager = "auto-worktree.package-manager"

	// Tmux session management configuration
	ConfigTmuxEnabled        = "auto-worktree.tmux-enabled"
	ConfigTmuxAutoInstall    = "auto-worktree.tmux-auto-install"
	ConfigTmuxLayout         = "auto-worktree.tmux-layout"
	ConfigTmuxShell          = "auto-worktree.tmux-shell"
	ConfigTmuxWindowCount    = "auto-worktree.tmux-window-count"
	ConfigTmuxIdleThreshold  = "auto-worktree.tmux-idle-threshold"
	ConfigTmuxMetadataDir    = "auto-worktree.tmux-metadata-dir"
	ConfigTmuxLogCommands    = "auto-worktree.tmux-log-commands"
	ConfigTmuxPostCreateHook = "auto-worktree.tmux-post-create-hook"
	ConfigTmuxPostResumeHook = "auto-worktree.tmux-post-resume-hook"
	ConfigTmuxPreKillHook    = "auto-worktree.tmux-pre-kill-hook"
)

// Valid values for specific configuration keys
var (
	ValidIssueProviders = []string{"github", "gitlab", "jira", "linear"}
	ValidAITools        = []string{"claude", "codex", "gemini", "jules", "skip"}
)

// ConfigScope represents the scope of a git config operation
type ConfigScope string

const (
	// ConfigScopeLocal uses --local flag (repository-specific)
	ConfigScopeLocal ConfigScope = "local"
	// ConfigScopeGlobal uses --global flag (user-wide)
	ConfigScopeGlobal ConfigScope = "global"
	// ConfigScopeAuto checks local first, then falls back to global
	ConfigScopeAuto ConfigScope = "auto"
)

// Config provides git config operations for a repository
type Config struct {
	// RootPath is the repository root directory
	RootPath string
	// executor is the GitExecutor used to run git commands
	executor GitExecutor
}

// NewConfig creates a new Config instance with a real git executor
func NewConfig(rootPath string) *Config {
	return &Config{
		RootPath: rootPath,
		executor: NewGitExecutor(),
	}
}

// NewConfigWithExecutor creates a new Config instance with a custom executor (for testing)
func NewConfigWithExecutor(rootPath string, executor GitExecutor) *Config {
	return &Config{
		RootPath: rootPath,
		executor: executor,
	}
}

// Get retrieves a configuration value
// If scope is ConfigScopeAuto, it checks local first, then global
func (c *Config) Get(key string, scope ConfigScope) (string, error) {
	var args []string

	switch scope {
	case ConfigScopeLocal:
		args = []string{"config", "--local", "--get", key}
	case ConfigScopeGlobal:
		args = []string{"config", "--global", "--get", key}
	case ConfigScopeAuto:
		// Try local first
		value, err := c.Get(key, ConfigScopeLocal)
		if err == nil && value != "" {
			return value, nil
		}
		// Fall back to global
		return c.Get(key, ConfigScopeGlobal)
	default:
		return "", fmt.Errorf("invalid config scope: %s", scope)
	}

	output, err := c.executor.ExecuteInDir(c.RootPath, args...)
	if err != nil {
		// git config returns exit code 1 if key not found - check error message
		if strings.Contains(err.Error(), "failed") {
			return "", nil
		}
		return "", fmt.Errorf("failed to get config %s: %w", key, err)
	}

	return output, nil
}

// GetWithDefault retrieves a configuration value, returning defaultValue if not set
func (c *Config) GetWithDefault(key string, defaultValue string, scope ConfigScope) string {
	value, err := c.Get(key, scope)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

// Set sets a configuration value
func (c *Config) Set(key, value string, scope ConfigScope) error {
	if scope == ConfigScopeAuto {
		// Default to local for auto scope when setting
		scope = ConfigScopeLocal
	}

	var args []string
	switch scope {
	case ConfigScopeLocal:
		args = []string{"config", "--local", key, value}
	case ConfigScopeGlobal:
		args = []string{"config", "--global", key, value}
	default:
		return fmt.Errorf("invalid config scope: %s", scope)
	}

	if _, err := c.executor.ExecuteInDir(c.RootPath, args...); err != nil {
		return fmt.Errorf("failed to set config %s=%s: %w", key, value, err)
	}

	return nil
}

// Unset removes a configuration value
func (c *Config) Unset(key string, scope ConfigScope) error {
	if scope == ConfigScopeAuto {
		// Unset from both local and global
		_ = c.Unset(key, ConfigScopeLocal)
		_ = c.Unset(key, ConfigScopeGlobal)
		return nil
	}

	var args []string
	switch scope {
	case ConfigScopeLocal:
		args = []string{"config", "--local", "--unset", key}
	case ConfigScopeGlobal:
		args = []string{"config", "--global", "--unset", key}
	default:
		return fmt.Errorf("invalid config scope: %s", scope)
	}

	_, _ = c.executor.ExecuteInDir(c.RootPath, args...) // Ignore errors - key might not exist
	return nil
}

// GetBool retrieves a boolean configuration value
func (c *Config) GetBool(key string, scope ConfigScope) (bool, error) {
	var args []string

	switch scope {
	case ConfigScopeLocal:
		args = []string{"config", "--local", "--get", "--bool", key}
	case ConfigScopeGlobal:
		args = []string{"config", "--global", "--get", "--bool", key}
	case ConfigScopeAuto:
		// Try local first
		value, err := c.GetBool(key, ConfigScopeLocal)
		if err == nil {
			return value, nil
		}
		// Fall back to global
		return c.GetBool(key, ConfigScopeGlobal)
	default:
		return false, fmt.Errorf("invalid config scope: %s", scope)
	}

	output, err := c.executor.ExecuteInDir(c.RootPath, args...)
	if err != nil {
		// git config returns exit code 1 if key not found
		if strings.Contains(err.Error(), "failed") {
			return false, fmt.Errorf("config key not found: %s", key)
		}
		return false, fmt.Errorf("failed to get config %s: %w", key, err)
	}

	return output == "true", nil
}

// GetBoolWithDefault retrieves a boolean configuration value, returning defaultValue if not set
func (c *Config) GetBoolWithDefault(key string, defaultValue bool, scope ConfigScope) bool {
	value, err := c.GetBool(key, scope)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetIntWithDefault retrieves an integer configuration value, returning defaultValue if not set or invalid
func (c *Config) GetIntWithDefault(key string, defaultValue int, scope ConfigScope) int {
	value, err := c.Get(key, scope)
	if err != nil || value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil || intValue <= 0 {
		return defaultValue
	}

	return intValue
}

// SetBool sets a boolean configuration value
func (c *Config) SetBool(key string, value bool, scope ConfigScope) error {
	return c.Set(key, strconv.FormatBool(value), scope)
}

// Validate checks if a configuration value is valid for a given key
func (c *Config) Validate(key, value string) error {
	switch key {
	case ConfigIssueProvider:
		for _, valid := range ValidIssueProviders {
			if value == valid {
				return nil
			}
		}
		return fmt.Errorf("invalid issue provider: %s (must be one of: %s)", value, strings.Join(ValidIssueProviders, ", "))

	case ConfigAITool:
		for _, valid := range ValidAITools {
			if value == valid {
				return nil
			}
		}
		return fmt.Errorf("invalid AI tool: %s (must be one of: %s)", value, strings.Join(ValidAITools, ", "))

	case ConfigIssueAutoselect, ConfigPRAutoselect, ConfigRunHooks, ConfigFailOnHookError,
		ConfigIssueTemplatesDisabled, ConfigIssueTemplatesNoPrompt, ConfigIssueTemplatesDetected,
		ConfigAutoInstall:
		// These should be boolean values
		if value != "true" && value != "false" {
			return fmt.Errorf("invalid boolean value: %s (must be 'true' or 'false')", value)
		}
		return nil

	// No specific validation for other keys
	default:
		return nil
	}
}

// SetValidated sets a configuration value after validating it
func (c *Config) SetValidated(key, value string, scope ConfigScope) error {
	if err := c.Validate(key, value); err != nil {
		return err
	}
	return c.Set(key, value, scope)
}

// GetIssueProvider returns the configured issue provider
func (c *Config) GetIssueProvider() string {
	return c.GetWithDefault(ConfigIssueProvider, "", ConfigScopeAuto)
}

// GetAITool returns the configured AI tool
func (c *Config) GetAITool() string {
	return c.GetWithDefault(ConfigAITool, "", ConfigScopeAuto)
}

// GetIssueAutoselect returns whether issue autoselect is enabled
func (c *Config) GetIssueAutoselect() bool {
	return c.GetBoolWithDefault(ConfigIssueAutoselect, false, ConfigScopeAuto)
}

// GetPRAutoselect returns whether PR autoselect is enabled
func (c *Config) GetPRAutoselect() bool {
	return c.GetBoolWithDefault(ConfigPRAutoselect, false, ConfigScopeAuto)
}

// GetRunHooks returns whether git hooks should be run (default: true)
func (c *Config) GetRunHooks() bool {
	return c.GetBoolWithDefault(ConfigRunHooks, true, ConfigScopeAuto)
}

// GetFailOnHookError returns whether to fail on hook errors (default: false)
func (c *Config) GetFailOnHookError() bool {
	return c.GetBoolWithDefault(ConfigFailOnHookError, false, ConfigScopeAuto)
}

// GetCustomHooks returns the list of custom hooks to execute
// Parses space or comma-separated hook names from configuration
func (c *Config) GetCustomHooks() []string {
	value := c.GetWithDefault(ConfigCustomHooks, "", ConfigScopeAuto)
	if value == "" {
		return []string{}
	}

	// Replace commas with spaces for uniform parsing
	value = strings.ReplaceAll(value, ",", " ")

	// Split on whitespace and filter empty strings
	var hooks []string
	for _, hook := range strings.Fields(value) {
		if hook != "" {
			hooks = append(hooks, hook)
		}
	}

	return hooks
}

// GetAutoInstall returns whether to automatically install dependencies (default: true)
func (c *Config) GetAutoInstall() bool {
	return c.GetBoolWithDefault(ConfigAutoInstall, true, ConfigScopeAuto)
}

// GetPackageManager returns the configured package manager override
func (c *Config) GetPackageManager() string {
	return c.GetWithDefault(ConfigPackageManager, "", ConfigScopeAuto)
}

// UnsetAll removes all auto-worktree configuration
func (c *Config) UnsetAll(scope ConfigScope) error {
	keys := []string{
		ConfigIssueProvider,
		ConfigAITool,
		ConfigIssueAutoselect,
		ConfigPRAutoselect,
		ConfigJiraServer,
		ConfigJiraProject,
		ConfigGitLabServer,
		ConfigGitLabProject,
		ConfigLinearTeam,
		ConfigRunHooks,
		ConfigFailOnHookError,
		ConfigCustomHooks,
		ConfigIssueTemplatesDir,
		ConfigIssueTemplatesDisabled,
		ConfigIssueTemplatesNoPrompt,
		ConfigIssueTemplatesDetected,
		ConfigAutoInstall,
		ConfigPackageManager,
	}

	for _, key := range keys {
		_ = c.Unset(key, scope)
	}

	return nil
}
