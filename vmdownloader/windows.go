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
	WinServer2025
	WinServer2022
)

// windowsManualURLs returns known manual download sources for each Windows version.
func windowsManualURLs(version int, editionName string) string {
	switch version {
	case Win11:
		return "Windows 11 官方镜像: https://www.microsoft.com/software-download/windows11\n  或访问: https://next.itellyou.cn/"
	case Win10:
		return "Windows 10 官方镜像: https://www.microsoft.com/software-download/windows10\n  或访问: https://next.itellyou.cn/"
	case Win7:
		return "Windows 7 官方镜像: https://www.microsoft.com/software-download/windows7\n  或访问: https://next.itellyou.cn/"
	case WinServer2025:
		return "Windows Server 2025 评估版:\n  https://www.microsoft.com/zh-cn/evalcenter/download-windows-server-2025\n  https://www.microsoft.com/en-us/evalcenter/download-windows-server-2025"
	case WinServer2022:
		return "Windows Server 2022 评估版:\n  https://www.microsoft.com/zh-cn/evalcenter/download-windows-server-2022\n  https://www.microsoft.com/en-us/evalcenter/download-windows-server-2022"
	default:
		return "Windows 镜像下载: https://next.itellyou.cn/"
	}
}

// ghcrManualHint returns instructions for checking GHCR connectivity.
func ghcrManualHint() string {
	return "提示: 可尝试在 PVE 系统工具中切换 GHCR 镜像源以获得更稳定的下载"
}

// DownloadWindowsISO resumes a pending download when status is provided, or starts a new download for the given version/edition.
func DownloadWindowsISO(ctx context.Context, d Downloader, quickGetPath, isoPath, statusPath string, status *downloader.DownloadStatus, version int, editionName string) (string, error) {
	// Resume from status
	if status != nil && version < 0 {
		realPath := strings.TrimSuffix(status.TargetFile, ".syn")
		fmt.Println("downloading:", filepath.Base(realPath))
		return downloadAndMove(ctx, d, statusPath, status, realPath)
	}

	if version < 0 {
		return "", errors.New("windows version missing")
	}

	if version == Win7 || version == WinServer2025 || version == WinServer2022 {
		target, ghcrErr := downloadWindowsFromGHCR(ctx, isoPath, version, editionName)
		if ghcrErr == nil {
			return target, nil
		}
		return "", fmt.Errorf("GHCR 下载失败\n%s\n%s",
			windowsManualURLs(version, editionName),
			ghcrManualHint())
	}

	if editionName == "" {
		return "", errors.New("windows edition missing")
	}

	// Clean up old status files before starting a fresh download.
	if status != nil {
		_ = os.Remove(status.TargetFile)
		_ = os.Remove(statusPath)
	}

	// Priority 1: GHCR with Chinese mirrors (fast in China)
	target, ghcrErr := downloadWindowsFromGHCR(ctx, isoPath, version, editionName)
	if ghcrErr == nil {
		return target, nil
	}
	fmt.Println("GHCR 下载失败，尝试从微软官方源获取下载链接...")
	fmt.Println(ghcrManualHint())

	// Priority 2: resolve official download URL via quickget
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
	if err == nil {
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

	// All sources failed — show manual instructions
	return "", fmt.Errorf("所有下载源均不可用\n%s\n%s",
		windowsManualURLs(version, editionName),
		ghcrManualHint())
}

func resolveWindowsURL(ctx context.Context, d Downloader, quickGetPath, tag, winVer, editionName string) (string, int64, time.Time, error) {
	args := []string{"--url", "windows", winVer, editionName}
	fmt.Println("获取下载URL，15s 超时...")
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
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
