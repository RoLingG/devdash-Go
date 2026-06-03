// ============================================================================
// mod_log.go — 日志查看器模块
//
// 功能：
//   - 从 stdin (pipe) 或指定文件读取日志
//   - / 键输入路径：输入目录则列出 .log 文件供选择，输入文件则直接加载
//   - 输入框支持 ←/→ 光标移动、Home/End 跳转
//   - 根据关键词自动着色（ERROR=红, WARN=黄, INFO=绿, DEBUG=灰）
//   - 非输入模式下直接键入字符过滤日志，Esc 清除过滤
//   - ↑↓ 滚动
//   - 底部统计：各级别日志数量
//
// 使用方式：
//   cat app.log | devdash.exe          # 从 pipe 读取
//   devdash.exe --log app.log          # 从文件读取
//   devdash.exe                        # 按 / 输入路径
// ============================================================================

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// ==================== 消息类型 ====================

// logLine 一行日志
type logLine struct {
	raw   string // 原始文本
	level string // 检测到的级别：ERROR/WARN/INFO/DEBUG/OTHER
}

// logLoadMsg 从文件或 stdin 加载完成后的消息
type logLoadMsg struct {
	lines []logLine
	err   error
}

// logDirMsg 目录扫描结果
type logDirMsg struct {
	dir   string
	files []string
}

// ==================== 加载函数 ====================

// loadLogFromStdin 从 stdin 读取日志（pipe 场景）
func loadLogFromStdin() tea.Msg {
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		var lines []logLine
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			raw := scanner.Text()
			lines = append(lines, logLine{raw: raw, level: detectLevel(raw)})
		}
		return logLoadMsg{lines: lines}
	}
	return logLoadMsg{lines: nil}
}

// loadLogFromFile 从文件读取日志
func loadLogFromFile(path string) tea.Msg {
	f, err := os.Open(path)
	if err != nil {
		return logLoadMsg{err: fmt.Errorf("cannot open: %s (%v)", path, err)}
	}
	defer f.Close()
	var lines []logLine
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		lines = append(lines, logLine{raw: raw, level: detectLevel(raw)})
	}
	if err := scanner.Err(); err != nil {
		return logLoadMsg{lines: lines, err: fmt.Errorf("read error: %v", err)}
	}
	return logLoadMsg{lines: lines}
}

// scanLogDir 扫描目录下所有 .log 文件
func scanLogDir(dir string) tea.Msg {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return logDirMsg{dir: dir, files: nil}
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ".log" {
			files = append(files, e.Name())
		}
	}
	return logDirMsg{dir: dir, files: files}
}

// ==================== 日志级别检测 ====================

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

// ==================== 模型 ====================

type logModel struct {
	all      []logLine // 所有日志行
	filtered []logLine // 过滤后的日志行
	width    int
	height   int
	scroll   int
	filter   string // 当前过滤正则
	loaded   bool
	errMsg   string
	logPath  string // 当前日志文件路径

	// 路径输入模式
	inputMode   bool   // 是否处于路径输入模式
	inputValue  string // 当前输入的路径
	inputCursor int    // 光标位置（字节偏移）

	// 目录文件列表模式
	dirListing bool     // 是否在显示目录文件列表
	dirPath    string   // 当前目录路径
	dirFiles   []string // 目录下的 .log 文件列表
	dirCursor  int      // 文件列表中当前选中的索引
}

func (m logModel) Init() tea.Cmd {
	// 管道模式：stdin 有数据时自动加载
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		return loadLogFromStdin
	}
	// 命令行参数：--log <file>
	for i, arg := range os.Args[1:] {
		if arg == "--log" {
			fileIdx := i + 2 // os.Args[1:][i+1] = os.Args[i+2]
			if fileIdx < len(os.Args) {
				logPath := os.Args[fileIdx]
				return func() tea.Msg { return loadLogFromFile(logPath) }
			}
		}
	}
	// 默认：显示路径输入界面
	return nil
}

func (m *logModel) UpdateSize(w, h int) { m.width = w; m.height = h }

// ==================== 更新逻辑 ====================

