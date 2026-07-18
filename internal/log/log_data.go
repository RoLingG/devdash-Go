package log

import (
	"bufio"
	"fmt"
	"io"
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

// 大文件阈值
const (
	MaxFileSize  = 100 * 1024 * 1024 // 100MB
	MaxLines     = 1000000           // 100万行
	WarnFileSize = 50 * 1024 * 1024  // 50MB 开始警告
)

// LoadMsg 从文件或 stdin 加载完成后的消息
type LoadMsg struct {
	Lines []Line
	Err   error
	Warn  string // 大文件警告信息
	Trunc bool   // 是否被截断
}

// DirMsg 目录扫描结果
type DirMsg struct {
	Dir   string
	Files []string
}

// TailDataMsg tail -f 文件变化消息（导出供 main.go 跨模块路由使用）
type TailDataMsg struct {
	Lines []Line // 新增的日志行
	Err   error  // 错误（EOF 表示 channel 关闭）
	Done  bool   // channel 已关闭，停止监听
}

// watchFile 启动 goroutine 监听文件变化，通过 channel 返回新增内容
func watchFile(path string, offset int64, done <-chan struct{}) <-chan []byte {
	ch := make(chan []byte, 1)
	go func() {
		defer close(ch)
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		f.Seek(offset, io.SeekStart)

		buf := make([]byte, 64*1024) // 64KB 读缓冲
		for {
			select {
			case <-done:
				return
			default:
			}
			n, err := f.Read(buf) // 阻塞读取内容
			if n > 0 {
				// 复制后再发给 channel，避免数据竞争
				data := make([]byte, n)
				copy(data, buf[:n])
				select {
				case ch <- data:
				case <-done:
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return
			}
		}
	}()
	return ch
}

// receiveTailCmd 阻塞等待 channel 返回数据的 Cmd
func receiveTailCmd(ch <-chan []byte) tea.Cmd {
	return func() tea.Msg {
		data, ok := <-ch
		if !ok {
			return TailDataMsg{Done: true}
		}
		var lines []Line
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			raw := scanner.Text()
			if raw != "" {
				lines = append(lines, Line{Raw: raw, Level: detectLevel(raw)})
			}
		}
		return TailDataMsg{Lines: lines}
	}
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
			if raw == "" {
				continue
			}
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

	// 检测文件大小
	stat, err := f.Stat()
	if err != nil {
		return LoadMsg{Err: fmt.Errorf("cannot stat: %s (%v)", path, err)}
	}
	fileSize := stat.Size()

	// 检查文件大小限制
	if fileSize > MaxFileSize {
		sizeMB := float64(fileSize) / 1024 / 1024
		return LoadMsg{Err: fmt.Errorf("文件过大（%.1fMB），超过限制（%dMB）", sizeMB, MaxFileSize/1024/1024)}
	}

	var lines []Line
	var warnMsg string
	truncated := false

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		raw := scanner.Text()
		lines = append(lines, Line{Raw: raw, Level: detectLevel(raw)})

		// 检查行数限制
		if len(lines) >= MaxLines {
			truncated = true
			warnMsg = fmt.Sprintf("文件行数过多（>%d行），已截断显示", MaxLines)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return LoadMsg{Lines: lines, Err: fmt.Errorf("read error: %v", err)}
	}

	// 生成警告信息
	if warnMsg == "" && fileSize >= WarnFileSize {
		sizeMB := float64(fileSize) / 1024 / 1024
		warnMsg = fmt.Sprintf("大文件警告：%.1fMB，%d行", sizeMB, len(lines))
	}

	return LoadMsg{Lines: lines, Warn: warnMsg, Trunc: truncated}
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
