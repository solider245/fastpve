package main

import (
	"context"
	"fmt"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/urfave/cli/v3"
)

func virtioCommand() *cli.Command {
	return &cli.Command{
		Name:  "virtio",
		Usage: "Download VirtIO driver ISO",
		Flags: []cli.Flag{
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
				Usage: "Override status file path for VirtIO ISO",
			},
		},
		Action: downloadVirtio,
	}
}

func downloadVirtio(ctx context.Context, cmd *cli.Command) error {
	isoPath := cmd.String("iso-path")
	cachePath := cmd.String("cache-path")
	if err := ensureDirs(isoPath, cachePath); err != nil {
		return err
	}
	statusPath := cmd.String("status-path")
	if statusPath == "" {
		statusPath = defaultStatusPath(cachePath, "windows_virtio.ops")
	}
	resume := cmd.Bool("resume")

	downer := downloader.NewDownloader()
	var status *downloader.DownloadStatus
	if resume {
		status, _ = vmdownloader.IsStatusValid(downer, statusPath)
	}
	target, err := vmdownloader.DownloadVirtIO(ctx, downer, isoPath, statusPath, status)
	if err != nil {
		return err
	}
	fmt.Println("VirtIO ISO ready:", target)
	return nil
}
