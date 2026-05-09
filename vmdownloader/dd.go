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

func FinalImageName(url string) string {
	fileName := path.Base(url)
	lower := strings.ToLower(fileName)
	switch {
	case strings.HasSuffix(lower, ".gz"):
		return strings.TrimSuffix(fileName, ".gz")
	case strings.HasSuffix(lower, ".xz"):
		return strings.TrimSuffix(fileName, ".xz")
	case strings.HasSuffix(lower, ".zst"), strings.HasSuffix(lower, ".zstd"):
		name := strings.TrimSuffix(fileName, ".zst")
		if name == fileName {
			name = strings.TrimSuffix(fileName, ".zstd")
		}
		return name
	default:
		return fileName
	}
}

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
		// Check if the final image already exists in isoPath
		finalName := FinalImageName(url)
		if finalName != "" {
			if _, err := os.Stat(filepath.Join(isoPath, finalName)); err == nil {
				fmt.Println("镜像已存在:", finalName)
				return finalName, nil
			}
		}

		downloadURL := url
		if strings.Contains(url, "github.com") {
			proxyURL := strings.Replace(url, "github.com", "gh.linkease.net:5443", 1)
			if _, _, err := d.HeadInfo(proxyURL); err == nil {
				downloadURL = proxyURL
			}
		}
		headSize, headModTime, err := d.HeadInfo(downloadURL)
		if err != nil {
			return "", fmt.Errorf("cannot reach download URL: %w", err)
		}
		fileName := path.Base(downloadURL)
		if fileName == "" || fileName == "." || fileName == "/" {
			return "", fmt.Errorf("cannot determine filename from URL: %s", url)
		}
		status = &downloader.DownloadStatus{
			Url:        downloadURL,
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
