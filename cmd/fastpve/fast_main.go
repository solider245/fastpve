package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	cliApp := &cli.App{
		Name:  "fastpve",
		Usage: "PVE AI 运维助手 — 自然语言管理 Proxmox VE",
		Action: func(c *cli.Context) error {
			InitConfig()
			if err := InitDB(); err == nil {
				startPerfCollector(5 * time.Minute)
			}
			// AI First: 有 Key → AI 会话，无 Key → 配置向导
			if Cfg != nil && Cfg.AIKey != "" {
				return runInteractiveAI()
			}
			return setupWizard()
		},
		Commands: []*cli.Command{
			{
				Name:  "menu",
				Usage: "传统 TUI 菜单（所有功能的传统操作界面）",
				Action: func(c *cli.Context) error {
					InitConfig()
					_ = InitDB()
					return mainPrompt()
				},
			},
			{
				Name:  "ai",
				Usage: "AI 单次查询 — fastpve ai \"查看系统状态\"",
				Action: func(c *cli.Context) error {
					input := strings.Join(c.Args().Slice(), " ")
					if input == "" {
						data, _ := os.ReadFile(os.Stdin.Name())
						input = strings.TrimSpace(string(data))
					}
					return runAI(input)
				},
			},
			{
				Name:  "pi",
				Usage: "PI 协议 — JSON stdin/stdout 后端接口",
				Action: func(c *cli.Context) error {
					return runPI()
				},
			},
			{
				Name:  "install-plugin",
				Usage: "安装终端无感插件（自动识别 zsh/bash）",
				Action: func(c *cli.Context) error {
					return installPlugin()
				},
			},
			{
				Name:  "uninstall-plugin",
				Usage: "移除终端无感插件",
				Action: func(c *cli.Context) error {
					return disablePlugin()
				},
			},
			{
				Name:  "upgrade",
				Usage: "自更新到最新版本",
				Action: func(c *cli.Context) error {
					return selfUpgrade()
				},
			},
			{
				Name:  "version",
				Usage: "显示版本号",
				Action: func(c *cli.Context) error {
					fmt.Println(version)
					return nil
				},
			},
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
