package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/utils"
)

// diagnoseGPUPassthrough 诊断 GPU 直通环境
func diagnoseGPUPassthrough() error {
	ctx := context.TODO()
	fmt.Println("")
	fmt.Println("========== GPU 直通环境检测 ==========")

	// 1. 检测 IOMMU 是否开启
	fmt.Println("[1/6] 检测 IOMMU...")
	iommuOut, _ := utils.BatchOutput(ctx, []string{
		`dmesg | grep -E 'IOMMU|DMAR' | head -5 || echo '未检测到 IOMMU 或 DMAR'`,
	}, 5)
	fmt.Print(string(iommuOut))
	iommuOK := strings.Contains(string(iommuOut), "IOMMU") || strings.Contains(string(iommuOut), "DMAR")

	// 2. 检测内核启动参数
	fmt.Println("[2/6] 检测内核启动参数...")
	cmdlineOut, _ := utils.BatchOutput(ctx, []string{
		`cat /proc/cmdline`,
	}, 3)
	cmdlineStr := string(cmdlineOut)
	fmt.Print(cmdlineStr)
	cmdlineOK := strings.Contains(cmdlineStr, "intel_iommu=on") || strings.Contains(cmdlineStr, "amd_iommu=on")

	// 3. 列出 IOMMU 分组和设备
	fmt.Println("[3/6] 检测 IOMMU 分组...")
	groupsOut, _ := utils.BatchOutput(ctx, []string{
		`find /sys/kernel/iommu_groups/ -type l 2>/dev/null | head -20 || echo '无法获取 IOMMU 分组'`,
	}, 5)
	fmt.Print(string(groupsOut))
	groupsOK := len(groupsOut) > 0 && !strings.Contains(string(groupsOut), "无法获取")

	// 4. 检测 VGA/GPU 设备
	fmt.Println("[4/6] 检测 GPU 设备...")
	gpuOut, _ := utils.BatchOutput(ctx, []string{
		`lspci -nn | grep -E 'VGA|Display|3D' || echo '未检测到 GPU 设备'`,
	}, 5)
	fmt.Print(string(gpuOut))
	gpuOK := !strings.Contains(string(gpuOut), "未检测到")

	// 5. 检测黑名单配置
	fmt.Println("[5/6] 检测黑名单配置...")
	blacklistOut, _ := utils.BatchOutput(ctx, []string{
		`cat /etc/modprobe.d/blacklist.conf 2>/dev/null || echo '无 blacklist.conf'`,
	}, 3)
	fmt.Print(string(blacklistOut))
	blacklistOK := strings.Contains(string(blacklistOut), "blacklist")

	// 6. 检测 vfio-pci 是否已绑定
	fmt.Println("[6/6] 检测 vfio-pci 绑定...")
	vfioOut, _ := utils.BatchOutput(ctx, []string{
		`dmesg | grep 'vfio' | tail -3 || echo '未检测到 vfio 相关消息'`,
	}, 5)
	fmt.Print(string(vfioOut))
	vfioOK := !strings.Contains(string(vfioOut), "未检测到")

	// 汇总显示
	fmt.Println("")
	fmt.Println("--- 检测结果 ---")

	checkMark := "✓"
	crossMark := "✗"

	if iommuOK {
		fmt.Printf("  %s IOMMU\n", checkMark)
	} else {
		fmt.Printf("  %s IOMMU\n", crossMark)
	}
	if cmdlineOK {
		fmt.Printf("  %s 内核启动参数\n", checkMark)
	} else {
		fmt.Printf("  %s 内核启动参数\n", crossMark)
	}
	if groupsOK {
		fmt.Printf("  %s IOMMU 分组\n", checkMark)
	} else {
		fmt.Printf("  %s IOMMU 分组\n", crossMark)
	}
	if gpuOK {
		fmt.Printf("  %s GPU 设备\n", checkMark)
	} else {
		fmt.Printf("  %s GPU 设备\n", crossMark)
	}
	if blacklistOK {
		fmt.Printf("  %s 黑名单配置\n", checkMark)
	} else {
		fmt.Printf("  %s 黑名单配置\n", crossMark)
	}
	if vfioOK {
		fmt.Printf("  %s vfio-pci 绑定\n", checkMark)
	} else {
		fmt.Printf("  %s vfio-pci 绑定\n", crossMark)
	}

	// 给出建议
	fmt.Println("")
	fmt.Println("--- 建议 ---")
	if !iommuOK {
		fmt.Println("  - IOMMU 未开启，请在 BIOS 中开启 VT-d (Intel) 或 AMD-Vi (AMD)")
		fmt.Println("  - 或在内核启动参数中添加 intel_iommu=on/amd_iommu=on")
	}
	if !cmdlineOK {
		fmt.Println("  - 未检测到 iommu 内核启动参数")
		fmt.Println("  - 编辑 /etc/default/grub，在 GRUB_CMDLINE_LINUX_DEFAULT 中添加 intel_iommu=on iommu=pt")
		fmt.Println("  - 然后运行: proxmox-boot-tool refresh 或 update-grub")
	}
	if !groupsOK {
		fmt.Println("  - 无法获取 IOMMU 分组信息，IOMMU 可能未正确启用")
	}
	if !gpuOK {
		fmt.Println("  - 未检测到 GPU 设备，请确认硬件已正确安装")
	}
	if !blacklistOK {
		fmt.Println("  - 未配置 GPU 驱动黑名单")
		fmt.Println("  - 创建 /etc/modprobe.d/blacklist.conf，添加:")
		fmt.Println("    blacklist i915")
		fmt.Println("    blacklist snd_hda_intel")
		fmt.Println("    blacklist snd_hda_codec_hdmi")
		fmt.Println("    options vfio_iommu_type1 allow_unsafe_interrupts=1")
	}
	if !vfioOK {
		fmt.Println("  - vfio-pci 未绑定 GPU 设备")
		fmt.Println("  - 确认 /etc/modules 包含 vfio、vfio_iommu_type1、vfio_pci")
		fmt.Println("  - 确认 /etc/modprobe.d/i915.conf 配置了正确的设备 ID")
		fmt.Println("  - 运行 update-initramfs -u 更新内核")
	}
	if iommuOK && cmdlineOK && groupsOK && gpuOK && blacklistOK && vfioOK {
		fmt.Println("  GPU 直通环境已就绪，无需额外操作")
	}

	return nil
}

