#!/usr/bin/env bash

# Exit on error
set -e

# Check if shellcheck is installed
if ! command -v shellcheck &>/dev/null; then
  echo "Error: shellcheck is not installed" >&2
  if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Try: brew install shellcheck" >&2
  else
    echo "Consult your OS package manager to install shellcheck" >&2
  fi
  exit 1
fi

# Get the repository root directory
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

# Start the timer
start_time=$(bash "$repo_root/scripts/utils/get_timestamp.sh")

# Determine number of parallel jobs
if command -v nproc &>/dev/null; then
  parallel_jobs=$(nproc)
elif [[ "$OSTYPE" == "darwin"* ]]; then
  parallel_jobs=$(sysctl -n hw.ncpu)
else
  parallel_jobs=4
fi

# Find shell scripts and validate in parallel
errors=$(git ls-files --cached --others --exclude-standard -z |
  grep -z '\.sh$' |
  xargs -0 -n 1 -P "$parallel_jobs" bash -c 'shellcheck "$0"' 2>&1) || true

# Calculate total elapsed time
end_time=$(bash "$repo_root/scripts/utils/get_timestamp.sh")
total_elapsed=$((end_time - start_time))

# Check and report errors
if [[ -n $errors ]]; then
  echo "ShellCheck found issues in the following files:" >&2
  echo "$errors" >&2
  echo "" >&2
  echo "Total time elapsed: ${total_elapsed}ms" >&2
  exit 1
fi

echo "All shell scripts passed ShellCheck validation."
echo "Total time elapsed: ${total_elapsed}ms"
