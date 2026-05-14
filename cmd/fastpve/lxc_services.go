package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/utils"
)

type lxcServicePreset struct {
	Name        string
	Description string
	BaseOS      string // keyword for pveam available
	DefHostname string
	Cores       int
	Memory      int
	Disk        int
	Privileged  bool   // Docker-based services need privileged + nesting
	SetupScript string // bash script run inside container
	PostMessage string // %s replaced with container IP
}

var lxcServicePresets = []lxcServicePreset{
	{
		Name:        "ttyd - Web SSH 终端",
		Description: "浏览器端远程终端，方便随时管理服务器",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "ttyd",
		Cores:       1, Memory: 512, Disk: 4,
		Privileged: false,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq ttyd
cat > /etc/systemd/system/ttyd.service << 'EOF'
[Unit]
Description=ttyd Web Terminal
After=network.target

[Service]
ExecStart=/usr/bin/ttyd -p 7681 bash
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl enable --now ttyd.service`,
		PostMessage: "访问 http://%s:7681 使用 Web SSH 终端",
	},
	{
		Name:        "Nginx Proxy Manager",
		Description: "反向代理 + SSL 证书管理，方便对外暴露服务",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "npm",
		Cores:       1, Memory: 512, Disk: 8,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq docker.io
systemctl enable --now docker
docker run -d --restart=unless-stopped --name npm \
  -p 80:80 -p 443:443 -p 81:81 \
  -v npm_data:/data \
  -v npm_ssl:/etc/letsencrypt \
  jc21/nginx-proxy-manager:latest`,
		PostMessage: "访问 http://%s:81 管理 NPM（默认账号: admin@example.com / changeme）",
	},
	{
		Name:        "AdGuard Home",
		Description: "全网 DNS 广告过滤 + DHCP 服务器，轻量高效",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "adguard",
		Cores:       1, Memory: 256, Disk: 4,
		Privileged: false,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq curl
curl -sSL https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | bash`,
		PostMessage: "访问 http://%s:80 初始化 AdGuard Home（端口 53 为 DNS）",
	},
	{
		Name:        "Tailscale",
		Description: "Mesh VPN 组网，远程访问你的所有设备",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "tailscale",
		Cores:       1, Memory: 256, Disk: 4,
		Privileged: false,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq curl
curl -fsSL https://tailscale.com/install.sh | sh
echo ''
echo '==========================================='
echo 'Tailscale 已安装！请手动运行: tailscale up'
echo '==========================================='`,
		PostMessage: "容器内执行 tailscale up 完成认证后即可组网",
	},
	{
		Name:        "Uptime Kuma",
		Description: "美观的网站/服务监控面板，支持通知告警",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "uptime",
		Cores:       1, Memory: 512, Disk: 8,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq docker.io
systemctl enable --now docker
docker run -d --restart=unless-stopped --name uptime-kuma \
  -p 3001:3001 \
  -v uptime-kuma_data:/app/data \
  louislam/uptime-kuma:latest`,
		PostMessage: "访问 http://%s:3001 使用 Uptime Kuma",
	},
	{
		Name:        "Docker + Portainer",
		Description: "Docker 容器运行环境 + 可视化面板管理",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "docker",
		Cores:       2, Memory: 1024, Disk: 16,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq docker.io docker-compose-v2
systemctl enable --now docker
docker run -d --restart=unless-stopped --name portainer \
  -p 9000:9000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v portainer_data:/data \
  portainer/portainer-ce:latest`,
		PostMessage: "访问 http://%s:9000 初始化 Portainer",
	},
	{
		Name:        "OpenWRT — 软路由系统",
		Description: "轻量级软路由，支持防火墙/QoS/负载均衡",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "openwrt",
		Cores:       1, Memory: 256, Disk: 2,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq docker.io
systemctl enable --now docker
docker run -d --restart=unless-stopped --name openwrt --network host --privileged \
  -v openwrt_data:/etc/openwrt \
  ghcr.io/solider245/openwrt-docker:latest 2>/dev/null || \
docker run -d --restart=unless-stopped --name openwrt --network host --privileged \
  openwrtorg/imagebuilder:latest /sbin/init`,
		PostMessage: "OpenWRT 已部署，请连接容器控制台进行配置: pct enter %d",
	},
	{
		Name:        "Home Assistant — 智能家居",
		Description: "开源智能家居平台，支持大量 IoT 设备集成",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "homeassistant",
		Cores:       2, Memory: 1024, Disk: 8,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq docker.io
systemctl enable --now docker
docker run -d --restart=unless-stopped --name homeassistant \
  -p 8123:8123 \
  -v ha_config:/config \
  ghcr.io/home-assistant/home-assistant:stable`,
		PostMessage: "访问 http://%s:8123 初始化 Home Assistant",
	},
	{
		Name:        "Jellyfin — 媒体服务器",
		Description: "开源媒体管理平台，管理电影/音乐/电视节目",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "jellyfin",
		Cores:       2, Memory: 2048, Disk: 16,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq docker.io
systemctl enable --now docker
mkdir -p /media /config
docker run -d --restart=unless-stopped --name jellyfin \
  -p 8096:8096 \
  -v jellyfin_config:/config \
  -v /media:/media \
  jellyfin/jellyfin:latest`,
		PostMessage: "访问 http://%s:8096 配置 Jellyfin",
	},
	{
		Name:        "WireGuard — VPN 网关",
		Description: "轻量高速 VPN，用于远程访问内网服务",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "wireguard",
		Cores:       1, Memory: 256, Disk: 2,
		Privileged: true,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq wireguard-tools iptables
mkdir -p /etc/wireguard
cd /etc/wireguard
umask 077
wg genkey | tee privatekey | wg pubkey > publickey
echo ""
echo "============================================"
echo "WireGuard 已安装"
echo "公钥: $(cat publickey)"
echo "私钥: $(cat privatekey)"
echo "请编辑 /etc/wireguard/wg0.conf 配置客户端"
echo "============================================"`,
		PostMessage: "容器已创建，执行 pct enter %d 后配置 wg0.conf",
	},
	{
		Name:        "Frp — 内网穿透",
		Description: "将内网服务暴露到公网，支持 TCP/UDP/HTTP",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "frp",
		Cores:       1, Memory: 256, Disk: 2,
		Privileged: false,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq curl
FRP_VERSION=$(curl -s https://api.github.com/repos/fatedier/frp/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/v//')
curl -sSL "https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_amd64.tar.gz" | tar xz -C /usr/local
ln -sf "/usr/local/frp_${FRP_VERSION}_linux_amd64/frps" /usr/local/bin/frps
ln -sf "/usr/local/frp_${FRP_VERSION}_linux_amd64/frpc" /usr/local/bin/frpc
echo "Frp ${FRP_VERSION} 已安装"
echo "服务端: frps -c frps.toml"
echo "客户端: frpc -c frpc.toml"`,
		PostMessage: "Frp 已安装到容器内，pct enter %d 后创建配置文件",
	},
	{
		Name:        "code-server — Web IDE",
		Description: "浏览器端的 VS Code，随时随地编程",
		BaseOS:      "ubuntu-24.04",
		DefHostname: "codeserver",
		Cores:       1, Memory: 512, Disk: 4,
		Privileged: false,
		SetupScript: `set -e
apt-get update -qq
apt-get install -y -qq curl
curl -fsSL https://code-server.dev/install.sh | sh
systemctl enable --now code-server@$(whoami)
sed -i 's/127.0.0.1:8080/0.0.0.0:8080/' ~/.config/code-server/config.yaml
systemctl restart code-server@$(whoami)`,
		PostMessage: "访问 http://%s:8080 使用 code-server（密码在 ~/.config/code-server/config.yaml）",
	},
}

func findLXCTemplate(keyword string) (name string, err error) {
	out, err := utils.BatchOutput(context.TODO(), []string{"pveam available 2>/dev/null || true"}, 60)
	if err != nil {
		return "", err
	}
	templates := parsePveamAvailable(out)

	var matches []lxcTemplate
	kw := strings.ToLower(keyword)
	for _, t := range templates {
		if strings.Contains(strings.ToLower(t.name), kw) {
			matches = append(matches, t)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("未找到匹配 %q 的模板", keyword)
	}
	selected := matches[len(matches)-1] // latest match

	listOut, _ := utils.BatchOutput(context.TODO(), []string{"pveam list local 2>/dev/null || true"}, 10)
	needDL := true
	for _, t := range parsePveamList(listOut) {
		if t.name == selected.name {
			needDL = false
			break
		}
	}
	if needDL {
		url := fmt.Sprintf("https://download.proxmox.com/images/%s/%s", selected.section, selected.name)
		dest := fmt.Sprintf("/var/lib/vz/template/cache/%s", selected.name)
		if err := utils.DownloadFile(url, dest); err != nil {
			return "", err
		}
	} else {
		fmt.Printf("模板 %s 已存在\n", selected.name)
	}

	return selected.name, nil
}

func promptDeployLXCService() error {
	items := make([]string, len(lxcServicePresets)+1)
	for i, p := range lxcServicePresets {
		items[i] = fmt.Sprintf("%s — %s (%d核/%dMB/%dGB)", p.Name, p.Description, p.Cores, p.Memory, p.Disk)
	}
	items[len(lxcServicePresets)] = "返回"

	prompt := promptui.Select{
		Label: "选择要部署的服务",
		Items: items,
		Size:  15,
	}
	idx, _, err := prompt.Run()
	if err != nil || idx >= len(lxcServicePresets) {
		return errContinue
	}
	preset := lxcServicePresets[idx]

	fmt.Println("==========================================")
	fmt.Printf("  部署: %s\n", preset.Name)
	fmt.Printf("  说明: %s\n", preset.Description)
	fmt.Println("==========================================")

	fmt.Println("正在查找系统模板...")
	templateName, err := findLXCTemplate(preset.BaseOS)
	if err != nil {
		return err
	}

	_, vmid, err := resolveStorageAndVMID()
	if err != nil {
		return err
	}

	hostnamePrompt := promptui.Prompt{
		Label:   "主机名",
		Default: preset.DefHostname,
	}
	hostname, err := hostnamePrompt.Run()
	if err != nil {
		return errContinue
	}
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		hostname = preset.DefHostname
	}

	pwdPrompt := promptui.Prompt{
		Label: "root密码",
		Mask:  '*',
	}
	password, err := pwdPrompt.Run()
	if err != nil {
		return errContinue
	}

	cores, _ := promptIntWithDefault("CPU核数", preset.Cores)
	memory, _ := promptIntWithDefault("内存 (MB)", preset.Memory)
	disk, _ := promptIntWithDefault("磁盘 (GB)", preset.Disk)
	bridge := promptBridge()

	privilegeFlag := "1"
	features := "nesting=1"
	if preset.Privileged {
		privilegeFlag = "0"
	} else {
		features = ""
	}

	createCmd := fmt.Sprintf(
		"pct create %d local:vztmpl/%s --hostname %s --password '%s' --rootfs local:%dG --memory %d --cores %d --swap 512 --net0 name=eth0,bridge=%s,ip=dhcp --unprivileged %s",
		vmid, templateName, hostname, password, disk, memory, cores, bridge, privilegeFlag,
	)
	if features != "" {
		createCmd += " --features " + features
	}

	fmt.Println("正在创建容器...")
	if err := utils.BatchRunStdout(context.TODO(), []string{createCmd}, 120); err != nil {
		return err
	}

	fmt.Println("正在启动容器...")
	if err := utils.BatchRunStdout(context.TODO(), []string{fmt.Sprintf("pct start %d", vmid)}, 30); err != nil {
		return err
	}

	fmt.Println("等待容器就绪...")
	time.Sleep(5 * time.Second)

	// Write setup script to temp file and push into container
	tmpFile := fmt.Sprintf("/tmp/lxc-service-setup-%d.sh", vmid)
	scriptContent := "#!/bin/bash\n" + preset.SetupScript + "\n"
	if err := os.WriteFile(tmpFile, []byte(scriptContent), 0755); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	fmt.Println("正在安装服务...")
	setupScripts := []string{
		"set -e",
		fmt.Sprintf("pct push %d %s /tmp/setup.sh", vmid, tmpFile),
		fmt.Sprintf("pct exec %d -- bash /tmp/setup.sh", vmid),
	}
	if err := utils.BatchRunStdout(context.TODO(), setupScripts, 180); err != nil {
		return fmt.Errorf("服务安装失败: %w", err)
	}

	// Get container IP
	ipOut, err := utils.BatchOutput(context.TODO(), []string{
		fmt.Sprintf("pct exec %d -- hostname -I 2>/dev/null | awk '{print $1}'", vmid),
	}, 10)
	ip := "?"
	if err == nil {
		if f := strings.Fields(string(ipOut)); len(f) > 0 {
			ip = strings.TrimSpace(f[0])
		}
	}

	fmt.Println("")
	fmt.Println("==========================================")
	fmt.Printf("  ✅ %s 部署完成！\n", preset.Name)
	fmt.Printf("  容器 ID: %d\n", vmid)
	if ip != "?" {
		msg := strings.ReplaceAll(preset.PostMessage, "%s", ip)
		fmt.Printf("  %s\n", msg)
	}
	fmt.Println("==========================================")
	return nil
}
