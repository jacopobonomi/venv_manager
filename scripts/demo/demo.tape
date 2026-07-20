# VHS tape for venv-manager AI watch mode demo.
#
# Render with (from repo root):
#   vhs scripts/demo/demo.tape
# Produces demo.gif in the current directory.
#
# Requires: vhs, and venv-manager on PATH.

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set PlaybackSpeed 1.0

# --- Setup (hidden) ---
Hide
Type "export DEMO_DIR=$(mktemp -d) && export XDG_CONFIG_HOME=$DEMO_DIR/xdg && mkdir -p $XDG_CONFIG_HOME/venv-manager && echo '{\"base_dir\":\"'$DEMO_DIR'/venvs\"}' > $XDG_CONFIG_HOME/venv-manager/config.json"
Enter
Sleep 300ms
Type "cp $(git rev-parse --show-toplevel)/scripts/demo/ai_writes_code.sh $DEMO_DIR/ai_writes_code.sh && chmod +x $DEMO_DIR/ai_writes_code.sh"
Enter
Sleep 300ms
Type "cd $DEMO_DIR && venv-manager create ai-demo >/dev/null && clear"
Enter
Sleep 500ms
Show

# --- Scene 1: pitch ---
Type "# 🤖 venv-manager: AI writes Python, the venv installs itself"
Sleep 2500ms
Enter
Sleep 500ms
Type "clear"
Enter

# --- Scene 2: launch watch, then run the AI simulator ---
Type "# 1) watch keeps the venv in sync with the file"
Enter
Sleep 1s
Type "venv-manager watch app.py --venv ai-demo &"
Enter
Sleep 1500ms
Type "clear"
Enter
Type "# 2) an AI agent starts writing code..."
Enter
Sleep 1500ms
Type "./ai_writes_code.sh app.py"
Enter

# Full run of ai_writes_code.sh (~15s) + rich install (~5s)
Sleep 22s

# --- Scene 3: prove it works ---
Type "clear"
Enter
Type "# The venv has requests + rich. No manual pip install."
Enter
Sleep 1500ms
Type "venv-manager packages ai-demo | grep -E 'requests|rich'"
Enter
Sleep 3s

Type "clear"
Enter
Type "# Full describe: everything an agent needs to reason about the env."
Enter
Sleep 1500ms
Type "venv-manager describe ai-demo | head -20"
Enter
Sleep 4s

# --- Scene 4: outro ---
Type "clear"
Enter
Type "# Zero manual pip install. Zero venv drift."
Enter
Sleep 1500ms
Type "# github.com/jacopobonomi/venv_manager"
Enter
Sleep 3500ms

# --- Cleanup (hidden) ---
Hide
Type "kill %1 2>/dev/null; venv-manager remove ai-demo >/dev/null 2>&1; rm -rf $DEMO_DIR"
Enter
