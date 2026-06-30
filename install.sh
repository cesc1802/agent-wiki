#!/usr/bin/env sh
# install.sh — download and install the nvtwiki CLI from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cesc1802/agent-wiki/master/install.sh | sh
#
# Environment overrides:
#   VERSION   Release tag to install (e.g. v1.2.3). Default: latest release.
#   BINDIR    Install directory. Default: $HOME/.local/bin.
#
# Supports macOS (darwin), Linux, and Windows (via Git Bash / MSYS / Cygwin),
# on amd64 and arm64. Verifies the sha256 checksum before installing.
set -eu

REPO="cesc1802/agent-wiki"
BINARY="nvtwiki"
BINDIR="${BINDIR:-$HOME/.local/bin}"

err() {
  echo "install: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"
}

need uname
need tar
# Either curl or wget is acceptable for downloads.
if command -v curl >/dev/null 2>&1; then
  DL="curl -fsSL"
  DL_O="curl -fsSL -o"
elif command -v wget >/dev/null 2>&1; then
  DL="wget -qO-"
  DL_O="wget -qO"
else
  err "need curl or wget to download"
fi

# --- Detect OS -------------------------------------------------------------
os="$(uname -s)"
case "$os" in
  Linux) OS="linux" ;;
  Darwin) OS="darwin" ;;
  MINGW* | MSYS* | CYGWIN* | Windows_NT) OS="windows" ;;
  *) err "unsupported OS: $os" ;;
esac

# --- Detect architecture ---------------------------------------------------
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

# Windows is published for amd64 only.
if [ "$OS" = "windows" ] && [ "$ARCH" != "amd64" ]; then
  err "no published windows build for $ARCH"
fi

# --- Resolve version -------------------------------------------------------
TAG="${VERSION:-}"
if [ -z "$TAG" ]; then
  api="https://api.github.com/repos/${REPO}/releases/latest"
  TAG="$($DL "$api" | grep '"tag_name"' | head -n1 | cut -d '"' -f4)"
  [ -n "$TAG" ] || err "could not resolve latest release tag from GitHub API"
fi
VER="${TAG#v}" # asset filenames drop the leading v

# --- Build names -----------------------------------------------------------
base="${BINARY}_${VER}_${OS}_${ARCH}"
if [ "$OS" = "windows" ]; then
  archive="${base}.zip"
  binfile="${BINARY}.exe"
else
  archive="${base}.tar.gz"
  binfile="${BINARY}"
fi
url="https://github.com/${REPO}/releases/download/${TAG}/${archive}"
sums_url="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

# --- Download into a temp dir ----------------------------------------------
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "install: downloading ${BINARY} ${TAG} (${OS}/${ARCH})"
$DL_O "${tmp}/${archive}" "$url" || err "download failed: $url"

# --- Verify checksum -------------------------------------------------------
if $DL_O "${tmp}/checksums.txt" "$sums_url" 2>/dev/null; then
  expected="$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')"
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual="$(sha256sum "${tmp}/${archive}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      actual="$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')"
    else
      actual=""
      echo "install: warning — no sha256 tool found, skipping verification" >&2
    fi
    if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
      err "checksum mismatch for ${archive} (expected ${expected}, got ${actual})"
    fi
    [ -n "$actual" ] && echo "install: checksum verified"
  else
    echo "install: warning — ${archive} not listed in checksums.txt, skipping verification" >&2
  fi
else
  echo "install: warning — checksums.txt unavailable, skipping verification" >&2
fi

# --- Extract ---------------------------------------------------------------
if [ "$OS" = "windows" ]; then
  need unzip
  unzip -qo "${tmp}/${archive}" -d "$tmp"
else
  tar -xzf "${tmp}/${archive}" -C "$tmp"
fi
[ -f "${tmp}/${binfile}" ] || err "expected ${binfile} not found in archive"

# --- Install ---------------------------------------------------------------
mkdir -p "$BINDIR"
chmod +x "${tmp}/${binfile}"
mv "${tmp}/${binfile}" "${BINDIR}/${binfile}"

echo "install: installed ${binfile} to ${BINDIR}/${binfile}"
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) echo "install: note — ${BINDIR} is not on your PATH; add it to use '${BINARY}' directly" ;;
esac
