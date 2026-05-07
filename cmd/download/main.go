package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	if err := newApp().Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func newApp() *cli.Command {
	return &cli.Command{
		Name:  "fastpve-download",
		Usage: "Download VM images (Windows/Ubuntu/iStoreOS/DD/VirtIO)",
		Commands: []*cli.Command{
			windowsCommand(),
			ubuntuCommand(),
			istoreCommand(),
			ddCommand(),
			virtioCommand(),
		},
	}
}
