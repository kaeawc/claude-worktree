# Configuration Management Implementation

This document describes the implementation of the configuration management system for the auto-worktree Go rewrite (Issue #74).

## Overview

The configuration system uses `git config` to store repository-specific and global settings, matching the behavior of the existing `aw.sh` script. All settings are stored with the `auto-worktree.*` prefix.

## Files Created

- `internal/git/config.go` - Core configuration implementation
- `internal/git/config_test.go` - Comprehensive test suite (21 test cases)
- `internal/git/config_example_test.go` - Usage examples

## Files Modified

- `internal/git/repository.go` - Added `Config` field to `Repository` struct

## Features

### Configuration Scopes

The system supports three configuration scopes:

1. **Local** (`ConfigScopeLocal`) - Repository-specific settings (stored in `.git/config`)
2. **Global** (`ConfigScopeGlobal`) - User-wide settings (stored in `~/.gitconfig`)
3. **Auto** (`ConfigScopeAuto`) - Checks local first, falls back to global

### Supported Configuration Keys

All configuration keys are defined as constants in `config.go`:

#### Issue Provider Configuration
- `auto-worktree.issue-provider` - Provider type (github, gitlab, jira, linear)

#### AI Tool Configuration
- `auto-worktree.ai-tool` - AI tool name (claude, codex, gemini, jules, skip)
- `auto-worktree.issue-autoselect` - Auto-select issues (boolean)
- `auto-worktree.pr-autoselect` - Auto-select PRs (boolean)

#### Provider-Specific Settings

**JIRA:**
- `auto-worktree.jira-server` - JIRA server URL
- `auto-worktree.jira-project` - Default JIRA project key

**GitLab:**
- `auto-worktree.gitlab-server` - GitLab server URL (for self-hosted)
- `auto-worktree.gitlab-project` - Default GitLab project path

**Linear:**
- `auto-worktree.linear-team` - Default Linear team key

#### Hook Configuration
- `auto-worktree.run-hooks` - Enable/disable git hooks (default: true)
- `auto-worktree.fail-on-hook-error` - Fail on hook errors (default: false)
- `auto-worktree.custom-hooks` - Custom hooks to run

#### Issue Template Configuration
- `auto-worktree.issue-templates-dir` - Issue templates directory
- `auto-worktree.issue-templates-disabled` - Disable issue templates
- `auto-worktree.issue-templates-no-prompt` - Skip template prompts
- `auto-worktree.issue-templates-detected` - Templates detected flag

### Core Methods

#### Basic Operations
```go
// Get a configuration value
value, err := config.Get(key, scope)

// Set a configuration value
err := config.Set(key, value, scope)

// Remove a configuration value
err := config.Unset(key, scope)
```

#### Boolean Operations
```go
// Get boolean value
enabled, err := config.GetBool(key, scope)

// Set boolean value
err := config.SetBool(key, true, scope)

// Get with default fallback
enabled := config.GetBoolWithDefault(key, true, scope)
```

#### Validation
```go
// Validate a value before setting
err := config.Validate(key, value)

// Set with automatic validation
err := config.SetValidated(key, value, scope)
```

#### Helper Methods
```go
// Convenience methods with built-in defaults
provider := config.GetIssueProvider()
aiTool := config.GetAITool()
autoselect := config.GetIssueAutoselect()
runHooks := config.GetRunHooks() // defaults to true
failOnError := config.GetFailOnHookError() // defaults to false
```

#### Bulk Operations
```go
// Remove all auto-worktree configuration
err := config.UnsetAll(scope)
```

### Validation

The system validates values for specific configuration keys:

- **Issue providers**: Must be one of: github, gitlab, jira, linear
- **AI tools**: Must be one of: claude, codex, gemini, jules, skip
- **Boolean values**: Must be "true" or "false" (git also accepts yes/no, 1/0)

## Usage Examples

### Basic Usage
```go
repo, _ := git.NewRepository()

// Set configuration
repo.Config.Set(git.ConfigIssueProvider, "github", git.ConfigScopeLocal)
repo.Config.SetBool(git.ConfigIssueAutoselect, true, git.ConfigScopeLocal)

// Get configuration
provider := repo.Config.GetIssueProvider()
autoselect := repo.Config.GetIssueAutoselect()
```

### Local vs Global Configuration
```go
// Set global default
repo.Config.Set(git.ConfigAITool, "claude", git.ConfigScopeGlobal)

// Override with local setting
repo.Config.Set(git.ConfigAITool, "codex", git.ConfigScopeLocal)

// Auto scope checks local first, then global
tool, _ := repo.Config.Get(git.ConfigAITool, git.ConfigScopeAuto) // returns "codex"
```

### Provider-Specific Configuration
```go
// Configure JIRA
repo.Config.Set(git.ConfigIssueProvider, "jira", git.ConfigScopeLocal)
repo.Config.Set(git.ConfigJiraServer, "https://jira.example.com", git.ConfigScopeLocal)
repo.Config.Set(git.ConfigJiraProject, "PROJ", git.ConfigScopeLocal)
```

### Command-Line Compatibility

Settings can be managed directly with `git config`:

```bash
# Set configuration
git config auto-worktree.issue-provider github
git config auto-worktree.ai-tool claude

# Get configuration
git config --get auto-worktree.issue-provider

# List all auto-worktree settings
git config --get-regexp '^auto-worktree\.'

# Remove configuration
git config --unset auto-worktree.issue-provider
```

## Testing

The implementation includes comprehensive tests:

- **21 test cases** covering all functionality
- **100% pass rate** on all configuration tests
- Tests for local, global, and auto scopes
- Validation tests for all constrained values
- Error handling tests
- Integration tests with actual git commands

Run tests with:
```bash
go test ./internal/git -run TestConfig -v
```

## Backward Compatibility

The implementation is fully backward compatible with the existing `aw.sh` configuration:

- Uses the same `auto-worktree.*` prefix
- Stores values in the same git config locations
- Supports the same configuration keys
- Values set by `aw.sh` can be read by the Go implementation
- Values set by the Go implementation can be read by `aw.sh`

## Acceptance Criteria Status

- ✅ Settings persist per-repository
- ✅ Different repos can use different providers
- ✅ Configuration survives git operations
- ✅ Backward compatible with existing aw.sh configs
- ✅ Read/write git config values
- ✅ Repository-specific configuration
- ✅ Provider-specific config (JIRA, GitLab, Linear)
- ✅ Configuration validation
- ✅ Default values and fallbacks

## Future Enhancements

Potential improvements for future iterations:

1. Configuration migration tool (if config schema changes)
2. Configuration export/import functionality
3. Configuration templates for common setups
4. Interactive configuration wizard
5. Configuration validation on startup
