package vmdownloader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/quickget"
	"github.com/solider245/fastpve/utils"
)

const (
	Win11 = iota
	Win10
	Win7
)

// DownloadWindowsISO resumes a pending download when status is provided, or starts a new download for the given version/edition.
// version should match the quickget expectation (e.g. 0 for Win11, 1 for Win10).
func DownloadWindowsISO(ctx context.Context, d Downloader, quickGetPath, isoPath, statusPath string, status *downloader.DownloadStatus, version int, editionName string) (string, error) {
	if status != nil && version < 0 {
		realPath := strings.TrimSuffix(status.TargetFile, ".syn")
		fmt.Println("downloading:", filepath.Base(realPath))
		return downloadAndMove(ctx, d, statusPath, status, realPath)
	}

	if version < 0 {
		return "", errors.New("windows version missing")
	}

	if version == Win7 {
		return downloadWindowsFromGHCR(ctx, isoPath, version, editionName)
	}

	if editionName == "" {
		return "", errors.New("windows edition missing")
	}

	// Clean up old status files before starting a fresh download.
	if status != nil {
		_ = os.Remove(status.TargetFile)
		_ = os.Remove(statusPath)
		status = nil
	}

	var winVer string
	if version == 0 {
		winVer = "11"
	} else {
		winVer = "10"
	}
	tag := strings.Join([]string{
		"windows",
		winVer,
		utils.CleanString(editionName),
	}, "-")

	urlStr, totalSize, modTime, err := resolveWindowsURL(ctx, d, quickGetPath, tag, winVer, editionName)
	if err != nil {
		fmt.Println("Resolve Windows download URL failed:", err, "\n尝试使用 GHCR 作为备用下载源...")
		target, ghcrErr := downloadWindowsFromGHCR(ctx, isoPath, version, editionName)
		if ghcrErr == nil {
			return target, nil
		}
		return "", fmt.Errorf("resolve windows url: %w; GHCR fallback: %v", err, ghcrErr)
	}

	status = &downloader.DownloadStatus{
		Url:        urlStr,
		TargetFile: filepath.Join(isoPath, tag+".iso.syn"),
		TotalSize:  totalSize,
		ModTime:    modTime,
	}
	realPath := strings.TrimSuffix(status.TargetFile, ".syn")
	fmt.Println("downloading:", filepath.Base(realPath))
	return downloadAndMove(ctx, d, statusPath, status, realPath)
}

func resolveWindowsURL(ctx context.Context, d Downloader, quickGetPath, tag, winVer, editionName string) (string, int64, time.Time, error) {
	args := []string{"--url", "windows", winVer, editionName}
	fmt.Println("获取下载URL，30s 超时...")
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	urlStr, _ := quickget.GetSystemURL(ctx2, quickGetPath, args)
	if urlStr != "" {
		if err := d.PutRemoteURL(ctx, tag, urlStr); err != nil && !errors.Is(err, downloader.ErrRemoteURLCacheDisabled) {
			return "", 0, time.Time{}, err
		}
		totalSize, modTime, err := d.HeadInfo(urlStr)
		if err != nil {
			return "", 0, time.Time{}, err
		}
		fmt.Println("获取下载URL成功，开始下载:", urlStr)
		return urlStr, totalSize, modTime, nil
	}

	if !d.RemoteURLCacheEnabled() {
		return "", 0, time.Time{}, errors.New("获取下载URL失败，且未启用远程缓存")
	}

	fmt.Println("获取下载URL失败，重新获取...")
	urls, err := d.GetRemoteURLs(ctx, tag)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	var candidates []string
	for _, u := range urls {
		if strings.Contains(u, "virtio-win") {
			continue
		}
		candidates = append(candidates, u)
	}
	return SelectFirstReachable(d, candidates)
}

func DownloadVirtIO(ctx context.Context, d Downloader, isoPath, statusPath string, status *downloader.DownloadStatus) (string, error) {
	if status != nil {
		realPath := strings.TrimSuffix(status.TargetFile, ".syn")
		fmt.Println("downloading:", filepath.Base(realPath), "url=\n", status.Url)
		if _, err := downloadAndMove(ctx, d, statusPath, status, realPath); err == nil {
			return realPath, nil
		}
	}

	urls := []string{
		"https://dl.istoreos.com/iStoreOS/Virtual/virtio-win-0.1.271.iso",
		"https://fw0.koolcenter.com/iStoreOS/Virtual/virtio-win-0.1.271.iso",
		"https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.271-1/virtio-win-0.1.271.iso",
	}
	virtioURL, totalSize, modTime, err := SelectFirstReachable(d, urls)
	if err != nil {
		return "", err
	}

	status = &downloader.DownloadStatus{
		Url:        virtioURL,
		TargetFile: filepath.Join(isoPath, path.Base(virtioURL)+".syn"),
		TotalSize:  totalSize,
		ModTime:    modTime,
	}

	realPath := strings.TrimSuffix(status.TargetFile, ".syn")
	fmt.Println("downloading:", filepath.Base(realPath), "url=\n", status.Url)
	return downloadAndMove(ctx, d, statusPath, status, realPath)
}
