# venv-manager 🐍

A powerful CLI tool for managing Python virtual environments with ease.

![example_cli](https://github.com/jacopobonomi/venv_manager/blob/main/terminal_example.gif)

## Features ✨

- Create and manage environments
- List all environments
- Install packages and track dependencies
- Clone environments
- Upgrade packages globally or per environment
- Clean cache and temporary files
- Smart environment activation

| Feature | venv-manager | virtualenv | pyenv-virtualenv | Poetry | Pipenv |
|---------|-------------|------------|-----------------|--------|--------|
| **Create and manage environments** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **List all environments** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Install packages and track dependencies** | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Clone environments** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Upgrade packages globally or per environment** | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Clean cache and temporary files** | ✅ | ❌ | ❌ | ✅ | ❌ |

## One command install 🚀
```bash
curl -sSL https://raw.githubusercontent.com/jacopobonomi/venv_manager/main/install.sh | bash
```

## Installation

```bash
# Clone repository
git clone https://github.com/jacopobonomi/venv-manager
cd venv-manager

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

# Global operations
venv-manager --global clean
```

## Commands 📖

| Command | Description |
|---------|-------------|
| `create <name> [version]` | Create new environment |
| `activate <name>` | Activate environment |
| `deactivate` | Deactivate current environment |
| `list` | Show all environments |
| `remove <name>` | Delete environment |
| `clone <src> <dst>` | Clone environment |
| `packages <name>` | List installed packages |
| `install <name> <reqs>` | Install requirements |
| `upgrade <name>` | Upgrade packages |
| `clean <name>` | Clean cache files |

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
