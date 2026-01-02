package session

import (
	"os"

	"github.com/kaeawc/auto-worktree/internal/git"
)

// TmuxConfig holds tmux session configuration
type TmuxConfig struct {
	Enabled        bool
	AutoInstall    bool
	Layout         string
	Shell          string
	WindowCount    int
	IdleThreshold  int // minutes
	LogCommands    bool
	PostCreateHook string
	PostResumeHook string
	PreKillHook    string
}

// DefaultTmuxConfig returns default tmux configuration
func DefaultTmuxConfig() *TmuxConfig {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	return &TmuxConfig{
		Enabled:       true,
		AutoInstall:   true,
		Layout:        "tiled",
		Shell:         shell,
		WindowCount:   1,
		IdleThreshold: 120,
		LogCommands:   true,
	}
}

// tmuxConfigLoader handles loading tmux configuration from git config
type tmuxConfigLoader struct {
	repo *git.Repository
	cfg  *TmuxConfig
}

// newTmuxConfigLoader creates a new loader with defaults
func newTmuxConfigLoader(repo *git.Repository) *tmuxConfigLoader {
	return &tmuxConfigLoader{
		repo: repo,
		cfg:  DefaultTmuxConfig(),
	}
}

// load loads all configuration values and returns the config
func (l *tmuxConfigLoader) load() *TmuxConfig {
	if l.repo == nil {
		return l.cfg
	}

	l.loadBooleanConfigs()
	l.loadStringConfigs()
	l.loadIntegerConfigs()

	return l.cfg
}

// loadBooleanConfigs loads all boolean configuration keys
func (l *tmuxConfigLoader) loadBooleanConfigs() {
	l.cfg.Enabled = l.repo.Config.GetBoolWithDefault(
		git.ConfigTmuxEnabled, l.cfg.Enabled, git.ConfigScopeAuto)

	l.cfg.AutoInstall = l.repo.Config.GetBoolWithDefault(
		git.ConfigTmuxAutoInstall, l.cfg.AutoInstall, git.ConfigScopeAuto)

	l.cfg.LogCommands = l.repo.Config.GetBoolWithDefault(
		git.ConfigTmuxLogCommands, l.cfg.LogCommands, git.ConfigScopeAuto)
}

// loadStringConfigs loads all string configuration keys
func (l *tmuxConfigLoader) loadStringConfigs() {
	l.cfg.Layout = l.repo.Config.GetWithDefault(
		git.ConfigTmuxLayout, l.cfg.Layout, git.ConfigScopeAuto)

	l.cfg.Shell = l.repo.Config.GetWithDefault(
		git.ConfigTmuxShell, l.cfg.Shell, git.ConfigScopeAuto)

	l.cfg.PostCreateHook = l.repo.Config.GetWithDefault(
		git.ConfigTmuxPostCreateHook, l.cfg.PostCreateHook, git.ConfigScopeAuto)

	l.cfg.PostResumeHook = l.repo.Config.GetWithDefault(
		git.ConfigTmuxPostResumeHook, l.cfg.PostResumeHook, git.ConfigScopeAuto)

	l.cfg.PreKillHook = l.repo.Config.GetWithDefault(
		git.ConfigTmuxPreKillHook, l.cfg.PreKillHook, git.ConfigScopeAuto)
}

// loadIntegerConfigs loads all integer configuration keys
func (l *tmuxConfigLoader) loadIntegerConfigs() {
	l.cfg.WindowCount = l.repo.Config.GetIntWithDefault(
		git.ConfigTmuxWindowCount, l.cfg.WindowCount, git.ConfigScopeAuto)

	l.cfg.IdleThreshold = l.repo.Config.GetIntWithDefault(
		git.ConfigTmuxIdleThreshold, l.cfg.IdleThreshold, git.ConfigScopeAuto)
}

// LoadTmuxConfig loads tmux configuration from git config
func LoadTmuxConfig(repo *git.Repository) (*TmuxConfig, error) {
	loader := newTmuxConfigLoader(repo)
	return loader.load(), nil
}
