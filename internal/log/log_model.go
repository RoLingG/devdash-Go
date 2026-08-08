package log

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model 日志查看器模块状态
type Model struct {
	all      []Line
	filtered []Line
	width    int
	height   int
	filter   string
	loaded   bool
	errMsg   string
	warnMsg  string // 大文件警告
	logPath  string

	page      int
	cursor    int
	pageSize  int
	scrollOff int

	pageInput      bool
	pageInputValue string

	input      component.InputModel
	dirListing bool
	dirPath    string
	dirList    component.ListModel

	levelOverlay bool
	levelIdx     int
	levelSel     map[int]bool

	tailFMode bool
	tailCh    <-chan []byte
	tailDone  chan struct{}

	cachedLines []Line
	cachedMtime time.Time
	cachedPath  string
}

func (m *Model) reloadFromCache() bool {
	if m.cachedPath == "" {
		return false
	}
	info, err := os.Stat(m.logPath)
	if err != nil {
		return false
	}
	// 文件未变化
	if m.logPath == m.cachedPath && info.ModTime() == m.cachedMtime {
		m.all = m.cachedLines
		m.loaded = true
		m.applyFilter()
		return true
	}
	return false
}

const logPageSize = 10

// SetRecent 设置最近记录列表（转发给 InputModel）
func (m *Model) SetRecent(items []string) { m.input.SetRecent(items) }

// levelOpts level filter 选项
var levelOpts = []string{"ALL", "INFO", "WARN", "ERROR", "DEBUG"}

// selectedLevels 返回当前选中的级别切片，空切片表示 ALL
func (m *Model) selectedLevels() []string {
	var levels []string
	for i := 1; i < len(levelOpts); i++ {
		if m.levelSel[i] {
			levels = append(levels, levelOpts[i])
		}
	}
	return levels
}

func (m *Model) totalPages() int {
	n := len(m.filtered)
	if n == 0 {
		return 1
	}
	return (n + logPageSize - 1) / logPageSize
}

func (m *Model) clampPage() {
	tp := m.totalPages()
	if m.page >= tp {
		m.page = tp - 1
	}
	if m.page < 0 {
		m.page = 0
	}
}

