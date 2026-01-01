#!/usr/bin/env bash
#
# Install the Go pre-commit hook
# This script creates a symlink from .git/hooks/pre-commit to scripts/go-hooks/pre-commit
# Works with both regular repositories and git worktrees
#
# If a shellcheck hook is already installed, this will combine both hooks
#

set -e

# Get the repository root
repo_root=$(git rev-parse --show-toplevel)

# Get the git common directory (handles both regular repos and worktrees)
git_common_dir=$(git rev-parse --git-common-dir)

# Define paths
go_hook_source="$repo_root/scripts/go-hooks/pre-commit"
shellcheck_hook_source="$repo_root/scripts/shellcheck/pre-commit"
hook_target="$git_common_dir/hooks/pre-commit"

# Create hooks directory if it doesn't exist
mkdir -p "$git_common_dir/hooks"

# Check if hook source exists
if [[ ! -f "$go_hook_source" ]]; then
  echo "Error: Go hook source not found at $go_hook_source" >&2
  exit 1
fi

# Make the hook executable
chmod +x "$go_hook_source"

# Check if a pre-commit hook already exists
if [[ -f "$hook_target" ]] || [[ -L "$hook_target" ]]; then
  # Check if it's pointing to the shellcheck hook
  if [[ -L "$hook_target" ]]; then
    existing_target=$(readlink "$hook_target")
    if [[ "$existing_target" == *"shellcheck/pre-commit"* ]]; then
      echo "ShellCheck pre-commit hook detected."
      echo "Creating a combined hook for both ShellCheck and Go validation..."

      # Create a combined hook
      cat > "$hook_target" <<'EOF'
#!/usr/bin/env bash
#
# Combined pre-commit hook for ShellCheck and Go validation
# This hook runs both shellcheck and Go validation tools
#

# Get the repository root
repo_root=$(git rev-parse --show-toplevel)

# Run ShellCheck hook
if [[ -f "$repo_root/scripts/shellcheck/pre-commit" ]]; then
  bash "$repo_root/scripts/shellcheck/pre-commit"
  shellcheck_result=$?
else
  shellcheck_result=0
fi

# Run Go hook
if [[ -f "$repo_root/scripts/go-hooks/pre-commit" ]]; then
  bash "$repo_root/scripts/go-hooks/pre-commit"
  go_result=$?
else
  go_result=0
fi

# Exit with error if either hook failed
if [[ $shellcheck_result -ne 0 ]] || [[ $go_result -ne 0 ]]; then
  exit 1
fi

exit 0
EOF
      chmod +x "$hook_target"
      echo "Combined pre-commit hook installed successfully!"
      echo "The hook will run ShellCheck on .sh files and Go validation on .go files before each commit."
      exit 0
    fi
  fi

  echo "A pre-commit hook already exists at $hook_target"
  read -p "Do you want to replace it? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Installation cancelled."
    exit 0
  fi
  rm -f "$hook_target"
fi

# Check if shellcheck hook exists but isn't installed
if [[ -f "$shellcheck_hook_source" ]]; then
  echo "Both ShellCheck and Go hooks available."
  echo "Creating a combined hook for both..."

  cat > "$hook_target" <<'EOF'
#!/usr/bin/env bash
#
# Combined pre-commit hook for ShellCheck and Go validation
# This hook runs both shellcheck and Go validation tools
#

# Get the repository root
repo_root=$(git rev-parse --show-toplevel)

# Run ShellCheck hook
if [[ -f "$repo_root/scripts/shellcheck/pre-commit" ]]; then
  bash "$repo_root/scripts/shellcheck/pre-commit"
  shellcheck_result=$?
else
  shellcheck_result=0
fi

# Run Go hook
if [[ -f "$repo_root/scripts/go-hooks/pre-commit" ]]; then
  bash "$repo_root/scripts/go-hooks/pre-commit"
  go_result=$?
else
  go_result=0
fi

# Exit with error if either hook failed
if [[ $shellcheck_result -ne 0 ]] || [[ $go_result -ne 0 ]]; then
  exit 1
fi

exit 0
EOF
  chmod +x "$hook_target"
  echo "Combined pre-commit hook installed successfully!"
  echo "The hook will run ShellCheck on .sh files and Go validation on .go files before each commit."
else
  # Only install Go hook
  ln -s "$go_hook_source" "$hook_target"
  echo "Go pre-commit hook installed successfully!"
  echo "The hook will run Go validation on staged .go files before each commit."
fi

echo ""
echo "To bypass the hook for a single commit (not recommended):"
echo "  git commit --no-verify"
echo ""
echo "To uninstall the hook:"
echo "  rm $hook_target"
