package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/solider245/fastpve/downloader"
	"github.com/solider245/fastpve/utils"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/manifoldco/promptui"
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

func (i *ddInstallInfo) getDisplayName() string {
	if i.DDImgName != "" {
		return filepath.Base(i.DDImgName)
	}
	return i.DDImgURL
}
func (i *ddInstallInfo) setDownloadOnly()           { i.DownloadOnly = true }
func (i *ddInstallInfo) getCores() int              { return i.Cores }
func (i *ddInstallInfo) getMemory() int             { return i.Memory }
func (i *ddInstallInfo) getDisk() int               { return i.Disk }

func promptForDD() error {
	isoPath := defaultISOPath
	cachePath := defaultCachePath
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
	next, err := promptDownloadInstall(info, needDownload)
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

func createDDVM(ctx context.Context, isoPath string, info *ddInstallInfo) error {
	useDisk, vmid, err := resolveStorageAndVMID()
	if err != nil {
		return err
	}
	imgName := info.DDImgName
	vmName := strings.TrimSuffix(imgName, filepath.Ext(imgName))
	bridge := promptBridge()

	biosFlag := "ovmf"
	ostype := "l26"
	machine := "q35"
	if info.BIOSMode == vmdownloader.BIOSSeaBIOS {
		biosFlag = "seabios"
		ostype = "other"
		machine = "i440fx"
	}

	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine %s --bios %s --cpu host --net0 virtio,bridge=%s --agent enabled=1`,
			vmName, info.Memory, info.Cores, machine, biosFlag, bridge),
	}
	if info.BIOSMode == vmdownloader.BIOSUEFI {
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

	err = runVMCreationScript(ctx, scripts, vmid)
	if err != nil {
		return err
	}
	fmt.Println("创建虚拟机：", vmid, "成功")
	utils.BatchRun(ctx, []string{fmt.Sprintf("qm start %d", vmid)}, 10)
	waitVMIP(ctx, vmid)
	return nil
}
