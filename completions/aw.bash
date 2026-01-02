# Bash completion for auto-worktree
# This file provides tab completion for the auto-worktree command and its 'aw' alias

_auto_worktree_completions() {
  local cur prev words cword
  _init_completion || return

  # Define available commands
  local commands="new resume issue create pr list cleanup settings help"

  # If we're completing the first argument (the command)
  if [[ $cword -eq 1 ]]; then
    mapfile -t COMPREPLY < <(compgen -W "$commands" -- "$cur")
    return 0
  fi

  # Command-specific completions for subsequent arguments
  local command="${words[1]}"

  case "$command" in
    issue)
      # Provide dynamic issue number completion from GitHub
      if command -v gh &>/dev/null; then
        local issues
        # Fetch open issues and format as "number - title"
        mapfile -t issues < <(gh issue list --limit 100 --state open --json number,title \
          --jq '.[] | "\(.number)\t\(.title)"' 2>/dev/null | awk -F'\t' '{print $1}')

        if [[ ${#issues[@]} -gt 0 ]]; then
          mapfile -t COMPREPLY < <(compgen -W "${issues[*]}" -- "$cur")
        fi
      fi
      ;;
    pr)
      # Provide dynamic PR number completion from GitHub
      if command -v gh &>/dev/null; then
        local prs
        # Fetch open PRs and format as "number - title"
        mapfile -t prs < <(gh pr list --limit 100 --state open --json number,title \
          --jq '.[] | "\(.number)\t\(.title)"' 2>/dev/null | awk -F'\t' '{print $1}')

        if [[ ${#prs[@]} -gt 0 ]]; then
          mapfile -t COMPREPLY < <(compgen -W "${prs[*]}" -- "$cur")
        fi
      fi
      ;;
    settings)
      # Provide settings subcommands
      if [[ $cword -eq 2 ]]; then
        local settings_commands="set get list reset"
        mapfile -t COMPREPLY < <(compgen -W "$settings_commands" -- "$cur")
      fi
      ;;
    new|resume|create|list|cleanup|help)
      # These commands don't have specific completions
      COMPREPLY=()
      ;;
  esac

  return 0
}

# Register completion for both 'auto-worktree' and 'aw' commands
complete -F _auto_worktree_completions auto-worktree
complete -F _auto_worktree_completions aw
