package vmdownloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/utils"
)

func DownloadIstoreIMG(ctx context.Context, d Downloader, isoPath, cachePath, statusPath string, status *downloader.DownloadStatus, ver int) (string, error) {
	switch {
	case status != nil:
		baseFileName := filepath.Base(status.TargetFile)
		fmt.Println("downloading:", baseFileName, "url=\n", status.Url)
		if err := DownloadFile(ctx, d, statusPath, status); err != nil {
			return "", err
		}
		return unzipIstore(ctx, cachePath, isoPath, baseFileName, status.TargetFile)
	case ver >= 0:
		urls, err := GetIstoreUrls(ctx, d, ver)
		if err != nil {
			urls = DefaultIstoreUrls(ver)
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
		fmt.Println("downloading:", filepath.Base(status.TargetFile), "url=\n", status.Url)
		if err := DownloadFile(ctx, d, statusPath, status); err != nil {
			return "", err
		}
		return unzipIstore(ctx, cachePath, isoPath, filepath.Base(status.TargetFile), status.TargetFile)
	default:
		return "", errors.New("no istore download target provided")
	}
}

func unzipIstore(ctx context.Context, cachePath, isoPath, baseFileName, targetFile string) (string, error) {
	fmt.Println("download OK, unzipping and moving file...")
	targetFileName := strings.TrimSuffix(baseFileName, ".gz")
	targetFilePath := filepath.Join(isoPath, targetFileName)
	err := utils.BatchRun(ctx, []string{
		fmt.Sprintf("gunzip -k %s", targetFile),
		fmt.Sprintf("mv %s/%s %s", cachePath, targetFileName, targetFilePath),
		fmt.Sprintf("rm -f %s", targetFile),
	}, 0)
	if err != nil {
		return "", err
	}
	return targetFileName, nil
}

func DefaultIstoreUrls(ver int) []string {
	if ver == 0 {
		return []string{
			"https://dl.istoreos.com/iStoreOS/x86_64_efi/istoreos-24.10.1-2025052311-x86-64-squashfs-combined-efi.img.gz",
		}
	}
	return []string{
		"https://dl.istoreos.com/iStoreOS-22.03/x86_64_efi/istoreos-22.03.7-2025051615-x86-64-squashfs-combined-efi.img.gz",
	}
}

func GetIstoreUrls(ctx context.Context, d Downloader, ver int) ([]string, error) {
	var name string
	if ver == 0 {
		name = "iStoreOS"
	} else {
		name = "iStoreOS-22.03"
	}
	versionStr := fmt.Sprintf("https://fw0.koolcenter.com/%s/x86_64_efi/version.index", name)
	client := d.DefaultClient()
	req, err := http.NewRequestWithContext(ctx, "GET", versionStr, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	versionResp, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	lastVer := strings.Replace(string(versionResp), "\n", "", -1)
	return []string{
		fmt.Sprintf("https://fw.d4ctech.com/%s/x86_64_efi/istoreos-%s-x86-64-squashfs-combined-efi.img.gz", name, lastVer),
		fmt.Sprintf("https://dl.istoreos.com/%s/x86_64_efi/istoreos-%s-x86-64-squashfs-combined-efi.img.gz", name, lastVer),
		fmt.Sprintf("https://fw0.koolcenter.com/%s/x86_64_efi/istoreos-%s-x86-64-squashfs-combined-efi.img.gz", name, lastVer),
	}, nil
}
