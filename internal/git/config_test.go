package git

import (
	"fmt"
	"testing"
)

func TestNewConfig(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()

	config := NewConfigWithExecutor(repoPath, fake)
	if config == nil {
		t.Fatal("NewConfig returned nil")
	}
	if config.RootPath != repoPath {
		t.Errorf("Expected RootPath=%s, got %s", repoPath, config.RootPath)
	}
}

func TestConfig_SetAndGet(t *testing.T) {
	repoPath := "/fake/repo"

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
			fake := NewFakeGitExecutor()
			config := NewConfigWithExecutor(repoPath, fake)

			// Configure fake to return the value we set
			var scopeFlag string
			if tt.scope == ConfigScopeLocal {
				scopeFlag = "--local"
			} else {
				scopeFlag = "--global"
			}
			fake.SetResponse(fmt.Sprintf("config %s --get %s", scopeFlag, tt.key), tt.value)

			// Set the config
			err := config.Set(tt.key, tt.value, tt.scope)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			// Verify Set command was executed
			if len(fake.Commands) < 1 {
				t.Fatal("Expected Set to execute a command")
			}
			setCmd := fake.Commands[0]
			expectedSetCmd := []string{"[in:" + repoPath + "]", "config", scopeFlag, tt.key, tt.value}
			if !equalStringSlices(setCmd, expectedSetCmd) {
				t.Errorf("Set command mismatch:\nGot:  %v\nWant: %v", setCmd, expectedSetCmd)
			}

			// Get the config
			got, err := config.Get(tt.key, tt.scope)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			if got != tt.value {
				t.Errorf("Expected value=%s, got %s", tt.value, got)
			}

			// Verify Get command was executed
			if len(fake.Commands) < 2 {
				t.Fatal("Expected Get to execute a command")
			}
			getCmd := fake.Commands[1]
			expectedGetCmd := []string{"[in:" + repoPath + "]", "config", scopeFlag, "--get", tt.key}
			if !equalStringSlices(getCmd, expectedGetCmd) {
				t.Errorf("Get command mismatch:\nGot:  %v\nWant: %v", getCmd, expectedGetCmd)
			}
		})
	}
}

// Helper function to compare string slices
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestConfig_GetWithDefault(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Test with non-existent key - configure fake to return error
	fake.SetError("config --local --get nonexistent.key", fmt.Errorf("failed"))
	value := config.GetWithDefault("nonexistent.key", "default-value", ConfigScopeLocal)
	if value != "default-value" {
		t.Errorf("Expected default value, got %s", value)
	}

	// Test with existing key
	fake.SetResponse("config --local --get test.key", "actual-value")
	value = config.GetWithDefault("test.key", "default-value", ConfigScopeLocal)
	if value != "actual-value" {
		t.Errorf("Expected actual value, got %s", value)
	}
}

func TestConfig_GetBool(t *testing.T) {
	repoPath := "/fake/repo"

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
			fake := NewFakeGitExecutor()
			config := NewConfigWithExecutor(repoPath, fake)

			// Configure fake to return boolean value
			var boolResponse string
			if tt.setValue == "true" || tt.setValue == "yes" || tt.setValue == "1" {
				boolResponse = "true"
			} else {
				boolResponse = "false"
			}
			fake.SetResponse(fmt.Sprintf("config --local --get --bool %s", tt.key), boolResponse)

			// Get as boolean
			got, err := config.GetBool(tt.key, ConfigScopeLocal)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}

			// Verify GetBool command was executed
			if len(fake.Commands) < 1 {
				t.Fatal("Expected GetBool to execute a command")
			}
			cmd := fake.Commands[0]
			expectedCmd := []string{"[in:" + repoPath + "]", "config", "--local", "--get", "--bool", tt.key}
			if !equalStringSlices(cmd, expectedCmd) {
				t.Errorf("GetBool command mismatch:\nGot:  %v\nWant: %v", cmd, expectedCmd)
			}
		})
	}
}

func TestConfig_GetBoolNotFound(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake to return error for non-existent key
	fake.SetError("config --local --get --bool nonexistent.bool", fmt.Errorf("failed"))

	_, err := config.GetBool("nonexistent.bool", ConfigScopeLocal)
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

func TestConfig_SetBool(t *testing.T) {
	repoPath := "/fake/repo"

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
			fake := NewFakeGitExecutor()
			config := NewConfigWithExecutor(repoPath, fake)

			// Configure fake to return the boolean value
			boolStr := "false"
			if tt.value {
				boolStr = "true"
			}
			fake.SetResponse(fmt.Sprintf("config --local --get --bool %s", tt.key), boolStr)

			err := config.SetBool(tt.key, tt.value, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("SetBool failed: %v", err)
			}

			// Verify SetBool command was executed
			if len(fake.Commands) < 1 {
				t.Fatal("Expected SetBool to execute a command")
			}
			setCmd := fake.Commands[0]
			expectedSetCmd := []string{"[in:" + repoPath + "]", "config", "--local", tt.key, boolStr}
			if !equalStringSlices(setCmd, expectedSetCmd) {
				t.Errorf("SetBool command mismatch:\nGot:  %v\nWant: %v", setCmd, expectedSetCmd)
			}

			got, err := config.GetBool(tt.key, ConfigScopeLocal)
			if err != nil {
				t.Fatalf("GetBool failed: %v", err)
			}

			if got != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, got)
			}
		})
	}
}

