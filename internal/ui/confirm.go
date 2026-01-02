package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	keyEnter = "enter"
)

// ConfirmModel represents a yes/no confirmation dialog
type ConfirmModel struct {
	prompt   string
	choice   bool
	selected int // 0 = Yes, 1 = No
	quitting bool
}

// NewConfirmModel creates a new confirmation dialog
func NewConfirmModel(prompt string) ConfirmModel {
	return ConfirmModel{
		prompt:   prompt,
		selected: 1, // Default to "No" for safety
	}
}

// Init initializes the confirmation dialog.
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles user input for the confirmation dialog.
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyCtrlC, "q", keyEsc:
			m.quitting = true
			m.choice = false

			return m, tea.Quit

		case "left", "h":
			m.selected = 0

		case "right", "l":
			m.selected = 1

		case "y":
			m.selected = 0
			m.choice = true
			m.quitting = true

			return m, tea.Quit

		case "n":
			m.selected = 1
			m.choice = false
			m.quitting = true

			return m, tea.Quit

		case keyEnter:
			m.choice = m.selected == 0
			m.quitting = true

			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the confirmation dialog.
func (m ConfirmModel) View() string {
	if m.quitting {
		return ""
	}

	yesStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen)

	noStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed)

	unselectedStyle := lipgloss.NewStyle().
		Padding(0, 2)

	var yesButton, noButton string
	if m.selected == 0 {
		yesButton = yesStyle.Render("Yes")
		noButton = unselectedStyle.Render("No")
	} else {
		yesButton = unselectedStyle.Render("Yes")
		noButton = noStyle.Render("No")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesButton, "  ", noButton)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		WarningStyle.Render(m.prompt),
		"",
		buttons,
		"",
		HelpStyle.Render("Use arrow keys or y/n to select, enter to confirm"),
	)

	return BoxStyle.Render(content)
}

// GetChoice returns true if the user confirmed, false otherwise
func (m ConfirmModel) GetChoice() bool {
	return m.choice
}
