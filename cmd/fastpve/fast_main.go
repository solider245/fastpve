package main

import (
	"fmt"
	"log"
	"os"

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
		},
	}
	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