func (m *Model) Init(lastLogPath string) tea.Cmd {
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		return LoadFromStdin
	}
	for i, arg := range os.Args[1:] {
		if arg == "--log" {
			fileIdx := i + 2
			if fileIdx < len(os.Args) {
				logPath := os.Args[fileIdx]
				m.logPath = logPath
				return LoadFromFileCmd(logPath)
			}
		}
	}
	if lastLogPath != "" {
		m.logPath = lastLogPath
		return LoadFromFileCmd(lastLogPath)
	}
	return nil
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active || m.pageInput }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		if msg.Err != nil {
			m.errMsg = msg.Err.Error()
			m.loaded = true
			// 通过命令行参数加载时，不打开输入框，直接显示错误信息
			// 只有通过用户手动输入路径时才打开输入框
			if m.input.Active {
				m.input.Prompt = "Log file path:"
				m.input.Open(m.logPath)
			}
			return m, nil
		}
		lines := msg.Lines
		if lines == nil {
			lines = SampleLogLines()
		}
		m.all = lines
		m.loaded = true
		m.warnMsg = msg.Warn
		m.cachedLines = lines
		m.cachedPath = m.logPath
		if info, err := os.Stat(m.logPath); err == nil {
			m.cachedMtime = info.ModTime()
		}
		m.input.Active = false
		m.dirListing = false
		m.errMsg = ""
		m.pageSize = logPageSize
		m.applyFilter()

	case DirMsg:
		if len(msg.Files) == 0 {
			m.errMsg = "No .log files found in: " + msg.Dir
			m.input.Prompt = "Log file path:"
			m.input.Open(m.dirPath)
			m.dirListing = false
			return m, nil
		}
		m.dirListing = true
		m.dirPath = msg.Dir
		m.dirList.SetItems(msg.Files)
		m.input.Active = false
		return m, nil

	case TailDataMsg:
		if msg.Done || msg.Err != nil {
			// channel 关闭或出错，停止监听
			m.tailFMode = false
			return m, nil
		}
		// 追加新行到日志
		m.all = append(m.all, msg.Lines...)
		m.applyFilter()
		// 继续接收下一块数据
		return m, receiveTailCmd(m.tailCh)

	case tea.PasteMsg:
		if m.input.Active {
			return m, m.input.Update(msg, nil)
		}
		m.filter += msg.Content
		m.applyFilter()
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		if m.dirListing {
			switch key {
			case "up", "k":
				m.dirList.MoveUp()
			case "down", "j":
				m.dirList.MoveDown()
			case "enter":
				selected := m.dirList.Selected()
				fullPath := filepath.Join(m.dirPath, selected)
				m.logPath = fullPath
				m.dirListing = false
				m.filter = ""
				m.errMsg = ""
				return m, tea.Batch(LoadFromFileCmd(fullPath), ui.UpdateCfgCmd("logPath", fullPath))
			case "esc":
				m.dirListing = false
				m.input.Prompt = "Log file path:"
				m.input.Open(m.dirPath)
			}
			return m, nil
		}

		// 页码跳转输入模式
		if m.pageInput {
			switch key {
			case "enter":
				m.pageInput = false
				var pageNum int
				if _, err := fmt.Sscanf(m.pageInputValue, "%d", &pageNum); err == nil {
					m.page = pageNum - 1
					m.clampPage()
					m.cursor = 0
					m.scrollOff = 0
				}
				m.pageInputValue = ""
			case "esc":
				m.pageInput = false
				m.pageInputValue = ""
			case "backspace":
				if len(m.pageInputValue) > 0 {
					m.pageInputValue = m.pageInputValue[:len(m.pageInputValue)-1]
				}
			default:
				if len(key) == 1 && key >= "0" && key <= "9" {
					m.pageInputValue += key
				}
			}
			return m, nil
		}

		// Level filter 选择界面
		if m.levelOverlay {
			switch key {
			case "up", "k":
				if m.levelIdx > 0 {
					m.levelIdx--
				}
			case "down", "j":
				if m.levelIdx < len(levelOpts)-1 {
					m.levelIdx++
				}
			case "enter", "space":
				if m.levelIdx == 0 {
					// All — 清除所有选中
					m.levelSel = map[int]bool{}
				} else {
					m.levelSel[m.levelIdx] = !m.levelSel[m.levelIdx]
				}
				m.applyFilter()
			case "esc", "ctrl+l":
				m.levelOverlay = false
			}
			return m, nil
		}

		if m.input.Active {
			return m, tea.Batch(
				m.input.Update(msg, func(path string) func() tea.Msg {
					if path != "" {
						m.errMsg = ""
						info, err := os.Stat(path)
						if err != nil {
							m.errMsg = "Path not found: " + path
							return nil
						}
						if info.IsDir() {
							return ScanDirCmd(path)
						}
						m.logPath = path
						if m.reloadFromCache() {
							return nil
						}
						m.filter = ""
						return LoadFromFileCmd(path)
					}
					return nil
				}),
				ui.UpdateCfgCmd("logPath", m.logPath),
			)
		}

		switch key {
		// 页内光标移动
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			start := m.page * logPageSize
			end := start + logPageSize
			if end > len(m.filtered) {
				end = len(m.filtered)
			}
			pageEntries := end - start
			if m.cursor < pageEntries-1 {
				m.cursor++
			}
		// 翻页
		case "left":
			if m.page > 0 {
				m.page--
				m.cursor = 0
				m.scrollOff = 0
			}
		case "right":
			if m.page < m.totalPages()-1 {
				m.page++
				m.cursor = 0
				m.scrollOff = 0
			}
		// 快速翻页
		case "ctrl+up":
			m.page -= 10
			if m.page < 0 {
				m.page = 0
			}
			m.cursor = 0
			m.scrollOff = 0
		case "ctrl+down":
			m.page += 10
			m.clampPage()
			m.cursor = 0
			m.scrollOff = 0
		case "home":
			m.page = 0
			m.cursor = 0
			m.scrollOff = 0
		case "end":
			m.page = m.totalPages() - 1
			if m.page < 0 {
				m.page = 0
			}
			start := m.page * logPageSize
			end := start + logPageSize
			if end > len(m.filtered) {
				end = len(m.filtered)
			}
			m.cursor = end - start - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.scrollOff = 0
		// 页码跳转
		case "ctrl+p":
			m.pageInput = true
			m.pageInputValue = ""
		// 其他功能键
		case "ctrl+l":
			m.levelOverlay = true
			m.levelIdx = 0
			if m.levelSel == nil {
				m.levelSel = map[int]bool{}
			}
		case "/":
			m.input.Prompt = "Log file path:"
			m.input.Open(m.logPath)
		case "ctrl+f":
			m.tailFMode = !m.tailFMode
			if m.tailFMode && m.logPath != "" {
				// 获取当前文件大小作为起始偏移
				info, err := os.Stat(m.logPath)
				if err != nil {
					m.tailFMode = false
					return m, nil
				}
				offset := info.Size()
				m.tailDone = make(chan struct{})
				m.tailCh = watchFile(m.logPath, offset, m.tailDone)
				return m, receiveTailCmd(m.tailCh)
			}
			if !m.tailFMode && m.tailDone != nil {
				// 关闭 goroutine
				close(m.tailDone)
				m.tailDone = nil
				m.tailCh = nil
			}
		case "ctrl+r":
			if m.logPath != "" {
				// tail 模式下重启 goroutine
				if m.tailFMode {
					if m.tailDone != nil {
						close(m.tailDone)
					}
					info, err := os.Stat(m.logPath)
					if err == nil {
						m.tailDone = make(chan struct{})
						m.tailCh = watchFile(m.logPath, info.Size(), m.tailDone)
					}
				}
				if m.reloadFromCache() {
					if m.tailFMode {
						return m, receiveTailCmd(m.tailCh)
					}
					return m, nil
				}
				if m.tailFMode {
					return m, tea.Batch(LoadFromFileCmd(m.logPath), receiveTailCmd(m.tailCh))
				}
				return m, LoadFromFileCmd(m.logPath)
			}
		case "enter":
			m.applyFilter()
		case "ctrl+u":
			m.filter = ""
			m.applyFilter()
		case "backspace":
			if ui.RuneLen(m.filter) > 0 {
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

func (m *Model) applyFilter() {
	// level 过滤
	levels := m.selectedLevels()
	result := m.all
	if len(levels) > 0 {
		levelSet := map[string]bool{}
		for _, l := range levels {
			levelSet[l] = true
		}
		var byLevel []Line
		for _, line := range m.all {
			if levelSet[line.Level] {
				byLevel = append(byLevel, line)
			}
		}
		result = byLevel
	}

	// 正则过滤
	if m.filter != "" {
		re, err := regexp.Compile("(?i)" + m.filter)
		if err != nil {
			m.errMsg = "Invalid regex"
			return
		}
		var byRegex []Line
		for _, line := range result {
			if re.MatchString(line.Raw) {
				byRegex = append(byRegex, line)
			}
		}
		result = byRegex
	}

	m.errMsg = ""
	m.filtered = result
	m.page = m.totalPages() - 1
	m.cursor = 0
}

func (m *Model) View() string {
	cardWidth := m.width
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 目录文件列表模式
	if m.dirListing {
		return ui.RenderDirListCard(ui.DirListCardOpts{
			Title:     "Log Files",
			DirPath:   m.dirPath,
			DirList:   &m.dirList,
			Height:    m.height,
			CardWidth: cardWidth,
			ErrMsg:    m.errMsg,
		})
	}

	// 路径输入模式
	if m.input.Active {
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       "Open Log",
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			ErrMsg:      m.errMsg,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	// 页码跳转输入模式
	if m.pageInput {
		inputStyle := lipgloss.NewStyle().Foreground(ui.ColAccent)
		mutedStyle := lipgloss.NewStyle().Foreground(ui.ColMuted)
		content := fmt.Sprintf("\n  %s %s%s\n\n  %s",
			inputStyle.Render("Go to page:"),
			inputStyle.Render(m.pageInputValue),
			inputStyle.Render("|"),
			mutedStyle.Render("Enter confirm  Esc cancel"))
		return ui.Card("Page Jump", content, ui.ColAccent, cardWidth)
	}

	// Level filter 选择界面
	if m.levelOverlay {
		return m.renderLevelFilter(cardWidth)
	}

	// 未加载
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to enter log file path")
		return ui.Card("Log Viewer", emptyContent, ui.ColMuted, cardWidth)
	}

	// 加载失败
	if m.errMsg != "" {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("  ✗ "+m.errMsg) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to enter another path")
		return ui.Card("Log Viewer", errContent, ui.ColRed, cardWidth)
	}

	var lines []string

	// 页信息
	start := m.page * logPageSize
	end := start + logPageSize
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	totalP := m.totalPages()

	// 标题行：过滤信息 + 页码
	filterInfo := ""
	hasLevel := len(m.selectedLevels()) > 0
	if m.filter != "" {
		hint := lipgloss.NewStyle().Foreground(ui.ColMuted).Render(" ctrl+u clear")
		filterInfo = fmt.Sprintf(" Filter: %s (%d/%d)", m.filter, len(m.filtered), len(m.all)) + hint
	} else if hasLevel {
		filterInfo = fmt.Sprintf(" %d/%d entries", len(m.filtered), len(m.all))
	} else {
		filterInfo = fmt.Sprintf(" %d entries", len(m.filtered))
	}
	// level filter 徽标
	levels := m.selectedLevels()
	if len(levels) > 0 {
		levelBadge := "Level: " + strings.Join(levels, "/")
		filterInfo += " | " + lipgloss.NewStyle().Foreground(ui.ColGreen).Render(levelBadge)
	}
	// tail -f 徽标
	if m.tailFMode {
		filterInfo += " | " + lipgloss.NewStyle().Foreground(ui.ColGreen).Bold(true).Render("Following")
	}
	pageInfo := fmt.Sprintf("Page %d/%d  [%d-%d/%d]", m.page+1, totalP, start+1, end, len(m.filtered))
	if m.errMsg != "" {
		filterInfo += " | " + lipgloss.NewStyle().Foreground(ui.ColRed).Render(m.errMsg)
	}
	headerLine := lipgloss.NewStyle().Foreground(ui.ColAccent).Render(filterInfo) +
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+pageInfo)
	lines = append(lines, headerLine)
	lines = append(lines, "")

	// 大文件警告
	if m.warnMsg != "" {
		warnStyle := lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true)
		lines = append(lines, warnStyle.Render("  ⚠ "+m.warnMsg))
		lines = append(lines, "")
	}

	// 空状态提示
	if len(m.filtered) == 0 {
		emptyMsg := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  🔍 没有找到匹配的日志行，尝试修改过滤条件")
		lines = append(lines, emptyMsg)
		return ui.Card("Log Viewer", lipgloss.JoinVertical(lipgloss.Left, lines...), ui.ColAccent, cardWidth)
	}

	maxRawW := m.width - 12
	if maxRawW < 10 {
		maxRawW = 10
	}

	// 构建当前页所有换行后的渲染行
	cursorStyle := lipgloss.NewStyle().Foreground(ui.ColPrimary).Bold(true)
	var wrappedLines []string
	cursorLine := 0
	cursorEndLine := 0
	foundCursor := false
	for i := start; i < end; i++ {
		l := m.filtered[i]
		wrapped := wrapLine(l.Raw, maxRawW)
		isCursor := i-start == m.cursor
		if isCursor {
			cursorLine = len(wrappedLines)
			cursorEndLine = cursorLine + len(wrapped)
			foundCursor = true
		}
		for j, wl := range wrapped {
			colored := colorizeLog(Line{Level: l.Level, Raw: wl}, m.filter)
			if isCursor {
				if j == 0 {
					wrappedLines = append(wrappedLines, cursorStyle.Render(">")+colored)
				} else {
					wrappedLines = append(wrappedLines, cursorStyle.Render(" ")+colored)
				}
			} else {
				wrappedLines = append(wrappedLines, "  "+colored)
			}
		}
	}

	// Box 可显示内容行 = (m.height - 3) - Box开销(4) - header+空行(2) - 空行+stats(2)
	maxVisible := m.height - 3 - 4 - 2 - 2
	if maxVisible < 1 {
		maxVisible = 1
	}

	// 自动调整 scrollOff 确保 cursor 全部行可见
	if foundCursor {
		if cursorLine < m.scrollOff {
			m.scrollOff = cursorLine
		} else if cursorEndLine > m.scrollOff+maxVisible {
			m.scrollOff = cursorEndLine - maxVisible
		}
	}
	if m.scrollOff < 0 {
		m.scrollOff = 0
	}
	// 防御性代码，防止特殊情况超出可视边界，导致内容下方出现空白行
	if len(wrappedLines) > maxVisible && m.scrollOff > len(wrappedLines)-maxVisible {
		m.scrollOff = len(wrappedLines) - maxVisible
	}

	// 截取可见部分
	visibleEnd := m.scrollOff + maxVisible
	if visibleEnd > len(wrappedLines) {
		visibleEnd = len(wrappedLines)
	}
	lines = append(lines, wrappedLines[m.scrollOff:visibleEnd]...)

	lines = append(lines, "")
	lines = append(lines, m.stats())
	return ui.Box("Log", strings.Join(lines, "\n"), m.width)
}

