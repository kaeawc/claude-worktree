package environment

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RealInstaller implements the Installer interface
type RealInstaller struct {
	// OnProgress is called with progress messages during installation
	OnProgress func(message string)
}

// NewInstaller creates a new RealInstaller instance
func NewInstaller(onProgress func(string)) *RealInstaller {
	return &RealInstaller{
		OnProgress: onProgress,
	}
}

// Install runs the package manager installation command
func (i *RealInstaller) Install(result *DetectionResult) *InstallResult {
	if result.ProjectType == ProjectTypeNone || result.PackageManager == PackageManagerNone {
		return &InstallResult{
			Success: true,
			Message: "No package manager detected, skipping installation",
		}
	}

	// Check if package manager is available
	if !i.IsAvailable(result.PackageManager) {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Package manager '%s' not found in PATH", result.PackageManager),
			Error:   fmt.Errorf("package manager '%s' not available", result.PackageManager),
		}
	}

	// Get install command
	cmd, args := i.getInstallCommand(result.PackageManager)
	if cmd == "" {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Unknown package manager: %s", result.PackageManager),
			Error:   fmt.Errorf("unknown package manager: %s", result.PackageManager),
		}
	}

	// Notify progress
	if i.OnProgress != nil {
		i.OnProgress(fmt.Sprintf("Installing dependencies with %s...", result.PackageManager))
	}

	// Execute install command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Dir = result.WorktreePath

	// Capture output
	output, err := execCmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Failed to install dependencies: %s", strings.TrimSpace(string(output))),
			Error:   err,
		}
	}

	return &InstallResult{
		Success: true,
		Message: fmt.Sprintf("Successfully installed dependencies with %s", result.PackageManager),
	}
}

// IsAvailable checks if the package manager command is available
func (i *RealInstaller) IsAvailable(pm PackageManager) bool {
	cmd := i.getCommandName(pm)
	if cmd == "" {
		return false
	}

	_, err := exec.LookPath(cmd)

	return err == nil
}

// getCommandName returns the command name for a package manager
//
//nolint:goconst // Command names are OS-level commands, not constants
func (i *RealInstaller) getCommandName(pm PackageManager) string {
	switch pm {
	case PackageManagerNPM:
		return "npm"
	case PackageManagerYarn:
		return "yarn"
	case PackageManagerPNPM:
		return "pnpm"
	case PackageManagerBun:
		return "bun"
	case PackageManagerUV:
		return "uv"
	case PackageManagerPoetry:
		return "poetry"
	case PackageManagerPip:
		return "pip"
	case PackageManagerBundle:
		return "bundle"
	case PackageManagerGoMod:
		return "go"
	case PackageManagerCargo:
		return "cargo"
	default:
		return ""
	}
}

// getInstallCommand returns the command and args for installing dependencies
func (i *RealInstaller) getInstallCommand(pm PackageManager) (string, []string) {
	switch pm {
	case PackageManagerNPM:
		return "npm", []string{"install", "--silent"}
	case PackageManagerYarn:
		return "yarn", []string{"install", "--silent"}
	case PackageManagerPNPM:
		return "pnpm", []string{"install", "--silent"}
	case PackageManagerBun:
		return "bun", []string{"install", "--silent"}
	case PackageManagerUV:
		return "uv", []string{"sync"}
	case PackageManagerPoetry:
		return "poetry", []string{"install", "--quiet"}
	case PackageManagerPip:
		return "pip", []string{"install", "-r", "requirements.txt", "--quiet"}
	case PackageManagerBundle:
		return "bundle", []string{"install", "--quiet"}
	case PackageManagerGoMod:
		return "go", []string{"mod", "download"}
	case PackageManagerCargo:
		return "cargo", []string{"fetch", "--quiet"}
	default:
		return "", nil
	}
}
