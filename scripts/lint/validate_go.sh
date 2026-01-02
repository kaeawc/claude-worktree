#!/bin/bash
# Validate Go code using golangci-lint
# This script runs the same linting checks as CI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Version must match CI (.github/workflows/pull_request.yml)
GOLANGCI_LINT_VERSION="v2.7.2"

echo "Running Go lint validation..."

# Function to install golangci-lint
install_golangci_lint() {
    echo -e "${YELLOW}Installing golangci-lint ${GOLANGCI_LINT_VERSION}...${NC}"

    # Determine install directory
    if [ -n "$(go env GOPATH)" ]; then
        INSTALL_DIR="$(go env GOPATH)/bin"
    else
        INSTALL_DIR="${HOME}/go/bin"
    fi

    # Create directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Install using official installer
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
        sh -s -- -b "$INSTALL_DIR" "$GOLANGCI_LINT_VERSION"

    echo -e "${GREEN}✓ Installed golangci-lint ${GOLANGCI_LINT_VERSION}${NC}"
}

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}golangci-lint not found.${NC}"
    install_golangci_lint
else
    # Check version
    current_version=$(golangci-lint --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    expected_version="${GOLANGCI_LINT_VERSION#v}"

    if [ "$current_version" != "$expected_version" ]; then
        echo -e "${YELLOW}⚠️  Local version $current_version doesn't match CI version $GOLANGCI_LINT_VERSION${NC}"
        install_golangci_lint
    else
        echo -e "${GREEN}✓ golangci-lint version $current_version matches CI${NC}"
    fi
fi

# Run golangci-lint
echo "Running golangci-lint..."
if golangci-lint run --timeout=5m; then
    echo -e "${GREEN}✓ Linting passed!${NC}"
else
    echo -e "${RED}✗ Linting failed${NC}"
    exit 1
fi

# Run go vet
echo "Running go vet..."
if go vet ./...; then
    echo -e "${GREEN}✓ go vet passed!${NC}"
else
    echo -e "${RED}✗ go vet failed${NC}"
    exit 1
fi

# Run go fmt check
echo "Checking go fmt..."
if [ -n "$(gofmt -l .)" ]; then
    echo -e "${RED}✗ Code is not formatted${NC}"
    echo "Run: go fmt ./..."
    gofmt -l .
    exit 1
else
    echo -e "${GREEN}✓ Code is formatted!${NC}"
fi

echo -e "${GREEN}All lint checks passed!${NC}"
