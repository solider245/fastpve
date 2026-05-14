package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/solider245/fastpve/utils"
	"github.com/manifoldco/promptui"
)

type lxcTemplate struct {
	section     string
	name        string
	description string
}

type lxcContainer struct {
	vmid   int
	status string
}

type lxcPreset struct {
	Name        string
	Keyword     string // keyword to match in `pveam available`
	DefHostname string
	Cores       int
	Memory      int
	Disk        int
}

var lxcPresets = []lxcPreset{
	{"Ubuntu 24.04 LTS", "ubuntu-24.04", "ubuntu2404", 2, 1024, 8},
	{"Ubuntu 22.04 LTS", "ubuntu-22.04", "ubuntu2204", 2, 1024, 8},
	{"Debian 12", "debian-12", "debian12", 2, 1024, 8},
	{"Debian 11", "debian-11", "debian11", 2, 1024, 8},
	{"Alpine Latest", "alpine", "alpine", 1, 512, 4},
	{"CentOS 9 Stream", "centos-9", "centos9", 2, 1024, 8},
	{"RockyLinux 9", "rockylinux-9", "rocky9", 2, 1024, 8},
}

func promptQuickCreateLXC() error {
	items := make([]string, len(lxcPresets)+1)
	for i, p := range lxcPresets {
		items[i] = fmt.Sprintf("%s (%d核/%dMB/%dGB)", p.Name, p.Cores, p.Memory, p.Disk)
	}
	items[len(lxcPresets)] = "返回"

	prompt := promptui.Select{
		Label: "选择快速创建的系统",
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil || idx >= len(lxcPresets) {
		return errContinue
	}
	preset := lxcPresets[idx]

	fmt.Printf("正在查找 %s 模板...\n", preset.Name)
	templateName, err := findLXCTemplate(preset.Keyword)
	if err != nil {
		return err
	}

	_, vmid, err := resolveStorageAndVMID()
	if err != nil {
		return err
	}

	hostname, err := promptLXCBasicInfo(preset.DefHostname)
	if err != nil {
		return err
	}
	password, err := promptLXCPassword()
	if err != nil {
		return err
	}

	cores, _ := promptIntWithDefault("CPU核数", preset.Cores)
	memory, _ := promptIntWithDefault("内存 (MB)", preset.Memory)
	disk, _ := promptIntWithDefault("磁盘 (GB)", preset.Disk)
	bridge := promptBridge()

	scripts := []string{
		"set -e",
		fmt.Sprintf("pct create %d local:vztmpl/%s --hostname %s --password '%s' --rootfs local:%dG --memory %d --cores %d --swap 512 --net0 name=eth0,bridge=%s,ip=dhcp --unprivileged 1 --features nesting=1",
			vmid, templateName, hostname, password, disk, memory, cores, bridge),
		fmt.Sprintf("pct start %d", vmid),
	}
	if err = utils.BatchRunStdout(context.TODO(), scripts, 120); err != nil {
		return err
	}
	fmt.Printf("LXC容器 %d (%s) 已创建并启动\n", vmid, hostname)
	return nil
}

func promptForLXC() error {
	items := []menuItem{
		{"快速创建", promptQuickCreateLXC},
		{"部署服务", promptDeployLXCService},
		{"创建容器", promptCreateLXC},
		{"管理容器", promptManageLXC},
		{"模板管理", promptLXCTemplates},
		{"返回", func() error { return errContinue }},
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}

	prompt := promptui.Select{
		Label: "LXC容器管理",
		Items: labels,
	}

	for {
		idx, _, err := prompt.Run()
		if err != nil {
			return errContinue
		}
		err = items[idx].action()
		if err == errContinue {
			return errContinue
		}
		if err != nil {
			fmt.Println("操作失败:", err)
		}
	}
}

func promptCreateLXC() error {
	fmt.Println("正在获取已下载的LXC模板列表...")
	out, err := utils.BatchOutput(context.TODO(), []string{"pveam list local 2>/dev/null || true"}, 10)
	if err != nil {
		return err
	}
	templates := parsePveamList(out)
	if len(templates) == 0 {
		fmt.Println("暂无已下载的LXC模板，请先通过「模板管理」下载")
		return nil
	}

	tplNames := make([]string, len(templates)+1)
	for i, t := range templates {
		tplNames[i] = fmt.Sprintf("%s  %s", t.name, t.description)
	}
	tplNames[len(templates)] = "返回"

	prompt := promptui.Select{
		Label: "选择LXC模板",
		Items: tplNames,
	}
	idx, _, err := prompt.Run()
	if err != nil || idx >= len(templates) {
		return errContinue
	}

	_, vmid, err := resolveStorageAndVMID()
	if err != nil {
		return err
	}

	selected := templates[idx]
	defName := strings.TrimSuffix(selected.name, ".tar.zst")
	defName = strings.TrimSuffix(defName, ".tar.gz")
	defName = strings.TrimSuffix(defName, ".tar.xz")

	hostnamePrompt := promptui.Prompt{
		Label:   "主机名",
		Default: defName,
	}
	hostname, err := hostnamePrompt.Run()
	if err != nil {
		return errContinue
	}
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		hostname = defName
	}

	pwdPrompt := promptui.Prompt{
		Label: "root密码",
		Mask:  '*',
	}
	password, err := pwdPrompt.Run()
	if err != nil {
		return errContinue
	}

	promptCore := promptui.Prompt{
		Label:   "CPU核数",
		Default: "2",
	}
	coresStr, _ := promptCore.Run()
	cores, _ := strconv.Atoi(strings.TrimSpace(coresStr))
	if cores < 1 {
		cores = 2
	}

	promptMem := promptui.Prompt{
		Label:   "内存 (MB)",
		Default: "1024",
	}
	memStr, _ := promptMem.Run()
	memory, _ := strconv.Atoi(strings.TrimSpace(memStr))
	if memory < 128 {
		memory = 1024
	}

	promptDisk := promptui.Prompt{
		Label:   "磁盘 (GB)",
		Default: "8",
	}
	diskStr, _ := promptDisk.Run()
	disk, _ := strconv.Atoi(strings.TrimSpace(diskStr))
	if disk < 1 {
		disk = 8
	}

	bridge := promptBridge()

	scripts := []string{
		"set -e",
		fmt.Sprintf("pct create %d local:vztmpl/%s --hostname %s --password '%s' --storage local --memory %d --cores %d --swap 512 --net0 name=eth0,bridge=%s,ip=dhcp --unprivileged 1 --features nesting=1",
			vmid, selected.name, hostname, password, memory, cores, bridge),
		fmt.Sprintf("pct start %d", vmid),
	}
	err = utils.BatchRunStdout(context.TODO(), scripts, 120)
	if err != nil {
		return err
	}
	fmt.Printf("LXC容器 %d (%s) 已创建并启动\n", vmid, hostname)
	return nil
}

