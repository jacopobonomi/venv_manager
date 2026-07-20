# VHS tape for the AI watch mode demo.
# Do NOT run this tape directly — use `make demo` from the repo root.
# The Makefile prepares the working dir, then invokes vhs.
#
# Env expected: DEMO_VENV (name of the venv), CWD == working dir with
# app.py absent and ai_writes_code.sh present.

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set PlaybackSpeed 1.0

# --- Scene 1: pitch ---
Type "# venv-manager: AI writes Python, the venv installs itself"
Sleep 2500ms
Enter
Sleep 800ms
Type "clear"
Enter

# --- Scene 2: launch watch ---
Type "# 1) 'watch' keeps the venv in sync with the file"
Enter
Sleep 1200ms
Type "venv-manager watch app.py --venv vhs-demo &"
Enter
Sleep 1500ms
Type "clear"
Enter

# --- Scene 3: AI writes progressively ---
Type "# 2) an AI agent starts writing code..."
Enter
Sleep 1500ms
Type "./ai_writes_code.sh app.py"
Enter

# ai_writes_code.sh runs ~15s; give ~7s buffer for the final rich install
Sleep 22s

# --- Scene 4: prove it worked ---
Type "clear"
Enter
Type "# The venv now has requests + rich. No manual pip install."
Enter
Sleep 1500ms
Type "venv-manager packages vhs-demo | grep -E 'requests|rich'"
Enter
Sleep 3s

Type "clear"
Enter
Type "# Full JSON snapshot — everything an AI agent needs about the env."
Enter
Sleep 1500ms
Type "venv-manager describe vhs-demo | head -12"
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
