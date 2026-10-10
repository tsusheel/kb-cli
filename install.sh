#!/usr/bin/env bash
# Knowledge Base CLI (kb) Universal Installer
# Works on all Linux distributions (Ubuntu, Debian, Fedora, Arch, Alpine, CentOS, openSUSE, etc.) and macOS
set -euo pipefail

REPO="tsusheel/kb-cli"
BINARY_NAME="kb"

echo "=== Knowledge Base CLI (kb) Installer ==="

# Check download tool
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget"
else
    echo "Error: Neither 'curl' nor 'wget' was found. Please install one of them first." >&2
    exit 1
fi

http_get() {
    local url="$1"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url"
    else
        wget -qO- "$url"
    fi
}

download_file() {
    local url="$1"
    local dest="$2"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url" -o "$dest"
    else
        wget -qO "$dest" "$url"
    fi
}

# 1. Detect Operating System
OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_NAME="linux" ;;
    Darwin*)    OS_NAME="darwin" ;;
    *)
        echo "Error: Operating system '${OS}' is not supported by this installer script." >&2
        echo "Please download the binary manually from https://github.com/${REPO}/releases" >&2
        exit 1
        ;;
esac

# 2. Detect CPU Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64)   ARCH_NAME="amd64" ;;
    aarch64|arm64)  ARCH_NAME="arm64" ;;
    *)
        echo "Error: Architecture '${ARCH}' is not supported by this installer script." >&2
        echo "Please download or build from source: https://github.com/${REPO}" >&2
        exit 1
        ;;
esac

echo "Detected OS: ${OS_NAME} (${ARCH_NAME})"

# 3. Fetch latest release version tag
echo "Fetching latest release information..."
LATEST_JSON="$(http_get "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
TAG="$(echo "${LATEST_JSON}" | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')"

if [ -z "${TAG}" ]; then
    echo "Error: Could not query GitHub API for the latest release tag." >&2
    echo "Please visit https://github.com/${REPO}/releases directly." >&2
    exit 1
fi

VERSION="${TAG#v}"
echo "Latest version: ${TAG}"

# 4. Construct download archive URL
ARCHIVE_NAME="kb_${VERSION}_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"

# 5. Download and extract
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'kb-install')"
cleanup() {
    rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

echo "Downloading ${ARCHIVE_NAME}..."
download_file "${DOWNLOAD_URL}" "${TMP_DIR}/${ARCHIVE_NAME}"

echo "Extracting..."
tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}"

if [ ! -f "${TMP_DIR}/${BINARY_NAME}" ]; then
    echo "Error: Binary '${BINARY_NAME}' not found in downloaded archive." >&2
    exit 1
fi

chmod +x "${TMP_DIR}/${BINARY_NAME}"

# 6. Install to appropriate bin directory
INSTALL_DIR="/usr/local/bin"

if [ -w "${INSTALL_DIR}" ]; then
    cp "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
elif command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    echo "Root privileges needed to install to ${INSTALL_DIR}."
    sudo cp "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    sudo chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
    cp "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"

    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            echo ""
            echo "Notice: ${INSTALL_DIR} is not in your current PATH."
            echo "Add this to your shell profile (~/.bashrc or ~/.zshrc):"
            echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
            ;;
    esac
fi

echo ""
echo "kb installed successfully to ${INSTALL_DIR}/${BINARY_NAME}!"
echo ""
echo "Get started by running:"
echo "  kb --help"
