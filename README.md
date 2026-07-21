# venv-manager

[![CI](https://github.com/jacopobonomi/venv_manager/actions/workflows/ci.yml/badge.svg)](https://github.com/jacopobonomi/venv_manager/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jacopobonomi/venv-manager.svg)](https://pkg.go.dev/github.com/jacopobonomi/venv-manager)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/jacopobonomi/venv_manager)](https://github.com/jacopobonomi/venv_manager/releases)
[![jacopobonomi/venv-manager MCP server](https://glama.ai/mcp/servers/jacopobonomi/venv_manager/badges/score.svg)](https://glama.ai/mcp/servers/jacopobonomi/venv_manager)

A Python virtual-environment runtime for humans **and** AI agents.

Written in Go. One static binary, no runtime deps beyond `python3` (or `uv`, if available).

![demo](scripts/demo/demo.gif)

The GIF above is real: `venv-manager watch app.py --venv X` monitors a file, scans its imports with a tiny AST-lite parser, and pip-installs whatever is missing — every time the file changes. Point it at a script an LLM is iterating on and the venv converges as the code does.

---

## Why

Two failure modes drove this tool:

1. **Human sprawl.** Venvs multiply across `~`, cache directories eat GB, activation syntax varies by shell, and cloning "the env that worked" means copy-pasting `pip freeze` between terminals.
2. **Agent sprawl.** LLMs generating Python routinely `pip install` into the wrong interpreter, forget to set `VIRTUAL_ENV`, leave half-installed packages behind after a crash, and have no typed API to reason about environment state — only shells.

`venv-manager` solves (1) with a clean CLI and (2) with a **Model Context Protocol server**, typed **JSON snapshots**, **ephemeral venvs** with OS-level sandboxing, and a **file watcher** that keeps a venv in sync with an evolving script.

---

## Install

```bash
curl -sSL https://raw.githubusercontent.com/jacopobonomi/venv_manager/main/install.sh | bash
```

Or from source:

```bash
git clone https://github.com/jacopobonomi/venv_manager && cd venv_manager
make install
```

Requires Go 1.21+ to build, Python 3.x at runtime.

---

## AI integration

### MCP server

Exposes venv operations as native [Model Context Protocol](https://modelcontextprotocol.io/) tools. Agentic clients (Claude Desktop, Cursor, Zed) call typed tools with JSON Schemas instead of guessing shell invocations.

Wire it up in Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json`):

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

Tools exposed (JSON-RPC 2.0 over stdio):

| Tool | Purpose |
|---|---|
| `list_venvs` | Names of all managed venvs. |
| `create_venv` | `{name, python_version?}` → new venv, uses `uv` if configured. |
| `remove_venv` | `{name}` → recursive delete. |
| `describe_venv` | `{name}` → full snapshot: python version, packages, size, freeze hash, activation commands per shell. |
| `install_packages` | `{name, packages[] | requirements_file}` → pip install with combined stdout+stderr returned. |
| `run_in_venv` | `{name, command[]}` → exec in the venv with `VIRTUAL_ENV` set and `PATH` prepended. Captured output. |
| `exec_ephemeral` | `{packages[], python_version?, command[]}` → create-install-run-destroy in a single call. |
| `snapshot_venv` | `{name, label?}` → capture pip freeze; enables `rollback_venv`. |
| `list_snapshots` | `{name}` → newest-first. |
| `rollback_venv` | `{name, snapshot_id?}` → uninstall all, reinstall from snapshot. |
| `scan_imports` | `{path, venv?}` → third-party imports found; when `venv` is passed, reports which are missing. |
| `doctor` | Python versions on `PATH`, `uv` availability, broken venvs. |

Implementation: ~350 LOC, zero third-party MCP deps. Newline-delimited JSON-RPC 2.0 on stdin/stdout.

### Ephemeral execution (`uvx`-style, sandboxed)

```bash
# create → install → run → destroy, all in one call
venv-manager exec --with requests -- python -c "import requests; print(requests.__version__)"

# with an OS sandbox: no network, no writes outside /tmp + the ephemeral venv
venv-manager exec --sandbox --with pandas -- python untrusted.py
```

`--sandbox` uses `sandbox-exec` on macOS and `bwrap` on Linux. Deny-by-default profile with explicit allow-lists for the venv path, `/tmp`, and process management. Network is unshared.

### File watcher

```bash
venv-manager watch app.py --venv myenv
```

`fsnotify` on the parent directory (survives editor atomic-rename writes), 500 ms debounce, then:

1. AST-lite regex scan of `.py` files (skips docstrings, relative imports, and vendored dirs like `.venv`, `.git`, `__pycache__`, `node_modules`)
2. Filter against a stdlib module set
3. Resolve import-name → pip-package aliases (`cv2` → `opencv-python`, `sklearn` → `scikit-learn`, `PIL` → `Pillow`, `bs4` → `beautifulsoup4`, `yaml` → `PyYAML`, ...)
4. Diff against installed packages
5. `pip install` the delta

The venv is always a superset of the current file's requirements. This is the loop the demo GIF above exercises.

### JSON snapshot as a single-call context primer

```bash
venv-manager describe myenv
```

```json
{
  "name": "myenv",
  "path": "/Users/me/.venvs/myenv",
  "python_version": "3.12.6",
  "python_path": "/Users/me/.venvs/myenv/bin/python",
  "pip_path": "/Users/me/.venvs/myenv/bin/pip",
  "packages": ["requests==2.34.2", "rich==15.0.0", ...],
  "package_count": 12,
  "size_bytes": 45123456,
  "size_human": "43.03 MB",
  "modified_at": "2026-07-20T15:41:35Z",
  "freeze_hash": "sha256:2c58d830...",
  "activation": {
    "bash": "source /Users/me/.venvs/myenv/bin/activate",
    "zsh":  "source /Users/me/.venvs/myenv/bin/activate",
    "fish": "source /Users/me/.venvs/myenv/bin/activate.fish",
    "pwsh": "/Users/me/.venvs/myenv/bin\\Activate.ps1",
    "cmd":  "/Users/me/.venvs/myenv/bin\\activate.bat"
  }
}
```

One tool call, everything an agent needs to reason about the environment. `freeze_hash` lets an agent detect drift between two `describe` calls in O(1) instead of diffing package lists.

---

## Commands

| Command | Description |
|---|---|
| `create <name> [--python VER]` | Create a venv. Uses `uv` when `use_uv: true` in config. |
| `list [--json]` | List venvs. |
| `remove <name>` | Delete a venv. |
| `rename <old> <new>` | Rename and re-generate activation scripts via `python -m venv --upgrade`. |
| `clone <src> <dst>` | Fresh venv seeded with `pip freeze` of source. |
| `packages <name> [--json]` | Installed packages. |
| `install <name> <requirements>` | `pip install -r`. |
| `upgrade [name] [--global]` | Upgrade outdated packages (per venv or all). |
| `clean [name] [--global]` | Purge pip cache + `__pycache__` dirs. |
| `size [name] [--global] [--json]` | Disk usage. |
| `activate <name>` | Print shell command for `eval $(...)`. |
| `deactivate` | Print `deactivate`. |
| `run <name> -- <cmd>` | Execute in a venv without activating; inherited stdio. |
| `exec [--with pkgs] [-r req] [--python V] [--sandbox] [--keep] -- <cmd>` | Ephemeral venv run. |
| `describe <name>` | Full JSON snapshot (see above). |
| `scan <path> [--venv N] [--json]` | Extract third-party imports; check against venv. |
| `watch <path> --venv N` | Auto-install missing imports on file change. |
| `snapshot <name> [-l LABEL]` | Capture pip-freeze state. |
| `snapshots <name> [--json]` | List snapshots (newest first). |
| `rollback <name> [snapshot-id]` | Uninstall all, reinstall from snapshot. |
| `export <name>` | Print portable manifest (name + python version + freeze) as JSON. |
| `import <manifest.json>` | Recreate venv from manifest. |
| `prune [--days N] [--dry-run] [--json]` | Remove venvs unused for N days. |
| `doctor [--json]` | Diagnose python versions, uv, broken venvs. |
| `config show|path|init` | Show / locate / bootstrap the config. |
| `mcp` | Model Context Protocol server on stdio. |
| `tui` | Bubble Tea TUI browser. |
| `completion [bash|zsh|fish|powershell]` | Shell completion scripts. |

Most read commands also accept `--json` for stable, machine-parseable output.

---

## Configuration

`~/.config/venv-manager/config.json` (respects `$XDG_CONFIG_HOME` and `$VENV_MANAGER_CONFIG`):

```json
{
  "base_dir": "/custom/path/to/venvs",
  "default_python": "3.12",
  "use_uv": true,
  "prune_after_days": 90
}
```

Bootstrap: `venv-manager config init`.

## uv backend

If [`uv`](https://github.com/astral-sh/uv) is on `PATH` and `use_uv: true`, `create` runs `uv venv`. Typically 10–100× faster than `python -m venv` on cold cache.

---

## Development

```bash
make build            # go build -o bin/venv-manager
make test             # unit tests
make demo             # regenerate scripts/demo/demo.gif via VHS
go test -tags=integration ./internal/manager/...   # integration tests (real pip, real PyPI)
```

CI runs `go vet`, `go test -race` on Ubuntu + macOS, and integration tests on Ubuntu with Python 3.12.

Architecture:

```
cmd/venv-manager/           cobra CLI
internal/manager/           core operations (create, install, snapshot, scan, watch, exec, describe, ...)
internal/config/            XDG-aware JSON config
internal/mcp/               JSON-RPC 2.0 MCP server (stdio)
internal/tui/               Bubble Tea browser
internal/utils/             platform helpers, size formatting
```

## License

MIT.

## Author

[Jacopo Bonomi](https://github.com/jacopobonomi)
