# FastPVE Plus — 架构图

## 整体架构

```mermaid
flowchart TB
    subgraph 用户入口
        A1["海外用户<br/>curl dd-install.sh"]
        A2["国内用户<br/>curl dd-install-cn.sh<br/>（走 gh.565600.xyz 代理）"]
    end

    subgraph 安装层
        B1["dd-install.sh<br/>直连优先 → 3s超时切代理"]
        B2["dd-install-cn.sh<br/>KSpeeder(gh.linkease.net:5443)优先<br/>回落 gh.565600.xyz"]
        B3["/usr/local/bin/fastpve<br/>/usr/local/bin/fastpve-download"]
    end

    subgraph 二进制发布
        C1["GitHub Releases<br/>solider245/fastpve"]
    end

    subgraph 交互菜单
        D1["主菜单<br/>0-5: 官方功能<br/>6: DD-安装更多系统"]
        D2["DD 预设菜单<br/>5个分类 20个系统<br/>+ 自定义URL"]
        D3["配置界面<br/>CPU/内存/磁盘<br/>URL确认"]
    end

    subgraph 下载引擎
        E1["vmdownloader.DownloadDDImage()<br/>HEAD → 下载 → 解压"]
        E2["downloader.Downloader<br/>HTTP Range 断点续传<br/>进度上报"]
    end

    subgraph 虚拟机创建
        F1["qm create<br/>+ qm set scsi0 import-from"]
        F2["PVE 宿主机<br/>KVM 虚拟机"]
    end

    A1 --> B1
    A2 --> B2
    B1 & B2 --> B3
    C1 --> B1
    C1 --> B2
    B3 --> D1
    D1 --> D2
    D2 --> D3
    D3 --> E1
    E1 --> E2
    E1 --> F1
    F1 --> F2
```

## DD 预设系统数据结构

```mermaid
classDiagram
    class DDPresetCategory {
        +string Name
        +[]DDPreset Presets
    }
    class DDPreset {
        +string Name
        +string Description
        +[]string URLs
        +int BIOS
        +string OSType
        +string Machine
        +int MinDiskGB
        +int RecMemoryMB
        +int RecCores
    }
    class ddPresetInstallInfo {
        +DDPreset Preset
        +string DDImgName
        +string DownloadURL
        +int Memory
        +int Cores
        +int Disk
        +bool DownloadOnly
    }

    DDPresetCategory "1" --> "*" DDPreset
    ddPresetInstallInfo --> DDPreset
    DDPresetCategory : +AllDDPresetCategories() []DDPresetCategory
```

## 代码分层

```
cmd/fastpve/
  prompt.go              ← 主菜单（0-6 + q）
  prompt_for_dd.go       ← 自定义 URL DD 安装（兜底）
  prompt_for_dd_presets.go  ← 预设系统展平菜单 + 安装流程
  prompt_for_win.go      ← Windows ISO 安装
  prompt_for_ubuntu.go   ← Ubuntu ISO 安装
  prompt_for_istoreos.go ← iStoreOS IMG 安装

cmd/download/
  main.go                ← fastpve-download CLI 入口
  dd.go                  ← dd 子命令

vmdownloader/
  dd.go                  ← DownloadDDImage + decompressDD
  presets.go             ← 20个系统预设定义
  istore.go              ← iStoreOS 下载逻辑
  windows.go             ← Windows ISO 下载
  ubuntu.go              ← Ubuntu ISO 下载
  vmdownloader.go        ← Downloader 接口 + DownloadFile

downloader/
  downloader.go          ← HTTP downloader
  resumable.go           ← 断点续传引擎

quickget/
  qm.go                  ← qm list / pvesm status 解析
```

## 安装流程时序

```mermaid
sequenceDiagram
    participant U as 用户(PVE)
    participant S as 安装脚本
    participant R as GitHub Releases
    participant K as KSpeeder 代理
    participant F as fastpve 二进制

    U->>S: curl dd-install-cn.sh | bash
    S->>K: 下载 FastPVE 二进制
    K-->>S: 二进制文件
    alt KSpeeder 不可用
        S->>R: 回落 gh.565600.xyz
        R-->>S: 二进制文件
    end
    S->>S: SHA256 校验
    S->>F: cp → /usr/local/bin/
    S->>F: exec fastpve

    U->>F: 选择 DD-安装更多系统
    F->>U: 展开 20 个系统列表
    U->>F: 选择系统（如 Debian 12）
    F->>U: 显示推荐配置 + URL
    U->>F: 确认 CPU/内存/磁盘
    F->>R: 下载 DD 镜像（.img.gz）
    F->>F: 解压 → /var/lib/vz/template/iso/
    F->>F: qm create + qm set scsi0 import-from
    F-->>U: VM 创建完成 (VMID: xxx)
```
