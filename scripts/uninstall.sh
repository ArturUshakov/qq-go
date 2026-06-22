#!/usr/bin/env sh
set -eu
INSTALL_PATH="/usr/local/bin/qq"
if [ -w "$(dirname "$INSTALL_PATH")" ] || [ "$(id -u)" -eq 0 ]; then rm -f "$INSTALL_PATH"; elif command -v sudo >/dev/null 2>&1; then sudo rm -f "$INSTALL_PATH"; else printf 'Ошибка: нужны права на удаление %s\n' "$INSTALL_PATH" >&2; exit 1; fi
printf 'qq удалён: %s\n' "$INSTALL_PATH"
