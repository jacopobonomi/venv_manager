# VHS tape for venv-manager AI watch mode demo.
#
# Render with (from repo root):
#   vhs scripts/demo/demo.tape
# Produces demo.gif in the current directory.
#
# Requires: vhs, venv-manager on PATH, git.

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set PlaybackSpeed 1.0

# --- Hidden setup: fresh mktemp dir + isolated config + venv created ---
Hide
Type "source scripts/demo/setup.sh"
Enter
Sleep 800ms
Show

# --- Scene 1: pitch ---
Type "# venv-manager: AI writes Python, the venv installs itself"
Sleep 2500ms
Enter
Sleep 500ms
Type "clear"
Enter

# --- Scene 2: launch watch ---
Type "# 1) watch keeps the venv in sync with the file"
Enter
Sleep 1s
Type "venv-manager watch app.py --venv ai-demo &"
Enter
Sleep 1500ms
Type "clear"
Enter

# --- Scene 3: simulated AI agent writes progressively ---
Type "# 2) an AI agent starts writing code..."
Enter
Sleep 1500ms
Type "./ai_writes_code.sh app.py"
Enter

# ai_writes_code.sh runs ~15s; give ~7s more for the final rich install
Sleep 22s

# --- Scene 4: prove it worked ---
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

# --- Scene 5: outro ---
Type "clear"
Enter
Type "# Zero manual pip install. Zero venv drift."
Enter
Sleep 1500ms
Type "# github.com/jacopobonomi/venv_manager"
Enter
Sleep 3500ms

# --- Hidden cleanup ---
Hide
Type "kill %1 2>/dev/null; venv-manager remove ai-demo >/dev/null 2>&1; rm -rf $DEMO_DIR"
Enter
