#!/usr/bin/env bash
#
# Validate all Go code in the repository
# This script runs gofmt, go vet, and golangci-lint on all Go files
#
# Usage: scripts/go-hooks/validate_go_code.sh
#

set -e

# Get the repository root
repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

echo "Validating Go code in $repo_root"
echo "========================================"
echo ""

# Check if Go is installed
if ! command -v go &>/dev/null; then
  echo "Error: Go is not installed" >&2
  echo "Please install Go from https://go.dev/dl/" >&2
  exit 1
fi

echo "Go version:"
go version
echo ""

# Find all Go files
go_files=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" || true)

if [[ -z "$go_files" ]]; then
  echo "No Go files found in the repository"
  exit 0
fi

# Run gofmt check
echo "1. Checking code formatting with gofmt..."
unformatted=$(gofmt -l . | grep -v vendor || true)
if [[ -n "$unformatted" ]]; then
  echo "Error: The following files are not formatted with gofmt:" >&2
  echo "$unformatted" >&2
  echo "" >&2
  echo "Run 'make fmt' or 'gofmt -w .' to fix formatting" >&2
  exit 1
fi
echo "✓ All files are properly formatted"
echo ""

# Run go vet
echo "2. Running go vet..."
if ! go vet ./...; then
  echo "" >&2
  echo "Error: go vet found issues" >&2
  exit 1
fi
echo "✓ go vet passed"
echo ""

# Run golangci-lint if available
if command -v golangci-lint &>/dev/null; then
  echo "3. Running golangci-lint..."
  if ! golangci-lint run --timeout=5m; then
    echo "" >&2
    echo "Error: golangci-lint found issues" >&2
    exit 1
  fi
  echo "✓ golangci-lint passed"
  echo ""
else
  echo "3. Skipping golangci-lint (not installed)"
  echo "   To install: brew install golangci-lint (macOS)"
  echo "   Or see: https://golangci-lint.run/usage/install/"
  echo ""
fi

# Run staticcheck if available
if command -v staticcheck &>/dev/null; then
  echo "4. Running staticcheck..."
  if ! staticcheck ./...; then
    echo "" >&2
    echo "Error: staticcheck found issues" >&2
    exit 1
  fi
  echo "✓ staticcheck passed"
  echo ""
else
  echo "4. Skipping staticcheck (not installed)"
  echo "   To install: go install honnef.co/go/tools/cmd/staticcheck@latest"
  echo ""
fi

# Run tests
echo "5. Running tests..."
if ! go test -v -race ./...; then
  echo "" >&2
  echo "Error: tests failed" >&2
  exit 1
fi
echo "✓ All tests passed"
echo ""

echo "========================================"
echo "✓ All validations passed!"
