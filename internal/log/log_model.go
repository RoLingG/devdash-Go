package log

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
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

	levelOverlay bool         // level filter 选择界面是否打开
	levelIdx     int          // 当前光标位置（0=All, 1=INFO, 2=WARN, 3=ERROR, 4=DEBUG）
	levelSel     map[int]bool // level filter 多选状态

	followMode bool      // tail -f 跟随模式
	lastMtime  time.Time // 上次文件修改时间
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
			m.input.Prompt = "Log file path:"
			m.input.Open(m.logPath)
			return m, nil
		}
		lines := msg.Lines
		if lines == nil {
			lines = SampleLogLines()
		}
		m.all = lines
		m.loaded = true
		m.input.Active = false
		m.dirListing = false
		m.errMsg = ""
		m.pageSize = logPageSize
		m.applyFilter()

	case DirMsg:
		if msg.Files == nil || len(msg.Files) == 0 {
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

	case tailTickMsg:
		if !m.followMode || m.logPath == "" {
			return m, nil
		}
		info, err := os.Stat(m.logPath)
		if err != nil {
			return m, nil
		}
		if info.ModTime().After(m.lastMtime) {
			m.lastMtime = info.ModTime()
			return m, tea.Batch(LoadFromFileCmd(m.logPath), tailTickCmd(2*time.Second))
		}
		return m, tailTickCmd(2*time.Second)

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
		case "[":
			if m.page > 0 {
				m.page--
				m.cursor = 0
				m.scrollOff = 0
			}
		case "]":
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
			m.followMode = !m.followMode
			if m.followMode && m.logPath != "" {
				info, err := os.Stat(m.logPath)
				if err == nil {
					m.lastMtime = info.ModTime()
				}
				return m, tailTickCmd(2 * time.Second)
			}
		case "ctrl+r":
			if m.logPath != "" {
				return m, func() tea.Msg { return LoadFromFile(m.logPath) }
			}
		case "enter":
			m.applyFilter()
		case "esc":
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
		re, err := regexp.Compile(m.filter)
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
	cardWidth := m.width - 2
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
		filterInfo = fmt.Sprintf(" Filter: %s (%d/%d)", m.filter, len(m.filtered), len(m.all))
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
	if m.followMode {
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
			colored := colorizeLog(Line{Level: l.Level, Raw: wl})
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

func colorizeLog(l Line) string {
	var color lipgloss.Color
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
	return "  " + lipgloss.NewStyle().Foreground(color).Render(l.Raw)
}
