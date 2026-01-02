// Package environment handles environment setup for worktrees (dependency installation)
package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Setup handles environment setup for a worktree (dependency installation)
type Setup struct {
	worktreePath string
}

// NewSetup creates a new environment setup handler
func NewSetup(worktreePath string) *Setup {
	return &Setup{worktreePath: worktreePath}
}

// Run detects the project type and installs dependencies
func (s *Setup) Run() error {
	detected := false

	// Try each project type
	if s.detectGoProject() {
		detected = true

		s.installGoModules()
	}

	if s.detectNodeProject() {
		detected = true

		s.installNodeModules()
	}

	if s.detectPythonProject() {
		detected = true

		s.installPythonDeps()
	}

	if s.detectRubyProject() {
		detected = true

		s.installRubyGems()
	}

	if s.detectRustProject() {
		detected = true

		s.buildRustProject()
	}

	if !detected {
		// No recognized project type found
		return nil
	}

	return nil
}

// Go project detection and setup
func (s *Setup) detectGoProject() bool {
	return fileExists(filepath.Join(s.worktreePath, "go.mod"))
}

func (s *Setup) installGoModules() {
	if !commandExists("go") {
		fmt.Println("⚠ go not found, skipping dependency installation")
		return
	}

	fmt.Println("\nDetected Go project (go.mod)")
	fmt.Println("Running go mod download...")

	cmd := exec.CommandContext(context.Background(), "go", "mod", "download")
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ go mod download had issues (continuing anyway)")
	} else {
		fmt.Println("✓ Dependencies downloaded")
	}
}

// Node.js project detection and setup
func (s *Setup) detectNodeProject() bool {
	return fileExists(filepath.Join(s.worktreePath, "package.json"))
}

func (s *Setup) installNodeModules() {
	fmt.Println("\nDetected Node.js project (package.json)")

	// Detect package manager
	pkgManager := s.detectNodePackageManager()

	switch pkgManager {
	case "bun":
		s.runNodeInstall("bun", []string{"install"})
	case "pnpm":
		s.runNodeInstall("pnpm", []string{"install", "--silent"})
	case "yarn":
		s.runNodeInstall("yarn", []string{"install", "--silent"})
	default:
		s.runNodeInstall("npm", []string{"install", "--silent"})
	}
}

func (s *Setup) detectNodePackageManager() string {
	// Check for lock files
	if fileExists(filepath.Join(s.worktreePath, "bun.lockb")) {
		return "bun"
	}

	if fileExists(filepath.Join(s.worktreePath, "pnpm-lock.yaml")) {
		return "pnpm"
	}

	if fileExists(filepath.Join(s.worktreePath, "yarn.lock")) {
		return "yarn"
	}

	return "npm"
}

func (s *Setup) runNodeInstall(manager string, args []string) {
	if !commandExists(manager) {
		fmt.Printf("⚠ %s not found, skipping dependency installation\n", manager)
		return
	}

	fmt.Printf("Running %s install...\n", manager)

	cmd := exec.CommandContext(context.Background(), manager, args...)
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠ %s install had issues (continuing anyway)\n", manager)
	} else {
		fmt.Printf("✓ Dependencies installed (%s)\n", manager)
	}
}

// Python project detection and setup
func (s *Setup) detectPythonProject() bool {
	return fileExists(filepath.Join(s.worktreePath, "requirements.txt")) ||
		fileExists(filepath.Join(s.worktreePath, "pyproject.toml"))
}

func (s *Setup) installPythonDeps() {
	// Check for uv (modern Python package manager)
	if s.detectUvProject() {
		s.runUvSync()
		return
	}

	// Check for Poetry
	if fileExists(filepath.Join(s.worktreePath, "poetry.lock")) && commandExists("poetry") {
		s.runPoetryInstall()
		return
	}

	// Fall back to pip
	if fileExists(filepath.Join(s.worktreePath, "requirements.txt")) {
		s.runPipInstall()
	}
}

func (s *Setup) detectUvProject() bool {
	if !commandExists("uv") {
		return false
	}

	// Check for uv.lock
	if fileExists(filepath.Join(s.worktreePath, "uv.lock")) {
		return true
	}

	// Check for [tool.uv] in pyproject.toml
	pyproject := filepath.Join(s.worktreePath, "pyproject.toml")
	if fileExists(pyproject) {
		// Clean the path and ensure it's within the worktree to prevent path traversal
		cleanPath := filepath.Clean(pyproject)
		cleanWorktree := filepath.Clean(s.worktreePath)

		if !strings.HasPrefix(cleanPath, cleanWorktree) {
			return false
		}

		content, err := os.ReadFile(cleanPath)

		if err == nil && strings.Contains(string(content), "[tool.uv]") {
			return true
		}
	}

	return false
}

func (s *Setup) runUvSync() {
	fmt.Println("\nDetected Python project (uv)")
	fmt.Println("Running uv sync...")

	cmd := exec.CommandContext(context.Background(), "uv", "sync")
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ uv sync had issues (continuing anyway)")
	} else {
		fmt.Println("✓ Dependencies installed (uv + .venv)")
	}
}

func (s *Setup) runPoetryInstall() {
	fmt.Println("\nDetected Python project (pyproject.toml)")
	fmt.Println("Running poetry install...")

	cmd := exec.CommandContext(context.Background(), "poetry", "install", "--quiet")
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ poetry install had issues (continuing anyway)")
	} else {
		fmt.Println("✓ Dependencies installed (poetry)")
	}
}

func (s *Setup) runPipInstall() {
	fmt.Println("\nDetected Python project (requirements.txt)")

	pipCmd := "pip"
	if commandExists("pip3") {
		pipCmd = "pip3"
	} else if !commandExists("pip") {
		fmt.Println("⚠ pip not found, skipping dependency installation")
		return
	}

	fmt.Println("Installing Python dependencies...")

	cmd := exec.CommandContext(context.Background(), pipCmd, "install", "-q", "-r", "requirements.txt")
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ pip install had issues (continuing anyway)")
	} else {
		fmt.Println("✓ Dependencies installed (pip)")
	}
}

// Ruby project detection and setup
func (s *Setup) detectRubyProject() bool {
	return fileExists(filepath.Join(s.worktreePath, "Gemfile"))
}

func (s *Setup) installRubyGems() {
	if !commandExists("bundle") {
		fmt.Println("⚠ bundle not found, skipping dependency installation")
		return
	}

	fmt.Println("\nDetected Ruby project (Gemfile)")
	fmt.Println("Running bundle install...")

	cmd := exec.CommandContext(context.Background(), "bundle", "install", "--quiet")
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ bundle install had issues (continuing anyway)")
	} else {
		fmt.Println("✓ Dependencies installed")
	}
}

// Rust project detection and setup
func (s *Setup) detectRustProject() bool {
	return fileExists(filepath.Join(s.worktreePath, "Cargo.toml"))
}

func (s *Setup) buildRustProject() {
	if !commandExists("cargo") {
		fmt.Println("⚠ cargo not found, skipping dependency installation")
		return
	}

	fmt.Println("\nDetected Rust project (Cargo.toml)")
	fmt.Println("Running cargo check...")

	cmd := exec.CommandContext(context.Background(), "cargo", "check", "--quiet")
	cmd.Dir = s.worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ cargo check had issues (continuing anyway)")
	} else {
		fmt.Println("✓ Dependencies downloaded")
	}
}

// Utility functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
