package ui

import (
	"testing"
)

func TestNewProviderMenuModel(t *testing.T) {
	model := NewProviderMenuModel()

	if model == nil {
		t.Fatal("NewProviderMenuModel() returned nil")
	}

	if model.choice != ProviderNone {
		t.Errorf("Initial choice = %v, want %v", model.choice, ProviderNone)
	}

	// Verify we have the expected number of provider options
	expectedItemCount := 4 // GitHub, GitLab, JIRA, Linear
	if len(model.list.Items()) != expectedItemCount {
		t.Errorf("Provider item count = %d, want %d", len(model.list.Items()), expectedItemCount)
	}
}

func TestNewAIToolMenuModel(t *testing.T) {
	model := NewAIToolMenuModel()

	if model == nil {
		t.Fatal("NewAIToolMenuModel() returned nil")
	}

	if model.choice != AIToolNone {
		t.Errorf("Initial choice = %v, want %v", model.choice, AIToolNone)
	}

	// Verify we have the expected number of AI tool options
	expectedItemCount := 4 // Claude, Codex, Gemini, Skip
	if len(model.list.Items()) != expectedItemCount {
		t.Errorf("AI tool item count = %d, want %d", len(model.list.Items()), expectedItemCount)
	}
}

func TestProviderValues(t *testing.T) {
	providers := map[string]Provider{
		"github": ProviderGitHub,
		"gitlab": ProviderGitLab,
		"jira":   ProviderJira,
		"linear": ProviderLinear,
	}

	for name, provider := range providers {
		if string(provider) != name {
			t.Errorf("Provider %s has value %q, want %q", name, provider, name)
		}
	}
}

func TestAIToolValues(t *testing.T) {
	tools := map[string]AITool{
		"claude": AIToolClaude,
		"codex":  AIToolCodex,
		"gemini": AIToolGemini,
		"skip":   AIToolSkip,
	}

	for name, tool := range tools {
		if string(tool) != name {
			t.Errorf("AI tool %s has value %q, want %q", name, tool, name)
		}
	}
}
