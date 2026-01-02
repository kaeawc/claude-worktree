package environment

import (
	"fmt"
)

// SetupOptions contains options for environment setup
type SetupOptions struct {
	// AutoInstall controls whether to automatically install dependencies
	AutoInstall bool

	// ConfiguredPackageManager overrides auto-detection if set
	ConfiguredPackageManager string

	// OnProgress is called with progress messages
	OnProgress func(message string)

	// OnWarning is called with warning messages
	OnWarning func(message string)
}

// Setup performs complete environment setup for a worktree
func Setup(worktreePath string, opts *SetupOptions) error {
	if opts == nil {
		opts = &SetupOptions{
			AutoInstall: true,
		}
	}

	// Skip if auto-install is disabled
	if !opts.AutoInstall {
		if opts.OnProgress != nil {
			opts.OnProgress("Auto-install disabled, skipping environment setup")
		}

		return nil
	}

	// Create detector
	detector := NewDetector(opts.ConfiguredPackageManager)

	// Detect project type and package manager
	if opts.OnProgress != nil {
		opts.OnProgress("Detecting project type...")
	}

	result, err := detector.Detect(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to detect project: %w", err)
	}

	// Nothing to install
	if result.ProjectType == ProjectTypeNone {
		if opts.OnProgress != nil {
			opts.OnProgress("No package manager detected, skipping installation")
		}

		return nil
	}

	if opts.OnProgress != nil {
		opts.OnProgress(fmt.Sprintf("Detected %s project with %s package manager", result.ProjectType, result.PackageManager))
	}

	// Create installer
	installer := NewInstaller(opts.OnProgress)

	// Run installation
	installResult := installer.Install(result)

	if !installResult.Success {
		// Warn but don't fail
		warningMsg := fmt.Sprintf("Warning: %s", installResult.Message)

		if opts.OnWarning != nil {
			opts.OnWarning(warningMsg)
		}

		if opts.OnProgress != nil {
			opts.OnProgress("Continuing anyway...")
		}

		return nil // Don't return error, just warn
	}

	if opts.OnProgress != nil {
		opts.OnProgress(installResult.Message)
	}

	return nil
}
