// Package cmd provides command-line interface handlers for auto-worktree operations.
package cmd

import (
	"fmt"
	"runtime"
	"strings"
)

// Platform represents the operating system type
type Platform string

// Platform constants for supported operating systems
const (
	PlatformMacOS   Platform = "darwin"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
)

// InstallOption represents a single installation method
type InstallOption struct {
	Platform Platform // Empty means "Other" or cross-platform
	Label    string   // Display label (e.g., "macOS", "Linux", "Other")
	Command  string   // Install command or instruction
	IsURL    bool     // If true, Command is a URL to documentation
}

// ProviderInstallInfo contains installation and configuration instructions for a provider
type ProviderInstallInfo struct {
	CLIName        string          // Name of the CLI tool (e.g., "gh", "glab")
	ProviderName   string          // Human-readable name (e.g., "GitHub", "GitLab")
	InstallOptions []InstallOption // Platform-specific install options
	AuthSteps      []string        // Post-install authentication/configuration steps
}

// GetCurrentPlatform returns the current operating system platform
func GetCurrentPlatform() Platform {
	return Platform(runtime.GOOS)
}

// FormatNotInstalledError formats an error message for when a CLI is not installed
func (p *ProviderInstallInfo) FormatNotInstalledError() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s CLI is not installed.\n\n", p.CLIName))
	sb.WriteString("Install with:\n")

	currentPlatform := GetCurrentPlatform()
	options := p.getRelevantOptions(currentPlatform)

	for _, opt := range options {
		if opt.IsURL {
			sb.WriteString(fmt.Sprintf("  %s: See %s\n", opt.Label, opt.Command))
		} else {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", opt.Label, opt.Command))
		}
	}

	if len(p.AuthSteps) > 0 {
		sb.WriteString("\nAfter installation:\n")

		for _, step := range p.AuthSteps {
			sb.WriteString(fmt.Sprintf("  %s\n", step))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// FormatNotAuthenticatedError formats an error message for when a CLI is not authenticated
func (p *ProviderInstallInfo) FormatNotAuthenticatedError() string {
	if len(p.AuthSteps) == 0 {
		return fmt.Sprintf("%s CLI is not authenticated", p.CLIName)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s CLI is not authenticated.\n\n", p.CLIName))

	if len(p.AuthSteps) == 1 {
		sb.WriteString(fmt.Sprintf("Run: %s", p.AuthSteps[0]))
	} else {
		sb.WriteString("Configure with:\n")

		for _, step := range p.AuthSteps {
			sb.WriteString(fmt.Sprintf("  %s\n", step))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// getRelevantOptions returns install options relevant to the current platform
func (p *ProviderInstallInfo) getRelevantOptions(current Platform) []InstallOption {
	relevant := make([]InstallOption, 0, len(p.InstallOptions))
	otherOptions := make([]InstallOption, 0, len(p.InstallOptions))

	for _, opt := range p.InstallOptions {
		switch opt.Platform {
		case current:
			relevant = append(relevant, opt)
		case "":
			otherOptions = append(otherOptions, opt)
		}
	}

	// If we have platform-specific options, show those first, then "Other" options
	// If no platform-specific options, show all "Other" options
	if len(relevant) > 0 {
		return append(relevant, otherOptions...)
	}

	// No platform-specific match, show all options
	return append(relevant, p.InstallOptions...)
}

// Provider-specific install information

// GitHubInstallInfo returns installation info for GitHub CLI
func GitHubInstallInfo() *ProviderInstallInfo {
	return &ProviderInstallInfo{
		CLIName:      "gh",
		ProviderName: "GitHub",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install gh"},
			{Platform: PlatformLinux, Label: "Linux", Command: "See https://github.com/cli/cli/blob/trunk/docs/install_linux.md", IsURL: true},
			{Platform: PlatformWindows, Label: "Windows", Command: "winget install GitHub.cli"},
		},
		AuthSteps: []string{"gh auth login"},
	}
}

// GitLabInstallInfo returns installation info for GitLab CLI
func GitLabInstallInfo() *ProviderInstallInfo {
	return &ProviderInstallInfo{
		CLIName:      "glab",
		ProviderName: "GitLab",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install glab"},
			{Platform: PlatformLinux, Label: "Linux", Command: "See https://gitlab.com/gitlab-org/cli#installation", IsURL: true},
			{Platform: PlatformWindows, Label: "Windows", Command: "scoop install glab"},
		},
		AuthSteps: []string{"glab auth login"},
	}
}

// JIRAInstallInfo returns installation info for JIRA CLI
func JIRAInstallInfo() *ProviderInstallInfo {
	return &ProviderInstallInfo{
		CLIName:      "jira",
		ProviderName: "JIRA",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install ankitpokhrel/jira-cli/jira-cli"},
			{Platform: PlatformLinux, Label: "Linux", Command: "See https://github.com/ankitpokhrel/jira-cli#installation", IsURL: true},
			{Platform: "", Label: "Docker", Command: "docker pull ghcr.io/ankitpokhrel/jira-cli:latest"},
		},
		AuthSteps: []string{"jira init"},
	}
}

// LinearInstallInfo returns installation info for Linear CLI
func LinearInstallInfo() *ProviderInstallInfo {
	return &ProviderInstallInfo{
		CLIName:      "linear",
		ProviderName: "Linear",
		InstallOptions: []InstallOption{
			{Platform: PlatformMacOS, Label: "macOS", Command: "brew install schpet/tap/linear"},
			{Platform: "", Label: "Deno", Command: "deno install -A --reload -f -g -n linear jsr:@schpet/linear-cli"},
			{Platform: "", Label: "Other", Command: "See https://github.com/schpet/linear-cli#installation", IsURL: true},
		},
		AuthSteps: []string{
			"Create API key at https://linear.app/settings/account/security",
			"Set environment variable: export LINEAR_API_KEY=lin_api_...",
		},
	}
}
