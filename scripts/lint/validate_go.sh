#!/bin/bash
# Validate Go code using golangci-lint
# This script runs the same linting checks as CI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Running Go lint validation..."

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}golangci-lint not found. Installing...${NC}"
    if command -v brew &> /dev/null; then
        brew install golangci-lint
    else
        echo -e "${RED}Please install golangci-lint manually:${NC}"
        echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
        exit 1
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
