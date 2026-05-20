#!/usr/bin/env sh
set -eu
INSTALL_DIR="${QQ_INSTALL_DIR:-$HOME/.local/bin}"
rm -f "$INSTALL_DIR/qq"
printf 'qq удален из %s\n' "$INSTALL_DIR"
