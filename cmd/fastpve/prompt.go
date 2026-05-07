package main

import (
	"errors"
	"os"
	"sort"

	"github.com/manifoldco/promptui"
)

type mainSelection int

const (
	selectChangeSources mainSelection = iota
	selectInstallDocker
	selectInstallIstore
	selectInstallWindows
	selectInstallUbuntu
	selectOneClickGPUPassThrough
	selectInstallDD
)

const (
	selectQuit mainSelection = 255
)

var errContinue = errors.New("continue")

var (
	mainMenu = map[string]mainSelection{
		"0、更换软件源":      selectChangeSources,
		"1、安装Docker":   selectInstallDocker,
		"2、安装iStoreOS": selectInstallIstore,
		"3、安装Windows":  selectInstallWindows,
		"4、安装Ubuntu":   selectInstallUbuntu,
		// 目前只做Intel核显直通
		"5、一键核显直通":   selectOneClickGPUPassThrough,
		"6、安装DD镜像":    selectInstallDD,
		"q、退出":       selectQuit,
	}
)

// https://github.com/nicksnyder/go-i18n
func getMainSelection(mainItems []string) (mainSelection, error) {
	prompt := promptui.Select{
		Label: "按上下键操作：",
		Items: mainItems,
	}

	_, result, err := prompt.Run()
	if err != nil {
		os.Exit(-1)
		//return mainSelection(selectQuit), nil
	}
	if idx, ok := mainMenu[result]; ok {
		return mainSelection(idx), nil
	}

	return -2, errors.New("item not found")
}

func mainPrompt() error {
	mainItems := make([]string, len(mainMenu))
	i := 0
	for k := range mainMenu {
		mainItems[i] = k
		i++
	}
	mainItems = mainItems[:i]
	sort.Strings(mainItems)

MAINLOOP:
	for {
		selectIdx, err := getMainSelection(mainItems)
		if err != nil {
			return err
		}
		switch selectIdx {
		case selectChangeSources:
			err = promptForSources()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectInstallDocker:
			err = promptForDocker()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectInstallIstore:
			err = promptForIstore()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectInstallWindows:
			err = promptInstallWindows()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectInstallUbuntu:
			err = promptForUbuntu()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectOneClickGPUPassThrough:
			err = promptForGPUPassThrough()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectInstallDD:
			err = promptForDD()
			if err == errContinue {
				continue MAINLOOP
			}
			return err
		case selectQuit:
			break MAINLOOP
		}
	}

	return nil
}
