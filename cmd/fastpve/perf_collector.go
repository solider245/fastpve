package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/solider245/fastpve/utils"
)

// startPerfCollector launches a background goroutine that collects system
// performance metrics every `interval` and stores them in SQLite.
func startPerfCollector(interval time.Duration) {
	if db == nil {
		return
	}
	go func() {
		// Collect once immediately, then on ticker
		collectAndStorePerf()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			collectAndStorePerf()
		}
	}()
}

func collectAndStorePerf() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var s perfSnapshot

	// CPU load (loadavg: 1m 5m 15m)
	if out, err := utils.BatchOutput(ctx, []string{
		`cat /proc/loadavg | awk '{printf "%s %s %s", $1, $2, $3}'`,
	}, 5); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) == 3 {
			s.CPU1m, _ = strconv.ParseFloat(parts[0], 64)
			s.CPU5m, _ = strconv.ParseFloat(parts[1], 64)
			s.CPU15m, _ = strconv.ParseFloat(parts[2], 64)
		}
	}

	// Memory (used_mb, total_mb)
	if out, err := utils.BatchOutput(ctx, []string{
		`free -m | awk '/Mem/{printf "%d %d", $3, $2}'`,
	}, 5); err == nil {
		fmt.Sscanf(string(out), "%d %d", &s.MemUsedMB, &s.MemTotalMB)
	}

	// Disk root & vz usage %
	if out, err := utils.BatchOutput(ctx, []string{
		`df -h / | awk 'NR==2{printf "%d", $5+0}'`,
	}, 5); err == nil {
		s.DiskRootPct, _ = strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	}
	if out, err := utils.BatchOutput(ctx, []string{
		`df -h /var/lib/vz 2>/dev/null | awk 'NR==2{printf "%d", $5+0}' || echo "0"`,
	}, 5); err == nil {
		s.DiskVzPct, _ = strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	}

	// VM counts (running, total)
	if out, err := utils.BatchOutput(ctx, []string{
		`qm list 2>/dev/null | awk 'NR>1{print $2}' | sort | uniq -c | awk '{running+=$1} END{printf "%d %d", running, NR}' || echo "0 0"`,
	}, 5); err == nil {
		fmt.Sscanf(string(out), "%d %d", &s.VMRunning, &s.VMTotal)
	}

	// CT counts
	if out, err := utils.BatchOutput(ctx, []string{
		`pct list 2>/dev/null | awk 'NR>1{print $2}' | sort | uniq -c | awk '{running+=$1} END{printf "%d %d", running, NR}' || echo "0 0"`,
	}, 5); err == nil {
		fmt.Sscanf(string(out), "%d %d", &s.CTRunning, &s.CTTotal)
	}

	// ZFS ARC size
	if out, err := utils.BatchOutput(ctx, []string{
		`cat /proc/spl/kstat/zfs/arcstats 2>/dev/null | grep -E '^(size|c_max) ' | awk '{print $3/1024/1024}' | tr '\n' ' ' || echo "0 0"`,
	}, 5); err == nil {
		fmt.Sscanf(string(out), "%d %d", &s.ArcSizeMB, &s.ArcMaxMB)
	}

	_ = dbSavePerfSnapshot(s)
}
