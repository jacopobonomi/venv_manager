# venv-manager demo

Two ways to generate the "AI watch mode" demo GIF for the README.

## Option 1 — VHS (fully automated)

[VHS](https://github.com/charmbracelet/vhs) turns a declarative `.tape` file into a terminal GIF. Zero manual recording.

```bash
brew install vhs        # or: go install github.com/charmbracelet/vhs@latest
cd scripts/demo
vhs demo.tape           # produces demo.gif
```

The tape:
1. creates a fresh venv (`ai-demo`)
2. starts `venv-manager watch app.py --venv ai-demo` in the background
3. runs `ai_writes_code.sh`, which appends `import requests` then `from rich import print_json` with pauses — simulating an AI agent iterating
4. shows the venv sync in real time
5. runs the finished script inside the venv

## Option 2 — asciinema + record Claude Desktop live

For a more authentic demo showing a real MCP integration:

1. Start `asciinema rec demo.cast`
2. Open Claude Desktop configured with the `venv-manager` MCP server (see main README)
3. Ask Claude to write and run a Python script that needs `requests` and `rich`
4. Watch Claude call `scan_imports`, `install_packages`, and `run_in_venv` tools
5. Convert with `agg demo.cast demo.gif`

This variant requires manual acting but shows the real Claude Desktop → MCP flow, which is the actual product story.
