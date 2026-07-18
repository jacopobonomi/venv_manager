# venv-manager 🐍

A powerful CLI tool for managing Python virtual environments with ease.

![example_cli](https://github.com/jacopobonomi/venv_manager/blob/main/terminal_example.gif)

## Features ✨

- Create and manage environments (optional `uv` backend for 10-100x faster venvs)
- List, rename, remove, clone environments
- Install packages and track dependencies
- Upgrade packages globally or per environment
- Clean cache and temporary files
- Smart environment activation
- Shell completion for bash, zsh, fish, and powershell
- Size check for environments
- `run` a command inside a venv without activating it
- `doctor` for environment diagnostics
- `prune` stale/unused environments
- `export` / `import` a venv manifest as JSON
- Config file at `~/.config/venv-manager/config.json`
- `--json` output on `list`, `packages`, `size`, `doctor`, `prune`
- Interactive TUI (`venv-manager tui`) powered by Bubble Tea
- **AI-friendly**: `describe` (full JSON snapshot), `exec` (ephemeral venvs like `uvx`), `mcp` (Model Context Protocol server for Claude Desktop / Cursor / Zed)
- **Sandboxed execution** with `--sandbox` flag (macOS `sandbox-exec`, Linux `bwrap`)

| Feature | venv-manager | virtualenv | pyenv-virtualenv | Poetry | Pipenv |
|---------|-------------|------------|-----------------|--------|--------|
| **Create and manage environments** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **List all environments** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Install packages and track dependencies** | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Clone environments** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Upgrade packages globally or per environment** | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Clean cache and temporary files** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Shell completion** | ✅ | ❌ | ❌ | ✅ | ❌ |

## One command install 🚀
```bash
curl -sSL https://raw.githubusercontent.com/jacopobonomi/venv_manager/main/install.sh | bash
```

## Installation

```bash
# Clone repository
git clone https://github.com/jacopobonomi/venv_manager
cd venv_manager

# Install 
make install
```

## Usage 💻

```bash
# Create environment (python-version is optional)
venv-manager create myenv [python-version]

# Activate
eval "$(venv-manager activate myenv)"

# Install packages
venv-manager install myenv requirements.txt

# List packages
venv-manager packages myenv

# Upgrade all packages
venv-manager upgrade myenv

# Check environment size
venv-manager size myenv

# Check all environments size
venv-manager size --global

# Global operations
venv-manager --global clean

# Enable shell completion (bash example)
source <(venv-manager completion bash)
```

## Commands 📖

| Command | Description |
|---------|-------------|
| `create <n> [version]` | Create new environment |
| `activate <n>` | Activate environment |
| `deactivate` | Deactivate current environment |
| `list` | Show all environments |
| `remove <n>` | Delete environment |
| `clone <src> <dst>` | Clone environment |
| `packages <n>` | List installed packages |
| `install <n> <reqs>` | Install requirements |
| `upgrade <n>` | Upgrade packages |
| `clean <n>` | Clean cache files |
| `size <n>` | Check environment size |
| `rename <old> <new>` | Rename an environment |
| `run <n> -- <cmd>` | Run a command inside a venv without activating it |
| `doctor` | Diagnose environment (python versions, uv, broken venvs) |
| `prune [--days N] [--dry-run]` | Remove venvs unused for N days |
| `export <n>` | Print venv manifest as JSON |
| `import <manifest.json>` | Recreate venv from a manifest |
| `config show\|path\|init` | Show / locate / bootstrap the config file |
| `tui` | Interactive terminal UI |
| `describe <n>` | Print a full JSON snapshot of a venv |
| `exec [--with pkg,...] [--sandbox] -- <cmd>` | Ephemeral venv: install, run, cleanup |
| `mcp` | Run as MCP server over stdio (for AI clients) |
| `completion [bash\|zsh\|fish\|powershell]` | Generate shell completion scripts |

## Configuration

Config lives at `~/.config/venv-manager/config.json` (override with `$VENV_MANAGER_CONFIG` or `$XDG_CONFIG_HOME`):

```json
{
  "base_dir": "/custom/path/to/venvs",
  "default_python": "3.12",
  "use_uv": true,
  "prune_after_days": 90
}
```

Bootstrap it: `venv-manager config init`.

## AI integration

**MCP server** — expose venv-manager as native tools to AI clients that speak the Model Context Protocol.

Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "venv-manager": {
      "command": "venv-manager",
      "args": ["mcp"]
    }
  }
}
```

Tools exposed: `list_venvs`, `create_venv`, `remove_venv`, `describe_venv`, `install_packages`, `run_in_venv`, `exec_ephemeral`, `doctor`.

**Ephemeral execution** — run AI-generated code without polluting global state:

```bash
venv-manager exec --with requests -- python -c "import requests; print(requests.__version__)"
venv-manager exec --sandbox --with pandas -- python untrusted_script.py
```

`--sandbox` blocks network and restricts filesystem writes (macOS: `sandbox-exec`, Linux: `bwrap`).

## uv backend

If [`uv`](https://github.com/astral-sh/uv) is installed and `use_uv: true` is set in the config, `create` uses `uv venv` — dramatically faster than `python -m venv`.

## Development 🛠️

Requirements:

- Go 1.21+
- Python 3.x

## Contributing 🤝

1. Fork the repository
2. Create feature branch
3. Commit changes
4. Open pull request

## License 📄

MIT License - See LICENSE file

## Author ✍️

[Jacopo Bonomi](https://github.com/jacopobonomi)