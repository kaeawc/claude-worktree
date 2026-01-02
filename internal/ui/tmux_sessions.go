package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kaeawc/auto-worktree/internal/session"
)

// SessionListItem represents a session in the sessions list
type SessionListItem struct {
	metadata *session.Metadata
}

// NewSessionListItem creates a new session list item
func NewSessionListItem(metadata *session.Metadata) SessionListItem {
	return SessionListItem{
		metadata: metadata,
	}
}

// Title returns the display title for the session
func (i SessionListItem) Title() string {
	statusIcon := statusIcon(i.metadata.Status)
	return fmt.Sprintf("%s %s", statusIcon, i.metadata.SessionName)
}

// Description returns the description for the session
func (i SessionListItem) Description() string {
	age := time.Since(i.metadata.CreatedAt)
	ageStr := formatDuration(age)

	details := []string{
		fmt.Sprintf("Branch: %s", i.metadata.BranchName),
		fmt.Sprintf("Age: %s", ageStr),
	}

	if i.metadata.WindowCount > 0 {
		details = append(details, fmt.Sprintf("Windows: %d", i.metadata.WindowCount))
	}

	if i.metadata.Dependencies.Installed {
		details = append(details, fmt.Sprintf("Deps: %s", i.metadata.Dependencies.PackageManager))
	}

	return strings.Join(details, " | ")
}

// FilterValue returns the value used for filtering
func (i SessionListItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.metadata.SessionName, i.metadata.BranchName, i.metadata.WorktreePath)
}

// Metadata returns the underlying metadata
func (i SessionListItem) Metadata() *session.Metadata {
	return i.metadata
}

// SessionListModel represents the sessions list UI component
type SessionListModel struct {
	list      list.Model
	items     []SessionListItem
	choice    *SessionListItem
	err       error
	filtering bool
}

// NewSessionList creates a new sessions list
func NewSessionList(title string, items []SessionListItem) SessionListModel {
	// Convert items to list.Item
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list
	l := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = HeaderStyle

	return SessionListModel{
		list:      l,
		items:     items,
		filtering: false,
	}
}

// Init initializes the model
func (m SessionListModel) Init() tea.Cmd {
	return nil
}

// Update handles updates (key presses, etc.)
func (m SessionListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := BoxStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.filtering {
				// Exit filter mode
				m.filtering = false
				m.list.SetShowFilter(false)

				return m, nil
			}

			// Quit
			m.err = fmt.Errorf("canceled")

			return m, tea.Quit

		case "enter":
			// Select current item
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				if item, ok := selectedItem.(SessionListItem); ok {
					m.choice = &item

					return m, tea.Quit
				}
			}

			return m, nil

		case "/":
			// Toggle filter mode
			m.filtering = !m.filtering
			m.list.SetShowFilter(m.filtering)

			if m.filtering {
				m.list.ResetFilter()
			}
		}
	}

	m.list, _ = m.list.Update(msg)

	return m, nil
}

// View renders the sessions list
func (m SessionListModel) View() string {
	var s strings.Builder

	// Show the list
	s.WriteString(m.list.View())
	s.WriteString("\n\n")

	// Show instructions
	s.WriteString(SubtleStyle.Render("Press / to filter, Enter to attach, q/Esc to quit"))

	return BoxStyle.Render(s.String())
}

// Choice returns the selected item
func (m SessionListModel) Choice() *SessionListItem {
	return m.choice
}

// Err returns any error
func (m SessionListModel) Err() error {
	return m.err
}

// statusIcon returns an emoji icon for the session status
func statusIcon(status session.Status) string {
	switch status {
	case session.StatusRunning:
		return "🟢"
	case session.StatusPaused:
		return "⏸️"
	case session.StatusIdle:
		return "💤"
	case session.StatusNeedsAttention:
		return "⚠️"
	case session.StatusFailed:
		return "🔴"
	default:
		return "❓"
	}
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}

	if d < time.Hour {
		minutes := int(d.Minutes())

		return fmt.Sprintf("%dm", minutes)
	}

	if d < 24*time.Hour {
		hours := int(d.Hours())

		return fmt.Sprintf("%dh", hours)
	}

	days := int(d.Hours()) / 24

	return fmt.Sprintf("%dd", days)
}
