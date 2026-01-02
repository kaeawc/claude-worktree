# Feature Parity: Bash (aw.sh) vs Go (auto-worktree) Implementation

## Overview

This document tracks feature parity between the original Bash implementation (`aw.sh`) and the Go rewrite (`auto-worktree`).

## Core Commands

### 1. Interactive Menu (`aw`)

**Bash Implementation:**
- Shows menu with all available commands
- Loops back after each operation
- Supports Escape to exit

**Go Implementation (RunInteractiveMenu):**
- ✅ Menu with all available commands
- ✅ Loops back after each operation
- ✅ Supports Escape/Ctrl-C to exit
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Menu display and item order
- [ ] Menu selection and routing
- [ ] Loop behavior after operations
- [ ] Escape/Ctrl-C exit behavior

---

### 2. New Worktree (`aw new`)

**Bash Implementation (_aw_new):**
- Interactive branch name entry (defaults to random name)
- Validates branch name (alphanumeric, dashes, underscores)
- Checks out new branch
- Creates worktree
- Runs post-create hooks
- Shows next steps

**Go Implementation (RunNew):**
- ✅ Interactive branch name entry
- ✅ Default random name generation
- ✅ Branch name validation
- ✅ Worktree creation
- ✅ Hook execution
- ✅ Next steps messaging
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Random name generation
- [ ] Branch name validation rules
- [ ] Special character handling in names
- [ ] Duplicate branch detection
- [ ] Worktree creation errors
- [ ] Hook execution and failures
- [ ] Terminal output and prompts

---

### 3. List Worktrees (`aw list`, `aw ls`)

**Bash Implementation (_aw_list):**
- Lists all existing worktrees
- Shows current worktree with indicator
- Shows detached HEAD states
- Formats output nicely
- Shows modification times

**Go Implementation (RunList):**
- ✅ Lists all worktrees
- ✅ Shows current worktree indicator
- ✅ Shows branch information
- ✅ Formatted output
- ✅ Shows timestamps
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] List output format
- [ ] Current worktree indicator
- [ ] Detached HEAD handling
- [ ] Empty repository (no worktrees)
- [ ] Sorting and ordering

---

### 4. Resume Worktree (`aw resume`)

**Bash Implementation (_aw_resume):**
- Shows last accessed worktree (via git config)
- Offers to resume it
- Can navigate to worktree directory
- Shows tmux session info if available

**Go Implementation (RunResume):**
- ✅ Fetches last worktree via session manager
- ✅ Filters out main branch
- ✅ Navigates to worktree or shows options
- ✅ Handles tmux sessions
- **Status:** COMPLETE (with tmux support)

**Testing Gaps:**
- [ ] Last worktree tracking
- [ ] Main branch filtering
- [ ] Directory navigation
- [ ] Tmux session detection
- [ ] No previous worktree fallback

---

### 5. Work on Issue (`aw issue [id]`)

**Bash Implementation (_aw_issue):**
- Supports multiple providers: GitHub, GitLab, JIRA, Linear
- Interactive issue selection with filtering
- Direct mode: `aw issue 123` or `aw issue PROJ-123`
- Auto-select mode with AI (if enabled)
- Detects closed/merged issues
- Generates branch names: `work/<number>-<title>`
- Creates worktree or resumes existing
- Shows available labels

**Go Implementation (RunIssue):**
- ✅ GitHub support (complete)
- ❌ GitLab support (not implemented)
- ❌ JIRA support (not implemented)
- ❌ Linear support (not implemented)
- ✅ Interactive mode with filtering
- ✅ Direct mode with issue ID/number
- ❌ AI auto-select (deferred)
- ✅ Closed issue detection
- ✅ Branch name generation
- ✅ Worktree creation/resume
- **Status:** PARTIAL (GitHub only)

**Testing Gaps:**
- [ ] GitHub issue listing
- [ ] GitLab issue listing (need implementation)
- [ ] JIRA issue listing (need implementation)
- [ ] Linear issue listing (need implementation)
- [ ] Interactive filtering
- [ ] Direct issue ID parsing
- [ ] Closed/merged detection
- [ ] Branch name generation rules
- [ ] Provider switching
- [ ] API authentication failures
- [ ] Rate limiting handling

---

### 6. Create Issue (`aw create`)

**Bash Implementation (_aw_create_issue):**
- Supports GitHub, GitLab, JIRA, Linear
- Interactive template-based creation
- Field-by-field completion
- AI-assisted content generation (if enabled)
- Shows created issue details
- Offers to work on created issue

**Go Implementation (RunCreate):**
- ❌ Not implemented
- **Status:** NOT IMPLEMENTED

**Testing Gaps:**
- [ ] All creation paths for all providers
- [ ] Template detection and usage
- [ ] Field validation
- [ ] AI content generation
- [ ] Success feedback

---

### 7. Review PR/MR (`aw pr [num]`)

**Bash Implementation (_aw_pr):**
- Supports GitHub and GitLab
- Interactive PR/MR list with filtering
- Direct mode: `aw pr 123`
- Shows PR/MR details and status
- Detects merged PRs
- Creates worktree for review
- Shows reviewer checklist

