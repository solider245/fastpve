package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/utils"
)

const notifyConfigPath = "/etc/fastpve/notify.conf"

func promptHealthCheck() error {
	items := []menuItem{
		{"查看系统状态", runHealthCheck},
		{"配置通知渠道", configNotification},
		{"设置定时检查", setupScheduledCheck},
		{"测试通知", testNotification},
		{"返回", func() error { return errContinue }},
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}

	prompt := promptui.Select{
		Label: "系统状态与通知",
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

func runHealthCheck() error {
	ctx := context.TODO()

	cmds := []struct {
		label string
		cmd   string
	}{
		{"PVE 版本", "pveversion 2>/dev/null || echo 'N/A'"},
		{"运行时间", "uptime -p 2>/dev/null | sed 's/up //' || echo 'N/A'"},
		{"CPU 负载", "cat /proc/loadavg 2>/dev/null | awk '{printf \"1m: %s  5m: %s  15m: %s\", $1, $2, $3}' || echo 'N/A'"},
		{"内存", "free -h | awk '/Mem/{printf \"已用: %s / 总计: %s\", $3, $2}'"},
		{"磁盘 /", "df -h / | awk 'NR==2{printf \"已用: %s / 总计: %s (%s)\", $3, $2, $5}'"},
		{"磁盘 /var/lib/vz", "df -h /var/lib/vz 2>/dev/null | awk 'NR==2{printf \"已用: %s / 总计: %s (%s)\", $3, $2, $5}' || echo 'N/A'"},
	}

	fmt.Println("")
	fmt.Println("========== FastPVE 系统状态 ==========")
	for _, c := range cmds {
		out, _ := utils.BatchOutput(ctx, []string{c.cmd}, 5)
		val := strings.TrimSpace(string(out))
		fmt.Printf("  %-12s %s\n", c.label+":", val)
	}

	// VM and CT counts
	vmOut, _ := utils.BatchOutput(ctx, []string{"qm list 2>/dev/null | awk 'NR>1{print $2}' | sort | uniq -c | awk '{printf \"%s: %s  \", $2, $1}' || true"}, 5)
	ctOut, _ := utils.BatchOutput(ctx, []string{"pct list 2>/dev/null | awk 'NR>1{print $2}' | sort | uniq -c | awk '{printf \"%s: %s  \", $2, $1}' || true"}, 5)
	fmt.Printf("  VM 状态:     %s\n", strings.TrimSpace(string(vmOut)))
	fmt.Printf("  CT 状态:     %s\n", strings.TrimSpace(string(ctOut)))

	// Last backup check
	backupOut, _ := utils.BatchOutput(ctx, []string{
		"ls -1t /var/lib/vz/dump/*.log 2>/dev/null | head -1 | xargs grep -l 'OK' 2>/dev/null || echo '无备份记录'",
	}, 5)
	if strings.TrimSpace(string(backupOut)) != "无备份记录" {
		fmt.Printf("  最后备份:    %s\n", "最近备份正常")
	} else {
		fmt.Printf("  最后备份:    %s\n", "无备份记录")
	}

	fmt.Println("======================================")
	fmt.Println("")

	// Prompt notification check
	if _, err := os.Stat(notifyConfigPath); err == nil {
		confirmPrompt := promptui.Prompt{
			Label: "发送此报告到通知渠道? (y/N)",
		}
		resp, _ := confirmPrompt.Run()
		if strings.ToLower(strings.TrimSpace(resp)) == "y" {
			return sendNotification("FastPVE 健康报告", collectHealthReport(ctx))
		}
	}

	return nil
}

func collectHealthReport(ctx context.Context) string {
	cmds := []string{
		"echo '=== FastPVE 系统状态 ==='",
		"echo \"时间: $(date '+%Y-%m-%d %H:%M')\"",
		"pveversion",
		"uptime -p | sed 's/up //' | xargs echo 运行:",
		"free -h | awk '/Mem/{printf \"内存: %s/%s\\n\", $3, $2}'",
		"df -h / | awk 'NR==2{printf \"磁盘: %s/%s (%s)\\n\", $3, $2, $5}'",
		"echo \"VM: $(qm list 2>/dev/null | grep -c running) 运行中\"",
		"echo \"CT: $(pct list 2>/dev/null | grep -c running) 运行中\"",
	}
	out, err := utils.BatchOutput(ctx, cmds, 10)
	if err != nil {
		return "健康检查失败"
	}
	return strings.TrimSpace(string(out))
}

func sendNotification(title, body string) error {
	data, err := os.ReadFile(notifyConfigPath)
	if err != nil {
		return fmt.Errorf("通知未配置，请先运行「配置通知渠道」")
	}
	lines := strings.Split(string(data), "\n")
	var token, chatID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TELEGRAM_BOT_TOKEN=") {
			token = strings.TrimPrefix(line, "TELEGRAM_BOT_TOKEN=")
		}
		if strings.HasPrefix(line, "TELEGRAM_CHAT_ID=") {
			chatID = strings.TrimPrefix(line, "TELEGRAM_CHAT_ID=")
		}
	}
	if token == "" || chatID == "" {
		return fmt.Errorf("通知配置不完整，请检查 %s", notifyConfigPath)
	}

	ctx := context.TODO()
	payload := fmt.Sprintf(
		`chat_id=%s&text=%s&parse_mode=HTML`,
		chatID, urlEncode(fmt.Sprintf("<b>%s</b>\n\n%s", title, body)),
	)
	return utils.BatchRunStdout(ctx, []string{
		fmt.Sprintf(`curl -s -X POST "https://api.telegram.org/bot%s/sendMessage" -d '%s' > /dev/null`, token, payload),
	}, 15)
}

