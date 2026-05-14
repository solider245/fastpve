package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/solider245/fastpve/utils"
)

var ErrNoRedirectFound = errors.New("no redirect found")
var ErrLoopDetected = errors.New("loop detected")
var ErrTooManyRedirects = errors.New("loop max reached")

type DownloadStatus struct {
	Url        string    `json:"url"`
	TargetFile string    `json:"target_file"`
	TotalSize  int64     `json:"total_size"`
	Curr       int64     `json:"curr"`
	ModTime    time.Time `json:"mod_time"`
}

func ReadUpdateDownload(statusPath string) (*DownloadStatus, error) {
	var status DownloadStatus
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(data, &status)
	return &status, nil
}

func UpdateDownloadStatus(status *DownloadStatus, statusPath string) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return os.WriteFile(statusPath, data, 0644)
}

type ProgressInfo struct {
	Status   *DownloadStatus
	Progress int64
	Speed    int64
}

func (d *Downloader) ResumableDownloader(ctx context.Context,
	urlStr, filePath string,
	status *DownloadStatus,
	progressCh chan *ProgressInfo) error {
	var req *http.Request

	// Open file for writing
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	isValid := d.statusIsValid(urlStr, file, status)

	if isValid {
		// Resume download from current position
		req, err = http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-", status.Curr))
	} else {
		status.Curr = 0
		req, err = http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			return err
		}
	}
	file.Seek(status.Curr, io.SeekStart)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}
	var once sync.Once
	defer once.Do(func() {
		resp.Body.Close()
	})

	const waitTimeout = time.Second * 60
	const sizeMin = 8 * 1024 * 1024
	const sizeMax = 256 * 1024 * 1024
	buf := make([]byte, 2*1024*1024)
	var n int64
	var defSize int64 = sizeMin

	doneCh := make(chan struct{})
	loopCh := make(chan struct{}, 1)
	timeoutTick := time.NewTimer(waitTimeout)
	go func() {
	LOOP_SEL:
		for {
			select {
			case <-timeoutTick.C:
				fmt.Println("Download timeout")
				once.Do(func() {
					resp.Body.Close()
				})
				break LOOP_SEL
			case <-loopCh:
				utils.ResetTimer(timeoutTick, waitTimeout)
			case <-doneCh:
				break LOOP_SEL
			}
		}
	}()

	for {
		now := time.Now()
		n, err = io.CopyBuffer(file, io.LimitReader(resp.Body, defSize), buf)
		status.Curr += n
		select {
		case loopCh <- struct{}{}:
		default:
		}
		since := time.Since(now)
		if since < time.Second*5 {
			defSize *= 2
			if defSize > sizeMax {
				defSize = sizeMax
			}
		} else {
			defSize = defSize / 2
			if defSize < sizeMin {
				defSize = sizeMin
			}
		}
		speed := 1000 * n / (since.Milliseconds() + 1)
		var progress int64
		if status.TotalSize > 0 {
			progress = status.Curr * 100 / status.TotalSize
		}

		select {
		case progressCh <- &ProgressInfo{
			Status:   status,
			Speed:    speed,
			Progress: progress,
		}:
		default:
			// Not block here
		}

		//UpdateDownloadStatus(status, statusPath)
		if err != nil || n == 0 {
			break
		}
	}

	if err == io.EOF {
		err = nil
	}
	return err
}

func (d *Downloader) HeadInfo(urlStr string) (int64, time.Time, error) {
	founds := []string{urlStr}
	var total int64
	var remoteModTime time.Time
	var err error
	var loopCount int
	for loopCount < 5 {
		loopCount++
		total, remoteModTime, err = d.headInfo(urlStr)
		if err != nil {
			return 0, time.Time{}, err
		}
		if total > 4096 {
			return total, remoteModTime, nil
		}
		nextUrl, err2 := d.realLocation(urlStr)
		if err2 != nil {
			log.Println("realLocation error: ", err2)
			return total, remoteModTime, nil
		}
		for _, s := range founds {
			if s == nextUrl {
				return 0, time.Time{}, ErrLoopDetected
			}
		}
		founds = append(founds, nextUrl)
		urlStr = nextUrl
	}
	return 0, time.Time{}, ErrTooManyRedirects
}

func (d *Downloader) headInfo(urlStr string) (int64, time.Time, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, urlStr, nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, time.Time{}, err
	}
	resp.Body.Close()
	remoteModTime, _ := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	return resp.ContentLength, remoteModTime, nil
}

func (d *Downloader) realLocation(urlStr string) (string, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", err
	}
	resp, err := d.noRedirectClient.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	//log.Println("statusCode=", resp.StatusCode, "loc=", resp.Header.Get("Location"), "url=", urlStr)
	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		return resp.Header.Get("Location"), nil
	}
	return "", ErrNoRedirectFound
}

func (d *Downloader) DownloadStatusVerify(status *DownloadStatus, remoteSize int64, remoteModTime time.Time) bool {
	if status.Curr < status.TotalSize {
		//remoteSize, remoteModTime, err := d.HeadInfo(status.Url)
		//if err != nil {
		//	return false
		//}
		if status.TotalSize != remoteSize ||
			!status.ModTime.Equal(remoteModTime) {
			return false
		}
	}
	if status.Curr > 0 {
		fi, err := os.Stat(status.TargetFile)
		if err != nil {
			return false
		}
		if fi.Size() < status.Curr {
			return false
		}
	}
	return true
}

func (d *Downloader) statusIsValid(urlStr string, file *os.File, status *DownloadStatus) bool {
	remoteSize, remoteModTime, err := d.HeadInfo(urlStr)
	if err != nil {
		fmt.Println("Range request failed, err=", err)
		return false
	}

	if status.TotalSize != remoteSize ||
		!status.ModTime.Equal(remoteModTime) {
		status.ModTime = remoteModTime
		status.TotalSize = remoteSize
		return false
	}

	if file != nil {
		p0, _ := file.Seek(0, io.SeekStart)
		p1, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			return false
		}
		if p1-p0 < status.Curr {
			return false
		}
	}
	return true
}
