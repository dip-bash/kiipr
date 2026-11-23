#!/usr/bin/env bash
# kiipr uninstaller - removes the binary. Your clipboard history and snippets
# are kept.
#
#   curl -sSL https://raw.githubusercontent.com/dip-bash/kiipr/main/uninstall.sh | bash
#
set -euo pipefail

BINARY_NAME="kiipr"
INSTALL_DIR="/usr/local/bin"

SUDO=""
if [ "$EUID" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    fi
fi

TARGET="${INSTALL_DIR}/${BINARY_NAME}"
if [ ! -f "$TARGET" ]; then
    if command -v kiipr >/dev/null 2>&1; then
        TARGET="$(command -v kiipr)"
    else
        echo "no kiipr binary found in ${INSTALL_DIR} or PATH"
        exit 0
    fi
fi

# stop the daemon if it's running
if command -v kiipr >/dev/null 2>&1; then
    kiipr stop >/dev/null 2>&1 || true
fi

echo "==> Removing $TARGET..."
if [ -n "$SUDO" ]; then
    $SUDO rm -f "$TARGET"
else
    rm -f "$TARGET"
fi

echo "==> kiipr is uninstalled (your clipboard history and snippets are kept)"
