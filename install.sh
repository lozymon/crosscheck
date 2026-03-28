#!/bin/sh
set -e

REPO="lozymon/crosscheck"
BINARY="crosscheck"
INSTALL_DIR="/usr/local/bin"

# Detect OS.
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Resolve the latest release version from GitHub.
VERSION="$(curl -sSfL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version" >&2
  exit 1
fi

ARCHIVE="${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing crosscheck ${VERSION} (${OS}/${ARCH})..."

# Download and extract to a temp directory.
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -sSfL "$URL" | tar -xz -C "$TMP"

# Install binary.
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  ln -sf "$INSTALL_DIR/$BINARY" "$INSTALL_DIR/cx"
else
  sudo mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  sudo ln -sf "$INSTALL_DIR/$BINARY" "$INSTALL_DIR/cx"
fi

echo "Installed to $INSTALL_DIR/$BINARY"
echo "Both 'crosscheck' and 'cx' are available."
crosscheck --version