func TestConfig_GetBoolWithDefault(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Test with non-existent key
	fake.SetError("config --local --get --bool nonexistent.bool", fmt.Errorf("failed"))
	value := config.GetBoolWithDefault("nonexistent.bool", true, ConfigScopeLocal)
	if value != true {
		t.Errorf("Expected default true, got %v", value)
	}

	// Test with existing key
	fake.SetResponse("config --local --get --bool test.bool", "false")
	value = config.GetBoolWithDefault("test.bool", true, ConfigScopeLocal)
	if value != false {
		t.Errorf("Expected false, got %v", value)
	}
}

func TestConfig_Unset(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure initial response for Get
	fake.SetResponse("config --local --get test.unset", "value")

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

	// Verify Unset command was executed
	unsetFound := false
	for _, cmd := range fake.Commands {
		if len(cmd) >= 4 && cmd[1] == "config" && cmd[2] == "--local" && cmd[3] == "--unset" {
			unsetFound = true
			expectedCmd := []string{"[in:" + repoPath + "]", "config", "--local", "--unset", "test.unset"}
			if !equalStringSlices(cmd, expectedCmd) {
				t.Errorf("Unset command mismatch:\nGot:  %v\nWant: %v", cmd, expectedCmd)
			}
			break
		}
	}
	if !unsetFound {
		t.Error("Expected Unset to execute a command")
	}

	// Configure fake to return empty after unset
	fake.SetResponse("config --local --get test.unset", "")

	// Verify it's gone
	value, _ = config.Get("test.unset", ConfigScopeLocal)
	if value != "" {
		t.Errorf("Expected empty value after unset, got %s", value)
	}
}

func TestConfig_AutoScope(t *testing.T) {
	repoPath := "/fake/repo"

	// Test 1: Only global value set (local not found)
	fake1 := NewFakeGitExecutor()
	config1 := NewConfigWithExecutor(repoPath, fake1)

	fake1.SetResponse("config --global --get test.auto", "global-value")
	fake1.SetError("config --local --get test.auto", fmt.Errorf("failed"))

	value, err := config1.Get("test.auto", ConfigScopeAuto)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "global-value" {
		t.Errorf("Expected global-value, got %s", value)
	}

	// Test 2: Both local and global set - local should take precedence
	fake2 := NewFakeGitExecutor()
	config2 := NewConfigWithExecutor(repoPath, fake2)

	fake2.SetResponse("config --local --get test.auto", "local-value")
	fake2.SetResponse("config --global --get test.auto", "global-value")

	value, err = config2.Get("test.auto", ConfigScopeAuto)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "local-value" {
		t.Errorf("Expected local-value, got %s", value)
	}
}

