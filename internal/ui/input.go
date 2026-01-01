package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel represents a text input model.
type InputModel struct {
	textInput textinput.Model
	err       error
	prompt    string
	value     string
}

// NewInput creates a new input model.
func NewInput(prompt, placeholder string) InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.Width = 50

	return InputModel{
		textInput: ti,
		prompt:    prompt,
	}
}

// Init initializes the input model.
func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles input updates.
func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			m.value = m.textInput.Value()
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

// View renders the input.
func (m InputModel) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		HeaderStyle.Render(m.prompt),
		m.textInput.View(),
		SubtleStyle.Render("(press Enter to confirm, Esc to cancel)"),
	)
}

// Value returns the input value.
func (m InputModel) Value() string {
	return m.value
}

// Err returns any error that occurred.
func (m InputModel) Err() error {
	return m.err
}
