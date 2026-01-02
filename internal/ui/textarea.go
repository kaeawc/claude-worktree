package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// TextAreaModel represents a multi-line text input model.
type TextAreaModel struct {
	textarea textarea.Model
	err      error
	prompt   string
	value    string
}

// NewTextArea creates a new textarea model.
func NewTextArea(prompt, placeholder string) TextAreaModel {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.CharLimit = 0 // No limit

	return TextAreaModel{
		textarea: ta,
		prompt:   prompt,
	}
}

// Init initializes the textarea model.
func (m TextAreaModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles textarea updates.
func (m TextAreaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyCtrlD:
			// Ctrl+D to submit
			m.value = m.textarea.Value()
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			return m, tea.Quit
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)

	return m, cmd
}

// View renders the textarea.
func (m TextAreaModel) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		HeaderStyle.Render(m.prompt),
		m.textarea.View(),
		SubtleStyle.Render("(press Ctrl+D to confirm, Esc to cancel)"),
	)
}

// Value returns the textarea value.
func (m TextAreaModel) Value() string {
	return m.value
}

// Err returns any error that occurred.
func (m TextAreaModel) Err() error {
	return m.err
}
