#!/bin/bash

# Source this file from ~/.zshrc to load the shell function `auto-worktree`
#
# Usage:
#   auto-worktree                    # Interactive menu
#   auto-worktree new                # Create new worktree
#   auto-worktree resume             # Resume existing worktree
#   auto-worktree issue [id]         # Work on an issue (GitHub #123, GitLab #456, or JIRA PROJ-123)
#   auto-worktree pr [num]           # Review a GitHub PR or GitLab MR
#   auto-worktree list               # List existing worktrees
#   auto-worktree settings           # Configure per-repository settings
#
# Configuration (per-repository via git config):
#   git config auto-worktree.issue-provider github|gitlab|jira  # Set issue provider
#   git config auto-worktree.jira-server <URL>                  # Set JIRA server URL
#   git config auto-worktree.jira-project <KEY>                 # Set default JIRA project
#   git config auto-worktree.gitlab-server <URL>                # Set GitLab server URL (for self-hosted)
#   git config auto-worktree.gitlab-project <GROUP/PROJECT>     # Set default GitLab project path
#   git config auto-worktree.linear-team <TEAM>                 # Set default Linear team
#   git config auto-worktree.ai-tool <name>                     # claude|codex|gemini|jules|skip
#   git config auto-worktree.issue-autoselect <bool>            # true/false for AI auto-select
#   git config auto-worktree.pr-autoselect <bool>               # true/false for AI auto-select
#   git config auto-worktree.run-hooks <bool>                   # true/false to enable/disable git hooks (default: true)
#   git config auto-worktree.fail-on-hook-error <bool>          # true/false to fail on hook errors (default: false)
#   git config auto-worktree.custom-hooks "<hook1> <hook2>"     # Space or comma-separated list of custom hooks to run

# ============================================================================
# Dependencies check
# ============================================================================

_aw_check_deps() {
  local missing=()

  if ! command -v gum &> /dev/null; then
    missing+=("gum (install with: brew install gum)")
  fi

  if ! command -v jq &> /dev/null; then
    missing+=("jq (install with: brew install jq)")
  fi

  # Note: gh and jira are optional based on project configuration
  # We'll check for them when needed based on the issue provider setting

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Missing required dependencies:"
    for dep in "${missing[@]}"; do
      echo "  - $dep"
    done
    return 1
  fi

  return 0
}

_aw_check_issue_provider_deps() {
  # Check for issue provider specific dependencies
  local provider="$1"

  case "$provider" in
    "github")
      if ! command -v gh &> /dev/null; then
        gum style --foreground 1 "Error: GitHub CLI (gh) is required for GitHub issue integration"
        echo "Install with: brew install gh"
        return 1
      fi
      ;;
    "gitlab")
      if ! command -v glab &> /dev/null; then
        gum style --foreground 1 "Error: GitLab CLI (glab) is required for GitLab issue integration"
        echo ""
        echo "Install with:"
        echo "  • macOS:     brew install glab"
        echo "  • Linux:     See https://gitlab.com/gitlab-org/cli#installation"
        echo "  • Windows:   scoop install glab"
        echo ""
        echo "After installation, authenticate with GitLab:"
        echo "  glab auth login"
        return 1
      fi
      ;;
    "jira")
      if ! command -v jira &> /dev/null; then
        gum style --foreground 1 "Error: JIRA CLI is required for JIRA issue integration"
        echo ""
        echo "Install with:"
        echo "  • macOS:     brew install ankitpokhrel/jira-cli/jira-cli"
        echo "  • Linux:     See https://github.com/ankitpokhrel/jira-cli#installation"
        echo "  • Docker:    docker pull ghcr.io/ankitpokhrel/jira-cli:latest"
        echo ""
        echo "After installation, configure JIRA:"
        echo "  jira init"
        return 1
      fi
      ;;
    "linear")
      if ! command -v linear &> /dev/null; then
        gum style --foreground 1 "Error: Linear CLI is required for Linear issue integration"
        echo ""
        echo "Install with:"
        echo "  • macOS:     brew install schpet/tap/linear"
        echo "  • Deno:      deno install -A --reload -f -g -n linear jsr:@schpet/linear-cli"
        echo "  • Other:     See https://github.com/schpet/linear-cli#installation"
        echo ""
        echo "After installation, configure Linear:"
        echo "  1. Create an API key at https://linear.app/settings/account/security"
        echo "  2. Set environment variable: export LINEAR_API_KEY=your_key_here"
        return 1
      fi
      ;;
  esac

  return 0
}

# ============================================================================
# AI Command Resolution
# ============================================================================

# Global variables for AI tool selection
# Note: AI_CMD and AI_RESUME_CMD are arrays to properly handle arguments in zsh
AI_CMD=()
AI_CMD_NAME=""
AI_RESUME_CMD=()

# Load saved AI tool preference (per-repository git config)
_load_ai_preference() {
  git config --get auto-worktree.ai-tool 2>/dev/null || echo ""
}

# Save AI tool preference (per-repository git config)
_save_ai_preference() {
  local tool="$1"
  if [[ -z "$tool" ]]; then
    git config --unset auto-worktree.ai-tool 2>/dev/null
  else
    git config auto-worktree.ai-tool "$tool"
  fi
}

_aw_get_issue_autoselect() {
  local value
  value=$(git config --get --bool auto-worktree.issue-autoselect 2>/dev/null || echo "")
  if [[ -z "$value" ]]; then
    echo "true"
  else
    echo "$value"
  fi
}

_aw_get_pr_autoselect() {
  local value
  value=$(git config --get --bool auto-worktree.pr-autoselect 2>/dev/null || echo "")
  if [[ -z "$value" ]]; then
    echo "true"
  else
    echo "$value"
  fi
}

# Check if auto-select is disabled
_is_autoselect_disabled() {
  [[ "$(_aw_get_issue_autoselect)" == "false" ]]
}

# Disable auto-select
_disable_autoselect() {
  git config auto-worktree.issue-autoselect false
}

# Enable auto-select
_enable_autoselect() {
  git config auto-worktree.issue-autoselect true
}

# Check if PR auto-select is disabled
_is_pr_autoselect_disabled() {
  [[ "$(_aw_get_pr_autoselect)" == "false" ]]
}

# Disable PR auto-select
_disable_pr_autoselect() {
  git config auto-worktree.pr-autoselect false
}

# Enable PR auto-select
_enable_pr_autoselect() {
  git config auto-worktree.pr-autoselect true
}

# AI-powered issue selection - filters to top 5 issues in priority order
_ai_select_issues() {
  local issues="$1"
  local highlighted_issues="$2"
  local repo_info="$3"

  # Create a temporary file with the issue list
  local temp_issues=$(mktemp)
  echo "$issues" > "$temp_issues"

  # Prepare the prompt for AI
  local prompt="Analyze the following GitHub issues and select the top 5 issues that would be best to work on next. Consider:
- Priority labels (high priority, urgent, etc.)
- Issue type (bug fixes are often higher priority than features)
- Labels like 'good first issue' or 'help wanted'
- Issue complexity and impact
- Any context from the repository: $repo_info

Return ONLY the top 5 issue numbers in priority order (one per line), formatted as just the numbers (e.g., '42').

Issues:
$(cat "$temp_issues")

Return only the 5 issue numbers, one per line, nothing else."

  # Use the configured AI tool to select issues
  _resolve_ai_command || {
    rm -f "$temp_issues"
    return 1
  }

  if [[ "${AI_CMD[1]}" == "skip" ]]; then
    rm -f "$temp_issues"
    return 1
  fi

  # Run AI command with the prompt
  local selected_numbers
  selected_numbers=$(echo "$prompt" | "${AI_CMD[@]}" --no-tty 2>/dev/null | grep -E '^[0-9]+$' | head -5)

  rm -f "$temp_issues"

  if [[ -z "$selected_numbers" ]]; then
    return 1
  fi

  # Filter highlighted issues to only include selected ones, in priority order
  local filtered=""
  while IFS= read -r num; do
    local matching_issue=$(echo "$highlighted_issues" | grep -E "^(● )?#${num} \|" | head -1)
    if [[ -n "$matching_issue" ]]; then
      filtered+="${matching_issue}"$'\n'
    fi
  done <<< "$selected_numbers"

  echo "$filtered"
}

# AI-powered Linear issue selection - filters to top 5 issues in priority order
_ai_select_linear_issues() {
  local issues="$1"
  local highlighted_issues="$2"

  # Create a temporary file with the issue list
  local temp_issues=$(mktemp)
  echo "$issues" > "$temp_issues"

  # Prepare the prompt for AI
  local prompt="Analyze the following Linear issues and select the top 5 issues that would be best to work on next. Consider:
- Priority and status
- Issue complexity and impact
- Dependencies between issues
- Team capacity and workflow

Return ONLY the top 5 issue IDs in priority order (one per line), formatted as issue IDs (e.g., 'TEAM-42').

Issues:
$(cat "$temp_issues")

Return only the 5 issue IDs, one per line, nothing else."

  # Use the configured AI tool to select issues
  _resolve_ai_command || {
    rm -f "$temp_issues"
    return 1
  }

  if [[ "${AI_CMD[1]}" == "skip" ]]; then
    rm -f "$temp_issues"
    return 1
  fi

  # Run AI command with the prompt
  local selected_ids
  selected_ids=$(echo "$prompt" | "${AI_CMD[@]}" --no-tty 2>/dev/null | grep -E '^[A-Z][A-Z0-9]+-[0-9]+$' | head -5)

  rm -f "$temp_issues"

  if [[ -z "$selected_ids" ]]; then
    return 1
  fi

  # Filter highlighted issues to only include selected ones, in priority order
  local filtered=""
  while IFS= read -r id; do
    local matching_issue=$(echo "$highlighted_issues" | grep -E "^(● )?${id} \|" | head -1)
    if [[ -n "$matching_issue" ]]; then
      filtered+="${matching_issue}"$'\n'
    fi
  done <<< "$selected_ids"

  echo "$filtered"
}

# AI-powered PR selection - filters to top 5 PRs in priority order
_ai_select_prs() {
  local prs="$1"
  local highlighted_prs="$2"
  local current_user="$3"
  local repo_info="$4"

  # Create a temporary file with the PR list
  local temp_prs=$(mktemp)
  echo "$prs" > "$temp_prs"

  # Prepare the prompt for AI
  local prompt="Analyze the following GitHub Pull Requests and select the top 5 PRs that would be best to review next. Consider the following criteria in priority order:

1. PRs where the current user ($current_user) was requested as a reviewer (highest priority)
2. PRs with no reviews yet (need attention)
3. Smaller PRs with fewer changes (easier to review, faster feedback)
4. PRs with 100% passing checks (✓ status) - prefer these over failing (✗) or pending (○)
5. Author reputation: prefer maintainers/core contributors over occasional contributors

Return ONLY the top 5 PR numbers in priority order (one per line), formatted as just the numbers (e.g., '42').

Repository: $repo_info
Current user: $current_user

Pull Requests:
$(cat "$temp_prs")

Return only the 5 PR numbers, one per line, nothing else."

  # Use the configured AI tool to select PRs
  _resolve_ai_command || {
    rm -f "$temp_prs"
    return 1
  }

  if [[ "${AI_CMD[1]}" == "skip" ]]; then
    rm -f "$temp_prs"
    return 1
  fi

  # Run AI command with the prompt
  local selected_numbers
  selected_numbers=$(echo "$prompt" | "${AI_CMD[@]}" --no-tty 2>/dev/null | grep -E '^[0-9]+$' | head -5)

  rm -f "$temp_prs"

  if [[ -z "$selected_numbers" ]]; then
    return 1
  fi

  # Filter highlighted PRs to only include selected ones, in priority order
  local filtered=""
  while IFS= read -r num; do
    local matching_pr=$(echo "$highlighted_prs" | grep -E "^(● )?#${num} \|" | head -1)
    if [[ -n "$matching_pr" ]]; then
      filtered+="${matching_pr}"$'\n'
    fi
  done <<< "$selected_numbers"

  echo "$filtered"
}

# Install AI tool via interactive menu
_install_ai_tool() {
  echo ""
  gum style --foreground 3 "No AI coding assistant found."
  echo ""

  local choice=$(gum choose \
    "Install Claude Code (Anthropic)" \
    "Install Codex CLI (OpenAI)" \
    "Install Gemini CLI (Google)" \
    "Install Google Jules CLI (Google)" \
    "Skip - don't use an AI tool" \
    "Cancel")

  case "$choice" in
    "Install Claude Code (Anthropic)")
      echo ""
      gum style --foreground 6 "Install Claude Code with one of the following methods:"
      echo "  • macOS:   brew install claude"
      echo "  • npm:     npm install -g @anthropic-ai/claude-code"
      echo ""
      echo "For more information, visit: https://github.com/anthropics/claude-code"
      echo ""
      return 1
      ;;
    "Install Codex CLI (OpenAI)")
      echo ""
      gum style --foreground 6 "Install Codex CLI with:"
      echo "  • npm:     npm install -g @openai/codex-cli"
      echo ""
      echo "For more information, visit: https://github.com/openai/codex"
      echo ""
      return 1
      ;;
    "Install Gemini CLI (Google)")
      echo ""
      gum style --foreground 6 "Install Gemini CLI with:"
      echo "  • npm:     npm install -g @google/gemini-cli"
      echo ""
      echo "For more information, visit: https://github.com/google-gemini/gemini-cli"
      echo ""
      return 1
      ;;
    "Install Google Jules CLI (Google)")
      echo ""
      gum style --foreground 6 "Install Google Jules CLI with:"
      echo "  • npm:     npm install -g @google/jules"
      echo ""
      echo "For more information, visit: https://jules.google/docs"
      echo ""
      return 1
      ;;
    "Skip - don't use an AI tool")
      AI_CMD=(skip)
      AI_CMD_NAME="none"
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

_resolve_ai_command() {
  local claude_available=false
  local codex_available=false
  local gemini_available=false
  local jules_available=false
  local claude_path=""
  local codex_path=""
  local gemini_path=""
  local jules_path=""

  # Check which tools are available and get their full paths
  claude_path=$(command -v claude 2>/dev/null)
  [[ -n "$claude_path" ]] && claude_available=true

  codex_path=$(command -v codex 2>/dev/null)
  [[ -n "$codex_path" ]] && codex_available=true

  gemini_path=$(command -v gemini 2>/dev/null)
  [[ -n "$gemini_path" ]] && gemini_available=true

  jules_path=$(command -v jules 2>/dev/null)
  [[ -n "$jules_path" ]] && jules_available=true

  # Check for saved preference first
  local saved_pref=$(_load_ai_preference)

  if [[ -n "$saved_pref" ]]; then
    case "$saved_pref" in
      claude)
        if [[ "$claude_available" == true ]]; then
          AI_CMD=("$claude_path" --dangerously-skip-permissions)
          AI_CMD_NAME="Claude Code"
          AI_RESUME_CMD=("$claude_path" --dangerously-skip-permissions --continue)
          return 0
        fi
        ;;
      codex)
        if [[ "$codex_available" == true ]]; then
          AI_CMD=("$codex_path" --yolo)
          AI_CMD_NAME="Codex"
          AI_RESUME_CMD=("$codex_path" resume --last)
          return 0
        fi
        ;;
      gemini)
        if [[ "$gemini_available" == true ]]; then
          AI_CMD=("$gemini_path" --yolo)
          AI_CMD_NAME="Gemini CLI"
          AI_RESUME_CMD=("$gemini_path" --resume)
          return 0
        fi
        ;;
      jules)
        if [[ "$jules_available" == true ]]; then
          AI_CMD=("$jules_path")
          AI_CMD_NAME="Google Jules CLI"
          AI_RESUME_CMD=("$jules_path")
          return 0
        fi
        ;;
      skip)
        AI_CMD=(skip)
        AI_CMD_NAME="none"
        return 0
        ;;
    esac
    # If we get here, saved preference is no longer valid (tool uninstalled)
    # Fall through to normal selection
  fi

  # Count available tools
  local available_count=0
  [[ "$claude_available" == true ]] && ((available_count++))
  [[ "$codex_available" == true ]] && ((available_count++))
  [[ "$gemini_available" == true ]] && ((available_count++))
  [[ "$jules_available" == true ]] && ((available_count++))

  # If multiple tools are available, let user choose
  if [[ $available_count -gt 1 ]]; then
    echo ""
    gum style --foreground 6 "Multiple AI coding assistants detected!"
    echo ""

    # Build menu options dynamically
    local options=()
    [[ "$claude_available" == true ]] && options+=("Claude Code (Anthropic)")
    [[ "$codex_available" == true ]] && options+=("Codex CLI (OpenAI)")
    [[ "$gemini_available" == true ]] && options+=("Gemini CLI (Google)")
    [[ "$jules_available" == true ]] && options+=("Google Jules CLI (Google)")
    options+=("Skip - don't use an AI tool")

    local choice=$(gum choose "${options[@]}")

    case "$choice" in
      "Claude Code (Anthropic)")
        AI_CMD=("$claude_path" --dangerously-skip-permissions)
        AI_CMD_NAME="Claude Code"
        AI_RESUME_CMD=("$claude_path" --dangerously-skip-permissions --continue)
        ;;
      "Codex CLI (OpenAI)")
        AI_CMD=("$codex_path" --yolo)
        AI_CMD_NAME="Codex"
        AI_RESUME_CMD=("$codex_path" resume --last)
        ;;
      "Gemini CLI (Google)")
        AI_CMD=("$gemini_path" --yolo)
        AI_CMD_NAME="Gemini CLI"
        AI_RESUME_CMD=("$gemini_path" --resume)
        ;;
      "Google Jules CLI (Google)")
        AI_CMD=("$jules_path")
        AI_CMD_NAME="Google Jules CLI"
        AI_RESUME_CMD=("$jules_path")
        ;;
      "Skip - don't use an AI tool")
        AI_CMD=(skip)
        AI_CMD_NAME="none"
        return 0
        ;;
      *)
        return 1
        ;;
    esac

    # Ask if this choice should be saved
    echo ""
    if gum confirm "Save this as your default choice?"; then
      case "$choice" in
        "Claude Code (Anthropic)")
          _save_ai_preference "claude"
          gum style --foreground 2 "Saved Claude Code as default"
          ;;
        "Codex CLI (OpenAI)")
          _save_ai_preference "codex"
          gum style --foreground 2 "Saved Codex as default"
          ;;
        "Gemini CLI (Google)")
          _save_ai_preference "gemini"
          gum style --foreground 2 "Saved Gemini CLI as default"
          ;;
        "Google Jules CLI (Google)")
          _save_ai_preference "jules"
          gum style --foreground 2 "Saved Google Jules CLI as default"
          ;;
        "Skip - don't use an AI tool")
          _save_ai_preference "skip"
          gum style --foreground 2 "Saved preference to skip AI tool"
          ;;
      esac
      echo ""
    fi

    return 0
  fi

  # Only one tool available - use it
  if [[ "$claude_available" == true ]]; then
    AI_CMD=("$claude_path" --dangerously-skip-permissions)
    AI_CMD_NAME="Claude Code"
    AI_RESUME_CMD=("$claude_path" --dangerously-skip-permissions --continue)
    return 0
  fi

  if [[ "$codex_available" == true ]]; then
    AI_CMD=("$codex_path" --yolo)
    AI_CMD_NAME="Codex"
    AI_RESUME_CMD=("$codex_path" resume --last)
    return 0
  fi

  if [[ "$gemini_available" == true ]]; then
    AI_CMD=("$gemini_path" --yolo)
    AI_CMD_NAME="Gemini CLI"
    AI_RESUME_CMD=("$gemini_path" --resume)
    return 0
  fi

  if [[ "$jules_available" == true ]]; then
    AI_CMD=("$jules_path")
    AI_CMD_NAME="Google Jules CLI"
    AI_RESUME_CMD=("$jules_path")
    return 0
  fi

  # No tools available - show installation menu
  _install_ai_tool
  return $?
}

# ============================================================================
# Word lists for random names
# ============================================================================

