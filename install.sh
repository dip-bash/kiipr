#!/usr/bin/env bash
# kiipr installer - downloads the latest release binary for your CPU and
# installs it to /usr/local/bin (no root needed if you own that dir).
#
#   curl -sSL https://raw.githubusercontent.com/dip-bash/kiipr/main/install.sh | bash
#
set -euo pipefail

REPO="dip-bash/kiipr"
BINARY_NAME="kiipr"
INSTALL_DIR="/usr/local/bin"

# Upgrading from the old name: stop the old watcher so it releases the
# database before the first kiipr run moves it.
if command -v suchi >/dev/null 2>&1; then
    suchi stop >/dev/null 2>&1 || true
fi

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        BIN_FILE="kiipr-linux-x86_64"
        ;;
    aarch64|arm64)
        BIN_FILE="kiipr-linux-aarch64"
        ;;
    *)
        echo "Error: Unsupported Linux architecture ($ARCH). Kiipr supports x86_64 and aarch64 only."
        exit 1
        ;;
esac

SUDO=""
if [ "$EUID" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        echo "Error: Root privileges or sudo are required to install into ${INSTALL_DIR}."
        exit 1
    fi
fi

URL="https://github.com/${REPO}/releases/latest/download/${BIN_FILE}"
echo "==> Downloading Kiipr ($BIN_FILE)..."
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

if ! curl --fail --location --progress-bar "$URL" --output "$TMP_FILE"; then
    echo "Error: Failed to download binary from $URL"
    exit 1
fi

if [ ! -s "$TMP_FILE" ]; then
    echo "Error: Downloaded file is empty."
    exit 1
fi

echo "==> Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
if [ -n "$SUDO" ]; then
    $SUDO install -m 755 "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
else
    install -m 755 "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo "==> Kiipr installed successfully!"

# ---- systemd user unit --------------------------------------------------------
UNIT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
UNIT_FILE="$UNIT_DIR/kiipr.service"
UNIT_BIN="${INSTALL_DIR}/${BINARY_NAME}"

if [ "$(id -u)" != 0 ] && command -v systemctl >/dev/null 2>&1 \
   && systemctl --user --version >/dev/null 2>&1; then
    mkdir -p "$UNIT_DIR"
    cat > "$UNIT_FILE" <<EOF
[Unit]
Description=kiipr clipboard history watcher
After=graphical-session.target
PartOf=graphical-session.target

[Service]
ExecStart=${UNIT_BIN} daemon --foreground
Restart=on-failure
RestartSec=2

[Install]
WantedBy=graphical-session.target
EOF
    systemctl --user daemon-reload >/dev/null 2>&1 || true
    systemctl --user enable --now kiipr.service >/dev/null 2>&1 || true
    # Replace the unit of the old name, if any, so it cannot start a second
    # watcher at the next login.
    OLD_UNIT="$UNIT_DIR/suchi.service"
    if [ -f "$OLD_UNIT" ]; then
        systemctl --user disable --now suchi.service >/dev/null 2>&1 || true
        rm -f "$OLD_UNIT"
        systemctl --user daemon-reload >/dev/null 2>&1 || true
        echo "==> removed the old suchi systemd unit"
    fi
    echo "==> systemd user unit installed at $UNIT_FILE"
    echo "    Start:   systemctl --user start kiipr.service"
    echo "    Status:  systemctl --user status kiipr.service"
    echo "    Logs:    journalctl --user -u kiipr.service -f"
else
    echo "    (skipped systemd user unit - not available or running as root)"
fi

echo "Run 'kiipr daemon' to start the clipboard watcher manually."
