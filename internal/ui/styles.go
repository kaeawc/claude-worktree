package ui

import "github.com/charmbracelet/lipgloss"

// Common styles used across UI components
// Updated to use gum color scheme (ANSI 1-6) from theme.go
var (
	// Color palette - using gum color scheme
	primaryColor = ColorCyan   // Cyan for highlights/primary actions
	successColor = ColorGreen  // Green for success
	errorColor   = ColorRed    // Red for errors
	warningColor = ColorYellow // Yellow for warnings
	subtleColor  = lipgloss.Color("241") // Gray for subtle text

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
			BorderForeground(ColorBlue). // Blue for borders/info
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)
)
