#!/usr/bin/env bash
# Simulates an AI agent progressively generating a Python script.
# Each stage adds a new third-party import; `venv-manager watch` picks it up
# and installs it into the target venv.
#
# Usage: ./ai_writes_code.sh <target-file>

set -e
FILE="${1:-app.py}"

banner() {
  printf '\n\033[36m🤖 [AI] %s\033[0m\n' "$1"
}

banner "drafting a script to fetch a URL and pretty-print JSON..."
cat > "$FILE" <<'EOF'
import sys
EOF
sleep 4

banner "I need 'requests' to fetch the URL — adding it."
cat > "$FILE" <<'EOF'
import sys
import requests
EOF
sleep 6

banner "adding 'rich' for pretty output."
cat > "$FILE" <<'EOF'
import sys
import requests
from rich import print_json

resp = requests.get(sys.argv[1])
print_json(resp.text)
EOF
sleep 5

banner "script complete."
