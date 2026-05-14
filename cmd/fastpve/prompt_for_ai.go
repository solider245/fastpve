package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/solider245/fastpve/utils"
)

type aiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var aiHistory []aiMessage

func promptAI() error {
	items := []menuItem{
		{"AI 对话", aiChat},
		{"分析系统健康", aiAnalyzeHealth},
		{"分析日志", aiAnalyzeLog},
		{"清空对话历史", aiClearHistory},
		{"返回", func() error { return errContinue }},
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = fmt.Sprintf("%d、%s", i, it.label)
	}
	prompt := promptui.Select{
		Label: "AI 助手",
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

func aiChat() error {
	fmt.Println("")
	fmt.Println("========== AI 对话 ==========")
	fmt.Println("输入 exit / quit / 退出 返回")
	fmt.Println("==============================")

	if len(aiHistory) == 0 {
		aiHistory = append(aiHistory, aiMessage{
			Role:    "system",
			Content: "你是一个 Proxmox VE 运维助手。用户正在使用 FastPVE 工具管理 PVE 服务器。请用中文简洁回答，给出可直接执行的命令。",
		})
	}

	for {
		prompt := promptui.Prompt{
			Label: "你",
		}
		input, err := prompt.Run()
		if err != nil {
			return nil
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		lower := strings.ToLower(input)
		if lower == "exit" || lower == "quit" || lower == "退出" {
			break
		}

		aiHistory = append(aiHistory, aiMessage{Role: "user", Content: input})
		reply, err := callAI(aiHistory)
		if err != nil {
			fmt.Printf("AI 请求失败: %v\n", err)
			continue
		}
		aiHistory = append(aiHistory, aiMessage{Role: "assistant", Content: reply})
		fmt.Printf("\nAI: %s\n\n", reply)
	}
	return nil
}

func aiAnalyzeHealth() error {
	report := collectHealthReport(context.TODO())

	fmt.Println("正在分析系统状态...")
	messages := []aiMessage{
		{Role: "system", Content: "你是一个 Proxmox VE 运维专家。请分析以下系统状态报告，给出优化建议和潜在问题。用中文简洁回答。"},
		{Role: "user", Content: report},
	}
	reply, err := callAI(messages)
	if err != nil {
		return err
	}
	fmt.Printf("\nAI 分析结果:\n%s\n\n", reply)
	return nil
}

func aiAnalyzeLog() error {
	prompt := promptui.Prompt{
		Label: "粘贴日志内容（或输入日志文件路径）",
	}
	input, err := prompt.Run()
	if err != nil {
		return errContinue
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	logContent := input
	// If input looks like a file path, read it
	if strings.Contains(input, "/") || strings.Contains(input, "\\") {
		if data, err := os.ReadFile(input); err == nil {
			logContent = string(data)
		}
		// If read fails, treat as raw text
	}

	fmt.Println("正在分析日志...")
	messages := []aiMessage{
		{Role: "system", Content: "你是一个 Proxmox VE 运维专家。请分析以下日志，找出异常原因和修复步骤。用中文简洁回答。"},
		{Role: "user", Content: logContent},
	}
	reply, err := callAI(messages)
	if err != nil {
		return err
	}
	fmt.Printf("\nAI 分析结果:\n%s\n\n", reply)
	return nil
}

func aiClearHistory() error {
	aiHistory = nil
	fmt.Println("对话历史已清空")
	return nil
}

func callAI(messages []aiMessage) (string, error) {
	payload := struct {
		Model       string      `json:"model"`
		Messages    []aiMessage `json:"messages"`
		Temperature float64     `json:"temperature"`
	}{
		Model:       Cfg.AIModel,
		Messages:    messages,
		Temperature: 0.7,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Temp file avoids shell quoting issues
	tmpFile := "/tmp/fastpve-ai-payload.json"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	// Ensure jq is available
	_ = utils.BatchRun(context.TODO(), []string{
		"command -v jq >/dev/null 2>&1 || apt-get install -y -qq jq >/dev/null 2>&1",
	}, 30)

	cmd := fmt.Sprintf(
		`curl -s -X POST "%s" -H "Content-Type: application/json" -H "Authorization: Bearer %s" -d @%s 2>/dev/null | jq -r '.choices[0].message.content // (.error.message // "API_ERROR")' 2>/dev/null || echo 'REQUEST_FAILED'`,
		Cfg.AIURL, Cfg.AIKey, tmpFile,
	)

	out, err := utils.BatchOutput(context.TODO(), []string{cmd}, 60)
	result := strings.TrimSpace(string(out))

	if err != nil || result == "" || result == "REQUEST_FAILED" {
		if result == "API_ERROR" || result == "REQUEST_FAILED" {
			return "", fmt.Errorf("AI 返回错误，请检查 API 地址和密钥是否正确")
		}
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}
	return result, nil
}