_WORKTREE_WORDS=(
  able acid acme acre aged airy alfa ally also alto amid
  apex aqua arch area army arte atom aunt auto away axis
  baby back ball band bank barn base bath bead beam bean
  bear beat been beer bell belt bend best beta bike bill
  bird bits blaz blob blue boat body bold bolt bomb bond
  bone book boot born boss both bowl brad brew brig brown
  buck bulb bulk bull burn bush busy byte cafe cage cake
  call calm came camp cane cape card care carl cart case
  cash cast cave cell cent chad chef chip city clam clan
  clay clip club coal coat code coil coin cold cole come
  cone cook cool cope copy cord core cork corn cost cove
  crab crew crop crow cube cure curl dark dash data date
  dawn days dead deal dean dear debt deck deep deer dell
  demo deny desk dial dice died diet dime dine dirt disc
  dish dive dock does dome done doom door dose dove down
  doze drag draw drew drip drop drum dual duck dude dune
  dusk dust duty each earl earn ease east easy echo edge
  edit else ends epic even ever exam exit eyed eyes face
  fact fade fail fair fall fame fang fare farm fast fate
  fawn fear feat feed feel feet fell felt fern fest file
  fill film find fine fire firm fish fist five flag flat
  flaw fled flee flew flex flip flow flux foam fold folk
  fond font food fool foot ford fork form fort foul four
  free from fuel full fund fuse gain game gang gate gave
  gear gene gift girl give glad glow glue goal goat goes
  gold golf gone good gore grab grad gram gray grew grey
  grid grim grin grip grit grow gulf gust hack hail hair
  half hall halt hand hang hard hare harm hart hash hate
  haul have hawk haze head heal heap hear heat heck held
  help herb here hero hide high hike hill hint hire hold
  hole holy home hone hood hook hope horn hose host hour
  huge hulk hunt hurt icon idea idle inch info into iron
  isle item jack jade jane jazz jean jest joey john join
  joke july jump june junk jury just keen keep kent kept
  keys kick kind king kirk kiss kite knee knew know kong
  lace lack lady laid lake lamb lame lamp land lane lark
  last late lava lawn lead leaf lean leap left lend lens
  lent less levy lied lift like lily limb lime line link
  lion list lite live load loaf loan lock loft logo lone
  long look loop lord lore lose loss lost loud love luck
  luke lump lung lurk made mage maid mail main make male
  mall malt mann many maps mare mark mars mart mask mass
  mast mate math matt maze mead meal mean meat meek meet
  meld melt memo mesh mess mica mice mild mile milk mill
  mime mind mine mint miss mist moan mock mode mold molt
  monk moon moor moss most moth move mule murk muse mush
  musk must mute myth nail name navy near neat neck need
  nest newt next nice nick nine node noon norm nose note
  nova obey odor okay omen once only onto open oral oven
  over pace pack pact page paid pail pain pair pale palm
  pane park part pass past path pave pawn peak pear peat
  peck peek peer pelt penn perk pest pets peak pier pike
  pile pill pine pink pint pipe pith plan play plea plot
  plow plug plum plus poem poet pole poll polo pond pony
  pool poor pore pork port pose post pour pray prey prim
  prod prom prop puck pull pulp pump punk pure push quit
  quiz race rack raft rage raid rail rain rake ramp rand
  rang rank rare rate rave rays read real ream reap rear
  reed reef reel rely rend rent rest rice rich rick ride
  rife rift rift ring riot ripe rise risk rite road roam
  roar robe rock rode role roll rome roof room root rope
  rose rosy ruby rude ruin rule rump rune rung runs runt
  rush rust ruth sack safe saga sage said sail sake sale
  salt same sand sane sang sank save sawn says scan scar
  seal seam sean sear seat sect seed seek seem seen seep
  self sell send sent sept sets shad shah sham shed shin
  ship shoe shop shot show shut sick side sift sign silk
  sill silo silt sing sink site size skew skin skip slab
  slam slap slat slaw sled slew slim slip slit slow slug
  snap snip snow soak soap soar sock soda soft soil sold
  sole solo some song soon soot sore sort soul soup sour
  span spar spec sped spin spit spot spun spur stab stag
  star stay stem step stew stir stop stow stub stud subs
  such suit sulk sung sunk sure surf swan swap sway swim
  tail take tale talk tall tame tang tank tape tart task
  taxi team tear tech teem teen tell temp tend tent term
  test text than that thaw thee them then they thin this
  thou thud thus tick tide tidy tied tier tile till tilt
  time tint tiny tips tire toad toil told toll tomb tome
  tone tony took tool torn toss tour town toys trap tray
  tree trek trim trio trip trod trot troy true tuba tube
  tuck tuft tune turf turn tusk twas twig twin type ugly
  undo unit upon used user vain vale vane vary vase vast
  veil vein vent verb very vest veto vice view vile vine
  void volt vote wade wage wait wake walk wall wand wane
  want ward warm warn warp wars wary wash wasp wave wavy
  waxy ways weak wear weed week weep weld well went wept
  were west what when whey whim whip whom wide wife wild
  will wilt wind wine wing wink wipe wire wise wish with
  woke wolf womb wood wool word wore work worm worn wove
  wrap wren yang yard yarn yawn year yell your zero zest
  zinc zone zoom
)

_WORKTREE_COLORS=(
  red orange yellow green blue purple pink brown black white
  gray cyan magenta teal navy coral salmon peach mint lime
  gold silver bronze ruby jade amber ivory onyx pearl slate
  crimson scarlet maroon olive azure indigo violet lavender
  turquoise aqua beige cream tan khaki rust copper plum rose
)

# ============================================================================
# Helper functions
# ============================================================================

_aw_ensure_git_repo() {
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    gum style --foreground 1 "Error: Not in a git repository"
    return 1
  fi
  return 0
}

_aw_get_repo_info() {
  _AW_GIT_ROOT=$(git rev-parse --show-toplevel)
  _AW_SOURCE_FOLDER=$(basename "$_AW_GIT_ROOT")
  _AW_WORKTREE_BASE="$HOME/worktrees/$_AW_SOURCE_FOLDER"
}

_aw_prune_worktrees() {
  local count_before=$(git worktree list --porcelain 2>/dev/null | grep -c "^worktree " || echo 0)
  git worktree prune 2>/dev/null
  local count_after=$(git worktree list --porcelain 2>/dev/null | grep -c "^worktree " || echo 0)
  local pruned=$((count_before - count_after))
  if [[ $pruned -gt 0 ]]; then
    gum style --foreground 3 "Pruned $pruned orphaned worktree(s)"
    echo ""
  fi
}

_aw_generate_random_name() {
  # In zsh, arrays are 1-indexed, so we need to add 1 to the result of modulo
  local color_idx=$(( ($RANDOM % ${#_WORKTREE_COLORS[@]}) + 1 ))
  local word1_idx=$(( ($RANDOM % ${#_WORKTREE_WORDS[@]}) + 1 ))
  local word2_idx=$(( ($RANDOM % ${#_WORKTREE_WORDS[@]}) + 1 ))

  local color=${_WORKTREE_COLORS[$color_idx]}
  local word1=${_WORKTREE_WORDS[$word1_idx]}
  local word2=${_WORKTREE_WORDS[$word2_idx]}
  echo "${color}-${word1}-${word2}"
}

_aw_sanitize_branch_name() {
  echo "$1" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$//'
}

_aw_get_file_mtime() {
  # Get file modification time in Unix timestamp format
  # Works on both macOS/BSD and Linux
  # Returns: Unix timestamp (seconds since epoch)
  local file_path="$1"

  if [[ "$(uname)" == "Darwin" ]] || [[ "$(uname)" == *"BSD"* ]]; then
    # macOS/BSD syntax
    stat -f %m "$file_path" 2>/dev/null
  else
    # Linux syntax
    stat -c %Y "$file_path" 2>/dev/null
  fi
}

# ============================================================================
# Project configuration (git config based)
# ============================================================================

_aw_get_issue_provider() {
  # Get the configured issue provider
  # Returns: github, gitlab, jira, linear, or empty string if not configured
  git config --get auto-worktree.issue-provider 2>/dev/null || echo ""
}

_aw_set_issue_provider() {
  # Set the issue provider for this repository
  local provider="$1"

  if [[ "$provider" != "github" ]] && [[ "$provider" != "jira" ]] && [[ "$provider" != "gitlab" ]] && [[ "$provider" != "linear" ]]; then
    gum style --foreground 1 "Error: Invalid provider. Must be 'github', 'gitlab', 'jira', or 'linear'"
    return 1
  fi

  git config auto-worktree.issue-provider "$provider"
  gum style --foreground 2 "✓ Issue provider set to: $provider"
}

_aw_get_jira_server() {
  # Get the configured JIRA server URL
  git config --get auto-worktree.jira-server 2>/dev/null || echo ""
}

_aw_set_jira_server() {
  # Set the JIRA server URL for this repository
  local server="$1"
  git config auto-worktree.jira-server "$server"
  gum style --foreground 2 "✓ JIRA server set to: $server"
}

_aw_get_jira_project() {
  # Get the configured default JIRA project key
  git config --get auto-worktree.jira-project 2>/dev/null || echo ""
}

_aw_set_jira_project() {
  # Set the default JIRA project key for this repository
  local project="$1"
  git config auto-worktree.jira-project "$project"
  gum style --foreground 2 "✓ JIRA project set to: $project"
}

_aw_get_gitlab_server() {
  # Get the configured GitLab server URL
  git config --get auto-worktree.gitlab-server 2>/dev/null || echo ""
}

_aw_set_gitlab_server() {
  # Set the GitLab server URL for this repository
  local server="$1"
  git config auto-worktree.gitlab-server "$server"
  gum style --foreground 2 "✓ GitLab server set to: $server"
}

_aw_get_gitlab_project() {
  # Get the configured default GitLab project path
  git config --get auto-worktree.gitlab-project 2>/dev/null || echo ""
}

_aw_set_gitlab_project() {
  # Set the default GitLab project path for this repository
  local project="$1"
  git config auto-worktree.gitlab-project "$project"
  gum style --foreground 2 "✓ GitLab project set to: $project"
}

_aw_get_issue_templates_dir() {
  # Get the configured issue templates directory for current provider
  git config --get auto-worktree.issue-templates-dir 2>/dev/null || echo ""
}

_aw_set_issue_templates_dir() {
  # Set the issue templates directory for this repository
  local dir="$1"
  git config auto-worktree.issue-templates-dir "$dir"
  gum style --foreground 2 "✓ Issue templates directory set to: $dir"
}

_aw_get_issue_templates_disabled() {
  # Check if user has disabled issue templates
  # Returns: "true" or "" (empty string means enabled)
  git config --get auto-worktree.issue-templates-disabled 2>/dev/null || echo ""
}

_aw_set_issue_templates_disabled() {
  # Disable issue templates for this repository
  local disabled="$1"  # "true" or "false"
  git config auto-worktree.issue-templates-disabled "$disabled"
}

_aw_get_issue_templates_prompt_disabled() {
  # Check if user wants to skip template prompts in future
  # Returns: "true" or "" (empty string means should prompt)
  git config --get auto-worktree.issue-templates-no-prompt 2>/dev/null || echo ""
}

_aw_set_issue_templates_prompt_disabled() {
  # Set whether to prompt for templates in future
  local disabled="$1"  # "true" or "false"
  git config auto-worktree.issue-templates-no-prompt "$disabled"
}

_aw_get_issue_templates_detected_flag() {
  # Check if we've already notified user about detected templates
  git config --get auto-worktree.issue-templates-detected 2>/dev/null || echo ""
}

_aw_set_issue_templates_detected_flag() {
  # Set flag that we've notified user about templates
  git config auto-worktree.issue-templates-detected "true"
}

_aw_detect_issue_templates() {
  # Auto-detect issue templates for the current provider
  # Args: $1 = provider (github, gitlab, jira, linear)
  # Returns: List of template files (one per line), or empty if none found
  local provider="$1"
  local templates_dir=""

  # Check if user has a custom templates directory configured
  local custom_dir=$(_aw_get_issue_templates_dir)
  if [[ -n "$custom_dir" ]] && [[ -d "$custom_dir" ]]; then
    templates_dir="$custom_dir"
  else
    # Use conventional directories based on provider
    case "$provider" in
      github)
        if [[ -d ".github/ISSUE_TEMPLATE" ]]; then
          templates_dir=".github/ISSUE_TEMPLATE"
        fi
        ;;
      gitlab)
        if [[ -d ".gitlab/issue_templates" ]]; then
          templates_dir=".gitlab/issue_templates"
        fi
        ;;
      jira)
        if [[ -d ".jira/issue_templates" ]]; then
          templates_dir=".jira/issue_templates"
        fi
        ;;
      linear)
        if [[ -d ".linear/issue_templates" ]]; then
          templates_dir=".linear/issue_templates"
        fi
        ;;
    esac
  fi

  # Find all .md files in the templates directory
  if [[ -n "$templates_dir" ]] && [[ -d "$templates_dir" ]]; then
    find "$templates_dir" -maxdepth 1 -name "*.md" -type f 2>/dev/null | sort
  fi
}

_aw_get_template_default_dir() {
  # Get the default template directory for a provider
  # Args: $1 = provider (github, gitlab, jira, linear)
  # Returns: Default directory path
  local provider="$1"

  case "$provider" in
    github)
      echo ".github/ISSUE_TEMPLATE"
      ;;
    gitlab)
      echo ".gitlab/issue_templates"
      ;;
    jira)
      echo ".jira/issue_templates"
      ;;
    linear)
      echo ".linear/issue_templates"
      ;;
  esac
}

_aw_configure_issue_templates() {
  # Interactive configuration for issue templates
  # Args: $1 = provider (github, gitlab, jira, linear)
  # Returns: 0 if templates configured/enabled, 1 if user opts out
  local provider="$1"

  echo ""
  gum style --foreground 6 "Configure Issue Templates"
  echo ""

  # Try to auto-detect templates
  local detected_templates=$(_aw_detect_issue_templates "$provider")
  local default_dir=$(_aw_get_template_default_dir "$provider")

  if [[ -n "$detected_templates" ]]; then
    local template_count=$(echo "$detected_templates" | wc -l | tr -d ' ')
    gum style --foreground 2 "✓ Found $template_count template(s) in $default_dir"
    echo ""

    if gum confirm "Use these templates for issue creation?"; then
      _aw_set_issue_templates_disabled "false"
      return 0
    fi
  else
    gum style --foreground 3 "No templates found in $default_dir"
    echo ""
  fi

  # Ask if user wants to specify custom directory
  if gum confirm "Specify a custom templates directory?"; then
    echo ""
    gum style --foreground 6 "Templates directory path:"
    local custom_dir=$(gum input --placeholder "$default_dir")

    if [[ -n "$custom_dir" ]]; then
      if [[ -d "$custom_dir" ]]; then
        _aw_set_issue_templates_dir "$custom_dir"
        _aw_set_issue_templates_disabled "false"
        return 0
      else
        gum style --foreground 1 "Error: Directory does not exist: $custom_dir"
      fi
    fi
  fi

  # User doesn't want templates - ask about future prompts
  echo ""
  gum style --foreground 3 "Templates will not be used."
  echo ""

  if gum confirm "Skip template prompts for future issue creation?"; then
    _aw_set_issue_templates_prompt_disabled "true"
    gum style --foreground 4 "To re-enable templates later, run:"
    echo "  git config auto-worktree.issue-templates-no-prompt false"
  else
    _aw_set_issue_templates_prompt_disabled "false"
  fi

  _aw_set_issue_templates_disabled "true"
  return 1
}

_aw_configure_jira() {
  # Interactive configuration for JIRA
  echo ""
  gum style --foreground 6 "Configure JIRA for this repository"
  echo ""

  # Get JIRA server URL
  local current_server=$(_aw_get_jira_server)
  gum style --foreground 6 "JIRA Server URL:"
  local server=$(gum input --placeholder "https://your-company.atlassian.net" \
    --value "$current_server")

  if [[ -z "$server" ]]; then
    gum style --foreground 3 "Cancelled"
    return 1
  fi

  _aw_set_jira_server "$server"

  # Get default JIRA project key
  local current_project=$(_aw_get_jira_project)
  echo ""
  gum style --foreground 6 "Default JIRA Project Key (optional, can filter issues):"
  local project=$(gum input --placeholder "PROJ" \
    --value "$current_project")

  if [[ -n "$project" ]]; then
    _aw_set_jira_project "$project"
  fi

  echo ""
  gum style --foreground 2 "JIRA configuration complete!"
  echo ""
  echo "Note: Make sure you've configured the JIRA CLI:"
  echo "  jira init"
  echo ""
}

_aw_configure_gitlab() {
  # Interactive configuration for GitLab
  echo ""
  gum style --foreground 6 "Configure GitLab for this repository"
  echo ""

  # Get GitLab server URL
  local current_server=$(_aw_get_gitlab_server)
  local server=$(gum input --placeholder "https://gitlab.com (or https://gitlab.example.com for self-hosted)" \
    --value "$current_server" \
    --header "GitLab Server URL (leave empty for gitlab.com default):")

  if [[ -n "$server" ]]; then
    _aw_set_gitlab_server "$server"
  fi

  # Get default GitLab project path
  local current_project=$(_aw_get_gitlab_project)
  local project=$(gum input --placeholder "group/project" \
    --value "$current_project" \
    --header "Default GitLab Project Path (optional, can filter issues/MRs):")

  if [[ -n "$project" ]]; then
    _aw_set_gitlab_project "$project"
  fi

  echo ""
  gum style --foreground 2 "GitLab configuration complete!"
  echo ""
  echo "Note: Make sure you've authenticated with the GitLab CLI:"
  echo "  glab auth login"
  echo ""
}

_aw_get_linear_team() {
  # Get the configured default Linear team key
  git config --get auto-worktree.linear-team 2>/dev/null || echo ""
}

_aw_set_linear_team() {
  # Set the default Linear team key for this repository
  local team="$1"
  git config auto-worktree.linear-team "$team"
  gum style --foreground 2 "✓ Linear team set to: $team"
}

_aw_configure_linear() {
  # Interactive configuration for Linear
  echo ""
  gum style --foreground 6 "Configure Linear for this repository"
  echo ""

  # Get default Linear team key
  local current_team=$(_aw_get_linear_team)
  local team=$(gum input --placeholder "TEAM" \
    --value "$current_team" \
    --header "Default Linear Team Key (optional, can filter issues):")

  if [[ -n "$team" ]]; then
    _aw_set_linear_team "$team"
  fi

  echo ""
  gum style --foreground 2 "Linear configuration complete!"
  echo ""
  echo "Note: Make sure you've configured the Linear CLI:"
  echo "  1. Create an API key at https://linear.app/settings/account/security"
  echo "  2. Set environment variable: export LINEAR_API_KEY=your_key_here"
  echo ""
}

_aw_issue_provider_label() {
  local provider="$1"
  case "$provider" in
    github) echo "GitHub Issues" ;;
    gitlab) echo "GitLab Issues" ;;
    jira) echo "JIRA" ;;
    linear) echo "Linear Issues" ;;
    *) echo "not set" ;;
  esac
}

_aw_ai_preference_label() {
  local pref="$1"
  case "$pref" in
    claude) echo "Claude Code" ;;
    codex) echo "Codex CLI" ;;
    gemini) echo "Gemini CLI" ;;
    jules) echo "Google Jules CLI" ;;
    skip) echo "skip AI tool" ;;
    *) echo "auto (prompt when needed)" ;;
  esac
}

_aw_bool_label() {
  local value="$1"
  if [[ "$value" == "true" ]]; then
    echo "enabled"
  else
    echo "disabled"
  fi
}

_aw_clear_issue_provider_settings() {
  git config --unset auto-worktree.issue-provider 2>/dev/null
  git config --unset auto-worktree.jira-server 2>/dev/null
  git config --unset auto-worktree.jira-project 2>/dev/null
  git config --unset auto-worktree.gitlab-server 2>/dev/null
  git config --unset auto-worktree.gitlab-project 2>/dev/null
  git config --unset auto-worktree.linear-team 2>/dev/null
  gum style --foreground 2 "✓ Issue provider settings cleared"
}

_aw_show_settings_summary() {
  local provider=$(_aw_get_issue_provider)
  local provider_label=$(_aw_issue_provider_label "$provider")
  local jira_server=$(_aw_get_jira_server)
  local jira_project=$(_aw_get_jira_project)
  local gitlab_server=$(_aw_get_gitlab_server)
  local gitlab_project=$(_aw_get_gitlab_project)
  local linear_team=$(_aw_get_linear_team)
  local ai_pref=$(_load_ai_preference)
  local ai_label=$(_aw_ai_preference_label "$ai_pref")
  local issue_autoselect=$(_aw_get_issue_autoselect)
  local pr_autoselect=$(_aw_get_pr_autoselect)

  gum style --border rounded --padding "0 1" --border-foreground 4 \
    "Settings Summary" \
    "Issue provider: $provider_label" \
    "JIRA server: ${jira_server:-(unset)}" \
    "JIRA project: ${jira_project:-(unset)}" \
    "GitLab server: ${gitlab_server:-(unset)}" \
    "GitLab project: ${gitlab_project:-(unset)}" \
    "Linear team: ${linear_team:-(unset)}" \
    "AI tool preference: $ai_label" \
    "Issue auto-select: $(_aw_bool_label "$issue_autoselect")" \
    "PR auto-select: $(_aw_bool_label "$pr_autoselect")"
}

