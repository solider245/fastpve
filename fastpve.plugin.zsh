# FastPVE ZSH 插件 — 无感 AI 集成
# 安装：放到 ~/.oh-my-zsh/custom/plugins/fastpve/ 或 source 本文件

# 核心：拦截 accept-line 小部件
fastpve-accept-line() {
    # 获取当前输入
    local cmd="$BUFFER"

    # 空输入直接换行
    if [[ -z "$cmd" ]]; then
        zle .accept-line
        return
    fi

    # 获取第一个词（命令名）
    local first_word="${cmd%% *}"

    # 检查是否是合法命令（type 返回成功=命令存在）
    if type "$first_word" >/dev/null 2>&1 || [[ "$first_word" =~ ^[./] ]]; then
        # 合法命令：正常执行
        zle .accept-line
    else
        # 非合法命令：清空缓冲区，交给 fastpve ai
        BUFFER=""
        zle .accept-line
        echo "$cmd" | fastpve ai 2>/dev/null
        # 如果 fastpve 不在 PATH 中或执行失败，显示友好提示
        if [[ $? -ne 0 ]]; then
            echo "FastPVE AI 不可用，请运行: fastpve 或检查安装"
        fi
    fi
}

# 替换默认 accept-line 小部件
zle -N accept-line fastpve-accept-line
