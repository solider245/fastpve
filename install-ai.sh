#!/bin/bash
# FastPVE AI 模式安装脚本
# 自动安装 zsh + oh-my-zsh + 配置 seamless AI

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

info "FastPVE AI 模式安装开始"

# 1. 确保 fastpve 已安装
if ! command -v fastpve &>/dev/null; then
    error "未找到 fastpve 命令，请先安装 FastPVE"
    error "下载地址: https://github.com/solider245/fastpve/releases"
    exit 1
fi

# 2. 检查 zsh，如果没有则安装
if ! command -v zsh &>/dev/null; then
    info "正在安装 zsh..."
    apt-get update -qq && apt-get install -y -qq zsh
fi

CURRENT_SHELL=$(basename "$SHELL")
if [ "$CURRENT_SHELL" != "zsh" ]; then
    info "当前 shell 是 $CURRENT_SHELL，建议切换到 zsh"
    info "运行: chsh -s $(which zsh)"
    info "然后退出重新登录"
fi

# 3. 安装 oh-my-zsh（如果没有）
if [ ! -d "$HOME/.oh-my-zsh" ]; then
    info "正在安装 oh-my-zsh..."
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended
fi

# 4. 安装 FastPVE zsh 插件
PLUGIN_DIR="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/plugins/fastpve"
mkdir -p "$PLUGIN_DIR"
cp "$(dirname "$0")/fastpve.plugin.zsh" "$PLUGIN_DIR/"
info "zsh 插件已安装到 $PLUGIN_DIR"

# 5. 配置 .zshrc 启用插件
if ! grep -q "fastpve" "$HOME/.zshrc" 2>/dev/null; then
    sed -i 's/plugins=(/plugins=(fastpve /' "$HOME/.zshrc"
    info "已在 .zshrc 中启用 fastpve 插件"
fi

# 6. 配置 DeepSeek API Key
if [ -f "/etc/fastpve/config.conf" ] && grep -q "^AI_KEY=" /etc/fastpve/config.conf 2>/dev/null; then
    info "检测到已配置 AI_KEY，跳过配置"
else
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  配置 DeepSeek API Key"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "注册地址: https://platform.deepseek.com"
    echo "输入后保存在 /etc/fastpve/config.conf"
    echo ""
    read -p "请输入 DeepSeek API Key (输入空跳过): " api_key
    if [ -n "$api_key" ]; then
        mkdir -p /etc/fastpve
        if [ -f "/etc/fastpve/config.conf" ]; then
            # 更新已有配置
            if grep -q "^#\?AI_KEY=" /etc/fastpve/config.conf; then
                sed -i "s|^#\?AI_KEY=.*|AI_KEY=$api_key|" /etc/fastpve/config.conf
            else
                echo "AI_KEY=$api_key" >> /etc/fastpve/config.conf
            fi
        else
            echo "# FastPVE 配置文件" > /etc/fastpve/config.conf
            echo "AI_KEY=$api_key" >> /etc/fastpve/config.conf
        fi
        info "API Key 已保存"

        # 测试连接
        echo ""
        echo "是否测试 AI 连接？(Y/n)"
        read -r test_conn
        if [ "$test_conn" != "n" ] && [ "$test_conn" != "N" ]; then
            echo "" | fastpve ai "你好" 2>/dev/null || echo "测试完成（如果没看到错误信息就是连接成功）"
        fi
    else
        warn "跳过 API Key 配置，AI 功能暂时不可用"
        warn "稍后可编辑 /etc/fastpve/config.conf 手动添加 AI_KEY="
    fi
fi

echo ""
info "安装完成！"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  使用方法"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  直接在 shell 中打字:"
echo "    几个VM在跑？          → AI 自动回答"
echo "    帮我装个Ubuntu        → AI 执行创建"
echo "    看看存储              → AI 显示概览"
echo "  (当前 session 需先执行: source ~/.zshrc)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
