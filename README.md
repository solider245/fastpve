# FastPVE Plus

Proxmox VE 一站式运维工具 — 从装机到管理，从日常维护到智能诊断。

```text
█   █████  █████  █████ █████ █████ █████
██  █   █ █     █   █   █     █     █
█ █ █   █ █     █   █   ████  ████   ███
█  █ █   █ █     █   █   █     █         █
█   █ █████  █████   █   █████ █████ █████
```

论坛讨论：https://www.koolcenter.com/t/topic/7777

---

## 功能速览

| 模块 | 说明 |
|------|------|
| 🖥️ **VM/CT 创建** | 一键 DD 安装 20+ 系统（Win/Linux/路由/NAS） |
| 🔧 **PVE 系统工具** | 换源 / Docker / 核显直通 / 去弹窗 / 清理 / 网络诊断 |
| 🤖 **AI 助手** | 自然语言管理 PVE，DeepSeek/Claude/Ollama 驱动 |
| 📊 **性能监控** | 后台采集 CPU/内存/磁盘/VM 趋势，AI 自动分析 |
| 📋 **审计日志** | 所有操作自动记录，可追溯查询 |
| 🔌 **PI 协议** | JSON stdin/stdout 接口，供外部工具调用 |

---

## 安装

**海外（直连）**：

```bash
bash -c "$(curl -sSL https://raw.githubusercontent.com/solider245/fastpve/main/dd-install.sh)"
```

**国内（代理，推荐）**：

```bash
bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install-cn.sh)"
```

安装后直接运行：

```bash
fastpve
```

---

## 🔧 PVE 系统工具

主菜单选择「PVE系统工具」进入，覆盖 PVE 日常运维所有高频操作：

### 系统维护
- **更换软件源** — 一键切换到国内镜像（中科大/清华/华为）
- **系统更新** — `apt update && apt dist-upgrade`
- **清理旧内核** — 列出可清理内核、确认后自动清理 + update-grub
- **安装常用工具** — curl / wget / vim / htop / net-tools / lsof

### 存储
- **存储概览** — ZFS 池 / 磁盘使用 / ARC 缓存 / pvesh 存储清单
- **备份 /etc/pve** — 打包备份 PVE 配置
- **清理 journal 日志** — 自定义保留大小

### 网络
- **备份网络配置** — 备份 `/etc/network/interfaces`
- **网络连通性检测** — 网关 / WAN Ping / DNS / MTU 全面检查
- **Fail2ban 配置** — 暴力破解防护
- **Let's Encrypt 证书** — pvenode ACME 一键申请

### 虚拟化
- **直通环境检测** — IOMMU / 内核参数 / GPU 分组 / 黑名单 / vfio 6 步检查
- **一键核显直通** — Intel/AMD GPU 直通配置
- **LXC 权限修复** — 提权 / UID 映射修复
- **Docker 安装** — PVE 宿主机直接安装 Docker

### 诊断工具
- btop / ncdu / iperf3 / pveperf / mtr / speedtest-cli / systemd-cgtop

### 配置
- **编辑配置** — 使用 `$EDITOR` 编辑 `/etc/fastpve/config.conf`

---

## 🤖 AI 助手

AI 助手让你用自然语言管理 PVE，不需要记命令、找菜单。

### 配置

编辑 `/etc/fastpve/config.conf`，填入 API Key：

```ini
AI_URL=https://api.deepseek.com/v1/chat/completions
AI_KEY=sk-your-key-here
AI_MODEL=deepseek-chat
```

配置后主菜单自动显示「AI 助手」入口。AI 默认使用 DeepSeek，兼容 OpenAI 格式，可换成 Claude、Ollama 等任意 OpenAI 兼容 API。

### 使用方式

**方式一：终端菜单内交互（推荐）**

主菜单 → AI 助手 → 「AI对话」，输入你的需求：

```
我: 看一下系统状态
🤖 好的，正在检查...
✅ CPU 负载 0.5，内存 45%，磁盘 / 62%，一切正常

我: 帮我更新系统
🤖 即将执行: 更新系统 (apt update && apt dist-upgrade)
确认执行? (y/N): y
✅ 系统已更新
```

**方式二：命令行直接调用**

```bash
fastpve ai 查看系统状态
fastpve ai 帮我备份网络配置
echo "还有多少空闲资源？" | fastpve ai
```

**方式三：zsh 插件 — 无感体验**

安装 zsh + oh-my-zsh + fastpve 插件后，在终端直接输入自然语言：

```bash
# 输错命令？自动转 AI
$ vim /etc/network/interfaces
zsh: command not found: vim
→ fastpve.ai: 看起来你想编辑网络配置，需要我帮你备份并打开编辑吗？

# 直接说需求
$ 查看虚拟机列表
  VMID NAME        STATUS
  100  ubuntu-srv  running
  101  win11       running
```

安装方式：

