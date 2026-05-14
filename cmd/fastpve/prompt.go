package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/solider245/fastpve/utils"
	"github.com/solider245/fastpve/vmdownloader"
	"github.com/manifoldco/promptui"
)

var errContinue = errors.New("continue")

type menuItem struct {
	label  string
	action func() error
}

func mainPrompt() error {
	var items []menuItem

	items = append(items,
		menuItem{"0、PVE系统工具", promptSystemTools},
		menuItem{"1、安装iStoreOS", promptForIstore},
		menuItem{"2、安装Windows", promptInstallWindows},
		menuItem{"3、安装Ubuntu", promptForUbuntu},
		menuItem{"4、VM管理", promptManageVMs},
		menuItem{"5、镜像管理", promptManageImages},
		menuItem{"6、LXC容器管理", promptForLXC},
	)

	// Inject all DD presets into the main menu
	idx := 7
	for _, cat := range vmdownloader.AllDDPresetCategories() {
		for i := range cat.Presets {
			p := cat.Presets[i]
			items = append(items, menuItem{
				label:  fmt.Sprintf("%d、%s", idx, p.Name),
				action: makeDDPresetAction(p),
			})
			idx++
		}
	}

	// Custom URL
	items = append(items, menuItem{
		label:  fmt.Sprintf("%d、自定义URL", idx),
		action: promptForDD,
	})

	// Quit
	items = append(items, menuItem{
		label:  "q、退出",
		action: func() error { return nil },
	})

	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.label
	}

	prompt := promptui.Select{
		Label: "按上下键操作：",
		Items: labels,
		Size:  25,
	}

	printPVEStatus()

	for {
		_, result, err := prompt.Run()
		if err != nil {
			fmt.Println("操作取消:", err); return err
		}

		for _, it := range items {
			if it.label == result {
				err := it.action()
				if err == errContinue {
					goto CONTINUE
				}
				return err
			}
		}
		// "q" or unrecognized → exit
		break
	CONTINUE:
	}

	return nil
}

func printPVEStatus() {
	ctx := context.TODO()
	verOut, _ := utils.BatchOutput(ctx, []string{"pveversion 2>/dev/null | head -1 || echo 'PVE ?'"}, 3)
	ver := strings.TrimSpace(string(verOut))
	memOut, _ := utils.BatchOutput(ctx, []string{"free -h | awk '/Mem/{print $3 \"/\" $2}'"}, 3)
	mem := strings.TrimSpace(string(memOut))
	diskOut, _ := utils.BatchOutput(ctx, []string{"df -h / | awk 'NR==2{print $3 \"/\" $2}'"}, 3)
	disk := strings.TrimSpace(string(diskOut))
	fmt.Printf("\n  %s | 内存: %s | 磁盘: %s\n\n", ver, mem, disk)
}