_aw_show_settings_warnings() {
  local provider=$(_aw_get_issue_provider)
  local warnings=()

  if [[ -z "$provider" ]]; then
    warnings+=("Issue provider not configured for this repository.")
  fi

  if [[ -d ".github/ISSUE_TEMPLATE" ]] || [[ -f ".github/ISSUE_TEMPLATE.md" ]]; then
    if [[ "$provider" != "github" ]]; then
      warnings+=("GitHub issue templates detected, but issue provider is not set to GitHub.")
    fi
  fi

  if [[ -n "$provider" ]]; then
    case "$provider" in
      github)
        if ! command -v gh &> /dev/null; then
          warnings+=("GitHub CLI (gh) not found. GitHub issue workflow will fail.")
        fi
        ;;
      gitlab)
        if ! command -v glab &> /dev/null; then
          warnings+=("GitLab CLI (glab) not found. GitLab issue workflow will fail.")
        fi
        ;;
      jira)
        if ! command -v jira &> /dev/null; then
          warnings+=("JIRA CLI (jira) not found. JIRA issue workflow will fail.")
        fi
        ;;
      linear)
        if ! command -v linear &> /dev/null; then
          warnings+=("Linear CLI (linear) not found. Linear issue workflow will fail.")
        fi
        ;;
    esac
  fi

  local ai_pref=$(_load_ai_preference)
  if [[ -n "$ai_pref" ]] && [[ "$ai_pref" != "skip" ]]; then
    case "$ai_pref" in
      claude)
        command -v claude &> /dev/null || warnings+=("AI preference set to Claude Code, but it is not installed.")
        ;;
      codex)
        command -v codex &> /dev/null || warnings+=("AI preference set to Codex CLI, but it is not installed.")
        ;;
      gemini)
        command -v gemini &> /dev/null || warnings+=("AI preference set to Gemini CLI, but it is not installed.")
        ;;
      jules)
        command -v jules &> /dev/null || warnings+=("AI preference set to Google Jules CLI, but it is not installed.")
        ;;
    esac
  fi

  if [[ ${#warnings[@]} -gt 0 ]]; then
    echo ""
    gum style --border rounded --padding "0 1" --border-foreground 3 \
      "Warnings / Suggestions" \
      "${warnings[@]}"
  fi
}

_aw_settings_issue_provider() {
  while true; do
    echo ""
    _aw_show_settings_summary

    local choice=$(gum choose \
      "Set issue provider" \
      "Configure JIRA" \
      "Configure GitLab" \
      "Configure Linear" \
      "Clear issue provider settings" \
      "Back")

    case "$choice" in
      "Set issue provider")
        local provider_choice=$(gum choose --header "Select issue provider" \
          "GitHub Issues" \
          "GitLab Issues" \
          "JIRA" \
          "Linear Issues" \
          "Unset" \
          "Back")

        case "$provider_choice" in
          "GitHub Issues") _aw_set_issue_provider "github" ;;
          "GitLab Issues") _aw_set_issue_provider "gitlab" ;;
          "JIRA") _aw_set_issue_provider "jira" ;;
          "Linear Issues") _aw_set_issue_provider "linear" ;;
          "Unset")
            git config --unset auto-worktree.issue-provider 2>/dev/null
            gum style --foreground 2 "✓ Issue provider unset"
            ;;
          *) ;;
        esac
        ;;
      "Configure JIRA") _aw_configure_jira ;;
      "Configure GitLab") _aw_configure_gitlab ;;
      "Configure Linear") _aw_configure_linear ;;
      "Clear issue provider settings") _aw_clear_issue_provider_settings ;;
      *) return 0 ;;
    esac
  done
}

_aw_settings_ai_tool() {
  while true; do
    local current_pref=$(_load_ai_preference)
    local current_label=$(_aw_ai_preference_label "$current_pref")

    local choice=$(gum choose --header "AI tool preference (current: $current_label)" \
      "Auto (prompt when needed)" \
      "Claude Code" \
      "Codex CLI" \
      "Gemini CLI" \
      "Google Jules CLI" \
      "Skip AI tool" \
      "Back")

    case "$choice" in
      "Auto (prompt when needed)")
        _save_ai_preference ""
        gum style --foreground 2 "✓ AI tool preference reset to auto"
        ;;
      "Claude Code")
        _save_ai_preference "claude"
        gum style --foreground 2 "✓ AI tool preference set to Claude Code"
        ;;
      "Codex CLI")
        _save_ai_preference "codex"
        gum style --foreground 2 "✓ AI tool preference set to Codex CLI"
        ;;
      "Gemini CLI")
        _save_ai_preference "gemini"
        gum style --foreground 2 "✓ AI tool preference set to Gemini CLI"
        ;;
      "Google Jules CLI")
        _save_ai_preference "jules"
        gum style --foreground 2 "✓ AI tool preference set to Google Jules CLI"
        ;;
      "Skip AI tool")
        _save_ai_preference "skip"
        gum style --foreground 2 "✓ AI tool preference set to skip"
        ;;
      *) return 0 ;;
    esac
  done
}

_aw_settings_autoselect() {
  while true; do
    local issue_autoselect=$(_aw_get_issue_autoselect)
    local pr_autoselect=$(_aw_get_pr_autoselect)
    local issue_label=$(_aw_bool_label "$issue_autoselect")
    local pr_label=$(_aw_bool_label "$pr_autoselect")

    local choice=$(gum choose --header "Auto-select settings (issues: $issue_label, PRs: $pr_label)" \
      "Toggle issue auto-select" \
      "Toggle PR auto-select" \
      "Reset auto-select to defaults" \
      "Back")

    case "$choice" in
      "Toggle issue auto-select")
        if [[ "$issue_autoselect" == "true" ]]; then
          _disable_autoselect
          gum style --foreground 2 "✓ Issue auto-select disabled"
        else
          _enable_autoselect
          gum style --foreground 2 "✓ Issue auto-select enabled"
        fi
        ;;
      "Toggle PR auto-select")
        if [[ "$pr_autoselect" == "true" ]]; then
          _disable_pr_autoselect
          gum style --foreground 2 "✓ PR auto-select disabled"
        else
          _enable_pr_autoselect
          gum style --foreground 2 "✓ PR auto-select enabled"
        fi
        ;;
      "Reset auto-select to defaults")
        git config --unset auto-worktree.issue-autoselect 2>/dev/null
        git config --unset auto-worktree.pr-autoselect 2>/dev/null
        gum style --foreground 2 "✓ Auto-select settings reset to defaults"
        ;;
      *) return 0 ;;
    esac
  done
}

_aw_settings_reset() {
  if ! gum confirm "Reset all auto-worktree settings for this repository?"; then
    gum style --foreground 3 "Cancelled"
    return 0
  fi

  _aw_clear_issue_provider_settings
  git config --unset auto-worktree.ai-tool 2>/dev/null
  git config --unset auto-worktree.issue-autoselect 2>/dev/null
  git config --unset auto-worktree.pr-autoselect 2>/dev/null
  gum style --foreground 2 "✓ All settings reset"
}

_aw_settings_menu() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  while true; do
    echo ""
    _aw_show_settings_summary
    _aw_show_settings_warnings

    local choice=$(gum choose \
      "Issue provider settings" \
      "AI tool preference" \
      "Auto-select settings" \
      "Reset settings" \
      "Back")

    case "$choice" in
      "Issue provider settings") _aw_settings_issue_provider ;;
      "AI tool preference") _aw_settings_ai_tool ;;
      "Auto-select settings") _aw_settings_autoselect ;;
      "Reset settings") _aw_settings_reset ;;
      *) return 0 ;;
    esac
  done
}

_aw_prompt_issue_provider() {
  # Prompt user to choose issue provider if not configured
  echo ""
  gum style --foreground 6 "Issue provider not configured for this repository"
  echo ""

  local choice=$(gum choose \
    "GitHub Issues" \
    "GitLab Issues" \
    "JIRA" \
    "Linear Issues" \
    "Cancel")

  case "$choice" in
    "GitHub Issues")
      _aw_set_issue_provider "github"
      ;;
    "GitLab Issues")
      _aw_set_issue_provider "gitlab"
      _aw_configure_gitlab
      ;;
    "JIRA")
      _aw_set_issue_provider "jira"
      _aw_configure_jira
      ;;
    "Linear Issues")
      _aw_set_issue_provider "linear"
      _aw_configure_linear
      ;;
    *)
      gum style --foreground 3 "Cancelled"
      return 1
      ;;
  esac
}