func (m *Model) renderLevelFilter(cardWidth int) string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ui.ColText)
	mutedStyle := lipgloss.NewStyle().Foreground(ui.ColMuted)
	selectedStyle := lipgloss.NewStyle().Foreground(ui.ColGreen)

	var sb strings.Builder
	sb.WriteString("\n")
	for i, opt := range levelOpts {
		// 光标指示
		cursor := "  "
		if i == m.levelIdx {
			cursor = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸ ")
		}

		// 选中状态
		checkbox := "[ ]"
		isSelected := i == 0 && len(m.levelSel) == 0 // All
		if i > 0 {
			isSelected = m.levelSel[i]
		}
		if isSelected {
			checkbox = selectedStyle.Render("[✓]")
		}

		// 高亮当前光标行
		label := descStyle.Render(opt)
		if i == m.levelIdx {
			label = keyStyle.Render(opt)
		}

		sb.WriteString(fmt.Sprintf("  %s %s %s\n", cursor, checkbox, label))
	}
	sb.WriteString("\n")
	sb.WriteString(mutedStyle.Render("  ↑↓/kj navigate  Enter/Space toggle  Esc close"))
	sb.WriteString("\n")
	return ui.Card("Level Filter", sb.String(), ui.ColAccent, cardWidth)
}

func (m *Model) stats() string {
	counts := map[string]int{}
	for _, l := range m.all {
		counts[l.Level]++
	}
	return fmt.Sprintf(" %s  %s  %s  %s  %s",
		lipgloss.NewStyle().Foreground(ui.ColRed).Render(fmt.Sprintf("ERROR:%d", counts["ERROR"])),
		lipgloss.NewStyle().Foreground(ui.ColAccent).Render(fmt.Sprintf("WARN:%d", counts["WARN"])),
		lipgloss.NewStyle().Foreground(ui.ColGreen).Render(fmt.Sprintf("INFO:%d", counts["INFO"])),
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("DEBUG:%d", counts["DEBUG"])),
		fmt.Sprintf("OTHER:%d", counts["OTHER"]),
	)
}

