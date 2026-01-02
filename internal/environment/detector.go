// Package environment provides functionality for detecting and setting up
// project environments including package manager detection and dependency installation.
package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RealDetector implements the Detector interface
type RealDetector struct {
	// ConfiguredPackageManager allows overriding auto-detection
	ConfiguredPackageManager string
}

// NewDetector creates a new RealDetector instance
func NewDetector(configuredPM string) *RealDetector {
	return &RealDetector{
		ConfiguredPackageManager: configuredPM,
	}
}

// DetectProjectType detects the type of project in the given directory
func (d *RealDetector) DetectProjectType(worktreePath string) (ProjectType, error) {
	// Check for Node.js
	if d.fileExists(filepath.Join(worktreePath, "package.json")) {
		return ProjectTypeNodeJS, nil
	}

	// Check for Go
	if d.fileExists(filepath.Join(worktreePath, "go.mod")) {
		return ProjectTypeGo, nil
	}

	// Check for Rust
	if d.fileExists(filepath.Join(worktreePath, "Cargo.toml")) {
		return ProjectTypeRust, nil
	}

	// Check for Ruby
	if d.fileExists(filepath.Join(worktreePath, "Gemfile")) {
		return ProjectTypeRuby, nil
	}

	// Check for Python - multiple possible files
	pythonFiles := []string{
		"requirements.txt",
		"pyproject.toml",
		"setup.py",
		"Pipfile",
		"uv.lock",
		"poetry.lock",
	}
	for _, file := range pythonFiles {
		if d.fileExists(filepath.Join(worktreePath, file)) {
			return ProjectTypePython, nil
		}
	}

	return ProjectTypeNone, nil
}

// DetectPackageManager detects the package manager for the project
func (d *RealDetector) DetectPackageManager(worktreePath string, projectType ProjectType) (PackageManager, error) {
	switch projectType {
	case ProjectTypeNodeJS:
		return d.detectNodeJSPackageManager(worktreePath)
	case ProjectTypePython:
		return d.detectPythonPackageManager(worktreePath)
	case ProjectTypeGo:
		return PackageManagerGoMod, nil
	case ProjectTypeRuby:
		return PackageManagerBundle, nil
	case ProjectTypeRust:
		return PackageManagerCargo, nil
	default:
		return PackageManagerNone, nil
	}
}

// Detect performs both project type and package manager detection
func (d *RealDetector) Detect(worktreePath string) (*DetectionResult, error) {
	projectType, err := d.DetectProjectType(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project type: %w", err)
	}

	if projectType == ProjectTypeNone {
		return &DetectionResult{
			ProjectType:    ProjectTypeNone,
			PackageManager: PackageManagerNone,
			WorktreePath:   worktreePath,
		}, nil
	}

	packageManager, err := d.DetectPackageManager(worktreePath, projectType)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package manager: %w", err)
	}

	return &DetectionResult{
		ProjectType:    projectType,
		PackageManager: packageManager,
		WorktreePath:   worktreePath,
	}, nil
}

// detectNodeJSPackageManager detects the Node.js package manager
// Priority: configured override > packageManager field in package.json > lock files (bun > pnpm > yarn > npm)
func (d *RealDetector) detectNodeJSPackageManager(worktreePath string) (PackageManager, error) {
	// Check for configured override first
	if d.ConfiguredPackageManager != "" {
		return PackageManager(d.ConfiguredPackageManager), nil
	}

	packageJSONPath := filepath.Join(worktreePath, "package.json")

	// Try to read packageManager field from package.json
	data, err := os.ReadFile(packageJSONPath) //nolint:gosec // File path is from worktree creation, not user input

	if err == nil {
		var pkgJSON struct {
			PackageManager string `json:"packageManager"`
		}

		if err := json.Unmarshal(data, &pkgJSON); err == nil && pkgJSON.PackageManager != "" {
			// packageManager field format: "npm@8.1.0" or "pnpm@7.0.0"
			pmName := strings.Split(pkgJSON.PackageManager, "@")[0]
			//nolint:goconst // These are JSON field values, not our constants
			switch pmName {
			case "bun":
				return PackageManagerBun, nil
			case "pnpm":
				return PackageManagerPNPM, nil
			case "yarn":
				return PackageManagerYarn, nil
			case "npm":
				return PackageManagerNPM, nil
			}
		}
	}

	// Fallback to lock file detection (prefer newer/faster tools)
	if d.fileExists(filepath.Join(worktreePath, "bun.lockb")) {
		return PackageManagerBun, nil
	}

	if d.fileExists(filepath.Join(worktreePath, "pnpm-lock.yaml")) {
		return PackageManagerPNPM, nil
	}

	if d.fileExists(filepath.Join(worktreePath, "yarn.lock")) {
		return PackageManagerYarn, nil
	}

	// Default to npm
	return PackageManagerNPM, nil
}

// detectPythonPackageManager detects the Python package manager
// Priority: configured override > uv > poetry > pip
func (d *RealDetector) detectPythonPackageManager(worktreePath string) (PackageManager, error) {
	// Check for configured override first
	if d.ConfiguredPackageManager != "" {
		return PackageManager(d.ConfiguredPackageManager), nil
	}

	// Check for uv (modern Python package manager)
	if d.fileExists(filepath.Join(worktreePath, "uv.lock")) {
		return PackageManagerUV, nil
	}

	// Check pyproject.toml for [tool.uv] section
	pyprojectPath := filepath.Join(worktreePath, "pyproject.toml")

	if d.fileExists(pyprojectPath) {
		data, err := os.ReadFile(pyprojectPath) //nolint:gosec // File path is from worktree creation, not user input

		if err == nil && strings.Contains(string(data), "[tool.uv]") {
			return PackageManagerUV, nil
		}
	}

	// Check for poetry
	if d.fileExists(filepath.Join(worktreePath, "poetry.lock")) {
		return PackageManagerPoetry, nil
	}

	// Default to pip
	return PackageManagerPip, nil
}

// fileExists checks if a file exists
func (d *RealDetector) fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return err == nil && !info.IsDir()
}
