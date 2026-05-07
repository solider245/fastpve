package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/linkease/fastpve/quickget"
	"github.com/linkease/fastpve/utils"
	"github.com/linkease/fastpve/vmdownloader"
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

func promptForDDPresets() error {
	categories := vmdownloader.AllDDPresetCategories()

	var catNames []string
	for _, cat := range categories {
		label := fmt.Sprintf("%s (%d个系统)", cat.Name, len(cat.Presets))
		catNames = append(catNames, label)
	}
	catNames = append(catNames, "自定义URL（手动输入DD镜像地址）")

	prompt := promptui.Select{
		Label: "选择DD镜像分类",
		Items: catNames,
	}
	catIdx, _, err := prompt.Run()
	if err != nil {
		return errContinue
	}

	if catIdx >= len(categories) {
		return promptForDD()
	}

	category := categories[catIdx]
	var sysNames []string
	for _, p := range category.Presets {
		sysNames = append(sysNames, fmt.Sprintf("%-25s %s", p.Name, p.Description))
	}
	sysNames = append(sysNames, "返回上级菜单")

	sysPrompt := promptui.Select{
		Label: fmt.Sprintf("%s — 选择系统", category.Name),
		Items: sysNames,
	}
	sysIdx, _, err := sysPrompt.Run()
	if err != nil {
		return errContinue
	}
	if sysIdx >= len(category.Presets) {
		return promptForDDPresets()
	}

	preset := category.Presets[sysIdx]
	return installFromDDPreset(preset)
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

	fmt.Println("install=", utils.ToString(info))

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

	return downloadAndCreateDDPresetVM(info)
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
	vmName := strings.TrimSuffix(info.DDImgName, filepath.Ext(info.DDImgName))
	vmName = strings.TrimSuffix(vmName, filepath.Ext(vmName))

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

	scripts := []string{
		"set -e",
		`export LC_ALL="en_US.UTF-8"`,
		fmt.Sprintf("export VMID=%d", vmid),
		fmt.Sprintf(`qm create $VMID --name "%s" --memory %d --scsihw virtio-scsi-single --cores %d --sockets 1 --machine %s --bios %s --cpu host --net0 virtio,bridge=vmbr0`,
			vmName, info.Memory, info.Cores, machine, biosFlag),
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
		return err
	}
	if !strings.Contains(string(out), "VMOK") {
		return errors.New("VM creation failed")
	}
	fmt.Println("创建虚拟机：", vmid, "成功")
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
