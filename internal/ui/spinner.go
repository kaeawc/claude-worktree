package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel represents a spinner with a message
type SpinnerModel struct {
	spinner spinner.Model
	message string
	done    bool
	err     error
}

// NewSpinnerModel creates a new spinner model
func NewSpinnerModel(message string) *SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &SpinnerModel{
		spinner: s,
		message: message,
		done:    false,
	}
}

// Init initializes the spinner model
func (m *SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages for the spinner
func (m *SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Allow Ctrl+C to quit
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd

	case SpinnerDoneMsg:
		m.done = true
		m.err = msg.Err

		return m, tea.Quit

	case SpinnerUpdateMsg:
		m.message = msg.Message

		return m, nil

	default:
		return m, nil
	}
}

// View renders the spinner
func (m *SpinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("✗ %s\n", m.err.Error())
		}

		return fmt.Sprintf("✓ %s\n", m.message)
	}

	return fmt.Sprintf("%s %s\n", m.spinner.View(), m.message)
}

// SpinnerDoneMsg signals that the spinner should stop
type SpinnerDoneMsg struct {
	Err error
}

// SpinnerUpdateMsg updates the spinner message
type SpinnerUpdateMsg struct {
	Message string
}
