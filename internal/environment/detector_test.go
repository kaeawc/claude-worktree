package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectType(t *testing.T) {
	// Create temporary directory for tests
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		files        []string
		expectedType ProjectType
	}{
		{
			name:         "nodejs project",
			files:        []string{"package.json"},
			expectedType: ProjectTypeNodeJS,
		},
		{
			name:         "go project",
			files:        []string{"go.mod"},
			expectedType: ProjectTypeGo,
		},
		{
			name:         "rust project",
			files:        []string{"Cargo.toml"},
			expectedType: ProjectTypeRust,
		},
		{
			name:         "ruby project",
			files:        []string{"Gemfile"},
			expectedType: ProjectTypeRuby,
		},
		{
			name:         "python project with requirements.txt",
			files:        []string{"requirements.txt"},
			expectedType: ProjectTypePython,
		},
		{
			name:         "python project with pyproject.toml",
			files:        []string{"pyproject.toml"},
			expectedType: ProjectTypePython,
		},
		{
			name:         "python project with setup.py",
			files:        []string{"setup.py"},
			expectedType: ProjectTypePython,
		},
		{
			name:         "no project files",
			files:        []string{},
			expectedType: ProjectTypeNone,
		},
		{
			name:         "nodejs takes priority over python",
			files:        []string{"package.json", "requirements.txt"},
			expectedType: ProjectTypeNodeJS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(testDir, file)
				if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			// Test detection
			detector := NewDetector("")
			projectType, err := detector.DetectProjectType(testDir)
			if err != nil {
				t.Fatalf("DetectProjectType() error = %v", err)
			}

			if projectType != tt.expectedType {
				t.Errorf("DetectProjectType() = %v, want %v", projectType, tt.expectedType)
			}
		})
	}
}

//nolint:gocognit // Test function with multiple scenarios
func TestDetectNodeJSPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		lockFiles    []string
		packageJSON  string
		expectedPM   PackageManager
		configuredPM string
	}{
		{
			name:       "bun lock file",
			lockFiles:  []string{"bun.lockb"},
			expectedPM: PackageManagerBun,
		},
		{
			name:       "pnpm lock file",
			lockFiles:  []string{"pnpm-lock.yaml"},
			expectedPM: PackageManagerPNPM,
		},
		{
			name:       "yarn lock file",
			lockFiles:  []string{"yarn.lock"},
			expectedPM: PackageManagerYarn,
		},
		{
			name:       "no lock file defaults to npm",
			lockFiles:  []string{},
			expectedPM: PackageManagerNPM,
		},
		{
			name:        "packageManager field in package.json",
			packageJSON: `{"packageManager": "pnpm@8.0.0"}`,
			expectedPM:  PackageManagerPNPM,
		},
		{
			name:        "packageManager field overrides lock file",
			lockFiles:   []string{"yarn.lock"},
			packageJSON: `{"packageManager": "bun@1.0.0"}`,
			expectedPM:  PackageManagerBun,
		},
		{
			name:       "bun priority over pnpm",
			lockFiles:  []string{"bun.lockb", "pnpm-lock.yaml"},
			expectedPM: PackageManagerBun,
		},
		{
			name:         "configured package manager overrides detection",
			lockFiles:    []string{"yarn.lock"},
			configuredPM: "npm",
			expectedPM:   PackageManagerNPM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create package.json
			packageJSONContent := "{}"
			if tt.packageJSON != "" {
				packageJSONContent = tt.packageJSON
			}
			if err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSONContent), 0644); err != nil {
				t.Fatalf("Failed to create package.json: %v", err)
			}

			// Create lock files
			for _, file := range tt.lockFiles {
				filePath := filepath.Join(testDir, file)
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create lock file %s: %v", file, err)
				}
			}

			// Test detection
			detector := NewDetector(tt.configuredPM)
			pm, err := detector.detectNodeJSPackageManager(testDir)
			if err != nil {
				t.Fatalf("detectNodeJSPackageManager() error = %v", err)
			}

			if pm != tt.expectedPM {
				t.Errorf("detectNodeJSPackageManager() = %v, want %v", pm, tt.expectedPM)
			}
		})
	}
}

func TestDetectPythonPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		files        map[string]string // filename -> content
		expectedPM   PackageManager
		configuredPM string
	}{
		{
			name:       "uv lock file",
			files:      map[string]string{"uv.lock": ""},
			expectedPM: PackageManagerUV,
		},
		{
			name: "pyproject.toml with [tool.uv]",
			files: map[string]string{
				"pyproject.toml": "[tool.uv]\nmanaged = true",
			},
			expectedPM: PackageManagerUV,
		},
		{
			name:       "poetry lock file",
			files:      map[string]string{"poetry.lock": ""},
			expectedPM: PackageManagerPoetry,
		},
		{
			name:       "defaults to pip",
			files:      map[string]string{"requirements.txt": ""},
			expectedPM: PackageManagerPip,
		},
		{
			name:       "uv takes priority over poetry",
			files:      map[string]string{"uv.lock": "", "poetry.lock": ""},
			expectedPM: PackageManagerUV,
		},
		{
			name: "configured package manager overrides detection",
			files: map[string]string{
				"poetry.lock": "",
			},
			configuredPM: "pip",
			expectedPM:   PackageManagerPip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create files
			for filename, content := range tt.files {
				filePath := filepath.Join(testDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Test detection
			detector := NewDetector(tt.configuredPM)
			pm, err := detector.detectPythonPackageManager(testDir)
			if err != nil {
				t.Fatalf("detectPythonPackageManager() error = %v", err)
			}

			if pm != tt.expectedPM {
				t.Errorf("detectPythonPackageManager() = %v, want %v", pm, tt.expectedPM)
			}
		})
	}
}

//nolint:gocognit // Test function with multiple scenarios
func TestDetect(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		files           map[string]string
		expectedProject ProjectType
		expectedPM      PackageManager
	}{
		{
			name: "nodejs with npm",
			files: map[string]string{
				"package.json": "{}",
			},
			expectedProject: ProjectTypeNodeJS,
			expectedPM:      PackageManagerNPM,
		},
		{
			name: "nodejs with pnpm",
			files: map[string]string{
				"package.json":   "{}",
				"pnpm-lock.yaml": "",
			},
			expectedProject: ProjectTypeNodeJS,
			expectedPM:      PackageManagerPNPM,
		},
		{
			name: "go project",
			files: map[string]string{
				"go.mod": "",
			},
			expectedProject: ProjectTypeGo,
			expectedPM:      PackageManagerGoMod,
		},
		{
			name: "python with poetry",
			files: map[string]string{
				"pyproject.toml": "",
				"poetry.lock":    "",
			},
			expectedProject: ProjectTypePython,
			expectedPM:      PackageManagerPoetry,
		},
		{
			name: "ruby project",
			files: map[string]string{
				"Gemfile": "",
			},
			expectedProject: ProjectTypeRuby,
			expectedPM:      PackageManagerBundle,
		},
		{
			name: "rust project",
			files: map[string]string{
				"Cargo.toml": "",
			},
			expectedProject: ProjectTypeRust,
			expectedPM:      PackageManagerCargo,
		},
		{
			name:            "no project",
			files:           map[string]string{},
			expectedProject: ProjectTypeNone,
			expectedPM:      PackageManagerNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create files
			for filename, content := range tt.files {
				filePath := filepath.Join(testDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create file %s: %v", filename, err)
				}
			}

			// Test detection
			detector := NewDetector("")
			result, err := detector.Detect(testDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			if result.ProjectType != tt.expectedProject {
				t.Errorf("Detect() ProjectType = %v, want %v", result.ProjectType, tt.expectedProject)
			}

			if result.PackageManager != tt.expectedPM {
				t.Errorf("Detect() PackageManager = %v, want %v", result.PackageManager, tt.expectedPM)
			}

			if result.WorktreePath != testDir {
				t.Errorf("Detect() WorktreePath = %v, want %v", result.WorktreePath, testDir)
			}
		})
	}
}