func (m logModel) Update(msg tea.Msg) (logModel, tea.Cmd) {
	switch msg := msg.(type) {

	// 日志加载完成
	case logLoadMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.inputMode = true
			m.inputValue = m.logPath
			m.inputCursor = runeLen(m.logPath)
			return m, nil
		}
		lines := msg.lines
		if lines == nil {
			lines = sampleLogLines()
		}
		m.all = lines
		m.filtered = lines
		m.loaded = true
		m.inputMode = false
		m.dirListing = false
		m.errMsg = ""
		m.scroll = len(m.filtered) - (m.height - 8)
		if m.scroll < 0 {
			m.scroll = 0
		}

	// 目录扫描完成
	case logDirMsg:
		if msg.files == nil || len(msg.files) == 0 {
			m.errMsg = "No .log files found in: " + msg.dir
			m.inputMode = true
			m.dirListing = false
			return m, nil
		}
		m.dirListing = true
		m.dirPath = msg.dir
		m.dirFiles = msg.files
		m.dirCursor = 0
		m.inputMode = false
		return m, nil

	// 粘贴
	case tea.PasteMsg:
		if m.inputMode {
			// 在光标位置插入粘贴内容
			m.inputValue = runeInsert(m.inputValue, msg.Content, m.inputCursor)
			m.inputCursor += runeLen(msg.Content)
		} else {
			m.filter += msg.Content
			m.applyFilter()
		}
		return m, nil

	// 按键
	case tea.KeyPressMsg:
		key := msg.String()

		// ---- 目录文件列表模式 ----
		if m.dirListing {
			switch key {
			case "up", "k":
				if m.dirCursor > 0 {
					m.dirCursor--
				}
			case "down", "j":
				if m.dirCursor < len(m.dirFiles)-1 {
					m.dirCursor++
				}
			case "enter":
				selected := m.dirFiles[m.dirCursor]
				fullPath := filepath.Join(m.dirPath, selected)
				m.logPath = fullPath
				m.dirListing = false
				m.filter = ""
				m.errMsg = ""
				return m, func() tea.Msg { return loadLogFromFile(fullPath) }
			case "esc":
				// 返回路径输入
				m.dirListing = false
				m.inputMode = true
				m.inputValue = m.dirPath
				m.inputCursor = runeLen(m.dirPath)
			}
			return m, nil
		}

		// ---- 路径输入模式 ----
		if m.inputMode {
			switch key {
			case "enter":
				path := strings.TrimSpace(m.inputValue)
				if path != "" {
					m.errMsg = ""
					// 检测是目录还是文件
					info, err := os.Stat(path)
					if err != nil {
						m.errMsg = "Path not found: " + path
						return m, nil
					}
					if info.IsDir() {
						// 目录：扫描 .log 文件
						return m, func() tea.Msg { return scanLogDir(path) }
					}
					// 文件：直接加载
					m.logPath = path
					m.filter = ""
					return m, func() tea.Msg { return loadLogFromFile(path) }
				}
				m.inputMode = false
			case "esc":
				m.inputMode = false
				m.inputValue = ""
				m.inputCursor = 0
				m.errMsg = ""
			case "left":
				if m.inputCursor > 0 {
					m.inputCursor--
				}
			case "right":
				if m.inputCursor < runeLen(m.inputValue) {
					m.inputCursor++
				}
			case "home":
				m.inputCursor = 0
			case "end":
				m.inputCursor = runeLen(m.inputValue)
			case "backspace":
				if m.inputCursor > 0 {
					m.inputValue = runeDeleteAt(m.inputValue, m.inputCursor-1)
					m.inputCursor--
				}
			case "delete":
				if m.inputCursor < runeLen(m.inputValue) {
					m.inputValue = runeDeleteAt(m.inputValue, m.inputCursor)
				}
			default:
				if len(key) == 1 && key >= " " {
					m.inputValue = runeInsert(m.inputValue, key, m.inputCursor)
					m.inputCursor++
				}
			}
			return m, nil
		}

		// ---- 正常模式 ----
		switch key {
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			maxScroll := len(m.filtered) - (m.height - 8)
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scroll < maxScroll {
				m.scroll++
			}
		case "/":
			m.inputMode = true
			m.inputValue = m.logPath
			m.inputCursor = runeLen(m.logPath)
		case "r":
			if m.logPath != "" {
				return m, func() tea.Msg { return loadLogFromFile(m.logPath) }
			}
		case "enter":
			m.applyFilter()
		case "esc":
			m.filter = ""
			m.filtered = m.all
			m.scroll = 0
		case "backspace":
			if runeLen(m.filter) > 0 {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.applyFilter()
			}
		default:
			if len(key) == 1 && key >= " " {
				m.filter += key
				m.applyFilter()
			}
		}
	}
	return m, nil
}

// applyFilter 用当前 filter 正则过滤日志
func (m *logModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.all
		m.scroll = 0
		return
	}
	re, err := regexp.Compile(m.filter)
	if err != nil {
		m.errMsg = "Invalid regex"
		return
	}
	m.errMsg = ""
	var filtered []logLine
	for _, line := range m.all {
		if re.MatchString(line.raw) {
			filtered = append(filtered, line)
		}
	}
	m.filtered = filtered
	m.scroll = 0
}

