package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Provider represents an issue tracking provider
type Provider string

const (
	ProviderNone   Provider = ""
	ProviderGitHub Provider = "github"
	ProviderGitLab Provider = "gitlab"
	ProviderJira   Provider = "jira"
	ProviderLinear Provider = "linear"
)

// ProviderItem represents a provider choice in the menu
type ProviderItem struct {
	title       string
	description string
	provider    Provider
}

func (i ProviderItem) Title() string       { return i.title }
func (i ProviderItem) Description() string { return i.description }
func (i ProviderItem) FilterValue() string { return i.title }

// ProviderMenuModel represents the provider selection menu
type ProviderMenuModel struct {
	list     list.Model
	choice   Provider
	quitting bool
}

// NewProviderMenuModel creates a new provider selection menu
func NewProviderMenuModel() *ProviderMenuModel {
	items := []list.Item{
		ProviderItem{
			title:       "GitHub",
			description: "Use GitHub Issues and Pull Requests",
			provider:    ProviderGitHub,
		},
		ProviderItem{
			title:       "GitLab",
			description: "Use GitLab Issues and Merge Requests",
			provider:    ProviderGitLab,
		},
		ProviderItem{
			title:       "JIRA",
			description: "Use Atlassian JIRA for issue tracking",
			provider:    ProviderJira,
		},
		ProviderItem{
			title:       "Linear",
			description: "Use Linear for issue tracking",
			provider:    ProviderLinear,
		},
	}

	const defaultWidth = 80
	const defaultHeight = 20

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = SelectedItemStyle
	delegate.Styles.SelectedDesc = HighlightStyle

	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.Title = "Select Issue Provider"
	l.Styles.Title = TitleStyle
	l.SetShowStatusBar(false)

	return &ProviderMenuModel{
		list:   l,
		choice: ProviderNone,
	}
}

func (m ProviderMenuModel) Init() tea.Cmd {
	return nil
}

func (m ProviderMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if selectedItem, ok := m.list.SelectedItem().(ProviderItem); ok {
				m.choice = selectedItem.provider
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ProviderMenuModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// GetChoice returns the selected provider
func (m ProviderMenuModel) GetChoice() Provider {
	return m.choice
}

// AITool represents an AI coding assistant tool
type AITool string

const (
	AIToolNone   AITool = ""
	AIToolClaude AITool = "claude"
	AIToolCodex  AITool = "codex"
	AIToolGemini AITool = "gemini"
	AIToolSkip   AITool = "skip"
)

// AIToolItem represents an AI tool choice in the menu
type AIToolItem struct {
	title       string
	description string
	tool        AITool
}

func (i AIToolItem) Title() string       { return i.title }
func (i AIToolItem) Description() string { return i.description }
func (i AIToolItem) FilterValue() string { return i.title }

// AIToolMenuModel represents the AI tool selection menu
type AIToolMenuModel struct {
	list     list.Model
	choice   AITool
	quitting bool
}

// NewAIToolMenuModel creates a new AI tool selection menu
func NewAIToolMenuModel() *AIToolMenuModel {
	items := []list.Item{
		AIToolItem{
			title:       "Claude Code (Anthropic)",
			description: "Install and configure Claude Code CLI",
			tool:        AIToolClaude,
		},
		AIToolItem{
			title:       "Codex CLI (OpenAI)",
			description: "Install and configure OpenAI Codex",
			tool:        AIToolCodex,
		},
		AIToolItem{
			title:       "Gemini",
			description: "Install and configure Google Gemini",
			tool:        AIToolGemini,
		},
		AIToolItem{
			title:       "Skip",
			description: "Don't use an AI tool",
			tool:        AIToolSkip,
		},
	}

	const defaultWidth = 80
	const defaultHeight = 20

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = SelectedItemStyle
	delegate.Styles.SelectedDesc = HighlightStyle

	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.Title = "Select AI Tool"
	l.Styles.Title = TitleStyle
	l.SetShowStatusBar(false)

	return &AIToolMenuModel{
		list:   l,
		choice: AIToolNone,
	}
}

func (m AIToolMenuModel) Init() tea.Cmd {
	return nil
}

func (m AIToolMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if selectedItem, ok := m.list.SelectedItem().(AIToolItem); ok {
				m.choice = selectedItem.tool
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m AIToolMenuModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// GetChoice returns the selected AI tool
func (m AIToolMenuModel) GetChoice() AITool {
	return m.choice
}
