# User Acceptance Testing Guide - Issue #88

This guide provides a structured approach to test the Go version of auto-worktree against the Bash version to ensure feature parity and familiarity.

## Pre-Test Setup

### Requirements
- [ ] Go 1.22+ installed
- [ ] Git 2.35+
- [ ] GitHub CLI (`gh`) installed and authenticated (for GitHub tests)
- [ ] macOS, Linux, or Windows system
- [ ] Tmux installed (optional, for tmux session tests)

### Build Instructions
```bash
# Clone the repository
git clone https://github.com/kaeawc/auto-worktree.git
cd auto-worktree

# Build the Go version
go build -o auto-worktree ./cmd/auto-worktree

# Make it executable
chmod +x auto-worktree

# Verify version
./auto-worktree version
```

### Test Repository Setup
Create a test repository for UAT:

```bash
# Create a test repo
mkdir ~/test-aw-repo
cd ~/test-aw-repo
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create initial commit
echo "Test project" > README.md
git add README.md
git commit -m "Initial commit"

# Create main branch if needed
git checkout -b main
```

## Test Scenarios

### 1. Interactive Menu

**Test Case 1.1: Menu Display**
- [ ] Run `./auto-worktree` (no arguments)
- [ ] Verify menu is displayed
- [ ] Verify all options are visible:
  - [ ] New Worktree
  - [ ] Resume Worktree
  - [ ] Work on Issue
  - [ ] Create Issue
  - [ ] Review PR
  - [ ] List Worktrees
  - [ ] View Tmux Sessions
  - [ ] Cleanup Worktrees
  - [ ] Settings
- [ ] Menu looks clean and readable
- [ ] Navigation works (arrow keys)

**Test Case 1.2: Menu Navigation**
- [ ] Navigate up/down through menu items
- [ ] Verify selection highlights correctly
- [ ] Press Enter on each item
- [ ] Verify correct command is executed
- [ ] After command, menu loops back
- [ ] Can press Escape/Ctrl-C to exit menu

**Test Case 1.3: Menu Exit Behavior**
- [ ] In menu, press Escape
- [ ] Verify menu exits cleanly
- [ ] Run `./auto-worktree` again
- [ ] Verify menu appears again (not cached state)

---

### 2. New Worktree Creation

**Test Case 2.1: Basic Creation**
- [ ] Run `./auto-worktree new`
- [ ] Prompted for branch name
- [ ] Enter a simple name: `feature/test-feature`
- [ ] Verify worktree is created
- [ ] Verify directory exists: `.git/worktrees/feature/test-feature`
- [ ] Verify branch exists: `git branch | grep feature/test-feature`
- [ ] Current working directory is in new worktree
- [ ] Git status shows clean repository

**Test Case 2.2: Default Random Name**
- [ ] Run `./auto-worktree new`
- [ ] When prompted for name, just press Enter (use default)
- [ ] Verify random name is generated
- [ ] Verify name format follows pattern (e.g., `work-<random>`)
- [ ] Verify worktree created with random name
- [ ] Verify branch created with same name

**Test Case 2.3: Branch Name Validation**
- [ ] Run `./auto-worktree new`
- [ ] Try invalid names:
  - [ ] Name with spaces: `my feature` (should fail or convert)
  - [ ] Name with special chars: `feature!@#$` (should fail or convert)
  - [ ] Very long name (>100 chars) (should be truncated)
- [ ] Verify error handling or sanitization
- [ ] Can retry with valid name

**Test Case 2.4: Duplicate Branch**
- [ ] Create a worktree: `feature/dup`
- [ ] Try to create another with same name
- [ ] Verify error is shown
- [ ] Can create with different name

**Test Case 2.5: Hook Execution**
- [ ] Configure a post-checkout hook:
  ```bash
  cat > .git/hooks/post-checkout << 'EOF'
  #!/bin/bash
  echo "Hook executed"
  EOF
  chmod +x .git/hooks/post-checkout
  ```