_aw_find_hook_paths() {
  # Find all possible hook directories
  # Returns paths separated by newlines
  local worktree_path="$1"
  local hook_paths=()

  # 1. Check custom git config core.hooksPath
  local custom_hooks_path=$(git -C "$worktree_path" config core.hooksPath 2>/dev/null)
  if [[ -n "$custom_hooks_path" ]]; then
    # Handle both absolute and relative paths
    if [[ "$custom_hooks_path" == /* ]]; then
      hook_paths+=("$custom_hooks_path")
    else
      hook_paths+=("$worktree_path/$custom_hooks_path")
    fi
  fi

  # 2. Check .husky directory (popular Node.js hook manager)
  if [[ -d "$worktree_path/.husky" ]]; then
    hook_paths+=("$worktree_path/.husky")
  fi

  # 3. Standard .git/hooks directory
  # For worktrees, use --git-common-dir to get the shared hooks directory
  local git_common_dir=$(git -C "$worktree_path" rev-parse --git-common-dir 2>/dev/null)
  if [[ -n "$git_common_dir" && -d "$git_common_dir/hooks" ]]; then
    hook_paths+=("$git_common_dir/hooks")
  fi

  # Print paths (one per line)
  for path in "${hook_paths[@]}"; do
    echo "$path"
  done
}

_aw_execute_hook() {
  # Execute a single git hook if it exists and is executable
  # Returns 0 on success, 1 on failure, 2 if hook doesn't exist
  local hook_path="$1"
  local worktree_path="$2"
  local hook_name=$(basename "$hook_path")

  if [[ ! -f "$hook_path" ]]; then
    return 2  # Hook doesn't exist
  fi

  if [[ ! -x "$hook_path" ]]; then
    return 2  # Hook not executable
  fi

  # Display hook execution
  echo ""
  gum style --foreground 6 "Running git hook: $hook_name"

  # Execute hook in worktree context
  # Pass standard git hook parameters for post-checkout: <prev-head> <new-head> <branch-flag>
  # For worktree creation, we use: 0000000000000000000000000000000000000000 HEAD 1
  local prev_head="0000000000000000000000000000000000000000"
  local new_head=$(git -C "$worktree_path" rev-parse HEAD 2>/dev/null || echo "HEAD")
  local branch_flag="1"  # 1 = branch checkout, 0 = file checkout

  # Set up PATH for hook execution
  # Git hooks run with minimal environment, so we need to ensure they have access to:
  # 1. User's current PATH (includes user-installed tools like gum, homebrew packages, etc.)
  # 2. Standard system directories (fallback for basic commands)
  # 3. Common package manager directories (Homebrew on macOS, etc.)
  local hook_path_env="$PATH"

  # Add common directories if not already in PATH (for robustness)
  local additional_paths="/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
  hook_path_env="$hook_path_env:$additional_paths"

  # Run hook with output displayed directly to user
  if (cd "$worktree_path" && PATH="$hook_path_env" "$hook_path" "$prev_head" "$new_head" "$branch_flag"); then
    gum style --foreground 2 "✓ Hook $hook_name completed successfully"
    return 0
  else
    return 1
  fi
}

_aw_run_git_hooks() {
  # Run git hooks during worktree setup
  # Executes hooks in order: post-checkout, post-clone, post-worktree, custom hooks
  local worktree_path="$1"

  # Check if hook execution is enabled (default: true)
  local run_hooks=$(git -C "$worktree_path" config --bool auto-worktree.run-hooks 2>/dev/null)
  if [[ "$run_hooks" == "false" ]]; then
    return 0
  fi

  # Get failure handling preference (default: false = warn only)
  local fail_on_error=$(git -C "$worktree_path" config --bool auto-worktree.fail-on-hook-error 2>/dev/null)
  if [[ -z "$fail_on_error" ]]; then
    fail_on_error="false"
  fi

  # Find all hook directories
  local hook_paths=()
  while IFS= read -r hook_dir_path; do
    hook_paths+=("$hook_dir_path")
  done < <(_aw_find_hook_paths "$worktree_path")

  if [[ ${#hook_paths[@]} -eq 0 ]]; then
    # No hook directories found, skip silently
    return 0
  fi

  # Define hooks to run in order
  # Note: post-checkout is already run by git automatically during worktree creation
  local hooks_to_run=("post-clone" "post-worktree")

  # Check for custom hooks config
  local custom_hooks=$(git -C "$worktree_path" config auto-worktree.custom-hooks 2>/dev/null)
  if [[ -n "$custom_hooks" ]]; then
    # Add custom hooks (space or comma separated)
    IFS=', ' read -ra custom_array <<< "$custom_hooks"
    hooks_to_run+=("${custom_array[@]}")
  fi

  local any_hook_ran=false
  local any_hook_failed=false
  local failed_hooks=()

  # Execute each hook in order
  for hook_name in "${hooks_to_run[@]}"; do
    local hook_found=false

    # Try to find and execute the hook in each hook directory
    for hook_dir in "${hook_paths[@]}"; do
      local hook_path="$hook_dir/$hook_name"

      _aw_execute_hook "$hook_path" "$worktree_path"
      local result=$?

      if [[ $result -eq 0 ]]; then
        # Hook succeeded
        hook_found=true
        any_hook_ran=true
        break  # Don't run same hook from other directories
      elif [[ $result -eq 1 ]]; then
        # Hook failed
        hook_found=true
        any_hook_ran=true
        any_hook_failed=true
        failed_hooks+=("$hook_name")

        # Display error with config hint
        echo ""
        gum style --foreground 1 "✗ Hook $hook_name failed"
        if [[ "$fail_on_error" == "true" ]]; then
          gum style --foreground 3 "To continue despite hook failures, run:"
          gum style --foreground 7 "  git config auto-worktree.fail-on-hook-error false"
          return 1
        else
          gum style --foreground 3 "⚠ Continuing despite hook failure (auto-worktree.fail-on-hook-error=false)"
          gum style --foreground 7 "  To fail on hook errors, run: git config auto-worktree.fail-on-hook-error true"
        fi
        break  # Don't try other directories for this hook
      fi
      # result == 2 means hook doesn't exist, continue to next directory
    done
  done

  if [[ "$any_hook_ran" == "true" ]]; then
    echo ""
  fi

  return 0
}

_aw_setup_environment() {
  # Automatically set up the development environment based on detected project files
  local worktree_path="$1"

  if [[ ! -d "$worktree_path" ]]; then
    return 0
  fi

  # Run git hooks before dependency installation
  _aw_run_git_hooks "$worktree_path"
  local hook_result=$?
  if [[ $hook_result -ne 0 ]]; then
    # Hook failed and fail-on-hook-error is true
    return 1
  fi

  local setup_ran=false

  # Node.js project
  if [[ -f "$worktree_path/package.json" ]]; then
    setup_ran=true
    echo ""
    gum style --foreground 6 "Detected Node.js project (package.json)"

    # Detect which package manager to use
    local pkg_manager="npm"
    local install_cmd=""

    # Check packageManager field in package.json
    if command -v jq &> /dev/null; then
      local pkg_mgr_field=$(jq -r '.packageManager // ""' "$worktree_path/package.json" 2>/dev/null)
      if [[ "$pkg_mgr_field" == bun* ]]; then
        pkg_manager="bun"
      elif [[ "$pkg_mgr_field" == pnpm* ]]; then
        pkg_manager="pnpm"
      elif [[ "$pkg_mgr_field" == yarn* ]]; then
        pkg_manager="yarn"
      fi
    fi

    # Check for lock files if packageManager field not found
    if [[ "$pkg_manager" == "npm" ]]; then
      if [[ -f "$worktree_path/bun.lockb" ]]; then
        pkg_manager="bun"
      elif [[ -f "$worktree_path/pnpm-lock.yaml" ]]; then
        pkg_manager="pnpm"
      elif [[ -f "$worktree_path/yarn.lock" ]]; then
        pkg_manager="yarn"
      fi
    fi

    # Run the appropriate package manager
    case "$pkg_manager" in
      bun)
        if command -v bun &> /dev/null; then
          if gum spin --spinner dot --title "Running bun install..." -- bun install --cwd "$worktree_path"; then
            gum style --foreground 2 "✓ Dependencies installed (bun)"
          else
            gum style --foreground 3 "⚠ bun install had issues (continuing anyway)"
          fi
        else
          gum style --foreground 3 "⚠ bun not found, skipping dependency installation"
        fi
        ;;
      pnpm)
        if command -v pnpm &> /dev/null; then
          if gum spin --spinner dot --title "Running pnpm install..." -- pnpm install --dir "$worktree_path" --silent; then
            gum style --foreground 2 "✓ Dependencies installed (pnpm)"
          else
            gum style --foreground 3 "⚠ pnpm install had issues (continuing anyway)"
          fi
        else
          gum style --foreground 3 "⚠ pnpm not found, skipping dependency installation"
        fi
        ;;
      yarn)
        if command -v yarn &> /dev/null; then
          if gum spin --spinner dot --title "Running yarn install..." -- sh -c "cd '$worktree_path' && yarn install --silent"; then
            gum style --foreground 2 "✓ Dependencies installed (yarn)"
          else
            gum style --foreground 3 "⚠ yarn install had issues (continuing anyway)"
          fi
        else
          gum style --foreground 3 "⚠ yarn not found, skipping dependency installation"
        fi
        ;;
      *)
        if command -v npm &> /dev/null; then
          if gum spin --spinner dot --title "Running npm install..." -- npm --prefix "$worktree_path" install --silent; then
            gum style --foreground 2 "✓ Dependencies installed (npm)"
          else
            gum style --foreground 3 "⚠ npm install had issues (continuing anyway)"
          fi
        else
          gum style --foreground 3 "⚠ npm not found, skipping dependency installation"
        fi
        ;;
    esac
  fi

  # Python project
  if [[ -f "$worktree_path/requirements.txt" ]] || [[ -f "$worktree_path/pyproject.toml" ]]; then
    setup_ran=true
    echo ""

    # Check if uv is available and configured
    local use_uv=false
    if command -v uv &> /dev/null; then
      # Check for uv.lock or [tool.uv] in pyproject.toml
      if [[ -f "$worktree_path/uv.lock" ]]; then
        use_uv=true
      elif [[ -f "$worktree_path/pyproject.toml" ]] && grep -q '\[tool\.uv\]' "$worktree_path/pyproject.toml" 2>/dev/null; then
        use_uv=true
      fi
    fi

    if [[ "$use_uv" == "true" ]]; then
      gum style --foreground 6 "Detected Python project (uv)"
      if gum spin --spinner dot --title "Running uv sync..." -- sh -c "cd '$worktree_path' && uv sync"; then
        gum style --foreground 2 "✓ Dependencies installed (uv + .venv)"
      else
        gum style --foreground 3 "⚠ uv sync had issues (continuing anyway)"
      fi
    elif [[ -f "$worktree_path/pyproject.toml" ]]; then
      gum style --foreground 6 "Detected Python project (pyproject.toml)"

      if command -v poetry &> /dev/null && [[ -f "$worktree_path/poetry.lock" ]]; then
        # Use poetry if poetry.lock exists
        if gum spin --spinner dot --title "Running poetry install..." -- poetry -C "$worktree_path" install --quiet; then
          gum style --foreground 2 "✓ Dependencies installed (poetry)"
        else
          gum style --foreground 3 "⚠ poetry install had issues (continuing anyway)"
        fi
      elif command -v pip &> /dev/null || command -v pip3 &> /dev/null; then
        # Fall back to pip
        local pip_cmd=$(command -v pip3 &> /dev/null && echo "pip3" || echo "pip")
        if gum spin --spinner dot --title "Installing Python dependencies..." -- $pip_cmd install -q -e "$worktree_path"; then
          gum style --foreground 2 "✓ Dependencies installed (pip)"
        else
          gum style --foreground 3 "⚠ pip install had issues (continuing anyway)"
        fi
      else
        gum style --foreground 3 "⚠ No Python package manager found"
      fi
    elif [[ -f "$worktree_path/requirements.txt" ]]; then
      gum style --foreground 6 "Detected Python project (requirements.txt)"

      if command -v pip &> /dev/null || command -v pip3 &> /dev/null; then
        local pip_cmd=$(command -v pip3 &> /dev/null && echo "pip3" || echo "pip")
        if gum spin --spinner dot --title "Installing Python dependencies..." -- $pip_cmd install -q -r "$worktree_path/requirements.txt"; then
          gum style --foreground 2 "✓ Dependencies installed (pip)"
        else
          gum style --foreground 3 "⚠ pip install had issues (continuing anyway)"
        fi
      else
        gum style --foreground 3 "⚠ pip not found, skipping dependency installation"
      fi
    fi
  fi

  # Ruby project
  if [[ -f "$worktree_path/Gemfile" ]]; then
    setup_ran=true
    echo ""
    gum style --foreground 6 "Detected Ruby project (Gemfile)"

    if command -v bundle &> /dev/null; then
      if gum spin --spinner dot --title "Running bundle install..." -- bundle install --gemfile="$worktree_path/Gemfile" --quiet; then
        gum style --foreground 2 "✓ Dependencies installed"
      else
        gum style --foreground 3 "⚠ bundle install had issues (continuing anyway)"
      fi
    else
      gum style --foreground 3 "⚠ bundle not found, skipping dependency installation"
    fi
  fi

  # Go project
  if [[ -f "$worktree_path/go.mod" ]]; then
    setup_ran=true
    echo ""
    gum style --foreground 6 "Detected Go project (go.mod)"

    if command -v go &> /dev/null; then
      if gum spin --spinner dot --title "Running go mod download..." -- sh -c "cd '$worktree_path' && go mod download"; then
        gum style --foreground 2 "✓ Dependencies downloaded"
      else
        gum style --foreground 3 "⚠ go mod download had issues (continuing anyway)"
      fi
    else
      gum style --foreground 3 "⚠ go not found, skipping dependency installation"
    fi
  fi

  # Rust project
  if [[ -f "$worktree_path/Cargo.toml" ]]; then
    setup_ran=true
    echo ""
    gum style --foreground 6 "Detected Rust project (Cargo.toml)"

    if command -v cargo &> /dev/null; then
      if gum spin --spinner dot --title "Running cargo fetch..." -- sh -c "cd '$worktree_path' && cargo fetch --quiet"; then
        gum style --foreground 2 "✓ Dependencies fetched"
      else
        gum style --foreground 3 "⚠ cargo fetch had issues (continuing anyway)"
      fi
    else
      gum style --foreground 3 "⚠ cargo not found, skipping dependency installation"
    fi
  fi

  if [[ "$setup_ran" == "true" ]]; then
    echo ""
  fi

  return 0
}

_aw_extract_issue_number() {
  # Extract issue number from branch name patterns like:
  # work/123-description, issue-123, 123-fix-something
  local branch="$1"
  echo "$branch" | grep -oE '(^|[^0-9])([0-9]+)' | head -1 | grep -oE '[0-9]+' | head -1
}

_aw_extract_jira_key() {
  # Extract JIRA key from branch name patterns like:
  # work/PROJ-123-description, PROJ-456-fix-something
  # JIRA keys are typically PROJECT-NUMBER format
  local branch="$1"
  echo "$branch" | grep -oE '[A-Z][A-Z0-9]+-[0-9]+' | head -1
}

_aw_extract_linear_key() {
  # Extract Linear key from branch name patterns like:
  # work/TEAM-123-description, TEAM-456-fix-something
  # Linear keys are typically TEAM-NUMBER format (similar to JIRA)
  local branch="$1"
  echo "$branch" | grep -oE '[A-Z][A-Z0-9]+-[0-9]+' | head -1
}

_aw_extract_issue_id() {
  # Extract either GitHub/GitLab issue number, JIRA key, or Linear key from branch name
  # Returns the ID and sets _AW_DETECTED_ISSUE_TYPE to "github", "gitlab", "jira", or "linear"
  local branch="$1"

  # Check configured provider first to disambiguate JIRA vs Linear
  # Both use the same pattern: TEAM-123
  local provider=$(_aw_get_issue_provider)

  # Try JIRA/Linear key first (more specific pattern)
  local key=$(_aw_extract_jira_key "$branch")
  if [[ -n "$key" ]]; then
    if [[ "$provider" == "linear" ]]; then
      _AW_DETECTED_ISSUE_TYPE="linear"
    else
      # Default to jira if pattern matches (for backwards compatibility)
      _AW_DETECTED_ISSUE_TYPE="jira"
    fi
    echo "$key"
    return 0
  fi

  # Try GitHub/GitLab issue number
  # Both use numeric IDs, so we rely on configured provider to distinguish
  local issue_num=$(_aw_extract_issue_number "$branch")
  if [[ -n "$issue_num" ]]; then
    # Check configured provider to determine type
    if [[ "$provider" == "gitlab" ]]; then
      _AW_DETECTED_ISSUE_TYPE="gitlab"
    else
      _AW_DETECTED_ISSUE_TYPE="github"
    fi
    echo "$issue_num"
    return 0
  fi

  _AW_DETECTED_ISSUE_TYPE=""
  return 1
}

_aw_check_issue_merged() {
  # Check if an issue or its linked PR was merged into main
  # Returns 0 if merged, 1 if not merged or error
  local issue_num="$1"

  if [[ -z "$issue_num" ]]; then
    return 1
  fi

  # First check if issue is closed
  local issue_state=$(gh issue view "$issue_num" --json state --jq '.state' 2>/dev/null)

  if [[ "$issue_state" != "CLOSED" ]]; then
    return 1
  fi

  # Check if there's a linked PR that was merged
  # GitHub's stateReason can tell us if it was completed (often means PR merged)
  local state_reason=$(gh issue view "$issue_num" --json stateReason --jq '.stateReason' 2>/dev/null)

  if [[ "$state_reason" == "COMPLETED" ]]; then
    return 0
  fi

  # Also check for PRs that reference this issue and are merged
  local merged_prs=$(gh pr list --state merged --search "closes #$issue_num OR fixes #$issue_num OR resolves #$issue_num" --json number --jq 'length' 2>/dev/null)

  if [[ "$merged_prs" -gt 0 ]] 2>/dev/null; then
    return 0
  fi

  return 1
}

_aw_check_issue_closed() {
  # Check if an issue is closed (regardless of merge/PR status)
  # Returns 0 if closed, 1 if open or error
  # Sets _AW_ISSUE_HAS_PR=true if there's an open PR for this issue
  local issue_num="$1"

  if [[ -z "$issue_num" ]]; then
    return 1
  fi

  # Check if issue is closed
  local issue_state=$(gh issue view "$issue_num" --json state --jq '.state' 2>/dev/null)

  if [[ "$issue_state" != "CLOSED" ]]; then
    return 1
  fi

  # Check if there's an open PR that references this issue
  local open_prs=$(gh pr list --state open --search "closes #$issue_num OR fixes #$issue_num OR resolves #$issue_num" --json number --jq 'length' 2>/dev/null)

  if [[ "$open_prs" -gt 0 ]] 2>/dev/null; then
    _AW_ISSUE_HAS_PR=true
  else
    _AW_ISSUE_HAS_PR=false
  fi

  return 0
}

_aw_check_branch_pr_merged() {
  # Check if the branch itself has a merged PR (regardless of issue linkage)
  # Returns 0 if merged, 1 if not
  local branch_name="$1"

  if [[ -z "$branch_name" ]]; then
    return 1
  fi

  # Check if there's a merged PR for this branch
  local pr_state=$(gh pr view "$branch_name" --json state,mergedAt --jq '.state' 2>/dev/null)

  if [[ "$pr_state" == "MERGED" ]]; then
    return 0
  fi

  return 1
}

_aw_get_default_branch() {
  # Detect the default branch (main or master)
  # Returns the branch name or empty string if not found

  # First try to get from remote
  local default_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')

  if [[ -n "$default_branch" ]]; then
    echo "$default_branch"
    return 0
  fi

  # Fallback: check if main or master exists locally
  if git show-ref --verify --quiet refs/heads/main 2>/dev/null; then
    echo "main"
    return 0
  elif git show-ref --verify --quiet refs/heads/master 2>/dev/null; then
    echo "master"
    return 0
  fi

  # Last resort: try to get from remote branches
  if git show-ref --verify --quiet refs/remotes/origin/main 2>/dev/null; then
    echo "main"
    return 0
  elif git show-ref --verify --quiet refs/remotes/origin/master 2>/dev/null; then
    echo "master"
    return 0
  fi

  return 1
}

_aw_check_no_changes_from_default() {
  # Check if a worktree has no changes from the default branch HEAD
  # Returns 0 if no changes, 1 otherwise
  # Sets _AW_DEFAULT_BRANCH_NAME global variable
  local wt_path="$1"

  if [[ -z "$wt_path" ]] || [[ ! -d "$wt_path" ]]; then
    return 1
  fi

  # Get default branch name
  _AW_DEFAULT_BRANCH_NAME=$(_aw_get_default_branch)

  if [[ -z "$_AW_DEFAULT_BRANCH_NAME" ]]; then
    return 1
  fi

  # Get the current branch of the worktree
  local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null)

  # Don't check if this IS the default branch
  if [[ "$wt_branch" == "$_AW_DEFAULT_BRANCH_NAME" ]]; then
    return 1
  fi

  # Get the commit hash of the worktree HEAD
  local wt_head=$(git -C "$wt_path" rev-parse HEAD 2>/dev/null)

  # Get the commit hash of the default branch HEAD
  local default_head=$(git rev-parse "$_AW_DEFAULT_BRANCH_NAME" 2>/dev/null)

  if [[ -z "$wt_head" ]] || [[ -z "$default_head" ]]; then
    return 1
  fi

  # Check if they're the same
  if [[ "$wt_head" == "$default_head" ]]; then
    return 0
  fi

  return 1
}

# ============================================================================
# JIRA integration functions
# ============================================================================

_aw_jira_check_resolved() {
  # Check if a JIRA issue is resolved/done/closed
  # Returns 0 if resolved, 1 if not resolved or error
  local jira_key="$1"

  if [[ -z "$jira_key" ]]; then
    return 1
  fi

  # Get issue status using JIRA CLI
  local status=$(jira issue view "$jira_key" --plain --columns status 2>/dev/null | tail -1 | awk '{print $NF}')

  if [[ -z "$status" ]]; then
    return 1
  fi

  # Common resolved status names in JIRA
  # Note: Status names can vary by JIRA configuration, but these are common
  case "$status" in
    Done|Closed|Resolved|Complete|Completed)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

_aw_jira_list_issues() {
  # List JIRA issues using JQL
  # Returns formatted issue list similar to GitHub issues
  local project=$(_aw_get_jira_project)
  local jql="status != Done AND status != Closed AND status != Resolved"

  # If a default project is configured, filter by it
  if [[ -n "$project" ]]; then
    jql="project = $project AND ($jql)"
  fi

  # Use JIRA CLI to list issues
  # Output format: KEY | Summary | [Labels]
  jira issue list --jql "$jql" --plain --columns key,summary,labels --no-headers 2>/dev/null | \
    awk -F'\t' '{
      key = $1
      summary = $2
      labels = $3

      # Format similar to GitHub issue list
      printf "%s | %s", key, summary
      if (labels != "" && labels != "∅") {
        # Split labels and format them
        gsub(/,/, "][", labels)
        printf " | [%s]", labels
      }
      printf "\n"
    }'
}

_aw_jira_get_issue_details() {
  # Get JIRA issue details
  # Sets variables: title, body (description)
  local jira_key="$1"

  if [[ -z "$jira_key" ]]; then
    return 1
  fi

  # Get issue details in JSON format
  local issue_json=$(jira issue view "$jira_key" --plain --columns summary,description 2>/dev/null)

  if [[ -z "$issue_json" ]]; then
    return 1
  fi

  # Extract summary (title) and description (body)
  # The plain format outputs tab-separated values
  title=$(echo "$issue_json" | grep -A1 "Summary" | tail -1 | sed 's/^[[:space:]]*//')
  body=$(echo "$issue_json" | grep -A1 "Description" | tail -1 | sed 's/^[[:space:]]*//')

  # If description is empty or just "∅", set to empty string
  if [[ "$body" == "∅" ]] || [[ "$body" == "" ]]; then
    body=""
  fi

  return 0
}

# ============================================================================
# Linear integration functions
# ============================================================================

_aw_linear_check_completed() {
  # Check if a Linear issue is completed/done/canceled
  # Returns 0 if completed, 1 if not completed or error
  local issue_id="$1"

  if [[ -z "$issue_id" ]]; then
    return 1
  fi

  # Get issue details using Linear CLI
  # The 'linear issue view' command outputs markdown with issue details
  local issue_view=$(linear issue view "$issue_id" 2>/dev/null)

  if [[ -z "$issue_view" ]]; then
    return 1
  fi

  # Extract state from the output (looking for State: or Status: lines)
  local state=$(echo "$issue_view" | grep -i "State:" | sed 's/.*State:[[:space:]]*//i' | tr -d '\r\n')

  if [[ -z "$state" ]]; then
    # Try alternative format
    state=$(echo "$issue_view" | grep -i "Status:" | sed 's/.*Status:[[:space:]]*//i' | tr -d '\r\n')
  fi

  if [[ -z "$state" ]]; then
    return 1
  fi

  # Common completed status names in Linear
  case "$state" in
    Done|Completed|Canceled|Cancelled)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

_aw_linear_list_issues() {
  # List Linear issues
  # Returns formatted issue list similar to GitHub issues
  local team=$(_aw_get_linear_team)

  # List issues using Linear CLI
  # Default: lists unstarted issues assigned to you
  # Use -A to list all team's unstarted issues
  local linear_cmd="linear issue list"

  # If a team is configured, we'll use -A to get all team issues
  # Note: Linear CLI doesn't have direct team filtering in list command
  # but it respects the LINEAR_TEAM_ID config
  if [[ -n "$team" ]]; then
    linear_cmd="linear issue list -A"
  fi

  # Execute the command and parse output
  # Linear CLI outputs a table format, we need to parse it
  $linear_cmd 2>/dev/null | tail -n +2 | awk '{
    # Parse Linear CLI table output
    # Expected format: ID    Title    State    ...
    if (NF >= 3 && $1 ~ /^[A-Z]+-[0-9]+$/) {
      id = $1
      # Extract title (everything between ID and State columns)
      # This is a simplified parser - may need adjustment based on actual output
      title = ""
      for (i=2; i<NF; i++) {
        if (title != "") title = title " "
        title = title $i
      }

      # Format: TEAM-123 | Title
      printf "%s | %s\n", id, title
    }
  }'
}

_aw_linear_get_issue_details() {
  # Get Linear issue details
  # Sets variables: title, body (description)
  local issue_id="$1"

  if [[ -z "$issue_id" ]]; then
    return 1
  fi

  # Get issue details using Linear CLI
  local issue_view=$(linear issue view "$issue_id" 2>/dev/null)

  if [[ -z "$issue_view" ]]; then
    return 1
  fi

  # Extract title - Linear outputs markdown format
  # Title is typically in a heading or after "Title:" label
  title=$(linear issue title "$issue_id" 2>/dev/null)

  if [[ -z "$title" ]]; then
    # Fallback: parse from view output
    title=$(echo "$issue_view" | grep -i "^# " | head -1 | sed 's/^# //')
  fi

  # Extract description/body from the markdown output
  # The description is the content after the metadata section
  body=$(echo "$issue_view" | sed -n '/^## Description/,/^##/p' | grep -v "^##" | sed 's/^[[:space:]]*//')

  # If body is empty, try to get any content after the header
  if [[ -z "$body" ]]; then
    body=$(echo "$issue_view" | sed '1,/^---$/d' | sed '/^$/d' | head -20)
  fi

  return 0
}

# ============================================================================
# GitLab integration functions
# ============================================================================

_aw_gitlab_check_closed() {
  # Check if a GitLab issue or MR is closed/merged
  # Returns 0 if closed/merged, 1 if open or error
  local id="$1"
  local type="${2:-issue}"  # 'issue' or 'mr'

  if [[ -z "$id" ]]; then
    return 1
  fi

  # Build glab command with server option if configured
  local glab_cmd="glab"
  local server=$(_aw_get_gitlab_server)
  if [[ -n "$server" ]]; then
    glab_cmd="glab --host $server"
  fi

  # Get state using glab CLI
  local state
  if [[ "$type" == "mr" ]]; then
    state=$($glab_cmd mr view "$id" --json state --jq '.state' 2>/dev/null)
  else
    state=$($glab_cmd issue view "$id" --json state --jq '.state' 2>/dev/null)
  fi

  if [[ -z "$state" ]]; then
    return 1
  fi

  # GitLab states: "opened", "closed", "merged" (for MRs)
  case "$state" in
    closed|merged)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

_aw_gitlab_list_issues() {
  # List GitLab issues
  # Returns formatted issue list similar to GitHub issues
  local project=$(_aw_get_gitlab_project)

  # Build glab command with server option if configured
  local glab_cmd="glab"
  local server=$(_aw_get_gitlab_server)
  if [[ -n "$server" ]]; then
    glab_cmd="glab --host $server"
  fi

  # Add project filter if configured
  local project_args=""
  if [[ -n "$project" ]]; then
    project_args="--repo $project"
  fi

  # List open issues with glab
  $glab_cmd issue list --state opened --per-page 100 $project_args 2>/dev/null | \
    awk -F'\t' '{
      # glab output format: #NUMBER  TITLE  (LABELS)  (TIME)
      # Extract issue number, title, and labels
      if ($1 ~ /^#[0-9]+/) {
        number = $1
        title = $2
        labels = $3

        # Format: #123 | Title | [label1][label2]
        printf "%s | %s", number, title
        if (labels != "" && labels != "()") {
          # Clean up labels format
          gsub(/[()]/, "", labels)
          gsub(/, /, "][", labels)
          printf " | [%s]", labels
        }
        printf "\n"
      }
    }'
}

_aw_gitlab_get_issue_details() {
  # Get GitLab issue details
  # Sets variables: title, body (description)
  local issue_id="$1"

  if [[ -z "$issue_id" ]]; then
    return 1
  fi

  # Build glab command with server option if configured
  local glab_cmd="glab"
  local server=$(_aw_get_gitlab_server)
  if [[ -n "$server" ]]; then
    glab_cmd="glab --host $server"
  fi

  # Get issue details in JSON format
  local issue_json=$($glab_cmd issue view "$issue_id" --json title,description 2>/dev/null)

  if [[ -z "$issue_json" ]]; then
    return 1
  fi

  # Extract title and description using jq
  title=$(echo "$issue_json" | jq -r '.title // ""')
  body=$(echo "$issue_json" | jq -r '.description // ""')

  return 0
}

_aw_gitlab_list_mrs() {
  # List GitLab merge requests
  # Returns formatted MR list similar to GitHub PRs
  local project=$(_aw_get_gitlab_project)

  # Build glab command with server option if configured
  local glab_cmd="glab"
  local server=$(_aw_get_gitlab_server)
  if [[ -n "$server" ]]; then
    glab_cmd="glab --host $server"
  fi

  # Add project filter if configured
  local project_args=""
  if [[ -n "$project" ]]; then
    project_args="--repo $project"
  fi

  # List open MRs with glab
  $glab_cmd mr list --state opened --per-page 100 $project_args 2>/dev/null | \
    awk -F'\t' '{
      # glab output format: !NUMBER  TITLE  (BRANCH)  (TIME)
      # Extract MR number, title, and branch
      if ($1 ~ /^![0-9]+/) {
        number = $1
        title = $2
        branch = $3

        # Format: !123 | Title | (branch-name)
        printf "%s | %s", number, title
        if (branch != "") {
          printf " | %s", branch
        }
        printf "\n"
      }
    }'
}

_aw_gitlab_get_mr_details() {
  # Get GitLab MR details
  # Sets variables: title, body (description), source_branch, target_branch
  local mr_id="$1"

  if [[ -z "$mr_id" ]]; then
    return 1
  fi

  # Build glab command with server option if configured
  local glab_cmd="glab"
  local server=$(_aw_get_gitlab_server)
  if [[ -n "$server" ]]; then
    glab_cmd="glab --host $server"
  fi

  # Get MR details in JSON format
  local mr_json=$($glab_cmd mr view "$mr_id" --json title,description,sourceBranch,targetBranch 2>/dev/null)

  if [[ -z "$mr_json" ]]; then
    return 1
  fi

  # Extract details using jq
  title=$(echo "$mr_json" | jq -r '.title // ""')
  body=$(echo "$mr_json" | jq -r '.description // ""')
  source_branch=$(echo "$mr_json" | jq -r '.sourceBranch // ""')
  target_branch=$(echo "$mr_json" | jq -r '.targetBranch // ""')

  return 0
}

