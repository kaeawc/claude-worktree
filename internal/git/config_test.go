package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)
	if config == nil {
		t.Fatal("NewConfig returned nil")
	}
	if config.RootPath != repoPath {
		t.Errorf("Expected RootPath=%s, got %s", repoPath, config.RootPath)
	}
}

func TestConfig_SetAndGet(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	tests := []struct {
		name  string
		key   string
		value string
		scope ConfigScope
	}{
		{"local config", "test.key1", "value1", ConfigScopeLocal},
		{"global config", "test.key2", "value2", ConfigScopeGlobal},
		{"issue provider", ConfigIssueProvider, "github", ConfigScopeLocal},
		{"ai tool", ConfigAITool, "claude", ConfigScopeLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the config
			err := config.Set(tt.key, tt.value, tt.scope)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			// Get the config
			got, err := config.Get(tt.key, tt.scope)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			if got != tt.value {
				t.Errorf("Expected value=%s, got %s", tt.value, got)
			}

			// Cleanup
			config.Unset(tt.key, tt.scope)
		})
	}
}

func TestConfig_GetWithDefault(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Test with non-existent key
	value := config.GetWithDefault("nonexistent.key", "default-value", ConfigScopeLocal)
	if value != "default-value" {
		t.Errorf("Expected default value, got %s", value)
	}

	// Test with existing key
	config.Set("test.key", "actual-value", ConfigScopeLocal)
	value = config.GetWithDefault("test.key", "default-value", ConfigScopeLocal)
	if value != "actual-value" {
		t.Errorf("Expected actual value, got %s", value)
	}

	config.Unset("test.key", ConfigScopeLocal)
}

func TestConfig_GetBool(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	tests := []struct {
		name     string
		key      string
		setValue string
		want     bool
		wantErr  bool
	}{
		{"true value", "test.bool1", "true", true, false},
		{"false value", "test.bool2", "false", false, false},
		{"yes value", "test.bool3", "yes", true, false},
		{"no value", "test.bool4", "no", false, false},
		{"1 value", "test.bool5", "1", true, false},
		{"0 value", "test.bool6", "0", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the config value
			err := config.Set(tt.key, tt.setValue, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			// Get as boolean
			got, err := config.GetBool(tt.key, ConfigScopeLocal)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}

			// Cleanup
			config.Unset(tt.key, ConfigScopeLocal)
		})
	}
}

func TestConfig_GetBoolNotFound(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	_, err := config.GetBool("nonexistent.bool", ConfigScopeLocal)
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

func TestConfig_SetBool(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	tests := []struct {
		name  string
		key   string
		value bool
	}{
		{"set true", "test.bool.true", true},
		{"set false", "test.bool.false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.SetBool(tt.key, tt.value, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("SetBool failed: %v", err)
			}

			got, err := config.GetBool(tt.key, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("GetBool failed: %v", err)
			}

			if got != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, got)
			}

			config.Unset(tt.key, ConfigScopeLocal)
		})
	}
}

func TestConfig_GetBoolWithDefault(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Test with non-existent key
	value := config.GetBoolWithDefault("nonexistent.bool", true, ConfigScopeLocal)
	if value != true {
		t.Errorf("Expected default true, got %v", value)
	}

	// Test with existing key
	config.SetBool("test.bool", false, ConfigScopeLocal)
	value = config.GetBoolWithDefault("test.bool", true, ConfigScopeLocal)
	if value != false {
		t.Errorf("Expected false, got %v", value)
	}

	config.Unset("test.bool", ConfigScopeLocal)
}

func TestConfig_Unset(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Set a value
	config.Set("test.unset", "value", ConfigScopeLocal)

	// Verify it exists
	value, _ := config.Get("test.unset", ConfigScopeLocal)
	if value != "value" {
		t.Errorf("Expected value to be set")
	}

	// Unset it
	err := config.Unset("test.unset", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Unset failed: %v", err)
	}

	// Verify it's gone
	value, _ = config.Get("test.unset", ConfigScopeLocal)
	if value != "" {
		t.Errorf("Expected empty value after unset, got %s", value)
	}
}

func TestConfig_AutoScope(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Set global value
	config.Set("test.auto", "global-value", ConfigScopeGlobal)

	// Get with auto scope - should get global
	value, err := config.Get("test.auto", ConfigScopeAuto)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "global-value" {
		t.Errorf("Expected global-value, got %s", value)
	}

	// Set local value
	config.Set("test.auto", "local-value", ConfigScopeLocal)

	// Get with auto scope - should get local (overrides global)
	value, err = config.Get("test.auto", ConfigScopeAuto)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "local-value" {
		t.Errorf("Expected local-value, got %s", value)
	}

	// Cleanup
	config.Unset("test.auto", ConfigScopeLocal)
	config.Unset("test.auto", ConfigScopeGlobal)
}

