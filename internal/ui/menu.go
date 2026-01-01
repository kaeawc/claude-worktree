// Package ui provides terminal UI components using Bubbletea.
package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// MenuItem represents an item in the menu.
type MenuItem struct {
	title       string
	description string
	action      string
}

// NewMenuItem creates a new menu item.
func NewMenuItem(title, description, action string) MenuItem {
	return MenuItem{
		title:       title,
		description: description,
		action:      action,
	}
}

// Title returns the menu item title.
func (i MenuItem) Title() string { return i.title }

// Description returns the menu item description.
func (i MenuItem) Description() string { return i.description }

// FilterValue returns the value to filter on.
func (i MenuItem) FilterValue() string { return i.title }

// Action returns the action identifier for this menu item.
func (i MenuItem) Action() string { return i.action }

// MenuModel represents the main menu model.
type MenuModel struct {
	list     list.Model
	choice   string
	quitting bool
}

// NewMenu creates a new menu model.
func NewMenu(title string, items []MenuItem) MenuModel {
	const defaultWidth = 80
	const defaultHeight = 14

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	l := list.New(listItems, menuItemDelegate{}, defaultWidth, defaultHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return MenuModel{list: l}
}

// Init initializes the menu model.
func (m MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles menu updates.
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c", "esc":
			m.quitting = true

			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(MenuItem)
			if ok {
				m.choice = i.Action()
			}

			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

// View renders the menu.
func (m MenuModel) View() string {
	if m.quitting && m.choice == "" {
		return quitTextStyle.Render("Canceled")
	}

	return "\n" + m.list.View()
}

// Choice returns the selected menu item's action.
func (m MenuModel) Choice() string {
	return m.choice
}

// menuItemDelegate is a custom delegate for menu items.
type menuItemDelegate struct{}

func (d menuItemDelegate) Height() int                             { return 1 }
func (d menuItemDelegate) Spacing() int                            { return 0 }
func (d menuItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d menuItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(MenuItem)
	if !ok {
		return
	}

	str := i.Title()

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("▸ " + s[0])
		}
	}

	//nolint:errcheck // Error writing to writer is not actionable in render function
	fmt.Fprint(w, fn(str))
}