_aw_gitlab_check_mr_merged() {
  # Check if a GitLab MR is merged for a given branch
  # Returns 0 if merged, 1 if not merged or error
  local branch_name="$1"

  if [[ -z "$branch_name" ]]; then
    return 1
  fi

  # Build glab command with server option if configured
  local glab_cmd="glab"
  local server=$(_aw_get_gitlab_server)
  if [[ -n "$server" ]]; then
    glab_cmd="glab --host $server"
  fi

  # Check if there's a merged MR for this branch
  local mr_state=$($glab_cmd mr view "$branch_name" --json state 2>/dev/null | jq -r '.state')

  if [[ "$mr_state" == "merged" ]]; then
    return 0
  fi

  return 1
}

_aw_has_unpushed_commits() {
  # Check if a worktree has unpushed commits
  # Returns 0 if there are unpushed commits, 1 if not
  # Sets _AW_UNPUSHED_COUNT to the number of unpushed commits
  local wt_path="$1"

  if [[ -z "$wt_path" ]] || [[ ! -d "$wt_path" ]]; then
    return 1
  fi

  # Get the current branch
  local branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null)

  if [[ -z "$branch" ]] || [[ "$branch" == "HEAD" ]]; then
    # Detached HEAD state, no upstream to compare
    return 1
  fi

  # Get the upstream branch
  local upstream=$(git -C "$wt_path" rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null)

  if [[ -z "$upstream" ]]; then
    # No upstream configured - check if there are any commits at all
    local commit_count=$(git -C "$wt_path" rev-list --count HEAD 2>/dev/null)
    if [[ "$commit_count" -gt 0 ]] 2>/dev/null; then
      _AW_UNPUSHED_COUNT=$commit_count
      return 0
    else
      return 1
    fi
  fi

  # Count commits ahead of upstream
  local ahead=$(git -C "$wt_path" rev-list --count @{u}..HEAD 2>/dev/null)

  if [[ "$ahead" -gt 0 ]] 2>/dev/null; then
    _AW_UNPUSHED_COUNT=$ahead
    return 0
  fi

  return 1
}

_aw_create_worktree() {
  local branch_name="$1"
  local initial_context="${2:-}"
  local worktree_name=$(_aw_sanitize_branch_name "$branch_name")
  local worktree_path="$_AW_WORKTREE_BASE/$worktree_name"

  mkdir -p "$_AW_WORKTREE_BASE"

  # Check if branch already exists
  local branch_exists=false
  if git show-ref --verify --quiet "refs/heads/${branch_name}"; then
    branch_exists=true
    local existing_worktree=$(git worktree list --porcelain | grep -A2 "^worktree " | grep -B1 "branch refs/heads/${branch_name}$" | head -1 | sed 's/^worktree //')
    if [[ -n "$existing_worktree" ]]; then
      gum style --foreground 1 "Error: Branch '${branch_name}' already has a worktree at:"
      echo "  $existing_worktree"
      return 1
    fi
    gum style --foreground 3 "Branch '${branch_name}' exists, creating worktree for it..."
  fi

  local base_branch=$(git symbolic-ref --short HEAD 2>/dev/null || echo "main")

  echo ""
  gum style --border rounded --padding "0 1" --border-foreground 4 \
    "Creating worktree" \
    "  Path:   $worktree_path" \
    "  Branch: $branch_name" \
    $([[ "$branch_exists" == "false" ]] && echo "  Base:   $base_branch")

  local worktree_cmd_success=false
  if [[ "$branch_exists" == "true" ]]; then
    if gum spin --spinner dot --title "Creating worktree..." -- git worktree add "$worktree_path" "$branch_name"; then
      worktree_cmd_success=true
    fi
  else
    if gum spin --spinner dot --title "Creating worktree..." -- git worktree add -b "$branch_name" "$worktree_path" "$base_branch"; then
      worktree_cmd_success=true
    fi
  fi

  if [[ "$worktree_cmd_success" == "true" ]]; then
    # Set up the development environment
    _aw_setup_environment "$worktree_path"

    cd "$worktree_path" || return 1

    # Set terminal title to branch name
    printf '\033]0;%s\007' "$branch_name"

    _resolve_ai_command || return 1

    if [[ "${AI_CMD[1]}" != "skip" ]]; then
      gum style --foreground 2 "Starting $AI_CMD_NAME..."
      if [[ -n "$initial_context" ]]; then
        "${AI_CMD[@]}" "$initial_context"
      else
        "${AI_CMD[@]}"
      fi
    else
      gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
    fi
  else
    gum style --foreground 1 "Failed to create worktree"
    return 1
  fi
}

# ============================================================================
# List worktrees
# ============================================================================

