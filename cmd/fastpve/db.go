package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

const dbPath = "/var/lib/fastpve/fastpve.db"

func InitDB() error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS messages (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            role TEXT NOT NULL,
            content TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS config (
            key TEXT PRIMARY KEY,
            value TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS perf_history (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            cpu_load_1m REAL,
            cpu_load_5m REAL,
            cpu_load_15m REAL,
            mem_used_mb INTEGER,
            mem_total_mb INTEGER,
            disk_root_used_pct REAL,
            disk_vz_used_pct REAL,
            vm_running INTEGER,
            vm_total INTEGER,
            ct_running INTEGER,
            ct_total INTEGER,
            zfs_arc_size_mb INTEGER,
            zfs_arc_max_mb INTEGER
        )`,
		`CREATE TABLE IF NOT EXISTS audit_log (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            source TEXT NOT NULL DEFAULT 'ai',
            tool TEXT NOT NULL,
            args TEXT,
            ok BOOLEAN NOT NULL,
            result TEXT
        )`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("创建表失败: %w", err)
		}
	}
	return nil
}

// ==================== 消息历史 ====================

func dbSaveMessage(role, content string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec("INSERT INTO messages (role, content, created_at) VALUES (?, ?, ?)", role, content, time.Now())
	return err
}

func dbGetHistory(limit int) ([]aiMessage, error) {
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query("SELECT role, content FROM messages ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []aiMessage
	for rows.Next() {
		var m aiMessage
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			continue
		}
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func dbClearHistory() error {
	if db == nil {
		return nil
	}
	_, err := db.Exec("DELETE FROM messages")
	return err
}

// ==================== 配置存储 ====================

func dbGetConfig(key string) (string, error) {
	if db == nil {
		return "", nil
	}
	var val string
	err := db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func dbSetConfig(key, value string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, value)
	return err
}

// ==================== 性能历史 ====================

type perfSnapshot struct {
	CPU1m, CPU5m, CPU15m   float64
	MemUsedMB, MemTotalMB  int
	DiskRootPct, DiskVzPct float64
	VMRunning, VMTotal     int
	CTRunning, CTTotal     int
	ArcSizeMB, ArcMaxMB    int
}

func dbSavePerfSnapshot(s perfSnapshot) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`INSERT INTO perf_history
		(cpu_load_1m, cpu_load_5m, cpu_load_15m, mem_used_mb, mem_total_mb,
		 disk_root_used_pct, disk_vz_used_pct, vm_running, vm_total,
		 ct_running, ct_total, zfs_arc_size_mb, zfs_arc_max_mb)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.CPU1m, s.CPU5m, s.CPU15m,
		s.MemUsedMB, s.MemTotalMB,
		s.DiskRootPct, s.DiskVzPct,
		s.VMRunning, s.VMTotal,
		s.CTRunning, s.CTTotal,
		s.ArcSizeMB, s.ArcMaxMB)
	if err != nil {
		return err
	}
	// 保留 7 天，清理旧数据
	_, _ = db.Exec("DELETE FROM perf_history WHERE collected_at < datetime('now', '-7 days')")
	return nil
}

