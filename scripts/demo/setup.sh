#!/usr/bin/env bash
# Prepare a clean, isolated environment for the demo. Sourced (not executed)
# by demo.tape so the exported vars and `cd` persist in vhs's shell.
#
# Usage:  source scripts/demo/setup.sh

set -e

DEMO_DIR="$(mktemp -d)"
export DEMO_DIR
export XDG_CONFIG_HOME="$DEMO_DIR/xdg"
mkdir -p "$XDG_CONFIG_HOME/venv-manager"
cat > "$XDG_CONFIG_HOME/venv-manager/config.json" <<CFG
{"base_dir": "$DEMO_DIR/venvs"}
CFG

REPO_ROOT="$(git rev-parse --show-toplevel)"
cp "$REPO_ROOT/scripts/demo/ai_writes_code.sh" "$DEMO_DIR/ai_writes_code.sh"
chmod +x "$DEMO_DIR/ai_writes_code.sh"

cd "$DEMO_DIR"
venv-manager create ai-demo >/dev/null
clear
