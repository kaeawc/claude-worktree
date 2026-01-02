package session

import (
	"fmt"
	"time"

	"github.com/kaeawc/auto-worktree/internal/environment"
)

// InstallDependencies automatically detects and installs dependencies for a session
func InstallDependencies(metadata *Metadata, onProgress func(string)) error {
	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	// Set up progress callback
	progressCallback := onProgress
	if progressCallback == nil {
		progressCallback = func(string) {} // No-op if not provided
	}

	progressCallback(fmt.Sprintf("Setting up dependencies for %s", metadata.SessionName))

	// Set up environment with auto-install
	opts := &environment.SetupOptions{
		AutoInstall: true,
		OnProgress:  progressCallback,
		OnWarning: func(msg string) {
			progressCallback(fmt.Sprintf("Warning: %s", msg))
		},
	}

	// Run environment setup
	if err := environment.Setup(metadata.WorktreePath, opts); err != nil {
		return fmt.Errorf("failed to set up environment: %w", err)
	}

	// Detect and record the project type and package manager
	detector := environment.NewDetector("")
	result, err := detector.Detect(metadata.WorktreePath)

	if err != nil {
		progressCallback(fmt.Sprintf("Note: Could not detect project type: %v", err))
		return nil // Don't fail on detection errors
	}

	if result.ProjectType != environment.ProjectTypeNone {
		now := time.Now()
		metadata.Dependencies = DependenciesInfo{
			Installed:      true,
			ProjectType:    string(result.ProjectType),
			PackageManager: string(result.PackageManager),
			InstalledAt:    &now,
		}

		progressCallback(fmt.Sprintf("Installed dependencies using %s", result.PackageManager))
	}

	return nil
}
