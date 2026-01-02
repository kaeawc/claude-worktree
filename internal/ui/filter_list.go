package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	filterKeyCtrlC = "ctrl+c"
	filterKeyEsc   = "esc"
	filterKeyEnter = "enter"
)

// FilterableListItem represents an item in the filterable list
type FilterableListItem struct {
	number      int
	title       string
	labels      []string
	hasWorktree bool // Mark if worktree exists
}

// NewFilterableListItem creates a new filterable list item
func NewFilterableListItem(number int, title string, labels []string, hasWorktree bool) FilterableListItem {
	return FilterableListItem{
		number:      number,
		title:       title,
		labels:      labels,
		hasWorktree: hasWorktree,
	}
}

// Number returns the issue number
func (i FilterableListItem) Number() int {
	return i.number
}

// Title returns the title for the list item display.
func (i FilterableListItem) Title() string {
	prefix := ""
	if i.hasWorktree {
		prefix = "● "
	}
	return fmt.Sprintf("%s#%d | %s", prefix, i.number, i.title)
}

// Description returns the description for the list item display.
func (i FilterableListItem) Description() string {
	if len(i.labels) == 0 {
		return ""
	}
	labelStrs := make([]string, len(i.labels))
	for idx, label := range i.labels {
		labelStrs[idx] = fmt.Sprintf("[%s]", label)
	}
	return strings.Join(labelStrs, " ")
}

// FilterValue returns the value used for filtering the list item.
func (i FilterableListItem) FilterValue() string {
	// Allow filtering by number or title
	return fmt.Sprintf("%d %s", i.number, i.title)
}

// FilterListModel represents a filterable list UI component
type FilterListModel struct {
	list        list.Model
	filterInput textinput.Model
	title       string
	items       []FilterableListItem
	choice      *FilterableListItem
	err         error
	filtering   bool
}

// NewFilterList creates a new filterable list
func NewFilterList(title string, items []FilterableListItem) FilterListModel {
	// Convert items to list.Item
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list
	l := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = HeaderStyle

	// Create filter input
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 50
	ti.Width = 50

	return FilterListModel{
		list:        l,
		filterInput: ti,
		title:       title,
		items:       items,
		filtering:   false,
	}
}

// Init initializes the model
func (m FilterListModel) Init() tea.Cmd {
	return nil
}

// Update handles updates (key presses, filter changes)
func (m FilterListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := BoxStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case filterKeyCtrlC, "q":
			if m.filtering {
				// Exit filter mode
				m.filtering = false
				m.filterInput.SetValue("")
				m.filterInput.Blur()
				return m, nil
			}
			// Quit
			m.err = fmt.Errorf("canceled")
			return m, tea.Quit

		case filterKeyEsc:
			if m.filtering {
				// Exit filter mode
				m.filtering = false
				m.filterInput.SetValue("")
				m.filterInput.Blur()
				return m, nil
			}
			// Quit
			m.err = fmt.Errorf("canceled")
			return m, tea.Quit

		case filterKeyEnter:
			if m.filtering {
				// Exit filter mode and keep the filter
				m.filtering = false
				m.filterInput.Blur()
				return m, nil
			}

			// Select current item
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				if item, ok := selectedItem.(FilterableListItem); ok {
					m.choice = &item
					return m, tea.Quit
				}
			}
			return m, nil

		case "/":
			if !m.filtering {
				// Enter filter mode
				m.filtering = true
				m.filterInput.Focus()
				return m, textinput.Blink
			}
		}
	}

	// Update appropriate component based on mode
	var cmd tea.Cmd
	if m.filtering {
		m.filterInput, cmd = m.filterInput.Update(msg)

		// Apply filter
		query := strings.ToLower(m.filterInput.Value())
		if query != "" {
			filteredItems := make([]list.Item, 0)
			for _, item := range m.items {
				filterValue := strings.ToLower(item.FilterValue())
				if strings.Contains(filterValue, query) {
					filteredItems = append(filteredItems, item)
				}
			}
			m.list.SetItems(filteredItems)
		} else {
			// Reset to all items
			allItems := make([]list.Item, len(m.items))
			for i, item := range m.items {
				allItems[i] = item
			}
			m.list.SetItems(allItems)
		}

		return m, cmd
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the filterable list
func (m FilterListModel) View() string {
	var s strings.Builder

	// Show the list
	s.WriteString(m.list.View())
	s.WriteString("\n\n")

	// Show filter input if in filtering mode
	if m.filtering {
		s.WriteString(SubtleStyle.Render("Filter: "))
		s.WriteString(m.filterInput.View())
		s.WriteString("\n")
		s.WriteString(SubtleStyle.Render("(press Enter to apply, Esc to cancel)"))
	} else {
		s.WriteString(SubtleStyle.Render("Press / to filter, Enter to select, q/Esc to quit"))
	}

	return BoxStyle.Render(s.String())
}

// Choice returns the selected item
func (m FilterListModel) Choice() *FilterableListItem {
	return m.choice
}

// Err returns any error
func (m FilterListModel) Err() error {
	return m.err
}
