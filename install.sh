#!/usr/bin/env bash
# Installs the latest warpping release for Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/YOUR_GITHUB_USERNAME/warpping/main/install.sh | bash
#
set -euo pipefail

REPO="YOUR_GITHUB_USERNAME/warpping"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

echo "Fetching latest warpping release info..."
latest_tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep -m1 '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"

if [ -z "$latest_tag" ]; then
  echo "Could not determine the latest release tag." >&2
  exit 1
fi

version="${latest_tag#v}"
asset="warpping_${version}_linux_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${latest_tag}/${asset}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading ${asset} (${latest_tag})..."
curl -fsSL "$url" -o "${tmpdir}/${asset}"

echo "Extracting..."
tar -xzf "${tmpdir}/${asset}" -C "$tmpdir"

echo "Installing to ${INSTALL_DIR}/warpping (may prompt for sudo)..."
if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "${tmpdir}/warpping" "${INSTALL_DIR}/warpping"
else
  sudo install -m 0755 "${tmpdir}/warpping" "${INSTALL_DIR}/warpping"
fi

echo "Installed: $(${INSTALL_DIR}/warpping -h 2>&1 | head -1 || true)"
echo "Run 'warpping' to start."