- [ ] Create new worktree
- [ ] Verify hook message is displayed
- [ ] Verify hook was actually run (check if file was modified, etc.)

---

### 3. List Worktrees

**Test Case 3.1: List Display**
- [ ] Create 3 worktrees
- [ ] Run `./auto-worktree list`
- [ ] Verify all 3 are listed
- [ ] Verify columns are clean and aligned
- [ ] Main/initial worktree is shown first or marked as "main"

**Test Case 3.2: Current Worktree Indicator**
- [ ] Navigate to different worktrees
- [ ] Run `./auto-worktree list` from each
- [ ] Verify current worktree is marked with indicator (e.g., `*` or `→`)
- [ ] Other worktrees don't have indicator

**Test Case 3.3: Worktree Information**
- [ ] List should show for each worktree:
  - [ ] Path
  - [ ] Branch name
  - [ ] Last access time (if available)
  - [ ] Status (merged/stale/active)

**Test Case 3.4: Empty Repository**
- [ ] In a fresh repo with no worktrees
- [ ] Run `./auto-worktree list`
- [ ] Verify it shows cleanly (not crash)
- [ ] Shows "no worktrees" or minimal output

---

### 4. Resume Worktree

**Test Case 4.1: Resume Last Worktree**
- [ ] Create 2 worktrees: `feature1`, `feature2`
- [ ] Work in `feature1`, then navigate back to main
- [ ] Run `./auto-worktree resume`
- [ ] Verify it offers to resume `feature1` (last accessed)
- [ ] Accept resume
- [ ] Verify you're in `feature1` worktree

**Test Case 4.2: No Previous Worktree**
- [ ] In main repo with no previous resume history
- [ ] Run `./auto-worktree resume`
- [ ] Verify appropriate message (no previous worktree)
- [ ] Can create new or select from list

**Test Case 4.3: Main Branch Filtering**
- [ ] Resume should never offer main branch
- [ ] Only offer actual worktrees

**Test Case 4.4: Tmux Session Resume**
- [ ] If tmux is installed:
  - [ ] Create worktree with auto-worktree
  - [ ] Verify tmux session is created
  - [ ] Resume worktree
  - [ ] Verify tmux session is restored/shown

---

### 5. Work on GitHub Issue

**Prerequisite Setup:**
- [ ] Have a GitHub repository with test issues
- [ ] `gh` CLI is authenticated: `gh auth status`
- [ ] Configure repo: `git remote add origin https://github.com/YOUR_USER/YOUR_REPO.git`

**Test Case 5.1: Interactive Issue Selection**
- [ ] Run `./auto-worktree issue`
- [ ] Verify list of open issues is displayed
- [ ] Type to filter by number or title
- [ ] Select an issue
- [ ] Verify worktree is created with branch name: `work/<number>-<title>`

