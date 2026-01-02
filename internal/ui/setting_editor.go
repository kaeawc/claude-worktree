package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SettingEditorModel represents an editor for a single setting
type SettingEditorModel struct {
	setting    SettingItem
	valueType  string
	textInput  textinput.Model
	list       list.Model
	newValue   string
	err        error
	quitting   bool
	editingStr bool // true when editing string value
}

// NewSettingEditor creates a new setting editor based on the value type
func NewSettingEditor(setting SettingItem) *SettingEditorModel {
	model := &SettingEditorModel{
		setting:   setting,
		valueType: setting.ValueType,
	}

	switch setting.ValueType {
	case "string":
		// Create text input
		ti := textinput.New()
		ti.Placeholder = "Enter value..."
		ti.Focus()
		ti.Width = 50
		if setting.CurrentVal != "" {
			ti.SetValue(setting.CurrentVal)
		}
		model.textInput = ti
		model.editingStr = true

	case "bool":
		// Create list with true/false options
		items := []list.Item{
			MenuItem{title: "true", description: "Enable this setting", action: "true"},
			MenuItem{title: "false", description: "Disable this setting", action: "false"},
		}
		const defaultWidth = 60
		const defaultHeight = 10
		l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
		l.Title = fmt.Sprintf("Set %s", setting.Title())
		l.Styles.Title = TitleStyle
		l.SetShowStatusBar(false)
		model.list = l

	case "select":
		// Create list with available options
		items := make([]list.Item, len(setting.Options))
		for i, opt := range setting.Options {
			items[i] = MenuItem{
				title:       opt,
				description: "",
				action:      opt,
			}
		}
		const defaultWidth = 60
		const defaultHeight = 15
		l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
		l.Title = fmt.Sprintf("Set %s", setting.Title())
		l.Styles.Title = TitleStyle
		l.SetShowStatusBar(false)
		model.list = l
	}

	return model
}

// Init initializes the editor
func (m SettingEditorModel) Init() tea.Cmd {
	if m.editingStr {
		return textinput.Blink
	}
	return nil
}

// Update handles user input
func (m SettingEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editingStr {
		return m.updateStringEditor(msg)
	}
	return m.updateListEditor(msg)
}

func (m SettingEditorModel) updateStringEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			m.newValue = m.textInput.Value()
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m SettingEditorModel) updateListEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case keyCtrlC, "q", keyEsc:
			m.err = fmt.Errorf("canceled")
			m.quitting = true
			return m, tea.Quit

		case keyEnter:
			if item, ok := m.list.SelectedItem().(MenuItem); ok {
				m.newValue = item.Action()
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the editor
func (m SettingEditorModel) View() string {
	if m.quitting {
		return ""
	}

	if m.editingStr {
		return fmt.Sprintf(
			"\n%s\n\n%s\n\n%s\n\n%s\n",
			TitleStyle.Render(fmt.Sprintf("Edit: %s", m.setting.Title())),
			SubtleStyle.Render(m.setting.Description()),
			m.textInput.View(),
			HelpStyle.Render("Enter to confirm • Esc to cancel"),
		)
	}

	return "\n" + m.list.View()
}

// GetValue returns the new value
func (m SettingEditorModel) GetValue() string {
	return m.newValue
}

// Err returns any error
func (m SettingEditorModel) Err() error {
	return m.err
}

// ScopeSelectorModel allows choosing local or global scope
type ScopeSelectorModel struct {
	list     list.Model
	scope    string
	quitting bool
}

// NewScopeSelector creates a new scope selector
func NewScopeSelector() *ScopeSelectorModel {
	items := []list.Item{
		MenuItem{
			title:       "Local (this repository only)",
			description: "Save to .git/config",
			action:      "local",
		},
		MenuItem{
			title:       "Global (all repositories)",
			description: "Save to ~/.gitconfig",
			action:      "global",
		},
	}

	const defaultWidth = 60
	const defaultHeight = 10

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = SelectedItemStyle
	delegate.Styles.SelectedDesc = HighlightStyle

	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.Title = "Choose Scope"
	l.Styles.Title = TitleStyle
	l.SetShowStatusBar(false)

	return &ScopeSelectorModel{
		list: l,
	}
}

// Init initializes the scope selector
func (m ScopeSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m ScopeSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case keyCtrlC, "q", keyEsc:
			m.quitting = true
			return m, tea.Quit

		case keyEnter:
			if item, ok := m.list.SelectedItem().(MenuItem); ok {
				m.scope = item.Action()
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the scope selector
func (m ScopeSelectorModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// GetScope returns the selected scope
func (m ScopeSelectorModel) GetScope() string {
	return m.scope
}
