#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — China proxy mirrors
# Usage: bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"

INSTALL_DIR="/usr/local/bin"
STUB="${INSTALL_DIR}/fastpve"

# skip if already installed — launch directly
if [[ -f "$STUB" ]]; then
    exec "$STUB" "$@"
fi

REPO="solider245/fastpve"
DIR="/var/lib/vz/template/cache"
mkdir -p "$DIR"

cat > "$STUB" << 'LAUNCHER'
#!/bin/bash
set -euo pipefail
DIR="/var/lib/vz/template/cache"

DL() { local n="$1"; local f="$DIR/$n"; local last_err
	if [[ ! -f "$f" ]] && [[ ! -x "$f" ]]; then
		REPO="solider245/fastpve"
		for u in "https://gh.565600.xyz/https://github.com/${REPO}/releases/download/latest/${n}" \
		         "https://gh.linkease.net:5443/${REPO}/releases/download/latest/${n}" \
		         "https://github.com/${REPO}/releases/download/latest/${n}"; do
			last_err=$(curl -fSL --progress-bar -o "$f" "$u" 2>&1) && { chmod +x "$f"; return 0; }
		done
		echo "下载失败: $last_err" >&2
		return 1
	fi
}
echo "[INFO] 首次运行，正在下载 (~25MB) ..."
if ! DL "FastPVE"; then
    echo "[ERROR] FastPVE 下载失败，所有镜像源均不可用" >&2
    echo "  请手动下载: curl -fsSL https://github.com/solider245/fastpve/releases/download/latest/FastPVE -o /usr/local/bin/fastpve" >&2
    exit 1
fi
if ! DL "fastpve-download"; then
    echo "[ERROR] fastpve-download 下载失败" >&2
    echo "  请手动下载: curl -fsSL https://github.com/solider245/fastpve/releases/download/latest/fastpve-download -o /usr/local/bin/fastpve-download" >&2
    exit 1
fi
echo "[INFO] 下载完成"

exec "$DIR/FastPVE" "$@"
LAUNCHER

chmod +x "$STUB"
echo "安装完成，运行 fastpve 开始使用"
