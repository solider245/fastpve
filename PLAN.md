## 并行开发实施计划

### 总览

```
Batch 1（完全并行，7 个独立任务）
┌─────────────────────────────────────────────────────┐
│ A: prompt_toolbox.go  │ B: prompt_storage.go        │
│ (CLI 工具箱)            │ (存储概览+备份)              │
├─────────────────────────┼────────────────────────────┤
│ C: prompt_network.go   │ D: lxc_services.go 扩展     │
│ (网络诊断+安全)          │ (6 个新服务预设)             │
├─────────────────────────┼────────────────────────────┤
│ E: prompt_for_lxc.go   │ F: prompt_for_gpupass.go    │
│ (LXC 权限修复向导)       │ (GPU 直通排障增强)           │
├─────────────────────────┼────────────────────────────┤
│ G: config.go            │                           │
│ (HOOK 配置项)            │                           │
└─────────────────────────┴────────────────────────────┘

Batch 2（1 个任务，依赖 Batch 1 的函数签名）
┌─────────────────────────────────────────────────────┐
│ H: prompt_system_tools.go（菜单集成）                │
│ 把所有新入口加入 PVE系统工具菜单                       │
└─────────────────────────────────────────────────────┘
```

---

### Batch 1-A: CLI 工具箱（新建 `cmd/fastpve/prompt_toolbox.go`）

**函数签名**（后续菜单集成直接引用这些函数）：
```go
func promptToolbox() error          // 入口
func runBtop() error                // 现代化资源监控
func runNCDU() error                // 磁盘空间分析
func runIperf3() error              // 网络带宽测试
func runPVEPerf() error             // PVE 硬件基线
func runSystemdCgtop() error        // VM/CT 资源占用
func runMTR() error                 // 路由追踪
func runSpeedtest() error           // 外网带宽测试
```

**子菜单结构**：
```
0、btop — 现代化资源监控
1、ncdu — 磁盘空间分析
2、iperf3 — 网络带宽测试
3、pveperf — PVE 硬件性能基线
4、systemd-cgtop — VM/CT 资源占用
5、mtr — 路由追踪诊断
6、speedtest-cli — 外网带宽测试
7、返回
```

**实现要点**：
- 每个工具一键安装（`apt install -y -qq`）+ 即时运行
- 运行完成后按任意键返回菜单
- 使用 `utils.BatchRunStdout` 实时输出

---

### Batch 1-B: 存储概览与备份（新建 `cmd/fastpve/prompt_storage.go`）

**函数签名**：
```go
func promptStorageTools() error     // 入口
func showStorageOverview() error    // 存储概览
func backupPVEConfig() error        // 备份 /etc/pve
func cleanJournalLogs() error       // 清理 journal 日志
```

**存储概览输出示例**：
```
========== 存储概览 ==========
ZFS 池:
  rpool  1.82T  1.2T  620G  /rpool

存储:
  local (dir)       总: 98G  已用: 45G  (46%)
  local-lvm (lvmthin) 总: 892G  已用: 523G  (59%)

磁盘 (df):
  /                 总: 98G  已用: 45G  (46%)
  /var/lib/vz       总: 892G  已用: 523G  (59%)

ZFS ARC: 4.0G / 64.0G (6%)
==============================
```

**实现要点**：
- 调用 `zpool list`、`pvesh get /storage`、`df -h`、`arc_summary` 或 `cat /proc/spl/kstat/zfs/arcstats`
- 备份：`tar czf /root/pve-backup-$(date +%F).tar.gz /etc/pve`
- 日志清理：`journalctl --vacuum-size=500M`

---

### Batch 1-C: 网络诊断与安全（新建 `cmd/fastpve/prompt_network.go`）

**函数签名**：
```go
func promptNetworkTools() error      // 入口
func backupNetworkConfig() error     // 备份网络配置
func checkNetworkConnectivity() error // 连通性检测
func setupLetsEncrypt() error        // Let's Encrypt 证书
func setupFail2ban() error           // fail2ban 一键部署
```

