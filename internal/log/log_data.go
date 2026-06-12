package log

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Line 一行日志
type Line struct {
	Raw   string
	Level string
}

// LoadMsg 从文件或 stdin 加载完成后的消息
type LoadMsg struct {
	Lines []Line
	Err   error
}

// DirMsg 目录扫描结果
type DirMsg struct {
	Dir   string
	Files []string
}

// tailTickMsg tail -f 定时检查消息
type tailTickMsg struct{}

// tailTickCmd 延迟 duration 后发送 tailTickMsg（链式调用实现周期监听）
func tailTickCmd(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return tailTickMsg{}
	})
}

// LoadFromStdin 从 stdin 读取日志（pipe 场景）
func LoadFromStdin() tea.Msg {
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		var lines []Line
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			raw := scanner.Text()
			lines = append(lines, Line{Raw: raw, Level: detectLevel(raw)})
		}
		return LoadMsg{Lines: lines}
	}
	return LoadMsg{Lines: nil}
}

// LoadFromFile 从文件读取日志
func LoadFromFile(path string) tea.Msg {
	f, err := os.Open(path)
	if err != nil {
		return LoadMsg{Err: fmt.Errorf("cannot open: %s (%v)", path, err)}
	}
	defer f.Close()
	var lines []Line
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		lines = append(lines, Line{Raw: raw, Level: detectLevel(raw)})
	}
	if err := scanner.Err(); err != nil {
		return LoadMsg{Lines: lines, Err: fmt.Errorf("read error: %v", err)}
	}
	return LoadMsg{Lines: lines}
}

// ScanDir 扫描目录下所有 .log 文件
func ScanDir(dir string) tea.Msg {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirMsg{Dir: dir, Files: nil}
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ".log" {
			files = append(files, e.Name())
		}
	}
	return DirMsg{Dir: dir, Files: files}
}

// LoadFromFileCmd 返回一个执行 LoadFromFile 的 Cmd
func LoadFromFileCmd(path string) tea.Cmd {
	return func() tea.Msg { return LoadFromFile(path) }
}

// ScanDirCmd 返回一个执行 ScanDir 的 Cmd
func ScanDirCmd(dir string) tea.Cmd {
	return func() tea.Msg { return ScanDir(dir) }
}

func detectLevel(s string) string {
	upper := strings.ToUpper(s)
	switch {
	case strings.Contains(upper, "ERROR") || strings.Contains(upper, "FATAL") || strings.Contains(upper, "PANIC"):
		return "ERROR"
	case strings.Contains(upper, "WARN"):
		return "WARN"
	case strings.Contains(upper, "INFO"):
		return "INFO"
	case strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE"):
		return "DEBUG"
	default:
		return "OTHER"
	}
}

// SampleLogLines 无数据时显示的示例日志
func SampleLogLines() []Line {
	return []Line{
		{"2024-01-15 10:23:01 INFO  server started on :8080", "INFO"},
		{"2024-01-15 10:23:05 INFO  connected to database", "INFO"},
		{"2024-01-15 10:23:08 WARN  slow query detected (2300ms)", "WARN"},
		{"2024-01-15 10:23:10 INFO  GET /api/users 200 45ms", "INFO"},
		{"2024-01-15 10:23:12 ERROR connection refused: redis://localhost:6379", "ERROR"},
		{"2024-01-15 10:23:13 INFO  retrying connection...", "INFO"},
		{"2024-01-15 10:23:14 INFO  connected to redis", "INFO"},
		{"2024-01-15 10:23:15 DEBUG cache hit for user:123", "DEBUG"},
		{"2024-01-15 10:23:16 INFO  POST /api/login 200 120ms", "INFO"},
		{"2024-01-15 10:23:18 WARN  rate limit approaching: 90/100", "WARN"},
		{"2024-01-15 10:23:20 ERROR failed to send email: timeout", "ERROR"},
		{"2024-01-15 10:23:21 INFO  GET /api/dashboard 200 350ms", "INFO"},
		{"2024-01-15 10:23:22 DEBUG query plan: seq scan on users", "DEBUG"},
		{"2024-01-15 10:23:25 INFO  background job completed: cleanup", "INFO"},
		{"2024-01-15 10:23:30 WARN  disk usage at 85%", "WARN"},
		{"2024-01-15 10:23:35 INFO  server started on :8080 (sample data)", "INFO"},
	}
}
