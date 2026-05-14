package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/solider245/fastpve/quickget"
	"github.com/solider245/fastpve/utils"
	"github.com/manifoldco/promptui"
)

const (
	defaultISOPath   = "/var/lib/vz/template/iso/"
	defaultCachePath = "/var/lib/vz/template/cache"
)

// resolveStorageAndVMID returns the preferred storage and next available VMID.
func resolveStorageAndVMID() (useDisk string, vmid int, err error) {
	disks, err := quickget.DiskStatus()
	if err != nil {
		return "", 0, err
	}
	useDisk = "local"
	if len(disks) > 0 {
		useDisk = disks[0]
	}
	for _, disk := range disks {
		if disk == "local-lvm" {
			useDisk = "local-lvm"
			break
		}
	}

	items, err := quickget.QMList()
	if err != nil {
		return "", 0, err
	}
	vmid = 100
	if len(items) > 0 {
		sort.Slice(items, func(i, j int) bool {
			return items[i].VMID < items[j].VMID
		})
		vmid = items[len(items)-1].VMID + 1
	}
	return useDisk, vmid, nil
}

// waitVMIP polls the QEMU agent for the VM's IP address.
func waitVMIP(ctx context.Context, vmid int) {
	fmt.Printf("等待 VM %d 获取IP...\n", vmid)
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		ip := getVMIP(vmid)
		if ip != "-" {
			fmt.Printf("VM %d IP: %s\n", vmid, ip)
			return
		}
	}
}

// runVMCreationScript executes a batch of qm commands and checks for the VMOK marker.
// On failure it cleans up the partial VM.
func runVMCreationScript(ctx context.Context, scripts []string, vmid int) error {
	out, err := utils.BatchOutput(ctx, scripts, 0)
	if err != nil {
		utils.BatchRun(ctx, []string{
			fmt.Sprintf("qm destroy %d --purge 2>/dev/null; true", vmid),
		}, 10)
		return err
	}
	if !strings.Contains(string(out), "VMOK") {
		utils.BatchRun(ctx, []string{
			fmt.Sprintf("qm destroy %d --purge 2>/dev/null; true", vmid),
		}, 10)
		return errors.New("VM creation failed")
	}
	return nil
}

// promptBridge asks the user to confirm or change the network bridge.
func promptBridge() string {
	bridge := "vmbr0"
	prompt := promptui.Prompt{
		Label:   "网络桥接 (默认 vmbr0)",
		Default: bridge,
	}
	if b, err := prompt.Run(); err == nil && b != "" {
		bridge = strings.TrimSpace(b)
	}
	return bridge
}

// promptDownloadInstall is a shared confirmation dialog for download-and-install flows.
type downloadInstallInfo interface {
	getDisplayName() string
	setDownloadOnly()
	getCores() int
	getMemory() int
	getDisk() int
}

func promptDownloadInstall(info downloadInstallInfo, needDownload bool) (bool, error) {
	var items []string
	if needDownload {
		items = []string{"下载并安装", "仅下载", "退出"}
	} else {
		items = []string{"安装", "退出"}
	}
	displayName := info.getDisplayName()
	prompt := promptui.Select{
		Label: fmt.Sprintf("选择完成，继续安装%s：（CPU：%d,内存：%dMB,硬盘：%dGB）",
			displayName,
			info.getCores(),
			info.getMemory(),
			info.getDisk()),
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return false, err
	}
	if idx == 0 {
		return true, nil
	}
	if needDownload && idx == 1 {
		info.setDownloadOnly()
		return true, nil
	}
	return false, nil
}

// promptIntWithDefault prompts for an integer value with a default.
func promptIntWithDefault(label string, def int) (int, error) {
	validate := func(input string) error { return nil }
	prompt := promptui.Prompt{
		Label:    fmt.Sprintf("%s (默认 %d)", label, def),
		Default:  fmt.Sprintf("%d", def),
		Validate: validate,
	}
	result, err := prompt.Run()
	if err != nil {
		return 0, err
	}
	result = strings.TrimSpace(result)
	if result == "" {
		return def, nil
	}
	var val int
	fmt.Sscanf(result, "%d", &val)
	if val <= 0 {
		val = def
	}
	return val, nil
}
