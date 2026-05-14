package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	cliApp := &cli.App{
		Name:  "fastpve",
		Usage: "Fast install systems on pve!",
		Action: func(c *cli.Context) error {
			InitConfig()
			return mainPrompt()
		},
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "show version",
				Action: func(c *cli.Context) error {
					fmt.Println(version)
					return nil
				},
			},
			{
				Name:  "ai",
				Usage: "AI 助手 - 自然语言管理 PVE",
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
				Usage: "PI 协议 - JSON 输入输出后端",
				Action: func(c *cli.Context) error {
					return runPI()
				},
			},
		},
	}
	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
