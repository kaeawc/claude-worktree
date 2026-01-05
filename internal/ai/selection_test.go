package ai

import (
	"testing"
)

func TestParseNumericIDs(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		limit    int
		expected []string
	}{
		{
			name:     "simple numeric list",
			output:   "150\n152\n145\n148\n146",
			limit:    5,
			expected: []string{"150", "152", "145", "148", "146"},
		},
		{
			name:     "with extra text",
			output:   "Here are the numbers:\n150\n152\nSome text\n145",
			limit:    5,
			expected: []string{"150", "152", "145"},
		},
		{
			name:     "limit to 3",
			output:   "150\n152\n145\n148\n146",
			limit:    3,
			expected: []string{"150", "152", "145"},
		},
		{
			name:     "empty output",
			output:   "",
			limit:    5,
			expected: []string{},
		},
		{
			name:     "no numeric lines",
			output:   "foo\nbar\nbaz",
			limit:    5,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseNumericIDs(tt.output, tt.limit)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseNumericIDs() returned %d items, expected %d", len(result), len(tt.expected))
				return
			}
			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("ParseNumericIDs()[%d] = %s, expected %s", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestParseLinearIDs(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		limit    int
		expected []string
	}{
		{
			name:     "simple Linear ID list",
			output:   "TEAM-42\nPROJ-123\nFOO-1",
			limit:    5,
			expected: []string{"TEAM-42", "PROJ-123", "FOO-1"},
		},
		{
			name:     "with extra text",
			output:   "Here are the issues:\nTEAM-42\nSome text\nPROJ-123",
			limit:    5,
			expected: []string{"TEAM-42", "PROJ-123"},
		},
		{
			name:     "limit to 2",
			output:   "TEAM-42\nPROJ-123\nFOO-1",
			limit:    2,
			expected: []string{"TEAM-42", "PROJ-123"},
		},
		{
			name:     "empty output",
			output:   "",
			limit:    5,
			expected: []string{},
		},
		{
			name:     "no valid Linear IDs",
			output:   "foo\n123\nbar-baz",
			limit:    5,
			expected: []string{},
		},
		{
			name:     "mixed case (should fail)",
			output:   "Team-42\nPROJ-123",
			limit:    5,
			expected: []string{"PROJ-123"}, // Team-42 is lowercase, should be rejected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLinearIDs(tt.output, tt.limit)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseLinearIDs() returned %d items, expected %d", len(result), len(tt.expected))
				return
			}
			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("ParseLinearIDs()[%d] = %s, expected %s", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"42", true},
		{"", false},
		{"abc", false},
		{"12a", false},
		{"1 2", false},
		{"-5", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isNumeric(tt.input)
			if result != tt.expected {
				t.Errorf("isNumeric(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsLinearID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"TEAM-42", true},
		{"PROJ-123", true},
		{"A-1", true},
		{"ABC123-456", true},
		{"", false},
		{"TEAM", false},
		{"team-42", false},  // lowercase
		{"TEAM-", false},    // no number
		{"-42", false},      // no prefix
		{"TEAM-ABC", false}, // non-numeric suffix
		{"TEAM 42", false},  // space instead of hyphen
		{"TEAM--42", false}, // double hyphen
		{"123-456", false},  // numeric prefix
		{"Te-Am-42", false}, // multiple hyphens
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isLinearID(tt.input)
			if result != tt.expected {
				t.Errorf("isLinearID(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