func promptManageLXC() error {
	out, err := utils.BatchOutput(context.TODO(), []string{"pct list 2>/dev/null || true"}, 10)
	if err != nil {
		return err
	}
	containers := parsePCTList(out)
	if len(containers) == 0 {
		fmt.Println("当前没有LXC容器")
		return nil
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].vmid < containers[j].vmid
	})

	ctLabels := make([]string, len(containers)+1)
	for i, ct := range containers {
		ctLabels[i] = fmt.Sprintf("%d [%s]", ct.vmid, ct.status)
	}
	ctLabels[len(containers)] = "返回"

	ctPrompt := promptui.Select{
		Label: "选择LXC容器",
		Items: ctLabels,
	}
	ctIdx, _, err := ctPrompt.Run()
	if err != nil || ctIdx >= len(containers) {
		return errContinue
	}
	ct := containers[ctIdx]

	actions := []string{"启动", "停止", "重启", "进入控制台", "删除", "返回"}
	actPrompt := promptui.Select{
		Label: fmt.Sprintf("操作 CT %d", ct.vmid),
		Items: actions,
	}
	actIdx, _, err := actPrompt.Run()
	if err != nil {
		return errContinue
	}

	switch actIdx {
	case 0:
		_, err = utils.BatchOutput(context.TODO(), []string{fmt.Sprintf("pct start %d", ct.vmid)}, 60)
	case 1:
		_, err = utils.BatchOutput(context.TODO(), []string{fmt.Sprintf("pct stop %d", ct.vmid)}, 60)
	case 2:
		_, err = utils.BatchOutput(context.TODO(), []string{fmt.Sprintf("pct restart %d", ct.vmid)}, 60)
	case 3:
		utils.BatchRunStdout(context.TODO(), []string{fmt.Sprintf("pct enter %d", ct.vmid)}, 0)
		return nil
	case 4:
		confirmPrompt := promptui.Prompt{
			Label: fmt.Sprintf("确认删除 CT %d? (yes/NO)", ct.vmid),
		}
		confirm, _ := confirmPrompt.Run()
		if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
			return nil
		}
		utils.BatchRun(context.TODO(), []string{
			fmt.Sprintf("pct stop %d --force 2>/dev/null; true", ct.vmid),
			fmt.Sprintf("pct destroy %d", ct.vmid),
		}, 30)
		fmt.Printf("CT %d 已删除\n", ct.vmid)
		return nil
	case 5:
		return errContinue
	}

	if err != nil {
		fmt.Println("操作失败:", err)
	} else {
		fmt.Println("操作成功")
	}
	return nil
}

