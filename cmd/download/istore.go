package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/urfave/cli/v3"
)

func istoreCommand() *cli.Command {
	return &cli.Command{
		Name:  "istore",
		Usage: "Download iStoreOS image",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "version",
				Usage:   "iStoreOS version: 24.10 or 22.03",
				Value:   "24.10",
				Aliases: []string{"v"},
			},
			&cli.BoolFlag{
				Name:  "resume",
				Usage: "Resume from existing status if present",
				Value: true,
			},
			&cli.StringFlag{
				Name:  "iso-path",
				Usage: "Directory for final image",
				Value: defaultISOPath,
			},
			&cli.StringFlag{
				Name:  "cache-path",
				Usage: "Directory for partial downloads/status files",
				Value: defaultCachePath,
			},
			&cli.StringFlag{
				Name:  "status-path",
				Usage: "Override status file path for iStoreOS image",
			},
		},
		Action: downloadIstore,
	}
}

func downloadIstore(ctx context.Context, cmd *cli.Command) error {
	isoPath := cmd.String("iso-path")
	cachePath := cmd.String("cache-path")
	if err := ensureDirs(isoPath, cachePath); err != nil {
		return err
	}
	statusPath := cmd.String("status-path")
	if statusPath == "" {
		statusPath = defaultStatusPath(cachePath, "istore_install.ops")
	}
	resume := cmd.Bool("resume")

	ver, err := parseIstoreVersion(cmd.String("version"))
	if err != nil {
		return err
	}

	downer := downloader.NewDownloader()
	var status *downloader.DownloadStatus
	if resume {
		status, _ = vmdownloader.IsStatusValid(downer, statusPath)
	}
	target, err := vmdownloader.DownloadIstoreIMG(ctx, downer, isoPath, cachePath, statusPath, status, ver)
	if err != nil {
		return err
	}
	fmt.Println("iStoreOS image ready:", filepath.Join(isoPath, target))
	return nil
}
