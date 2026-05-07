#!/bin/bash
set -euo pipefail

# FastPVE Plus — 国内安装脚本（代理优先）
# Usage: bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"

VERSION="0.1.9-dd1"
REPO="solider245/fastpve"
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"
KSPEEDER="https://gh.linkease.net:5443"
FALLBACK_PROXY="https://gh.565600.xyz/"
TEMP_DIR="${TEMP_DIR:-/tmp}"
INSTALL_DIR="/usr/local/bin"

BINARY="FastPVE-${VERSION}"
DOWNLOADER="fastpve-download"

SHA_BIN="64df3cb1c3e0acc17caee6a5abb80fb6fc12b91d5598e83762aeb46ad1efd42f"
SHA_DL="ae32ac55a5f604a5c5152b9d9ff77eedaae366d49a7b8199f04e5f63cafed705"

info()  { echo -e "\033[1;34m[INFO]\033[0m $*"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $*" >&2; }

if [[ "$(uname -s)" != "Linux" ]]; then
    error "FastPVE only runs on Linux (Proxmox VE)."
    exit 1
fi

download_file() {
    local url=$1 dest=$2
    # KSspeeder local proxy (preferred), then public fallback
    local ks_url="${KSPEEDER}/${REPO}/releases/download/v${VERSION}/$(basename "$dest")"
    local fb_url="${FALLBACK_PROXY}${url}"

    info "downloading $(basename "$dest") ..."
    if command -v curl &>/dev/null; then
        curl -fL --connect-timeout 3 --max-time 600 --progress-bar -o "$dest" "$ks_url" || \
        (info "retrying via fallback proxy..." && \
         curl -fL --connect-timeout 5 --max-time 600 --progress-bar -o "$dest" "$fb_url")
    elif command -v wget &>/dev/null; then
        wget -q --timeout=3 --show-progress -O "$dest" "$ks_url" || \
        (info "retrying via fallback proxy..." && \
         wget -q --timeout=5 --show-progress -O "$dest" "$fb_url")
    else
        error "curl or wget is required."
        exit 1
    fi
}

verify_sha() {
    local file=$1 expected=$2 actual
    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        error "sha256sum or shasum is required."
        exit 1
    fi
    if [[ "$actual" != "$expected" ]]; then
        error "checksum mismatch for $(basename "$file")"
        rm -f "$file"
        exit 1
    fi
}

# Download main binary
BIN_PATH="${TEMP_DIR}/${BINARY}"
if [[ ! -f "$BIN_PATH" ]]; then
    download_file "${BASE_URL}/${BINARY}" "$BIN_PATH"
    verify_sha "$BIN_PATH" "$SHA_BIN"
    chmod +x "$BIN_PATH"
fi

# Download helper binary
DL_PATH="${TEMP_DIR}/${DOWNLOADER}"
if [[ ! -f "$DL_PATH" ]]; then
    download_file "${BASE_URL}/${DOWNLOADER}" "$DL_PATH"
    verify_sha "$DL_PATH" "$SHA_DL"
    chmod +x "$DL_PATH"
fi

# Install to system PATH
info "installing to ${INSTALL_DIR} ..."
cp "$BIN_PATH" "${INSTALL_DIR}/fastpve"
cp "$DL_PATH" "${INSTALL_DIR}/fastpve-download"
info "install complete: fastpve, fastpve-download"

info "FastPVE Plus v${VERSION} — 20+ 系统一键安装"
info "starting interactive menu..."
echo ""
exec "${INSTALL_DIR}/fastpve"