// ==================== 视图 ====================

func (m logModel) View() string {
	// 卡片宽度
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}
	//if cardWidth > 100 {
	//	cardWidth = 100
	//}

	// 目录文件列表模式
	if m.dirListing {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colSecondary).Render("  📂 " + m.dirPath))
		sb.WriteString("\n\n")

		maxShow := m.height - 10
		if maxShow < 5 {
			maxShow = 5
		}
		start := 0
		if m.dirCursor >= maxShow {
			start = m.dirCursor - maxShow + 1
		}
		end := start + maxShow
		if end > len(m.dirFiles) {
			end = len(m.dirFiles)
		}

		for i, f := range m.dirFiles[start:end] {
			idx := start + i
			if idx == m.dirCursor {
				sb.WriteString(lipgloss.NewStyle().Foreground(colAccent).Render("  > " + f))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render("    " + f))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  ↑↓ select  Enter open  Esc back"))
		if m.errMsg != "" {
			sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(colRed).Render(m.errMsg))
		}
		return card("Log Files", sb.String(), colSecondary, cardWidth)
	}

	// 路径输入模式
	if m.inputMode {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render("  File or directory path:"))
		sb.WriteString("\n")

		// 绘制输入行，光标位置用 | 标记
		before := runeSubstr(m.inputValue, 0, m.inputCursor)
		after := runeSubstr(m.inputValue, m.inputCursor, runeLen(m.inputValue))
		inputLine := "  > " + before + lipgloss.NewStyle().Foreground(colAccent).Render("|") + after
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render(inputLine))
		sb.WriteString("\n\n")

		if m.errMsg != "" {
			sb.WriteString("  " + lipgloss.NewStyle().Foreground(colRed).Render("✗ "+m.errMsg))
			sb.WriteString("\n\n")
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  Enter confirm  ←→ cursor  Home/End  Esc cancel"))
		return card("Open Log", sb.String(), colSecondary, cardWidth)
	}

	// 未加载
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(colMuted).Render("  Press '/' to enter log file path")
		return card("Log Viewer", emptyContent, colMuted, cardWidth)
	}

	var lines []string

	// 过滤状态栏
	filterInfo := ""
	if m.filter != "" {
		filterInfo = fmt.Sprintf(" Filter: %s (%d/%d lines)", m.filter, len(m.filtered), len(m.all))
	} else {
		filterInfo = fmt.Sprintf(" Showing all %d lines", len(m.filtered))
	}
	if m.errMsg != "" {
		filterInfo += " | " + lipgloss.NewStyle().Foreground(colRed).Render(m.errMsg)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(colAccent).Render(filterInfo))
	lines = append(lines, "")

	// 日志内容
	viewH := m.height - 10
	if viewH < 5 {
		viewH = 5
	}
	start := m.scroll
	end := start + viewH
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for _, l := range m.filtered[start:end] {
		lines = append(lines, colorizeLog(l))
	}

	// 底部统计
	lines = append(lines, "")
	lines = append(lines, m.stats())

	return strings.Join(lines, "\n")
}

// stats 统计各级别日志数量
func (m logModel) stats() string {
	counts := map[string]int{}
	for _, l := range m.all {
		counts[l.level]++
	}
	return fmt.Sprintf(" %s  %s  %s  %s  %s",
		lipgloss.NewStyle().Foreground(colRed).Render(fmt.Sprintf("ERROR:%d", counts["ERROR"])),
		lipgloss.NewStyle().Foreground(colAccent).Render(fmt.Sprintf("WARN:%d", counts["WARN"])),
		lipgloss.NewStyle().Foreground(colGreen).Render(fmt.Sprintf("INFO:%d", counts["INFO"])),
		lipgloss.NewStyle().Foreground(colMuted).Render(fmt.Sprintf("DEBUG:%d", counts["DEBUG"])),
		fmt.Sprintf("OTHER:%d", counts["OTHER"]),
	)
}

// colorizeLog 根据日志级别着色
func colorizeLog(l logLine) string {
	var color lipgloss.Color
	switch l.level {
	case "ERROR":
		color = colRed
	case "WARN":
		color = colAccent
	case "INFO":
		color = colGreen
	case "DEBUG":
		color = colMuted
	default:
		return "  " + l.raw
	}
	return "  " + lipgloss.NewStyle().Foreground(color).Render(l.raw)
}

// sampleLogLines 无数据时显示的示例日志
func sampleLogLines() []logLine {
	return []logLine{
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
