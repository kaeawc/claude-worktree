package environment

import (
	"testing"
)

func TestGetCommandName(t *testing.T) {
	installer := NewInstaller(nil)

	tests := []struct {
		pm          PackageManager
		expectedCmd string
	}{
		{PackageManagerNPM, "npm"},
		{PackageManagerYarn, "yarn"},
		{PackageManagerPNPM, "pnpm"},
		{PackageManagerBun, "bun"},
		{PackageManagerUV, "uv"},
		{PackageManagerPoetry, "poetry"},
		{PackageManagerPip, "pip"},
		{PackageManagerBundle, "bundle"},
		{PackageManagerGoMod, "go"},
		{PackageManagerCargo, "cargo"},
		{PackageManagerNone, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.pm), func(t *testing.T) {
			cmd := installer.getCommandName(tt.pm)
			if cmd != tt.expectedCmd {
				t.Errorf("getCommandName(%v) = %v, want %v", tt.pm, cmd, tt.expectedCmd)
			}
		})
	}
}

func TestGetInstallCommand(t *testing.T) {
	installer := NewInstaller(nil)

	tests := []struct {
		pm           PackageManager
		expectedCmd  string
		expectedArgs []string
	}{
		{
			pm:           PackageManagerNPM,
			expectedCmd:  "npm",
			expectedArgs: []string{"install", "--silent"},
		},
		{
			pm:           PackageManagerYarn,
			expectedCmd:  "yarn",
			expectedArgs: []string{"install", "--silent"},
		},
		{
			pm:           PackageManagerPNPM,
			expectedCmd:  "pnpm",
			expectedArgs: []string{"install", "--silent"},
		},
		{
			pm:           PackageManagerBun,
			expectedCmd:  "bun",
			expectedArgs: []string{"install", "--silent"},
		},
		{
			pm:           PackageManagerUV,
			expectedCmd:  "uv",
			expectedArgs: []string{"sync"},
		},
		{
			pm:           PackageManagerPoetry,
			expectedCmd:  "poetry",
			expectedArgs: []string{"install", "--quiet"},
		},
		{
			pm:           PackageManagerPip,
			expectedCmd:  "pip",
			expectedArgs: []string{"install", "-r", "requirements.txt", "--quiet"},
		},
		{
			pm:           PackageManagerBundle,
			expectedCmd:  "bundle",
			expectedArgs: []string{"install", "--quiet"},
		},
		{
			pm:           PackageManagerGoMod,
			expectedCmd:  "go",
			expectedArgs: []string{"mod", "download"},
		},
		{
			pm:           PackageManagerCargo,
			expectedCmd:  "cargo",
			expectedArgs: []string{"fetch", "--quiet"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.pm), func(t *testing.T) {
			cmd, args := installer.getInstallCommand(tt.pm)
			if cmd != tt.expectedCmd {
				t.Errorf("getInstallCommand(%v) cmd = %v, want %v", tt.pm, cmd, tt.expectedCmd)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("getInstallCommand(%v) args length = %v, want %v", tt.pm, len(args), len(tt.expectedArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("getInstallCommand(%v) args[%d] = %v, want %v", tt.pm, i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

func TestInstallWithNoPackageManager(t *testing.T) {
	installer := NewInstaller(nil)

	result := installer.Install(&DetectionResult{
		ProjectType:    ProjectTypeNone,
		PackageManager: PackageManagerNone,
		WorktreePath:   "/test/path",
	})

	if !result.Success {
		t.Errorf("Install() with no package manager should succeed, got Success = %v", result.Success)
	}

	if result.Message != "No package manager detected, skipping installation" {
		t.Errorf("Install() message = %v, want 'No package manager detected, skipping installation'", result.Message)
	}
}

func TestInstallProgressCallback(t *testing.T) {
	var progressMessages []string
	installer := NewInstaller(func(message string) {
		progressMessages = append(progressMessages, message)
	})

	// Test with unavailable package manager (won't actually install)
	result := installer.Install(&DetectionResult{
		ProjectType:    ProjectTypeNodeJS,
		PackageManager: "nonexistent-pm",
		WorktreePath:   "/test/path",
	})

	if result.Success {
		t.Errorf("Install() with unknown package manager should fail")
	}

	// Progress callback shouldn't be called for unknown package manager
	// because we fail before the installation attempt
	if len(progressMessages) > 0 {
		t.Errorf("Install() should not call progress callback for unknown package manager")
	}
}
