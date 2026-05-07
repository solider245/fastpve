package vmdownloader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/linkease/fastpve/downloader"
	"github.com/linkease/fastpve/utils"
)

func DownloadDDImage(ctx context.Context, d Downloader, isoPath, cachePath, statusPath string, status *downloader.DownloadStatus, url string) (string, error) {
	switch {
	case status != nil:
		baseFileName := filepath.Base(status.TargetFile)
		fmt.Println("downloading:", baseFileName, "url=\n", status.Url)
		if err := DownloadFile(ctx, d, statusPath, status); err != nil {
			return "", err
		}
		return decompressDD(ctx, cachePath, isoPath, status.TargetFile)
	case url != "":
		headSize, headModTime, err := d.HeadInfo(url)
		if err != nil {
			return "", fmt.Errorf("cannot reach download URL: %w", err)
		}
		fileName := path.Base(url)
		if fileName == "" || fileName == "." || fileName == "/" {
			return "", fmt.Errorf("cannot determine filename from URL: %s", url)
		}
		status = &downloader.DownloadStatus{
			Url:        url,
			TargetFile: filepath.Join(cachePath, fileName),
			TotalSize:  headSize,
			ModTime:    headModTime,
		}
		fmt.Println("downloading:", fileName, "url=\n", status.Url)
		if err := DownloadFile(ctx, d, statusPath, status); err != nil {
			return "", err
		}
		return decompressDD(ctx, cachePath, isoPath, status.TargetFile)
	default:
		return "", errors.New("no DD image URL or resume status provided")
	}
}

func decompressDD(ctx context.Context, cachePath, isoPath, targetFile string) (string, error) {
	baseFileName := filepath.Base(targetFile)
	lower := strings.ToLower(baseFileName)

	var decompressor string
	var extractedName string

	switch {
	case strings.HasSuffix(lower, ".gz"):
		decompressor = "gunzip -k"
		extractedName = strings.TrimSuffix(baseFileName, ".gz")
	case strings.HasSuffix(lower, ".xz"):
		decompressor = "unxz -k"
		extractedName = strings.TrimSuffix(baseFileName, ".xz")
	case strings.HasSuffix(lower, ".zst"), strings.HasSuffix(lower, ".zstd"):
		decompressor = "unzstd --keep"
		extractedName = strings.TrimSuffix(baseFileName, ".zst")
		if extractedName == baseFileName {
			extractedName = strings.TrimSuffix(baseFileName, ".zstd")
		}
	default:
		// No recognized compression extension, treat as raw and move to isoPath.
		fmt.Println("no compression detected, moving file to iso path...")
		destPath := filepath.Join(isoPath, baseFileName)
		if err := os.Rename(targetFile, destPath); err != nil {
			return "", err
		}
		return baseFileName, nil
	}

	fmt.Printf("decompressing %s...\n", baseFileName)
	extractedPath := filepath.Join(cachePath, extractedName)
	destPath := filepath.Join(isoPath, extractedName)
	err := utils.BatchRun(ctx, []string{
		fmt.Sprintf("%s %s", decompressor, targetFile),
		fmt.Sprintf("mv %s %s", extractedPath, destPath),
		fmt.Sprintf("rm -f %s", targetFile),
	}, 0)
	if err != nil {
		return "", err
	}
	return extractedName, nil
}
