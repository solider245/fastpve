package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/linkease/fastpve/quickget"
	"github.com/linkease/fastpve/utils"
	"github.com/linkease/fastpve/vmdownloader"
	"github.com/manifoldco/promptui"
)

func promptManageVMs() error {
	items, err := quickget.QMList()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("当前没有虚拟机")
		return errContinue
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].VMID < items[j].VMID
	})

	vmLabels := make([]string, len(items)+1)
	for i, vm := range items {
		memStr := "?"
		if vm.MemMB > 0 {
			memStr = fmt.Sprintf("%dMB", vm.MemMB)
		}
		diskStr := "?"
		if vm.BootDiskMB > 0 {
			diskStr = fmt.Sprintf("%dMB", vm.BootDiskMB)
		}
		ipStr := getVMIP(vm.VMID)
		vmLabels[i] = fmt.Sprintf("%d: %s [%s] MEM:%s DISK:%s IP:%s",
			vm.VMID, vm.Name, vm.Status, memStr, diskStr, ipStr)
	}
	vmLabels[len(items)] = "返回"

	vmPrompt := promptui.Select{
		Label: "选择虚拟机",
		Items: vmLabels,
	}
	vmIdx, _, err := vmPrompt.Run()
	if err != nil || vmIdx >= len(items) {
		return errContinue
	}

	vm := items[vmIdx]

	actions := []string{"启动", "停止", "重启", "查看详情", "删除", "返回"}
	actPrompt := promptui.Select{
		Label: fmt.Sprintf("操作 VM %d (%s)", vm.VMID, vm.Name),
		Items: actions,
	}
	actIdx, _, err := actPrompt.Run()
	if err != nil {
		return errContinue
	}

	ctx := context.TODO()
	switch actIdx {
	case 0:
		_, err = utils.BatchOutput(ctx, []string{fmt.Sprintf("qm start %d", vm.VMID)}, 60)
	case 1:
		_, err = utils.BatchOutput(ctx, []string{fmt.Sprintf("qm stop %d", vm.VMID)}, 60)
	case 2:
		_, err = utils.BatchOutput(ctx, []string{fmt.Sprintf("qm reboot %d", vm.VMID)}, 60)
	case 3:
		out, detailErr := utils.BatchOutput(ctx, []string{fmt.Sprintf("qm config %d --current 2>/dev/null || qm config %d", vm.VMID, vm.VMID)}, 5)
		if detailErr != nil {
			fmt.Println("获取详情失败:", detailErr)
		} else {
				fmt.Printf("VM %d (%s) 详情:\n%s\n", vm.VMID, vm.Name, strings.TrimSpace(string(out)))
		}
	case 4:
		confirmPrompt := promptui.Prompt{
			Label: fmt.Sprintf("确认删除 VM %d (%s)? (yes/NO)", vm.VMID, vm.Name),
		}
		confirm, _ := confirmPrompt.Run()
		if strings.ToLower(strings.TrimSpace(confirm)) == "yes" {
			_, err = utils.BatchOutput(ctx, []string{fmt.Sprintf("qm destroy %d --purge", vm.VMID)}, 120)
		}
	case 5:
		return errContinue
	}

	if err != nil {
		fmt.Println("操作失败:", err)
	} else {
		fmt.Println("操作成功")
	}
	return errContinue
}

