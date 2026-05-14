package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/solider245/fastpve/utils"
)

// -------- PI 协议 ----------

type toolResult struct {
	OK    bool   `json:"ok"`
	Data  string `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// runPI implements the PI protocol: read JSON from stdin, execute, write JSON to stdout.
func runPI() error {
	var req struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		result := toolResult{Error: fmt.Sprintf("解析输入失败: %v", err)}
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	result := executeTool(req.Tool, req.Args)
	return json.NewEncoder(os.Stdout).Encode(result)
}

func executeTool(tool string, args map[string]any) toolResult {
	ctx := context.TODO()
	var result toolResult

	switch tool {
	case "health_check":
		report := collectHealthReport(ctx)
		result = toolResult{OK: true, Data: report}
	case "storage_overview":
		zpoolOut, _ := utils.BatchOutput(ctx, []string{"zpool list 2>/dev/null || echo '无 ZFS 池'"}, 5)
		dfOut, _ := utils.BatchOutput(ctx, []string{"df -h / 2>/dev/null"}, 5)
		result = toolResult{OK: true, Data: fmt.Sprintf("ZFS: %s\n磁盘: %s",
			strings.TrimSpace(string(zpoolOut)), strings.TrimSpace(string(dfOut)))}
	case "vm_list":
		out, err := utils.BatchOutput(ctx, []string{"qm list 2>/dev/null || true"}, 5)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: strings.TrimSpace(string(out))}
		}
	case "lxc_list":
		out, err := utils.BatchOutput(ctx, []string{"pct list 2>/dev/null || true"}, 5)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: strings.TrimSpace(string(out))}
		}
	case "backup_network":
		ts := time.Now().Format("20060102_1504")
		err := utils.BatchRun(ctx, []string{fmt.Sprintf("cp /etc/network/interfaces /etc/network/interfaces.bak.%s", ts)}, 5)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: fmt.Sprintf("已备份到 /etc/network/interfaces.bak.%s", ts)}
		}
	case "remove_sub_nag":
		err := utils.BatchRun(ctx, []string{
			`sed -Ezi.bak "s/(Ext.Msg.show\(\{\s+title: gettext\('No valid sub)/void\(\{ \/\/\1/g" /usr/share/javascript/proxmox-widget-toolkit/proxmoxlib.js`,
			"systemctl restart pveproxy.service",
		}, 10)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: "订阅弹窗已去除"}
		}
	case "system_update":
		err := utils.BatchRun(ctx, []string{"apt update && apt dist-upgrade -y"}, 300)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: "系统已更新"}
		}
	case "install_tools":
		err := utils.BatchRun(ctx, []string{"apt install -y -qq curl wget vim htop net-tools lsof"}, 120)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: "常用工具已安装"}
		}
	case "query_perf_history":
		timespan := "24h"
		if v, ok := args["timespan"]; ok {
			if s, ok := v.(string); ok {
				timespan = s
			}
		}
		data, err := dbQueryPerfHistory(timespan)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: data}
		}
	case "query_audit_log":
		var filter string
		timespan := "24h"
		if v, ok := args["filter"]; ok {
			if s, ok := v.(string); ok {
				filter = s
			}
		}
		if v, ok := args["timespan"]; ok {
			if s, ok := v.(string); ok {
				timespan = s
			}
		}
		data, err := dbQueryAuditLog(filter, timespan)
		if err != nil {
			result = toolResult{Error: err.Error()}
		} else {
			result = toolResult{OK: true, Data: data}
		}
	default:
		result = toolResult{Error: fmt.Sprintf("未知工具: %s", tool)}
	}

	// 审计日志
	auditSource := "ai"
	msg := result.Data
	if msg == "" {
		msg = result.Error
	}
	logAudit(auditSource, tool, args, result.OK, msg)

	return result
}

// -------- AI Agent ----------

type toolCall struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
	Why  string         `json:"why"`
}

// runAI reads user input (CLI arg or stdin), calls LLM, and handles tool calls.
func runAI(input string) error {
	if input == "" {
		data, _ := io.ReadAll(os.Stdin)
		input = strings.TrimSpace(string(data))
	}
	if input == "" {
		fmt.Println("请输入问题")
		return nil
	}
	if Cfg == nil || Cfg.AIKey == "" {
		fmt.Println("请先在 /etc/fastpve/config.conf 中配置 AI_KEY")
		return nil
	}

	// Build message list with system prompt, optional history, and user input
	messages := []aiMessage{
		{Role: "system", Content: systemPrompt()},
	}
	if db != nil {
		history, _ := dbGetHistory(10)
		messages = append(messages, history...)
	}
	messages = append(messages, aiMessage{Role: "user", Content: input})

	// Save user message to history
	if db != nil {
		_ = dbSaveMessage("user", input)
	}

	// Call LLM
	reply, err := callAI(messages)
	if err != nil {
		fmt.Println("AI 请求失败:", err)
		return nil
	}

	// Try to parse as JSON tool call
	var tc toolCall
	if json.Unmarshal([]byte(reply), &tc) == nil && tc.Tool != "" {
		handleToolCall(tc)
	} else {
		// Plain text response
		fmt.Println(reply)
		if db != nil {
			_ = dbSaveMessage("assistant", reply)
		}
	}
	return nil
}

func handleToolCall(tc toolCall) {
	if tc.Why != "" {
		fmt.Printf("🤖 %s\n\n", tc.Why)
	}

	if toolNeedsConfirm(tc.Tool) {
		fmt.Printf("即将执行: %s\n", toolDescription(tc.Tool))
		fmt.Print("确认执行? (y/N): ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			fmt.Println("已取消")
			return
		}
	}

	result := executeTool(tc.Tool, tc.Args)
	if result.OK {
		fmt.Println("✅", result.Data)
	} else {
		fmt.Println("❌", result.Error)
	}
}

func toolNeedsConfirm(tool string) bool {
	readTools := map[string]bool{
		"health_check":       true,
		"storage_overview":   true,
		"vm_list":            true,
		"lxc_list":           true,
		"query_perf_history": true,
		"query_audit_log":    true,
	}
	return !readTools[tool]
}

func toolDescription(tool string) string {
	desc := map[string]string{
		"health_check":       "查看 PVE 系统健康状态",
		"storage_overview":   "查看存储概览",
		"vm_list":            "列出所有虚拟机",
		"lxc_list":           "列出所有 LXC 容器",
		"backup_network":     "备份网络配置",
		"remove_sub_nag":     "去除 PVE 订阅弹窗",
		"system_update":      "更新系统",
		"install_tools":      "安装常用工具",
		"query_perf_history": "查询性能趋势历史",
		"query_audit_log":    "查询操作审计日志",
	}
	if d, ok := desc[tool]; ok {
		return d
	}
	return tool
}

func systemPrompt() string {
	return `你是一个 Proxmox VE (PVE) 运维助手。你的特点是:
1. 回答简短直接，用中文
2. 当用户需要执行操作时，返回 JSON 格式的工具调用

**可用工具列表（用 JSON 格式返回工具调用）：**
- health_check — 查看 PVE 系统健康状态（CPU、内存、磁盘、VM/CT 数量）
- storage_overview — 查看存储概览（ZFS、磁盘使用）
- vm_list — 列出所有虚拟机
- lxc_list — 列出所有 LXC 容器
- query_perf_history — 查询性能历史趋势，参数 timespan: 1h/6h/24h/7d/30d
- query_audit_log — 查询操作审计日志，参数 filter: 工具名, timespan: 时间范围
- backup_network — 备份 /etc/network/interfaces 网络配置
- remove_sub_nag — 去除 PVE Web UI 订阅弹窗
- system_update — 运行 apt update && apt dist-upgrade 更新系统
- install_tools — 安装常用工具（curl、wget、vim、htop 等）

**工具调用格式：**
{"tool":"工具名","args":{},"why":"为什么要执行这个操作"}

**规则：**
- 用户闲聊（打招呼、问你是谁）→ 直接回复文字，不调工具
- 用户问状态/查询 → 调对应的读工具，显示结果
- 用户要执行操作（安装、备份、更新等）→ 返回工具调用 JSON
- 如果用户需求不明确，先文字询问澄清，不擅自调工具`
}
