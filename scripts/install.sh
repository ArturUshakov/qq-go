#!/usr/bin/env sh
set -eu

REPOSITORY="ArturUshakov/qq-go"
INSTALL_DIR="${QQ_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="qq"

info() { printf '%s\n' "$1"; }
fail() { printf 'Ошибка: %s\n' "$1" >&2; exit 1; }

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

detect_os() {
  case "$(uname -s)" in
    Linux*) printf 'linux' ;;
    Darwin*) printf 'darwin' ;;
    *) fail "неподдерживаемая ОС: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) fail "неподдерживаемая архитектура: $(uname -m)" ;;
  esac
}

fetch_latest_tag() {
  if command_exists curl; then
    curl -fsSL "https://api.github.com/repos/$REPOSITORY/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1
  elif command_exists wget; then
    wget -qO- "https://api.github.com/repos/$REPOSITORY/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1
  else
    fail "нужен curl или wget"
  fi
}

download() {
  url="$1"
  target="$2"
  if command_exists curl; then
    curl -fL "$url" -o "$target"
  elif command_exists wget; then
    wget -O "$target" "$url"
  else
    fail "нужен curl или wget"
  fi
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
TAG="${QQ_VERSION:-$(fetch_latest_tag)}"
[ -n "$TAG" ] || fail "не удалось определить последнюю версию"

ASSET="qq_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPOSITORY/releases/download/$TAG/$ASSET"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

info "Установка qq $TAG для $OS/$ARCH"
mkdir -p "$INSTALL_DIR"
download "$URL" "$TMP_DIR/$ASSET"
tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"
chmod +x "$TMP_DIR/$BINARY_NAME"
mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

info "qq установлен: $INSTALL_DIR/$BINARY_NAME"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    info "Добавьте в PATH: export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac

if command_exists qq; then
  qq version || true
else
  "$INSTALL_DIR/$BINARY_NAME" version || true
fi
