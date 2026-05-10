#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — 国内（代理）+ 多回退
# Usage: bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"

REPO="solider245/fastpve"
INSTALL_DIR="/usr/local/bin"

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
		info "fallback: ${u}"
	done
	return 1
}

if [[ -x "${INSTALL_DIR}/fastpve" && -x "${INSTALL_DIR}/fastpve-download" ]]; then
	info "检测到已安装，直接启动..."
	exec "${INSTALL_DIR}/fastpve"
fi

info "downloading FastPVE (~25MB) ..."
download "FastPVE"         || { echo "[ERROR] 所有下载源均失败"; exit 1; }
download "fastpve-download" || { echo "[ERROR] 所有下载源均失败"; exit 1; }
chmod +x "${INSTALL_DIR}/fastpve" "${INSTALL_DIR}/fastpve-download"

info "FastPVE Plus (latest) — 20+ 系统一键安装"
exec "${INSTALL_DIR}/fastpve"
