package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const configPath = "/etc/fastpve/config.conf"

// Config holds all FastPVE runtime configuration.
// Empty values disable the associated feature (config-as-toggle).
type Config struct {
	AIURL   string // API 地址
	AIKey   string // API 密钥
	AIModel string // 模型名

	NotifyTelegramToken  string
	NotifyTelegramChatID string

	ISOPath       string // ISO 下载目录
	CachePath     string // 模板缓存目录
	GHCRMirror    string // GHCR 镜像加速
	ProxmoxMirror string // Proxmox 镜像源
	DDProxyHosts  string // DD 下载代理（逗号分隔）

	VMCores   int
	VMMemory  int
	VMDisk    int
	VMBridge  string
	VMStorage string

	HealthDiskThreshold int // 磁盘使用率告警 (%)
	HealthMemThreshold  int
	DownloadTimeout     int // 秒
	DDCustomArgs        string

	// HOOK 脚本（创建 VM/CT 前后自动执行）
	HOOKPreCreate  string
	HOOKPostCreate string

	// 自动化
	AutoCleanupTmp bool // 退出时清理临时 iso 文件
	SkipConfirm    bool // 跳过确认提示（全自动化模式）
}

var Cfg *Config

func InitConfig() {
	c, err := LoadConfig()
	if err != nil {
		// File missing or parse error — defaults are fine
		c = defaultConfig()
	}
	Cfg = c
}

func defaultConfig() *Config {
	return &Config{
		AIURL:               "https://api.deepseek.com/v1/chat/completions",
		AIModel:             "deepseek-chat",
		ISOPath:             defaultISOPath,
		CachePath:           defaultCachePath,
		VMCores:             2,
		VMMemory:            2048,
		VMDisk:              64,
		VMBridge:            "vmbr0",
		VMStorage:           "local",
		HealthDiskThreshold: 90,
		HealthMemThreshold:  90,
		DownloadTimeout:     300,
	}
}

func LoadConfig() (*Config, error) {
	cfg := defaultConfig()

	f, err := os.Open(configPath)
	if err != nil {
		ensureConfigFile()
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}

		switch key {
		case "AI_URL":
			cfg.AIURL = val
		case "AI_KEY":
			cfg.AIKey = val
		case "AI_MODEL":
			cfg.AIModel = val
		case "NOTIFY_TELEGRAM_TOKEN":
			cfg.NotifyTelegramToken = val
		case "NOTIFY_TELEGRAM_CHAT_ID":
			cfg.NotifyTelegramChatID = val
		case "ISO_PATH":
			cfg.ISOPath = val
		case "CACHE_PATH":
			cfg.CachePath = val
		case "GHCR_MIRROR":
			cfg.GHCRMirror = val
		case "PROXMOX_MIRROR":
			cfg.ProxmoxMirror = val
		case "DD_PROXY_HOSTS":
			cfg.DDProxyHosts = val
		case "VM_CORES":
			cfg.VMCores = parseInt(val, cfg.VMCores)
		case "VM_MEMORY":
			cfg.VMMemory = parseInt(val, cfg.VMMemory)
		case "VM_DISK":
			cfg.VMDisk = parseInt(val, cfg.VMDisk)
		case "VM_BRIDGE":
			cfg.VMBridge = val
		case "VM_STORAGE":
			cfg.VMStorage = val
		case "HEALTH_DISK_THRESHOLD":
			cfg.HealthDiskThreshold = parseInt(val, cfg.HealthDiskThreshold)
		case "HEALTH_MEM_THRESHOLD":
			cfg.HealthMemThreshold = parseInt(val, cfg.HealthMemThreshold)
		case "DOWNLOAD_TIMEOUT":
			cfg.DownloadTimeout = parseInt(val, cfg.DownloadTimeout)
		case "DD_CUSTOM_ARGS":
			cfg.DDCustomArgs = val
		case "HOOK_PRE_CREATE":
			cfg.HOOKPreCreate = val
		case "HOOK_POST_CREATE":
			cfg.HOOKPostCreate = val
		case "AUTO_CLEANUP_TMP":
			cfg.AutoCleanupTmp = (val == "true" || val == "1")
		case "SKIP_CONFIRM":
			cfg.SkipConfirm = (val == "true" || val == "1")
		}
	}
	return cfg, scanner.Err()
}

// ensureConfigFile writes a fully-commented template if no config file exists.
func ensureConfigFile() {
	if _, err := os.Stat(configPath); err == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return
	}

	template := `# FastPVE 配置文件
# 修改后重启 fastpve 即可生效
# 空值或注释掉 = 关闭对应功能（配置即开关）


# ========== AI 助手 ==========
# 配置后主菜单自动显示「AI 助手」入口
# AI 默认使用 DeepSeek API，兼容 OpenAI 格式，可更换任意提供商
# AI_URL=https://api.deepseek.com/v1/chat/completions
# AI_KEY=sk-your-key-here
# AI_MODEL=deepseek-chat


# ========== 通知 (Telegram) ==========
# 配置后健康检查页面可推送报告到 Telegram
# NOTIFY_TELEGRAM_TOKEN=123456:ABC-DEF
# NOTIFY_TELEGRAM_CHAT_ID=123456789


# ========== 路径 ==========
# ISO_PATH=/var/lib/vz/template/iso/
# CACHE_PATH=/var/lib/vz/template/cache
# GHCR_MIRROR=https://ghcr.nju.edu.cn
# PROXMOX_MIRROR=http://mirrors.ustc.edu.cn/proxmox
# DD_PROXY_HOSTS=ghproxy.net,ghproxy.2020.de


# ========== 默认虚拟机配置 ==========
# VM_CORES=2
# VM_MEMORY=2048
# VM_DISK=64
# VM_BRIDGE=vmbr0
# VM_STORAGE=local


# ========== 阈值 ==========
# HEALTH_DISK_THRESHOLD=90
# HEALTH_MEM_THRESHOLD=90
# DOWNLOAD_TIMEOUT=300


# ========== HOOK 脚本 ==========
# 创建 VM/CT 前后自动执行的自定义脚本路径
# 脚本会传入 VMID 作为第一个参数
# HOOK_PRE_CREATE=/usr/local/bin/fastpve-pre-create.sh
# HOOK_POST_CREATE=/usr/local/bin/fastpve-post-create.sh


# ========== 自动化 ==========
# AUTO_CLEANUP_TMP=false
# SKIP_CONFIRM=false


# ========== 高级 ==========
# DD_CUSTOM_ARGS=--skip-parse
`
	_ = os.WriteFile(configPath, []byte(template), 0644)
}

func parseInt(s string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return v
}
