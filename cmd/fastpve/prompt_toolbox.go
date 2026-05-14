package main

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/utils"
)

func promptToolbox() error {
	items := []menuItem{
		{"btop — 现代化资源监控", runBtop},
		{"ncdu — 磁盘空间分析", runNCDU},
		{"iperf3 — 网络带宽测试", runIperf3},
		{"pveperf — PVE 硬件性能基线", runPVEPerf},
		{"systemd-cgtop — VM/CT 资源占用", runSystemdCgtop},
		{"mtr — 路由追踪诊断", runMTR},
		{"speedtest-cli — 外网带宽测试", runSpeedtest},
		{"返回", func() error { return errContinue }},
	}

	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}

	prompt := promptui.Select{
		Label: "命令行工具箱",
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

func runBtop() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"apt install -y -qq btop",
	}, 0)
	if err != nil {
		fmt.Println("btop 安装失败:", err)
		return nil
	}
	return utils.BatchRunStdout(ctx, []string{
		"btop",
	}, 0)
}

func runNCDU() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"apt install -y -qq ncdu",
	}, 0)
	if err != nil {
		fmt.Println("ncdu 安装失败:", err)
		return nil
	}
	return utils.BatchRunStdout(ctx, []string{
		"ncdu",
	}, 0)
}

func runIperf3() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"apt install -y -qq iperf3",
	}, 0)
	if err != nil {
		fmt.Println("iperf3 安装失败:", err)
		return nil
	}

	items := []string{"启动服务端 (Server)", "连接服务端 (Client)", "返回"}
	prompt := promptui.Select{
		Label: "iperf3 模式",
		Items: items,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil
	}

	switch idx {
	case 0:
		return utils.BatchRunStdout(ctx, []string{
			"iperf3 -s",
		}, 0)
	case 1:
		ipPrompt := promptui.Prompt{
			Label: "服务端 IP 地址",
		}
		ip, err := ipPrompt.Run()
		if err != nil {
			return nil
		}
		if ip == "" {
			fmt.Println("IP 地址不能为空")
			return nil
		}
		err = utils.BatchRunStdout(ctx, []string{
			fmt.Sprintf("iperf3 -c %s", ip),
		}, 0)
		if err != nil {
			return err
		}
		fmt.Print("\n按 Enter 键返回菜单...")
		fmt.Scanln()
		return nil
	}
	return nil
}

func runPVEPerf() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"pveperf",
	}, 0)
	if err != nil {
		return err
	}
	fmt.Print("\n按 Enter 键返回菜单...")
	fmt.Scanln()
	return nil
}

func runSystemdCgtop() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"apt install -y -qq systemd-cgtop",
	}, 0)
	if err != nil {
		fmt.Println("systemd-cgtop 安装失败:", err)
		return nil
	}
	return utils.BatchRunStdout(ctx, []string{
		"systemd-cgtop",
	}, 0)
}

func runMTR() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"apt install -y -qq mtr-tiny",
	}, 0)
	if err != nil {
		fmt.Println("mtr 安装失败:", err)
		return nil
	}

	ipPrompt := promptui.Prompt{
		Label: "目标 IP 或域名",
	}
	target, err := ipPrompt.Run()
	if err != nil {
		return nil
	}
	if target == "" {
		fmt.Println("目标地址不能为空")
		return nil
	}

	err = utils.BatchRunStdout(ctx, []string{
		fmt.Sprintf("mtr -r %s", target),
	}, 0)
	if err != nil {
		return err
	}
	fmt.Print("\n按 Enter 键返回菜单...")
	fmt.Scanln()
	return nil
}

func runSpeedtest() error {
	ctx := context.TODO()
	err := utils.BatchRunStdout(ctx, []string{
		"apt install -y -qq speedtest-cli",
	}, 0)
	if err != nil {
		fmt.Println("speedtest-cli 安装失败:", err)
		return nil
	}
	err = utils.BatchRunStdout(ctx, []string{
		"speedtest-cli --simple",
	}, 0)
	if err != nil {
		return err
	}
	fmt.Print("\n按 Enter 键返回菜单...")
	fmt.Scanln()
	return nil
}
