#!/usr/bin/env sh
# install.sh — download and install the nvtwiki CLI from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cesc1802/agent-wiki/master/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/cesc1802/agent-wiki/master/install.sh | sh -s -- --version v1.2.3 --global
#
# Options:
#   -v, --version <tag>   Release tag to install (e.g. v1.2.3 or 1.2.3). Default: latest release.
#   -d, --bindir <dir>    Install directory. Default: $HOME/.local/bin
#                         (or /usr/local/bin when --global is used).
#   -g, --global          Install system-wide to /usr/local/bin so the binary is on
#                         PATH everywhere; uses sudo when the directory is not writable.
#   -h, --help            Show this help and exit.
#
# Environment overrides (used only when the matching flag is absent):
#   VERSION   Release tag to install. Default: latest release.
#   BINDIR    Install directory. Default: $HOME/.local/bin.
#
# Supports macOS (darwin), Linux, and Windows (via Git Bash / MSYS / Cygwin),
# on amd64 and arm64. Verifies the sha256 checksum before installing.
set -eu

REPO="cesc1802/agent-wiki"
BINARY="nvtwiki"

err() {
  echo "install: $*" >&2
  exit 1
}

usage() {
  sed -n '2,21p' "$0" 2>/dev/null | sed 's/^# \{0,1\}//' || true
}

need() {
  command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"
}

# --- Parse arguments -------------------------------------------------------
# Flags take precedence over environment variables; env vars stay as fallbacks
# for backwards compatibility and for `curl | sh` without `-s --`.
VERSION="${VERSION:-}"
BINDIR="${BINDIR:-}"
GLOBAL=0
while [ $# -gt 0 ]; do
  case "$1" in
    -v | --version)
      [ $# -ge 2 ] || err "missing value for $1"
      VERSION="$2"
      shift 2
      ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    -d | --bindir)
      [ $# -ge 2 ] || err "missing value for $1"
      BINDIR="$2"
      shift 2
      ;;
    --bindir=*) BINDIR="${1#*=}"; shift ;;
    -g | --global) GLOBAL=1; shift ;;
    -h | --help) usage; exit 0 ;;
    *) err "unknown option: $1 (try --help)" ;;
  esac
done

# Resolve install directory: explicit --bindir/BINDIR wins, then --global, then user-local.
if [ -z "$BINDIR" ]; then
  if [ "$GLOBAL" -eq 1 ]; then
    BINDIR="/usr/local/bin"
  else
    BINDIR="$HOME/.local/bin"
  fi
fi

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
TAG="$VERSION"
if [ -z "$TAG" ]; then
  api="https://api.github.com/repos/${REPO}/releases/latest"
  TAG="$($DL "$api" | grep '"tag_name"' | head -n1 | cut -d '"' -f4)"
  [ -n "$TAG" ] || err "could not resolve latest release tag from GitHub API"
else
  # Accept a bare version (1.2.3) as well as a tag (v1.2.3).
  case "$TAG" in
    v*) ;;
    [0-9]*) TAG="v${TAG}" ;;
  esac
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
# Use sudo when the install directory exists but is not writable (typical for
# system-wide locations like /usr/local/bin), or when its parent is not writable.
SUDO=""
if [ -d "$BINDIR" ]; then
  [ -w "$BINDIR" ] || SUDO="sudo"
else
  parent="$(dirname "$BINDIR")"
  [ -w "$parent" ] || SUDO="sudo"
fi
if [ -n "$SUDO" ]; then
  command -v sudo >/dev/null 2>&1 || err "cannot write to ${BINDIR} and sudo not found; choose a writable --bindir"
  echo "install: ${BINDIR} requires elevated permissions; using sudo"
fi

chmod +x "${tmp}/${binfile}"
$SUDO mkdir -p "$BINDIR"
$SUDO mv "${tmp}/${binfile}" "${BINDIR}/${binfile}"

echo "install: installed ${binfile} to ${BINDIR}/${binfile}"
case ":$PATH:" in
  *":$BINDIR:"*)
    echo "install: '${BINARY}' is on your PATH — run it from anywhere"
    ;;
  *)
    echo "install: note — ${BINDIR} is not on your PATH."
    echo "install:   add it by appending this line to your shell profile (e.g. ~/.profile, ~/.bashrc, ~/.zshrc):"
    echo "install:     export PATH=\"${BINDIR}:\$PATH\""
    echo "install:   or re-run with --global to install to /usr/local/bin."
    ;;
esac
