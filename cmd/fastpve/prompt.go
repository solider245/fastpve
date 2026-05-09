package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/linkease/fastpve/vmdownloader"
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
		menuItem{"0、更换软件源", promptForSources},
		menuItem{"1、安装Docker", promptForDocker},
		menuItem{"2、安装iStoreOS", promptForIstore},
		menuItem{"3、安装Windows", promptInstallWindows},
		menuItem{"4、安装Ubuntu", promptForUbuntu},
		menuItem{"5、一键核显直通", promptForGPUPassThrough},
		menuItem{"6、VM管理", promptManageVMs},
		menuItem{"7、镜像管理", promptManageImages},
	)

	// Inject all DD presets into the main menu
	idx := 8
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

	for {
		_, result, err := prompt.Run()
		if err != nil {
			os.Exit(-1)
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
