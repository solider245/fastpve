package main

import (
	"context"
	"strings"

	"github.com/solider245/fastpve/quickget"
	"github.com/solider245/fastpve/utils"
	"github.com/manifoldco/promptui"
)

func promptForDocker() error {
	prompt := promptui.Select{
		Label: "加速安装Docker源选择：",
		Items: []string{"mirrors.tuna.tsinghua.edu.cn", "mirrors.huaweicloud.com", "mirrors.ustc.edu.cn"},
	}
	_, txt, err := prompt.Run()
	if err != nil {
		return err
	}
	scripts := strings.Split(quickget.DockerScripts(txt), "\n")
	return utils.BatchRunStdout(context.TODO(), scripts, 0)
}