**Go Implementation (RunPR):**
- ❌ Not fully implemented
- ✅ GitHub PR support (basic, needs testing)
- ❌ GitLab MR support (not implemented)
- ❌ AI auto-select (deferred)
- ❌ Reviewer checklist (not implemented)
- **Status:** PARTIAL (GitHub needs testing)

**Testing Gaps:**
- [ ] GitHub PR listing
- [ ] GitLab MR listing (need implementation)
- [ ] Interactive filtering
- [ ] Direct PR number parsing
- [ ] Merged PR detection
- [ ] Worktree creation for reviews
- [ ] Reviewer workflow

---

### 8. Cleanup Worktrees (`aw cleanup`)

**Bash Implementation (_aw_cleanup_interactive):**
- Interactive selection of worktrees to delete
- Shows worktree status (merged, stale, etc.)
- Confirms deletion
- Deletes branch and worktree
- Cleans up git references
- Prunes orphaned worktrees

**Go Implementation (RunCleanup):**
- ✅ Interactive worktree selection
- ✅ Shows worktree info
- ✅ Confirmation before deletion
- ✅ Worktree and branch deletion
- ✅ Git prune operation
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Worktree list display
- [ ] Merged branch detection
- [ ] Stale worktree detection
- [ ] Deletion confirmation
- [ ] Branch deletion
- [ ] Worktree deletion
- [ ] Git prune execution

---

### 9. Settings Menu (`aw settings`)

**Bash Implementation (_aw_settings_menu):**
- Configure issue provider (GitHub, GitLab, JIRA, Linear)
- Configure provider-specific settings (server URL, project key, etc.)
- Set AI tool preference (claude, codex, gemini, etc.)
- Enable/disable autoselect
- Configure issue templates
- View current settings
- Reset all settings

**Go Implementation (RunSettings):**
- ✅ Issue provider selection
- ⚠️ Partial provider configuration (GitHub only)
- ❌ AI tool configuration (deferred)
- ❌ Issue template configuration (not implemented)
- ✅ Settings display
- ✅ Settings reset
- **Status:** PARTIAL

**Testing Gaps:**
- [ ] Provider selection menu
- [ ] Provider-specific settings (server, project, team)
- [ ] Settings persistence
- [ ] Settings display/verification
- [ ] Settings reset

---

### 10. Remove Worktree (`aw remove`, `aw rm`)

**Bash Implementation:**
- Parses worktree reference from CLI
- Confirms deletion
- Deletes worktree and branch
- Similar to cleanup but single item

**Go Implementation (RunRemove):**
- ✅ Implemented in main.go routing
- **Status:** COMPLETE (needs testing)

**Testing Gaps:**
- [ ] CLI argument parsing
- [ ] Confirmation dialog
- [ ] Worktree deletion
- [ ] Branch deletion

---

### 11. Prune Worktrees (`aw prune`)

**Bash Implementation (_aw_prune_worktrees):**
- Silently removes orphaned worktrees
- Called on startup
- Cleans git worktree metadata

**Go Implementation (RunPrune):**
- ✅ Executes `git worktree prune`
- ✅ Called on startup in main.go
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Orphaned worktree detection
- [ ] Silent execution
- [ ] Git prune command

---

## Supporting Features

### A. Configuration Management

**Bash:**
- Uses git config (per-repository)
- Keys: issue-provider, jira-server, jira-project, gitlab-server, gitlab-project, linear-team, ai-tool, autoselect, pr-autoselect
- View/set via `git config`

**Go:**
- ✅ Git config support implemented
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Config read operations
- [ ] Config write operations
- [ ] Config key validation
- [ ] Fallback to defaults

---

### B. Git Hooks

**Bash:**
- post-checkout hook execution
- Custom hook support
- Configurable (enable/disable)
- Failure handling (continue or fail)

**Go:**
- ✅ Hook execution implemented
- ✅ Custom hook support
- ✅ Configurable behavior
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Hook discovery
- [ ] Hook execution
- [ ] Hook error handling
- [ ] Custom hook support

---

### C. Branch Name Validation & Generation

**Bash (_aw_sanitize_branch_name):**
- Lowercase all characters
- Replace spaces and special chars with dashes
- Remove leading/trailing dashes
- Max 40 characters for issue titles
- Alphanumeric, dashes, underscores allowed

**Go (git/names.go):**
- ✅ Same validation rules
- ✅ Max 40 character limit for issue titles
- **Status:** COMPLETE

**Testing Gaps:**
- [ ] Special character handling
- [ ] Unicode character handling
- [ ] Length limits
- [ ] Leading/trailing dash removal
- [ ] Consecutive dash collapsing

---

### D. Provider Integration

#### GitHub
**Bash:**
- Fetches issues via `gh` CLI
- Creates issues via `gh` CLI
- Supports issue filtering/searching
- Creates branches for issues
- Detects merged PRs

**Go:**
- ✅ Issue fetching via `gh` CLI
- ❌ Issue creation (deferred)
- ✅ Interactive filtering (UI component)
- ✅ Branch creation
- ❌ PR review workflow (deferred)
- **Status:** PARTIAL (Issue workflow complete)

