#!/bin/sh
set -e

REPO="Titovilal/middleman"
INSTALL_DIR="/usr/local/bin"
BINARY="mdm"

# Detect OS and arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS" && exit 1 ;;
esac

ASSET="mdm-${OS}-${ARCH}"

# Get latest release tag
TAG=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$TAG" ]; then
  echo "Error: could not find latest release"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

echo "Downloading ${BINARY} ${TAG} (${OS}/${ARCH})..."
curl -sL "$URL" -o "/tmp/${BINARY}"
chmod +x "/tmp/${BINARY}"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Done. Run 'mdm' to get started."