func TestConfig_Validate(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

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
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake to return the value
	fake.SetResponse("config --local --get "+ConfigIssueProvider, "github")

	// Test valid value
	err := config.SetValidated(ConfigIssueProvider, "github", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("SetValidated failed for valid value: %v", err)
	}

	// Verify Set command was executed
	if len(fake.Commands) < 1 {
		t.Fatal("Expected SetValidated to execute a command")
	}
	setCmd := fake.Commands[0]
	expectedSetCmd := []string{"[in:" + repoPath + "]", "config", "--local", ConfigIssueProvider, "github"}
	if !equalStringSlices(setCmd, expectedSetCmd) {
		t.Errorf("SetValidated command mismatch:\nGot:  %v\nWant: %v", setCmd, expectedSetCmd)
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
}

func TestConfig_HelperMethods(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake responses
	fake.SetResponse("config --local --get "+ConfigIssueProvider, "github")
	fake.SetError("config --global --get "+ConfigIssueProvider, fmt.Errorf("failed"))

	fake.SetResponse("config --local --get "+ConfigAITool, "claude")
	fake.SetError("config --global --get "+ConfigAITool, fmt.Errorf("failed"))

	fake.SetResponse("config --local --get --bool "+ConfigIssueAutoselect, "true")
	fake.SetResponse("config --local --get --bool "+ConfigPRAutoselect, "true")

	// For GetRunHooks default true
	fake.SetError("config --local --get --bool "+ConfigRunHooks, fmt.Errorf("failed"))
	fake.SetError("config --global --get --bool "+ConfigRunHooks, fmt.Errorf("failed"))

	// For GetFailOnHookError default false
	fake.SetError("config --local --get --bool "+ConfigFailOnHookError, fmt.Errorf("failed"))
	fake.SetError("config --global --get --bool "+ConfigFailOnHookError, fmt.Errorf("failed"))

	// Test GetIssueProvider
	if config.GetIssueProvider() != "github" {
		t.Error("GetIssueProvider failed")
	}

	// Test GetAITool
	if config.GetAITool() != "claude" {
		t.Error("GetAITool failed")
	}

	// Test GetIssueAutoselect
	if !config.GetIssueAutoselect() {
		t.Error("GetIssueAutoselect failed")
	}

	// Test GetPRAutoselect
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
}

func TestConfig_UnsetAll(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure initial responses
	fake.SetResponse("config --local --get "+ConfigIssueProvider, "github")
	fake.SetError("config --global --get "+ConfigIssueProvider, fmt.Errorf("failed"))

	// Verify they're set
	if config.GetIssueProvider() == "" {
		t.Error("Config not set")
	}

	// Unset all
	err := config.UnsetAll(ConfigScopeLocal)
	if err != nil {
		t.Fatalf("UnsetAll failed: %v", err)
	}

	// Verify UnsetAll executed multiple unset commands
	unsetCount := 0
	for _, cmd := range fake.Commands {
		if len(cmd) >= 4 && cmd[1] == "config" && cmd[2] == "--local" && cmd[3] == "--unset" {
			unsetCount++
		}
	}
	// Should unset all the config keys defined in UnsetAll
	expectedUnsetCount := 18 // Number of keys in UnsetAll method
	if unsetCount != expectedUnsetCount {
		t.Errorf("Expected %d unset commands, got %d", expectedUnsetCount, unsetCount)
	}

	// Configure responses to return empty after unset
	fake.SetResponse("config --local --get "+ConfigIssueProvider, "")
	fake.SetResponse("config --local --get "+ConfigAITool, "")

	// Verify they're gone
	if config.GetIssueProvider() != "" {
		t.Error("Config not unset")
	}
	if config.GetAITool() != "" {
		t.Error("Config not unset")
	}
}

func TestConfig_InvalidScope(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

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
	repoPath := "/fake/non-git-repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake to return error for non-git repository (not "failed" keyword)
	fake.SetError("config --local --get test.key", fmt.Errorf("git config --local --get test.key error in /fake/non-git-repo: not a git repository"))

	// Operations should fail gracefully
	_, err := config.Get("test.key", ConfigScopeLocal)
	if err == nil {
		t.Error("Expected error when operating on non-git repository")
	}
}

func TestConfig_ProviderSpecificConfigs(t *testing.T) {
	repoPath := "/fake/repo"

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
			fake := NewFakeGitExecutor()
			config := NewConfigWithExecutor(repoPath, fake)

			// Configure fake to return the value
			fake.SetResponse(fmt.Sprintf("config --local --get %s", tt.key), tt.value)

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

			// Verify commands were executed
			if len(fake.Commands) < 2 {
				t.Fatal("Expected Set and Get to execute commands")
			}
		})
	}
}

func TestConfig_AutoScopeSet(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake to return the value
	fake.SetResponse("config --local --get test.auto-set", "value")

	// Setting with auto scope should default to local
	err := config.Set("test.auto-set", "value", ConfigScopeAuto)
	if err != nil {
		t.Fatalf("Set with auto scope failed: %v", err)
	}

	// Verify Set command used --local flag (auto defaults to local for Set)
	if len(fake.Commands) < 1 {
		t.Fatal("Expected Set to execute a command")
	}
	setCmd := fake.Commands[0]
	expectedSetCmd := []string{"[in:" + repoPath + "]", "config", "--local", "test.auto-set", "value"}
	if !equalStringSlices(setCmd, expectedSetCmd) {
		t.Errorf("Set command mismatch:\nGot:  %v\nWant: %v", setCmd, expectedSetCmd)
	}

	// Should be able to get it with local scope
	value, err := config.Get("test.auto-set", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "value" {
		t.Errorf("Expected value, got %s", value)
	}
}

func TestConfig_EmptyValue(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake to return empty value
	fake.SetResponse("config --local --get test.empty", "")

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
}

// TestConfig_RealGitBehavior tests that our config operations match git's behavior
func TestConfig_RealGitBehavior(t *testing.T) {
	repoPath := "/fake/repo"
	fake := NewFakeGitExecutor()
	config := NewConfigWithExecutor(repoPath, fake)

	// Configure fake to simulate git's behavior
	fake.SetResponse("config --local --get test.behavior", "our-value")

	// Set a value using our config
	err := config.Set("test.behavior", "our-value", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify Set command was executed
	if len(fake.Commands) < 1 {
		t.Fatal("Expected Set to execute a command")
	}
	setCmd := fake.Commands[0]
	expectedSetCmd := []string{"[in:" + repoPath + "]", "config", "--local", "test.behavior", "our-value"}
	if !equalStringSlices(setCmd, expectedSetCmd) {
		t.Errorf("Set command mismatch:\nGot:  %v\nWant: %v", setCmd, expectedSetCmd)
	}

	// Read it back to verify
	gitValue, err := config.Get("test.behavior", ConfigScopeLocal)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if gitValue != "our-value" {
		t.Errorf("Git returned different value: %s", gitValue)
	}
}
