package vmdownloader

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/utils"
)

var ErrNoReachableURL = errors.New("no reachable download URL")

// Downloader captures the download behaviours needed by the vm downloader.
type Downloader interface {
	HeadInfo(urlStr string) (int64, time.Time, error)
	ResumableDownloader(ctx context.Context, urlStr, targetFile string, status *downloader.DownloadStatus, progressCh chan *downloader.ProgressInfo) error
	DownloadStatusVerify(status *downloader.DownloadStatus, remoteSize int64, remoteModTime time.Time) bool
	DefaultClient() *http.Client
	RemoteURLCacheEnabled() bool
	PutRemoteURL(ctx context.Context, key, urlStr string) error
	GetRemoteURLs(ctx context.Context, key string) ([]string, error)
}

// IsStatusValid validates an existing status file to ensure the remote target still matches.
func IsStatusValid(d Downloader, statusPath string) (*downloader.DownloadStatus, error) {
	var statusValid bool
	status, err := downloader.ReadUpdateDownload(statusPath)
	if err == nil {
		remoteSize, remoteModTime, err := d.HeadInfo(status.Url)
		if err != nil {
			return nil, err
		}
		statusValid = d.DownloadStatusVerify(status, remoteSize, remoteModTime)
		if !statusValid {
			return nil, errors.New("download status is invalid")
		}
	}
	return status, err
}

// DownloadFile downloads a file with progress reporting and persists status updates.
func DownloadFile(ctx context.Context, d Downloader, statusPath string, status *downloader.DownloadStatus) error {
	progressCh := make(chan *downloader.ProgressInfo, 8)
	go func() {
		for progress := range progressCh {
			downloader.UpdateDownloadStatus(progress.Status, statusPath)
			log.Println("speed=", utils.ByteCountDecimal(uint64(progress.Speed)), "progress=", progress.Progress)
		}
	}()
	err := d.ResumableDownloader(ctx, status.Url, status.TargetFile, status, progressCh)
	close(progressCh)
	if err == nil {
		time.Sleep(time.Second)
		os.Remove(statusPath)
	}
	return err
}

// SelectFirstReachable returns the first URL that responds with a valid HEAD response.
func SelectFirstReachable(d Downloader, urls []string) (string, int64, time.Time, error) {
	var lastErr error
	for _, u := range urls {
		totalSize, modTime, err := d.HeadInfo(u)
		if err != nil {
			lastErr = err
			continue
		}
		return u, totalSize, modTime, nil
	}
	if lastErr == nil {
		lastErr = ErrNoReachableURL
	}
	return "", 0, time.Time{}, lastErr
}

func downloadAndMove(ctx context.Context, d Downloader, statusPath string, status *downloader.DownloadStatus, destPath string) (string, error) {
	if err := DownloadFile(ctx, d, statusPath, status); err != nil {
		return "", err
	}
	if err := os.Rename(status.TargetFile, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}
