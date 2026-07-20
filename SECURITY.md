# Security Policy

## Supported Versions

Only the latest release is supported with security updates.

## Reporting a Vulnerability

Please report security issues **privately** via GitHub's [Security Advisories](https://github.com/jacopobonomi/venv_manager/security/advisories/new).

Do **not** open a public issue for security reports.

You can expect an initial response within 7 days. If the report is valid, a
patched release will follow, and you'll be credited in the release notes
unless you prefer to remain anonymous.

## Scope

In-scope:

- Command injection or arbitrary code execution via crafted venv names, package specifiers, or config values.
- Sandbox escape from `exec --sandbox` when run with the documented flag.
- MCP protocol issues that allow an untrusted MCP client to execute arbitrary code outside its declared tool surface.
- Path traversal or symlink attacks in `snapshot`, `import`, `install`, or file arguments.

Out of scope:

- Running arbitrary user-provided Python code (that's what `pip install` and `python` do — use `exec --sandbox` if you don't trust the input).
- Attacks that require an attacker already having local shell access as the user.
- Denial of service by filling disk with venvs (managed by `prune`).
