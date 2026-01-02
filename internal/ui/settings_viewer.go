package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingsViewerModel displays all current settings
type SettingsViewerModel struct {
	content  string
	quitting bool
}

// ConfigValue represents a config key-value pair with scope
type ConfigValue struct {
	Key   string
	Value string
	Scope string // "local", "global", or empty if not set
}

// NewSettingsViewer creates a new settings viewer
func NewSettingsViewer(localValues, globalValues map[string]string) *SettingsViewerModel {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Current Configuration") + "\n\n")

	// Group settings by category
	categories := map[string][]string{
		"Issue Provider": {
			"auto-worktree.issue-provider",
		},
		"AI Tool": {
			"auto-worktree.ai-tool",
		},
		"Auto-select": {
			"auto-worktree.issue-autoselect",
			"auto-worktree.pr-autoselect",
		},
		"Hooks": {
			"auto-worktree.run-hooks",
			"auto-worktree.fail-on-hook-error",
			"auto-worktree.custom-hooks",
		},
		"Issue Templates": {
			"auto-worktree.issue-templates-dir",
			"auto-worktree.issue-templates-disabled",
			"auto-worktree.issue-templates-no-prompt",
			"auto-worktree.issue-templates-detected",
		},
		"Provider Configuration": {
			"auto-worktree.jira-server",
			"auto-worktree.jira-project",
			"auto-worktree.gitlab-server",
			"auto-worktree.gitlab-project",
			"auto-worktree.linear-team",
		},
	}

	categoryOrder := []string{
		"Issue Provider",
		"AI Tool",
		"Auto-select",
		"Hooks",
		"Issue Templates",
		"Provider Configuration",
	}

	for _, category := range categoryOrder {
		keys := categories[category]
		hasValues := false

		var categoryContent strings.Builder
		categoryContent.WriteString(HeaderStyle.Render(category) + "\n")

		for _, key := range keys {
			localVal, hasLocal := localValues[key]
			globalVal, hasGlobal := globalValues[key]

			if !hasLocal && !hasGlobal {
				continue
			}

			hasValues = true
			shortKey := strings.TrimPrefix(key, "auto-worktree.")

			if hasLocal {
				scope := SubtleStyle.Render("[local]")
				value := SuccessStyle.Render(localVal)
				if localVal == "" {
					value = SubtleStyle.Render("(not set)")
				}
				categoryContent.WriteString(fmt.Sprintf("  %s %s %s\n", shortKey, scope, value))
			}

			if hasGlobal && (!hasLocal || globalVal != localVal) {
				scope := SubtleStyle.Render("[global]")
				value := InfoStyle.Render(globalVal)
				if globalVal == "" {
					value = SubtleStyle.Render("(not set)")
				}
				categoryContent.WriteString(fmt.Sprintf("  %s %s %s\n", shortKey, scope, value))
			}
		}

		if hasValues {
			b.WriteString(categoryContent.String())
			b.WriteString("\n")
		}
	}

	// Add help text at bottom
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Press q or Esc to return"))

	return &SettingsViewerModel{
		content: b.String(),
	}
}

// Init initializes the viewer
func (m SettingsViewerModel) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m SettingsViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "q", keyEsc, keyCtrlC, keyEnter:
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the viewer
func (m SettingsViewerModel) View() string {
	if m.quitting {
		return ""
	}

	// Wrap in a box
	return "\n" + lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Render(m.content) + "\n"
}
