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

	"github.com/solider245/fastpve/quickget"
	"github.com/solider245/fastpve/utils"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/manifoldco/promptui"
)

type ddPresetInstallInfo struct {
	Preset       vmdownloader.DDPreset
	DDImgName    string
	DownloadURL  string
	Memory       int
	Cores        int
	Disk         int
	DownloadOnly bool
}

func makeDDPresetAction(p vmdownloader.DDPreset) func() error {
	return func() error { return installFromDDPreset(p) }
}

func installFromDDPreset(preset vmdownloader.DDPreset) error {
	info := &ddPresetInstallInfo{
		Preset: preset,
		Memory: preset.RecMemoryMB,
		Cores:  preset.RecCores,
		Disk:   preset.MinDiskGB,
	}

	fmt.Println()
	fmt.Printf("  系统: %s\n", preset.Name)
	fmt.Printf("  说明: %s\n", preset.Description)
	if preset.BIOS == vmdownloader.BIOSUEFI {
		fmt.Println("  BIOS: UEFI (OVMF)")
	} else {
		fmt.Println("  BIOS: SeaBIOS (传统)")
	}
	fmt.Printf("  推荐配置: %d 核 / %d MB / %d GB 磁盘\n", preset.RecCores, preset.RecMemoryMB, preset.MinDiskGB)
	fmt.Println()

	downloadURL := ""
	if len(preset.URLs) > 0 {
		downloadURL = preset.URLs[0]
		fmt.Printf("  默认镜像: %s\n", downloadURL)
		changePrompt := promptui.Prompt{
			Label:     "是否修改镜像URL（回车确认默认）",
			Default:   downloadURL,
			AllowEdit: true,
		}
		result, err := changePrompt.Run()
		if err != nil {
			return errContinue
		}
		downloadURL = strings.TrimSpace(result)
	} else {
		urlPrompt := promptui.Prompt{
			Label: "此系统暂无内置URL，请输入DD镜像下载地址",
		}
		url, err := urlPrompt.Run()
		if err != nil {
			return errContinue
		}
		downloadURL = strings.TrimSpace(url)
	}
	if downloadURL == "" {
		return errors.New("镜像URL不能为空")
	}
	info.DownloadURL = downloadURL

	// Cache fast-path: if the final image already exists, skip to confirmation
	isoPath := "/var/lib/vz/template/iso/"
	if finalName := vmdownloader.FinalImageName(downloadURL); finalName != "" {
		if _, err := os.Stat(filepath.Join(isoPath, finalName)); err == nil {
			fmt.Println("  镜像已缓存:", finalName)
			fastItems := []string{"确认安装（推荐配置）", "修改配置", "退出"}
			fastPrompt := promptui.Select{
				Label: "快速安装",
				Items: fastItems,
			}
			idx, _, err := fastPrompt.Run()
			if err != nil || idx == 2 {
				return errContinue
			}
			if idx == 0 {
				info.DDImgName = finalName
				err = createDDPresetVM(context.TODO(), isoPath, info)
				if err != nil {
					return err
				}
				return errContinue
			}
			// idx == 1: fall through to normal prompt
		}
	}

	var err error
	fmt.Println()
	info.Cores, err = promptPVECoreWithDefault(preset.RecCores)
	if err != nil {
		return err
	}
	info.Memory, err = promptPVEMemoryWithDefault(preset.RecMemoryMB)
	if err != nil {
		return err
	}
	info.Disk, err = promptPVEDiskWithDefault(preset.MinDiskGB)
	if err != nil {
		return err
	}

	// debug: fmt.Println("install=", utils.ToString(info))

	actionItems := []string{"下载并安装", "仅下载", "退出"}
	actionPrompt := promptui.Select{
		Label: fmt.Sprintf("确认安装 %s（CPU:%d 内存:%dMB 磁盘:%dGB）",
			preset.Name, info.Cores, info.Memory, info.Disk),
		Items: actionItems,
	}
	actionIdx, _, err := actionPrompt.Run()
	if err != nil {
		return errContinue
	}
	if actionIdx == 2 {
		return nil
	}
	if actionIdx == 1 {
		info.DownloadOnly = true
	}

	err = downloadAndCreateDDPresetVM(info)
	if err != nil {
		return err
	}
	return errContinue
}

