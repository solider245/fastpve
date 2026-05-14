package main

import (
	"context"
	"fmt"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/urfave/cli/v3"
)

func ubuntuCommand() *cli.Command {
	return &cli.Command{
		Name:  "ubuntu",
		Usage: "Download Ubuntu ISO",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "version",
				Usage:   "Ubuntu version: 22.04-desktop, 22.04-server, 24.10-desktop, 24.10-server, 25.04-desktop, 25.04-server",
				Value:   "22.04-desktop",
				Aliases: []string{"v"},
			},
			&cli.BoolFlag{
				Name:  "resume",
				Usage: "Resume from existing status if present",
				Value: true,
			},
			&cli.StringFlag{
				Name:  "iso-path",
				Usage: "Directory for final ISO",
				Value: defaultISOPath,
			},
			&cli.StringFlag{
				Name:  "cache-path",
				Usage: "Directory for partial downloads/status files",
				Value: defaultCachePath,
			},
			&cli.StringFlag{
				Name:  "status-path",
				Usage: "Override status file path for Ubuntu ISO",
			},
		},
		Action: downloadUbuntu,
	}
}

func downloadUbuntu(ctx context.Context, cmd *cli.Command) error {
	isoPath := cmd.String("iso-path")
	cachePath := cmd.String("cache-path")
	if err := ensureDirs(isoPath, cachePath); err != nil {
		return err
	}
	statusPath := cmd.String("status-path")
	if statusPath == "" {
		statusPath = defaultStatusPath(cachePath, "ubuntu_install.ops")
	}
	resume := cmd.Bool("resume")

	ubuntuVer, err := parseUbuntuVersion(cmd.String("version"))
	if err != nil {
		return err
	}

	downer := downloader.NewDownloader()
	var status *downloader.DownloadStatus
	if resume {
		status, _ = vmdownloader.IsStatusValid(downer, statusPath)
	}
	target, err := vmdownloader.DownloadUbuntuISO(ctx, downer, isoPath, cachePath, statusPath, status, ubuntuVer)
	if err != nil {
		return err
	}
	fmt.Println("Ubuntu ISO ready:", target)
	return nil
}
