#!/bin/bash

# Source this file from ~/.zshrc to load the shell function `auto-worktree`
#
# Usage:
#   auto-worktree              # Interactive menu
#   auto-worktree new          # Create new worktree
#   auto-worktree resume       # Resume existing worktree
#   auto-worktree issue [num]  # Work on a GitHub issue
#   auto-worktree pr [num]     # Review a GitHub PR
#   auto-worktree list         # List existing worktrees

# ============================================================================
# Dependencies check
# ============================================================================

_aw_check_deps() {
  local missing=()

  if ! command -v gum &> /dev/null; then
    missing+=("gum (install with: brew install gum)")
  fi

  if ! command -v gh &> /dev/null; then
    missing+=("gh (install with: brew install gh)")
  fi

  if ! command -v jq &> /dev/null; then
    missing+=("jq (install with: brew install jq)")
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Missing required dependencies:"
    for dep in "${missing[@]}"; do
      echo "  - $dep"
    done
    return 1
  fi

  return 0
}

# ============================================================================
# AI Command Resolution
# ============================================================================

# Global variables for AI tool selection
AI_CMD=""
AI_CMD_NAME=""
AI_RESUME_CMD=""

# Config directory for storing preferences
_AW_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/auto-worktree"
_AW_PREF_FILE="$_AW_CONFIG_DIR/ai_tool_preference"

# Load saved AI tool preference
_load_ai_preference() {
  if [[ -f "$_AW_PREF_FILE" ]]; then
    cat "$_AW_PREF_FILE"
  fi
}

# Save AI tool preference
_save_ai_preference() {
  local tool="$1"
  mkdir -p "$_AW_CONFIG_DIR"
  echo "$tool" > "$_AW_PREF_FILE"
}

# Install AI tool via interactive menu
_install_ai_tool() {
  echo ""
  gum style --foreground 3 "No AI coding assistant found."
  echo ""

  local choice=$(gum choose \
    "Install Claude Code (Anthropic)" \
    "Install Codex CLI (OpenAI)" \
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
    "Skip - don't use an AI tool")
      AI_CMD="skip"
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

  # Check which tools are available
  command -v claude &> /dev/null && claude_available=true
  command -v codex &> /dev/null && codex_available=true

  # Check for saved preference first
  local saved_pref=$(_load_ai_preference)

  if [[ -n "$saved_pref" ]]; then
    case "$saved_pref" in
      claude)
        if [[ "$claude_available" == true ]]; then
          AI_CMD="claude --dangerously-skip-permissions"
          AI_CMD_NAME="Claude Code"
          AI_RESUME_CMD="claude --dangerously-skip-permissions --continue"
          return 0
        fi
        ;;
      codex)
        if [[ "$codex_available" == true ]]; then
          AI_CMD="codex --yolo"
          AI_CMD_NAME="Codex"
          AI_RESUME_CMD="codex resume --last"
          return 0
        fi
        ;;
      skip)
        AI_CMD="skip"
        AI_CMD_NAME="none"
        return 0
        ;;
    esac
    # If we get here, saved preference is no longer valid (tool uninstalled)
    # Fall through to normal selection
  fi

  # If both are available, let user choose
  if [[ "$claude_available" == true ]] && [[ "$codex_available" == true ]]; then
    echo ""
    gum style --foreground 6 "Multiple AI coding assistants detected!"
    echo ""

    local choice=$(gum choose \
      "Claude Code (Anthropic)" \
      "Codex CLI (OpenAI)" \
      "Skip - don't use an AI tool")

    case "$choice" in
      "Claude Code (Anthropic)")
        AI_CMD="claude --dangerously-skip-permissions"
        AI_CMD_NAME="Claude Code"
        AI_RESUME_CMD="claude --dangerously-skip-permissions --continue"
        ;;
      "Codex CLI (OpenAI)")
        AI_CMD="codex --yolo"
        AI_CMD_NAME="Codex"
        AI_RESUME_CMD="codex resume --last"
        ;;
      "Skip - don't use an AI tool")
        AI_CMD="skip"
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
    AI_CMD="claude --dangerously-skip-permissions"
    AI_CMD_NAME="Claude Code"
    AI_RESUME_CMD="claude --dangerously-skip-permissions --continue"
    return 0
  fi

  if [[ "$codex_available" == true ]]; then
    AI_CMD="codex --yolo"
    AI_CMD_NAME="Codex"
    AI_RESUME_CMD="codex resume --last"
    return 0
  fi

  # Neither tool available - show installation menu
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

