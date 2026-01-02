# Shell Completion for auto-worktree

This directory contains shell completion scripts for the `auto-worktree` command and its `aw` alias.

## Features

Both zsh and bash completions provide:

- **Command completion**: Tab-complete available commands (new, resume, issue, create, pr, list, cleanup, settings, help)
- **Dynamic issue completion**: When using `aw issue <TAB>`, fetch and display open issues from GitHub
- **Dynamic PR completion**: When using `aw pr <TAB>`, fetch and display open pull requests from GitHub
- **Settings subcommands**: Tab-complete settings operations (set, get, list, reset)
- **Both aliases**: Works for both `auto-worktree` and `aw` commands

## Installation

### Automatic Installation (Recommended)

The completion scripts are automatically loaded when you source `aw.sh` in your shell configuration file. The script detects your shell type and loads the appropriate completion file.

**Zsh** (`~/.zshrc`):
```bash
source /path/to/auto-worktree/aw.sh
```

**Bash** (`~/.bashrc` or `~/.bash_profile`):
```bash
source /path/to/auto-worktree/aw.sh
```

After sourcing, the completions will work immediately for both `auto-worktree` and `aw` commands.

### Manual Installation

If you prefer to load completion files separately, you can source them directly:

#### Zsh

Add to `~/.zshrc`:
```bash
source /path/to/auto-worktree/completions/aw.zsh
```

Or copy to a directory in your `$fpath`:
```bash
# Copy to zsh completions directory
mkdir -p ~/.zsh/completions
cp /path/to/auto-worktree/completions/aw.zsh ~/.zsh/completions/_auto_worktree

# Add to ~/.zshrc (before compinit)
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

#### Bash

Add to `~/.bashrc` or `~/.bash_profile`:
```bash
source /path/to/auto-worktree/completions/aw.bash
```

Or copy to system-wide bash completion directory:
```bash
# macOS with Homebrew
sudo cp /path/to/auto-worktree/completions/aw.bash /usr/local/etc/bash_completion.d/aw

# Linux
sudo cp /path/to/auto-worktree/completions/aw.bash /etc/bash_completion.d/aw
```

### Verifying Installation

After installation, verify completion is working:

```bash
# Type this and press TAB:
aw <TAB>

# You should see available commands:
# new resume issue create pr list cleanup settings help

# Try dynamic issue completion (requires gh CLI):
aw issue <TAB>

# Try dynamic PR completion (requires gh CLI):
aw pr <TAB>
```

## Requirements

### Core Functionality
- **Zsh**: Version 5.0 or later (uses `_arguments` and `_describe`)
- **Bash**: Version 4.0 or later (uses `_init_completion` from bash-completion package)

### Dynamic Completions (Optional)
- **GitHub CLI (`gh`)**: Required for dynamic issue/PR completion
  - Install: `brew install gh` (macOS) or see [GitHub CLI installation](https://cli.github.com/)
  - Must be authenticated: `gh auth login`

Without `gh`, command completion will still work, but issue/PR number completion will not be available.

## Troubleshooting

### Zsh: Completion not working

1. **Check if completion is loaded**:
   ```bash
   which _auto_worktree
   ```
   Should output: `_auto_worktree () { ... }`

2. **Rebuild completion cache**:
   ```bash
   rm -f ~/.zcompdump*
   exec zsh
   ```

3. **Check fpath** (if using manual installation):
   ```bash
   echo $fpath
   ```
   Ensure your completions directory is listed.

### Bash: Completion not working

1. **Check if bash-completion is installed**:
   ```bash
   # macOS
   brew install bash-completion@2

   # Linux (Debian/Ubuntu)
   sudo apt-get install bash-completion
   ```

2. **Ensure bash-completion is sourced** in `~/.bashrc`:
   ```bash
   # macOS with Homebrew
   [[ -r "/usr/local/etc/profile.d/bash_completion.sh" ]] && . "/usr/local/etc/profile.d/bash_completion.sh"

   # Linux
   [[ -r "/etc/bash_completion" ]] && . "/etc/bash_completion"
   ```

3. **Reload shell configuration**:
   ```bash
   source ~/.bashrc
   ```

### Issue/PR completion not showing

1. **Check if `gh` is installed and authenticated**:
   ```bash
   gh --version
   gh auth status
   ```

2. **Manually test GitHub CLI**:
   ```bash
   gh issue list --limit 10 --state open
   gh pr list --limit 10 --state open
   ```

3. **Check if you're in a git repository**:
   ```bash
   git rev-parse --show-toplevel
   ```

## Development

### Testing Changes

After modifying completion scripts:

**Zsh**:
```bash
# Reload completion
source completions/aw.zsh
# Or restart shell
exec zsh
```

**Bash**:
```bash
# Reload completion
source completions/aw.bash
# Or restart shell
exec bash
```

### Completion Script Structure

- **`aw.zsh`**: Zsh completion using `_arguments` and `_describe` functions
- **`aw.bash`**: Bash completion using `complete` builtin and `_init_completion` helper

Both scripts implement:
1. First-level command completion
2. Context-aware subcommand completion
3. Dynamic data fetching from GitHub (via `gh` CLI)

## See Also

- [Zsh Completion System Documentation](http://zsh.sourceforge.net/Doc/Release/Completion-System.html)
- [Bash Completion Documentation](https://github.com/scop/bash-completion)
- [GitHub CLI Documentation](https://cli.github.com/manual/)