func urlEncode(s string) string {
	s = strings.ReplaceAll(s, "&", "%26")
	s = strings.ReplaceAll(s, "=", "%3D")
	s = strings.ReplaceAll(s, "+", "%2B")
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, "#", "%23")
	return s
}

func configNotification() error {
	fmt.Println("")
	fmt.Println("需要先通过 Telegram @BotFather 创建一个机器人，获取 Token")
	fmt.Println("然后获取 Chat ID（向机器人发消息后访问 https://api.telegram.org/bot<TOKEN>/getUpdates）")
	fmt.Println("")

	tokenPrompt := promptui.Prompt{
		Label: "Telegram Bot Token",
		Mask:  '*',
	}
	token, err := tokenPrompt.Run()
	if err != nil {
		return errContinue
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("Token 不能为空")
	}

	chatPrompt := promptui.Prompt{
		Label: "Telegram Chat ID",
	}
	chatID, err := chatPrompt.Run()
	if err != nil {
		return errContinue
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("Chat ID 不能为空")
	}

	if err := os.MkdirAll(filepath.Dir(notifyConfigPath), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s\nTELEGRAM_CHAT_ID=%s\n", token, chatID)
	if err := os.WriteFile(notifyConfigPath, []byte(content), 0o600); err != nil {
		return err
	}
	fmt.Println("通知配置已保存:", notifyConfigPath)

	testPrompt := promptui.Prompt{
		Label: "发送测试消息? (Y/n)",
	}
	resp, _ := testPrompt.Run()
	if strings.ToLower(strings.TrimSpace(resp)) != "n" {
		return sendNotification("FastPVE", "通知配置成功 ✅")
	}
	return nil
}

func setupScheduledCheck() error {
	fmt.Println("")
	fmt.Println("设置定时健康检查，每日自动运行并推送通知到 Telegram")
	fmt.Println("")

	schedules := []struct {
		label string
		cron  string
	}{
		{"每天 8:00", "0 8 * * *"},
		{"每天 8:00 和 20:00", "0 8,20 * * *"},
		{"每 6 小时", "0 */6 * * *"},
		{"每 12 小时", "0 */12 * * *"},
		{"自定义", ""},
	}
	schedLabels := make([]string, len(schedules))
	for i, s := range schedules {
		schedLabels[i] = s.label
	}

	prompt := promptui.Select{
		Label: "选择检查频率",
		Items: schedLabels,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return errContinue
	}

	cronExpr := schedules[idx].cron
	if cronExpr == "" {
		cronPrompt := promptui.Prompt{
			Label:   "Cron 表达式 (分 时 日 月 周)",
			Default: "0 8 * * *",
		}
		expr, _ := cronPrompt.Run()
		expr = strings.TrimSpace(expr)
		if expr == "" {
			expr = "0 8 * * *"
		}
		cronExpr = expr
	}

	script := fmt.Sprintf(`#!/bin/bash
# FastPVE 定时健康检查 — 由 fastpve 生成
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

REPORT=$(cat <<'EOF'
=== FastPVE 系统状态 ===
EOF
)
REPORT="$REPORT
$(pveversion 2>/dev/null || echo 'PVE: N/A')
运行: $(uptime -p | sed 's/up //')
内存: $(free -h | awk '/Mem/{printf "%%s/%%s", $3, $2}')
磁盘: $(df -h / | awk 'NR==2{printf "%%s/%%s (%%s)", $3, $2, $5}')
VM: $(qm list 2>/dev/null | grep -c running) 运行中
CT: $(pct list 2>/dev/null | grep -c running) 运行中"

if [ -f %s ]; then
	source %s
	curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
		-d "chat_id=${TELEGRAM_CHAT_ID}" \
		-d "text=<b>FastPVE 定时报告</b>%%0A%%0A${REPORT}" \
		-d "parse_mode=HTML" > /dev/null
fi
`, notifyConfigPath, notifyConfigPath)

	scriptPath := "/usr/local/bin/fastpve-healthcheck.sh"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return err
	}

	// Add to cron
	cronLine := fmt.Sprintf("%s root %s\n", cronExpr, scriptPath)
	if err := os.WriteFile("/etc/cron.d/fastpve-healthcheck", []byte(cronLine), 0o644); err != nil {
		return err
	}

	fmt.Printf("定时检查已设置：%s\n", cronExpr)
	fmt.Printf("检查脚本: %s\n", scriptPath)
	fmt.Printf("Cron 配置: /etc/cron.d/fastpve-healthcheck\n")
	return nil
}

func testNotification() error {
	return sendNotification("FastPVE 测试", "这是一条测试消息 ✅\n如果你的 Telegram 收到了，说明配置正确！")
}
