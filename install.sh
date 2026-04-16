#!/usr/bin/env sh
set -eu

REPO="${ORQUESTADOR_AUDITOR_REPO:-victo/orquestador_auditor}"
VERSION="${ORQUESTADOR_AUDITOR_VERSION:-latest}"
INSTALL_DIR="${ORQUESTADOR_AUDITOR_INSTALL_DIR:-$HOME/.local/bin}"
BUNDLE="${ORQUESTADOR_AUDITOR_BUNDLE:-}"
SYNC_ALL="${ORQUESTADOR_AUDITOR_SYNC_ALL:-false}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$1" >&2
    exit 1
  }
}

need curl
need tar
need unzip

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$os" in
  linux) goos="linux" ;;
  darwin) goos="darwin" ;;
  *) printf 'unsupported os: %s\n' "$os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) printf 'unsupported arch: %s\n' "$arch" >&2; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)"
fi

if [ -z "$VERSION" ]; then
  printf 'could not resolve release version for %s\n' "$REPO" >&2
  exit 1
fi

archive="orquestador-auditor_${VERSION#v}_${goos}_${goarch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$archive"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

curl -fsSL "$url" -o "$tmpdir/$archive"
mkdir -p "$INSTALL_DIR"
tar -xzf "$tmpdir/$archive" -C "$tmpdir"
install "$tmpdir/orquestador-auditor" "$INSTALL_DIR/orquestador-auditor"

printf 'installed orquestador-auditor to %s\n' "$INSTALL_DIR/orquestador-auditor"

# Portable Tools Setup
PORTABLE_BIN_DIR="$HOME/.orquestador-auditor/bin"
mkdir -p "$PORTABLE_BIN_DIR"
PATH="$PORTABLE_BIN_DIR:$PATH"

download_github_binary() {
  repo="$1"
  pattern="$2"
  name="$3"
  printf 'Downloading %s from %s...\n' "$name" "$repo"
  
  api_url="https://api.github.com/repos/$repo/releases/latest"
  # Very basic JSON parsing with sed/grep for compatibility
  dl_url=$(curl -s "$api_url" | grep "browser_download_url" | grep "$pattern" | grep "$goos" | grep "$goarch" | head -n 1 | cut -d '"' -f 4)
  
  if [ -n "$dl_url" ]; then
    fn=$(basename "$dl_url")
    curl -fsSL "$dl_url" -o "$tmpdir/$fn"
    if [ "${fn##*.}" = "zip" ]; then
      unzip -o "$tmpdir/$fn" -d "$tmpdir/ext"
      find "$tmpdir/ext" -name "$name" -type f -exec cp {} "$PORTABLE_BIN_DIR/$name" \;
    elif [ "${fn##*.}" = "gz" ]; then
      tar -xzf "$tmpdir/$fn" -C "$tmpdir"
      find "$tmpdir" -name "$name" -type f -exec cp {} "$PORTABLE_BIN_DIR/$name" \;
    else
      cp "$tmpdir/$fn" "$PORTABLE_BIN_DIR/$name"
    fi
    chmod +x "$PORTABLE_BIN_DIR/$name"
    printf 'Successfully installed %s\n' "$name"
  fi
}

if [ "$BUNDLE" = "full" ] || [ "$BUNDLE" = "core-web" ]; then
  download_github_binary "projectdiscovery/nuclei" "nuclei" "nuclei"
  download_github_binary "projectdiscovery/katana" "katana" "katana"
  download_github_binary "ffuf/ffuf" "ffuf" "ffuf"
  download_github_binary "gitleaks/gitleaks" "gitleaks" "gitleaks"
fi

if [ -n "$BUNDLE" ]; then
  "$INSTALL_DIR/orquestador-auditor" install --bundle "$BUNDLE" --execute
fi

if [ "$SYNC_ALL" = "true" ]; then
  "$INSTALL_DIR/orquestador-auditor" sync --all
fi
