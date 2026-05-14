package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

type providerDef struct {
	label string
	url   string
	model string
}

var aiProviders = []providerDef{
	{"DeepSeek（默认，推荐）", "https://api.deepseek.com/v1/chat/completions", "deepseek-chat"},
	{"OpenAI", "https://api.openai.com/v1/chat/completions", "gpt-4o"},
	{"Claude（OpenAI 兼容）", "https://api.anthropic.com/v1/chat/completions", "claude-sonnet-4-20250514"},
	{"Ollama（本地）", "http://localhost:11434/v1/chat/completions", "llama3"},
	{"自定义", "", ""},
}

func setupWizard() error {
	fmt.Println(`
╔══════════════════════════════════════════╗
║        FastPVE — PVE AI 运维助手         ║
║    先配置 AI 密钥，开启智能运维体验       ║
╚══════════════════════════════════════════╝
	`)

	// 1. 选择 AI 提供商
	labels := make([]string, len(aiProviders))
	for i, p := range aiProviders {
		labels[i] = p.label
	}
	prompt := promptui.Select{
		Label: "选择 AI 提供商",
		Items: labels,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return nil
	}
	prov := aiProviders[idx]
	fmt.Println()

	// 2. API Key
	key := ""
	if idx < 4 { // 非自定义有默认值，但仍然需要输入 key（Ollama 除外）
		if idx == 3 {
			// Ollama: key 可选
			keyPrompt := promptui.Prompt{
				Label:   "API 密钥（Ollama 可留空）",
				Mask:    '*',
				Default: "ollama",
			}
			key, _ = keyPrompt.Run()
		} else {
			for key == "" {
				keyPrompt := promptui.Prompt{
					Label: "API 密钥",
					Mask:  '*',
				}
				key, _ = keyPrompt.Run()
				key = strings.TrimSpace(key)
			}
		}
	} else {
		// 自定义：全部手动输入
		for key == "" {
			keyPrompt := promptui.Prompt{
				Label: "API 密钥",
				Mask:  '*',
			}
			key, _ = keyPrompt.Run()
			key = strings.TrimSpace(key)
		}
	}
	fmt.Println()

	// 3. API URL
	urlPrompt := promptui.Prompt{
		Label:   "API 地址",
		Default: prov.url,
	}
	url, _ := urlPrompt.Run()
	url = strings.TrimSpace(url)
	if url == "" {
		url = prov.url
	}

	// 4. 模型名
	modelPrompt := promptui.Prompt{
		Label:   "模型名",
		Default: prov.model,
	}
	model, _ := modelPrompt.Run()
	model = strings.TrimSpace(model)
	if model == "" {
		model = prov.model
	}

	// 5. 写入配置
	if err := writeConfig(url, key, model); err != nil {
		fmt.Printf("写入配置失败: %v\n", err)
		return nil
	}

	// 6. 重新加载配置
	InitConfig()

	fmt.Println("✅ 配置已保存到 /etc/fastpve/config.conf")
	fmt.Println()

	// 7. 插件安装
	pluginPrompt := promptui.Select{
		Label: "安装终端无感插件？输入自然语言自动转 AI",
		Items: []string{"zsh 插件（推荐）", "bash 兼容", "跳过"},
	}
	pi, _, _ := pluginPrompt.Run()
	if pi < 2 {
		installPlugin()
	}

	fmt.Println()
	fmt.Println("✅ 配置完成！试试以下命令：")
	fmt.Println()
	fmt.Println("  fastpve          — AI 交互会话")
	fmt.Println("  fastpve menu     — 传统 TUI 菜单")
	fmt.Println("  fastpve ai xxx   — 一句话查询")
	fmt.Println()
	return nil
}

func writeConfig(apiURL, apiKey, model string) error {
	raw, _ := os.ReadFile(configPath)
	lines := strings.Split(string(raw), "\n")

	type pair struct{ key, val string }
	replacements := []pair{{"AI_URL=", apiURL}, {"AI_KEY=", apiKey}, {"AI_MODEL=", model}}
	found := 0

	out := make([]string, 0, len(lines))
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		matched := false
		for i, p := range replacements {
			// Match both "AI_XXX=..." and "# AI_XXX=..."
			base := strings.TrimPrefix(trim, "# ")
			if strings.HasPrefix(base, p.key) {
				out = append(out, p.key+p.val)
				found |= 1 << i
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, l)
		}
	}
	if found != 7 { // missing some lines — append
		out = append(out, "", "# AI 配置（由 setup wizard 生成）")
		for _, p := range replacements {
			out = append(out, p.key+p.val)
		}
	}
	return os.WriteFile(configPath, []byte(strings.Join(out, "\n")), 0644)
}

// runInteractiveAI starts an interactive AI session in the terminal.
func runInteractiveAI() error {
	if Cfg == nil || Cfg.AIKey == "" {
		return setupWizard()
	}

	fmt.Println("\n🤖 FastPVE AI 助手已就绪（输入 /help 查看命令，/menu 进入传统菜单）")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("你: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "/exit", "/quit", "exit", "quit", "退出":
			fmt.Println("再见！")
			return nil
		case "/menu":
			return mainPrompt()
		case "/help":
			printAIHelp()
			continue
		case "/clear":
			if db != nil {
				_ = dbClearHistory()
			}
			fmt.Println("对话历史已清空")
			continue
		}

		runAI(input)
		fmt.Println()
	}
	return nil
}

func printAIHelp() {
	fmt.Println(`
  /menu   — 传统 TUI 菜单
  /clear  — 清空对话历史
  /exit   — 退出
  直接输入需求即可，例如：
    「查看系统状态」
    「帮我更新系统」
    「最近性能有异常吗？」
	`)
}
