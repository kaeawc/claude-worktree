# Contributing to auto-worktree

Thank you for your interest in contributing to auto-worktree! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Code Quality](#code-quality)
- [Testing](#testing)
- [Building](#building)
- [Submitting Changes](#submitting-changes)

## Development Setup

### Prerequisites

**Required:**
- Go 1.25 or later ([download](https://go.dev/dl/))
- Git

**Recommended for local development:**
- [golangci-lint](https://golangci-lint.run/usage/install/) - for comprehensive linting
- [staticcheck](https://staticcheck.io/docs/getting-started/) - for static analysis

### Installation

#### Install golangci-lint

**macOS:**
```bash
brew install golangci-lint
```

**Linux:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Windows:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### Install staticcheck

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Clone the Repository

```bash
git clone https://github.com/kaeawc/auto-worktree.git
cd auto-worktree
```

### Download Dependencies

```bash
go mod download
# or
make deps
```

## Development Workflow

### Available Make Targets

The project includes a Makefile with convenient targets for common development tasks:

```bash
make help           # Show all available commands
make build          # Build the binary for current platform
make test           # Run tests
make test-verbose   # Run tests with verbose output
make coverage       # Run tests with coverage report
make lint           # Run golangci-lint
make fmt            # Format code with gofmt
make vet            # Run go vet
make staticcheck    # Run staticcheck
make install        # Install binary to GOPATH/bin
make clean          # Remove build artifacts
make deps           # Download dependencies
make tidy           # Tidy and verify dependencies
make verify         # Verify dependencies
make build-all      # Build for all platforms (linux, darwin, windows)
make ci             # Run all CI checks locally
```

### Typical Development Cycle

1. **Make your changes**
   ```bash
   # Edit code in your favorite editor
   ```

2. **Format code**
   ```bash
   make fmt
   ```

3. **Run linters and static analysis**
   ```bash
   make vet
   make lint
   make staticcheck
   ```

4. **Run tests**
   ```bash
   make test
   # or with coverage
   make coverage
   ```

5. **Build and test locally**
   ```bash
   make build
   ./build/auto-worktree version
   ```

6. **Run all CI checks before committing**
   ```bash
   make ci
   ```

## Code Quality

### Pre-commit Hooks (Optional)

The project provides optional pre-commit hooks that automatically validate your code before each commit. These hooks check:
- **Go files**: Code formatting (gofmt) and linting (golangci-lint)
- **Shell scripts**: ShellCheck validation

#### Installing Pre-commit Hooks

To install the pre-commit hooks:

```bash
# Install combined hooks (Go + ShellCheck)
scripts/go-hooks/install_pre_commit_hook.sh
```

The installer will:
- Detect if you already have ShellCheck hooks installed
- Create a combined hook that runs both validations
- Prompt you before replacing any existing hooks

#### Bypassing Hooks

To bypass the pre-commit hooks for a single commit (not recommended):

```bash
git commit --no-verify
```

#### Uninstalling Hooks

```bash
rm .git/hooks/pre-commit
```

### Code Style Guidelines

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (enforced by CI)
- Follow the guidelines from [Effective Go](https://go.dev/doc/effective_go)
- Keep functions focused and reasonably sized
- Add comments for exported functions and types
- Use meaningful variable and function names

### Linting

The project uses golangci-lint with a comprehensive set of linters configured in `.golangci.yml`. The configuration includes:

- **Error checking**: errcheck, gosec
- **Code quality**: gocritic, gocyclo, gocognit
- **Style**: stylecheck, revive, whitespace
- **Performance**: prealloc
- **Best practices**: bodyclose, exportloopref

Run the linter locally:

```bash
make lint
```

### Static Analysis

In addition to golangci-lint, we use staticcheck for additional static analysis:

```bash
make staticcheck
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make coverage
```

### Writing Tests

- Place test files next to the code they test with `_test.go` suffix
- Use table-driven tests where appropriate
- Aim for high test coverage of critical paths
- Use `t.Helper()` for test helper functions
- Use meaningful test names that describe what is being tested

Example:
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic case", "input", "output"},
        {"edge case", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Feature(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Building

### Build for Current Platform

```bash
make build
```

The binary will be in `build/auto-worktree`.

### Build for All Platforms

```bash
make build-all
```

This creates binaries for:
- Linux (amd64, arm64)
- macOS/Darwin (amd64, arm64)
- Windows (amd64)

Binaries will be in `build/{platform}/auto-worktree-{arch}`.

### Install Locally

```bash
make install
```

This installs the binary to `$GOPATH/bin/auto-worktree`.

## Submitting Changes

### Before Submitting a Pull Request

1. **Ensure all tests pass**
   ```bash
   make test
   ```

2. **Run all CI checks locally**
   ```bash
   make ci
   ```

3. **Format your code**
   ```bash
   make fmt
   ```

4. **Update documentation** if you changed user-facing behavior

5. **Add tests** for new functionality

### Pull Request Process

1. **Fork the repository** and create a new branch from `main`
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes** following the guidelines above

3. **Commit your changes** with clear, descriptive commit messages
   ```bash
   git commit -m "Add feature: description of feature"
   ```

4. **Push to your fork**
   ```bash
   git push origin feature/my-new-feature
   ```

5. **Open a Pull Request** on GitHub with:
   - Clear description of the changes
   - Reference to any related issues
   - Confirmation that tests pass locally

### Commit Message Guidelines

- Use the imperative mood ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Reference issues and PRs where appropriate
- Provide context in the body if needed

Example:
```
Add support for custom worktree locations

Allows users to specify a custom directory for worktrees via
the --worktree-dir flag or AUTO_WORKTREE_DIR environment variable.

Fixes #123
```

## CI/CD

The project uses GitHub Actions for continuous integration. On every pull request and push to `main`, the following checks run:

1. **Test Matrix**: Tests run on Linux, macOS, and Windows with Go 1.25 and 1.24
2. **Linting**: golangci-lint runs on Linux with Go 1.25
3. **Static Analysis**: staticcheck runs on all code
4. **Build Matrix**: Builds are created for all supported platforms and architectures

All checks must pass before a PR can be merged.

### Running CI Checks Locally

To run the same checks that CI runs:

```bash
make ci
```

This runs:
- Dependency verification
- `go vet`
- staticcheck
- golangci-lint
- All tests

## Questions?

If you have questions about contributing, feel free to:
- Open an issue for discussion
- Ask in your pull request
- Check existing documentation in the repository

Thank you for contributing to auto-worktree!