**网络配置备份**：
- `cp /etc/network/interfaces /etc/network/interfaces.bak.$(date +%Y%m%d%H%M)`
- 显示备份路径

**连通性检测**：
- ping 网关、外网、PVE 更新服务器
- MTU 检测 (`ping -M do -s 1472`)
- DNS 解析检测

**Let's Encrypt 引导**：
- 交互式：输入域名、邮箱
- 自动执行：`pvenode acme account register` + `pvenode acme cert order`
- 检测当前证书过期时间

**fail2ban 部署**：
- `apt install -y fail2ban`
- 写入 PVE Web UI 保护配置（`/etc/fail2ban/jail.local`，监控 pveproxy 日志）
- 启动服务

---

### Batch 1-D: LXC 服务预设扩展（修改 `cmd/fastpve/lxc_services.go`）

在 `lxcServicePresets` 切片末尾追加 6 个新预设：

| 服务 | BaseOS | Cores/Mem/Disk | Privileged | 备注 |
|------|--------|---------------|------------|------|
| OpenWRT | ubuntu-24.04 | 1/256/2 | false | `apt install openwrt` 或用 Docker |
| Home Assistant | ubuntu-24.04 | 2/1024/8 | true | Docker: `ghcr.io/home-assistant/home-assistant` |
| Jellyfin | ubuntu-24.04 | 2/2048/16 | true | Docker: `jellyfin/jellyfin` |
| WireGuard | ubuntu-24.04 | 1/256/2 | false | `apt install wireguard` + 交互式配置 |
| Frp | ubuntu-24.04 | 1/256/2 | false | 下载 frp 最新 release |
| code-server | ubuntu-24.04 | 1/512/4 | false | `curl -fsSL https://code-server.dev/install.sh \| sh` |

**注意**：OpenWRT、WireGuard、Frp 的安装脚本需要更复杂的前期准备（下载二进制、生成配置），

---

### Batch 1-E: LXC 权限修复向导（修改 `cmd/fastpve/prompt_for_lxc.go`）

**新增函数**：
```go
func promptLXCPermissionFix() error
```

**实现流程**：
1. 列出所有 LXC 容器
2. 用户选择一个需要修权限的容器
3. 检测当前容器类型（privileged/unprivileged）
4. 显示两种方案：
   - 方案 A：开启特权模式（`pct set <CTID> --unprivileged 0`）
   - 方案 B：配置 UID 映射（编辑 `/etc/pve/lxc/<CTID>.conf` + `/etc/subuid` + `/etc/subgid`）
5. 执行用户选择的方案

**新增菜单项**：在 `promptForLXC` 的 items 中添加 `{"权限修复", promptLXCPermissionFix}`（放在"模板管理"之后）

---

### Batch 1-F: GPU 直通排障增强（修改 `cmd/fastpve/prompt_for_gpupassthrough.go`）

**新增诊断函数**：
```go
func diagnoseGPUPassthrough() error
```

**诊断流程**：
1. 检测 IOMMU 是否开启 → `dmesg | grep -E 'IOMMU|DMAR'`
2. 列出 IOMMU 分组 → `find /sys/kernel/iommu_groups/ -type l`
3. 显示可直通设备（VGA + Audio）
4. 检测 Nvidia 卡 → 提示 reset bug 和 `kvm.ignore_msrs=1`
5. 检测 ACS 跨组问题 → 提示 `pcie_acs_override=downstream`
6. 检测驱动冲突 → 检查是否 blacklist
7. 生成汇总报告和建议

**在 `promptForGPUPassThrough` 前增加诊断菜单项**：
- `{"直通环境检测", diagnoseGPUPassthrough}`
- `{"一键核显直通", promptForGPUPassThrough}`（保留）

---

### Batch 1-G: HOOK 配置项（修改 `cmd/fastpve/config.go`）

