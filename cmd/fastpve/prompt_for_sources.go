package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/solider245/fastpve/quickget"
	"github.com/solider245/fastpve/utils"
	"github.com/manifoldco/promptui"
)

type sourceInfo struct {
	source string
	dns    string
}

func promptForSources() error {
	keys := []string{"oneclick", "reverse", "sourceOnly", "dnsOnly"}
	titles := []string{"一键优化网络（源+DNS+LXC等）", "恢复官方", "仅换源", "仅换DNS"}
	prompt := promptui.Select{
		Label: "操作选择：",
		Items: titles,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	switch keys[idx] {
	case "oneclick":
		return promptOneClick()
	case "reverse":
		return runReverse()
	case "sourceOnly":
		return promptSourceOnly()
	case "dnsOnly":
		return promptDnsOnly()
	}
	return nil
}

func promptOneClick() error {
	info := &sourceInfo{}
	err := promptSource(info, true)
	if err != nil {
		return err
	}
	err = promptDns(info, true)
	if err != nil {
		return err
	}

	prompt := promptui.Select{
		Label: "安装成功后重启？",
		Items: []string{"重启（推荐）", "不重启，我自己重启"},
	}
	var reboot bool
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx == 0 {
		reboot = true
	}

	switch info.source {
	case "mirrors.tuna.tsinghua.edu.cn":
		err = useTunaSource()
	case "mirrors.ustc.edu.cn":
		err = useUstcSource()
	default:
	}
	if err != nil {
		return err
	}

	if info.dns != "" {
		err = writeDNS(info.dns)
		if err != nil {
			return err
		}
	}

	if err = writeCeph(); err != nil {
		return err
	}

	if _, err := os.Stat("/etc/apt/sources.list.d/pve-enterprise.list"); err == nil {
		utils.BatchRun(context.TODO(), []string{
			"mv /etc/apt/sources.list.d/pve-enterprise.list /etc/apt/sources.list.d/pve-enterprise.list.bak",
		}, 0)
	}

	if _, err := os.Stat("/usr/share/perl5/PVE/APLInfo.pm_back"); err != nil {
		return utils.BatchRun(context.TODO(), []string{
			"cp /usr/share/perl5/PVE/APLInfo.pm /usr/share/perl5/PVE/APLInfo.pm_back",
			`sed -i.bak 's|http://download.proxmox.com|https://mirrors.ustc.edu.cn/proxmox|g' /usr/share/perl5/PVE/APLInfo.pm`,
		}, 0)
	}

	if err = runAptUpdate(); err != nil {
		return err
	}
	if reboot {
		fmt.Println("等待 10s 后重启...")
		time.Sleep(time.Second * 10)
		utils.BatchRun(context.TODO(), []string{"reboot"}, 0)
	}
	return nil
}

func promptSource(info *sourceInfo, notChanged bool) error {
	keys := []string{"mirrors.tuna.tsinghua.edu.cn", "mirrors.ustc.edu.cn"}
	items := []string{
		"清华大学（mirrors.tuna.tsinghua.edu.cn）",
		"中科大（mirrors.ustc.edu.cn）",
	}
	origLen := len(items)
	if notChanged {
		items = append(items, "不修改源")
	}
	prompt := promptui.Select{
		Label: "选择源：（/etc/apt/sources.list）",
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx < origLen {
		info.source = keys[idx]
	}
	return nil
}

func promptDns(info *sourceInfo, notChanged bool) error {
	items := []string{"223.5.5.5", "119.29.29.29"}
	origLen := len(items)
	if notChanged {
		items = append(items, "不修改DNS")
	}
	prompt := promptui.Select{
		Label: "选择DNS：",
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx < origLen {
		info.dns = items[idx]
	}
	return nil
}

func useTunaSource() error {
	ss := []string{
		"deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free non-free-firmware",
		"deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free non-free-firmware",
		"deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-backports main contrib non-free non-free-firmware",
		"deb https://mirrors.tuna.tsinghua.edu.cn/debian-security bookworm-security main contrib non-free non-free-firmware",
	}
	err := os.WriteFile("/etc/apt/sources.list", []byte(strings.Join(ss, "\n")), 0644)
	if err != nil {
		return err
	}
	ss = []string{
		"deb https://mirrors.tuna.tsinghua.edu.cn/proxmox/debian/pve bookworm pve-no-subscription",
	}
	f := "/etc/apt/sources.list.d/pve-no-subscription.list"
	if _, err := os.Stat(f); err == nil {
		return os.WriteFile(f, []byte(strings.Join(ss, "\n")), 0644)
	}
	return nil
}

func useUstcSource() error {
	ss := []string{
		"deb https://mirrors.ustc.edu.cn/debian/ bookworm main contrib non-free non-free-firmware",
		"deb https://mirrors.ustc.edu.cn/debian/ bookworm-updates main contrib non-free non-free-firmware",
		"deb https://mirrors.ustc.edu.cn/debian/ bookworm-backports main contrib non-free non-free-firmware",
		"deb https://mirrors.ustc.edu.cn/debian-security bookworm-security main contrib non-free non-free-firmware",
	}
	err := os.WriteFile("/etc/apt/sources.list", []byte(strings.Join(ss, "\n")), 0644)
	if err != nil {
		return err
	}
	ss = []string{
		"deb https://mirrors.ustc.edu.cn/proxmox/debian/pve bookworm pve-no-subscription",
	}
	f := "/etc/apt/sources.list.d/pve-no-subscription.list"
	if _, err := os.Stat(f); err == nil {
		return os.WriteFile(f, []byte(strings.Join(ss, "\n")), 0644)
	}
	return nil
}

func writeCeph() error {
	if _, err := os.Stat("/etc/apt/sources.list.d/ceph.list"); err != nil {
		return nil
	}
	return utils.BatchRun(context.TODO(), []string{
		"CEPH_CODENAME=`ceph -v | grep ceph | awk '{print $(NF-1)}'`",
		`source /etc/os-release`,
		`echo "deb https://mirrors.ustc.edu.cn/proxmox/debian/ceph-$CEPH_CODENAME $VERSION_CODENAME no-subscription" > /etc/apt/sources.list.d/ceph.list`,
	}, 0)
}

func writeDNS(dns string) error {
	fIn, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return err
	}
	lines := strings.Split(string(fIn), "\n")
	var search string
	for _, line := range lines {
		if strings.Contains(line, "search") {
			search = line
			break
		}
	}
	newLine := strings.Join([]string{search, "nameserver " + dns}, "\n")
	return os.WriteFile("/etc/resolv.conf", []byte(newLine), 0644)
}

func promptSourceOnly() error {
	info := &sourceInfo{}
	err := promptSource(info, false)
	if err != nil {
		return err
	}
	switch info.source {
	case "mirrors.tuna.tsinghua.edu.cn":
		err = useTunaSource()
	case "mirrors.ustc.edu.cn":
		err = useUstcSource()
	default:
	}
	if err != nil {
		return err
	}
	return runAptUpdate()
}

func promptDnsOnly() error {
	info := &sourceInfo{}
	err := promptDns(info, false)
	if err != nil {
		return err
	}
	dns := info.dns
	if dns == "" {
		return nil
	}
	return writeDNS(dns)
}

func runReverse() error {
	prompt := promptui.Select{
		Label: "恢复后重启？",
		Items: []string{"重启（推荐）", "不重启，我自己重启"},
	}
	var reboot bool
	idx, _, err := prompt.Run()
	if err != nil {
		return err
	}
	if idx == 0 {
		reboot = true
	}
	b, err := quickget.PveReverseScripts()
	if err != nil {
		return err
	}
	ss := strings.Split(string(b), "\n")
	err = utils.BatchRun(context.TODO(), ss, 0)
	if err != nil {
		return err
	}
	if err = runAptUpdate(); err != nil {
		return err
	}
	if reboot {
		fmt.Println("等待 10s 后重启...")
		time.Sleep(time.Second * 10)
		utils.BatchRun(context.TODO(), []string{"reboot"}, 0)
	}
	return nil
}

func runAptUpdate() error {
	return utils.BatchRunStdout(context.TODO(), []string{
		"apt update && apt dist-upgrade -y",
	}, 0)
}