**Testing Gaps:**
- [ ] `gh` CLI availability check
- [ ] Authentication verification
- [ ] Issue listing from API
- [ ] Pagination handling
- [ ] Error cases (not found, private repo, etc.)
- [ ] Closed/merged issue detection

#### GitLab
**Bash:**
- Fetches issues/MRs via GitLab API
- Configurable server URL
- Project selection
- Issue creation support

**Go:**
- ❌ Not implemented
- **Status:** NOT IMPLEMENTED

#### JIRA
**Bash:**
- Fetches issues via JIRA API
- Configurable server URL
- Project key selection
- Issue type support

**Go:**
- ❌ Not implemented
- **Status:** NOT IMPLEMENTED

#### Linear
**Bash:**
- Fetches issues via Linear API
- Team selection
- Issue cycle support

**Go:**
- ❌ Not implemented
- **Status:** NOT IMPLEMENTED

---

### E. Tmux Session Management

**Bash:**
- Basic tmux awareness (if available)
- Shows tmux session info in resume

**Go:**
- ✅ Full tmux session creation
- ✅ Session tracking and management
- ✅ Automatic session creation for new worktrees
- ✅ Session info in resume
- **Status:** ENHANCED (better than bash)

**Testing Gaps:**
- [ ] Tmux availability detection
- [ ] Session creation
- [ ] Session naming
- [ ] Session tracking in metadata
- [ ] Session resumption

---

## Feature Matrix Summary

| Feature | Bash | Go | Status | Priority |
|---------|------|----|---------|-----------|
| Interactive Menu | ✅ | ✅ | Complete | HIGH |
| New Worktree | ✅ | ✅ | Complete | HIGH |
| List Worktrees | ✅ | ✅ | Complete | HIGH |
| Resume Worktree | ✅ | ✅ | Complete | MEDIUM |
| Work on Issue (GitHub) | ✅ | ✅ | Complete | HIGH |
| Work on Issue (GitLab) | ✅ | ❌ | Missing | MEDIUM |
| Work on Issue (JIRA) | ✅ | ❌ | Missing | MEDIUM |
| Work on Issue (Linear) | ✅ | ❌ | Missing | MEDIUM |
| Create Issue | ✅ | ❌ | Missing | MEDIUM |
| Review PR/MR | ✅ | ⚠️ | Partial | MEDIUM |
| Cleanup Worktrees | ✅ | ✅ | Complete | MEDIUM |
| Settings Menu | ✅ | ⚠️ | Partial | MEDIUM |
| Remove Worktree | ✅ | ✅ | Complete | LOW |
| Prune Worktrees | ✅ | ✅ | Complete | LOW |
| Config Management | ✅ | ✅ | Complete | MEDIUM |
| Git Hooks | ✅ | ✅ | Complete | MEDIUM |
| Branch Name Validation | ✅ | ✅ | Complete | HIGH |
| AI Auto-select | ✅ | ❌ | Deferred | LOW |
| Tmux Integration | ✅ | ✅ | Enhanced | MEDIUM |

---

## Implementation Completeness by Category

### Fully Complete (Ready for Parity Testing)
- Interactive Menu
- New Worktree
- List Worktrees
- Resume Worktree
- Cleanup Worktrees
- Remove Worktree
- Prune Worktrees
- Config Management
- Git Hooks
- Branch Name Validation

### Partially Complete (Need Provider Implementation)
- Work on Issue (GitHub done, GitLab/JIRA/Linear missing)
- Review PR/MR (GitHub implementation exists, needs testing)
- Settings (basic done, provider config needs work)

### Not Implemented (Deferred)
- Create Issue
- AI Auto-select
- Full PR/MR review workflow

---

## Testing Strategy

### Phase 1: Unit Tests (In Progress)
- Existing test files cover most packages
- Need to add: command integration tests, provider stubs

### Phase 2: Integration Tests (To Do)
- Test each command with real git repositories
- Test provider interactions with stubs/mocks
- Test configuration persistence

### Phase 3: E2E Tests (To Do)
- Full workflow tests (new → work on issue → cleanup)
- Multi-command sequences
- Interactive menu flows

### Phase 4: Provider Tests (To Do)
- Provider interface testing
- Provider stub implementations
- API failure handling

### Phase 5: Edge Cases (To Do)
- Special character handling
- Large branch name edge cases
- Missing dependencies
- Network failures
- Rate limiting

### Phase 6: Cross-Platform (To Do)
- macOS-specific paths and behaviors
- Linux-specific paths and behaviors
- Windows support (if applicable)

### Phase 7: Performance (To Do)
- Benchmarks for critical operations
- Memory usage profiling
- API call optimization

---

## Success Criteria

For issue #88 to be complete:
1. All unit tests pass
2. All integration tests pass
3. Feature matrix shows ✅ or ⚠️ (no ❌ for completed features)
4. Cross-platform compatibility verified
5. No performance regressions vs. Bash version
6. User workflows feel familiar (tested via UAT)