/**
参照脚本:

vi /etc/default/grub
# 修改or添加
GRUB_CMDLINE_LINUX_DEFAULT="quiet intel_iommu=on iommu=pt pcie_acs_override=downstream"

# 更新grub
proxmox-boot-tool refresh

vi /etc/modules
# 添加
vfio
vfio_iommu_type1
vfio_pci


# 这里可能没有"pve-"这个前缀
vi /etc/modprobe.d/pve-blacklist.conf
# 添加
blacklist i915
blacklist snd_hda_intel
blacklist snd_hda_codec_hdmi
options vfio_iommu_type1 allow_unsafe_interrupts=1

# 根据输出确认是N100还是N150，如果都不是提示不支持该核显的直通
lscpu | grep "Model name"


/etc/modprobe.d/i915.conf
# 添加（冒号后面分别是GPU设备id和音频设备id，intel核显好像固定式8086开头）
# N150的设备ID为：46d4, N100的设备ID为: 46d1
# N150和N100的声卡设备ID为：54c8
options vfio-pci ids=8086:,8086:

# 更新内核
update-initramfs -u
**/

// promptForGPUPassThrough 实现一键核显直通功能
// 对于下面的文件修改的函数，重复执行不会重复添加参数（大概）
// /etc/modules和/etc/modprobe.d/pve-blacklist.conf可能不存在
func promptForGPUPassThrough() error {
	// 步骤 1: 修改 /etc/default/grub 文件
	err := modifyGrubConfig()
	if err != nil {
		return fmt.Errorf("修改 GRUB 配置文件失败: %w", err)
	}

	// 步骤 2: 更新 GRUB
	err = updateGRUB()
	if err != nil {
		return fmt.Errorf("更新 GRUB 失败: %w", err)
	}

	// 步骤 3: 修改 /etc/modules 文件（这个暂时不清楚有什么用）
	err = modifyModulesFile()
	if err != nil {
		return fmt.Errorf("修改 /etc/modules 文件失败: %w", err)
	}

	// 步骤 4: 修改 /etc/modprobe.d/pve-blacklist.conf 文件 （防止宿主机驱动 GPU）
	err = modifyBlacklistConfig()
	if err != nil {
		return fmt.Errorf("修改黑名单配置文件失败: %w", err)
	}

	// 步骤 5: 检测 CPU 型号（展示只支持N100和N150）
	cpuModel, err := detectCPUModel()
	if err != nil {
		return fmt.Errorf("检测 CPU 型号失败: %w", err)
	}

	// 步骤 6: 根据 CPU 型号修改 /etc/modprobe.d/i915.conf 文件
	err = modifyI915Config(cpuModel)
	if err != nil {
		return fmt.Errorf("修改 i915 配置文件失败: %w", err)
	}

	// 步骤 7: 更新内核（这一步貌似是更新前面的配置）
	err = updateInitramfs()
	if err != nil {
		return fmt.Errorf("更新内核失败: %w", err)
	}
	var reboot bool
	// 步骤 8: 提示用户重启系统
	prompt := promptui.Select{
		Label: "配置已完成，是否重启系统使配置生效？",
		Items: []string{"是", "否"},
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx == 0 {
		reboot = true
	}

	if reboot {
		return utils.BatchRun(context.TODO(), []string{"reboot"}, 0)
	}
	return nil
}

// modifyGrubConfig 修改 /etc/default/grub 文件
func modifyGrubConfig() error {
	grubFile := "/etc/default/grub"
	data, err := os.ReadFile(grubFile)
	if err != nil {
		return err
	}

	content := string(data)
	param := "intel_iommu=on iommu=pt pcie_acs_override=downstream"

	// 检查是否已包含参数
	if strings.Contains(content, param) {
		log.Println("GRUB 参数已存在，跳过修改")
		return nil
	}

	// 添加参数到 GRUB_CMDLINE_LINUX_DEFAULT
	if strings.Contains(content, "GRUB_CMDLINE_LINUX_DEFAULT=") {
		// 使用正则表达式匹配并修改配置行
		re := regexp.MustCompile(`GRUB_CMDLINE_LINUX_DEFAULT="([^"]*)"`)
		content = re.ReplaceAllString(content, fmt.Sprintf(`GRUB_CMDLINE_LINUX_DEFAULT="%s"`, param))
	} else {
		// 如果找不到配置行，添加新行
		content += fmt.Sprintf("\nGRUB_CMDLINE_LINUX_DEFAULT=\"%s\"\n", param)
	}

	return os.WriteFile(grubFile, []byte(content), 0644)

}

// updateGRUB 更新 GRUB
func updateGRUB() error {
	return utils.BatchRunStdout(context.TODO(), []string{"proxmox-boot-tool refresh"}, 0)
}

// modifyModulesFile 修改 /etc/modules 文件
func modifyModulesFile() error {
	modulesFile := "/etc/modules"

	// 检查文件是否存在
	if _, err := os.Stat(modulesFile); os.IsNotExist(err) {
		// 文件不存在，创建一个新文件
		fmt.Printf("创建新的 %s 文件", modulesFile)
		if err := os.WriteFile(modulesFile, []byte{}, 0644); err != nil {
			return fmt.Errorf("创建 %s 文件失败: %w", modulesFile, err)
		}
	} else if err != nil {
		// 处理其他错误
		return fmt.Errorf("检查 %s 文件失败: %w", modulesFile, err)
	}

	data, err := os.ReadFile(modulesFile)
	if err != nil {
		return err
	}

	content := string(data)
	requiredModules := []string{
		"vfio",
		"vfio_iommu_type1",
		"vfio_pci",
	}

	var lines []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	// 添加缺失的模块
	for _, module := range requiredModules {
		if !contains(lines, module) {
			lines = append(lines, module)
		}
	}

	// 写入更新后的内容
	newContent := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(modulesFile, []byte(newContent), 0644)
}

// modifyBlacklistConfig 修改 /etc/modprobe.d/pve-blacklist.conf 文件
func modifyBlacklistConfig() error {
	blacklistFile := "/etc/modprobe.d/blacklist.conf"

	// 确保文件存在
	if _, err := os.Stat(blacklistFile); os.IsNotExist(err) {
		if err := os.WriteFile(blacklistFile, []byte{}, 0644); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(blacklistFile)
	if err != nil {
		return err
	}

	content := string(data)
	requiredLines := []string{
		"blacklist i915",
		"blacklist snd_hda_intel",
		"blacklist snd_hda_codec_hdmi",
		"options vfio_iommu_type1 allow_unsafe_interrupts=1",
	}

	// 添加标记行，用于检测是否已添加配置
	markerLine := "# Added by fastpve for GPU passthrough"

	// 如果已包含标记行，说明配置已添加
	if strings.Contains(content, markerLine) {
		log.Println("黑名单配置已存在，跳过修改")
		return nil
	}

	// 添加配置
	newConfig := markerLine + "\n" + strings.Join(requiredLines, "\n") + "\n"
	content += "\n" + newConfig

	return os.WriteFile(blacklistFile, []byte(content), 0644)
}

// detectCPUModel 检测 CPU 型号
func detectCPUModel() (string, error) {
	output, err := exec.Command("lscpu").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Model name") {
			fields := strings.SplitN(line, ":", 2)
			if len(fields) == 2 {
				model := strings.TrimSpace(fields[1])
				if strings.Contains(model, "N150") {
					return "N150", nil
				} else if strings.Contains(model, "N100") {
					return "N100", nil
				}
				return "", errors.New("不支持该核显的直通")
			}
		}
	}
	return "", errors.New("未检测到 CPU 型号")
}

// modifyI915Config 根据 CPU 型号修改 /etc/modprobe.d/i915.conf 文件（Intel核显名称就是i915,如果独显文件名可能是vfio.conf）
func modifyI915Config(cpuModel string) error {
	i915File := "/etc/modprobe.d/i915.conf"
	// 确定型号的cpu其设备id貌似是固定的
	var gpuID, audioID string
	switch cpuModel {
	case "N150":
		gpuID = "46d4"
		audioID = "54c8"
	case "N100":
		gpuID = "46d1"
		audioID = "54c8"
	default:
		return errors.New("不支持的 CPU 型号")
	}
	content := fmt.Sprintf("options vfio-pci ids=8086:%s,8086:%s\n", gpuID, audioID)
	err := os.WriteFile(i915File, []byte(content), 0644)
	return err
}

// updateInitramfs 更新内核
func updateInitramfs() error {
	return utils.BatchRunStdout(context.TODO(), []string{"update-initramfs -u"}, 0)
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
