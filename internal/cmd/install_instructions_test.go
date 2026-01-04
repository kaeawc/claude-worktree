package cmd

import (
	"strings"
	"testing"
)

func TestGitHubInstallInfo(t *testing.T) {
	info := GitHubInstallInfo()

	if info.CLIName != "gh" {
		t.Errorf("expected CLIName 'gh', got '%s'", info.CLIName)
	}

	if info.ProviderName != "GitHub" {
		t.Errorf("expected ProviderName 'GitHub', got '%s'", info.ProviderName)
	}

	if len(info.InstallOptions) != 3 {
		t.Errorf("expected 3 install options, got %d", len(info.InstallOptions))
	}

	if len(info.AuthSteps) != 1 || info.AuthSteps[0] != "gh auth login" {
		t.Errorf("expected auth step 'gh auth login', got %v", info.AuthSteps)
	}
}

func TestGitLabInstallInfo(t *testing.T) {
	info := GitLabInstallInfo()

	if info.CLIName != "glab" {
		t.Errorf("expected CLIName 'glab', got '%s'", info.CLIName)
	}

	if info.ProviderName != "GitLab" {
		t.Errorf("expected ProviderName 'GitLab', got '%s'", info.ProviderName)
	}

	if len(info.InstallOptions) != 3 {
		t.Errorf("expected 3 install options, got %d", len(info.InstallOptions))
	}

	if len(info.AuthSteps) != 1 || info.AuthSteps[0] != "glab auth login" {
		t.Errorf("expected auth step 'glab auth login', got %v", info.AuthSteps)
	}
}

func TestJIRAInstallInfo(t *testing.T) {
	info := JIRAInstallInfo()

	if info.CLIName != "jira" {
		t.Errorf("expected CLIName 'jira', got '%s'", info.CLIName)
	}

	if info.ProviderName != "JIRA" {
		t.Errorf("expected ProviderName 'JIRA', got '%s'", info.ProviderName)
	}

	if len(info.InstallOptions) != 3 {
		t.Errorf("expected 3 install options, got %d", len(info.InstallOptions))
	}

	if len(info.AuthSteps) != 1 || info.AuthSteps[0] != "jira init" {
		t.Errorf("expected auth step 'jira init', got %v", info.AuthSteps)
	}
}

func TestLinearInstallInfo(t *testing.T) {
	info := LinearInstallInfo()

	if info.CLIName != "linear" {
		t.Errorf("expected CLIName 'linear', got '%s'", info.CLIName)
	}

	if info.ProviderName != "Linear" {
		t.Errorf("expected ProviderName 'Linear', got '%s'", info.ProviderName)
	}

	if len(info.InstallOptions) != 3 {
		t.Errorf("expected 3 install options, got %d", len(info.InstallOptions))
	}

	// Check that correct tap is used
	found := false
	for _, opt := range info.InstallOptions {
		if strings.Contains(opt.Command, "schpet/tap/linear") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected install option with 'schpet/tap/linear'")
	}

	if len(info.AuthSteps) != 2 {
		t.Errorf("expected 2 auth steps for Linear, got %d", len(info.AuthSteps))
	}

	// Check that auth steps mention API key
	hasAPIKeyStep := false
	for _, step := range info.AuthSteps {
		if strings.Contains(step, "LINEAR_API_KEY") {
			hasAPIKeyStep = true
			break
		}
	}
	if !hasAPIKeyStep {
		t.Error("expected auth step mentioning LINEAR_API_KEY")
	}
}

func TestFormatNotInstalledError(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install test"},
			{Platform: PlatformLinux, Label: "Linux", Command: "apt install test"},
		},
		AuthSteps: []string{"test auth"},
	}

	msg := info.FormatNotInstalledError()

	if !strings.Contains(msg, "test CLI is not installed") {
		t.Error("expected message to contain 'test CLI is not installed'")
	}

	if !strings.Contains(msg, "Install with:") {
		t.Error("expected message to contain 'Install with:'")
	}

	if !strings.Contains(msg, "After installation:") {
		t.Error("expected message to contain 'After installation:'")
	}

	if !strings.Contains(msg, "test auth") {
		t.Error("expected message to contain auth step")
	}
}

func TestFormatNotInstalledError_URLOption(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		InstallOptions: []InstallOption{
			{Platform: "", Label: "Other", Command: "https://example.com/install", IsURL: true},
		},
		AuthSteps: []string{},
	}

	msg := info.FormatNotInstalledError()

	if !strings.Contains(msg, "See https://example.com/install") {
		t.Errorf("expected URL to be prefixed with 'See', got: %s", msg)
	}
}

func TestFormatNotAuthenticatedError(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		AuthSteps:    []string{"test auth login"},
	}

	msg := info.FormatNotAuthenticatedError()

	if !strings.Contains(msg, "test CLI is not authenticated") {
		t.Error("expected message to contain 'test CLI is not authenticated'")
	}

	if !strings.Contains(msg, "Run: test auth login") {
		t.Errorf("expected message to contain 'Run: test auth login', got: %s", msg)
	}
}

func TestFormatNotAuthenticatedError_MultipleSteps(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		AuthSteps:    []string{"Step 1", "Step 2"},
	}

	msg := info.FormatNotAuthenticatedError()

	if !strings.Contains(msg, "Configure with:") {
		t.Error("expected message to contain 'Configure with:' for multiple steps")
	}

	if !strings.Contains(msg, "Step 1") || !strings.Contains(msg, "Step 2") {
		t.Error("expected message to contain all auth steps")
	}
}

func TestFormatNotAuthenticatedError_NoSteps(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		AuthSteps:    []string{},
	}

	msg := info.FormatNotAuthenticatedError()

	if msg != "test CLI is not authenticated" {
		t.Errorf("expected simple message, got: %s", msg)
	}
}

func TestGetRelevantOptions_PlatformSpecific(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install test"},
			{Platform: PlatformLinux, Label: "Linux", Command: "apt install test"},
			{Platform: PlatformWindows, Label: "Windows", Command: "winget install test"},
			{Platform: "", Label: "Other", Command: "https://example.com", IsURL: true},
		},
	}

	// Test macOS
	options := info.getRelevantOptions(PlatformMacOS)
	if len(options) != 2 { // macOS + Other
		t.Errorf("expected 2 options for macOS, got %d", len(options))
	}
	if options[0].Label != "macOS" {
		t.Errorf("expected first option to be macOS, got %s", options[0].Label)
	}

	// Test Linux
	options = info.getRelevantOptions(PlatformLinux)
	if len(options) != 2 { // Linux + Other
		t.Errorf("expected 2 options for Linux, got %d", len(options))
	}
	if options[0].Label != "Linux" {
		t.Errorf("expected first option to be Linux, got %s", options[0].Label)
	}
}

func TestGetRelevantOptions_NoPlatformMatch(t *testing.T) {
	info := &ProviderInstallInfo{
		CLIName:      "test",
		ProviderName: "Test",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install test"},
		},
	}

	// Test with a platform that has no specific option
	options := info.getRelevantOptions(PlatformLinux)
	if len(options) != 1 {
		t.Errorf("expected 1 option when no platform match, got %d", len(options))
	}
}

func TestGetCurrentPlatform(t *testing.T) {
	platform := GetCurrentPlatform()

	// Just verify it returns a valid platform (darwin, linux, or windows)
	if platform != PlatformMacOS && platform != PlatformLinux && platform != PlatformWindows {
		// This is fine - could be running on FreeBSD or other OS
		t.Logf("Current platform: %s", platform)
	}
}