_aw_setup_environment() {
  # Automatically set up the development environment based on detected project files
  local worktree_path="$1"

  if [[ ! -d "$worktree_path" ]]; then
    return 0
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

    if [[ "$AI_CMD" != "skip" ]]; then
      gum style --foreground 2 "Starting $AI_CMD_NAME..."
      if [[ -n "$initial_context" ]]; then
        $AI_CMD "$initial_context"
      else
        $AI_CMD
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
      commit_timestamp=$(find "$wt_path" -maxdepth 3 -type f -not -path '*/.git/*' -exec stat -f %m {} \; 2>/dev/null | sort -rn | head -1)
    fi

    # Check if this worktree is linked to a merged issue or has a merged PR
    local issue_num=$(_aw_extract_issue_number "$wt_branch")
    local is_merged=false
    local merged_indicator=""
    local merge_reason=""

    if [[ -n "$issue_num" ]] && _aw_check_issue_merged "$issue_num"; then
      is_merged=true
      merge_reason="issue #$issue_num"
      merged_indicator=" $(gum style --foreground 5 "[merged #$issue_num]")"
    elif _aw_check_branch_pr_merged "$wt_branch"; then
      is_merged=true
      merge_reason="PR"
      merged_indicator=" $(gum style --foreground 5 "[PR merged]")"
    elif [[ -n "$issue_num" ]] && _aw_check_issue_closed "$issue_num"; then
      # Issue is closed but no PR (either open or merged)
      if [[ "$_AW_ISSUE_HAS_PR" == "false" ]]; then
        # Check for unpushed commits
        if _aw_has_unpushed_commits "$wt_path"; then
          # Has unpushed work - mark as closed but with warning
          is_merged=true
          merge_reason="issue #$issue_num closed (⚠ $_AW_UNPUSHED_COUNT unpushed)"
          merged_indicator=" $(gum style --foreground 3 "[closed #$issue_num ⚠]")"
        else
          # No unpushed work - safe to clean up
          is_merged=true
          merge_reason="issue #$issue_num closed"
          merged_indicator=" $(gum style --foreground 5 "[closed #$issue_num]")"
        fi
      fi
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
# Issue integration
# ============================================================================

_aw_issue() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  local issue_num="${1:-}"

  if [[ -z "$issue_num" ]]; then
    gum spin --spinner dot --title "Fetching issues..." -- sleep 0.1

    local issues=$(gh issue list --limit 100 --state open --json number,title,labels \
      --template '{{range .}}#{{.number}} | {{.title}}{{if .labels}} |{{range .labels}} [{{.name}}]{{end}}{{end}}{{"\n"}}{{end}}' 2>/dev/null)

    if [[ -z "$issues" ]]; then
      gum style --foreground 1 "No open issues found or not in a GitHub repository"
      return 1
    fi

    local selection=$(echo "$issues" | gum filter --placeholder "Type to filter issues...")

    if [[ -z "$selection" ]]; then
      gum style --foreground 3 "Cancelled"
      return 0
    fi

    issue_num=$(echo "$selection" | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
  fi

  # Fetch issue details including body
  # Use --jq to extract fields directly, which handles control characters better
  local title=$(gh issue view "$issue_num" --json title --jq '.title' 2>/dev/null)
  local body=$(gh issue view "$issue_num" --json body --jq '.body // ""' 2>/dev/null)

  if [[ -z "$title" ]]; then
    gum style --foreground 1 "Could not fetch issue #$issue_num"
    return 1
  fi

  # Generate suggested branch name
  local sanitized=$(_aw_sanitize_branch_name "$title" | cut -c1-40)
  local suggested="work/${issue_num}-${sanitized}"

  echo ""
  gum style --border rounded --padding "0 1" --border-foreground 5 -- \
    "Issue #${issue_num}" \
    "$title"

  echo ""
  local branch_name=$(gum input --value "$suggested" --placeholder "Branch name" --header "Confirm branch name:")

  if [[ -z "$branch_name" ]]; then
    gum style --foreground 3 "Cancelled"
    return 0
  fi

  # Prepare context to pass to Claude
  local claude_context="I'm working on GitHub issue #${issue_num}.

Title: ${title}

${body}

Ask clarifying questions about the intended work if you can think of any."

  # Set terminal title for GitHub issue
  printf '\033]0;GitHub Issue #%s - %s\007' "$issue_num" "$title"

  _aw_create_worktree "$branch_name" "$claude_context"
}

# ============================================================================
# PR review integration
# ============================================================================

_aw_pr() {
  _aw_ensure_git_repo || return 1
  _aw_get_repo_info

  local pr_num="${1:-}"

  if [[ -z "$pr_num" ]]; then
    gum spin --spinner dot --title "Fetching pull requests..." -- sleep 0.1

    local prs=$(gh pr list --limit 100 --state open --json number,title,author,headRefName,baseRefName,labels,statusCheckRollup 2>/dev/null | \
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
      )"')

    if [[ -z "$prs" ]]; then
      gum style --foreground 1 "No open PRs found or not in a GitHub repository"
      return 1
    fi

    local selection=$(echo "$prs" | gum filter --placeholder "Type to filter PRs... (✓=passing ✗=failing ○=pending)")

    if [[ -z "$selection" ]]; then
      gum style --foreground 3 "Cancelled"
      return 0
    fi

    pr_num=$(echo "$selection" | sed 's/^#//' | cut -d'|' -f1 | tr -d ' ')
  fi

  # Get PR details
  local pr_data=$(gh pr view "$pr_num" --json number,title,headRefName,baseRefName,author 2>/dev/null)

  if [[ -z "$pr_data" ]]; then
    gum style --foreground 1 "Could not fetch PR #$pr_num"
    return 1
  fi

  local title=$(echo "$pr_data" | jq -r '.title')
  local head_ref=$(echo "$pr_data" | jq -r '.headRefName')
  local base_ref=$(echo "$pr_data" | jq -r '.baseRefName')
  local author=$(echo "$pr_data" | jq -r '.author.login')

  echo ""
  gum style --border rounded --padding "0 1" --border-foreground 5 -- \
    "PR #${pr_num} by @${author}" \
    "$title" \
    "" \
    "$head_ref -> $base_ref"

  # Create worktree for the PR
  local worktree_name="pr-${pr_num}"
  local worktree_path="$_AW_WORKTREE_BASE/$worktree_name"

  mkdir -p "$_AW_WORKTREE_BASE"

  # Check if worktree already exists
  if [[ -d "$worktree_path" ]]; then
    gum style --foreground 3 "Worktree already exists at $worktree_path"
    if gum confirm "Switch to existing worktree?"; then
      cd "$worktree_path" || return 1
      # Set terminal title for GitHub PR
      printf '\033]0;GitHub PR #%s - %s\007' "$pr_num" "$title"

      _resolve_ai_command || return 1

      if [[ "$AI_CMD" != "skip" ]]; then
        gum style --foreground 2 "Starting $AI_CMD_NAME..."
        $AI_CMD
      else
        gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
      fi
      return 0
    else
      return 1
    fi
  fi

  # Check if the head_ref branch is already used by another worktree
  local existing_worktree=$(git worktree list --porcelain | grep -A2 "^worktree " | grep -B1 "branch refs/heads/${head_ref}$" | head -1 | sed 's/^worktree //')

  if [[ -n "$existing_worktree" ]]; then
    echo ""
    gum style --foreground 3 "Branch '${head_ref}' is already checked out in another worktree:"
    echo "  $existing_worktree"
    echo ""

    if gum confirm "Switch to existing worktree instead?"; then
      cd "$existing_worktree" || return 1
      # Set terminal title for GitHub PR
      printf '\033]0;GitHub PR #%s - %s\007' "$pr_num" "$title"

      _resolve_ai_command || return 1

      if [[ "$AI_CMD" != "skip" ]]; then
        gum style --foreground 2 "Starting $AI_CMD_NAME for PR review..."
        $AI_CMD
      else
        gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
      fi
      return 0
    else
      gum style --foreground 6 "Creating detached worktree for PR review..."

      # Fetch the PR and create a detached worktree
      if ! gum spin --spinner dot --title "Fetching PR..." -- git fetch origin "pull/${pr_num}/head" 2>/dev/null; then
        gum style --foreground 1 "Failed to fetch PR"
        return 1
      fi

      # Create detached worktree at FETCH_HEAD
      if ! gum spin --spinner dot --title "Creating worktree..." -- git worktree add --detach "$worktree_path" FETCH_HEAD; then
        gum style --foreground 1 "Failed to create worktree"
        return 1
      fi
    fi
  else
    # Normal path: branch not in use, create worktree with it
    # Fetch the PR branch and create worktree with proper tracking
    gum spin --spinner dot --title "Fetching PR branch..." -- git fetch origin "pull/${pr_num}/head:${head_ref}" 2>/dev/null || \
      git fetch origin "${head_ref}:${head_ref}" 2>/dev/null

    # Create worktree with the PR branch
    if ! gum spin --spinner dot --title "Creating worktree..." -- git worktree add "$worktree_path" "$head_ref"; then
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

  # Set terminal title for GitHub PR
  printf '\033]0;GitHub PR #%s - %s\007' "$pr_num" "$title"

  _resolve_ai_command || return 1

  echo ""
  if [[ "$AI_CMD" != "skip" ]]; then
    gum style --foreground 2 "Starting $AI_CMD_NAME for PR review..."
    $AI_CMD
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
      commit_timestamp=$(find "$wt_path" -maxdepth 3 -type f -not -path '*/.git/*' -exec stat -f %m {} \; 2>/dev/null | sort -rn | head -1)
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

  if [[ "$AI_CMD" != "skip" ]]; then
    $AI_RESUME_CMD
  else
    gum style --foreground 3 "Skipping AI tool - worktree is ready for manual work"
  fi
}

# ============================================================================
# Main menu
# ============================================================================

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
    "Review PR" \
    "Cancel")

  case "$choice" in
    "New worktree")    _aw_new true ;;
    "Resume worktree") _aw_resume ;;
    "Work on issue")   _aw_issue ;;
    "Review PR")       _aw_pr ;;
    *)                 return 0 ;;
  esac
}

# ============================================================================
# Main entry point
# ============================================================================

auto-worktree() {
  _aw_check_deps || return 1

  case "${1:-}" in
    new)    shift; _aw_new "$@" ;;
    issue)  shift; _aw_issue "$@" ;;
    pr)     shift; _aw_pr "$@" ;;
    resume) shift; _aw_resume ;;
    list)   shift; _aw_list ;;
    help|--help|-h)
      echo "Usage: auto-worktree [command] [args]"
      echo ""
      echo "Commands:"
      echo "  new           Create a new worktree"
      echo "  resume        Resume an existing worktree"
      echo "  issue [num]   Work on a GitHub issue"
      echo "  pr [num]      Review a GitHub pull request"
      echo "  list          List existing worktrees"
      echo ""
      echo "Run without arguments for interactive menu."
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
# Zsh completion
# ============================================================================

_auto_worktree() {
  local -a commands
  commands=(
    'new:Create a new worktree'
    'resume:Resume an existing worktree'
    'issue:Work on a GitHub issue'
    'pr:Review a GitHub pull request'
    'list:List existing worktrees'
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

compdef _auto_worktree auto-worktree