_aw_list() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info
  _aw_prune_worktrees

  local worktree_list=$(git worktree list --porcelain 2>/dev/null | grep "^worktree " | sed 's/^worktree //')
  local worktree_count=$(echo "$worktree_list" | grep -c . 2>/dev/null || echo 0)

  if [[ $worktree_count -le 1 ]]; then
    gum style --foreground 8 "No additional worktrees for $_AW_SOURCE_FOLDER"
    return 0
  fi

  local now=$(date +%s)
  local one_day=$((24 * 60 * 60))
  local four_days=$((4 * 24 * 60 * 60))

  local oldest_wt_path=""
  local oldest_wt_branch=""
  local oldest_age=0

  # Track merged worktrees for cleanup prompt
  local -a merged_wt_paths=()
  local -a merged_wt_branches=()
  local -a merged_wt_issues=()

  local output=""

  while IFS= read -r wt_path; do
    [[ "$wt_path" == "$_AW_GIT_ROOT" ]] && continue
    [[ ! -d "$wt_path" ]] && continue

    local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    local commit_timestamp=$(git -C "$wt_path" log -1 --format=%ct 2>/dev/null)

    if [[ -z "$commit_timestamp" ]] || ! [[ "$commit_timestamp" =~ ^[0-9]+$ ]]; then
      commit_timestamp=$(find "$wt_path" -maxdepth 3 -type f -not -path '*/.git/*' -print0 2>/dev/null | while IFS= read -r -d '' file; do _aw_get_file_mtime "$file"; done | sort -rn | head -1)
    fi

    # Check if this worktree is linked to a merged/resolved issue or has a merged PR
    # Try to detect both GitHub issues and JIRA keys
    local issue_id=$(_aw_extract_issue_id "$wt_branch")
    local is_merged=false
    local merged_indicator=""
    local merge_reason=""

    # _AW_DETECTED_ISSUE_TYPE is set by _aw_extract_issue_id
    if [[ -n "$issue_id" ]]; then
      if [[ "$_AW_DETECTED_ISSUE_TYPE" == "jira" ]]; then
        # Check if JIRA issue is resolved
        if _aw_jira_check_resolved "$issue_id"; then
          is_merged=true
          merge_reason="JIRA $issue_id"
          merged_indicator=" $(gum style --foreground 5 "[resolved $issue_id]")"
        fi
      elif [[ "$_AW_DETECTED_ISSUE_TYPE" == "gitlab" ]]; then
        # Check if GitLab issue is closed
        if _aw_gitlab_check_closed "$issue_id" "issue"; then
          # Check for unpushed commits
          if _aw_has_unpushed_commits "$wt_path"; then
            # Has unpushed work - mark as closed but with warning
            is_merged=true
            merge_reason="issue #$issue_id closed (⚠ $_AW_UNPUSHED_COUNT unpushed)"
            merged_indicator=" $(gum style --foreground 3 "[closed #$issue_id ⚠]")"
          else
            # No unpushed work - safe to clean up
            is_merged=true
            merge_reason="issue #$issue_id closed"
            merged_indicator=" $(gum style --foreground 5 "[closed #$issue_id]")"
          fi
        fi
      elif [[ "$_AW_DETECTED_ISSUE_TYPE" == "linear" ]]; then
        # Check if Linear issue is completed
        if _aw_linear_check_completed "$issue_id"; then
          is_merged=true
          merge_reason="Linear $issue_id"
          merged_indicator=" $(gum style --foreground 5 "[completed $issue_id]")"
        fi
      elif [[ "$_AW_DETECTED_ISSUE_TYPE" == "github" ]]; then
        # Check if GitHub issue is merged
        if _aw_check_issue_merged "$issue_id"; then
          is_merged=true
          merge_reason="issue #$issue_id"
          merged_indicator=" $(gum style --foreground 5 "[merged #$issue_id]")"
        elif _aw_check_issue_closed "$issue_id"; then
          # Issue is closed but no PR (either open or merged)
          if [[ "$_AW_ISSUE_HAS_PR" == "false" ]]; then
            # Check for unpushed commits
            if _aw_has_unpushed_commits "$wt_path"; then
              # Has unpushed work - mark as closed but with warning
              is_merged=true
              merge_reason="issue #$issue_id closed (⚠ $_AW_UNPUSHED_COUNT unpushed)"
              merged_indicator=" $(gum style --foreground 3 "[closed #$issue_id ⚠]")"
            else
              # No unpushed work - safe to clean up
              is_merged=true
              merge_reason="issue #$issue_id closed"
              merged_indicator=" $(gum style --foreground 5 "[closed #$issue_id]")"
            fi
          fi
        fi
      fi
    fi

    # Also check for merged PRs/MRs if no issue was detected
    if [[ "$is_merged" == "false" ]]; then
      # Check for GitLab MRs (mr-{number} pattern in path)
      if [[ "$wt_path" =~ mr-([0-9]+) ]]; then
        local mr_num="${BASH_REMATCH[1]}"
        if _aw_gitlab_check_closed "$mr_num" "mr"; then
          is_merged=true
          merge_reason="MR"
          merged_indicator=" $(gum style --foreground 5 "[MR merged]")"
        fi
      # Check for GitHub PRs
      elif _aw_check_branch_pr_merged "$wt_branch"; then
        is_merged=true
        merge_reason="PR"
        merged_indicator=" $(gum style --foreground 5 "[PR merged]")"
      fi
    fi

    # Check for worktrees with no changes from default branch (only if not already flagged as merged/closed)
    if [[ "$is_merged" == "false" ]] && ! _aw_has_unpushed_commits "$wt_path" && _aw_check_no_changes_from_default "$wt_path"; then
      is_merged=true
      merge_reason="no changes from $_AW_DEFAULT_BRANCH_NAME"
      merged_indicator=" $(gum style --foreground 8 "[no changes]")"
    fi

    if [[ "$is_merged" == "true" ]]; then
      merged_wt_paths+=("$wt_path")
      merged_wt_branches+=("$wt_branch")
      merged_wt_issues+=("$merge_reason")
    fi

    if [[ -z "$commit_timestamp" ]] || ! [[ "$commit_timestamp" =~ ^[0-9]+$ ]]; then
      output+="  $(gum style --foreground 8 "$(basename "$wt_path")") ($wt_branch) [unknown]${merged_indicator}\n"
      continue
    fi

    local age=$((now - commit_timestamp))
    local age_days=$((age / one_day))
    local age_hours=$((age / 3600))

    # Build age string and color inline to avoid zsh variable assignment echo bug
    if [[ $age -lt $one_day ]]; then
      output+="  $(basename "$wt_path") ($wt_branch) $(gum style --foreground 2 "[${age_hours}h ago]")${merged_indicator}\n"
    elif [[ $age -lt $four_days ]]; then
      output+="  $(basename "$wt_path") ($wt_branch) $(gum style --foreground 3 "[${age_days}d ago]")${merged_indicator}\n"
    else
      output+="  $(basename "$wt_path") ($wt_branch) $(gum style --foreground 1 "[${age_days}d ago]")${merged_indicator}\n"
      # Only track as stale if not already marked as merged
      if [[ "$is_merged" == "false" ]] && [[ $age -gt $oldest_age ]]; then
        oldest_age=$age
        oldest_wt_path="$wt_path"
        oldest_wt_branch="$wt_branch"
      fi
    fi
  done <<< "$worktree_list"

  if [[ -n "$output" ]]; then
    gum style --border rounded --padding "0 1" --border-foreground 4 \
      "Worktrees for $_AW_SOURCE_FOLDER"
    echo -e "$output"
  fi

  # Collect all worktrees to clean up (merged + stale)
  local -a cleanup_wt_paths=()
  local -a cleanup_wt_branches=()
  local -a cleanup_wt_reasons=()

  # Add merged worktrees
  if [[ ${#merged_wt_paths} -gt 0 ]]; then
    local i=1
    while [[ $i -le ${#merged_wt_paths} ]]; do
      cleanup_wt_paths+=("${merged_wt_paths[$i]}")
      cleanup_wt_branches+=("${merged_wt_branches[$i]}")
      cleanup_wt_reasons+=("merged (${merged_wt_issues[$i]})")
      ((i++))
    done
  fi

  # Add stale worktree
  if [[ -n "$oldest_wt_path" ]]; then
    local days=$((oldest_age / one_day))
    cleanup_wt_paths+=("$oldest_wt_path")
    cleanup_wt_branches+=("$oldest_wt_branch")
    cleanup_wt_reasons+=("stale (${days}d old)")
  fi

  # Prompt for batch cleanup
  if [[ ${#cleanup_wt_paths} -gt 0 ]]; then
    echo ""
    gum style --foreground 5 "Worktrees that can be cleaned up:"
    echo ""

    local i=1
    while [[ $i -le ${#cleanup_wt_paths} ]]; do
      local c_path="${cleanup_wt_paths[$i]}"
      local c_branch="${cleanup_wt_branches[$i]}"
      local c_reason="${cleanup_wt_reasons[$i]}"
      echo "  • $(basename "$c_path") ($c_branch) - $c_reason"
      ((i++))
    done

    echo ""
    if gum confirm "Clean up all these worktrees and delete their branches?"; then
      local i=1
      while [[ $i -le ${#cleanup_wt_paths} ]]; do
        local c_path="${cleanup_wt_paths[$i]}"
        local c_branch="${cleanup_wt_branches[$i]}"

        echo ""
        gum spin --spinner dot --title "Removing $(basename "$c_path")..." -- git worktree remove --force "$c_path"
        gum style --foreground 2 "Worktree removed."

        if git show-ref --verify --quiet "refs/heads/${c_branch}"; then
          git branch -D "$c_branch" 2>/dev/null
          gum style --foreground 2 "Branch deleted."
        fi

        ((i++))
      done
    fi
  fi
}

# ============================================================================
# New worktree
# ============================================================================

_aw_new() {
  local skip_list="${1:-false}"

  _aw_ensure_git_repo || return 1
  _aw_get_repo_info
  _aw_prune_worktrees

  # Show existing worktrees (unless called from menu which already showed them)
  if [[ "$skip_list" == "false" ]]; then
    _aw_list
  fi

  echo ""

  local branch_input=$(gum input --placeholder "Branch name (leave blank for random)")

  local branch_name=""
  local worktree_name=""

  if [[ -z "$branch_input" ]]; then
    # Generate a unique random name
    local attempts=0
    local max_attempts=50
    while [[ $attempts -lt $max_attempts ]]; do
      worktree_name="$(_aw_generate_random_name)"
      branch_name="work/${worktree_name}"

      # Check if branch already exists
      if ! git show-ref --verify --quiet "refs/heads/${branch_name}" 2>/dev/null; then
        break  # Branch doesn't exist, we can use this name
      fi
      ((attempts++))
    done

    if [[ $attempts -ge $max_attempts ]]; then
      gum style --foreground 1 "Failed to generate unique branch name after $max_attempts attempts"
      gum style --foreground 3 "Try specifying a branch name manually"
      return 1
    fi

    gum style --foreground 6 "Generated: $branch_name"
  else
    branch_name="$branch_input"
  fi

  _aw_create_worktree "$branch_name"
}

# ============================================================================
# Issue template helpers
# ============================================================================

_aw_parse_template() {
  # Parse a markdown template file
  # Args: $1 = template file path
  # Outputs: Template content as-is (for now, just return the content)
  local template_file="$1"

  if [[ ! -f "$template_file" ]]; then
    gum style --foreground 1 "Error: Template file not found: $template_file"
    return 1
  fi

  cat "$template_file"
}

_aw_extract_template_sections() {
  # Extract section headers from a markdown template
  # Args: $1 = template file path
  # Returns: List of section headers (lines starting with # or ##)
  local template_file="$1"

  if [[ ! -f "$template_file" ]]; then
    return 1
  fi

  # Strip YAML frontmatter first, then extract sections
  sed '/^---$/,/^---$/d' "$template_file" | grep -E '^#{1,2} ' | sed 's/^#* //'
}

_aw_extract_section_content() {
  # Extract content for a specific section from template
  # Args: $1 = template file path, $2 = section name
  # Returns: Content between this section header and the next section header
  local template_file="$1"
  local section_name="$2"

  if [[ ! -f "$template_file" ]]; then
    return 1
  fi

  # Strip YAML frontmatter and extract content for this section
  local content=$(sed '/^---$/,/^---$/d' "$template_file" | \
    awk -v section="$section_name" '
      BEGIN { in_section=0; found=0 }
      /^#{1,2} / {
        if (in_section) {
          exit
        }
        section_header = $0
        gsub(/^#* /, "", section_header)
        if (section_header == section) {
          in_section=1
          found=1
          next
        }
      }
      in_section { print }
    ')

  echo "$content"
}

# ============================================================================
# Issue creation helpers
# ============================================================================

_aw_create_issue_github() {
  # Create a GitHub issue
  # Args: $1 = title, $2 = body
  local title="$1"
  local body="$2"

  if [[ -z "$title" ]]; then
    gum style --foreground 1 "Error: Title is required"
    return 1
  fi

  local issue_url=$(gh issue create --title "$title" --body "$body" 2>&1)

  if [[ $? -eq 0 ]]; then
    gum style --foreground 2 "✓ Issue created: $issue_url"
    echo "$issue_url"
    return 0
  else
    gum style --foreground 1 "Error creating issue: $issue_url"
    return 1
  fi
}

_aw_create_issue_gitlab() {
  # Create a GitLab issue
  # Args: $1 = title, $2 = body
  local title="$1"
  local body="$2"

  if [[ -z "$title" ]]; then
    gum style --foreground 1 "Error: Title is required"
    return 1
  fi

  local issue_url=$(glab issue create --title "$title" --description "$body" 2>&1)

  if [[ $? -eq 0 ]]; then
    gum style --foreground 2 "✓ Issue created: $issue_url"
    echo "$issue_url"
    return 0
  else
    gum style --foreground 1 "Error creating issue: $issue_url"
    return 1
  fi
}

_aw_create_issue_jira() {
  # Create a JIRA issue
  # Args: $1 = title, $2 = body
  local title="$1"
  local body="$2"

  if [[ -z "$title" ]]; then
    gum style --foreground 1 "Error: Summary is required"
    return 1
  fi

  # Get default project
  local project=$(_aw_get_jira_project)

  if [[ -z "$project" ]]; then
    echo ""
    gum style --foreground 6 "JIRA Project Key:"
    project=$(gum input --placeholder "PROJ")
    if [[ -z "$project" ]]; then
      gum style --foreground 1 "Error: Project key is required"
      return 1
    fi
  fi

  local issue_key=$(jira issue create --project "$project" --type "Task" \
    --summary "$title" --body "$body" --plain --no-input 2>&1 | grep -oE '[A-Z]+-[0-9]+' | head -1)

  if [[ -n "$issue_key" ]]; then
    gum style --foreground 2 "✓ Issue created: $issue_key"
    echo "$issue_key"
    return 0
  else
    gum style --foreground 1 "Error creating JIRA issue"
    return 1
  fi
}

_aw_create_issue_linear() {
  # Create a Linear issue
  # Args: $1 = title, $2 = body
  local title="$1"
  local body="$2"

  if [[ -z "$title" ]]; then
    gum style --foreground 1 "Error: Title is required"
    return 1
  fi

  # Get default team
  local team=$(_aw_get_linear_team)

  if [[ -z "$team" ]]; then
    echo ""
    gum style --foreground 6 "Linear Team Key:"
    team=$(gum input --placeholder "TEAM")
    if [[ -z "$team" ]]; then
      gum style --foreground 1 "Error: Team key is required"
      return 1
    fi
  fi

  # Create issue using Linear CLI
  # Format: linear issue create -t "title" -d "description" --team TEAM
  local issue_id=$(linear issue create -t "$title" -d "$body" --team "$team" 2>&1 | grep -oE '[A-Z]+-[0-9]+' | head -1)

  if [[ -n "$issue_id" ]]; then
    gum style --foreground 2 "✓ Issue created: $issue_id"
    echo "$issue_id"
    return 0
  else
    gum style --foreground 1 "Error creating Linear issue"
    return 1
  fi
}

_aw_manual_template_walkthrough() {
  # Walk user through template sections manually
  # Args: $1 = template file path
  # Returns: Issue body as markdown
  local template_file="$1"
  local body=""

  # Read template content
  local template_content=$(cat "$template_file")

  # For now, use gum write to let user edit the template
  echo ""
  gum style --foreground 6 "Edit the issue template:"
  gum style --foreground 8 "Fill in the template sections (Ctrl+D when done, Ctrl+C to cancel)"
  echo ""

  body=$(echo "$template_content" | gum write --width 80 --height 20)

  # Check if user cancelled
  if [[ $? -ne 0 ]]; then
    return 1
  fi

  echo "$body"
}

_aw_ai_generate_issue_content() {
  # Use AI to generate issue content from a prompt
  # Args: $1 = user title/prompt, $2 = template file (optional)
  # Returns: Generated issue body content
  local user_prompt="$1"
  local template_file="$2"

  # Check if AI is available
  if [[ "${AI_CMD[1]}" == "skip" ]] || [[ -z "${AI_CMD[*]}" ]]; then
    return 1
  fi

  # Build the prompt for the AI
  local ai_prompt=""
  local template_content=""

  if [[ -n "$template_file" ]] && [[ -f "$template_file" ]]; then
    # Strip YAML frontmatter from template
    template_content=$(sed '/^---$/,/^---$/d' "$template_file")
  fi

  # Create a detailed prompt for the AI
  if [[ -n "$template_content" ]]; then
    ai_prompt="Generate a GitHub issue based on this request: ${user_prompt}

Fill out this template with detailed, helpful content:

${template_content}

Requirements:
- Write in clear, professional language
- Be specific and actionable
- Include relevant examples where applicable
- Fill out ALL sections of the template

Output ONLY the filled template content (no extra commentary)."
  else
    ai_prompt="Generate a detailed GitHub issue description for: ${user_prompt}

Include:
- Clear problem statement or feature request
- Specific details and context
- Expected outcomes or behavior
- Any relevant examples

Output the issue body in markdown format."
  fi

  # Show what we're doing
  echo ""
  gum style --foreground 6 "Generating issue content with ${AI_CMD_NAME}..."
  echo ""

  # Create output file (BSD/macOS compatible)
  # On BSD/macOS, XXXXXX must be at the end of the template, so we create
  # the temp file without .md extension and then rename it
  local output_file=$(mktemp /tmp/aw_issue_XXXXXX)

  # Check if mktemp succeeded
  if [[ -z "$output_file" ]] || [[ ! -f "$output_file" ]]; then
    gum style --foreground 3 "Failed to create temporary file"
    return 1
  fi

  # Add .md extension
  mv "$output_file" "${output_file}.md"
  output_file="${output_file}.md"

  # Execute AI in headless mode with -p flag
  if "${AI_CMD[@]}" -p "$ai_prompt" > "$output_file" 2>&1; then
    # AI completed successfully
    if [[ -s "$output_file" ]]; then
      echo "$output_file"
      return 0
    else
      gum style --foreground 3 "AI generated empty output"
      [[ -n "$output_file" ]] && rm "$output_file"
      return 1
    fi
  else
    # AI failed - unset default AI tool
    gum style --foreground 3 "AI generation failed"
    gum style --foreground 3 "Removing ${AI_CMD_NAME} as default AI tool"
    git config --unset auto-worktree.ai-tool 2>/dev/null || true
    [[ -n "$output_file" ]] && rm "$output_file"
    return 1
  fi
}

_aw_parse_ai_variations() {
  # Parse AI output to extract variations
  # Args: $1 = output file from AI
  # Returns: Displays variations and lets user choose
  local output_file="$1"

  if [[ ! -f "$output_file" ]]; then
    return 1
  fi

  # For now, just return the entire content
  # In a more sophisticated version, we'd parse the variations
  cat "$output_file"
}

_aw_fill_template_section_by_section() {
  # Walk through template sections interactively
  # Args: $1 = template file, $2 = issue title
  local template_file="$1"
  local issue_title="$2"

  # Extract sections from template
  local sections=$(_aw_extract_template_sections "$template_file")

  if [[ -z "$sections" ]]; then
    # No sections found, fall back to full template edit
    _aw_manual_template_walkthrough "$template_file"
    return $?
  fi

  echo ""
  gum style --foreground 6 "Fill out each section of the template:"
  echo ""

  local filled_content=""

  while IFS= read -r section_name; do
    echo ""
    gum style --foreground 4 --bold "## $section_name"
    echo ""

    # Extract the existing content for this section from the template
    local section_template_content=$(_aw_extract_section_content "$template_file" "$section_name")

    # Show template content as context (if it exists)
    if [[ -n "$section_template_content" ]]; then
      # Trim leading/trailing blank lines for display
      local trimmed_content=$(echo "$section_template_content" | sed '/./,$!d' | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}')

      if [[ -n "$trimmed_content" ]]; then
        echo ""
        gum style --foreground 8 "Template guidance:"
        echo "$trimmed_content" | head -20
        echo ""
      fi
    fi

    # Ask user to provide content for this section, pre-populated with template content
    gum style --foreground 6 "Fill in or edit this section (Ctrl+D when done, Ctrl+C to cancel, leave blank to skip):"
    echo ""
    local section_content=$(echo "$section_template_content" | gum write --width 80 --height 15 \
      --char-limit 0)

    # Check if user cancelled
    if [[ $? -ne 0 ]]; then
      return 1
    fi

    # Only add section to filled content if it's not blank
    if [[ -n "$section_content" ]]; then
      filled_content+="## ${section_name}
${section_content}

"
    fi
  done <<< "$sections"

  echo "$filled_content"
}

_aw_create_issue() {
  # Create a new issue interactively
  # Supports both interactive mode and CLI flags
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  # Parse CLI flags
  local flag_title=""
  local flag_body=""
  local flag_template=""
  local flag_no_template=false
  local flag_no_worktree=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --title)
        flag_title="$2"
        shift 2
        ;;
      --body)
        flag_body="$2"
        shift 2
        ;;
      --template)
        flag_template="$2"
        shift 2
        ;;
      --no-template)
        flag_no_template=true
        shift
        ;;
      --no-worktree)
        flag_no_worktree=true
        shift
        ;;
      *)
        gum style --foreground 1 "Unknown option: $1"
        return 1
        ;;
    esac
  done

  # Determine issue provider
  local provider=$(_aw_get_issue_provider)

  # If not configured, prompt user to choose
  if [[ -z "$provider" ]]; then
    _aw_prompt_issue_provider || return 1
    provider=$(_aw_get_issue_provider)
  fi

  # Check for provider-specific dependencies
  _aw_check_issue_provider_deps "$provider" || return 1

  # Variables for issue creation
  local title=""
  local body=""
  local use_template=false
  local template_file=""

  # Non-interactive mode (CLI flags provided)
  if [[ -n "$flag_title" ]]; then
    title="$flag_title"
    body="$flag_body"

    if [[ -n "$flag_template" ]]; then
      if [[ -f "$flag_template" ]]; then
        template_file="$flag_template"
        body=$(_aw_parse_template "$template_file")
      else
        gum style --foreground 1 "Error: Template file not found: $flag_template"
        return 1
      fi
    fi
  else
    # Interactive mode
    # Check if templates are configured
    local templates_disabled=$(_aw_get_issue_templates_disabled)
    local no_prompt=$(_aw_get_issue_templates_prompt_disabled)

    # Show re-enable instructions if templates are disabled
    if [[ "$no_prompt" == "true" ]]; then
      echo ""
      gum style --foreground 8 "Note: Template prompts are disabled."
      gum style --foreground 8 "To re-enable: git config auto-worktree.issue-templates-no-prompt false"
    fi

    # Determine if we should prompt for template configuration
    # If no_prompt is false (or not set), we should always prompt/check templates
    # If no_prompt is true, skip prompting
    if [[ "$no_prompt" != "true" ]] && [[ "$flag_no_template" != true ]]; then
      # Check if templates are actually available
      local detected_templates=$(_aw_detect_issue_templates "$provider")

      # Show one-time notification if templates detected for the first time
      if [[ -n "$detected_templates" ]] && [[ -z "$(_aw_get_issue_templates_detected_flag)" ]]; then
        local template_count=$(echo "$detected_templates" | wc -l | tr -d ' ')
        echo ""
        gum style --foreground 2 "✓ Detected $template_count issue template(s) in $(_aw_get_template_default_dir "$provider")"
        gum style --foreground 8 "Templates will be available when creating issues"
        _aw_set_issue_templates_detected_flag
        echo ""
      fi

      # Prompt for configuration if:
      # - Templates were never configured (templates_disabled is empty)
      # - Templates were previously disabled (templates_disabled is "true")
      # - Templates are enabled but none are detected (need to configure location)
      if [[ -z "$templates_disabled" ]] || [[ "$templates_disabled" == "true" ]] || [[ -z "$detected_templates" ]]; then
        # Ask user to configure templates
        if _aw_configure_issue_templates "$provider"; then
          use_template=true
        fi
      else
        # Templates are configured, enabled, and detected - use them
        use_template=true
      fi
    elif [[ "$templates_disabled" != "true" ]] && [[ "$flag_no_template" != true ]]; then
      # no_prompt is true, but templates are enabled - use them silently if available
      local detected_templates=$(_aw_detect_issue_templates "$provider")
      if [[ -n "$detected_templates" ]]; then
        use_template=true
      fi
    fi

    # Get title/prompt
    echo ""
    gum style --foreground 6 "Enter issue title/prompt:"
    title=$(gum input --placeholder "Issue title or brief description" --width 80)

    if [[ $? -ne 0 ]] || [[ -z "$title" ]]; then
      gum style --foreground 3 "Cancelled"
      return 0
    fi

    # Check if AI is available and offer AI-assisted generation
    local use_ai=false
    local ai_output_file=""

    # Resolve AI command to check availability
    # Only resolve if not already set (to avoid re-prompting)
    if [[ -z "${AI_CMD[*]}" ]] || [[ "${AI_CMD[1]}" == "" ]]; then
      _resolve_ai_command
    fi

    if [[ -n "${AI_CMD[*]}" ]] && [[ "${AI_CMD[1]}" != "skip" ]]; then
      echo ""
      if gum confirm "Use ${AI_CMD_NAME} to help generate the issue?"; then
        use_ai=true
      fi
    fi

    # Handle template-based or simple flow
    if [[ "$use_template" == true ]]; then
      # Get available templates
      local templates=$(_aw_detect_issue_templates "$provider")

      if [[ -n "$templates" ]]; then
        # Let user choose template
        echo ""
        gum style --foreground 6 "Choose an issue template:"
        echo ""

        # Build template choices (show basenames)
        local template_choices=()
        while IFS= read -r tmpl; do
          template_choices+=("$(basename "$tmpl")")
        done <<< "$templates"

        template_choices+=("No template (simple form)")

        local choice=$(printf '%s\n' "${template_choices[@]}" | gum choose --height 10)

        if [[ $? -ne 0 ]] || [[ -z "$choice" ]]; then
          gum style --foreground 3 "Cancelled"
          return 0
        fi

        if [[ "$choice" == "No template (simple form)" ]]; then
          # Simple body input
          if [[ "$use_ai" == true ]]; then
            # Use AI to generate content without template
            ai_output_file=$(_aw_ai_generate_issue_content "$title" "" "")
            if [[ -n "$ai_output_file" ]] && [[ -f "$ai_output_file" ]]; then
              # Let user review and edit the AI-generated content
              echo ""
              gum style --foreground 6 "AI-generated content (review and edit if needed):"
              gum style --foreground 8 "Ctrl+D when done, Ctrl+C to cancel"
              echo ""
              body=$(cat "$ai_output_file" | gum write --width 80 --height 20)
              if [[ $? -ne 0 ]]; then
                rm "$ai_output_file"
                gum style --foreground 3 "Issue creation cancelled"
                return 0
              fi
              rm "$ai_output_file"
            else
              # Fall back to manual input
              echo ""
              gum style --foreground 6 "Enter issue description:"
              gum style --foreground 8 "Ctrl+D to finish, Ctrl+C to cancel"
              echo ""
              body=$(gum write --width 80 --height 15)
              if [[ $? -ne 0 ]]; then
                gum style --foreground 3 "Issue creation cancelled"
                return 0
              fi
            fi
          else
            echo ""
            gum style --foreground 6 "Enter issue description:"
            gum style --foreground 8 "Ctrl+D to finish, Ctrl+C to cancel"
            echo ""
            body=$(gum write --width 80 --height 15)
            if [[ $? -ne 0 ]]; then
              gum style --foreground 3 "Issue creation cancelled"
              return 0
            fi
          fi
        else
          # Find the selected template file
          template_file=$(echo "$templates" | grep "/${choice}$")

          if [[ -f "$template_file" ]]; then
            # Choose how to fill out the template
            if [[ "$use_ai" == true ]]; then
              # AI is available - offer AI generation
              ai_output_file=$(_aw_ai_generate_issue_content "$title" "$template_file")
              if [[ -n "$ai_output_file" ]] && [[ -f "$ai_output_file" ]]; then
                # Let user review and edit AI-generated content
                echo ""
                gum style --foreground 6 "AI-generated content (review and edit if needed):"
                gum style --foreground 8 "Ctrl+D when done, Ctrl+C to cancel"
                echo ""
                body=$(cat "$ai_output_file" | gum write --width 80 --height 20 --char-limit 0)
                if [[ $? -ne 0 ]]; then
                  rm "$ai_output_file"
                  gum style --foreground 3 "Issue creation cancelled"
                  return 0
                fi
                rm "$ai_output_file"
              else
                # AI failed, fall back to section-by-section
                echo ""
                gum style --foreground 3 "AI generation failed, using section-by-section"
                body=$(_aw_fill_template_section_by_section "$template_file" "$title")
                if [[ $? -ne 0 ]]; then
                  gum style --foreground 3 "Issue creation cancelled"
                  return 0
                fi
              fi
            else
              # No AI - offer section-by-section or full edit
              echo ""
              if gum confirm "Fill out template section-by-section? (Recommended)"; then
                body=$(_aw_fill_template_section_by_section "$template_file" "$title")
                if [[ $? -ne 0 ]]; then
                  gum style --foreground 3 "Issue creation cancelled"
                  return 0
                fi
              else
                # Let user edit the whole template at once
                body=$(_aw_manual_template_walkthrough "$template_file")
                if [[ $? -ne 0 ]]; then
                  gum style --foreground 3 "Issue creation cancelled"
                  return 0
                fi
              fi
            fi
          else
            gum style --foreground 1 "Error: Template file not found"
            return 1
          fi
        fi
      else
        # No templates found, fall back to simple input
        if [[ "$use_ai" == true ]]; then
          # Use AI without template
          ai_output_file=$(_aw_ai_generate_issue_content "$title" "" "")
          if [[ -n "$ai_output_file" ]] && [[ -f "$ai_output_file" ]]; then
            echo ""
            gum style --foreground 6 "AI-generated content (review and edit if needed):"
            gum style --foreground 8 "Ctrl+D when done, Ctrl+C to cancel"
            echo ""
            body=$(cat "$ai_output_file" | gum write --width 80 --height 20)
            if [[ $? -ne 0 ]]; then
              rm "$ai_output_file"
              gum style --foreground 3 "Issue creation cancelled"
              return 0
            fi
            rm "$ai_output_file"
          else
            echo ""
            gum style --foreground 6 "Enter issue description:"
            gum style --foreground 8 "Ctrl+D to finish, Ctrl+C to cancel"
            echo ""
            body=$(gum write --width 80 --height 15)
            if [[ $? -ne 0 ]]; then
              gum style --foreground 3 "Issue creation cancelled"
              return 0
            fi
          fi
        else
          echo ""
          gum style --foreground 6 "Enter issue description:"
          gum style --foreground 8 "Ctrl+D to finish"
          echo ""
          body=$(gum write --width 80 --height 15)
          if [[ $? -ne 0 ]]; then
            gum style --foreground 3 "Issue creation cancelled"
            return 0
          fi
        fi
      fi
    else
      # Simple title/body input (no templates)
      if [[ "$use_ai" == true ]]; then
        # Use AI without template
        ai_output_file=$(_aw_ai_generate_issue_content "$title" "" "")
        if [[ -n "$ai_output_file" ]] && [[ -f "$ai_output_file" ]]; then
          echo ""
          gum style --foreground 6 "AI-generated content (review and edit if needed):"
          gum style --foreground 8 "Ctrl+D when done, Ctrl+C to cancel"
          echo ""
          body=$(cat "$ai_output_file" | gum write --width 80 --height 20)
          if [[ $? -ne 0 ]]; then
            rm "$ai_output_file"
            gum style --foreground 3 "Issue creation cancelled"
            return 0
          fi
          rm "$ai_output_file"
        else
          echo ""
          gum style --foreground 6 "Enter issue description:"
          gum style --foreground 8 "Ctrl+D to finish"
          echo ""
          body=$(gum write --width 80 --height 15)
          if [[ $? -ne 0 ]]; then
            gum style --foreground 3 "Issue creation cancelled"
            return 0
          fi
        fi
      else
        echo ""
        gum style --foreground 6 "Enter issue description:"
        gum style --foreground 8 "Ctrl+D to finish"
        echo ""
        body=$(gum write --width 80 --height 15)
        if [[ $? -ne 0 ]]; then
          gum style --foreground 3 "Issue creation cancelled"
          return 0
        fi
      fi
    fi
  fi

  # Show preview and confirm before creating the issue
  echo ""
  gum style --foreground 6 --bold "Issue Preview:"
  echo ""
  gum style --foreground 4 "Title: $title"
  echo ""
  gum style --foreground 8 "Body:"
  echo "$body" | head -20
  if [[ $(echo "$body" | wc -l) -gt 20 ]]; then
    echo ""
    gum style --foreground 8 "(... truncated, full content will be included in issue)"
  fi
  echo ""

  # Confirm before creating
  if ! gum confirm "Create this issue?"; then
    gum style --foreground 3 "Issue creation cancelled"
    return 0
  fi

  # Create the issue
  echo ""
  gum style --foreground 6 "Creating issue..."
  echo ""

  local result=""
  case "$provider" in
    github)
      result=$(_aw_create_issue_github "$title" "$body")
      ;;
    gitlab)
      result=$(_aw_create_issue_gitlab "$title" "$body")
      ;;
    jira)
      result=$(_aw_create_issue_jira "$title" "$body")
      ;;
    linear)
      result=$(_aw_create_issue_linear "$title" "$body")
      ;;
    *)
      gum style --foreground 1 "Error: Unknown provider: $provider"
      return 1
      ;;
  esac

  if [[ $? -ne 0 ]]; then
    return 1
  fi

  # Post-creation options
  if [[ "$flag_no_worktree" != true ]]; then
    echo ""
    if gum confirm "Create worktree for this issue?"; then
      # Extract issue ID from result
      local issue_id=""
      if [[ "$provider" == "github" ]] || [[ "$provider" == "gitlab" ]]; then
        issue_id=$(echo "$result" | grep -oE '#[0-9]+' | tr -d '#' | head -1)
        if [[ -z "$issue_id" ]]; then
          issue_id=$(echo "$result" | grep -oE '/[0-9]+$' | tr -d '/' | head -1)
        fi
      elif [[ "$provider" == "jira" ]] || [[ "$provider" == "linear" ]]; then
        issue_id=$(echo "$result" | grep -oE '[A-Z]+-[0-9]+' | head -1)
      fi

      if [[ -n "$issue_id" ]]; then
        _aw_issue "$issue_id"
      else
        gum style --foreground 3 "Could not extract issue ID from result"
      fi
    fi
  fi

  # Offer to create another issue
  if [[ "$flag_no_worktree" != true ]]; then
    echo ""
    if gum confirm "Create another issue?"; then
      _aw_create_issue
    fi
  fi

  return 0
}

# ============================================================================
# Issue integration
# ============================================================================

_aw_issue() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  # Determine issue provider
  local provider=$(_aw_get_issue_provider)

  # If not configured, prompt user to choose
  if [[ -z "$provider" ]]; then
    _aw_prompt_issue_provider || return 1
    provider=$(_aw_get_issue_provider)
  fi

  # Check for provider-specific dependencies
  _aw_check_issue_provider_deps "$provider" || return 1

  # Detect if argument is GitHub/GitLab issue number or JIRA key
  local issue_id="${1:-}"
  local issue_type=""

  if [[ -n "$issue_id" ]]; then
    # Auto-detect issue type from input
    if [[ "$issue_id" =~ ^[A-Z][A-Z0-9]+-[0-9]+$ ]]; then
      issue_type="jira"
    elif [[ "$issue_id" =~ ^[0-9]+$ ]]; then
      # Both GitHub and GitLab use numbers, so use the configured provider
      issue_type="$provider"
    else
      gum style --foreground 1 "Invalid issue format. Expected: issue number (e.g., 123) or JIRA key (e.g., PROJ-123)"
      return 1
    fi

    # Validate issue type matches provider (only warn for JIRA mismatch)
    if [[ "$issue_type" == "jira" ]] && [[ "$provider" != "jira" ]]; then
      gum style --foreground 3 "Warning: This repository is configured for $provider, but you provided a JIRA issue ID"
      if ! gum confirm "Continue anyway?"; then
        return 0
      fi
      provider="jira"
    fi
  fi

  if [[ -z "$issue_id" ]]; then
    gum spin --spinner dot --title "Fetching issues..." -- sleep 0.1

    local issues=""
    if [[ "$provider" == "jira" ]]; then
      issues=$(_aw_jira_list_issues)
    elif [[ "$provider" == "gitlab" ]]; then
      issues=$(_aw_gitlab_list_issues)
    elif [[ "$provider" == "linear" ]]; then
      issues=$(_aw_linear_list_issues)
    else
      issues=$(gh issue list --limit 100 --state open --json number,title,labels \
        --template '{{range .}}#{{.number}} | {{.title}}{{if .labels}} |{{range .labels}} [{{.name}}]{{end}}{{end}}{{"\n"}}{{end}}' 2>/dev/null)
    fi

    if [[ -z "$issues" ]]; then
      if [[ "$provider" == "jira" ]]; then
        gum style --foreground 1 "No open JIRA issues found"
      elif [[ "$provider" == "gitlab" ]]; then
        gum style --foreground 1 "No open GitLab issues found"
      elif [[ "$provider" == "linear" ]]; then
        gum style --foreground 1 "No open Linear issues found"
      else
        gum style --foreground 1 "No open GitHub issues found"
      fi
      return 1
    fi

    # Detect which issues have active worktrees
    local active_issues=()
    local worktree_list=$(git worktree list --porcelain 2>/dev/null | grep "^worktree " | sed 's/^worktree //')
    if [[ -n "$worktree_list" ]]; then
      while IFS= read -r wt_path; do
        if [[ -d "$wt_path" ]]; then
          local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
          if [[ -n "$wt_branch" ]]; then
            if [[ "$provider" == "jira" ]]; then
              local wt_issue=$(_aw_extract_jira_key "$wt_branch")
            elif [[ "$provider" == "linear" ]]; then
              local wt_issue=$(_aw_extract_linear_key "$wt_branch")
            else
              # Both GitHub and GitLab use issue numbers
              local wt_issue=$(_aw_extract_issue_number "$wt_branch")
            fi
            if [[ -n "$wt_issue" ]]; then
              active_issues+=("$wt_issue")
            fi
          fi
        fi
      done <<< "$worktree_list"
    fi

    # Add highlighting for issues with active worktrees
    local highlighted_issues=""
    while IFS= read -r issue_line; do
      if [[ -n "$issue_line" ]]; then
        # Extract issue ID from the line
        local line_issue=$(echo "$issue_line" | sed 's/^● *//' | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
        # Check if this issue has an active worktree
        local is_active=false
        for active in "${active_issues[@]}"; do
          if [[ "$active" == "$line_issue" ]]; then
            is_active=true
            break
          fi
        done
        # Add indicator if active
        if [[ "$is_active" == "true" ]]; then
          if [[ "$provider" == "jira" ]] || [[ "$provider" == "linear" ]]; then
            highlighted_issues+="● $issue_line"$'\n'
          else
            highlighted_issues+="$(echo "$issue_line" | sed 's/^#/● #/')"$'\n'
          fi
        else
          highlighted_issues+="$issue_line"$'\n'
        fi
      fi
    done <<< "$issues"

    # Build the selection list with auto-select options
    local selection_list=""
    if ! _is_autoselect_disabled; then
      # Auto-select is enabled - show auto-select options at the top
      selection_list="⚡ Auto select"$'\n'
      selection_list+="🚫 Do not show me auto select again"$'\n'
      selection_list+="$highlighted_issues"
    else
      # Auto-select is disabled - add re-enable option at the end
      selection_list="$highlighted_issues"
      selection_list+="⚡ Auto select next issue"$'\n'
    fi

    local selection=$(echo "$selection_list" | gum filter --placeholder "Type to filter issues... (● = active worktree)")

    if [[ -z "$selection" ]]; then
      gum style --foreground 3 "Cancelled"
      return 0
    fi

    # Handle special auto-select options (GitHub and Linear)
    if [[ ("$provider" == "github" || "$provider" == "linear") ]] && [[ "$selection" == "⚡ Auto select" ]]; then
      gum spin --spinner dot --title "AI is selecting best issues..." -- sleep 0.5

      local filtered_issues=""
      if [[ "$provider" == "github" ]]; then
        filtered_issues=$(_ai_select_issues "$issues" "$highlighted_issues" "${REPO_OWNER}/${REPO_NAME}")
      elif [[ "$provider" == "linear" ]]; then
        filtered_issues=$(_ai_select_linear_issues "$issues" "$highlighted_issues")
      fi

      if [[ -z "$filtered_issues" ]]; then
        gum style --foreground 1 "AI selection failed, showing all issues"
        filtered_issues="$highlighted_issues"
      else
        echo ""
        gum style --foreground 2 "✓ AI selected top 5 issues in priority order"
        echo ""
      fi

      # Show the filtered list
      selection=$(echo "$filtered_issues" | gum filter --placeholder "Select an issue from AI recommendations")

      if [[ -z "$selection" ]]; then
        gum style --foreground 3 "Cancelled"
        return 0
      fi

      issue_id=$(echo "$selection" | sed 's/^● *//' | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')

    elif [[ ("$provider" == "github" || "$provider" == "linear") ]] && [[ "$selection" == "🚫 Do not show me auto select again" ]]; then
      _disable_autoselect
      gum style --foreground 3 "Auto-select disabled. You can re-enable it from the bottom of the issue list."
      # Recursively call to show the updated list
      _aw_issue
      return $?

    elif [[ "$provider" == "github" ]] && [[ "$selection" == "⚡ Auto select next issue" ]]; then
      _enable_autoselect
      gum style --foreground 2 "Auto-select re-enabled!"
      # Recursively call to show the updated list
      _aw_issue
      return $?

    else
      # Normal issue selection (works for both GitHub and JIRA)
      issue_id=$(echo "$selection" | sed 's/^● *//' | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
    fi
  fi

  # Fetch issue details including body
  local title=""
  local body=""

  if [[ "$provider" == "jira" ]]; then
    _aw_jira_get_issue_details "$issue_id" || {
      gum style --foreground 1 "Could not fetch JIRA issue $issue_id"
      return 1
    }
  elif [[ "$provider" == "gitlab" ]]; then
    _aw_gitlab_get_issue_details "$issue_id" || {
      gum style --foreground 1 "Could not fetch GitLab issue #$issue_id"
      return 1
    }
  elif [[ "$provider" == "linear" ]]; then
    _aw_linear_get_issue_details "$issue_id" || {
      gum style --foreground 1 "Could not fetch Linear issue $issue_id"
      return 1
    }
  else
    title=$(gh issue view "$issue_id" --json title --jq '.title' 2>/dev/null)
    body=$(gh issue view "$issue_id" --json body --jq '.body // ""' 2>/dev/null)

    if [[ -z "$title" ]]; then
      gum style --foreground 1 "Could not fetch GitHub issue #$issue_id"
      return 1
    fi
  fi

  # Check if a worktree already exists for this issue
  local existing_worktree=""
  local worktree_list=$(git worktree list --porcelain 2>/dev/null | grep "^worktree " | sed 's/^worktree //')
  if [[ -n "$worktree_list" ]]; then
    while IFS= read -r wt_path; do
      if [[ -d "$wt_path" ]]; then
        local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
        if [[ -n "$wt_branch" ]]; then
          local wt_issue=""
          if [[ "$provider" == "jira" ]]; then
            wt_issue=$(_aw_extract_jira_key "$wt_branch")
          elif [[ "$provider" == "linear" ]]; then
            wt_issue=$(_aw_extract_linear_key "$wt_branch")
          else
            wt_issue=$(_aw_extract_issue_number "$wt_branch")
          fi
          if [[ "$wt_issue" == "$issue_id" ]]; then
            existing_worktree="$wt_path"
            break
          fi
        fi
      fi
    done <<< "$worktree_list"
  fi

  # If an active worktree exists for this issue, offer to resume it
  if [[ -n "$existing_worktree" ]]; then
    echo ""
    if [[ "$provider" == "jira" ]]; then
      gum style --foreground 3 "Active worktree found for JIRA issue $issue_id:"
    else
      gum style --foreground 3 "Active worktree found for GitHub issue #$issue_id:"
    fi
    echo "  $existing_worktree"
    echo ""

    if gum confirm "Resume existing worktree?"; then
      cd "$existing_worktree" || return 1

      # Set terminal title
      if [[ "$provider" == "jira" ]]; then
        printf '\033]0;JIRA %s - %s\007' "$issue_id" "$title"
      else
        printf '\033]0;GitHub Issue #%s - %s\007' "$issue_id" "$title"
      fi

      _resolve_ai_command || return 1

      if [[ "${AI_CMD[1]}" != "skip" ]]; then
        gum style --foreground 2 "Starting $AI_CMD_NAME..."
        "${AI_CMD[@]}"
      else
        gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
      fi
      return 0
    else
      echo ""
      gum style --foreground 3 "Continuing to create new worktree..."
      echo ""
    fi
  fi

  # Generate suggested branch name
  local sanitized=$(_aw_sanitize_branch_name "$title" | cut -c1-40)
  local suggested=""

  if [[ "$provider" == "jira" ]]; then
    suggested="work/${issue_id}-${sanitized}"
  else
    suggested="work/${issue_id}-${sanitized}"
  fi

  echo ""
  if [[ "$provider" == "jira" ]]; then
    gum style --border rounded --padding "0 1" --border-foreground 5 -- \
      "JIRA ${issue_id}" \
      "$title"
  else
    gum style --border rounded --padding "0 1" --border-foreground 5 -- \
      "Issue #${issue_id}" \
      "$title"
  fi

  echo ""
  gum style --foreground 6 "Confirm branch name:"
  local branch_name=$(gum input --value "$suggested" --placeholder "Branch name")

  if [[ -z "$branch_name" ]]; then
    gum style --foreground 3 "Cancelled"
    return 0
  fi

  # Prepare context to pass to AI tool
  local ai_context=""
  if [[ "$provider" == "jira" ]]; then
    ai_context="I'm working on JIRA issue ${issue_id}.

Title: ${title}

${body}

Ask clarifying questions about the intended work if you can think of any."
  else
    ai_context="I'm working on GitHub issue #${issue_id}.

Title: ${title}

${body}

Ask clarifying questions about the intended work if you can think of any."
  fi

  # Set terminal title
  if [[ "$provider" == "jira" ]]; then
    printf '\033]0;JIRA %s - %s\007' "$issue_id" "$title"
  else
    printf '\033]0;GitHub Issue #%s - %s\007' "$issue_id" "$title"
  fi

  _aw_create_worktree "$branch_name" "$ai_context"
}

# ============================================================================
# PR review integration
# ============================================================================

_aw_pr() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  # Determine provider - check if GitLab is configured, otherwise assume GitHub
  local provider=$(_aw_get_issue_provider)
  if [[ -z "$provider" ]] || [[ "$provider" == "jira" ]]; then
    # Default to GitHub for PR workflow, or detect from remote
    provider="github"
  fi

  # Check for provider-specific dependencies
  _aw_check_issue_provider_deps "$provider" || return 1

  local pr_num="${1:-}"

  if [[ -z "$pr_num" ]]; then
    if [[ "$provider" == "gitlab" ]]; then
      gum spin --spinner dot --title "Fetching merge requests..." -- sleep 0.1
    else
      gum spin --spinner dot --title "Fetching pull requests..." -- sleep 0.1
    fi

    local prs=""
    if [[ "$provider" == "gitlab" ]]; then
      # List GitLab MRs
      local gitlab_server=$(_aw_get_gitlab_server)
      local glab_cmd="glab"
      if [[ -n "$gitlab_server" ]]; then
        glab_cmd="glab --host $gitlab_server"
      fi

      prs=$($glab_cmd mr list --state opened --per-page 100 2>/dev/null | \
        awk -F'\t' '{
          if ($1 ~ /^![0-9]+/) {
            number = substr($1, 2)  # Remove ! prefix
            title = $2
            branch = $3
            gsub(/[()]/, "", branch)  # Remove parentheses
            printf "#%s | ○ | %s | %s\n", number, title, branch
          }
        }')
    else
      # List GitHub PRs with detailed information for AI selection
      prs=$(gh pr list --limit 100 --state open --json number,title,author,headRefName,baseRefName,labels,statusCheckRollup,reviews,additions,deletions,reviewRequests 2>/dev/null | \
        jq -r '.[] | "#\(.number) | \(
          if (.statusCheckRollup | length == 0) then "○"
          elif (.statusCheckRollup | all(.state == "SUCCESS")) then "✓"
          elif (.statusCheckRollup | any(.state == "FAILURE" or .state == "ERROR")) then "✗"
          else "○"
          end
        ) | \(.title) | @\(.author.login)\(
          if (.labels | length > 0) then " |" + ([.labels[].name] | map(" [\(.)]") | join(""))
          else ""
          end
        ) | +\(.additions)/-\(.deletions) | \(
          if (.reviews | length) > 0 then "reviews:\(.reviews | length)"
          else "reviews:0"
          end
        )\(
          if (.reviewRequests | length) > 0 then " | requested:[" + ([.reviewRequests[].login] | join(",")) + "]"
          else ""
          end
        ) | \(.headRefName)"')
    fi

    if [[ -z "$prs" ]]; then
      if [[ "$provider" == "gitlab" ]]; then
        gum style --foreground 1 "No open MRs found or not in a GitLab repository"
      else
        gum style --foreground 1 "No open PRs found or not in a GitHub repository"
      fi
      return 1
    fi

    # Detect which PRs/MRs have active worktrees
    local active_prs=()
    local worktree_list=$(git worktree list --porcelain 2>/dev/null | grep "^worktree " | sed 's/^worktree //')
    if [[ -n "$worktree_list" ]]; then
      while IFS= read -r wt_path; do
        if [[ -d "$wt_path" ]]; then
          local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
          # Check if worktree path contains pr-{number} or mr-{number} pattern
          if [[ "$wt_path" =~ (pr|mr)-([0-9]+) ]]; then
            active_prs+=("${BASH_REMATCH[2]}")
          fi
          # Also check by branch name in case PR/MR uses the actual head branch
          if [[ -n "$wt_branch" ]]; then
            # Extract PR/MR number from branch in prs data
            local matching_pr=$(echo "$prs" | grep -E " \| ${wt_branch}\$" | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
            if [[ -n "$matching_pr" ]]; then
              active_prs+=("$matching_pr")
            fi
          fi
        fi
      done <<< "$worktree_list"
    fi

    # Add highlighting for PRs with active worktrees
    local highlighted_prs=""
    while IFS= read -r pr_line; do
      if [[ -n "$pr_line" ]]; then
        # Extract PR number from the line
        local line_pr=$(echo "$pr_line" | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
        # Check if this PR has an active worktree
        local is_active=false
        for active in "${active_prs[@]}"; do
          if [[ "$active" == "$line_pr" ]]; then
            is_active=true
            break
          fi
        done
        # Add indicator if active, and remove the headRefName we added temporarily
        local display_line=$(echo "$pr_line" | sed 's/ | [^|]*$//')
        if [[ "$is_active" == "true" ]]; then
          highlighted_prs+="$(echo "$display_line" | sed 's/^#/● #/')"$'\n'
        else
          highlighted_prs+="$display_line"$'\n'
        fi
      fi
    done <<< "$prs"

    # Build the selection list with auto-select options (GitHub only)
    local selection_list=""
    if [[ "$provider" == "github" ]]; then
      if ! _is_pr_autoselect_disabled; then
        # Auto-select is enabled - show auto-select options at the top
        selection_list="⚡ Auto select"$'\n'
        selection_list+="🚫 Do not show me auto select again"$'\n'
        selection_list+="$highlighted_prs"
      else
        # Auto-select is disabled - add re-enable option at the end
        selection_list="$highlighted_prs"
        selection_list+="⚡ Auto select next PR"$'\n'
      fi
    else
      # GitLab - no auto-select
      selection_list="$highlighted_prs"
    fi

    if [[ "$provider" == "gitlab" ]]; then
      local selection=$(echo "$selection_list" | gum filter --placeholder "Type to filter MRs... (● = active worktree, ○=pending)")
    else
      local selection=$(echo "$selection_list" | gum filter --placeholder "Type to filter PRs... (● = active worktree, ✓=passing ✗=failing ○=pending)")
    fi

    if [[ -z "$selection" ]]; then
      gum style --foreground 3 "Cancelled"
      return 0
    fi

    # Handle special auto-select options (GitHub only)
    if [[ "$provider" == "github" ]] && [[ "$selection" == "⚡ Auto select" ]]; then
      gum spin --spinner dot --title "AI is selecting best PRs..." -- sleep 0.5

      # Get current GitHub user
      local current_user=$(gh api user -q .login 2>/dev/null || echo "unknown")

      local filtered_prs=$(_ai_select_prs "$prs" "$highlighted_prs" "$current_user" "${REPO_OWNER}/${REPO_NAME}")

      if [[ -z "$filtered_prs" ]]; then
        gum style --foreground 1 "AI selection failed, showing all PRs"
        filtered_prs="$highlighted_prs"
      else
        echo ""
        gum style --foreground 2 "✓ AI selected top 5 PRs in priority order"
        echo ""
      fi

      # Show the filtered list
      selection=$(echo "$filtered_prs" | gum filter --placeholder "Select a PR from AI recommendations")

      if [[ -z "$selection" ]]; then
        gum style --foreground 3 "Cancelled"
        return 0
      fi

      pr_num=$(echo "$selection" | sed 's/^● *//' | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')

    elif [[ "$provider" == "github" ]] && [[ "$selection" == "🚫 Do not show me auto select again" ]]; then
      _disable_pr_autoselect
      gum style --foreground 3 "Auto-select disabled. You can re-enable it from the bottom of the PR list."
      # Recursively call to show the updated list
      _aw_pr
      return $?

    elif [[ "$provider" == "github" ]] && [[ "$selection" == "⚡ Auto select next PR" ]]; then
      _enable_pr_autoselect
      gum style --foreground 2 "Auto-select re-enabled!"
      # Recursively call to show the updated list
      _aw_pr
      return $?

    else
      pr_num=$(echo "$selection" | sed 's/^● *//' | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
    fi
  fi

  # Get PR/MR details
  local title=""
  local head_ref=""
  local base_ref=""
  local author=""

  if [[ "$provider" == "gitlab" ]]; then
    local gitlab_server=$(_aw_get_gitlab_server)
    local glab_cmd="glab"
    if [[ -n "$gitlab_server" ]]; then
      glab_cmd="glab --host $gitlab_server"
    fi

    local mr_data=$($glab_cmd mr view "$pr_num" --json title,sourceBranch,targetBranch,author 2>/dev/null)

    if [[ -z "$mr_data" ]]; then
      gum style --foreground 1 "Could not fetch MR !$pr_num"
      return 1
    fi

    title=$(echo "$mr_data" | jq -r '.title')
    head_ref=$(echo "$mr_data" | jq -r '.sourceBranch')
    base_ref=$(echo "$mr_data" | jq -r '.targetBranch')
    author=$(echo "$mr_data" | jq -r '.author.username // .author.login // ""')
  else
    local pr_data=$(gh pr view "$pr_num" --json number,title,headRefName,baseRefName,author 2>/dev/null)

    if [[ -z "$pr_data" ]]; then
      gum style --foreground 1 "Could not fetch PR #$pr_num"
      return 1
    fi

    title=$(echo "$pr_data" | jq -r '.title')
    head_ref=$(echo "$pr_data" | jq -r '.headRefName')
    base_ref=$(echo "$pr_data" | jq -r '.baseRefName')
    author=$(echo "$pr_data" | jq -r '.author.login')
  fi

  # Check if a worktree already exists for this PR/MR
  local existing_worktree=""
  local worktree_prefix="pr"
  if [[ "$provider" == "gitlab" ]]; then
    worktree_prefix="mr"
  fi
  local worktree_name="${worktree_prefix}-${pr_num}"
  local worktree_path="$_AW_WORKTREE_BASE/$worktree_name"

  # First check the standard pr-{number} or mr-{number} path
  if [[ -d "$worktree_path" ]]; then
    existing_worktree="$worktree_path"
  else
    # Also check if the head_ref branch is used by another worktree
    existing_worktree=$(git worktree list --porcelain 2>/dev/null | grep -A2 "^worktree " | grep -B1 "branch refs/heads/${head_ref}$" | head -1 | sed 's/^worktree //')
  fi

  # If an active worktree exists for this PR/MR, offer to resume it
  if [[ -n "$existing_worktree" ]]; then
    echo ""
    if [[ "$provider" == "gitlab" ]]; then
      gum style --foreground 3 "Active worktree found for MR !$pr_num:"
    else
      gum style --foreground 3 "Active worktree found for PR #$pr_num:"
    fi
    echo "  $existing_worktree"
    echo ""

    if gum confirm "Resume existing worktree?"; then
      cd "$existing_worktree" || return 1

      # Set terminal title
      if [[ "$provider" == "gitlab" ]]; then
        printf '\033]0;GitLab MR !%s - %s\007' "$pr_num" "$title"
      else
        printf '\033]0;GitHub PR #%s - %s\007' "$pr_num" "$title"
      fi

      _resolve_ai_command || return 1

      if [[ "${AI_CMD[1]}" != "skip" ]]; then
        gum style --foreground 2 "Starting $AI_CMD_NAME..."
        "${AI_CMD[@]}"
      else
        gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
      fi
      return 0
    else
      echo ""
      gum style --foreground 3 "Continuing to create new worktree..."
      echo ""
    fi
  fi

  echo ""
  if [[ "$provider" == "gitlab" ]]; then
    gum style --border rounded --padding "0 1" --border-foreground 5 -- \
      "MR !${pr_num} by @${author}" \
      "$title" \
      "" \
      "$head_ref -> $base_ref"
  else
    gum style --border rounded --padding "0 1" --border-foreground 5 -- \
      "PR #${pr_num} by @${author}" \
      "$title" \
      "" \
      "$head_ref -> $base_ref"
  fi

  # Create worktree for the PR/MR
  mkdir -p "$_AW_WORKTREE_BASE"

  # Fetch the PR/MR branch and create worktree with proper tracking
  if [[ "$provider" == "gitlab" ]]; then
    gum spin --spinner dot --title "Fetching MR branch..." -- git fetch origin "merge-requests/${pr_num}/head:${head_ref}" 2>/dev/null || \
      git fetch origin "${head_ref}:${head_ref}" 2>/dev/null
  else
    gum spin --spinner dot --title "Fetching PR branch..." -- git fetch origin "pull/${pr_num}/head:${head_ref}" 2>/dev/null || \
      git fetch origin "${head_ref}:${head_ref}" 2>/dev/null
  fi

  # Create worktree with the PR/MR branch
  if ! gum spin --spinner dot --title "Creating worktree..." -- git worktree add "$worktree_path" "$head_ref" 2>/dev/null; then
    # If failed (likely because branch is already checked out elsewhere), try detached
    if [[ "$provider" == "gitlab" ]]; then
      gum style --foreground 6 "Branch already in use, creating detached worktree for MR review..."

      if ! gum spin --spinner dot --title "Fetching MR..." -- git fetch origin "merge-requests/${pr_num}/head" 2>/dev/null; then
        gum style --foreground 1 "Failed to fetch MR"
        return 1
      fi
    else
      gum style --foreground 6 "Branch already in use, creating detached worktree for PR review..."

      if ! gum spin --spinner dot --title "Fetching PR..." -- git fetch origin "pull/${pr_num}/head" 2>/dev/null; then
        gum style --foreground 1 "Failed to fetch PR"
        return 1
      fi
    fi

    if ! gum spin --spinner dot --title "Creating worktree..." -- git worktree add --detach "$worktree_path" FETCH_HEAD; then
      gum style --foreground 1 "Failed to create worktree"
      return 1
    fi
  fi

  cd "$worktree_path" || return 1

  # Fetch base branch for comparison
  git fetch origin "$base_ref" 2>/dev/null

  echo ""
  gum style --border rounded --padding "0 1" --border-foreground 6 \
    "Changes vs $base_ref"

  # Disable pager for diff output to avoid hanging
  git --no-pager diff --stat "origin/${base_ref}...HEAD" 2>/dev/null || git --no-pager diff --stat HEAD~5...HEAD 2>/dev/null

  # Set up the development environment (install dependencies)
  _aw_setup_environment "$worktree_path"

  # Set terminal title
  if [[ "$provider" == "gitlab" ]]; then
    printf '\033]0;GitLab MR !%s - %s\007' "$pr_num" "$title"
  else
    printf '\033]0;GitHub PR #%s - %s\007' "$pr_num" "$title"
  fi

  _resolve_ai_command || return 1

  echo ""
  if [[ "${AI_CMD[1]}" != "skip" ]]; then
    if [[ "$provider" == "gitlab" ]]; then
      gum style --foreground 2 "Starting $AI_CMD_NAME for MR review..."
    else
      gum style --foreground 2 "Starting $AI_CMD_NAME for PR review..."
    fi
    "${AI_CMD[@]}"
  else
    gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
  fi
}

# ============================================================================
# Resume worktree
# ============================================================================

_aw_resume() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info
  _aw_prune_worktrees

  local worktree_list=$(git worktree list --porcelain 2>/dev/null | grep "^worktree " | sed 's/^worktree //')
  local worktree_count=$(echo "$worktree_list" | grep -c . 2>/dev/null || echo 0)

  if [[ $worktree_count -le 1 ]]; then
    gum style --foreground 8 "No additional worktrees for $_AW_SOURCE_FOLDER"
    return 0
  fi

  local now=$(date +%s)
  local one_day=$((24 * 60 * 60))
  local four_days=$((4 * 24 * 60 * 60))

  # Build selection list with formatted display
  local -a worktree_paths=()
  local -a worktree_displays=()

  while IFS= read -r wt_path; do
    [[ "$wt_path" == "$_AW_GIT_ROOT" ]] && continue
    [[ ! -d "$wt_path" ]] && continue

    local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    local commit_timestamp=$(git -C "$wt_path" log -1 --format=%ct 2>/dev/null)

    if [[ -z "$commit_timestamp" ]] || ! [[ "$commit_timestamp" =~ ^[0-9]+$ ]]; then
      commit_timestamp=$(find "$wt_path" -maxdepth 3 -type f -not -path '*/.git/*' -print0 2>/dev/null | while IFS= read -r -d '' file; do _aw_get_file_mtime "$file"; done | sort -rn | head -1)
    fi

    # Build display string
    local display="$(basename "$wt_path") ($wt_branch)"

    if [[ -n "$commit_timestamp" ]] && [[ "$commit_timestamp" =~ ^[0-9]+$ ]]; then
      local age=$((now - commit_timestamp))
      local age_days=$((age / one_day))
      local age_hours=$((age / 3600))

      if [[ $age -lt $one_day ]]; then
        display="$display [${age_hours}h ago]"
      else
        display="$display [${age_days}d ago]"
      fi
    fi

    worktree_paths+=("$wt_path")
    worktree_displays+=("$display")
  done <<< "$worktree_list"

  if [[ ${#worktree_paths[@]} -eq 0 ]]; then
    gum style --foreground 8 "No additional worktrees for $_AW_SOURCE_FOLDER"
    return 0
  fi

  echo ""
  gum style --border rounded --padding "0 1" --border-foreground 4 \
    "Resume a worktree for $_AW_SOURCE_FOLDER"
  echo ""

  # Create selection string from displays
  local selection_list=""
  local i=1
  while [[ $i -le ${#worktree_displays[@]} ]]; do
    selection_list+="${worktree_displays[$i]}"
    if [[ $i -lt ${#worktree_displays[@]} ]]; then
      selection_list+=$'\n'
    fi
    ((i++))
  done

  local selected=$(echo "$selection_list" | gum filter --placeholder "Select worktree to resume...")

  if [[ -z "$selected" ]]; then
    gum style --foreground 3 "Cancelled"
    return 0
  fi

  # Find the corresponding path
  local selected_path=""
  local i=1
  while [[ $i -le ${#worktree_displays[@]} ]]; do
    if [[ "${worktree_displays[$i]}" == "$selected" ]]; then
      selected_path="${worktree_paths[$i]}"
      break
    fi
    ((i++))
  done

  if [[ -z "$selected_path" ]]; then
    gum style --foreground 1 "Error: Could not find selected worktree"
    return 1
  fi

  echo ""
  gum style --foreground 2 "Resuming session in:"
  echo "  $selected_path"
  echo ""

  cd "$selected_path" || return 1

  # Set terminal title to the branch name
  local branch_name=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
  printf '\033]0;%s\007' "$branch_name"

  _resolve_ai_command || return 1

  if [[ "${AI_CMD[1]}" != "skip" ]]; then
    # Check if a conversation exists to resume
    # Claude Code stores conversation history in .claude directory
    if [[ -d ".claude" ]] || [[ -f ".claude.json" ]]; then
      # Conversation exists, try to resume
      "${AI_RESUME_CMD[@]}"
    else
      # No conversation found, start a fresh session
      gum style --foreground 3 "No conversation found to continue"
      gum style --foreground 6 "Starting fresh session in worktree..."
      echo ""
      "${AI_CMD[@]}"
    fi
  else
    gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
  fi
}

# ============================================================================
# Main menu
# ============================================================================

_aw_cleanup_interactive() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  local current_path=$(pwd)
  local worktree_list=$(git worktree list --porcelain 2>/dev/null | grep "^worktree " | sed 's/^worktree //')
  local worktree_count=$(echo "$worktree_list" | grep -c . 2>/dev/null || echo 0)

  if [[ $worktree_count -le 1 ]]; then
    gum style --foreground 8 "No additional worktrees to clean up for $_AW_SOURCE_FOLDER"
    return 0
  fi

  local now=$(date +%s)
  local one_day=$((24 * 60 * 60))
  local four_days=$((4 * 24 * 60 * 60))

  # Build list of worktrees with their display information
  local -a wt_choices=()
  local -a wt_paths=()
  local -a wt_branches=()
  local -a wt_warnings=()

  while IFS= read -r wt_path; do
    [[ "$wt_path" == "$_AW_GIT_ROOT" ]] && continue
    [[ "$wt_path" == "$current_path" ]] && continue
    [[ ! -d "$wt_path" ]] && continue

    local wt_branch=$(git -C "$wt_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    local commit_timestamp=$(git -C "$wt_path" log -1 --format=%ct 2>/dev/null)

    if [[ -z "$commit_timestamp" ]] || ! [[ "$commit_timestamp" =~ ^[0-9]+$ ]]; then
      commit_timestamp=$(find "$wt_path" -maxdepth 3 -type f -not -path '*/.git/*' -print0 2>/dev/null | while IFS= read -r -d '' file; do _aw_get_file_mtime "$file"; done | sort -rn | head -1)
    fi

    # Check merge/close status
    local issue_num=$(_aw_extract_issue_number "$wt_branch")
    local status_tag=""
    local warning_msg=""

    if [[ -n "$issue_num" ]] && _aw_check_issue_merged "$issue_num"; then
      status_tag="[merged #$issue_num]"
    elif _aw_check_branch_pr_merged "$wt_branch"; then
      status_tag="[PR merged]"
    elif [[ -n "$issue_num" ]] && _aw_check_issue_closed "$issue_num"; then
      if [[ "$_AW_ISSUE_HAS_PR" == "false" ]]; then
        if _aw_has_unpushed_commits "$wt_path"; then
          status_tag="[closed #$issue_num ⚠ $_AW_UNPUSHED_COUNT unpushed]"
          warning_msg="⚠ HAS UNPUSHED COMMITS"
        else
          status_tag="[closed #$issue_num]"
        fi
      fi
    elif ! _aw_has_unpushed_commits "$wt_path" && _aw_check_no_changes_from_default "$wt_path"; then
      status_tag="[no changes]"
    fi

    # Build age string
    local age_str=""
    if [[ -n "$commit_timestamp" ]] && [[ "$commit_timestamp" =~ ^[0-9]+$ ]]; then
      local age=$((now - commit_timestamp))
      local age_days=$((age / one_day))
      local age_hours=$((age / 3600))

      if [[ $age -lt $one_day ]]; then
        age_str="[${age_hours}h ago]"
      else
        age_str="[${age_days}d ago]"
      fi
    else
      age_str="[unknown]"
    fi

    # Build display string
    local display_name="$(basename "$wt_path") ($wt_branch) $age_str"
    if [[ -n "$status_tag" ]]; then
      display_name="$display_name $status_tag"
    fi

    wt_choices+=("$display_name")
    wt_paths+=("$wt_path")
    wt_branches+=("$wt_branch")
    wt_warnings+=("$warning_msg")
  done <<< "$worktree_list"

  if [[ ${#wt_choices[@]} -eq 0 ]]; then
    gum style --foreground 8 "No worktrees available to clean up (excluding current worktree)"
    return 0
  fi

  # Show selection UI
  gum style --border rounded --padding "0 1" --border-foreground 4 \
    "Select worktrees to clean up (space to select, enter to confirm)"
  echo ""

  local selected=$(printf '%s\n' "${wt_choices[@]}" | gum choose --no-limit --height 15)

  if [[ -z "$selected" ]]; then
    gum style --foreground 8 "No worktrees selected for cleanup"
    return 0
  fi

  # Find indices of selected worktrees
  local -a selected_indices=()
  local i=1
  while IFS= read -r selected_item; do
    local j=1
    while [[ $j -le ${#wt_choices[@]} ]]; do
      if [[ "${wt_choices[$j]}" == "$selected_item" ]]; then
        selected_indices+=($j)
        break
      fi
      ((j++))
    done
    ((i++))
  done <<< "$selected"

  # Show what will be deleted and confirm
  echo ""
  gum style --foreground 5 "Worktrees selected for cleanup:"
  echo ""

  local has_warnings=false
  for idx in "${selected_indices[@]}"; do
    local display="${wt_choices[$idx]}"
    local warning="${wt_warnings[$idx]}"

    if [[ -n "$warning" ]]; then
      echo "  • $display"
      echo "    $(gum style --foreground 1 "$warning")"
      has_warnings=true
    else
      echo "  • $display"
    fi
  done

  echo ""
  if [[ "$has_warnings" == "true" ]]; then
    gum style --foreground 3 "⚠ Warning: Some worktrees have unpushed commits!"
    echo ""
  fi

  if ! gum confirm "Delete these worktrees and their branches?"; then
    gum style --foreground 8 "Cleanup cancelled"
    return 0
  fi

  # Perform cleanup
  for idx in "${selected_indices[@]}"; do
    local c_path="${wt_paths[$idx]}"
    local c_branch="${wt_branches[$idx]}"

    echo ""
    gum spin --spinner dot --title "Removing $(basename "$c_path")..." -- git worktree remove --force "$c_path"
    gum style --foreground 2 "✓ Worktree removed: $(basename "$c_path")"

    if git show-ref --verify --quiet "refs/heads/${c_branch}"; then
      git branch -D "$c_branch" 2>/dev/null
      gum style --foreground 2 "✓ Branch deleted: $c_branch"
    fi
  done

  echo ""
  gum style --foreground 2 "Cleanup complete!"
}

_aw_menu() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  # Show existing worktrees
  _aw_list

  echo ""

  local choice=$(gum choose \
    "New worktree" \
    "Resume worktree" \
    "Work on issue" \
    "Create issue" \
    "Review PR" \
    "Cleanup worktrees" \
    "Settings" \
    "Cancel")

  case "$choice" in
    "New worktree")       _aw_new true ;;
    "Resume worktree")    _aw_resume ;;
    "Work on issue")      _aw_issue ;;
    "Create issue")       _aw_create_issue ;;
    "Review PR")          _aw_pr ;;
    "Cleanup worktrees")  _aw_cleanup_interactive ;;
    "Settings")           _aw_settings_menu ;;
    *)                    return 0 ;;
  esac
}

# ============================================================================
# Main entry point
# ============================================================================

auto-worktree() {
  _aw_check_deps || return 1

  case "${1:-}" in
    new)     shift; _aw_new "$@" ;;
    issue)   shift; _aw_issue "$@" ;;
    create)  shift; _aw_create_issue "$@" ;;
    pr)      shift; _aw_pr "$@" ;;
    resume)  shift; _aw_resume ;;
    list)    shift; _aw_list ;;
    cleanup) shift; _aw_cleanup_interactive ;;
    settings) shift; _aw_settings_menu ;;
    help|--help|-h)
      echo "Usage: auto-worktree [command] [args]"
      echo ""
      echo "Commands:"
      echo "  new             Create a new worktree"
      echo "  resume          Resume an existing worktree"
      echo "  issue [id]      Work on an issue (GitHub #123, GitLab #456, JIRA PROJ-123, or Linear TEAM-123)"
      echo "  create          Create a new issue with optional template"
      echo "  pr [num]        Review a GitHub PR or GitLab MR"
      echo "  list            List existing worktrees"
      echo "  cleanup         Interactively clean up worktrees"
      echo "  settings        Configure per-repository settings"
      echo ""
      echo "Run without arguments for interactive menu."
      echo ""
      echo "Create Issue Flags:"
      echo "  --title TEXT       Issue title (required for non-interactive mode)"
      echo "  --body TEXT        Issue description/body"
      echo "  --template PATH    Path to template file to use"
      echo "  --no-template      Skip template selection"
      echo "  --no-worktree      Don't offer to create worktree after issue creation"
      echo ""
      echo "Configuration:"
      echo "  First time using issues? Run 'auto-worktree issue' to configure"
      echo "  your issue provider (GitHub, GitLab, JIRA, or Linear) for this repository."
      ;;
    "")    _aw_menu ;;
    *)
      gum style --foreground 1 "Unknown command: $1"
      echo "Run 'auto-worktree help' for usage"
      return 1
      ;;
  esac
}

# ============================================================================
# Shell Completion
# ============================================================================
#
# Load shell completion for auto-worktree and aw commands.
# Completion files are located in the completions/ directory relative to this script.

# Determine the directory where this script is located
_AW_SCRIPT_DIR="${BASH_SOURCE[0]:-${(%):-%x}}"
_AW_SCRIPT_DIR="$(cd "$(dirname "$_AW_SCRIPT_DIR")" && pwd)"

# Load the appropriate completion file based on the current shell
if [[ -n "$ZSH_VERSION" ]]; then
  # Zsh completion
  if [[ -f "$_AW_SCRIPT_DIR/completions/aw.zsh" ]]; then
    # shellcheck disable=SC1091
    source "$_AW_SCRIPT_DIR/completions/aw.zsh"
  fi
elif [[ -n "$BASH_VERSION" ]]; then
  # Bash completion
  if [[ -f "$_AW_SCRIPT_DIR/completions/aw.bash" ]]; then
    # shellcheck disable=SC1091
    source "$_AW_SCRIPT_DIR/completions/aw.bash"
  fi
fi

# Clean up temporary variable
unset _AW_SCRIPT_DIR

# ============================================================================
# Worktree-aware 'aw' wrapper
# ============================================================================
#
# This function provides a convenient 'aw' alias that is worktree-aware:
# - When in a git repository with a local aw.sh file, it sources that version
#   (useful when developing auto-worktree itself - your changes take effect immediately)
# - Otherwise, it uses the globally-sourced auto-worktree function
# - Provides a shorter command: 'aw' instead of 'auto-worktree'
#
aw() {
  # Check if we're in a git repository
  local git_root
  git_root=$(git rev-parse --show-toplevel 2>/dev/null)

  # If we're in a git repo and there's a local aw.sh, source it
  # This allows developers working on auto-worktree to use their local changes
  if [[ -n "$git_root" && -f "$git_root/aw.sh" ]]; then
    # Only source if it's different from the currently loaded version
    local local_aw_path="$git_root/aw.sh"
    local current_aw_path="${_AW_SOURCE_PATH:-}"

    if [[ "$local_aw_path" != "$current_aw_path" ]]; then
      # shellcheck disable=SC1090
      source "$local_aw_path"
      # Track which version we sourced for future comparisons
      export _AW_SOURCE_PATH="$local_aw_path"
    fi
  fi

  # Call auto-worktree with all provided arguments
  auto-worktree "$@"
}
