#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — 国内（代理）
# Usage: bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"

REPO="solider245/fastpve"
RAW="https://gh.565600.xyz/https://raw.githubusercontent.com/${REPO}/main/bin"
INSTALL_DIR="/usr/local/bin"

info() { echo -e "\033[1;34m[INFO]\033[0m $*"; }

info "downloading FastPVE ..."
curl -fSL --progress-bar -o "${INSTALL_DIR}/fastpve"         "${RAW}/FastPVE"
curl -fSL --progress-bar -o "${INSTALL_DIR}/fastpve-download" "${RAW}/fastpve-download"
chmod +x "${INSTALL_DIR}/fastpve" "${INSTALL_DIR}/fastpve-download"

info "FastPVE Plus (latest) — 20+ 系统一键安装"
exec "${INSTALL_DIR}/fastpve"
