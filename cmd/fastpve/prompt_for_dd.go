package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/quickget"
	"github.com/solider245/fastpve/utils"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/manifoldco/promptui"
)

const (
	biosUEFI = iota
	biosSeaBIOS
)

type ddInstallInfo struct {
	DDImgURL     string `json:"ddImgURL"`
	DDImgName    string `json:"ddImgName"`
	BIOSMode     int    `json:"biosMode"`
	Memory       int    `json:"memory"`
	Cores        int    `json:"cores"`
	Disk         int    `json:"disk"`
	DownloadOnly bool   `json:"downloadOnly"`
}

func promptForDD() error {
	isoPath := "/var/lib/vz/template/iso/"
	cachePath := "/var/lib/vz/template/cache"
	downer := newDownloader()
	statusPath := filepath.Join(cachePath, "dd_install.ops")
	status, _ := vmdownloader.IsStatusValid(downer, statusPath)

	var existingImgs []string
	dirs, err := os.ReadDir(isoPath)
	if err == nil {
		existingImgs = getDDImgFiles(dirs)
	}

	info := &ddInstallInfo{
		BIOSMode: -1,
	}

	err = promptDDFiles(info, status, existingImgs)
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

	// BIOS mode selection
	info.BIOSMode, err = promptDDBios()
	if err != nil {
		return err
	}

		// debug: fmt.Println("install=", utils.ToString(info))
	var needDownload bool
	if (status != nil && info.DDImgURL == status.Url) ||
		info.DDImgURL != "" && info.DDImgName == "" {
		needDownload = true
	}
	next, err := promptDDDownloadInstall(info, needDownload)
	if err != nil {
		return err
	}
	if !next {
		return nil
	}

	ctx := context.TODO()
	if status != nil && info.DDImgURL == status.Url {
		info.DDImgName, err = vmdownloader.DownloadDDImage(ctx, downer, isoPath, cachePath, statusPath, status, "")
		if err != nil {
			return err
		}
	}
	if info.DDImgURL != "" && info.DDImgName == "" {
		status = nil
		info.DDImgName, err = vmdownloader.DownloadDDImage(ctx, downer, isoPath, cachePath, statusPath, status, info.DDImgURL)
		if err != nil {
			return err
		}
	}
	if info.DownloadOnly {
		fmt.Println("download complete:", info.DDImgName)
		return nil
	}

	return createDDVM(ctx, isoPath, info)
}

func getDDImgFiles(dirs []os.DirEntry) []string {
	var imgFiles []string
	for _, dir := range dirs {
		if !dir.IsDir() && strings.HasSuffix(strings.ToLower(dir.Name()), ".img") {
			imgFiles = append(imgFiles, dir.Name())
		}
	}
	return imgFiles
}

func promptDDFiles(info *ddInstallInfo, status *downloader.DownloadStatus, existingImgs []string) error {
	origLen := len(existingImgs)
	if status != nil {
		name := filepath.Base(status.TargetFile)
		name = strings.TrimSuffix(name, ".syn")
		progress := status.Curr * 100 / (status.TotalSize + 1)
		name = fmt.Sprintf("继续下载 %s(%02d%%)", name, progress)
		existingImgs = append(existingImgs, name)
	}
	existingImgs = append(existingImgs, "输入URL下载新镜像")

	prompt := promptui.Select{
		Label: "选择DD镜像文件",
		Items: existingImgs,
	}
	idx, file, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx < origLen {
		info.DDImgName = file
	} else if status != nil && idx == origLen {
		info.DDImgURL = status.Url
	} else {
		// User chose "输入URL下载新镜像"
		urlPrompt := promptui.Prompt{
			Label: "输入DD镜像下载URL",
		}
		url, err := urlPrompt.Run()
		if err != nil {
			return err
		}
		info.DDImgURL = strings.TrimSpace(url)
	}
	return nil
}

func promptDDBios() (int, error) {
	prompt := promptui.Select{
		Label: "选择BIOS模式",
		Items: []string{"UEFI (OVMF) — 推荐，支持GPT", "SeaBIOS — 传统BIOS，支持MBR"},
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return 0, err
	}
	return idx, nil
}

func promptDDDownloadInstall(info *ddInstallInfo, needDownload bool) (bool, error) {
	var items []string
	if needDownload {
		items = []string{"下载并安装", "仅下载", "退出"}
	} else {
		items = []string{"安装", "退出"}
	}
	displayName := info.DDImgName
	if displayName == "" {
		displayName = info.DDImgURL
	}
	prompt := promptui.Select{
		Label: fmt.Sprintf("选择完成，继续安装%s：（CPU：%d,内存：%dMB,硬盘：%dGB）",
			filepath.Base(displayName),
			info.Cores,
			info.Memory,
			info.Disk),
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return false, err
	}
	if idx == 0 {
		return true, nil
	}
	if needDownload {
		if idx == 1 {
			info.DownloadOnly = true
			return true, nil
		}
	}
	return false, nil
}

func createDDVM(ctx context.Context, isoPath string, info *ddInstallInfo) error {
	disks, err := quickget.DiskStatus()
	if err != nil {
		return err
	}
	useDisk := "local"
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
		return err
	}
	vmid := 100
	if len(items) > 0 {
		sort.Slice(items, func(i, j int) bool {
			return items[i].VMID < items[j].VMID
		})
		vmid = items[len(items)-1].VMID + 1
	}
	imgName := info.DDImgName
	vmName := strings.TrimSuffix(imgName, filepath.Ext(imgName))

	bridge := "vmbr0"
	bridgePrompt := promptui.Prompt{
		Label:   "网络桥接 (默认 vmbr0)",
		Default: bridge,
	}
	if b, err := bridgePrompt.Run(); err == nil && b != "" {
		bridge = strings.TrimSpace(b)
	}

	biosFlag := "ovmf"
	ostype := "l26"
	machine := "q35"
	if info.BIOSMode == biosSeaBIOS {
		biosFlag = "seabios"
		ostype = "other"
		machine = "i440fx"
	}

	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine %s --bios %s --cpu host --net0 virtio,bridge=%s --agent enabled=1`,
			vmName,
			info.Memory,
			info.Cores,
			machine,
			biosFlag,
			bridge),
	}

	if info.BIOSMode == biosUEFI {
		scripts = append(scripts,
			fmt.Sprintf("qm set $VMID -efidisk0 %s:1,format=raw,efitype=4m", useDisk),
		)
	}

	scripts = append(scripts,
		fmt.Sprintf("qm set $VMID --scsi0 %s:0,import-from=%s", useDisk, filepath.Join(isoPath, imgName)),
		fmt.Sprintf(`qm set $VMID  --scsi1 %s:%d`, useDisk, info.Disk),
		`qm set $VMID --boot order='scsi0'`,
		fmt.Sprintf(`qm set $VMID  --ostype %s`, ostype),
		`echo "VMOK"`,
	)

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
	fmt.Println("创建虚拟机：", vmid, "成功")
	// Auto-start and wait for IP
	utils.BatchRun(ctx, []string{fmt.Sprintf("qm start %d", vmid)}, 10)
	fmt.Printf("等待 VM %d 获取IP...\n", vmid)
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		ip := getVMIP(vmid)
		if ip != "-" {
			fmt.Printf("VM %d IP: %s\n", vmid, ip)
			break
		}
	}
	return nil
}
