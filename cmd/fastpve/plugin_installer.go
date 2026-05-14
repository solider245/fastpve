package main

import (
	"fmt"
	"os"
	"strings"
)

func installPlugin() error {
	// Check which shell is in use
	shell := os.Getenv("SHELL")
	hasZsh := strings.Contains(shell, "zsh")

	if hasZsh {
		return installZshPlugin()
	}
	return installBashHook()
}

func installZshPlugin() error {
	fmt.Println("正在安装 zsh 插件...")

	// Check for oh-my-zsh
	omz := os.Getenv("ZSH")
	if omz == "" {
		omz = ""
		if home := os.Getenv("HOME"); home != "" {
			_, err := os.Stat(home + "/.oh-my-zsh")
			if err == nil {
				omz = home + "/.oh-my-zsh"
			}
		}
	}

	if omz != "" {
		// oh-my-zsh: install as custom plugin
		pluginDir := omz + "/custom/plugins/fastpve"
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return fmt.Errorf("创建插件目录失败: %w", err)
		}
		pluginFile := pluginDir + "/fastpve.plugin.zsh"
		if err := os.WriteFile(pluginFile, []byte(zshPluginCode), 0644); err != nil {
			return fmt.Errorf("写入插件文件失败: %w", err)
		}

		// Enable in .zshrc
		zshrc := os.Getenv("HOME") + "/.zshrc"
		if raw, err := os.ReadFile(zshrc); err == nil {
			content := string(raw)
			if !strings.Contains(content, "fastpve") {
				// Add to plugins list
				newContent := strings.Replace(content,
					"plugins=(",
					"plugins=(\n  fastpve", 1)
				if newContent != content {
					_ = os.WriteFile(zshrc, []byte(newContent), 0644)
				}
			}
		}
		fmt.Println("✅ oh-my-zsh 插件已安装，重新打开终端或执行 `source ~/.zshrc` 生效")
	} else {
		// Plain zsh: source from .zshrc
		zshrc := os.Getenv("HOME") + "/.zshrc"
		installLine := "\n# FastPVE AI 助手\ncommand_not_found_handler() {\n  fastpve ai \"$*\"\n  return 0\n}\n"
		if raw, err := os.ReadFile(zshrc); err == nil {
			if !strings.Contains(string(raw), "fastpve") {
				f, _ := os.OpenFile(zshrc, os.O_APPEND|os.O_WRONLY, 0644)
				if f != nil {
					_, _ = f.WriteString(installLine)
					f.Close()
				}
			}
		}
		fmt.Println("✅ zsh 已配置，重新打开终端或执行 `source ~/.zshrc` 生效")
	}

	fmt.Println("  现在可以直接在终端输入自然语言，例如「查看系统状态」")
	fmt.Println()
	return nil
}

func installBashHook() error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("无法检测用户目录")
	}

	bashrc := home + "/.bashrc"
	hook := `
# FastPVE AI 助手 — 命令找不到时自动转 AI
command_not_found_handle() {
  fastpve ai "$*"
  return 0
}
`
	raw, err := os.ReadFile(bashrc)
	if err != nil {
		// .bashrc might not exist
		_ = os.WriteFile(bashrc, []byte(hook), 0644)
		fmt.Println("✅ .bashrc 已创建并配置，重新打开终端生效")
		return nil
	}

	if strings.Contains(string(raw), "fastpve") {
		fmt.Println("bash 插件已安装，无需重复配置")
		return nil
	}

	f, _ := os.OpenFile(bashrc, os.O_APPEND|os.O_WRONLY, 0644)
	if f != nil {
		_, _ = f.WriteString(hook)
		f.Close()
	}

	fmt.Println("✅ bash 已配置，重新打开终端或执行 `source ~/.bashrc` 生效")
	fmt.Println()
	return nil
}

func disablePlugin() error {
	shell := os.Getenv("SHELL")
	home := os.Getenv("HOME")
	if home == "" {
		home = "/root"
	}

	if strings.Contains(shell, "zsh") {
		// Remove from .zshrc
		zshrc := home + "/.zshrc"
		if raw, err := os.ReadFile(zshrc); err == nil {
			lines := strings.Split(string(raw), "\n")
			var kept []string
			skip := false
			for _, l := range lines {
				if strings.Contains(l, "command_not_found_handler") {
					skip = true
					continue
				}
				if skip && strings.Contains(l, "}") {
					skip = false
					continue
				}
				if !skip {
					kept = append(kept, l)
				}
			}
			_ = os.WriteFile(zshrc, []byte(strings.Join(kept, "\n")), 0644)
		}
		// Remove omz plugin
		omz := os.Getenv("ZSH")
		if omz != "" {
			pluginDir := omz + "/custom/plugins/fastpve"
			_ = os.RemoveAll(pluginDir)
		}
		fmt.Println("zsh 插件已移除")
	} else {
		bashrc := home + "/.bashrc"
		if raw, err := os.ReadFile(bashrc); err == nil {
			lines := strings.Split(string(raw), "\n")
			var kept []string
			skip := false
			for _, l := range lines {
				if strings.Contains(l, "command_not_found_handle") {
					skip = true
					continue
				}
				if skip && strings.Contains(l, "}") {
					skip = false
					continue
				}
				if !skip {
					kept = append(kept, l)
				}
			}
			_ = os.WriteFile(bashrc, []byte(strings.Join(kept, "\n")), 0644)
		}
		fmt.Println("bash 插件已移除")
	}
	return nil
}

// plugin install script (for zsh)
const zshPluginCode = `# FastPVE AI 助手 — zsh 插件
# 命令不存在时自动转 fastpve ai 处理
command_not_found_handler() {
  fastpve ai "$*"
  return 0
}

# 强制触发命令（跳过 AI 拦截）
alias forcetype='\'
`
