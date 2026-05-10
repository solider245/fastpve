#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — China proxy mirrors
# Usage: bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"

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
		for u in "https://gh.565600.xyz/https://github.com/${REPO}/releases/download/latest/${n}" \
		         "https://gh.linkease.net:5443/${REPO}/releases/download/latest/${n}" \
		         "https://github.com/${REPO}/releases/download/latest/${n}"; do
			if curl -fSL --progress-bar -o "$f" "$u" 2>/dev/null; then
				chmod +x "$f"; return 0
			fi
		done
		return 1
	fi
}
echo "[INFO] 首次运行，下载 FastPVE ..."
DL "FastPVE" && DL "fastpve-download" || { echo "[ERROR] 下载失败"; exit 1; }

exec "$DIR/FastPVE" "$@"
LAUNCHER

chmod +x "$STUB"
echo "安装完成，运行 fastpve 开始使用"
