package vmdownloader

// DDPreset defines a pre-configured DD image source.
type DDPreset struct {
	Name        string   // Display name
	Description string   // One-line description
	URLs        []string // DD image URLs, tried in order
	BIOS        int      // biosUEFI (0) or biosSeaBIOS (1)
	OSType      string   // PVE ostype string
	Machine     string   // q35 or i440fx
	MinDiskGB   int      // Minimum disk size
	RecMemoryMB int      // Recommended memory
	RecCores    int      // Recommended CPU cores
}

// DDPresetCategory groups related presets for the menu.
type DDPresetCategory struct {
	Name    string
	Presets []DDPreset
}

const (
	BIOSUEFI    = 0
	BIOSSeaBIOS = 1
)

// AllDDPresetCategories returns all DD preset categories.
func AllDDPresetCategories() []DDPresetCategory {
	return []DDPresetCategory{
		{
			Name: "DD-软路由/防火墙",
			Presets: []DDPreset{
				{
					Name:        "OpenWRT 官方",
					Description: "开源路由系统，官方主线版本",
					URLs: []string{
						"https://downloads.openwrt.org/releases/24.10.0/targets/x86/64/openwrt-24.10.0-x86-64-generic-ext4-combined-efi.img.gz",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   4,
					RecMemoryMB: 1024,
					RecCores:    2,
				},
				{
					Name:        "ImmortalWrt",
					Description: "OpenWRT 社区增强分支，预装常用插件",
					URLs: []string{
						"https://downloads.immortalwrt.org/releases/24.10.0/targets/x86/64/immortalwrt-24.10.0-x86-64-generic-ext4-combined-efi.img.gz",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   4,
					RecMemoryMB: 1024,
					RecCores:    2,
				},
				{
					Name:        "iStoreOS DD版",
					Description: "KoolCenter 软路由，应用商店丰富",
					URLs: []string{
						"https://fw0.koolcenter.com/iStoreOS/x86_64_efi/istoreos-24.10.1-2025052311-x86-64-squashfs-combined-efi.img.gz",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   4,
					RecMemoryMB: 2048,
					RecCores:    2,
				},
				{
					Name:        "RouterOS CHR",
					Description: "MikroTik 路由系统 (Cloud Hosted Router)",
					URLs: []string{
						"https://download.mikrotik.com/routeros/7.18.1/chr-7.18.1.img.zip",
					},
					BIOS:        BIOSSeaBIOS,
					OSType:      "l26",
					Machine:     "i440fx",
					MinDiskGB:   1,
					RecMemoryMB: 256,
					RecCores:    1,
				},
				{
					Name:        "pfSense CE",
					Description: "FreeBSD 企业级防火墙",
					URLs: []string{
						"https://atxfiles.netgate.com/mirror/downloads/pfSense-CE-2.7.2-RELEASE-amd64.img.gz",
					},
					BIOS:        BIOSUEFI,
					OSType:      "other",
					Machine:     "q35",
					MinDiskGB:   8,
					RecMemoryMB: 2048,
					RecCores:    2,
				},
				{
					Name:        "OPNsense",
					Description: "pfSense 分支，UI 更现代",
					URLs: []string{
						"https://mirror.dns-root.de/opnsense/releases/25.1/OPNsense-25.1-OpenSSL-vga-amd64.img.bz2",
					},
					BIOS:        BIOSUEFI,
					OSType:      "other",
					Machine:     "q35",
					MinDiskGB:   8,
					RecMemoryMB: 2048,
					RecCores:    2,
				},
			},
		},
		{
			Name: "DD-NAS/存储",
			Presets: []DDPreset{
				{
					Name:        "群晖 DSM (RR引导)",
					Description: "Redpill Recovery 引导，需自行配置引导参数",
					URLs: []string{
						"https://github.com/RROrg/rr/releases/download/25.3.1/rr-25.3.1.img.zip",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   32,
					RecMemoryMB: 4096,
					RecCores:    4,
				},
				{
					Name:        "TrueNAS Scale",
					Description: "开源企业级 NAS 系统",
					URLs: []string{
						"https://download.sys.truenas.net/TrueNAS-SCALE-Eel/25.04.1/TrueNAS-SCALE-25.04.1.iso",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   16,
					RecMemoryMB: 8192,
					RecCores:    4,
				},
				{
					Name:        "OpenMediaVault",
					Description: "Debian 系 NAS 系统，轻量稳定",
					URLs: []string{
						"https://downloads.sourceforge.net/project/openmediavault/7.5.2/openmediavault_7.5.2-amd64.iso",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   8,
					RecMemoryMB: 2048,
					RecCores:    2,
				},
				{
					Name:        "飞牛私有云",
					Description: "国产 NAS 系统，照片管理出色 (fnOS)",
					URLs: []string{
						"https://download.fnos.net/latest/fnos.img.gz",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   16,
					RecMemoryMB: 4096,
					RecCores:    4,
				},
			},
		},
		{
			Name: "DD-Windows",
			Presets: []DDPreset{
				{
					Name:        "Windows Server 2025",
					Description: "数据中心版，需自行激活（推荐去 dd.wiseadvice.cc 获取DD直链）",
					URLs:        []string{},
					BIOS:        BIOSUEFI,
					OSType:      "win11",
					Machine:     "q35",
					MinDiskGB:   40,
					RecMemoryMB: 4096,
					RecCores:    4,
				},
				{
					Name:        "Windows Server 2022",
					Description: "数据中心版，需自行激活（推荐去 dd.wiseadvice.cc 获取DD直链）",
					URLs:        []string{},
					BIOS:        BIOSUEFI,
					OSType:      "win10",
					Machine:     "q35",
					MinDiskGB:   40,
					RecMemoryMB: 4096,
					RecCores:    4,
				},
				{
					Name:        "Windows 11 LTSC",
					Description: "精简长期版，无商店和 Edge（推荐去 dd.wiseadvice.cc 获取DD直链）",
					URLs:        []string{},
					BIOS:        BIOSUEFI,
					OSType:      "win11",
					Machine:     "q35",
					MinDiskGB:   32,
					RecMemoryMB: 4096,
					RecCores:    4,
				},
				{
					Name:        "Windows 10 LTSC",
					Description: "最稳定的 Windows 精简版（推荐去 dd.wiseadvice.cc 获取DD直链）",
					URLs:        []string{},
					BIOS:        BIOSUEFI,
					OSType:      "win10",
					Machine:     "q35",
					MinDiskGB:   32,
					RecMemoryMB: 4096,
					RecCores:    4,
				},
			},
		},
		{
			Name: "DD-Linux服务器",
			Presets: []DDPreset{
				{
					Name:        "Debian 12",
					Description: "最通用的 Linux 服务器发行版",
					URLs: []string{
						"https://mirrors.tuna.tsinghua.edu.cn/debian/images/cloud/bookworm/latest/debian-12-genericcloud-amd64.raw",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   8,
					RecMemoryMB: 1024,
					RecCores:    2,
				},
				{
					Name:        "Rocky Linux 9",
					Description: "RHEL 9 兼容发行版，企业级稳定",
					URLs: []string{
						"https://mirrors.ustc.edu.cn/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   10,
					RecMemoryMB: 2048,
					RecCores:    2,
				},
				{
					Name:        "Arch Linux",
					Description: "滚动更新，始终最新",
					URLs: []string{
						"https://mirrors.tuna.tsinghua.edu.cn/archlinux/images/latest/Arch-Linux-x86_64-cloudimg.qcow2",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   8,
					RecMemoryMB: 1024,
					RecCores:    2,
				},
				{
					Name:        "Alpine Linux",
					Description: "超轻量，适合容器和基础服务",
					URLs: []string{
						"https://mirrors.ustc.edu.cn/alpine/latest-stable/releases/x86_64/alpine-virt-3.22.1-x86_64.iso",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   2,
					RecMemoryMB: 512,
					RecCores:    1,
				},
			},
		},
		{
			Name: "DD-其他",
			Presets: []DDPreset{
				{
					Name:        "Home Assistant OS",
					Description: "智能家居中枢系统",
					URLs: []string{
						"https://github.com/home-assistant/operating-system/releases/download/15.0/haos_generic-x86-64-15.0.img.xz",
					},
					BIOS:        BIOSUEFI,
					OSType:      "l26",
					Machine:     "q35",
					MinDiskGB:   32,
					RecMemoryMB: 2048,
					RecCores:    2,
				},
			},
		},
	}
}
