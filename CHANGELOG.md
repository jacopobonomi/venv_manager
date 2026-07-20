# Changelog

All notable changes to this project are documented here.
Format loosely follows [Keep a Changelog](https://keepachangelog.com/).

## [0.1.0] - 2026-07-20

First tagged release.

### Added
- `create`, `list`, `remove`, `rename`, `clone` — venv lifecycle.
- `install`, `packages`, `upgrade`, `clean` — pip operations, per-venv or `--global`.
- `activate`, `deactivate`, `run <name> -- <cmd>` — use a venv without full activation.
- `size` (with `--global` and `--json`).
- `doctor` — python versions on `PATH`, `uv` presence, broken venvs.
- `prune` — remove venvs unused for N days.
- `export` / `import` — portable JSON manifests.
- `describe` — single-call JSON snapshot (python version, packages, size, freeze hash, activation commands per shell).
- `exec [--with pkgs] [--sandbox] [--keep] -- <cmd>` — ephemeral venv à la `uvx`/`pipx run`, with optional OS sandbox (`sandbox-exec` on macOS, `bwrap` on Linux).
- `snapshot` / `snapshots` / `rollback` — capture pip-freeze state and restore.
- `scan <path> [--venv N]` — extract third-party imports, alias-resolve to pip names, report what's missing in a venv.
- `watch <path> --venv N` — fsnotify watcher that auto-installs missing imports on file change.
- `mcp` — Model Context Protocol server (JSON-RPC 2.0 over stdio) exposing 12 tools for AI clients (Claude Desktop, Cursor, Zed).
- `tui` — interactive Bubble Tea browser.
- `config show|path|init` — XDG-aware JSON config.
- `completion` — bash / zsh / fish / powershell.
- `uv` backend (auto-detected, opt-in via `use_uv: true`).
- Unit tests + integration tests (Python 3.12) + GitHub Actions CI on Ubuntu & macOS.
