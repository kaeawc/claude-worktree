package git

import (
	"fmt"
	"strings"
	"testing"
)

func TestRandomBranchName(t *testing.T) {
	// Generate a random branch name
	branchName := RandomBranchName()

	// Check that it starts with "work/"
	if !strings.HasPrefix(branchName, "work/") {
		t.Errorf("Expected branch name to start with 'work/', got: %s", branchName)
	}

	// Check that it has the correct format: work/color-adjective-animal
	parts := strings.Split(strings.TrimPrefix(branchName, "work/"), "-")
	if len(parts) != 3 {
		t.Errorf("Expected branch name to have 3 parts (color-adjective-animal), got %d parts: %s", len(parts), branchName)
	}

	// Verify each part is non-empty
	for i, part := range parts {
		if part == "" {
			t.Errorf("Expected part %d to be non-empty in branch name: %s", i, branchName)
		}
	}
}

func TestRandomBranchNameUniqueness(t *testing.T) {
	// Generate multiple branch names and check they're reasonably diverse
	names := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		name := RandomBranchName()
		names[name] = true
	}

	// With our word lists, we should get mostly unique names
	// (20 colors * 30 adjectives * 37 animals = 22,200 combinations)
	// So 100 iterations should produce mostly unique names
	uniqueCount := len(names)
	if uniqueCount < 90 {
		t.Errorf("Expected at least 90 unique names out of %d iterations, got %d", iterations, uniqueCount)
	}
}

func TestRandomBranchNameFormat(t *testing.T) {
	// Test that the generated name components come from our word lists
	branchName := RandomBranchName()
	parts := strings.Split(strings.TrimPrefix(branchName, "work/"), "-")

	if len(parts) != 3 {
		t.Fatalf("Expected 3 parts, got %d: %s", len(parts), branchName)
	}

	color := parts[0]
	adjective := parts[1]
	animal := parts[2]

	// Check color is in the list
	colorFound := false
	for _, c := range colors {
		if c == color {
			colorFound = true
			break
		}
	}
	if !colorFound {
		t.Errorf("Color '%s' not found in colors list", color)
	}

	// Check adjective is in the list
	adjectiveFound := false
	for _, a := range adjectives {
		if a == adjective {
			adjectiveFound = true
			break
		}
	}
	if !adjectiveFound {
		t.Errorf("Adjective '%s' not found in adjectives list", adjective)
	}

	// Check animal is in the list
	animalFound := false
	for _, a := range animals {
		if a == animal {
			animalFound = true
			break
		}
	}
	if !animalFound {
		t.Errorf("Animal '%s' not found in animals list", animal)
	}
}

func TestGenerateUniqueBranchName(t *testing.T) {
	// Create a custom executor that tracks calls and returns errors for show-ref
	callCount := 0
	customExec := &customTestExecutor{
		executeInDirFunc: func(dir string, args ...string) (string, error) {
			callCount++
			// If it's a show-ref command (checking if branch exists), return error (branch doesn't exist)
			if len(args) > 0 && args[0] == "show-ref" {
				return "", fmt.Errorf("branch not found")
			}
			return "", nil
		},
	}

	repo := &Repository{
		executor: customExec,
		RootPath: "/test/repo",
	}

	// Generate a unique branch name
	branchName, err := repo.GenerateUniqueBranchName(10)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify it's a valid branch name
	if !strings.HasPrefix(branchName, "work/") {
		t.Errorf("Expected branch name to start with 'work/', got: %s", branchName)
	}

	// Verify that BranchExists was called at least once
	if callCount < 1 {
		t.Errorf("Expected at least 1 call to check branch existence, got: %d", callCount)
	}
}

func TestGenerateUniqueBranchNameMaxAttemptsExceeded(t *testing.T) {
	// Create a custom executor that always reports branches exist
	customExec := &customTestExecutor{
		executeInDirFunc: func(dir string, args ...string) (string, error) {
			// If it's a show-ref command (checking if branch exists), return success (branch exists)
			if len(args) > 0 && args[0] == "show-ref" {
				return "abc123 refs/heads/test", nil
			}
			return "", nil
		},
	}

	repo := &Repository{
		executor: customExec,
		RootPath: "/test/repo",
	}

	// Try to generate a unique branch name with low max attempts
	_, err := repo.GenerateUniqueBranchName(3)
	if err == nil {
		t.Error("Expected error when max attempts exceeded, got nil")
	}

	expectedError := "failed to generate unique branch name after 3 attempts"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got: '%s'", expectedError, err.Error())
	}
}

// customTestExecutor is a custom executor for testing that allows custom logic
type customTestExecutor struct {
	executeFunc      func(args ...string) (string, error)
	executeInDirFunc func(dir string, args ...string) (string, error)
}

func (e *customTestExecutor) Execute(args ...string) (string, error) {
	if e.executeFunc != nil {
		return e.executeFunc(args...)
	}
	return "", nil
}

func (e *customTestExecutor) ExecuteInDir(dir string, args ...string) (string, error) {
	if e.executeInDirFunc != nil {
		return e.executeInDirFunc(dir, args...)
	}
	return "", nil
}