**Config 结构体新增字段**：
```go
HOOKPreCreate  string // 创建 VM/CT 前执行的脚本路径
HOOKPostCreate string // 创建完成后执行的脚本路径
AutoCleanupTmp bool   // 退出时清理临时 iso
SkipConfirm    bool   // 跳过确认提示（全自动化模式）
```

**`LoadConfig` switch 新增 case**：
```go
case "HOOK_PRE_CREATE":
    cfg.HOOKPreCreate = val
case "HOOK_POST_CREATE":
    cfg.HOOKPostCreate = val
case "AUTO_CLEANUP_TMP":
    cfg.AutoCleanupTmp = (val == "true" || val == "1")
case "SKIP_CONFIRM":
    cfg.SkipConfirm = (val == "true" || val == "1")
```

**配置模板新增**（`ensureConfigFile`）：
```ini
# ========== HOOK 脚本 ==========
# 创建 VM/CT 前后自动执行的自定义脚本
# HOOK_PRE_CREATE=/usr/local/bin/fastpve-pre-create.sh
# HOOK_POST_CREATE=/usr/local/bin/fastpve-post-create.sh

# ========== 自动化 ==========
# AUTO_CLEANUP_TMP=false
# SKIP_CONFIRM=false
```

---

### Batch 2-H: 菜单集成（修改 `cmd/fastpve/prompt_system_tools.go`）

在现有 items 的 "CPU 调度器切换" 之后、"系统状态与通知" 之前插入新条目：

```go
{"命令行工具箱", promptToolbox},
{"存储概览与备份", promptStorageTools},
{"网络诊断与安全", promptNetworkTools},
```

在 "系统状态与通知" 之后、AI 助手之前增加：
```go
{"LXC 权限修复", promptLXCPermissionFix},
{"GPU 直通排障", diagnoseGPUPassthrough},
```

**最终菜单顺序**：
```
0、更换软件源
1、安装Docker
2、一键核显直通
3、直通环境检测      ← 新增
4、去除订阅弹窗
5、安装常用工具
6、系统更新
7、清理旧内核
8、ZFS ARC 调优
9、CPU 调度器切换
10、命令行工具箱    ← 新增
11、存储概览与备份   ← 新增
12、网络诊断与安全   ← 新增
13、系统状态与通知
14、LXC 权限修复     ← 新增
15、AI 助手           （条件显示）
16、编辑配置
17、返回
```

---

### 执行次序

```
第 1 波（7 个代理同时启动）:
  A  prompt_toolbox.go       新建  ~80 行
  B  prompt_storage.go       新建  ~100 行
  C  prompt_network.go       新建  ~150 行
  D  lxc_services.go         修改  +6 个预设 (~60 行)
  E  prompt_for_lxc.go       修改  +1 个函数 (~50 行)
  F  prompt_for_gpupass.go   修改  +1 个函数 (~80 行)
  G  config.go               修改  +5 个字段 + 模板 (~30 行)

第 2 波（1 个代理）:
  H  prompt_system_tools.go  修改  +5 个菜单项 (~15 行)
```

**依赖关系**：
- H 依赖 A/B/C/E/F 的函数签名（已在本文档预定义，按签名写即可无等待）
- D/E 修改同一个包不同文件，无冲突
- A/B/C/F/G 互不依赖，完全独立

---

### 代码审查要点

合并前逐项检查：
- [ ] 每个新文件的 `package main` 和 import 正确
- [ ] `gofmt -w` 格式化无 diff
- [ ] `go vet ./cmd/fastpve/` 无警告
- [ ] `go build ./cmd/fastpve/` 编译通过
- [ ] 函数签名与菜单集成调用一致
- [ ] 中文标签编号连续不重复

---

### 回滚方案

每个修改文件在实施前会自动创建 git 备份。如果某个任务出现问题，只需回滚对应文件：
```bash
git checkout -- cmd/fastpve/<问题文件>.go
```
不影响其他并行任务的文件。
