package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// SettingItem represents a configurable setting
type SettingItem struct {
	Key         string
	title       string
	description string
	ValueType   string // "string", "bool", "select"
	Options     []string
	CurrentVal  string
}

// NewSettingItem creates a new setting item
func NewSettingItem(key, title, description, valueType string, options []string, currentVal string) SettingItem {
	return SettingItem{
		Key:         key,
		title:       title,
		description: description,
		ValueType:   valueType,
		Options:     options,
		CurrentVal:  currentVal,
	}
}

// Title returns the title with current value (implements list.Item)
func (i SettingItem) Title() string {
	if i.CurrentVal == "" {
		return fmt.Sprintf("%s: (not set)", i.title)
	}
	return fmt.Sprintf("%s: %s", i.title, i.CurrentVal)
}

// Description returns the description (implements list.Item)
func (i SettingItem) Description() string {
	return i.description
}

// FilterValue returns the value used for filtering (implements list.Item)
func (i SettingItem) FilterValue() string {
	return i.title
}

// SettingsMenuModel represents the main settings menu
type SettingsMenuModel struct {
	list     list.Model
	choice   string
	quitting bool
}

// NewSettingsMenuModel creates a new settings menu
func NewSettingsMenuModel(settings []SettingItem) *SettingsMenuModel {
	items := make([]list.Item, len(settings)+2) // +2 for View All and Reset
	for i, setting := range settings {
		items[i] = setting
	}

	// Add special menu items
	items[len(settings)] = MenuItem{
		title:       "View All Settings",
		description: "Display all current configuration values",
		action:      "view-all",
	}
	items[len(settings)+1] = MenuItem{
		title:       "Reset to Defaults",
		description: "Clear all auto-worktree configuration",
		action:      "reset",
	}

	const defaultWidth = 80
	const defaultHeight = 20

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = SelectedItemStyle
	delegate.Styles.SelectedDesc = HighlightStyle

	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.Title = "Settings"
	l.Styles.Title = TitleStyle
	l.SetShowStatusBar(false)

	return &SettingsMenuModel{
		list: l,
	}
}

// Init initializes the settings menu
func (m SettingsMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m SettingsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			item := m.list.SelectedItem()
			switch i := item.(type) {
			case SettingItem:
				m.choice = "edit:" + i.Key
			case MenuItem:
				m.choice = i.Action()
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the settings menu
func (m SettingsMenuModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// GetChoice returns the user's choice
func (m SettingsMenuModel) GetChoice() string {
	return m.choice
}
