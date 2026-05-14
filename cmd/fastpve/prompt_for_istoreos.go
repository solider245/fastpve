package main

import (
	"context"
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
	Istore2410 = iota
	Istore2203
)

type istoreInstallInfo struct {
	IstoreIMG    string `json:"istoreIMG"`
	IstoreVer    int    `json:"istoreVer"`
	Memory       int    `json:"memory"`
	Cores        int    `json:"cores"`
	Disk         int    `json:"disk"`
	DownloadOnly bool   `json:"downloadOnly"`
}

func (i *istoreInstallInfo) getDisplayName() string { return filepath.Base(i.IstoreIMG) }
func (i *istoreInstallInfo) setDownloadOnly()       { i.DownloadOnly = true }
func (i *istoreInstallInfo) getCores() int          { return i.Cores }
func (i *istoreInstallInfo) getMemory() int         { return i.Memory }
func (i *istoreInstallInfo) getDisk() int           { return i.Disk }

func promptForIstore() error {
	isoPath := defaultISOPath
	cachePath := defaultCachePath
	downer := newDownloader()
	statusPath := filepath.Join(cachePath, "istore_install.ops")
	status, _ := vmdownloader.IsStatusValid(downer, statusPath)

	var istoreIMGs []string
	dirs, err := os.ReadDir(isoPath)
	if err == nil {
		istoreIMGs = getIstoreIMG(dirs)
	}

	info := &istoreInstallInfo{
		IstoreVer: -1,
	}

	err = promptIstoreFiles(info, status, istoreIMGs)
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
	// 如果当前有状态文件且选择了断点续传  或  选择了全新下载，则标志着需要下载
	if (status != nil && info.IstoreIMG == status.TargetFile) ||
		info.IstoreVer >= 0 {
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
	if status != nil && info.IstoreIMG == status.TargetFile {
		// Continue download target file
		info.IstoreIMG, err = vmdownloader.DownloadIstoreIMG(ctx, downer, isoPath, cachePath, statusPath, status, -1)
		if err != nil {
			return err
		}
	}
	// 全新下载走这个逻辑
	if info.IstoreVer >= 0 {
		status = nil
		info.IstoreIMG, err = vmdownloader.DownloadIstoreIMG(ctx, downer, isoPath, cachePath, statusPath, status, info.IstoreVer)
		if err != nil {
			return err
		}
	}
	if info.DownloadOnly {
		return nil
	}

	return createIstoreVM(ctx, isoPath, info)
}

/*
*
提供断点续传，根据已有iso，全新下载等方式
根据选项填充info的相应字段
*/
func promptIstoreFiles(info *istoreInstallInfo, status *downloader.DownloadStatus, istoreIMGs []string) error {
	origWinLen := len(istoreIMGs)
	if status != nil {
		name := filepath.Base(status.TargetFile)
		name = strings.TrimSuffix(name, ".syn")
		progress := status.Curr * 100 / (status.TotalSize + 1)
		name = fmt.Sprintf("继续下载 %s(%02d%%)", name, progress)
		istoreIMGs = append(istoreIMGs, name)
	}
	istoreIMGs = append(istoreIMGs, "全新下载 iStore24.10")
	istoreIMGs = append(istoreIMGs, "全新下载 iStore22.03")
	prompt := promptui.Select{
		Label: "选择iStoreOS安装文件",
		Items: istoreIMGs,
	}
	idx, file, err := prompt.Run()
	if err != nil {
		return err
	}
	// 断点续传和根据已有iso下载,则填充info的IstoreIMG字段
	// 全新下载则填充info的IstoreVer字段
	if idx < origWinLen {
		info.IstoreIMG = file
	} else {
		if status != nil && idx == (len(istoreIMGs)-3) {
			info.IstoreIMG = status.TargetFile
		} else if idx >= (len(istoreIMGs) - 2) {
			info.IstoreVer = idx - (len(istoreIMGs) - 2)
		}
	}

	return nil
}

func getIstoreIMG(dirs []os.DirEntry) []string {
	var imgFiles []string
	for _, dir := range dirs {
		if !dir.IsDir() &&
			strings.HasPrefix(dir.Name(), "istoreos-") &&
			filepath.Ext(dir.Name()) == ".img" {
			imgFiles = append(imgFiles, dir.Name())
		}
	}
	return imgFiles
}

func createIstoreVM(ctx context.Context, isoPath string, info *istoreInstallInfo) error {
	useDisk, vmid, err := resolveStorageAndVMID()
	if err != nil {
		return err
	}
	imgName := filepath.Base(info.IstoreIMG)
	vmName := toBetterIstoreName(imgName)
	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine q35 --bios ovmf --cpu host --net0 virtio,bridge=vmbr0 --agent enabled=1`,
			vmName, info.Memory, info.Cores),
		fmt.Sprintf("qm set $VMID -efidisk0 %s:1,format=raw,efitype=4m", useDisk),
		fmt.Sprintf("qm set $VMID --scsi0 %s:0,import-from=%s", useDisk, filepath.Join(isoPath, imgName)),
		fmt.Sprintf(`qm set $VMID  --scsi1 %s:%d`, useDisk, info.Disk),
		`qm set $VMID --boot order='scsi0'`,
		`qm set $VMID  --ostype l26`,
		`echo "VMOK"`,
	}
	err = runVMCreationScript(ctx, scripts, vmid)
	if err != nil {
		return err
	}
	fmt.Println("创建虚拟机：", vmid, "成功")
	utils.BatchRun(ctx, []string{fmt.Sprintf("qm start %d", vmid)}, 10)
	waitVMIP(ctx, vmid)
	return nil
}
