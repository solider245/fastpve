package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/utils"
)

func promptStorageTools() error {
	items := []menuItem{
		{"存储概览", showStorageOverview},
		{"备份 /etc/pve 配置", backupPVEConfig},
		{"清理 journal 日志", cleanJournalLogs},
		{"返回", func() error { return errContinue }},
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}

	prompt := promptui.Select{
		Label: "存储与备份",
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

func showStorageOverview() error {
	ctx := context.TODO()

	zpoolOut, _ := utils.BatchOutput(ctx, []string{"zpool list 2>/dev/null || echo '无 ZFS 池'"}, 5)
	dfRootOut, _ := utils.BatchOutput(ctx, []string{"df -h / 2>/dev/null"}, 5)
	dfVzOut, _ := utils.BatchOutput(ctx, []string{"df -h /var/lib/vz 2>/dev/null || echo '/var/lib/vz 无单独分区'"}, 5)
	arcOut, _ := utils.BatchOutput(ctx, []string{"cat /proc/spl/kstat/zfs/arcstats 2>/dev/null | grep -E '^size|^c_max' || echo 'ZFS ARC 不可用'"}, 5)
	pveshOut, _ := utils.BatchOutput(ctx, []string{"pvesh get /storage 2>/dev/null --output-format json 2>/dev/null | jq -r '.[] | select(.content | contains(\"rootdir\") or contains(\"images\")) | .storage + \" (\" + .type + \")\"' 2>/dev/null || echo 'pvesh 不可用'"}, 5)

	var b strings.Builder
	b.WriteString("========== 存储概览 ==========\n")
	b.WriteString("ZFS 池:\n")
	b.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(string(zpoolOut))))
	b.WriteString("\n磁盘使用:\n")
	b.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(string(dfRootOut))))
	b.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(string(dfVzOut))))
	b.WriteString(fmt.Sprintf("\nZFS ARC: %s\n", strings.TrimSpace(string(arcOut))))
	b.WriteString(fmt.Sprintf("\n存储:\n  %s\n", strings.TrimSpace(string(pveshOut))))
	b.WriteString("\n备注: local 放 ISO/模板, local-lvm 放 VM 磁盘\n")
	b.WriteString("==============================\n")

	fmt.Print(b.String())
	return nil
}

func backupPVEConfig() error {
	ctx := context.TODO()
	backupPath := fmt.Sprintf("/root/pve-config-backup-%s.tar.gz", time.Now().Format("20060102_1504"))

	return utils.BatchRunStdout(ctx, []string{
		fmt.Sprintf("tar czf %s /etc/pve", backupPath),
		fmt.Sprintf("ls -lh %s", backupPath),
	}, 0)
}

func cleanJournalLogs() error {
	ctx := context.TODO()

	out, _ := utils.BatchOutput(ctx, []string{"journalctl --disk-usage"}, 5)
	fmt.Print(string(out))

	prompt := promptui.Prompt{
		Label:   "保留日志大小 (默认 500M)",
		Default: "500M",
	}
	result, err := prompt.Run()
	if err != nil {
		return errContinue
	}
	result = strings.TrimSpace(result)
	if result == "" {
		result = "500M"
	}

	return utils.BatchRunStdout(ctx, []string{
		fmt.Sprintf("journalctl --vacuum-size=%s", result),
	}, 0)
}
