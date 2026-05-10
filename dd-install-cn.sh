#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — China proxy mirrors
# Usage: bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"

REPO="solider245/fastpve"
INSTALL_DIR="/usr/local/bin"

# skip if already installed
[[ -x "${INSTALL_DIR}/fastpve" && -x "${INSTALL_DIR}/fastpve-download" ]] && exit 0

info() { echo -e "\033[1;34m[INFO]\033[0m $*"; }

download() {
	local filename="$1"
	local urls=(
		"https://gh.565600.xyz/https://github.com/${REPO}/releases/download/latest/${filename}"
		"https://gh.linkease.net:5443/${REPO}/releases/download/latest/${filename}"
		"https://github.com/${REPO}/releases/download/latest/${filename}"
	)
	for u in "${urls[@]}"; do
		if curl -fSL --progress-bar -o "${INSTALL_DIR}/${filename}" "$u" 2>/dev/null; then
			return 0
		fi
	done
	return 1
}

info "downloading FastPVE ..."
download "FastPVE"         || { echo "[ERROR] 下载失败"; exit 1; }
download "fastpve-download" || { echo "[ERROR] 下载失败"; exit 1; }
chmod +x "${INSTALL_DIR}/fastpve" "${INSTALL_DIR}/fastpve-download"

echo "安装完成，运行 fastpve 开始使用"