// wrapLine 按可见宽度将长行切为多行（纯文本，不含 ANSI 码）
func wrapLine(s string, maxW int) []string {
	if maxW < 1 || lipgloss.Width(s) <= maxW {
		return []string{s}
	}
	var result []string
	runes := []rune(s)
	w := 0
	start := 0
	for i, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > maxW {
			result = append(result, string(runes[start:i]))
			start = i
			w = rw
		} else {
			w += rw
		}
	}
	if start < len(runes) {
		result = append(result, string(runes[start:]))
	}
	return result
}

func colorizeLog(l Line, filter string) string {
	// 修改: lipgloss v2 中颜色类型改为标准库 image/color.Color
	var color color.Color
	switch l.Level {
	case "ERROR":
		color = ui.ColRed
	case "WARN":
		color = ui.ColAccent
	case "INFO":
		color = ui.ColGreen
	case "DEBUG":
		color = ui.ColMuted
	default:
		return "  " + l.Raw
	}

	raw := l.Raw
	levelStyle := lipgloss.NewStyle().Foreground(color)
	if filter != "" {
		idx := strings.Index(strings.ToLower(raw), strings.ToLower(filter))
		if idx >= 0 {
			highlight := lipgloss.NewStyle().Background(ui.ColAccent).Foreground(ui.ColText)
			before := levelStyle.Render(raw[:idx])
			match := highlight.Render(raw[idx : idx+len(filter)])
			after := levelStyle.Render(raw[idx+len(filter):])
			return "  " + before + match + after
		}
	}
	return "  " + levelStyle.Render(raw)
}
