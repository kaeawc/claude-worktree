package environment

// ProjectType represents the type of project detected
type ProjectType string

// Project type constants
const (
	// ProjectTypeNodeJS represents a Node.js project
	ProjectTypeNodeJS ProjectType = "nodejs"
	// ProjectTypePython represents a Python project
	ProjectTypePython ProjectType = "python"
	// ProjectTypeGo represents a Go project
	ProjectTypeGo ProjectType = "go"
	// ProjectTypeRuby represents a Ruby project
	ProjectTypeRuby ProjectType = "ruby"
	// ProjectTypeRust represents a Rust project
	ProjectTypeRust ProjectType = "rust"
	// ProjectTypeNone represents no detected project type
	ProjectTypeNone ProjectType = "none"
)

// PackageManager represents a detected package manager
type PackageManager string

// Package manager constants
const (
	// PackageManagerNPM represents the npm package manager for Node.js
	PackageManagerNPM PackageManager = "npm"
	// PackageManagerYarn represents the yarn package manager for Node.js
	PackageManagerYarn PackageManager = "yarn"
	// PackageManagerPNPM represents the pnpm package manager for Node.js
	PackageManagerPNPM PackageManager = "pnpm"
	// PackageManagerBun represents the bun package manager for Node.js
	PackageManagerBun PackageManager = "bun"

	// PackageManagerUV represents the uv package manager for Python
	PackageManagerUV PackageManager = "uv"
	// PackageManagerPoetry represents the poetry package manager for Python
	PackageManagerPoetry PackageManager = "poetry"
	// PackageManagerPip represents the pip package manager for Python
	PackageManagerPip PackageManager = "pip"

	// PackageManagerBundle represents the bundle package manager for Ruby
	PackageManagerBundle PackageManager = "bundle"

	// PackageManagerGoMod represents the go mod package manager for Go
	PackageManagerGoMod PackageManager = "go"

	// PackageManagerCargo represents the cargo package manager for Rust
	PackageManagerCargo PackageManager = "cargo"

	// PackageManagerNone represents no detected package manager
	PackageManagerNone PackageManager = "none"
)

// DetectionResult contains the results of project detection
type DetectionResult struct {
	ProjectType    ProjectType
	PackageManager PackageManager
	WorktreePath   string
}

// InstallResult contains the results of package installation
type InstallResult struct {
	Success bool
	Message string
	Error   error
}

// Detector interface for detecting project types and package managers
type Detector interface {
	// DetectProjectType detects the type of project in the given directory
	DetectProjectType(worktreePath string) (ProjectType, error)

	// DetectPackageManager detects the package manager for the project
	DetectPackageManager(worktreePath string, projectType ProjectType) (PackageManager, error)

	// Detect performs both project type and package manager detection
	Detect(worktreePath string) (*DetectionResult, error)
}

// Installer interface for installing dependencies
type Installer interface {
	// Install runs the package manager installation command
	Install(result *DetectionResult) *InstallResult

	// IsAvailable checks if the package manager command is available
	IsAvailable(pm PackageManager) bool
}
