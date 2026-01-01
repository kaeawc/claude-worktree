package ui

import "github.com/charmbracelet/lipgloss"

// Common styles used across UI components
var (
	// Color palette
	primaryColor = lipgloss.Color("170") // Purple
	successColor = lipgloss.Color("42")  // Green
	errorColor   = lipgloss.Color("196") // Red
	warningColor = lipgloss.Color("214") // Orange
	subtleColor  = lipgloss.Color("241") // Gray

	// Text styles
	TitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	SuccessStyle = lipgloss.NewStyle().Foreground(successColor)
	ErrorStyle   = lipgloss.NewStyle().Foreground(errorColor)
	WarningStyle = lipgloss.NewStyle().Foreground(warningColor)
	SubtleStyle  = lipgloss.NewStyle().Foreground(subtleColor)
	BoldStyle    = lipgloss.NewStyle().Bold(true)

	// Layout styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)
)
