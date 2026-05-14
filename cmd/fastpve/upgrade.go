package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/solider245/fastpve/utils"
)

func selfUpgrade() error {
	fmt.Println("正在检查 FastPVE 更新...")

	// Detect current binary path
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取当前路径: %w", err)
	}

	// Determine arch
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}

	// Try multiple mirrors for download
	urls := []string{
		fmt.Sprintf("https://github.com/solider245/fastpve/releases/latest/download/FastPVE"),
		fmt.Sprintf("https://gh.565600.xyz/https://github.com/solider245/fastpve/releases/latest/download/FastPVE"),
	}

	tmpFile := "/tmp/fastpve-upgrade"
	var lastErr error
	for _, u := range urls {
		fmt.Printf("  下载 %s ...\n", u)
		err := utils.BatchRun(context.TODO(), []string{
			fmt.Sprintf("curl -fSL --progress-bar -o %s %s 2>&1", tmpFile, u),
		}, 120)
		if err != nil {
			lastErr = err
			continue
		}
		// Verify it's an ELF binary
		out, _ := utils.BatchOutput(context.TODO(), []string{
			fmt.Sprintf("file %s | grep -q 'ELF' && echo ok", tmpFile),
		}, 5)
		if strings.TrimSpace(string(out)) != "ok" {
			lastErr = fmt.Errorf("下载文件不是有效的二进制")
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("下载失败: %w", lastErr)
	}

	// Replace binary
	if err := os.Rename(tmpFile, selfPath); err != nil {
		// Rename might fail if /tmp and /usr are different filesystems
		// Try cp + rm instead
		err := utils.BatchRun(context.TODO(), []string{
			fmt.Sprintf("cat %s > %s && chmod +x %s && rm -f %s", tmpFile, selfPath, selfPath, tmpFile),
		}, 10)
		if err != nil {
			return fmt.Errorf("更新失败: %w", err)
		}
	}

	fmt.Println("✅ 更新完成！")
	fmt.Println()
	return nil
}
