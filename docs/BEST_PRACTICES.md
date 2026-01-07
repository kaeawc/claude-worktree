# Best Practices for Safe Worktree Usage

This guide explains how to safely use git worktrees, especially when working with multiple worktrees, AI agents, or concurrent development workflows.

## Table of Contents

- [Understanding Git's Single-Process Design](#understanding-gits-single-process-design)
- [What's Shared vs. Isolated](#whats-shared-vs-isolated)
- [Safe vs. Unsafe Patterns](#safe-vs-unsafe-patterns)
- [Common Corruption Scenarios](#common-corruption-scenarios)
- [Safe Parallel Development Workflows](#safe-parallel-development-workflows)
- [Working with AI Agents](#working-with-ai-agents)
- [Recovery from Corruption](#recovery-from-corruption)

## Understanding Git's Single-Process Design

**Critical Fact**: Git is designed as a **single-process tool**. It was never intended to handle multiple concurrent operations accessing the same repository data simultaneously.

### Why This Matters

Git worktrees are an amazing feature that allow you to:
- Check out multiple branches simultaneously
- Work on different features without context switching
- Keep your main branch clean while experimenting

However, **worktrees do NOT make git operations thread-safe**. They only prevent branch-level conflicts (ensuring a branch is checked out in only one place).

### The Core Problem

When you run multiple git commands at the same time across different worktrees, they all access the same shared repository data. Without proper locking and coordination, this can lead to:

- **Corrupted object database** - Broken commits, missing objects
- **Invalid references** - Branches pointing to non-existent commits
- **Index corruption** - Staging area in inconsistent state
- **Lost commits** - Work disappearing or becoming unreachable
- **Repository requiring fsck repair** - Structural damage needing manual intervention

## What's Shared vs. Isolated

Understanding what's shared helps you recognize when concurrent operations are dangerous.

### Shared Across ALL Worktrees

These are stored in the main repository's `.git/` directory and accessed by all worktrees:

```
.git/
├── objects/          # Object database (ALL commits, trees, blobs)
├── refs/             # Branch and tag references
│   ├── heads/        # All branch pointers
│   └── tags/         # All tags
├── config            # Repository configuration
├── hooks/            # Git hooks
├── packed-refs       # Packed reference data
└── info/
    ├── refs          # Reference metadata
    └── exclude       # Ignore patterns
```

**Any git command that modifies these shared areas is dangerous when run concurrently.**

### Isolated Per Worktree

These are unique to each worktree:

```
.git/worktrees/<name>/
├── HEAD              # Current branch pointer (what branch is checked out)
├── index             # Staging area (what you've git added)
├── logs/             # Reflog for this worktree only
│   └── HEAD
├── ORIG_HEAD         # Previous HEAD value
├── FETCH_HEAD        # Last fetched commit
└── MERGE_HEAD        # Merge state
```

**Operations that only touch these isolated areas are safer, but most git commands also touch shared data.**

### What This Means

- **Safe**: Having different files checked out in each worktree
- **Safe**: Reading git history (`git log`, `git show`) in multiple worktrees
- **UNSAFE**: Running `git commit` in two worktrees simultaneously
- **UNSAFE**: Running `git rebase` while another worktree is committing
- **UNSAFE**: Running `git gc` while any worktree is active

## Safe vs. Unsafe Patterns

### Unsafe Patterns

These patterns can corrupt your repository:

#### 1. Parallel Git Operations

```bash
# UNSAFE: Background processes running git commands
cd ~/worktrees/repo/feature-1 && git commit -m "change" &
cd ~/worktrees/repo/feature-2 && git commit -m "change" &
wait

# UNSAFE: Shell job control with git
(cd worktree1 && git rebase main) &
(cd worktree2 && git push origin feature) &

# UNSAFE: Parallel script execution
parallel -j 3 'cd {} && git commit -m "auto commit"' ::: worktree1 worktree2 worktree3
```

#### 2. Multiple AI Agents

```bash
# UNSAFE: Multiple AI agents running simultaneously
# Terminal 1: Claude Code in worktree1
claude --dangerously-skip-permissions

# Terminal 2: Claude Code in worktree2 (CONCURRENT!)
claude --dangerously-skip-permissions

# Both agents may run git commands (commit, push, status) simultaneously
```

#### 3. Background Tools + Manual Operations

```bash
# UNSAFE: IDE with background git status + manual git rebase
# Your IDE is polling: `git status` every 2 seconds
# You run: `git rebase -i main`
# Potential conflict: rebase is rewriting history while status reads it

# UNSAFE: Pre-commit hook running while another worktree commits
# Worktree 1: Running pre-commit hook (may modify .git/)
# Worktree 2: You run git commit (also modifying .git/)
```

#### 4. Automated Workflows

```bash
# UNSAFE: CI/CD or automated scripts in parallel
# Cron job: git fetch && git merge (every 5 minutes)
# Your work: git rebase main (manual operation)
# Collision: Both touching refs simultaneously
```

### Safe Patterns

These patterns prevent corruption:

#### 1. Sequential Operations

```bash
# SAFE: Run one at a time
cd ~/worktrees/repo/feature-1
git commit -m "change"

cd ~/worktrees/repo/feature-2
git commit -m "change"

cd ~/worktrees/repo/feature-3
git rebase main
```

#### 2. Single Active Agent

```bash
# SAFE: Only one agent running at a time
cd ~/worktrees/repo/feature-1
claude --dangerously-skip-permissions
# Complete work, exit agent

cd ~/worktrees/repo/feature-2
claude --dangerously-skip-permissions
# Start next agent only after first is done
```

#### 3. Coordinated Operations

```bash
# SAFE: Pause background processes before git operations
# 1. Stop your IDE's git monitoring
# 2. Run your git rebase
# 3. Restart IDE monitoring

# SAFE: Use file locks or coordination
if [ ! -f /tmp/repo.lock ]; then
  touch /tmp/repo.lock
  git commit -m "safe commit"
  rm /tmp/repo.lock
fi
```

#### 4. Read-Only Operations

```bash
# SAFE: Multiple worktrees can READ simultaneously
cd worktree1 && git log --oneline &
cd worktree2 && git show HEAD &
cd worktree3 && git diff main &
wait

# Reading doesn't modify shared state
```

## Common Corruption Scenarios

### Scenario 1: Dual-Agent Development

**Setup:**
- Worktree 1: Claude Code working on issue #42
- Worktree 2: Claude Code working on issue #43

**What Happens:**
1. Agent 1 runs `git commit -m "Fix login"`
2. Agent 2 runs `git commit -m "Update styles"` (simultaneously)
3. Both write to `.git/objects/` at the same time
4. Object file corruption: One commit may have invalid data
5. Result: `git fsck` shows "bad object" errors

**How to Avoid:**
- Only run one agent per repository at a time
- Use `auto-worktree sessions` to track active agents
- Pause one agent before starting another

### Scenario 2: Rebase During Background Status

**Setup:**
- IDE running `git status` every 2 seconds in background
- You run `git rebase -i main` to clean up history

**What Happens:**
1. Rebase starts rewriting commits
2. Background status reads `.git/refs/heads/feature`
3. Rebase updates the reference mid-read
4. Status sees inconsistent state
5. Result: Reference corruption or "bad ref" errors

**How to Avoid:**
- Disable background git polling before rebase
- Use `git config extensions.worktreeConfig true` for isolation
- Close IDE during complex git operations

### Scenario 3: Concurrent Commits with Hooks

**Setup:**
- Worktree 1: Committing with pre-commit hook (runs linting, modifies files)
- Worktree 2: Simultaneously committing different changes

**What Happens:**
1. Both commits start writing to `.git/objects/`
2. Both pre-commit hooks may update `.git/config` or `.git/hooks/`
3. Race condition: One commit's objects may reference the other's tree
4. Result: Invalid commit graph, broken history

**How to Avoid:**
- Never commit simultaneously across worktrees
- If hooks are slow, ensure they're run sequentially
- Consider using local hooks (`.git/hooks/`) that don't modify shared state

### Scenario 4: Git GC During Active Work

**Setup:**
- Worktree 1: You're making commits
- Background: `git gc --auto` triggers (automatic garbage collection)

**What Happens:**
1. GC starts compacting `.git/objects/`
2. Your commit writes new objects
3. GC moves/deletes objects while you're writing
4. Result: "Object not found" errors, broken commits

**How to Avoid:**
- Disable auto-gc when using multiple worktrees: `git config gc.auto 0`
- Run `git gc` manually during idle time
- Never run GC while any worktree is active

## Safe Parallel Development Workflows

### Workflow 1: Time-Sliced Multi-Agent Development

Use a single agent at a time, switching between worktrees:

```bash
# Morning: Work on feature-1
cd ~/worktrees/repo/feature-1
aw issue 42
# Claude Code session, make progress, commit, exit

# Afternoon: Switch to feature-2
cd ~/worktrees/repo/feature-2
aw issue 43
# New Claude Code session, different work

# Evening: Review PR
cd ~/worktrees/repo/pr-123
aw pr 123
# Review changes, test, comment
```

**Key Principle**: Only one git-modifying process at a time.

### Workflow 2: Read-Only Review Workflow

Have multiple worktrees open for reading, but only modify in one:

```bash
# Worktree 1: Active development (only place you commit)
cd ~/worktrees/repo/feature-1
# Make changes, commit, push

# Worktree 2: Reference (read-only, no commits)
cd ~/worktrees/repo/main
# Browse code, check examples, NO git commands that modify

# Worktree 3: Testing (read-only)
cd ~/worktrees/repo/staging
# Run tests, verify behavior, NO commits
```

**Key Principle**: Only one worktree is "active" for git modifications.

### Workflow 3: Feature + Main Branch Pattern

Keep main branch always available for reference:

```bash
# Main worktree: main branch (read-only, pristine)
~/worktrees/repo/main/
# Never commit here, only pull

# Feature worktrees: Active development
~/worktrees/repo/feature-1/  # Issue #42
~/worktrees/repo/feature-2/  # Issue #43
~/worktrees/repo/pr-review/  # PR #100

# Work on ONE feature at a time, reference main as needed
```

### Workflow 4: Sequential Batch Operations

Process multiple worktrees one at a time:

```bash
# Update all worktrees sequentially
for worktree in ~/worktrees/repo/*/; do
  cd "$worktree"
  git fetch origin
  git merge --ff-only origin/main
  # Each operation completes before next starts
done

# NEVER parallelize this loop with & or parallel
```

## Working with AI Agents

AI agents (Claude Code, Cursor, GitHub Copilot Workspace) frequently run git commands in the background. Special care is needed.

### Safe AI Agent Practices

1. **One Agent Per Repository**
   ```bash
   # SAFE: Single agent session
   cd ~/worktrees/repo/feature-1
   claude --dangerously-skip-permissions

   # Wait for agent to exit before starting another
   ```

2. **Pause Before Manual Git Operations**
   ```bash
   # In Claude Code session:
   # 1. Type /pause (if available) or Ctrl+C to pause
   # 2. Run your manual git rebase
   # 3. Resume agent or restart session
   ```

3. **Disable Background Git in IDEs**
   ```bash
   # VS Code settings:
   "git.autorefresh": false
   "git.autofetch": false

   # IntelliJ IDEA: Disable VCS background operations
   # Settings → Version Control → Git → Uncheck "Auto-update"
   ```

4. **Use Status Command to Check Activity**
   ```bash
   # Before starting second agent, check if any are running
   aw sessions  # Shows active tmux sessions with agents

   # Only start new agent if no others are active
   ```

### Agent Coordination Strategies

#### Strategy 1: Session-Based Locking

```bash
# Create a lock file when starting an agent
start_agent() {
  local repo=$(basename $(git rev-parse --show-toplevel))
  local lockfile="/tmp/git-agent-${repo}.lock"

  if [ -f "$lockfile" ]; then
    echo "ERROR: Agent already running for $repo"
    echo "Existing: $(cat $lockfile)"
    return 1
  fi

  echo "$(date): $(pwd)" > "$lockfile"
  claude --dangerously-skip-permissions
  rm "$lockfile"
}
```

#### Strategy 2: Tmux Session Management

```bash
# Use auto-worktree's built-in session tracking
aw sessions  # View all active sessions

# Pause session before starting another in same repo
aw sessions  # Select session to pause

# Resume when ready
aw sessions  # Select session to resume
```

#### Strategy 3: Manual Coordination

```bash
# Simple terminal-based coordination
# Terminal 1: Set prompt to show "AGENT ACTIVE"
PS1="[AGENT] $PS1"
claude

# Terminal 2: Check other terminals before starting agent
# Only start if no other terminals show "AGENT ACTIVE"
```

## Recovery from Corruption

If you experience git corruption, here's how to recover:

### Step 1: Identify the Problem

```bash
# Check repository integrity
git fsck --full

# Common errors:
# - "missing blob" - Object database corruption
# - "dangling commit" - Unreachable commits
# - "bad ref" - Reference corruption
```

### Step 2: Assess the Damage

```bash
# Check which branches are affected
git branch -v

# Check if you can access recent commits
git log --oneline -10

# Verify working directory is clean
git status
```

### Step 3: Basic Recovery

```bash
# Try automatic repair first
git gc --prune=now
git fsck --full

# If successful, repository is fixed
```

### Step 4: Advanced Recovery

If basic recovery fails:

```bash
# 1. Backup current state
cp -r .git .git.backup

# 2. Try to recover from reflog
git reflog  # Find lost commits
git cherry-pick <lost-commit-hash>

# 3. Re-fetch from remote (if pushed)
git fetch origin
git reset --hard origin/main

# 4. Restore from backup (if you have one)
# Restore from Time Machine or backup service
```

### Step 5: Prevention for Next Time

```bash
# Enable more aggressive safety checks
git config core.fsyncObjectFiles true
git config transfer.fsckObjects true
git config fetch.fsckObjects true
git config receive.fsckObjects true

# Disable auto-gc
git config gc.auto 0

# Create regular backups
# Add to cron or use backup service
```

### When to Give Up and Re-Clone

If corruption is severe and you've pushed recent work:

```bash
# 1. Note any unpushed changes
git log origin/main..HEAD  # Commits not pushed

# 2. Re-clone from remote
cd ..
mv repo repo.corrupted
git clone <remote-url> repo

# 3. Re-create worktrees
cd repo
auto-worktree issue 42
# Manually re-apply any unpushed changes
```

## Related Issues

These GitHub issues are related to concurrent operation safety:

- [#174](https://github.com/user/auto-worktree/issues/174) - Git corruption from concurrent access
- [#168](https://github.com/user/auto-worktree/issues/168) - Worktree directories disappear
- [#170](https://github.com/user/auto-worktree/issues/170) - Terminal crashes
- [#175](https://github.com/user/auto-worktree/issues/175) - Claude Code lock file conflicts
- [#176](https://github.com/user/auto-worktree/issues/176) - Documentation: This document

## Additional Resources

- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree)
- [Pro Git Book - Git Internals](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain)
- [AGENTS.md](../AGENTS.md#safety-coordinating-multiple-ai-agents) - AI agent coordination guide
