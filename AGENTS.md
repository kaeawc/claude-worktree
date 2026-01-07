# Agent Guide for auto-worktree

This document provides essential context for AI agents working on the auto-worktree project.

## Safety: Coordinating Multiple AI Agents

**WARNING**: While worktrees safely isolate branches, git itself is NOT designed for concurrent operations. Running multiple AI agents simultaneously in different worktrees can corrupt your repository.

### Critical Safety Rules

1. **Run Only ONE AI Agent Per Repository at a Time**
   - Multiple agents = multiple concurrent git commands
   - Concurrent git commands = repository corruption risk
   - Even if agents are in different worktrees

2. **Pause Other Agents Before Git Operations**
   - Before rebasing, committing, or pushing in one worktree
   - Stop or pause any AI agents in other worktrees
   - Wait for the operation to complete

3. **Disable Background Git Operations**
   - IDE git status polling can interfere with agent operations
   - Disable auto-fetch and auto-refresh in VS Code, IntelliJ, etc.
   - See [docs/BEST_PRACTICES.md](docs/BEST_PRACTICES.md) for detailed instructions

### Why This Matters

Git worktrees share critical data:
- `.git/objects/` - Object database (all commits)
- `.git/refs/` - Branch references
- `.git/config` - Repository configuration
- Pack files and garbage collection

When multiple agents run git commands simultaneously, they can corrupt this shared data.

### Safe AI Agent Workflow

```bash
# Start first agent in worktree 1
cd ~/worktrees/repo/feature-1
auto-worktree issue 42
# Claude Code session starts
# Make changes, commit, push
# EXIT agent when done

# Only after first agent exits, start second agent
cd ~/worktrees/repo/feature-2
auto-worktree issue 43
# New Claude Code session
```

### Unsafe AI Agent Workflow

```bash
# Starting multiple agents simultaneously - DANGEROUS!
# Terminal 1:
cd ~/worktrees/repo/feature-1
claude --dangerously-skip-permissions &

# Terminal 2 (DON'T DO THIS):
cd ~/worktrees/repo/feature-2
claude --dangerously-skip-permissions &

# Both agents may run git commands simultaneously
# Result: Potential repository corruption
```

### Using auto-worktree Session Management

The `auto-worktree sessions` command helps coordinate agents:

```bash
# Check for active sessions before starting new agent
auto-worktree sessions

# Shows:
# 🟢 Running: feature-1 (claude)
# 💤 Idle: feature-2 (paused)

# Only start new agent if no others are RUNNING
# Paused or idle sessions are safe
```

### Tool-Specific Configuration

#### Claude Code

```bash
# Claude Code can run background git status
# Disable with flag or ensure only one instance runs
claude --dangerously-skip-permissions  # Use cautiously

# Check for running instances before starting
ps aux | grep "claude"
```

#### VS Code + AI Extensions

```json
// settings.json
{
  "git.autorefresh": false,
  "git.autofetch": false,
  "git.enabled": true
}
```

#### Cursor

```bash
# Cursor runs git commands via its AI agent
# Ensure no other Cursor windows are open for same repo
# Close other windows before starting new agent session
```

### Recovery from Corruption

If you experience repository corruption:

1. **Identify the problem:**
   ```bash
   git fsck --full
   ```

2. **Basic recovery:**
   ```bash
   git gc --prune=now
   git fsck --full
   ```

