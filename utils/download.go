package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type progressWriter struct {
	counter *int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	if p.counter != nil {
		atomic.AddInt64(p.counter, int64(n))
	}
	return n, nil
}

func reportDownloadProgress(name string, total int64, written *int64, stopCh <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lastBytes := int64(0)
	lastTime := time.Now()
	for {
		select {
		case <-stopCh:
			curr := atomic.LoadInt64(written)
			if curr > 0 && total > 0 {
				fmt.Printf("下载完成 %s: %s/%s\n", name, ByteCountDecimal(uint64(curr)), ByteCountDecimal(uint64(total)))
			}
			return
		case <-ticker.C:
			curr := atomic.LoadInt64(written)
			delta := curr - lastBytes
			elapsed := time.Since(lastTime).Seconds()
			speed := float64(delta) / (elapsed + 1e-6)
			if total > 0 {
				pct := curr * 100 / (total + 1)
				fmt.Printf("下载 %s: %02d%%, %s/s\n", name, pct, ByteCountDecimal(uint64(speed)))
			} else {
				fmt.Printf("下载 %s: %s, %s/s\n", name, ByteCountDecimal(uint64(curr)), ByteCountDecimal(uint64(speed)))
			}
			lastBytes = curr
			lastTime = time.Now()
		}
	}
}

// DownloadFile downloads a URL to a local file with progress reporting.
// Uses a .tmp suffix during download and renames on success.
func DownloadFile(url, dest string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "FastPVE/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: %s", resp.Status)
	}

	total := resp.ContentLength

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	tmp := dest + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	var written int64
	stopCh := make(chan struct{})
	go reportDownloadProgress(filepath.Base(dest), total, &written, stopCh)

	_, err = io.Copy(io.MultiWriter(out, &progressWriter{counter: &written}), resp.Body)
	close(stopCh)

	if err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, dest); err != nil {
		return err
	}

	return nil
}
