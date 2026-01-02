package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestGetWorktreeAgeColor(t *testing.T) {
	tests := []struct {
		name     string
		age      time.Duration
		expected lipgloss.Color
	}{
		{
			name:     "recent worktree (less than 1 day)",
			age:      12 * time.Hour,
			expected: ColorGreen,
		},
		{
			name:     "1-4 days old",
			age:      2 * 24 * time.Hour,
			expected: ColorYellow,
		},
		{
			name:     "stale worktree (more than 4 days)",
			age:      5 * 24 * time.Hour,
			expected: ColorRed,
		},
		{
			name:     "exactly 1 day",
			age:      24 * time.Hour,
			expected: ColorYellow,
		},
		{
			name:     "exactly 4 days",
			age:      4 * 24 * time.Hour,
			expected: ColorYellow,
		},
		{
			name:     "very fresh (minutes)",
			age:      30 * time.Minute,
			expected: ColorGreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorktreeAgeColor(tt.age)
			if result != tt.expected {
				t.Errorf("GetWorktreeAgeColor(%v) = %v, want %v", tt.age, result, tt.expected)
			}
		})
	}
}

func TestGetWorktreeAgeStyle(t *testing.T) {
	age := 3 * 24 * time.Hour
	style := GetWorktreeAgeStyle(age)

	// Verify the style has the correct foreground color
	expectedColor := ColorYellow
	if style.GetForeground() != expectedColor {
		t.Errorf("GetWorktreeAgeStyle(%v) foreground = %v, want %v", age, style.GetForeground(), expectedColor)
	}
}
