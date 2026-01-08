package git

import (
	"testing"
)

func TestHealthCheckSeverity_String(t *testing.T) {
	tests := []struct {
		severity HealthCheckSeverity
		expected string
	}{
		{SeverityOK, "OK"},
		{SeverityWarning, "Warning"},
		{SeverityError, "Error"},
		{SeverityCritical, "Critical"},
		{HealthCheckSeverity(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.expected {
				t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHealthCheckResult_GetMaxSeverity(t *testing.T) {
	tests := []struct {
		name     string
		issues   []HealthCheckIssue
		expected HealthCheckSeverity
	}{
		{
			name:     "no issues",
			issues:   []HealthCheckIssue{},
			expected: SeverityOK,
		},
		{
			name: "all warnings",
			issues: []HealthCheckIssue{
				{Severity: SeverityWarning},
				{Severity: SeverityWarning},
			},
			expected: SeverityWarning,
		},
		{
			name: "mixed severities",
			issues: []HealthCheckIssue{
				{Severity: SeverityOK},
				{Severity: SeverityWarning},
				{Severity: SeverityError},
			},
			expected: SeverityError,
		},
		{
			name: "critical present",
			issues: []HealthCheckIssue{
				{Severity: SeverityWarning},
				{Severity: SeverityCritical},
				{Severity: SeverityError},
			},
			expected: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &HealthCheckResult{
				Issues: tt.issues,
			}

			if got := result.GetMaxSeverity(); got != tt.expected {
				t.Errorf("GetMaxSeverity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHealthCheckResult_GetRepairableIssues(t *testing.T) {
	issues := []HealthCheckIssue{
		{Description: "issue1", Repairable: true},
		{Description: "issue2", Repairable: false},
		{Description: "issue3", Repairable: true},
		{Description: "issue4", Repairable: false},
	}

	result := &HealthCheckResult{
		Issues: issues,
	}

	repairable := result.GetRepairableIssues()

	if len(repairable) != 2 {
		t.Errorf("GetRepairableIssues() returned %d issues, want 2", len(repairable))
	}

	for _, issue := range repairable {
		if !issue.Repairable {
			t.Errorf("GetRepairableIssues() returned non-repairable issue: %v", issue)
		}
	}
}
