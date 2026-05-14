package vmdownloader

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/solider245/fastpve/downloader"
)

func DownloadUbuntuISO(ctx context.Context, d Downloader, isoPath, cachePath, statusPath string, status *downloader.DownloadStatus, ubuntuVer int) (string, error) {
	switch {
	case status != nil:
		baseFileName := filepath.Base(status.TargetFile)
		fmt.Println("downloading:", baseFileName, "url=\n", status.Url)
		targetFilePath := filepath.Join(isoPath, baseFileName)
		return downloadAndMove(ctx, d, statusPath, status, targetFilePath)
	case ubuntuVer >= 0:
		urls, err := GetUbuntuUrls(ctx, d, ubuntuVer)
		if err != nil {
			urls = DefaultUbuntuUrls(ubuntuVer)
		}
		urlStr, totalSize, modTime, err := SelectFirstReachable(d, urls)
		if err != nil {
			return "", err
		}
		status = &downloader.DownloadStatus{
			Url:        urlStr,
			TargetFile: filepath.Join(cachePath, path.Base(urlStr)),
			TotalSize:  totalSize,
			ModTime:    modTime,
		}
		fmt.Println("downloading:", path.Base(status.TargetFile), "url=\n", status.Url)
		targetFilePath := filepath.Join(isoPath, filepath.Base(status.TargetFile))
		return downloadAndMove(ctx, d, statusPath, status, targetFilePath)
	default:
		return "", errors.New("no ubuntu download target provided")
	}
}

func DefaultUbuntuUrls(ver int) []string {
	var versionStr1, versionStr2, versionStr3 string
	if ver&1 == 0 {
		versionStr3 = "desktop"
	} else {
		versionStr3 = "live-server"
	}
	switch ver {
	case 0, 1:
		versionStr1 = "22.04"
		versionStr2 = "22.04.5"
	case 2, 3:
		versionStr1 = "24.10"
		versionStr2 = "24.10"
	case 4, 5:
		versionStr1 = "25.04"
		versionStr2 = "25.04"
	}
	return []string{
		fmt.Sprintf("https://mirrors.ustc.edu.cn/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
	}
}

func GetUbuntuUrls(ctx context.Context, d Downloader, ver int) ([]string, error) {
	var versionStr1, versionStr2, versionStr3 string

	if ver&1 == 0 {
		versionStr3 = "desktop"
	} else {
		versionStr3 = "live-server"
	}

	switch ver {
	case 0, 1:
		versionStr1 = "22.04"
		versionStr2 = "22.04.5"
	case 2, 3:
		versionStr1 = "24.10"
		versionStr2 = "24.10"
	case 4, 5:
		versionStr1 = "25.04"
		versionStr2 = "25.04"
	default:
		return nil, fmt.Errorf("unknown Ubuntu version: %d", ver)
	}
	return []string{
		// 中科大
		fmt.Sprintf("https://mirrors.ustc.edu.cn/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
		// 清华
		fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
		// 华为
		fmt.Sprintf("https://repo.huaweicloud.com/ubuntu-releases/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
		// 官方
		fmt.Sprintf("https://releases.ubuntu.com/%s/ubuntu-%s-%s-amd64.iso", versionStr1, versionStr2, versionStr3),
	}, nil
}
