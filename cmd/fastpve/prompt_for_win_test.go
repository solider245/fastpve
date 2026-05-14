package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/vmdownloader"
)

type fakeDownloader struct {
	head map[string]headResult
}

type headResult struct {
	size int64
	mod  time.Time
	err  error
}

func (f *fakeDownloader) HeadInfo(urlStr string) (int64, time.Time, error) {
	res, ok := f.head[urlStr]
	if !ok {
		return 0, time.Time{}, errors.New("url not found")
	}
	return res.size, res.mod, res.err
}

func (f *fakeDownloader) ResumableDownloader(_ context.Context, urlStr, targetFile string, status *downloader.DownloadStatus, _ chan *downloader.ProgressInfo) error {
	res, ok := f.head[urlStr]
	if !ok {
		return errors.New("url not found")
	}
	if status.TotalSize == 0 {
		status.TotalSize = res.size
	}
	status.ModTime = res.mod
	return os.WriteFile(targetFile, []byte("ok"), 0644)
}

func (f *fakeDownloader) DownloadStatusVerify(status *downloader.DownloadStatus, remoteSize int64, remoteModTime time.Time) bool {
	return status.TotalSize == remoteSize && status.ModTime.Equal(remoteModTime)
}

func (f *fakeDownloader) DefaultClient() *http.Client { return &http.Client{} }

func (f *fakeDownloader) RemoteURLCacheEnabled() bool { return false }

func (f *fakeDownloader) PutRemoteURL(ctx context.Context, key, urlStr string) error {
	return downloader.ErrRemoteURLCacheDisabled
}

func (f *fakeDownloader) GetRemoteURLs(ctx context.Context, key string) ([]string, error) {
	return nil, downloader.ErrRemoteURLCacheDisabled
}

func TestSelectFirstReachable(t *testing.T) {
	now := time.Now().UTC()
	fake := &fakeDownloader{
		head: map[string]headResult{
			"https://example.com/bad": {err: errors.New("boom")},
			"https://example.com/good": {
				size: 10,
				mod:  now,
			},
		},
	}
	url, size, modTime, err := vmdownloader.SelectFirstReachable(fake, []string{
		"https://example.com/bad",
		"https://example.com/good",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/good" {
		t.Fatalf("unexpected url: %s", url)
	}
	if size != 10 {
		t.Fatalf("unexpected size: %d", size)
	}
	if !modTime.Equal(now) {
		t.Fatalf("unexpected mod time: %v", modTime)
	}
}

func TestIsStatusValid(t *testing.T) {
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.ops")
	now := time.Now().UTC()
	want := &downloader.DownloadStatus{
		Url:        "https://example.com/file.iso",
		TargetFile: filepath.Join(tmp, "file.iso.syn"),
		TotalSize:  100,
		Curr:       50,
		ModTime:    now,
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if err := os.WriteFile(statusPath, data, 0644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	fake := &fakeDownloader{
		head: map[string]headResult{
			want.Url: {size: want.TotalSize, mod: now},
		},
	}
	status, err := vmdownloader.IsStatusValid(fake, statusPath)
	if err != nil {
		t.Fatalf("expected valid status, got error: %v", err)
	}
	if status.Url != want.Url {
		t.Fatalf("status mismatch: %s", status.Url)
	}
}
