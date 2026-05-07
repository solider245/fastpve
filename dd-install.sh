#!/bin/bash
set -euo pipefail

# FastPVE Plus — 海外安装脚本（直连优先）
# Usage: bash -c "$(curl -sSL https://raw.githubusercontent.com/solider245/fastpve/main/dd-install.sh)"

REPO="solider245/fastpve"
BASE_URL="https://github.com/${REPO}/releases/download/latest"
PROXY="https://gh.linkease.net:5443"
FALLBACK_PROXY="https://gh.565600.xyz/"
TEMP_DIR="${TEMP_DIR:-/tmp}"
INSTALL_DIR="/usr/local/bin"

info()  { echo -e "\033[1;34m[INFO]\033[0m $*"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $*" >&2; }

if [[ "$(uname -s)" != "Linux" ]]; then
    error "FastPVE only runs on Linux (Proxmox VE)."
    exit 1
fi

download_link() {
    local name=$1 out=$2
    local url="${BASE_URL}/${name}"
    local ks_url="${PROXY}/${REPO}/releases/download/latest/${name}"
    local fb_url="${FALLBACK_PROXY}${url}"

    if command -v curl &>/dev/null; then
        info "downloading ${name} ..."
        curl -fSL --connect-timeout 3 --max-time 30 --progress-bar -o "$out" "$url" && return 0
        info "direct failed, trying KSpeeder..."
        curl -fSL --connect-timeout 5 --max-time 600 --progress-bar -o "$out" "$ks_url" && return 0
        info "retrying via fallback..."
        curl -fSL --connect-timeout 5 --max-time 600 --progress-bar -o "$out" "$fb_url" && return 0
    elif command -v wget &>/dev/null; then
        info "downloading ${name} ..."
        wget -q --timeout=3 --show-progress -O "$out" "$url" && return 0
        info "direct failed, trying KSpeeder..."
        wget -q --timeout=5 --show-progress -O "$out" "$ks_url" && return 0
        info "retrying via fallback..."
        wget -q --timeout=5 --show-progress -O "$out" "$fb_url" && return 0
    else
        error "curl or wget is required."
        exit 1
    fi
    return 1
}

checksum_verify() {
    local chkfile=$1 name=$2 file=$3 expected
    expected=$(grep "${name}" "$chkfile" | awk '{print $1}')
    if [[ -z "$expected" ]]; then
        error "checksum entry not found for ${name}"
        exit 1
    fi
    local actual
    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        error "sha256sum or shasum is required."
        exit 1
    fi
    if [[ "$actual" != "$expected" ]]; then
        error "checksum mismatch for ${name}"
        rm -f "$file"
        exit 1
    fi
}

# Download checksums
CHK_PATH="${TEMP_DIR}/checksums.txt"
download_link "checksums.txt" "$CHK_PATH" || { error "failed to download checksums"; exit 1; }

# Download main binary
BIN_PATH="${TEMP_DIR}/FastPVE"
download_link "FastPVE" "$BIN_PATH" || { error "failed to download FastPVE"; exit 1; }
checksum_verify "$CHK_PATH" "FastPVE" "$BIN_PATH"
chmod +x "$BIN_PATH"

# Download helper binary
DL_PATH="${TEMP_DIR}/fastpve-download"
download_link "fastpve-download" "$DL_PATH" || { error "failed to download fastpve-download"; exit 1; }
checksum_verify "$CHK_PATH" "fastpve-download" "$DL_PATH"
chmod +x "$DL_PATH"

# Install to system PATH
info "installing to ${INSTALL_DIR} ..."
cp "$BIN_PATH" "${INSTALL_DIR}/fastpve"
cp "$DL_PATH" "${INSTALL_DIR}/fastpve-download"
info "install complete: fastpve, fastpve-download"

info "FastPVE Plus (latest) — 20+ system installer"
info "starting interactive menu..."
echo ""
exec "${INSTALL_DIR}/fastpve"
