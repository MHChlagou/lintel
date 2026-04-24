#!/bin/sh
# Lintel install script for Linux and macOS.
#
# Usage (pipe to sh):
#   curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh
#
# Usage (run locally with flags):
#   ./install.sh [--version vX.Y.Z] [--install-dir /path] [--no-cosign]
#
# Configuration via environment (overridden by flags):
#   LINTEL_VERSION      release tag to install (default: latest)
#   LINTEL_INSTALL_DIR  destination directory   (default: /usr/local/bin,
#                      falling back to $HOME/.local/bin when that is not
#                      writable and no sudo is available)
#   LINTEL_VERIFY_COSIGN   auto|true|false (default auto: verify when cosign
#                         is on PATH, skip otherwise)
#
# The script verifies the SHA256 of the downloaded binary against the
# .sha256 sidecar shipped with every release. If `cosign` is installed,
# it additionally verifies the Sigstore bundle against the GitHub
# Actions OIDC issuer for the lintel repo.

set -eu

REPO="MHChlagou/lintel"
LINTEL_VERSION="${LINTEL_VERSION:-latest}"
LINTEL_INSTALL_DIR="${LINTEL_INSTALL_DIR:-/usr/local/bin}"
LINTEL_VERIFY_COSIGN="${LINTEL_VERIFY_COSIGN:-auto}"

usage() {
  cat <<EOF
Install Lintel.

Flags:
  --version <tag>          release tag (e.g. v0.2.3). Default: latest.
  --install-dir <path>     destination directory. Default: /usr/local/bin.
  --no-cosign              skip cosign signature verification.
  -h, --help               show this help.

The SHA256 check is never skipped.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version)      LINTEL_VERSION="$2"; shift 2 ;;
    --install-dir)  LINTEL_INSTALL_DIR="$2"; shift 2 ;;
    --no-cosign)    LINTEL_VERIFY_COSIGN=false; shift ;;
    -h|--help)      usage; exit 0 ;;
    *) echo "unknown flag: $1" >&2; usage >&2; exit 1 ;;
  esac
done

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "lintel-install: required tool not found: $1" >&2
    exit 1
  }
}
require curl
require uname
require mktemp

# Detect os/arch and map to the names used in release asset filenames.
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) echo "lintel-install: unsupported OS: $os" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)   arch=amd64 ;;
  aarch64|arm64)  arch=arm64 ;;
  *) echo "lintel-install: unsupported arch: $arch" >&2; exit 1 ;;
esac

asset="lintel-${os}-${arch}"
if [ "$LINTEL_VERSION" = "latest" ]; then
  base="https://github.com/${REPO}/releases/latest/download"
else
  base="https://github.com/${REPO}/releases/download/${LINTEL_VERSION}"
fi

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t lintel-install)
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT INT TERM

download() {
  # $1 = url, $2 = dest
  curl -fsSL --retry 3 --retry-delay 2 "$1" -o "$2"
}

echo "↓ downloading ${base}/${asset}"
download "${base}/${asset}"         "${tmp}/lintel"
download "${base}/${asset}.sha256"  "${tmp}/lintel.sha256"

# Pick whichever sha256 tool is available.
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "${tmp}/lintel" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "${tmp}/lintel" | awk '{print $1}')
else
  echo "lintel-install: no sha256 tool found (install sha256sum or shasum)" >&2
  exit 1
fi
expected=$(awk '{print $1}' "${tmp}/lintel.sha256")

if [ "$expected" != "$actual" ]; then
  echo "lintel-install: SHA256 mismatch — refusing to install" >&2
  echo "  expected: $expected" >&2
  echo "  actual:   $actual" >&2
  exit 1
fi
echo "✓ sha256 verified"

# Cosign is optional. In auto mode we only verify when it is already on
# $PATH; asking the user to install it just to run this script would
# push most users to --no-cosign, which defeats the point.
if [ "$LINTEL_VERIFY_COSIGN" = "auto" ]; then
  if command -v cosign >/dev/null 2>&1; then
    LINTEL_VERIFY_COSIGN=true
  else
    LINTEL_VERIFY_COSIGN=false
    echo "• cosign not installed; skipping signature verification (install cosign for stronger guarantees)"
  fi
fi

if [ "$LINTEL_VERIFY_COSIGN" = "true" ]; then
  command -v cosign >/dev/null 2>&1 || {
    echo "lintel-install: --cosign/LINTEL_VERIFY_COSIGN=true but cosign is not on PATH" >&2
    exit 1
  }
  download "${base}/${asset}.sigstore" "${tmp}/lintel.sigstore"
  cosign verify-blob \
    --bundle "${tmp}/lintel.sigstore" \
    --certificate-identity-regexp="^https://github.com/${REPO}/" \
    --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
    "${tmp}/lintel" >/dev/null
  echo "✓ cosign signature verified"
fi

chmod +x "${tmp}/lintel"

# Install. Try without sudo first; fall back to sudo if the dir is
# root-owned, or to $HOME/.local/bin if sudo is unavailable.
dest_dir="$LINTEL_INSTALL_DIR"
dest="${dest_dir}/lintel"

try_install() {
  # $1 = maybe_sudo ("" or "sudo")
  mkdir -p "$dest_dir" 2>/dev/null || $1 mkdir -p "$dest_dir"
  $1 mv "${tmp}/lintel" "$dest"
}

if [ -w "$dest_dir" ] 2>/dev/null || mkdir -p "$dest_dir" 2>/dev/null && [ -w "$dest_dir" ]; then
  try_install ""
  echo "✓ installed $dest"
elif command -v sudo >/dev/null 2>&1; then
  echo "• $dest_dir is not writable; using sudo"
  try_install sudo
  echo "✓ installed $dest"
else
  # No sudo, no write access — retry at a user-local path.
  fallback="${HOME}/.local/bin"
  echo "• $dest_dir is not writable and sudo is unavailable; falling back to $fallback"
  dest_dir="$fallback"
  dest="${dest_dir}/lintel"
  try_install ""
  echo "✓ installed $dest"
  case ":$PATH:" in
    *:"$dest_dir":*) ;;
    *) echo "  note: $dest_dir is not on \$PATH — add it to your shell profile" ;;
  esac
fi

echo
"$dest" version
