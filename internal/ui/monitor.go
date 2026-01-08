package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kaeawc/auto-worktree/internal/git"
)

const (
	iconCheckmark = "✅"
	iconWarning   = "⚠️"
	iconError     = "❌"
	iconCritical  = "🔴"
	iconInfo      = "ℹ️"
)

// MonitorModel represents the health monitoring UI
type MonitorModel struct {
	repo     *git.Repository
	interval time.Duration
	results  []*git.HealthCheckResult
	lastRun  time.Time
	running  bool
	err      error
	width    int
	height   int
}

// NewMonitor creates a new monitor model
func NewMonitor(repo *git.Repository, interval time.Duration) *MonitorModel {
	return &MonitorModel{
		repo:     repo,
		interval: interval,
		results:  nil,
		running:  false,
	}
}

// HealthCheckCompleteMsg signals that a health check has completed
type HealthCheckCompleteMsg struct {
	Results []*git.HealthCheckResult
	Error   error
}

// TickMsg signals that it's time to run another health check
type TickMsg time.Time

// Init initializes the monitor
func (m *MonitorModel) Init() tea.Cmd {
	return tea.Batch(
		m.checkHealth(),
		m.tick(),
	)
}

// Update handles messages
func (m *MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			// Manual refresh
			return m, m.checkHealth()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case HealthCheckCompleteMsg:
		m.results = msg.Results
		m.err = msg.Error
		m.lastRun = time.Now()
		m.running = false

		return m, nil

	case TickMsg:
		return m, tea.Batch(
			m.checkHealth(),
			m.tick(),
		)
	}

	return m, nil
}

// View renders the monitor UI
func (m *MonitorModel) View() string {
	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("🔍 Worktree Health Monitor"))
	b.WriteString("\n\n")

	// Status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	switch {
	case m.running:
		b.WriteString(statusStyle.Render("🔄 Checking..."))
	case m.lastRun.IsZero():
		b.WriteString(statusStyle.Render("⏸️  Initializing..."))
	default:
		nextRun := m.lastRun.Add(m.interval)
		timeUntil := time.Until(nextRun)
		b.WriteString(statusStyle.Render(fmt.Sprintf("⏰ Last check: %s | Next check in: %s",
			m.lastRun.Format("15:04:05"),
			formatMonitorDuration(timeUntil),
		)))
	}

	b.WriteString("\n")
	b.WriteString(statusStyle.Render(fmt.Sprintf("📊 Check interval: %s", formatMonitorDuration(m.interval))))
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		b.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Results
	if m.results != nil {
		b.WriteString(m.renderResults())
	} else {
		b.WriteString("No results yet...\n")
	}

	// Help
	b.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	b.WriteString(helpStyle.Render("Press 'r' to refresh now • 'q' or ESC to quit"))

	return b.String()
}

// renderResults formats the health check results for display
func (m *MonitorModel) renderResults() string {
	var b strings.Builder

	healthyCount := 0
	unhealthyCount := 0
	totalIssues := 0

	for _, result := range m.results {
		if result.Healthy {
			healthyCount++
		} else {
			unhealthyCount++
		}

		totalIssues += len(result.Issues)
	}

	// Summary
	summaryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	if unhealthyCount == 0 {
		b.WriteString(summaryStyle.Render("✅ All worktrees healthy!"))
	} else {
		warningStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))
		b.WriteString(warningStyle.Render(fmt.Sprintf("⚠️  %d unhealthy worktree(s) found", unhealthyCount)))
	}

	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Total worktrees: %d | Healthy: %d | Unhealthy: %d | Issues: %d\n",
		len(m.results), healthyCount, unhealthyCount, totalIssues))

	b.WriteString("\n")

	// Detailed results
	for _, result := range m.results {
		b.WriteString(m.renderWorktreeResult(result))
		b.WriteString("\n")
	}

	return b.String()
}

// renderWorktreeResult formats a single worktree result
func (m *MonitorModel) renderWorktreeResult(result *git.HealthCheckResult) string {
	var b strings.Builder

	// Worktree header
	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))
	b.WriteString(pathStyle.Render(fmt.Sprintf("📁 %s", result.WorktreePath)))
	b.WriteString(" ")

	// Status
	severity := result.GetMaxSeverity()
	var statusIcon string
	var statusColor string

	switch severity {
	case git.SeverityOK:
		statusIcon = iconCheckmark
		statusColor = "86"
	case git.SeverityWarning:
		statusIcon = iconWarning
		statusColor = "214"
	case git.SeverityError:
		statusIcon = iconError
		statusColor = "196"
	case git.SeverityCritical:
		statusIcon = iconCritical
		statusColor = "196"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor))

	if result.Healthy {
		b.WriteString(statusStyle.Render(fmt.Sprintf("%s Healthy", statusIcon)))
	} else {
		b.WriteString(statusStyle.Render(fmt.Sprintf("%s Unhealthy (%d issues)", statusIcon, len(result.Issues))))
	}

	b.WriteString("\n")

	// Issues (only show if unhealthy and not too many)
	if !result.Healthy && len(result.Issues) > 0 {
		// Limit to first 3 issues to avoid clutter
		maxDisplay := 3
		for i, issue := range result.Issues {
			if i >= maxDisplay {
				remaining := len(result.Issues) - maxDisplay
				b.WriteString(fmt.Sprintf("   ... and %d more issue(s)\n", remaining))

				break
			}

			var icon string

			switch issue.Severity {
			case git.SeverityOK:
				icon = iconInfo
			case git.SeverityWarning:
				icon = iconWarning
			case git.SeverityError:
				icon = iconError
			case git.SeverityCritical:
				icon = iconCritical
			}

			issueStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
			b.WriteString(issueStyle.Render(fmt.Sprintf("   %s %s: %s\n", icon, issue.Category, issue.Description)))
		}
	}

	return b.String()
}

// checkHealth runs a health check in the background
func (m *MonitorModel) checkHealth() tea.Cmd {
	return func() tea.Msg {
		m.running = true
		results, err := m.repo.PerformHealthCheckAll()

		return HealthCheckCompleteMsg{
			Results: results,
			Error:   err,
		}
	}
}

// tick creates a timer for the next health check
func (m *MonitorModel) tick() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// formatMonitorDuration formats a duration in a human-readable way
func formatMonitorDuration(d time.Duration) string {
	if d < 0 {
		return "overdue"
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60

		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}

		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dh", hours)
}
