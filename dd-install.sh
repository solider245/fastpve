#!/bin/bash
set -euo pipefail
# FastPVE Plus installer — puts 'fastpve' launcher on PATH
# Usage: bash -c "$(curl -sSL https://raw.githubusercontent.com/solider245/fastpve/main/dd-install.sh)"

INSTALL_DIR="/usr/local/bin"
STUB="${INSTALL_DIR}/fastpve"
DIR="/var/lib/vz/template/cache"
LOG="/tmp/fastpve-install.log"

log() { echo "$@" | tee -a "$LOG"; }

mkdir -p "$DIR"

# Always regenerate the stub so updates take effect
cat > "$STUB" << 'LAUNCHER'
#!/bin/bash
set -euo pipefail
DIR="/var/lib/vz/template/cache"
LOG="/tmp/fastpve-launcher.log"

log() { echo "$@" >> "$LOG"; }

DL() {
    local n="$1"
    local f="$DIR/$n"
    local last_err
    if [[ ! -f "$f" ]] && [[ ! -x "$f" ]]; then
        REPO="solider245/fastpve"
        for u in "https://github.com/${REPO}/releases/download/latest/${n}" \
                 "https://gh.linkease.net:5443/${REPO}/releases/download/latest/${n}" \
                 "https://gh.565600.xyz/https://github.com/${REPO}/releases/download/latest/${n}"; do
            log "尝试下载 $n 从 $u"
            last_err=$(curl -fSL --progress-bar -o "$f" "$u" 2>&1) && { chmod +x "$f"; log "下载 $n 成功"; return 0; }
            log "失败: $last_err"
        done
        echo "下载失败: $last_err" >&2
        return 1
    fi
    log "缓存命中: $f"
}
echo "[INFO] 首次运行，正在下载 (~25MB) ..."
log "=== 启动 FastPVE 下载 ==="
if ! DL "FastPVE"; then
    echo "[ERROR] FastPVE 下载失败，所有镜像源均不可用" >&2
    exit 1
fi
if ! DL "fastpve-download"; then
    echo "[ERROR] fastpve-download 下载失败" >&2
    exit 1
fi
echo "[INFO] 下载完成"

exec "$DIR/FastPVE" "$@"
LAUNCHER

chmod +x "$STUB"
log "stub 已更新: $STUB"

# Now launch (first-run will download binaries, subsequent runs skip download)
exec "$STUB" "$@"