func dbQueryPerfHistory(timespan string) (string, error) {
	if db == nil {
		return "数据库未初始化", nil
	}
	span := parseTimespan(timespan)

	row := db.QueryRow(fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(MIN(cpu_load_1m),0), COALESCE(MAX(cpu_load_1m),0),
			COALESCE(ROUND(AVG(cpu_load_1m),2),0),
			COALESCE(MIN(mem_used_pct),0), COALESCE(MAX(mem_used_pct),0),
			COALESCE(ROUND(AVG(mem_used_pct)),0),
			COALESCE(MIN(disk_root_used_pct),0), COALESCE(MAX(disk_root_used_pct),0),
			COALESCE(MIN(vm_running),0), COALESCE(MAX(vm_running),0),
			COALESCE(MIN(ct_running),0), COALESCE(MAX(ct_running),0)
		FROM (
			SELECT *, ROUND(CAST(mem_used_mb AS REAL)/NULLIF(mem_total_mb,1)*100) AS mem_used_pct
			FROM perf_history
			WHERE collected_at >= datetime('now', '%s')
		)
	`, span))

	var (
		count                  int
		cpuMin, cpuMax, cpuAvg float64
		memMin, memMax, memAvg float64
		diskMin, diskMax       float64
		vmMin, vmMax           int
		ctMin, ctMax           int
	)
	if err := row.Scan(&count, &cpuMin, &cpuMax, &cpuAvg, &memMin, &memMax, &memAvg,
		&diskMin, &diskMax, &vmMin, &vmMax, &ctMin, &ctMax); err != nil {
		return "", err
	}

	if count == 0 {
		return fmt.Sprintf("过去 %s 无性能数据（采集协程尚未运行）", timespan), nil
	}

	// 获取最新一条
	var curCPU, curMem, curDisk float64
	_ = db.QueryRow(fmt.Sprintf(`
		SELECT cpu_load_1m, ROUND(CAST(mem_used_mb AS REAL)/NULLIF(mem_total_mb,1)*100),
		       disk_root_used_pct FROM perf_history
		WHERE collected_at >= datetime('now', '%s')
		ORDER BY collected_at DESC LIMIT 1
	`, span)).Scan(&curCPU, &curMem, &curDisk)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== 性能趋势分析（过去 %s）===\n", timespan))
	b.WriteString(fmt.Sprintf("CPU 负载: 最低 %.2f, 最高 %.2f, 平均 %.2f, 当前 %.2f",
		cpuMin, cpuMax, cpuAvg, curCPU))
	if cpuMax > 4.0 {
		b.WriteString(" ⚠️ 峰值过高\n")
	} else {
		b.WriteString(" ✅ 正常\n")
	}

	b.WriteString(fmt.Sprintf("内存: 最低 %.0f%%, 最高 %.0f%%, 平均 %.0f%%, 当前 %.0f%%",
		memMin, memMax, memAvg, curMem))
	if memMax > 90 {
		b.WriteString(" ⚠️ 峰值过高\n")
	} else {
		b.WriteString(" ✅ 正常\n")
	}

	b.WriteString(fmt.Sprintf("磁盘 /: 最低 %.0f%%, 最高 %.0f%%, 当前 %.0f%%",
		diskMin, diskMax, curDisk))
	if diskMax > 90 {
		b.WriteString(" ⚠️ 接近满\n")
	} else {
		b.WriteString(" ✅ 正常\n")
	}

	b.WriteString(fmt.Sprintf("VM 运行数: 范围 %d-%d\n", vmMin, vmMax))
	b.WriteString(fmt.Sprintf("CT 运行数: 范围 %d-%d\n", ctMin, ctMax))
	b.WriteString(fmt.Sprintf("采样点数: %d\n", count))
	return b.String(), nil
}

func parseTimespan(s string) string {
	switch s {
	case "1h":
		return "-1 hours"
	case "6h":
		return "-6 hours"
	case "12h":
		return "-12 hours"
	case "24h":
		return "-24 hours"
	case "7d":
		return "-7 days"
	case "30d":
		return "-30 days"
	default:
		return "-24 hours"
	}
}

// ==================== 审计日志 ====================

func logAudit(source, tool string, args map[string]any, ok bool, result string) {
	if db == nil {
		return
	}
	argsStr := ""
	if len(args) > 0 {
		argsStr = fmt.Sprintf("%v", args)
	}
	// 截断过长结果
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	_, _ = db.Exec("INSERT INTO audit_log (source, tool, args, ok, result) VALUES (?, ?, ?, ?, ?)",
		source, tool, argsStr, ok, result)
}

func dbQueryAuditLog(filter, timespan string) (string, error) {
	if db == nil {
		return "数据库未初始化", nil
	}
	span := parseTimespan(timespan)

	var rows *sql.Rows
	var err error
	if filter != "" {
		rows, err = db.Query(fmt.Sprintf(`
			SELECT created_at, source, tool, args, ok, result
			FROM audit_log
			WHERE tool = ? AND created_at >= datetime('now', '%s')
			ORDER BY created_at DESC LIMIT 20
		`, span), filter)
	} else {
		rows, err = db.Query(fmt.Sprintf(`
			SELECT created_at, source, tool, args, ok, result
			FROM audit_log
			WHERE created_at >= datetime('now', '%s')
			ORDER BY created_at DESC LIMIT 20
		`, span))
	}
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== 操作审计（过去 %s）===\n", timespan))
	if filter != "" {
		b.WriteString(fmt.Sprintf("过滤工具: %s\n\n", filter))
	}

	count := 0
	for rows.Next() {
		var ts, source, tool, argsStr, result string
		var ok bool
		if err := rows.Scan(&ts, &source, &tool, &argsStr, &ok, &result); err != nil {
			continue
		}
		status := "✅"
		if !ok {
			status = "❌"
		}
		// 时间戳截取到秒
		ts = ts[:19]
		b.WriteString(fmt.Sprintf("%s [%s] %s %s", ts, source, status, tool))
		if argsStr != "" {
			b.WriteString(fmt.Sprintf(" %s", argsStr))
		}
		b.WriteString("\n")
		count++
	}

	if count == 0 {
		b.WriteString("无操作记录\n")
	}
	return b.String(), nil
}
