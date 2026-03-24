#!/bin/sh
# TunaAgent install script
# Usage: curl -fsSL https://get.tunaagent.dev/install.sh | bash

set -e

TUNA_VERSION="${TUNA_VERSION:-latest}"
TUNA_DIR="${TUNA_DIR:-/usr/local/bin}"
TUNA_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
TUNA_ARCH="$(uname -m)"

# Map arch names
case "$TUNA_ARCH" in
    x86_64)  TUNA_ARCH="amd64" ;;
    aarch64|arm64) TUNA_ARCH="arm64" ;;
    armv7l)  TUNA_ARCH="armv7" ;;
    *)
        echo "Unsupported architecture: $TUNA_ARCH"
        exit 1
        ;;
esac

# Map OS names
case "$TUNA_OS" in
    darwin) TUNA_OS="darwin" ;;
    linux)  TUNA_OS="linux" ;;
    *)
        echo "Unsupported OS: $TUNA_OS (only macOS and Linux supported)"
        exit 1
        ;;
esac

echo "Installing TunaAgent for $TUNA_OS/$TUNA_ARCH..."

if [ "$TUNA_VERSION" = "latest" ]; then
    TUNA_VERSION="$(curl -fsSL https://api.github.com/repos/andyeswong/tunaagent/releases/latest 2>/dev/null | grep '"tag_name"' | sed 's/.*"v\?\([^"]*\)".*/\1/' | tr -d 'v')"
    if [ -z "$TUNA_VERSION" ]; then
        echo "Could not determine latest version, using known release"
        TUNA_VERSION="v1.0.0"
    fi
fi

TUNA_VERSION="${TUNA_VERSION#v}"

echo "Version: $TUNA_VERSION"
echo "OS: $TUNA_OS | Arch: $TUNA_ARCH"

# Download each binary
for BINARY in tuna-agent tuna tuna-server; do
    ASSET="tunaagent_${TUNA_VERSION}_${TUNA_OS}_${TUNA_ARCH}.tar.gz"
    URL="https://github.com/andyeswong/tunaagent/releases/download/v${TUNA_VERSION}/${ASSET}"

    echo ""
    echo "Downloading $BINARY..."
    echo "  From: $URL"

    if curl -fsSL "$URL" -o "/tmp/${ASSET}"; then
        if tar -xzf "/tmp/${ASSET}" -C /tmp/ 2>/dev/null; then
            DEST="${TUNA_DIR}/${BINARY}"
            if mv "/tmp/${BINARY}" "$DEST" 2>/dev/null || mv "/tmp/tunaagent" "$DEST" 2>/dev/null; then
                chmod +x "$DEST"
                echo "  Installed: $DEST"
            else
                echo "  Warning: could not move binary to $TUNA_DIR"
                echo "  Binary is in /tmp/"
            fi
        else
            # Try as a zip
            if unzip -qo "/tmp/${ASSET}" -d /tmp/ 2>/dev/null; then
                DEST="${TUNA_DIR}/${BINARY}"
                mv "/tmp/${BINARY}" "$DEST" 2>/dev/null || true
                chmod +x "$DEST" 2>/dev/null || true
                echo "  Installed: $DEST"
            else
                echo "  Error: could not extract $ASSET"
                echo "  You may need to build from source:"
                echo "    git clone https://github.com/andyeswong/tunaagent.git"
                echo "    cd tunaagent && go build -o $BINARY ./cmd/$BINARY"
            fi
        fi
        rm -f "/tmp/${ASSET}"
    else
        echo "  Release not found: $ASSET"
        echo "  Building from source instead..."
        echo ""
        echo "  git clone https://github.com/andyeswong/tunaagent.git"
        echo "  cd tunaagent"
        echo "  go build -o $BINARY ./cmd/$BINARY"
    fi
done

echo ""
echo ""
echo " TunaAgent installed! 🐟"
echo ""
echo " Next steps:"
echo "   1. Run a TunaHub server:  https://github.com/andyeswong/tunaagent"
echo "   2. Register an agent:     tuna agent create --name my-laptop"
echo "   3. Connect:               tuna-agent"
echo ""
echo " Add to your shell (~/.bashrc or ~/.zshrc):"
echo "   alias tuna='tuna-agent'"
echo ""
