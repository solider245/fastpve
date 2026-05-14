package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

type configTemplate struct {
	Name   string
	Cores  int
	Memory int // MB
	Disk   int // GB
}

func promptConfigTemplate(defCores, defMemMB, defDiskGB int) (cores, memMB, diskGB int, err error) {
	templates := []configTemplate{
		{"微型", 1, 512, 5},
		{"小型", 2, 2048, 20},
		{"中型", 4, 4096, 40},
		{"大型", 8, 8192, 80},
		{fmt.Sprintf("推荐（%d核/%dMB/%dGB）", defCores, defMemMB, defDiskGB), defCores, defMemMB, defDiskGB},
	}

	items := make([]string, len(templates)+1)
	for i, t := range templates {
		items[i] = fmt.Sprintf("%s (%dC / %dMB / %dGB)", t.Name, t.Cores, t.Memory, t.Disk)
	}
	items[len(templates)] = "自定义"

	prompt := promptui.Select{
		Label: "选择配置模板",
		Items: items,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return 0, 0, 0, err
	}

	if idx == len(templates) {
		cores, err = promptIntWithDefault("CPU核数", defCores)
		if err != nil {
			return
		}
		memMB, err = promptIntWithDefault("内存大小/MB", defMemMB)
		if err != nil {
			return
		}
		diskGB, err = promptIntWithDefault("磁盘大小/GB", defDiskGB)
		if err != nil {
			return
		}
		return
	}

	t := templates[idx]
	return t.Cores, t.Memory, t.Disk, nil
}
