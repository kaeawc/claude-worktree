#compdef auto-worktree aw

# Zsh completion for auto-worktree
# This file provides tab completion for the auto-worktree command and its 'aw' alias

_auto_worktree() {
  local -a commands
  commands=(
    'new:Create a new worktree'
    'resume:Resume an existing worktree'
    'issue:Work on an issue (GitHub, GitLab, JIRA, or Linear)'
    'create:Create a new issue with optional template'
    'pr:Review a GitHub PR or GitLab MR'
    'list:List existing worktrees'
    'cleanup:Interactively clean up worktrees'
    'settings:Configure per-repository settings'
    'help:Show help message'
  )

  local curcontext="$curcontext" state

  _arguments -C \
    '1:command:->command' \
    '*::arg:->args'

  case $state in
    command)
      _describe -t commands 'auto-worktree commands' commands
      ;;
    args)
      case $words[1] in
        issue)
          local -a issues
          if command -v gh &>/dev/null; then
            issues=(${(f)"$(gh issue list --limit 100 --state open --json number,title \
              --jq '.[] | "\(.number):\(.title | gsub(":";" "))"' 2>/dev/null)"})
          fi
          if [[ ${#issues[@]} -gt 0 ]]; then
            _describe -t issues 'open issues' issues
          fi
          ;;
        pr)
          local -a prs
          if command -v gh &>/dev/null; then
            prs=(${(f)"$(gh pr list --limit 100 --state open --json number,title \
              --jq '.[] | "\(.number):\(.title | gsub(":";" "))"' 2>/dev/null)"})
          fi
          if [[ ${#prs[@]} -gt 0 ]]; then
            _describe -t prs 'open pull requests' prs
          fi
          ;;
      esac
      ;;
  esac
}

# Register completion for both 'auto-worktree' and 'aw' commands
compdef _auto_worktree auto-worktree
compdef _auto_worktree aw
