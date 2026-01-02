package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	promptStateBranch = "branch"
	promptStateDone   = "done"
	keyCtrlC          = "ctrl+c"
)

var (
	cleanupTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	cleanupWarningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	cleanupQuestionStyle = lipgloss.NewStyle().Bold(true)
	cleanupHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// CleanupPromptModel represents a prompt for cleaning up a worktree
type CleanupPromptModel struct {
	WorktreePath    string
	Branch          string
	CleanupReason   string
	UnpushedCount   int
	DeleteBranch    bool
	Confirmed       bool
	PromptState     string // "confirm" or "branch" or "done"
	Canceled        bool
	SkipAutoCleanup bool // If true, skip automatic cleanup and use interactive prompt
}

// NewCleanupPrompt creates a new cleanup prompt
func NewCleanupPrompt(worktreePath, branch, reason string, unpushedCount int, skipAutoCleanup bool) CleanupPromptModel {
	return CleanupPromptModel{
		WorktreePath:    worktreePath,
		Branch:          branch,
		CleanupReason:   reason,
		UnpushedCount:   unpushedCount,
		PromptState:     "confirm",
		SkipAutoCleanup: skipAutoCleanup,
	}
}

// Init initializes the cleanup prompt
func (m CleanupPromptModel) Init() tea.Cmd {
	return nil
}

// Update handles cleanup prompt updates
func (m CleanupPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.PromptState {
		case "confirm":
			switch msg.String() {
			case "y", "Y":
				// User confirmed cleanup
				if m.Branch != "" {
					// Move to branch deletion prompt
					m.PromptState = promptStateBranch
					return m, nil
				}
				// No branch, just confirm cleanup
				m.Confirmed = true
				m.PromptState = promptStateDone
				return m, tea.Quit

			case "n", "N", "q", keyCtrlC, "esc":
				// User canceled
				m.Canceled = true
				m.PromptState = promptStateDone
				return m, tea.Quit
			}

		case promptStateBranch:
			switch msg.String() {
			case "y", "Y":
				// User confirmed branch deletion
				m.DeleteBranch = true
				m.Confirmed = true
				m.PromptState = promptStateDone
				return m, tea.Quit

			case "n", "N":
				// User declined branch deletion
				m.DeleteBranch = false
				m.Confirmed = true
				m.PromptState = promptStateDone
				return m, tea.Quit

			case "q", keyCtrlC, "esc":
				// User canceled
				m.Canceled = true
				m.PromptState = promptStateDone
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View renders the cleanup prompt
func (m CleanupPromptModel) View() string {
	if m.PromptState == promptStateDone {
		return ""
	}

	var s string

	// Show worktree info
	s += cleanupTitleStyle.Render(fmt.Sprintf("Cleanup worktree: %s", m.WorktreePath)) + "\n"
	if m.Branch != "" {
		s += fmt.Sprintf("Branch: %s\n", m.Branch)
	}
	if m.CleanupReason != "" {
		s += fmt.Sprintf("Reason: %s\n", m.CleanupReason)
	}

	// Show warning if there are unpushed commits
	if m.UnpushedCount > 0 {
		s += "\n" + cleanupWarningStyle.Render(fmt.Sprintf("⚠ Warning: %d unpushed commit(s)", m.UnpushedCount)) + "\n"
	}

	s += "\n"

	// Show appropriate prompt based on state
	switch m.PromptState {
	case "confirm":
		s += cleanupQuestionStyle.Render("Remove this worktree?") + " "
		s += cleanupHintStyle.Render("[y/n]") + " "

	case promptStateBranch:
		s += cleanupQuestionStyle.Render(fmt.Sprintf("Delete branch '%s'?", m.Branch)) + " "
		s += cleanupHintStyle.Render("[y/n]") + " "
	}

	return s
}

// ShouldDeleteBranch returns true if the user confirmed branch deletion
func (m CleanupPromptModel) ShouldDeleteBranch() bool {
	return m.DeleteBranch
}

// WasConfirmed returns true if the user confirmed the cleanup
func (m CleanupPromptModel) WasConfirmed() bool {
	return m.Confirmed
}

// WasCanceled returns true if the user canceled the cleanup
func (m CleanupPromptModel) WasCanceled() bool {
	return m.Canceled
}

// CleanupConfirmationModel represents a batch cleanup confirmation
type CleanupConfirmationModel struct {
	MergedCount int
	StaleCount  int
	Confirmed   bool
	Canceled    bool
}

// NewCleanupConfirmation creates a new cleanup confirmation prompt
func NewCleanupConfirmation(mergedCount, staleCount int) CleanupConfirmationModel {
	return CleanupConfirmationModel{
		MergedCount: mergedCount,
		StaleCount:  staleCount,
	}
}

// Init initializes the confirmation prompt
func (m CleanupConfirmationModel) Init() tea.Cmd {
	return nil
}

// Update handles confirmation updates
func (m CleanupConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.Confirmed = true
			return m, tea.Quit

		case "n", "N", "q", keyCtrlC, "esc":
			m.Canceled = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the confirmation prompt
func (m CleanupConfirmationModel) View() string {
	var s string

	s += cleanupTitleStyle.Render("Auto-cleanup for merged worktrees") + "\n\n"

	if m.MergedCount > 0 {
		s += fmt.Sprintf("Found %d merged worktree(s) ready for automatic cleanup\n", m.MergedCount)
	}

	if m.StaleCount > 0 {
		s += fmt.Sprintf("Found %d stale worktree(s) that will require interactive confirmation\n", m.StaleCount)
	}

	if m.MergedCount == 0 && m.StaleCount == 0 {
		s += "No worktrees found that need cleanup.\n"
		return s
	}

	s += "\n"
	s += cleanupQuestionStyle.Render("Proceed with cleanup?") + " "
	s += cleanupHintStyle.Render("[y/n]") + " "

	return s
}

// WasConfirmed returns true if the user confirmed
func (m CleanupConfirmationModel) WasConfirmed() bool {
	return m.Confirmed
}

// WasCanceled returns true if the user canceled
func (m CleanupConfirmationModel) WasCanceled() bool {
	return m.Canceled
}