**Test Case 5.2: Direct Issue Number**
- [ ] Run `./auto-worktree issue 1` (if you have issue #1)
- [ ] Verify it skips to that issue directly
- [ ] Worktree is created for that issue

**Test Case 5.3: Issue Title in Branch Name**
- [ ] Create worktree for issue with title: "Fix authentication bug"
- [ ] Verify branch name is: `work/123-fix-authentication-bug` (or similar)
- [ ] Verify title is sanitized:
  - [ ] Lowercase
  - [ ] Spaces → dashes
  - [ ] Special chars removed
  - [ ] Max 40 characters

**Test Case 5.4: Closed Issue Warning**
- [ ] Close an issue on GitHub
- [ ] Run `./auto-worktree issue <number>`
- [ ] Verify warning is shown
- [ ] Can still create worktree if desired

**Test Case 5.5: Existing Worktree Resume**
- [ ] Create worktree for issue #5
- [ ] Work in it, then go back to main
- [ ] Try to work on issue #5 again
- [ ] Verify offer to resume existing worktree
- [ ] Accept resume
- [ ] Verify you're back in that worktree

---

### 6. Cleanup Worktrees

**Test Case 6.1: Interactive Selection**
- [ ] Create multiple worktrees with different states
- [ ] Run `./auto-worktree cleanup`
- [ ] Verify all worktrees are listed
- [ ] Select one to delete
- [ ] Verify confirmation dialog
- [ ] Verify worktree and branch are deleted

**Test Case 6.2: Merged Branch Detection**
- [ ] Create worktree for a feature
- [ ] Switch to main, merge the branch
- [ ] Run `./auto-worktree cleanup`
- [ ] Verify merged worktrees are marked/highlighted
- [ ] Clean up merged ones
- [ ] Verify branch is deleted

**Test Case 6.3: Stale Worktree**
- [ ] Create a worktree that's not accessed for a while
- [ ] Run `./auto-worktree cleanup`
- [ ] Verify old/stale worktrees are identified
- [ ] Can choose to clean them up

**Test Case 6.4: Cancellation**
- [ ] Run `./auto-worktree cleanup`
- [ ] Select worktrees for deletion
- [ ] When prompted for confirmation, choose "No"
- [ ] Verify worktrees are NOT deleted

---

### 7. Settings

**Test Case 7.1: Provider Selection**
- [ ] Run `./auto-worktree settings`
- [ ] Select "Issue Provider"
- [ ] Choose GitHub
- [ ] Verify setting is saved
- [ ] Exit settings
- [ ] Run `./auto-worktree issue`
- [ ] Verify it uses GitHub provider

**Test Case 7.2: View Settings**
- [ ] Run `./auto-worktree settings`
- [ ] View current settings
- [ ] Verify all configured values are shown

**Test Case 7.3: Reset Settings**
- [ ] Configure some settings
- [ ] Run `./auto-worktree settings`
- [ ] Choose "Reset"
- [ ] Confirm
- [ ] Verify all settings are cleared
- [ ] View settings - should show defaults

**Test Case 7.4: Provider-Specific Configuration**
- [ ] For GitLab:
  - [ ] Set server URL
  - [ ] Set project path
  - [ ] Verify settings saved
- [ ] For JIRA:
  - [ ] Set server URL
  - [ ] Set project key
  - [ ] Verify settings saved

---

### 8. Remove Worktree

**Test Case 8.1: Remove by Name**
- [ ] Create worktree: `feature/test`
- [ ] Run `./auto-worktree remove feature/test` (or similar syntax)
- [ ] Verify confirmation prompt
- [ ] Accept
- [ ] Verify worktree and branch are deleted

**Test Case 8.2: Non-Existent Worktree**
- [ ] Try to remove non-existent worktree
- [ ] Verify error message is clear

---

### 9. Prune Worktrees

**Test Case 9.1: Orphaned Worktree Cleanup**
- [ ] Manually delete a worktree directory (simulating orphan)
- [ ] Run `./auto-worktree prune`
- [ ] Verify orphaned metadata is cleaned up
- [ ] No errors

**Test Case 9.2: On Startup**
- [ ] Verify prune runs silently on startup
- [ ] No messages shown (silent operation)

---

### 10. Cross-Platform Testing

#### macOS Specific
- [ ] Test on macOS
- [ ] Verify paths use `/Users/username`
- [ ] Tmux integration works if installed
- [ ] Keyboard shortcuts work (Escape, etc.)

#### Linux Specific
- [ ] Test on Linux (Ubuntu, Fedora, etc.)
- [ ] Verify paths use `/home/username`
- [ ] Tmux integration works if installed
- [ ] Terminal colors display correctly

#### Windows Specific (if applicable)
- [ ] Test on Windows with Git Bash
- [ ] Verify path handling (backslashes)
- [ ] Verify line endings (CRLF handling)
- [ ] Terminal detection works

---

## Comparative Testing: Bash vs Go

### Setup
- [ ] Have both versions available:
  ```bash
  # Bash version
  source ~/path/to/aw.sh

  # Go version
  ~/path/to/auto-worktree
  ```

### Test Cases

**TC A: Feature Parity - Commands**
- [ ] Run same commands in both versions
- [ ] Verify output is functionally identical
- [ ] Branch names generated are identical
- [ ] Worktree paths are identical

**TC B: User Experience - Menu**
- [ ] Compare menu layouts
- [ ] Verify same options available
- [ ] Navigation feels consistent
- [ ] Help text is present

**TC C: User Experience - Error Messages**
- [ ] Compare error messages for same errors
- [ ] Go version should be as clear as Bash
- [ ] Error handling is consistent

**TC D: Performance**
- [ ] Time common operations:
  ```bash
  time ./auto-worktree new
  time ./auto-worktree list
  time ./auto-worktree issue
  ```
- [ ] Go version should be equal or faster
- [ ] No noticeable lag

**TC E: Configuration**
- [ ] Set configs in both versions
- [ ] Verify git config usage is identical
- [ ] Settings persist across runs

---

## Edge Case Testing

### Branch Names
- [ ] Test with Unicode: `café-feature`
- [ ] Test with numbers: `v1.0.0-release`
- [ ] Test with paths: `feature/sub/path`
- [ ] Test very long names (>100 chars)

### Configuration
- [ ] Missing git config (should use defaults)
- [ ] Invalid provider value (should error)
- [ ] Missing gh CLI (should show helpful message)

### Network (if testing with real GitHub)
- [ ] Test with network disabled (should fail gracefully)
- [ ] Test with slow network (should not hang)

---

## Test Results Template

### Test Session Info
- Date: _______________
- Platform: macOS / Linux / Windows
- Go Version: _______________
- Git Version: _______________
- Tester: _______________

### Passed Tests
```
Total Passed: _____ / _____
- Test 1.1: ✅/❌
- Test 1.2: ✅/❌
- Test 2.1: ✅/❌
... (list all tests)
```

### Failed Tests
```
- Test X.X: [Description of failure]
  Reproduction steps: ...
  Expected behavior: ...
  Actual behavior: ...

- Test Y.Y: [Description of failure]
  ...
```

### Issues Found
```
1. Issue: [Description]
   Severity: Critical/High/Medium/Low
   Steps to reproduce: ...
   Suggested fix: ...

2. Issue: [Description]
   ...
```

### Overall Assessment
- Functionality: Complete / Partial / Missing
- User Experience: Excellent / Good / Needs Improvement
- Performance: Excellent / Acceptable / Needs Optimization
- Feature Parity: 100% / [X]% / [List missing features]

### Recommendations
```
1. Priority issues to fix:
   - [Issue]
   - [Issue]

2. Nice-to-have improvements:
   - [Enhancement]
   - [Enhancement]

3. Documentation needs:
   - [Docs needed]
   - [Docs needed]
```

### Sign-Off
- Tester Name: _______________
- Date: _______________
- Approved: Yes / No
- Comments: _______________

---

## Acceptance Criteria

The Go version is ready for production when:

✅ All core commands work (new, list, resume, issue, cleanup, settings)
✅ Menu navigation and workflows feel natural
✅ Error messages are clear and helpful
✅ Performance is equal or better than Bash version
✅ Feature parity is 100% or documented gaps are acceptable
✅ No critical bugs found
✅ Cross-platform compatibility verified
✅ User feedback is positive
✅ Documentation is complete

---

## Sign-Off

- [ ] UAT Lead: _________________________ Date: _______
- [ ] Development Team: _________________________ Date: _______
- [ ] Product Manager: _________________________ Date: _______

---

## Next Steps

If UAT is successful:
1. [ ] Create release notes
2. [ ] Update README with Go version info
3. [ ] Announce deprecation of Bash version (if applicable)
4. [ ] Tag release in git
5. [ ] Publish to package managers (Homebrew, etc.)

If issues found:
1. [ ] Create GitHub issues for each bug/gap
2. [ ] Prioritize fixes
3. [ ] Schedule fix verification
4. [ ] Return to UAT once issues resolved