3. **Advanced recovery:**
   See [docs/BEST_PRACTICES.md - Recovery from Corruption](docs/BEST_PRACTICES.md#recovery-from-corruption)

4. **Prevention:**
   - Always follow the "one agent at a time" rule
   - Use `auto-worktree sessions` to track active agents
   - Disable background git operations in all tools

### Related Documentation

- [README.md - Safety Warning](README.md#safety-warning-concurrent-git-operations) - Quick overview
- [docs/BEST_PRACTICES.md](docs/BEST_PRACTICES.md) - Comprehensive safety guide
- [Issue #174](https://github.com/user/auto-worktree/issues/174) - Git corruption reports
- [Issue #176](https://github.com/user/auto-worktree/issues/176) - This documentation effort

## Project Overview

**auto-worktree** is a bash/zsh tool that enables safe, isolated workspaces for AI agent sessions using git worktrees. It provides an interactive TUI for creating worktrees, working on GitHub issues, reviewing PRs, and managing cleanup of merged/stale worktrees.

- **Type**: Bash/Zsh shell utility
- **Primary File**: `aw.sh` (single-file implementation)
- **Dependencies**: `gum`, `gh`, `jq` (all installable via Homebrew)
- **Target Shell**: zsh (with zsh completion support)
- **License**: MIT

## Repository Structure

```
.
├── LICENSE              # MIT license
├── README.md            # User documentation
├── aw.sh                # Main shell script (source from ~/.zshrc)
├── demo.gif             # Animated demo for README
└── demo/
    ├── demo-script.sh   # Simulated demo script
    ├── demo.cast        # asciinema recording
    └── record-demo.sh   # Records and converts demo to GIF
```

## Core Architecture

### Main Script: `aw.sh`

The entire tool is implemented as a **single bash script** with these components:

1. **Dependency Management** (`_aw_check_deps`)
   - Validates `gum`, `gh`, `jq` availability
   - Shows installation commands if missing

2. **Word Lists** (`_WORKTREE_WORDS`, `_WORKTREE_COLORS`)
   - Arrays used for generating random branch names
   - Pattern: `{color}-{word1}-{word2}` (e.g., `coral-apex-beam`)

3. **Helper Functions** (prefixed with `_aw_`)
   - Repository info gathering
   - Branch name sanitization
   - Issue/PR merge detection
   - Worktree creation and cleanup

4. **Core Commands**
   - `auto-worktree new` - Create new worktree with random or custom branch
   - `auto-worktree issue [num]` - Work on GitHub issue
   - `auto-worktree pr [num]` - Review GitHub PR
   - `auto-worktree list` - List/manage existing worktrees
   - `auto-worktree help` - Show help

5. **Zsh Completion** (`_auto_worktree`)
   - Auto-completes commands, issue numbers, PR numbers
   - Fetches live data from GitHub via `gh` CLI

### Key Design Patterns

#### Function Naming Convention
- **Public function**: `auto-worktree` (main entry point)
- **Private helpers**: `_aw_*` prefix (not meant for direct invocation)
- **Pattern**: All internal functions use `_aw_` namespace to avoid conflicts

#### Error Handling
- Functions return `1` on error, `0` on success
- Use `gum style --foreground 1` for error messages
- Early returns with `|| return 1` pattern

#### User Experience
- Interactive prompts powered by `gum` (choose, confirm, input, spin, style)
- Color-coded output:
  - Red (1): Errors, stale worktrees (>4 days)
  - Green (2): Success, recent worktrees (<1 day)
  - Yellow (3): Warnings, worktrees 1-4 days old
  - Blue (4): Info boxes
  - Magenta (5): Merged indicators
  - Cyan (6): Highlights

#### Worktree Lifecycle
1. **Creation**: `git worktree add -b <branch> <path> <base>`
2. **Storage**: `~/worktrees/<repo-name>/<worktree-name>/`
3. **Launch**: `claude --dangerously-skip-permissions` in worktree directory
4. **Cleanup**: `git worktree remove --force <path>` + optional branch deletion

## Important Implementation Details

### Branch Name Sanitization
```bash
_aw_sanitize_branch_name() {
  echo "$1" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$//'
}
```
- Converts to lowercase
- Replaces non-alphanumeric with dashes
- Collapses multiple dashes
- Strips leading/trailing dashes

### Issue/PR Merge Detection

**Issue merge detection** (`_aw_check_issue_merged`):
- Checks if issue state is `CLOSED`
- Checks if `stateReason` is `COMPLETED` (indicates PR was merged)
- Searches for merged PRs with "closes #N" / "fixes #N" / "resolves #N"

**PR merge detection** (`_aw_check_branch_pr_merged`):
- Uses `gh pr view <branch>` to check if PR exists and is merged
- Returns 0 if `state` is `MERGED`

### Worktree Age Calculation
- Uses commit timestamp from `git log -1 --format=%ct`
- Falls back to file modification time if no commits (via `_aw_get_file_mtime` helper)
- Cross-platform support: macOS/BSD (`stat -f %m`) and Linux (`stat -c %Y`)
- Age thresholds:
  - `<1 day`: Green, shows hours (e.g., `[2h ago]`)
  - `1-4 days`: Yellow, shows days (e.g., `[3d ago]`)
  - `>4 days`: Red (stale), shows days (e.g., `[7d ago]`)

### Cleanup Priority
1. **Merged worktrees** are prompted for cleanup first (higher priority)
2. **Stale worktrees** (>4 days) are prompted only if no merged ones exist
3. Only prompts for **one worktree at a time** for user control

## Common Development Tasks

### Testing Changes to `aw.sh`

Since this is a shell utility loaded in the user's shell:

```bash
# Reload after changes
source aw.sh

# Test in a git repository
cd /path/to/any/git/repo
auto-worktree list
auto-worktree new
```

**Important**: Always test in an actual git repository, not in the auto-worktree directory itself.

### Testing GitHub Integration

Requires:
- GitHub repository context
- `gh` CLI authenticated (`gh auth login`)
- Open issues/PRs to test with

```bash
# Test issue workflow
auto-worktree issue

# Test PR workflow  
auto-worktree pr

# Test with specific number
auto-worktree issue 42
auto-worktree pr 123
```

### Recording Demo

```bash
cd demo/
./record-demo.sh
```

Requirements:
- `asciinema` (install: `brew install asciinema`)
- `agg` (install: `brew install agg`)

Creates `demo.gif` in project root using `demo-script.sh` simulation.

## Code Style and Conventions

### Shell Style
- **Indentation**: 2 spaces (not tabs)
- **Line length**: Generally <100 chars, but not strict
- **Quoting**: Always quote variables: `"$variable"` not `$variable`
- **Arrays**: Use zsh array syntax `${#array[@]}`, `${array[$i]}`

### Variable Naming
- **Global state**: `_AW_UPPERCASE` (e.g., `_AW_GIT_ROOT`, `_AW_WORKTREE_BASE`)
- **Local vars**: `snake_case` (e.g., `branch_name`, `worktree_path`)
- **Function params**: Positional `$1`, `$2` or named locals

### Error Messages
```bash
# Error pattern
gum style --foreground 1 "Error: <message>"
return 1

# Warning pattern
gum style --foreground 3 "<message>"

# Success pattern
gum style --foreground 2 "<message>"
```

### gum UI Patterns

**Spinner for async operations**:
```bash
gum spin --spinner dot --title "Loading..." -- command args
```

**Bordered info box**:
```bash
gum style --border rounded --padding "0 1" --border-foreground 4 \
  "Title" \
  "Line 1" \
  "Line 2"
```

**User confirmation**:
```bash
if gum confirm "Proceed?"; then
  # user said yes
fi
```

**Interactive selection**:
```bash
choice=$(echo "$options" | gum filter --placeholder "Select...")
```

**User input**:
```bash
value=$(gum input --placeholder "Enter value")
value=$(gum input --value "default" --header "Confirm:")
```

## Known Quirks and Gotchas

### 1. Zsh Array Iteration Bug (Fixed)
**Historical issue** (fixed in commit `034fadc`):
- Arrays in zsh are 1-indexed, not 0-indexed
- Iteration must use `while [[ $i -le ${#array[@]} ]]` pattern
- Access with `${array[$i]}` not `${array[i]}`

### 2. Variable Assignment with Echo (Fixed)
**Historical issue** (fixed in commit `991ad67`):
- Cannot use `echo` for variable assignment in zsh in some contexts
- Use direct assignment or command substitution instead

### 3. `claude` Command Dependency
- Hardcoded call to `claude --dangerously-skip-permissions`
- Assumes Claude Code CLI is installed and in PATH
- No fallback if `claude` is not available

### 4. `gum`, `gh`, `jq` Required
- Tool **will not function** without these dependencies
- Dependency check happens at runtime, not install-time
- User must install via Homebrew on macOS

### 5. Worktree Paths
- Worktrees stored in `~/worktrees/<repo-name>/`
- **Not configurable** (hardcoded in `_aw_get_repo_info`)
- Multiple repos with same basename will share directory (potential conflict)

### 6. Merge Detection Rate Limits
- Calls `gh issue view` and `gh pr view` for each worktree in `list` command
- Can hit GitHub API rate limits with many worktrees
- No caching mechanism

### 7. Detached HEAD for PRs
- PR checkout uses `--detach` flag (commits on FETCH_HEAD)
- Worktree created from FETCH_HEAD, not a named branch
- User must create branch manually if they want to push changes

## GitHub Integration

### Required Setup
```bash
# Install and authenticate
brew install gh
gh auth login
```

### API Usage Patterns

**Fetch issues**:
```bash
gh issue list --limit 20 --state open --json number,title,labels
```

**Fetch PRs**:
```bash
gh pr list --limit 20 --state open --json number,title,author,headRefName,baseRefName
```

**Check issue state**:
```bash
gh issue view $num --json state,stateReason
```

**Check PR state**:
```bash
gh pr view $num --json state,mergedAt
```

**Checkout PR**:
```bash
gh pr checkout $num --detach
```

### JSON Processing with jq
All GitHub CLI output uses `--json` with `--jq` for extraction:
```bash
# Extract single field
title=$(echo "$data" | jq -r '.title')

# Template format for lists
--template '{{range .}}#{{.number}} | {{.title}}{{"\n"}}{{end}}'
```

## Testing Strategy

Since this is a shell utility without automated tests:

### Manual Testing Checklist

**Basic functionality**:
- [ ] `auto-worktree` shows interactive menu
- [ ] `auto-worktree new` creates worktree with random name
- [ ] `auto-worktree new` accepts custom branch name
- [ ] `auto-worktree list` shows existing worktrees
- [ ] `auto-worktree help` shows usage

**GitHub integration** (requires repo with issues/PRs):
- [ ] `auto-worktree issue` lists open issues
- [ ] `auto-worktree issue <num>` creates worktree for issue
- [ ] `auto-worktree pr` lists open PRs
- [ ] `auto-worktree pr <num>` creates worktree for PR review
- [ ] Merged issue/PR shows `[merged #N]` indicator
- [ ] Cleanup prompt appears for merged worktrees

**Edge cases**:
- [ ] Creating worktree for existing branch
- [ ] Branch name with special characters gets sanitized
- [ ] Missing dependencies show helpful error
- [ ] Running outside git repo shows error
- [ ] Repository with no worktrees shows appropriate message

**Zsh completion**:
- [ ] `auto-worktree <TAB>` shows commands
- [ ] `auto-worktree issue <TAB>` shows issue numbers
- [ ] `auto-worktree pr <TAB>` shows PR numbers

## Future Enhancement Ideas

Based on code review, potential improvements:

1. **Configurable worktree base path** - Allow user to set custom directory
2. **Dependency fallbacks** - Graceful degradation without `gh` (no GitHub features)
3. **Merge detection caching** - Cache GitHub API calls to avoid rate limits
4. **Named branches for PRs** - Option to create local branch instead of detached HEAD
5. **Batch cleanup** - Option to clean up multiple merged/stale worktrees at once
6. **Status indicators** - Show git status (dirty/clean) in `list` output
7. **Shell detection** - Support bash in addition to zsh
8. **Configuration file** - `.awrc` for user preferences

## Commands Reference

### No Commands to Memorize
This project has **no build, test, or lint commands** - it's a pure shell script utility.

### Development Workflow
1. Edit `aw.sh`
2. `source aw.sh` to reload in current shell
3. Test in a git repository: `auto-worktree <command>`
4. Commit changes when satisfied

### Installation
Add to `~/.zshrc`:
```bash
source /path/to/auto-worktree/aw.sh
```

## When Contributing

### Before Making Changes
- Read through `aw.sh` to understand the flow
- Test in a real git repository (not this one)
- Consider impact on existing worktrees
- Preserve backward compatibility with function signatures

### Code Changes
- Maintain `_aw_` prefix for internal functions
- Use `gum` for all UI interactions
- Follow existing color scheme (error=red, success=green, etc.)
- Quote all variable references
- Use `local` for function-scoped variables

### Documentation
- Update README.md for user-facing changes
- Update this AGENTS.md for implementation details
- Update function header comments in `aw.sh` if behavior changes

### Testing
- Test with and without GitHub integration
- Test dependency checking (temporarily rename a dependency)
- Test edge cases (special chars, long names, existing branches)
- Verify zsh completion still works

## Summary for Quick Start

**What this tool does**: Creates isolated git worktrees for AI agent sessions, with GitHub issue/PR integration.

**Key file**: `aw.sh` (single-file implementation)

**Dependencies**: `gum`, `gh`, `jq` (install via Homebrew)

**Testing**: Source the file, run `auto-worktree` commands in a git repo

**Common pitfall**: Must be in a git repository to use any command

**Critical pattern**: All internal functions use `_aw_` prefix; only `auto-worktree` is public

**Integration point**: Calls `claude --dangerously-skip-permissions` to launch Claude Code
