# FastPVE Plus

PVE 一键装机工具，20+ 系统可选。在 Proxmox VE 宿主机上跑一行命令，按提示选系统、配 CPU/内存/磁盘，自动下载镜像并创建虚拟机。

论坛讨论：https://www.koolcenter.com/t/topic/7777

## 安装

**海外 / 直连**：

```bash
bash -c "$(curl -sSL https://raw.githubusercontent.com/solider245/fastpve/main/dd-install.sh)"
```

**国内 / 代理**：

```bash
bash -c "$(curl -sSL https://gh.565600.xyz/https://raw.githubusercontent.com/solider245/fastpve/main/dd-install.sh)"
```

脚本自动下载 FastPVE Plus 二进制（SHA256 校验），直连失败自动走代理，然后拉起交互菜单。

## 支持的系统

### 内置安装（官方方式）

| 系统 | 说明 |
|------|------|
| iStoreOS | 软路由，官方镜像导入 |
| Windows 11/10/7 | ISO + VirtIO 驱动，UEFI/SeaBIOS 自适应 |
| Ubuntu 22.04/24.10/25.04 | 桌面版 / 服务器版 |
| Docker | PVE 宿主机直接安装 |

### DD 镜像安装（20+ 系统）

底层统一走 DD 镜像导入。选择「DD-安装更多系统」→ 选分类 → 选系统 → 自动下载安装。

**软路由 / 防火墙**

| 系统 | BIOS | 说明 |
|------|------|------|
| OpenWRT 官方 | UEFI | 开源路由系统 |
| ImmortalWrt | UEFI | OpenWRT 社区增强分支 |
| iStoreOS DD版 | UEFI | KoolCenter 软路由 |
| RouterOS CHR | BIOS | MikroTik 路由系统 |
| pfSense CE | UEFI | FreeBSD 企业级防火墙 |
| OPNsense | UEFI | pfSense 分支，UI 更现代 |

**NAS / 存储**

| 系统 | BIOS | 说明 |
|------|------|------|
| 群晖 DSM (RR) | UEFI | Redpill Recovery 引导 |
| TrueNAS Scale | UEFI | 开源企业级 NAS |
| OpenMediaVault | UEFI | Debian 系轻量 NAS |
| 飞牛私有云 | UEFI | 国产 NAS，照片管理出色 |

**Windows**

| 系统 | BIOS | 说明 |
|------|------|------|
| Windows Server 2025 | UEFI | 数据中心版 |
| Windows Server 2022 | UEFI | 数据中心版 |
| Windows 11 LTSC | UEFI | 精简长期版 |
| Windows 10 LTSC | UEFI | 最稳定精简版 |

**Linux 服务器**

| 系统 | BIOS | 说明 |
|------|------|------|
| Debian 12 | UEFI | 通用服务器 |
| Rocky Linux 9 | UEFI | RHEL 兼容 |
| Arch Linux | UEFI | 滚动更新 |
| Alpine Linux | UEFI | 超轻量 |

**其他**

| 系统 | BIOS | 说明 |
|------|------|------|
| Home Assistant OS | UEFI | 智能家居中枢 |
| 自定义 URL | 自选 | 手动输入任意 DD 镜像地址 |

## 镜像来源

内置系统优先从官方地址下载，DD 系统优先使用各项目官方 Release 镜像。所有镜像均不做修改。

## 编译

```bash
git clone https://github.com/solider245/fastpve.git
cd fastpve
make build          # FastPVE Plus
make download       # fastpve-download
```

## 免责声明

本存储库旨在方便安装虚拟机。使用者应遵守所在司法辖区法律法规及相关平台服务条款，对自身使用行为承担全部责任。本存储库按"现状（AS IS）"提供，不提供任何明示或暗示担保。
