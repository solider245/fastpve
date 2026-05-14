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

func promptNetworkTools() error {
	items := []menuItem{
		{"备份网络配置", backupNetworkConfig},
		{"网络连通性检测", checkNetworkConnectivity},
		{"Let's Encrypt 证书", setupLetsEncrypt},
		{"fail2ban 一键部署", setupFail2ban},
		{"返回", func() error { return errContinue }},
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}
	prompt := promptui.Select{
		Label: "网络诊断与安全",
		Items: labels,
	}
	for {
		idx, _, err := prompt.Run()
		if err != nil {
			return errContinue
		}
		err = items[idx].action()
		if err == errContinue {
			return errContinue
		}
		if err != nil {
			fmt.Println("操作失败:", err)
		}
	}
}

func backupNetworkConfig() error {
	ctx := context.TODO()
	ts := time.Now().Format("20060102_1504")
	return utils.BatchRunStdout(ctx, []string{
		fmt.Sprintf("cp /etc/network/interfaces /etc/network/interfaces.bak.%s", ts),
		fmt.Sprintf("echo '已备份到 /etc/network/interfaces.bak.%s'", ts),
	}, 5)
}

func checkNetworkConnectivity() error {
	ctx := context.TODO()
	fmt.Println("\n========== 网络连通性检测 ==========")

	// 1. 获取默认网关并检测网关连通性
	gwOut, err := utils.BatchOutput(ctx, []string{`ip route | grep default | awk '{print $3}'`}, 5)
	gw := strings.TrimSpace(string(gwOut))
	hasGateway := err == nil && gw != ""
	if hasGateway {
		fmt.Printf("  ✓ 默认网关: %s\n", gw)
	} else {
		fmt.Println("  ✗ 获取默认网关失败")
	}

	// 2. 外网连通性检测（优先 8.8.8.8，备选 223.5.5.5）
	_, err = utils.BatchOutput(ctx, []string{"ping -c 1 -W 5 8.8.8.8"}, 5)
	if err == nil {
		fmt.Println("  ✓ 外网连通性正常 (8.8.8.8)")
	} else {
		_, err = utils.BatchOutput(ctx, []string{"ping -c 1 -W 5 223.5.5.5"}, 5)
		if err == nil {
			fmt.Println("  ✓ 外网连通性正常 (223.5.5.5)")
		} else {
			fmt.Println("  ✗ 外网连通性检测失败 (8.8.8.8 和 223.5.5.5 均不可达)")
		}
	}

	// 3. DNS 解析检测
	_, err = utils.BatchOutput(ctx, []string{"host download.proxmox.com 2>/dev/null || nslookup download.proxmox.com 2>/dev/null"}, 5)
	if err == nil {
		fmt.Println("  ✓ DNS 解析正常 (download.proxmox.com)")
	} else {
		fmt.Println("  ✗ DNS 解析失败")
	}

	// 4. MTU 检测（ping 网关，禁止分片，检查是否需调整 MTU）
	if hasGateway {
		_, err = utils.BatchOutput(ctx, []string{fmt.Sprintf("ping -M do -c 1 -s 1472 %s", gw)}, 5)
		if err == nil {
			fmt.Printf("  ✓ MTU 检测通过 (1472 字节, 网关 %s)\n", gw)
		} else {
			fmt.Println("  ✗ MTU 检测失败, 可能需要调整 MTU 值")
		}
	} else {
		fmt.Println("  ✗ MTU 检测跳过: 无默认网关")
	}

	fmt.Println("===================================")
	return nil
}

func setupLetsEncrypt() error {
	ctx := context.TODO()

	// 如果已有证书，显示过期时间
	certFile := "/etc/pve/local/pveproxy-ssl.pem"
	if _, err := os.Stat(certFile); err == nil {
		out, _ := utils.BatchOutput(ctx, []string{fmt.Sprintf("openssl x509 -enddate -noout -in %s", certFile)}, 5)
		fmt.Printf("  当前证书: %s\n", strings.TrimSpace(string(out)))
	}

	// 输入域名
	domainPrompt := promptui.Prompt{
		Label: "请输入域名",
	}
	domain, err := domainPrompt.Run()
	if err != nil {
		return errContinue
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		fmt.Println("  域名不能为空")
		return nil
	}

	// 输入邮箱
	emailPrompt := promptui.Prompt{
		Label: "请输入邮箱地址",
	}
	email, err := emailPrompt.Run()
	if err != nil {
		return errContinue
	}
	email = strings.TrimSpace(email)
	if email == "" {
		fmt.Println("  邮箱不能为空")
		return nil
	}

	// 验证方式
	methods := []string{"standalone（独立模式）", "dns（DNS验证）"}
	methodPrompt := promptui.Select{
		Label: "选择验证方式",
		Items: methods,
	}
	methodIdx, _, err := methodPrompt.Run()
	if err != nil {
		return errContinue
	}
	methodNames := []string{"standalone", "dns"}
	method := methodNames[methodIdx]
	fmt.Printf("  验证方式: %s\n", method)

	// 执行 ACME 证书申请
	cmds := []string{
		fmt.Sprintf("pvenode acme account register --email %s", email),
		fmt.Sprintf("pvenode acme cert order --domain %s --method %s", domain, method),
	}
	fmt.Println("  正在申请 Let's Encrypt 证书...")
	if err := utils.BatchRunStdout(ctx, cmds, 120); err != nil {
		return fmt.Errorf("证书申请失败: %w", err)
	}
	fmt.Println("  证书申请成功")
	return nil
}

func setupFail2ban() error {
	ctx := context.TODO()

	// 1. 安装 fail2ban
	fmt.Println("  正在安装 fail2ban...")
	if err := utils.BatchRunStdout(ctx, []string{"apt install -y -qq fail2ban"}, 120); err != nil {
		return fmt.Errorf("安装 fail2ban 失败: %w", err)
	}

	// 2. 写入 PVE 保护配置
	jailContent := "[proxmox]\n" +
		"enabled = true\n" +
		"port = https,http,8006\n" +
		"filter = proxmox\n" +
		"logpath = /var/log/pveproxy/access.log\n" +
		"maxretry = 3\n" +
		"bantime = 3600\n"
	if err := os.WriteFile("/etc/fail2ban/jail.local", []byte(jailContent), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	fmt.Println("  已写入 /etc/fail2ban/jail.local")

	// 3. 启用并重启
	cmds := []string{
		"systemctl enable --now fail2ban",
		"systemctl restart fail2ban",
	}
	if err := utils.BatchRunStdout(ctx, cmds, 30); err != nil {
		return fmt.Errorf("启动 fail2ban 失败: %w", err)
	}

	// 4. 显示状态
	out, err := utils.BatchOutput(ctx, []string{"fail2ban-client status proxmox 2>/dev/null || fail2ban-client status"}, 10)
	if err != nil {
		fmt.Println("  获取 fail2ban 状态失败")
	} else {
		fmt.Printf("  fail2ban 状态:\n%s\n", string(out))
	}
	return nil
}
