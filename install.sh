#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RESET='\033[0m'

echo -e "${GREEN}Installing venv-manager...${RESET}"

# Get latest release
LATEST_RELEASE=$(curl -s https://api.github.com/repos/jacopobonomi/venv_manager/releases/latest | grep "tag_name" | cut -d '"' -f 4)

echo -e "${GREEN}Downloading venv-manager ${LATEST_RELEASE}...${RESET}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64) ARCH="arm64" ;;
esac

BINARY="venv-manager"
if [ "$OS" = "darwin" ]; then
  BINARY="venv-manager-darwin"
  [ "$ARCH" = "arm64" ] && BINARY="venv-manager-darwin-arm64"
elif [ "$OS" = "windows" ]; then
  BINARY="venv-manager.exe"
elif [ "$ARCH" = "arm64" ]; then
  BINARY="venv-manager-arm64"
fi

# Download correct binary
LATEST_RELEASE=$(curl -s https://api.github.com/repos/jacopobonomi/venv_manager/releases/latest | grep "tag_name" | cut -d '"' -f 4)
curl -Lo venv-manager "https://github.com/jacopobonomi/venv_manager/releases/download/${LATEST_RELEASE}/${BINARY}"
# Make executable and move to /usr/local/bin
chmod +x venv-manager
sudo mv venv-manager /usr/local/bin/

echo -e "${GREEN}✨ venv-manager installed successfully!${RESET}"
