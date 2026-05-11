#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — puts 'fastpve' launcher on PATH
# Usage: bash -c "$(curl -sSL https://raw.githubusercontent.com/solider245/fastpve/main/dd-install.sh)"

INSTALL_DIR="/usr/local/bin"
STUB="${INSTALL_DIR}/fastpve"

# skip if already installed (launcher exists)
[[ -f "$STUB" ]] && exit 0

REPO="solider245/fastpve"
DIR="/var/lib/vz/template/cache"
mkdir -p "$DIR"

cat > "$STUB" << 'LAUNCHER'
#!/bin/bash
set -euo pipefail
DIR="/var/lib/vz/template/cache"

DL() { local n="$1" f="$DIR/$n"
	if [[ ! -x "$f" ]]; then
		REPO="solider245/fastpve"
		for u in "https://github.com/${REPO}/releases/download/latest/${n}" \
		         "https://gh.linkease.net:5443/${REPO}/releases/download/latest/${n}" \
		         "https://gh.565600.xyz/https://github.com/${REPO}/releases/download/latest/${n}"; do
			if curl -fSL --progress-bar -o "$f" "$u" 2>/dev/null; then
				chmod +x "$f"; return 0
			fi
		done
		return 1
	fi
}
echo "[INFO] 首次运行，正在下载 (~25MB) ..."
if ! DL "FastPVE"; then
    echo "[ERROR] FastPVE 下载失败，所有镜像源均不可用"
    echo "  请手动下载: curl -fsSL https://github.com/solider245/fastpve/releases/download/latest/FastPVE -o /usr/local/bin/fastpve"
    exit 1
fi
if ! DL "fastpve-download"; then
    echo "[ERROR] fastpve-download 下载失败"
    echo "  请手动下载: curl -fsSL https://github.com/solider245/fastpve/releases/download/latest/fastpve-download -o /usr/local/bin/fastpve-download"
    exit 1
fi
echo "[INFO] 下载完成"

exec "$DIR/FastPVE" "$@"
LAUNCHER

chmod +x "$STUB"
echo "安装完成，运行 fastpve 开始使用"
