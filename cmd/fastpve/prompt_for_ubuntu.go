package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/utils"
	"github.com/solider245/fastpve/vmdownloader"
)

const (
	Ubuntu2204ForDesk = iota
	Ubuntu2204ForServer
	Ubuntu2410ForDesk
	Ubuntu2410ForServer
	Ubuntu2504ForDesk
	Ubuntu2504ForServer
)

type ubuntuInstallInfo struct {
	UbuntuISO    string `json:"ubuntuISO"`
	UbuntuVer    int    `json:"ubuntuVer"`
	Memory       int    `json:"memory"`
	Cores        int    `json:"cores"`
	Disk         int    `json:"disk"`
	DownloadOnly bool   `json:"downloadOnly"`
}

func (i *ubuntuInstallInfo) getDisplayName() string { return filepath.Base(i.UbuntuISO) }
func (i *ubuntuInstallInfo) setDownloadOnly()       { i.DownloadOnly = true }
func (i *ubuntuInstallInfo) getCores() int          { return i.Cores }
func (i *ubuntuInstallInfo) getMemory() int         { return i.Memory }
func (i *ubuntuInstallInfo) getDisk() int           { return i.Disk }

func promptForUbuntu() error {
	isoPath := defaultISOPath
	cachePath := defaultCachePath
	downer := newDownloader()
	statusPath := filepath.Join(cachePath, "ubuntu_install.ops")
	status, _ := vmdownloader.IsStatusValid(downer, statusPath)

	var ubuntuISOs []string
	dirs, err := os.ReadDir(isoPath)
	if err == nil {
		ubuntuISOs = getUbuntuISO(dirs)
	}

	info := &ubuntuInstallInfo{
		UbuntuVer: -1,
	}

	err = promptUbuntuFiles(info, status, ubuntuISOs)
	if err != nil {
		return err
	}
	info.Cores, err = promptPVECore()
	if err != nil {
		return err
	}
	info.Memory, err = promptPVEMemory()
	if err != nil {
		return err
	}
	info.Disk, err = promptPVEDisk()
	if err != nil {
		return err
	}

	var needDownload bool
	if (status != nil && info.UbuntuISO == status.TargetFile) ||
		info.UbuntuVer >= 0 {
		needDownload = true
	}
	next, err := promptDownloadInstall(info, needDownload)
	if err != nil {
		return err
	}
	if !next {
		return nil
	}

	ctx := context.TODO()
	if status != nil && info.UbuntuISO == status.TargetFile {
		// Continue download target file
		info.UbuntuISO, err = vmdownloader.DownloadUbuntuISO(ctx, downer, isoPath, cachePath, statusPath, status, -1)
		if err != nil {
			return err
		}
	}
	if info.UbuntuVer >= 0 {
		status = nil
		info.UbuntuISO, err = vmdownloader.DownloadUbuntuISO(ctx, downer, isoPath, cachePath, statusPath, status, info.UbuntuVer)
		if err != nil {
			return err
		}
	}
	if info.DownloadOnly {
		return nil
	}

	return createUbuntuVM(ctx, isoPath, info)
}

func promptUbuntuFiles(info *ubuntuInstallInfo, status *downloader.DownloadStatus, ubuntuISOs []string) error {
	origUbuntuLen := len(ubuntuISOs)
	if status != nil {
		name := filepath.Base(status.TargetFile)
		name = strings.TrimSuffix(name, ".syn")
		progress := status.Curr * 100 / (status.TotalSize + 1)
		name = fmt.Sprintf("继续下载 %s(%02d%%)", name, progress)
		ubuntuISOs = append(ubuntuISOs, name)
	}
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 22.04-desktop")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 22.04-live-server")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 24.10-desktop")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 24.10-live-server")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 25.04-desktop")
	ubuntuISOs = append(ubuntuISOs, "全新下载 Ubuntu 25.04-live-server")
	prompt := promptui.Select{
		Label: "选择Ubuntu安装文件",
		Items: ubuntuISOs,
	}
	idx, file, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx < origUbuntuLen {
		info.UbuntuISO = file
	} else {
		if status != nil && idx == (len(ubuntuISOs)-7) {
			info.UbuntuISO = status.TargetFile
		} else if idx >= (len(ubuntuISOs) - 6) {
			info.UbuntuVer = idx - (len(ubuntuISOs) - 6)
		}
	}

	return nil
}

func getUbuntuISO(dirs []os.DirEntry) []string {
	var isoFiles []string
	for _, dir := range dirs {
		if !dir.IsDir() &&
			strings.HasPrefix(dir.Name(), "ubuntu-") &&
			filepath.Ext(dir.Name()) == ".iso" {
			isoFiles = append(isoFiles, dir.Name())
		}
	}
	return isoFiles
}

func createUbuntuVM(ctx context.Context, isoPath string, info *ubuntuInstallInfo) error {
	useDisk, vmid, err := resolveStorageAndVMID()
	if err != nil {
		return err
	}
	imgName := filepath.Base(info.UbuntuISO)
	vmName := toBetterUbuntuName(imgName)
	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine q35 --bios ovmf --cpu host --net0 virtio,bridge=vmbr0`,
			vmName, info.Memory, info.Cores),
		fmt.Sprintf("qm set $VMID -efidisk0 %s:1,format=raw,efitype=4m", useDisk),
		fmt.Sprintf("qm set $VMID --scsi0 %s:%d", useDisk, info.Disk),
		fmt.Sprintf(`qm set $VMID --ide0 local:iso/%s,media=cdrom`, imgName),
		`qm set $VMID --boot order='scsi0;ide0'`,
		`qm set $VMID --agent enabled=1,fstrim_cloned_disks=1`,
		`qm set $VMID --ostype l26`,
		`echo "VMOK"`,
	}
	out, err := utils.BatchOutput(ctx, scripts, 0)
	if err != nil {
		return err
	}
	if strings.Contains(string(out), "VMOK") {
		fmt.Println("创建虚拟机：", vmid, "成功，正在启动...")
		utils.BatchRun(ctx, []string{fmt.Sprintf("qm start %d", vmid)}, 10)
		fmt.Printf("VM %d 已启动，请在控制台继续安装系统\n", vmid)
		return nil
	}
	return errors.New("VM creation failed")
}