```bash
# 方法一：通过 AI 助手一键安装
fastpve ai 帮我安装 zsh 和 AI 插件

# 方法二：手动运行安装脚本
bash install-ai.sh
```

### 内置工具

| 工具 | 说明 | 自动执行 |
|------|------|----------|
| `health_check` | 系统健康状态（CPU/内存/磁盘/VM/CT） | ✅ |
| `storage_overview` | ZFS/磁盘/存储概览 | ✅ |
| `vm_list` | 列出所有虚拟机 | ✅ |
| `lxc_list` | 列出所有 LXC 容器 | ✅ |
| `query_perf_history` | 性能趋势查询（1h/6h/24h/7d） | ✅ |
| `query_audit_log` | 操作审计追溯 | ✅ |
| `backup_network` | 备份网络配置 | ❌ 需确认 |
| `remove_sub_nag` | 去除订阅弹窗 | ❌ 需确认 |
| `system_update` | 更新系统 | ❌ 需确认 |
| `install_tools` | 安装常用工具 | ❌ 需确认 |

**安全设计**：查询类工具自动执行，操作类工具执行前要求 `y/N` 确认，防止误操作。

---

## 🔌 PI 协议

PI 协议提供 JSON 标准输入/输出接口，供外部 AI 前端或第三方工具调用 FastPVE 后端能力。

```bash
echo '{"tool":"health_check","args":{}}' | fastpve pi
# {"ok":true,"data":"=== FastPVE 系统状态 ===\\n时间: ..."}
```

可用于：
- 自定义 AI 前端对接
- CI/CD 自动化流程
- 远程管理脚本

---

## ⚙️ 配置

配置文件 `/etc/fastpve/config.conf`，修改后重启 `fastpve` 生效。

**配置即开关**：所有选项默认注释，取消注释即启用对应功能。

```ini
# ========== AI 助手 ==========
# AI_URL=https://api.deepseek.com/v1/chat/completions
# AI_KEY=sk-your-key-here
# AI_MODEL=deepseek-chat

# ========== 通知 (Telegram) ==========
# NOTIFY_TELEGRAM_TOKEN=123456:ABC-DEF
# NOTIFY_TELEGRAM_CHAT_ID=123456789

# ========== 默认虚拟机配置 ==========
# VM_CORES=2
# VM_MEMORY=2048
# VM_DISK=64
# VM_BRIDGE=vmbr0
# VM_STORAGE=local

# ========== HOOK 脚本 ==========
# 创建 VM/CT 前后自动执行
# HOOK_PRE_CREATE=/usr/local/bin/fastpve-pre-create.sh
# HOOK_POST_CREATE=/usr/local/bin/fastpve-post-create.sh
```

---

## 📦 兼容系统

### DD 镜像安装（20+ 系统）

| 分类 | 系统 | BIOS |
|------|------|------|
| 路由/防火墙 | OpenWRT / ImmortalWrt / iStoreOS / RouterOS / pfSense / OPNsense | UEFI |
| NAS/存储 | 群晖 DSM / TrueNAS Scale / OpenMediaVault / 飞牛 | UEFI |
| Windows | Server 2025/2022 / 11 LTSC / 10 LTSC | UEFI |
| Linux | Debian 12 / Rocky Linux 9 / Arch / Alpine | UEFI |
| 其他 | Home Assistant OS / 自定义 URL | 自选 |

### 内置安装

| 系统 | 说明 |
|------|------|
| iStoreOS | 软路由，官方镜像导入 |
| Windows 11/10/7 | ISO + VirtIO 驱动，UEFI/SeaBIOS 自适应 |
| Ubuntu 22.04/24.10/25.04 | 桌面版 / 服务器版 |
| Docker | PVE 宿主机直接安装 |

---

## 📊 架构

```text
┌─────────────────────────────────────────────┐
│              终端 / zsh 插件                  │
└──────────────────┬──────────────────────────┘
                   │
           ┌───────┴───────┐
           │   fastpve      │
           │   TUI 主菜单    │
           └───┬───┬───┬───┘
               │   │   │
    ┌──────────┘   │   └──────────┐
    ▼              ▼              ▼
┌─────────┐ ┌──────────┐ ┌──────────────┐
│ VM/CT   │ │ PVE 系统 │ │  AI 助手     │
│ 创建    │ │ 工具     │ │  / PI 协议   │
└─────────┘ └──────────┘ └──────┬───────┘
                                │
                     ┌──────────┴──────────┐
                     │      SQLite          │
                     │ 对话历史 / 性能趋势   │
                     │ 审计日志 / 配置      │
                     └─────────────────────┘
```

---

## 编译

```bash
git clone https://github.com/solider245/fastpve.git
cd fastpve
make build          # FastPVE Plus
make download       # fastpve-download
```

## 免责声明

本存储库旨在方便安装和管理虚拟机。使用者应遵守所在司法辖区法律法规及相关平台服务条款，对自身使用行为承担全部责任。本存储库按"现状（AS IS）"提供，不提供任何明示或暗示担保。
