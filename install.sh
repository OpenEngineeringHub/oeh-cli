#!/usr/bin/env bash
# Open Engineering Hub CLI — macOS & Linux Installer Script
# Usage: curl -fsSL https://install.openengineering.dev | sh

set -euo pipefail

REPO="open-engineering-hub/oeh-cli"
BIN_DIR="$HOME/.oeh/bin"
EXE_PATH="$BIN_DIR/oeh"

CYAN='\033[0;36m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo ""
echo -e "${CYAN}  Open Engineering Hub CLI Installer${RESET}"
echo -e "${CYAN}  ─────────────────────────────────${RESET}"
echo ""

mkdir -p "$BIN_DIR"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY_NAME="oeh-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"

echo -e "${YELLOW}  → Downloading OEH CLI (${OS}/${ARCH})...${RESET}"

if curl -fsSL "$DOWNLOAD_URL" -o "$EXE_PATH"; then
  chmod +x "$EXE_PATH"
  echo -e "${GREEN}  ✓ Downloaded to $EXE_PATH${RESET}"
else
  echo -e "${YELLOW}  ! Release download failed. Trying go install...${RESET}"
  if command -v go &>/dev/null; then
    go install "github.com/${REPO}@latest"
    echo -e "${GREEN}  ✓ Installed via Go${RESET}"
    exit 0
  else
    echo "❌ Download failed and Go is not installed."
    exit 1
  fi
fi

# Add PATH hint
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  echo ""
  echo -e "${YELLOW}  → Add OEH CLI to your PATH by running:${RESET}"
  echo "     echo 'export PATH=\"\$HOME/.oeh/bin:\$PATH\"' >> ~/.bashrc  # or ~/.zshrc"
  echo "     source ~/.bashrc"
fi

echo ""
echo -e "${GREEN}  ✓ Installation complete!${RESET}"
echo -e "${CYAN}  Get started:${RESET}"
echo "    oeh login --token <YOUR_TOKEN>"
echo "    oeh doctor"
echo ""