func promptManageImages() error {
	isoPath := "/var/lib/vz/template/iso/"
	dirs, err := os.ReadDir(isoPath)
	if err != nil {
		return err
	}

	var images []os.DirEntry
	for _, dir := range dirs {
		if !dir.IsDir() {
			lower := strings.ToLower(dir.Name())
			if strings.HasSuffix(lower, ".img") || strings.HasSuffix(lower, ".iso") || strings.HasSuffix(lower, ".qcow2") || strings.HasSuffix(lower, ".raw") {
				images = append(images, dir)
			}
		}
	}

	if len(images) == 0 {
		fmt.Println("没有已下载的镜像文件")
		return errContinue
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Name() < images[j].Name()
	})

	imgLabels := make([]string, len(images)+1)
	for i, img := range images {
		info, _ := img.Info()
		sizeStr := "?"
		if info != nil {
			s := info.Size()
			switch {
			case s > 1024*1024*1024:
				sizeStr = fmt.Sprintf("%.1fGB", float64(s)/(1024*1024*1024))
			case s > 1024*1024:
				sizeStr = fmt.Sprintf("%.1fMB", float64(s)/(1024*1024))
			default:
				sizeStr = fmt.Sprintf("%.1fKB", float64(s)/1024)
			}
		}
		source := getPresetSource(img.Name())
		imgLabels[i] = fmt.Sprintf("%s (%s) %s", img.Name(), sizeStr, source)
	}
	imgLabels[len(images)] = "返回"

	imgPrompt := promptui.Select{
		Label: "选择镜像",
		Items: imgLabels,
	}
	imgIdx, _, err := imgPrompt.Run()
	if err != nil || imgIdx >= len(images) {
		return errContinue
	}

	img := images[imgIdx]
	imgActions := []string{"删除", "安装", "返回"}
	imgActPrompt := promptui.Select{
		Label: fmt.Sprintf("操作: %s", img.Name()),
		Items: imgActions,
	}
	imgActIdx, _, err := imgActPrompt.Run()
	if err != nil {
		return errContinue
	}
	switch imgActIdx {
	case 0:
		confirmPrompt := promptui.Prompt{
			Label: fmt.Sprintf("确认删除 %s? (yes/NO)", img.Name()),
		}
		confirm, _ := confirmPrompt.Run()
		if strings.ToLower(strings.TrimSpace(confirm)) == "yes" {
			err := os.Remove(filepath.Join(isoPath, img.Name()))
			if err != nil {
				fmt.Println("删除失败:", err)
			} else {
				fmt.Println("删除成功:", img.Name())
			}
		}
	case 1:
		fmt.Println()
		info := &ddInstallInfo{
			DDImgName: img.Name(),
			BIOSMode:  -1,
			Memory:    2048,
			Cores:     2,
			Disk:      20,
		}
		info.Cores, info.Memory, info.Disk, err = promptConfigTemplate(2, 2048, 20)
		if err != nil {
			return err
		}
		info.BIOSMode, err = promptDDBios()
		if err != nil {
			return err
		}
		ctx := context.TODO()
		if err := createDDVM(ctx, isoPath, info); err != nil {
			return err
		}
	case 2:
	}
	return errContinue
}


func getPresetSource(filename string) string {
	for _, cat := range vmdownloader.AllDDPresetCategories() {
		for _, p := range cat.Presets {
			for _, url := range p.URLs {
				if vmdownloader.FinalImageName(url) == filename {
					return "← " + p.Name
				}
			}
		}
	}
	return "← 自定义URL"
}


func getVMIP(vmid int) string {
	out, err := utils.BatchOutput(context.TODO(), []string{
		fmt.Sprintf("qm agent %d network-get-interfaces 2>/dev/null || true", vmid),
	}, 5)
	if err != nil {
		return "-"
	}
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		return "-"
	}
	ifaces, ok := result["result"].([]interface{})
	if !ok {
		return "-"
	}
	var ips []string
	for _, iface := range ifaces {
		m, ok := iface.(map[string]interface{})
		if !ok {
			continue
		}
		if m["name"] == "lo" {
			continue
		}
		addrs, ok := m["ip-addresses"].([]interface{})
		if !ok {
			continue
		}
		for _, addr := range addrs {
			a, ok := addr.(map[string]interface{})
			if !ok {
				continue
			}
			if a["ip-address-type"] == "ipv4" {
				if ip, ok := a["ip-address"].(string); ok {
					ips = append(ips, ip)
				}
			}
		}
	}
	if len(ips) == 0 {
		return "-"
	}
	return strings.Join(ips, ",")
}
