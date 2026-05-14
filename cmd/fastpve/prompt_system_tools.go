package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/solider245/fastpve/utils"
	"github.com/manifoldco/promptui"
)

func promptSystemTools() error {
	items := []menuItem{
		{"更换软件源", promptForSources},
		{"安装Docker", promptForDocker},
		{"一键核显直通", promptForGPUPassThrough},
		{"去除订阅弹窗", promptRemoveSubscriptionNag},
		{"安装常用工具", promptInstallCommonTools},
		{"系统更新", promptSystemUpdate},
		{"清理旧内核", promptCleanOldKernels},
		{"ZFS ARC 调优", promptZFSTuning},
		{"CPU 调度器切换", promptCPUScaling},
		{"返回", func() error { return errContinue }},
	}

	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}

	prompt := promptui.Select{
		Label: "PVE系统工具",
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
		} else {
			fmt.Println("操作成功")
		}
	}
}

func promptRemoveSubscriptionNag() error {
	ctx := context.TODO()
	return utils.BatchRunStdout(ctx, []string{
		`sed -Ezi.bak "s/(Ext.Msg.show\(\{\s+title: gettext\('No valid sub)/void\(\{ \/\/\1/g" /usr/share/javascript/proxmox-widget-toolkit/proxmoxlib.js`,
		"systemctl restart pveproxy.service",
	}, 0)
}

func promptInstallCommonTools() error {
	ctx := context.TODO()
	return utils.BatchRunStdout(ctx, []string{
		"apt update && apt install -y curl wget vim htop net-tools lsof",
	}, 0)
}

func promptSystemUpdate() error {
	ctx := context.TODO()
	return utils.BatchRunStdout(ctx, []string{
		"apt update && apt dist-upgrade -y",
	}, 0)
}

func promptCleanOldKernels() error {
	ctx := context.TODO()
	// List old kernels first
	out, err := utils.BatchOutput(ctx, []string{
		"dpkg --list | grep -E 'linux-image-[0-9]' | grep -v $(uname -r | sed 's/-/.*/') | awk '{print $2}'",
	}, 5)
	if err != nil || strings.TrimSpace(string(out)) == "" {
		fmt.Println("  没有可清理的旧内核")
		return nil
	}
	kernels := strings.TrimSpace(string(out))
	fmt.Printf("  可清理的内核:\n%s\n", kernels)

	confirmPrompt := promptui.Prompt{
		Label: "确认清理以上内核？(yes/NO)",
	}
	confirm, _ := confirmPrompt.Run()
	if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
		return nil
	}
	return utils.BatchRunStdout(ctx, []string{
		"dpkg --list | grep -E 'linux-image-[0-9]' | grep -v $(uname -r) | awk '{print $2}' | xargs -r apt purge -y",
		"update-grub",
	}, 0)
}

func promptZFSTuning() error {
	ctx := context.TODO()
	out, err := utils.BatchOutput(ctx, []string{"zpool list -H 2>/dev/null || true"}, 5)
	if err != nil || strings.TrimSpace(string(out)) == "" {
		fmt.Println("  未检测到 ZFS 池")
		return nil
	}
	fmt.Printf("  ZFS 池:\n%s\n", strings.TrimSpace(string(out)))

	// Get total memory for default suggestion
	memOut, _ := utils.BatchOutput(ctx, []string{"free -m | awk '/Mem/{print $2}'"}, 3)
	totalMB := 4096
	if m, err := fmt.Sscanf(strings.TrimSpace(string(memOut)), "%d", &totalMB); m != 1 || err != nil {
		totalMB = 4096
	}
	defMB := totalMB / 2

	arcPrompt := promptui.Prompt{
		Label:   fmt.Sprintf("ZFS ARC 最大值 (MB, 默认 %d)", defMB),
		Default: fmt.Sprintf("%d", defMB),
	}
	result, err := arcPrompt.Run()
	if err != nil {
		return errContinue
	}
	result = strings.TrimSpace(result)
	if result == "" {
		result = fmt.Sprintf("%d", defMB)
	}
	var arcMB int
	fmt.Sscanf(result, "%d", &arcMB)
	if arcMB <= 0 {
		arcMB = defMB
	}

	arcBytes := int64(arcMB) * 1024 * 1024
	content := fmt.Sprintf("options zfs zfs_arc_max=%d\n", arcBytes)
	if err := os.WriteFile("/etc/modprobe.d/zfs.conf", []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  ZFS ARC 最大值已设置为 %dMB，重启后生效\n", arcMB)
	return nil
}

func promptCPUScaling() error {
	governorFile := "/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"
	data, err := os.ReadFile(governorFile)
	current := ""
	if err == nil {
		current = strings.TrimSpace(string(data))
	}
	fmt.Printf("  当前调度器: %s\n", current)

	items := []string{"performance（性能）", "powersave（省电）", "ondemand（按需）"}
	prompt := promptui.Select{
		Label: "选择CPU调度器",
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return errContinue
	}

	governors := []string{"performance", "powersave", "ondemand"}
	gov := governors[idx]

	return utils.BatchRunStdout(context.TODO(), []string{
		fmt.Sprintf("cpupower frequency-set -g %s 2>/dev/null || echo 'cpupower 未安装，请运行 apt install linux-cpupower'", gov),
	}, 0)
}