func promptLXCTemplates() error {
	items := []menuItem{
		{"下载模板", promptDownloadLXCTemplate},
		{"删除模板", promptRemoveLXCTemplate},
		{"返回", func() error { return errContinue }},
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}

	prompt := promptui.Select{
		Label: "LXC模板管理",
		Items: labels,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return errContinue
	}
	return items[idx].action()
}

func promptDownloadLXCTemplate() error {
	fmt.Println("正在获取可用模板列表（可能需要几秒钟）...")
	out, err := utils.BatchOutput(context.TODO(), []string{"pveam available 2>/dev/null || true"}, 60)
	if err != nil {
		return err
	}
	templates := parsePveamAvailable(out)
	if len(templates) == 0 {
		fmt.Println("获取模板列表失败，请检查网络和PVE源配置")
		return nil
	}

	tplNames := make([]string, len(templates)+1)
	for i, t := range templates {
		tplNames[i] = fmt.Sprintf("[%s] %s  %s", t.section, t.name, t.description)
	}
	tplNames[len(templates)] = "返回"

	prompt := promptui.Select{
		Label:   "选择要下载的模板",
		Items:   tplNames,
		Size:    15,
	}
	idx, _, err := prompt.Run()
	if err != nil || idx >= len(templates) {
		return errContinue
	}
	selected := templates[idx]

	fmt.Printf("正在下载 %s ...\n", selected.name)
	return utils.BatchRunStdout(context.TODO(), []string{
		fmt.Sprintf("pveam download local %s", selected.name),
	}, 300)
}

func promptRemoveLXCTemplate() error {
	out, err := utils.BatchOutput(context.TODO(), []string{"pveam list local 2>/dev/null || true"}, 10)
	if err != nil {
		return err
	}
	templates := parsePveamList(out)
	if len(templates) == 0 {
		fmt.Println("没有已下载的模板")
		return nil
	}

	tplNames := make([]string, len(templates)+1)
	for i, t := range templates {
		tplNames[i] = fmt.Sprintf("%s  %s", t.name, t.description)
	}
	tplNames[len(templates)] = "返回"

	prompt := promptui.Select{
		Label: "选择要删除的模板",
		Items: tplNames,
	}
	idx, _, err := prompt.Run()
	if err != nil || idx >= len(templates) {
		return errContinue
	}
	selected := templates[idx]

	confirmPrompt := promptui.Prompt{
		Label: fmt.Sprintf("确认删除 %s? (yes/NO)", selected.name),
	}
	confirm, _ := confirmPrompt.Run()
	if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
		return nil
	}

	return utils.BatchRunStdout(context.TODO(), []string{
		fmt.Sprintf("pveam remove local:vztmpl/%s", selected.name),
	}, 30)
}

func parsePveamList(out []byte) []lxcTemplate {
	lines := strings.Split(string(out), "\n")
	var templates []lxcTemplate
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "template-name") {
			continue
		}
		// Format: "storage:template-name   size   description"
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// First field is "storage:template-name"
		volID := fields[0]
		if idx := strings.Index(volID, ":"); idx >= 0 {
			volID = volID[idx+1:]
		}
		templates = append(templates, lxcTemplate{
			name:        volID,
			description: strings.Join(fields[2:], " "),
		})
	}
	return templates
}

func parsePveamAvailable(out []byte) []lxcTemplate {
	lines := strings.Split(string(out), "\n")
	var templates []lxcTemplate
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "section") {
			continue
		}
		// section  name-without-spaces  description with spaces
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		templates = append(templates, lxcTemplate{
			section:     fields[0],
			name:        fields[1],
			description: strings.Join(fields[2:], " "),
		})
	}
	return templates
}

func parsePCTList(out []byte) []lxcContainer {
	lines := strings.Split(string(out), "\n")
	var containers []lxcContainer
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		vmid, _ := strconv.Atoi(fields[0])
		if vmid == 0 {
			continue
		}
		containers = append(containers, lxcContainer{
			vmid:   vmid,
			status: fields[1],
		})
	}
	return containers
}

func promptLXCBasicInfo(defHostname string) (string, error) {
	prompt := promptui.Prompt{
		Label:   "主机名",
		Default: defHostname,
	}
	hostname, err := prompt.Run()
	if err != nil {
		return "", errContinue
	}
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		hostname = defHostname
	}
	return hostname, nil
}

func promptLXCPassword() (string, error) {
	prompt := promptui.Prompt{
		Label: "root密码",
		Mask:  '*',
	}
	password, err := prompt.Run()
	if err != nil {
		return "", errContinue
	}
	return password, nil
}
