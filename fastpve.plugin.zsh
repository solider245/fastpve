# FastPVE AI 助手 — zsh 插件
# 命令不存在时自动转 fastpve ai 处理
# 安装：fastpve install-plugin

command_not_found_handler() {
    fastpve ai "$*"
    return 0
}

# 强制使用原始命令（在命令前加反斜杠绕过 AI）
alias forcetype='\'
