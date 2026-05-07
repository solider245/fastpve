package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/linkease/fastpve/downloader"
	"github.com/linkease/fastpve/vmdownloader"
	"github.com/urfave/cli/v3"
)

func ddCommand() *cli.Command {
	var (
		isoPath   string
		cachePath string
		resume    bool
		biosMode  string
		url       string
	)

	return &cli.Command{
		Name:  "dd",
		Usage: "Download a DD disk image from a URL",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				Usage:       "URL of the DD image to download",
				Sources:     cli.EnvVars("DD_URL"),
				Destination: &url,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "iso-path",
				Usage:       "Directory for final .img file",
				Value:       defaultISOPath,
				Sources:     cli.EnvVars("ISO_PATH"),
				Destination: &isoPath,
			},
			&cli.StringFlag{
				Name:        "cache-path",
				Usage:       "Directory for download cache and temporary files",
				Value:       defaultCachePath,
				Sources:     cli.EnvVars("CACHE_PATH"),
				Destination: &cachePath,
			},
			&cli.BoolFlag{
				Name:        "resume",
				Usage:       "Resume the last incomplete download",
				Sources:     cli.EnvVars("RESUME"),
				Destination: &resume,
			},
			&cli.StringFlag{
				Name:        "bios",
				Usage:       "BIOS mode hint: uefi or bios (informational only for download)",
				Value:       "uefi",
				Sources:     cli.EnvVars("DD_BIOS"),
				Destination: &biosMode,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := ensureDirs(cachePath, isoPath); err != nil {
				return err
			}

			statusPath := defaultStatusPath(cachePath, "dd_install.ops")
			d := downloader.NewDownloader()

			var status *downloader.DownloadStatus
			if resume {
				var err error
				status, err = vmdownloader.IsStatusValid(d, statusPath)
				if err != nil {
					return fmt.Errorf("no valid resume state: %w", err)
				}
			}

			imgName, err := vmdownloader.DownloadDDImage(ctx, d, isoPath, cachePath, statusPath, status, url)
			if err != nil {
				return err
			}

			fmt.Println("download complete:", filepath.Join(isoPath, imgName))
			return nil
		},
	}
}
