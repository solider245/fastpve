package main

import (
	"context"
	"fmt"

	"github.com/linkease/fastpve/utils"
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