func TestConfig_Validate(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		// Valid issue providers
		{"valid github", ConfigIssueProvider, "github", false},
		{"valid gitlab", ConfigIssueProvider, "gitlab", false},
		{"valid jira", ConfigIssueProvider, "jira", false},
		{"valid linear", ConfigIssueProvider, "linear", false},
		{"invalid provider", ConfigIssueProvider, "invalid", true},

		// Valid AI tools
		{"valid claude", ConfigAITool, "claude", false},
		{"valid codex", ConfigAITool, "codex", false},
		{"valid gemini", ConfigAITool, "gemini", false},
		{"valid jules", ConfigAITool, "jules", false},
		{"valid skip", ConfigAITool, "skip", false},
		{"invalid ai tool", ConfigAITool, "invalid", true},

		// Boolean values
		{"valid bool true", ConfigIssueAutoselect, "true", false},
		{"valid bool false", ConfigIssueAutoselect, "false", false},
		{"invalid bool", ConfigIssueAutoselect, "yes", true},

		// No validation for other keys
		{"no validation", ConfigJiraServer, "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Validate(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SetValidated(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Test valid value
	err := config.SetValidated(ConfigIssueProvider, "github", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("SetValidated failed for valid value: %v", err)
	}

	value, _ := config.Get(ConfigIssueProvider, ConfigScopeLocal)
	if value != "github" {
		t.Errorf("Expected github, got %s", value)
	}

	// Test invalid value
	err = config.SetValidated(ConfigIssueProvider, "invalid", ConfigScopeLocal)
	if err == nil {
		t.Error("Expected error for invalid value, got nil")
	}

	config.Unset(ConfigIssueProvider, ConfigScopeLocal)
}

func TestConfig_HelperMethods(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Test GetIssueProvider
	config.Set(ConfigIssueProvider, "github", ConfigScopeLocal)
	if config.GetIssueProvider() != "github" {
		t.Error("GetIssueProvider failed")
	}

	// Test GetAITool
	config.Set(ConfigAITool, "claude", ConfigScopeLocal)
	if config.GetAITool() != "claude" {
		t.Error("GetAITool failed")
	}

	// Test GetIssueAutoselect
	config.SetBool(ConfigIssueAutoselect, true, ConfigScopeLocal)
	if !config.GetIssueAutoselect() {
		t.Error("GetIssueAutoselect failed")
	}

	// Test GetPRAutoselect
	config.SetBool(ConfigPRAutoselect, true, ConfigScopeLocal)
	if !config.GetPRAutoselect() {
		t.Error("GetPRAutoselect failed")
	}

	// Test GetRunHooks (default true)
	if !config.GetRunHooks() {
		t.Error("GetRunHooks should default to true")
	}

	// Test GetFailOnHookError (default false)
	if config.GetFailOnHookError() {
		t.Error("GetFailOnHookError should default to false")
	}

	// Cleanup
	config.UnsetAll(ConfigScopeLocal)
}

func TestConfig_UnsetAll(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Set multiple configs
	config.Set(ConfigIssueProvider, "github", ConfigScopeLocal)
	config.Set(ConfigAITool, "claude", ConfigScopeLocal)
	config.SetBool(ConfigIssueAutoselect, true, ConfigScopeLocal)
	config.Set(ConfigJiraServer, "https://jira.example.com", ConfigScopeLocal)

	// Verify they're set
	if config.GetIssueProvider() == "" {
		t.Error("Config not set")
	}

	// Unset all
	err := config.UnsetAll(ConfigScopeLocal)
	if err != nil {
		t.Fatalf("UnsetAll failed: %v", err)
	}

	// Verify they're gone
	if config.GetIssueProvider() != "" {
		t.Error("Config not unset")
	}
	if config.GetAITool() != "" {
		t.Error("Config not unset")
	}
}

func TestConfig_InvalidScope(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Test invalid scope for Get
	_, err := config.Get("test.key", ConfigScope("invalid"))
	if err == nil {
		t.Error("Expected error for invalid scope in Get")
	}

	// Test invalid scope for Set
	err = config.Set("test.key", "value", ConfigScope("invalid"))
	if err == nil {
		t.Error("Expected error for invalid scope in Set")
	}
}

func TestConfig_NonGitRepository(t *testing.T) {
	// Create a temporary directory that's not a git repository
	tmpDir, err := os.MkdirTemp("", "auto-worktree-non-git-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := NewConfig(tmpDir)

	// Operations should fail gracefully
	_, err = config.Get("test.key", ConfigScopeLocal)
	if err == nil {
		t.Error("Expected error when operating on non-git repository")
	}
}

func TestConfig_ProviderSpecificConfigs(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	tests := []struct {
		key   string
		value string
	}{
		{ConfigJiraServer, "https://jira.example.com"},
		{ConfigJiraProject, "PROJ"},
		{ConfigGitLabServer, "https://gitlab.example.com"},
		{ConfigGitLabProject, "group/project"},
		{ConfigLinearTeam, "ENG"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := config.Set(tt.key, tt.value, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			got, err := config.Get(tt.key, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			if got != tt.value {
				t.Errorf("Expected %s, got %s", tt.value, got)
			}

			config.Unset(tt.key, ConfigScopeLocal)
		})
	}
}

func TestConfig_AutoScopeSet(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Setting with auto scope should default to local
	err := config.Set("test.auto-set", "value", ConfigScopeAuto)
	if err != nil {
		t.Fatalf("Set with auto scope failed: %v", err)
	}

	// Should be able to get it with local scope
	value, err := config.Get("test.auto-set", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "value" {
		t.Errorf("Expected value, got %s", value)
	}

	config.Unset("test.auto-set", ConfigScopeLocal)
}

func TestConfig_EmptyValue(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Set an empty value
	err := config.Set("test.empty", "", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Set empty value failed: %v", err)
	}

	// Get should return empty string
	value, err := config.Get("test.empty", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string, got %s", value)
	}

	config.Unset("test.empty", ConfigScopeLocal)
}

// TestConfig_RealGitBehavior tests that our config operations match git's behavior
func TestConfig_RealGitBehavior(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	config := NewConfig(repoPath)

	// Set a value using our config
	err := config.Set("test.behavior", "our-value", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Read it directly with git
	cmd := exec.Command("git", "config", "--local", "--get", "test.behavior")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Direct git command failed: %v", err)
	}

	gitValue := strings.TrimSpace(string(output))
	if gitValue != "our-value" {
		t.Errorf("Git returned different value: %s", gitValue)
	}

	config.Unset("test.behavior", ConfigScopeLocal)
}
