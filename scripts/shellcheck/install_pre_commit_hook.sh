#!/usr/bin/env bash
#
# Install the shellcheck pre-commit hook
# This script creates a symlink from .git/hooks/pre-commit to scripts/shellcheck/pre-commit
# Works with both regular repositories and git worktrees
#

set -e

# Get the repository root
repo_root=$(git rev-parse --show-toplevel)

# Get the git common directory (handles both regular repos and worktrees)
git_common_dir=$(git rev-parse --git-common-dir)

# Define paths
hook_source="$repo_root/scripts/shellcheck/pre-commit"
hook_target="$git_common_dir/hooks/pre-commit"

# Create hooks directory if it doesn't exist
mkdir -p "$git_common_dir/hooks"

# Check if hook source exists
if [[ ! -f "$hook_source" ]]; then
  echo "Error: Hook source not found at $hook_source" >&2
  exit 1
fi

# Check if a pre-commit hook already exists
if [[ -f "$hook_target" ]] || [[ -L "$hook_target" ]]; then
  echo "A pre-commit hook already exists at $hook_target"
  read -p "Do you want to replace it? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Installation cancelled."
    exit 0
  fi
  rm -f "$hook_target"
fi

# Create the symlink (use absolute path to handle worktrees)
ln -s "$hook_source" "$hook_target"

echo "Pre-commit hook installed successfully!"
echo "The hook will run shellcheck on staged .sh files before each commit."
echo ""
echo "To bypass the hook for a single commit (not recommended):"
echo "  git commit --no-verify"
echo ""
echo "To uninstall the hook:"
echo "  rm $hook_target"
