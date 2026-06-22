#!/usr/bin/env sh
set -eu

REPOSITORY="SolasWyrd/qq-go"
INSTALL_PATH="/usr/local/bin/qq"

info() { printf '%s\n' "$1"; }
fail() { printf 'Ошибка: %s\n' "$1" >&2; exit 1; }
command_exists() { command -v "$1" >/dev/null 2>&1; }

detect_os() {
  case "$(uname -s)" in Linux*) printf linux ;; Darwin*) printf darwin ;; *) fail "неподдерживаемая ОС: $(uname -s)" ;; esac
}
detect_arch() {
  case "$(uname -m)" in x86_64|amd64) printf amd64 ;; arm64|aarch64) printf arm64 ;; *) fail "неподдерживаемая архитектура: $(uname -m)" ;; esac
}
download() {
  if command_exists curl; then curl -fsSL "$1" -o "$2"; elif command_exists wget; then wget -qO "$2" "$1"; else fail "нужен curl или wget"; fi
}
fetch_latest_tag() {
  if command_exists curl; then curl -fsSL "https://api.github.com/repos/$REPOSITORY/releases/latest"; else wget -qO- "https://api.github.com/repos/$REPOSITORY/releases/latest"; fi | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1
}
sha256_file() {
  if command_exists sha256sum; then sha256sum "$1" | awk '{print $1}'; elif command_exists shasum; then shasum -a 256 "$1" | awk '{print $1}'; else fail "нужен sha256sum или shasum"; fi
}

OS="$(detect_os)"; ARCH="$(detect_arch)"; TAG="${QQ_VERSION:-$(fetch_latest_tag)}"
[ -n "$TAG" ] || fail "не удалось определить последнюю версию"
ASSET="qq_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/$REPOSITORY/releases/download/$TAG"
TMP_DIR="$(mktemp -d)"; trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

info "Установка qq $TAG для $OS/$ARCH"
download "$BASE_URL/$ASSET" "$TMP_DIR/$ASSET"
download "$BASE_URL/checksums.txt" "$TMP_DIR/checksums.txt"
EXPECTED="$(awk -v file="$ASSET" '$2 == file || $2 == "*" file {print $1; exit}' "$TMP_DIR/checksums.txt")"
[ -n "$EXPECTED" ] || fail "checksum для $ASSET не найден"
ACTUAL="$(sha256_file "$TMP_DIR/$ASSET")"
[ "$EXPECTED" = "$ACTUAL" ] || fail "checksum архива не совпадает"
tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"
[ -f "$TMP_DIR/qq" ] || fail "бинарник qq не найден в архиве"
chmod 0755 "$TMP_DIR/qq"

if [ -w "$(dirname "$INSTALL_PATH")" ] || [ "$(id -u)" -eq 0 ]; then
  install -m 0755 "$TMP_DIR/qq" "$INSTALL_PATH"
elif command_exists sudo; then
  sudo install -m 0755 "$TMP_DIR/qq" "$INSTALL_PATH"
else
  fail "нужны права на запись в /usr/local/bin или sudo"
fi

info "qq установлен: $INSTALL_PATH"
"$INSTALL_PATH" completion install || info "Completion не установлен автоматически"
"$INSTALL_PATH" version || true
