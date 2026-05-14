package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

	// 创建表
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
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("创建表失败: %w", err)
		}
	}
	return nil
}

// 消息历史
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

	// 反转顺序（从旧到新）
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
	// 反转
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

// 配置存储
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
