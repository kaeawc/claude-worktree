# auto-worktree

A bash tool for safely running AI agents in isolated git worktrees. Create separate workspaces for each task, issue, or PR review - keeping your main branch pristine.

![Demo](demo.gif)

## Features

- **Isolated Workspaces**: Each task gets its own worktree - no branch conflicts or stashed changes
- **Issue Tracking Integration**: Work on GitHub issues or JIRA tickets with automatic branch naming
- **GitHub PR Reviews**: Review pull requests in isolated worktrees
- **Interactive TUI**: Beautiful menus powered by [gum](https://github.com/charmbracelet/gum)
- **Auto-cleanup**: Detects merged PRs, closed issues, and resolved JIRA tickets
- **Random Names**: Generates memorable branch names like `work/coral-apex-beam`
- **Tab Completion**: Full zsh completion for commands, issues, and PRs
- **AI Agent Support**: Integrates with Claude Code, Codex CLI, or Gemini CLI

## Installation

### Prerequisites

#### Required

```bash
brew install gum jq
```

- **gum** - Terminal UI components
- **jq** - JSON processor

#### Optional (based on what you use)

**For GitHub:**
```bash
brew install gh
gh auth login
```

**For JIRA:**
```bash
brew install ankitpokhrel/jira-cli/jira-cli
jira init
```

**For AI agents (choose one):**
- **Claude Code**: `brew install claude` or `npm install -g @anthropic-ai/claude-code`
- **Codex CLI**: `npm install -g @openai/codex-cli`
- **Gemini CLI**: See [gemini-cli](https://github.com/google/generative-ai-cli)

### Setup

Add to your `~/.zshrc`:

```bash
source /path/to/auto-worktree/aw.sh
```

## Usage

```bash
auto-worktree                  # Interactive menu
auto-worktree new              # Create new worktree
auto-worktree issue [id]       # Work on an issue (GitHub #123 or JIRA PROJ-123)
auto-worktree pr [num]         # Review a GitHub PR
auto-worktree list             # List existing worktrees
auto-worktree help             # Show help
```

### Create a New Worktree

```bash
auto-worktree new
```

Enter a branch name or leave blank for a random name like `work/mint-code-flux`.

### Work on Issues

The first time you run `auto-worktree issue`, you'll be prompted to choose between GitHub or JIRA for this repository. This preference is stored in git config.

**GitHub Issues:**
```bash
auto-worktree issue        # Select from open issues
auto-worktree issue 42     # Work on issue #42 directly
```

Creates a branch like `work/42-fix-login-bug` and launches your AI agent.

**JIRA Issues:**
```bash
auto-worktree issue             # Select from open JIRA issues
auto-worktree issue PROJ-123    # Work on JIRA-123 directly
```

Creates a branch like `work/PROJ-123-implement-feature` and launches your AI agent.

### Review a Pull Request

```bash
auto-worktree pr           # Select from open PRs
auto-worktree pr 123       # Review PR #123 directly
```

Checks out the PR in a new worktree and shows the diff stats.

### List Worktrees

```bash
auto-worktree list
```

Shows all worktrees with:
- Age indicators (green: recent, yellow: few days, red: stale)
- Merged PR/issue detection (GitHub and JIRA)
- Cleanup prompts for merged, resolved, or stale worktrees

## Configuration

Issue provider settings are stored per-repository using git config:

```bash
# View current configuration
git config --get auto-worktree.issue-provider   # github or jira

# Manual configuration
git config auto-worktree.issue-provider jira
git config auto-worktree.jira-server https://your-company.atlassian.net
git config auto-worktree.jira-project PROJ      # Optional: default project filter
```

Different repositories can use different issue providers.

## How It Works

1. **Worktrees** are stored in `~/worktrees/<repo-name>/`
2. Each worktree is a full copy of your repo on its own branch
3. Claude Code launches with `--dangerously-skip-permissions` for uninterrupted work
4. When done, use `list` to clean up merged worktrees and branches

## Example Workflows

### GitHub Workflow

```bash
# Start work on a GitHub issue
cd my-project
auto-worktree issue 42

# AI agent opens in ~/worktrees/my-project/work-42-add-feature/
# Make changes, commit, push, create PR

# Later, check for cleanup
auto-worktree list
# Shows "[merged #42]" indicator, prompts to clean up
```

### JIRA Workflow

```bash
# First time setup
cd my-work-project
auto-worktree issue
# Choose "JIRA" from the menu
# Enter JIRA server URL and project key

# Start work on a JIRA ticket
auto-worktree issue PROJ-456

# AI agent opens in ~/worktrees/my-work-project/work-PROJ-456-add-auth/
# Make changes, commit, push

# Later, when JIRA ticket is marked as Done
auto-worktree list
# Shows "[resolved PROJ-456]" indicator, prompts to clean up
```

## Tab Completion

The tool includes full zsh completion:

```bash
auto-worktree <TAB>        # Shows: new, issue, pr, list, help
auto-worktree issue <TAB>  # Shows open issues from GitHub
auto-worktree pr <TAB>     # Shows open PRs from GitHub
```

## Why Worktrees?

- **No context switching**: Keep multiple tasks in progress without stashing
- **Clean isolation**: Claude Code changes won't affect other branches
- **Easy cleanup**: Delete the folder and branch when done
- **Parallel work**: Run multiple Claude Code sessions on different tasks

## License

MIT
