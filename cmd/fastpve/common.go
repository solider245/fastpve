package main

import (
	"strconv"
	"strings"

	"github.com/solider245/fastpve/utils"
	"github.com/manifoldco/promptui"
)

func selectNumber(label1, label2 string,
	items []string,
	numBegin int) (int, error) {
	prompt := promptui.Select{
		Label: label1,
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return 0, err
	}
	if idx < len(items)-1 {
		return int(1 << (idx + numBegin)), nil
	}
	return promptInputNumber(label2)
}

func promptInputNumber(label string) (int, error) {
	prompt := promptui.Prompt{
		Label: label,
		Validate: func(input string) error {
			_, err := strconv.Atoi(input)
			return err
		},
	}
	input, err := prompt.Run()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(input)
}

func promptPVECore() (int, error) {
	cores := []string{"2核", "4核心", "8核心", "自定义"}
	return selectNumber("虚拟机核心：",
		"虚拟机CPU核心数：",
		cores,
		1)
}

func promptPVEMemory() (int, error) {
	memories := []string{"1024MB", "2048MB", "4096MB", "8192MB", "自定义"}
	return selectNumber("虚拟机内存大小：",
		"自定义内存（MB）：",
		memories,
		10)
}

func promptPVEDisk() (int, error) {
	disks := []string{"64GB", "128GB", "256GB", "512GB", "1TB", "自定义（GB）"}
	return selectNumber("选择硬盘",
		"自定义硬盘：（GB）",
		disks,
		6)
}

func toBetterWindowName(winName string) string {
	winName = utils.CleanString(strings.TrimSuffix(winName, ".iso"))
	ss := strings.Split(winName, "-")
	for i, s := range ss {
		if len(s) == 0 {
			continue
		}
		ss[i] = strings.ToUpper(s[0:1]) + strings.ToLower(s[1:])
	}
	return strings.Join(ss, "-")
}

func toBetterIstoreName(name string) string {
	name = utils.CleanString(name)
	ss := strings.Split(name, "-")
	if len(ss) >= 3 {
		return strings.Join(ss[:3], "-")
	}
	return name
}

func toBetterUbuntuName(name string) string {
	name = utils.CleanString(name)
	ss := strings.Split(name, "-")
	if len(ss) >= 3 {
		return strings.Join(ss[:3], "-")
	}
	return name
}