func downloadAndCreateDDPresetVM(info *ddPresetInstallInfo) error {
	isoPath := "/var/lib/vz/template/iso/"
	cachePath := "/var/lib/vz/template/cache"
	downer := newDownloader()
	statusPath := filepath.Join(cachePath, "dd_preset_install.ops")
	ctx := context.TODO()

	fmt.Println("开始下载", info.Preset.Name, "镜像...")
	imgName, err := vmdownloader.DownloadDDImage(ctx, downer, isoPath, cachePath, statusPath, nil, info.DownloadURL)
	if err != nil {
		return err
	}
	info.DDImgName = imgName

	if info.DownloadOnly {
		fmt.Println("download complete:", imgName)
		return nil
	}

	return createDDPresetVM(ctx, isoPath, info)
}

func createDDPresetVM(ctx context.Context, isoPath string, info *ddPresetInstallInfo) error {
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

	preset := info.Preset

	biosFlag := "ovmf"
	if preset.BIOS == vmdownloader.BIOSSeaBIOS {
		biosFlag = "seabios"
	}

	machine := preset.Machine
	if machine == "" {
		machine = "q35"
	}
	ostype := preset.OSType
	if ostype == "" {
		ostype = "l26"
	}

	baseName := strings.ReplaceAll(preset.Name, " ", "")
	counter := 1

	bridge := "vmbr0"
	bridgePrompt := promptui.Prompt{
		Label:   "网络桥接 (默认 vmbr0)",
		Default: bridge,
	}
	if b, err := bridgePrompt.Run(); err == nil && b != "" {
		bridge = strings.TrimSpace(b)
	}

	for {
		vmName := baseName
		if counter > 1 {
			vmName = fmt.Sprintf("%s-%d", baseName, counter)
		}

		scripts := []string{
			"set -e",
			`export LC_ALL="en_US.UTF-8"`,
			fmt.Sprintf("export VMID=%d", vmid),
			fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine %s --bios %s --cpu host --net0 virtio,bridge=%s --agent enabled=1`,
				vmName, info.Memory, info.Cores, machine, biosFlag, bridge),
		}

		if preset.BIOS == vmdownloader.BIOSUEFI {
			scripts = append(scripts,
				fmt.Sprintf("qm set $VMID -efidisk0 %s:1,format=raw,efitype=4m", useDisk),
			)
		}

		scripts = append(scripts,
			fmt.Sprintf("qm set $VMID --scsi0 %s:0,import-from=%s", useDisk, filepath.Join(isoPath, info.DDImgName)),
			fmt.Sprintf(`qm set $VMID  --scsi1 %s:%d`, useDisk, info.Disk),
			`qm set $VMID --boot order='scsi0'`,
			fmt.Sprintf(`qm set $VMID  --ostype %s`, ostype),
			`echo "VMOK"`,
		)

		out, err := utils.BatchOutput(ctx, scripts, 0)
		if err != nil {
			// Clean up partial VM on failure
			cleanupErr := utils.BatchRun(ctx, []string{
				fmt.Sprintf("qm destroy %d --purge 2>/dev/null; true", vmid),
			}, 10)
			if cleanupErr != nil {
				fmt.Println("清理残留失败:", cleanupErr)
			}
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
		counter++
		vmid++

		anotherPrompt := promptui.Prompt{
			Label:   "再来一台相同配置？(Y/n)",
			Default: "Y",
		}
		result, err := anotherPrompt.Run()
		if err != nil || strings.ToUpper(strings.TrimSpace(result)) != "Y" {
			break
		}
		fmt.Println()
	}
	return nil
}

func promptPVECoreWithDefault(def int) (int, error) {
	validate := func(input string) error { return nil }
	prompt := promptui.Prompt{
		Label:    fmt.Sprintf("CPU核数 (默认 %d)", def),
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
	var cores int
	fmt.Sscanf(result, "%d", &cores)
	if cores <= 0 {
		cores = def
	}
	return cores, nil
}

func promptPVEMemoryWithDefault(def int) (int, error) {
	validate := func(input string) error { return nil }
	prompt := promptui.Prompt{
		Label:    fmt.Sprintf("内存大小/MB (默认 %d)", def),
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
	var mem int
	fmt.Sscanf(result, "%d", &mem)
	if mem <= 0 {
		mem = def
	}
	return mem, nil
}

func promptPVEDiskWithDefault(def int) (int, error) {
	validate := func(input string) error { return nil }
	prompt := promptui.Prompt{
		Label:    fmt.Sprintf("磁盘大小/GB (默认 %d)", def),
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
	var disk int
	fmt.Sscanf(result, "%d", &disk)
	if disk <= 0 {
		disk = def
	}
	return disk, nil
}
