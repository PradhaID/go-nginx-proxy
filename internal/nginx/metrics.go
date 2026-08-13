package nginx

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	CpuPercent float64 `json:"cpuPercent"`
	MemKB      int64   `json:"memKB"`
	MemPercent float64 `json:"memPercent"`
	MemTotalKB int64   `json:"memTotalKB"`
	Pid        int     `json:"pid"`
	Procs      int     `json:"procs"`
}

type procSample struct {
	jiffies uint64
	rssKB   int64
	at      time.Time
}

// ticksPerSecond: jiffies per second on Linux (CLK_TCK, 100 on x86).
const ticksPerSecond = 100

type metricsCollector struct {
	mu   sync.Mutex
	last *procSample
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{}
}

// collect returns CPU (delta-based) and memory (RSS) usage for all nginx
// processes. Non-Linux platforms return zeroed metrics.
func (c *metricsCollector) collect() Metrics {
	m := Metrics{}
	if runtime.GOOS != "linux" {
		return m
	}
	if _, err := os.Stat("/proc"); err != nil {
		return m
	}

	var totalJiffies uint64
	var totalRSS int64
	var count int
	var pid int

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return m
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		dir := filepath.Join("/proc", e.Name())
		comm, err := os.ReadFile(filepath.Join(dir, "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) != "nginx" {
			continue
		}
		count++
		if pid == 0 {
			pid, _ = strconv.Atoi(e.Name())
		}
		if data, err := os.ReadFile(filepath.Join(dir, "stat")); err == nil {
			if utime, stime, ok := parseProcStat(data); ok {
				totalJiffies += utime + stime
			}
		}
		if data, err := os.ReadFile(filepath.Join(dir, "status")); err == nil {
			totalRSS += parseVmRSS(data)
		}
	}

	m.Pid = pid
	m.Procs = count
	m.MemKB = totalRSS
	if total := memTotalKB(); total > 0 {
		m.MemTotalKB = total
		m.MemPercent = float64(totalRSS) / float64(total) * 100
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.last != nil {
		dj := int64(totalJiffies) - int64(c.last.jiffies)
		dt := time.Since(c.last.at).Seconds()
		if dt > 0 {
			cpu := float64(dj) / dt / ticksPerSecond * 100
			if cpu < 0 {
				cpu = 0
			}
			m.CpuPercent = cpu
		}
	}
	c.last = &procSample{jiffies: totalJiffies, rssKB: totalRSS, at: time.Now()}
	return m
}

// parseProcStat parses utime (field 14) and stime (field 15) from
// /proc/<pid>/stat. The process name (field 2) is wrapped in parens and may
// contain spaces, so we start parsing after the last ')'.
func parseProcStat(data []byte) (utime, stime uint64, ok bool) {
	closeParen := bytes.LastIndexByte(data, ')')
	if closeParen < 0 {
		return
	}
	fields := strings.Fields(string(data[closeParen+1:]))
	// After the ')': fields[0] == stat field 3 (state).
	// utime == field 14 -> index 11, stime == field 15 -> index 12.
	if len(fields) < 13 {
		return
	}
	u, err1 := strconv.ParseUint(fields[11], 10, 64)
	st, err2 := strconv.ParseUint(fields[12], 10, 64)
	if err1 != nil || err2 != nil {
		return
	}
	return u, st, true
}

func parseVmRSS(data []byte) int64 {
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseInt(fields[1], 10, 64)
				return v // kB
			}
		}
	}
	return 0
}

func memTotalKB() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseInt(fields[1], 10, 64)
				return v
			}
		}
	}
	return 0
}
