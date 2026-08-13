package devtools

import (
	"fmt"
	"strings"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model DevTools 开发工具箱模块状态
type Model struct {
	width      int
	height     int
	tools      []Tool
	visible    []Tool // 当前功能区的工具子集
	section    int    // 当前功能区索引
	cursor     int    // 工具列表光标
	input      component.InputModel
	inputValue string   // 已确认的输入文本
	recent     []string // 会话内最近输入记录
	result     string   // 当前工具计算结果
	resultErr  error    // 当前工具计算错误
	outScroll  int      // 输出滚动偏移
}

// Init 初始化工具列表
func (m *Model) Init() tea.Cmd {
	if m.tools == nil {
		m.tools = builtinTools
	}
	m.visible = toolsInSection(m.tools, sections[m.section])
	return nil
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		if m.input.Active {
			return m, m.input.Update(msg, m.onSubmit)
		}
	case tea.KeyPressMsg:
		if m.input.Active {
			return m, m.input.Update(msg, m.onSubmit)
		}
		switch msg.String() {
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "home":
			m.cursor = 0
			m.compute()
		case "end":
			m.cursor = len(m.visible) - 1
			m.compute()
		case "tab":
			m.switchSection(1)
		case "shift+tab":
			m.switchSection(-1)
		case "/":
			m.input.Prompt = "Text:"
			m.input.Open("")
		case "pgup":
			m.outScroll -= 5
		case "pgdown":
			m.outScroll += 5
		case "ctrl+r":
			if m.inputValue == "" && m.isNoInput() {
				m.result, m.resultErr = "", nil // 清缓存，强制重新生成（如 UUID）
			}
			m.compute()
		case "ctrl+u":
			m.clear()
		}
	}
	return m, nil
}

// onSubmit 确认输入并重算，非空输入记入最近列表
func (m *Model) onSubmit(value string) func() tea.Msg {
	m.inputValue = value
	if value != "" {
		m.recent = ui.AddToRecent(m.recent, value, 10)
		m.input.SetRecent(m.recent)
	}
	m.compute()
	return nil
}

// compute 用当前输入重算当前工具结果
func (m *Model) compute() {
	m.outScroll = 0
	if m.cursor >= len(m.visible) {
		m.result, m.resultErr = "", nil
		return
	}
	t := m.visible[m.cursor]
	if m.inputValue == "" {
		if !t.NoInput { // 普通工具空输入不计算
			m.result, m.resultErr = "", nil
			return
		}
		if m.result != "" { // NoInput 工具结果已生成则跳过，避免移动光标反复生成
			return
		}
	}
	m.result, m.resultErr = t.Run(m.inputValue)
}

// clear 清空输入与结果
func (m *Model) clear() {
	m.inputValue = ""
	m.result, m.resultErr = "", nil
	m.outScroll = 0
}

// switchSection 切换功能区并重置光标
func (m *Model) switchSection(delta int) {
	total := len(sections)
	m.section += delta
	if m.section < 0 {
		m.section = total - 1
	}
	if m.section >= total {
		m.section = 0
	}
	m.visible = toolsInSection(m.tools, sections[m.section])
	m.cursor = 0
	m.compute()
}

// moveCursor 移动工具光标并自动重算
func (m *Model) moveCursor(delta int) {
	total := len(m.visible)
	if total == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	m.compute()
}

// isNoInput 当前工具是否无需输入
func (m *Model) isNoInput() bool {
	return m.cursor < len(m.visible) && m.visible[m.cursor].NoInput
}

func (m *Model) View() string {
	cardWidth := m.width
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 输入框打开时渲染输入卡片
	if m.input.Active {
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       "DevTools",
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	contentH := m.height - 7
	if contentH < 3 {
		contentH = 3
	}
	innerW := cardWidth - 4 // 边框 2 + 左右 padding 2
	leftW := 26
	rightW := innerW - leftW - 2 // 2 = 两列间距
	if rightW < 20 {
		rightW = 20
	}

	// ---- 左列：工具列表（滚动窗口）----
	type rowKind int
	const (
		rowGroup rowKind = iota
		rowTool
	)
	type displayRow struct {
		kind rowKind
		text string
		idx  int // rowTool 时的工具索引
	}
	var rows []displayRow
	var curGroup string
	for i, t := range m.visible {
		if t.Group != curGroup { // 新的分组起点，插入段标题行
			curGroup = t.Group
			rows = append(rows, displayRow{kind: rowGroup, text: t.Group})
		}
		rows = append(rows, displayRow{kind: rowTool, text: t.Name, idx: i})
	}

	// 找到光标对应的显示行号
	cursorRow := 0
	for i, r := range rows {
		if r.kind == rowTool && r.idx == m.cursor {
			cursorRow = i
			break
		}
	}

	window := contentH - 1 // 减去标题行
	if window < 1 {
		window = 1
	}
	start := 0
	if cursorRow >= window {
		start = cursorRow - window + 1
	}
	end := start + window
	if end > len(rows) {
		end = len(rows)
	}

	// 左列顶部：功能区 Tab 条
	var tabs []string
	for i, s := range sections {
		label := " " + s + " "
		if i == m.section {
			tabs = append(tabs, lipgloss.NewStyle().Bold(true).Foreground(ui.ColBgDark).Background(ui.ColPrimary).Render(label))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().Foreground(ui.ColMuted).Background(ui.ColBgMid).Render(label))
		}
	}
	left := []string{strings.Join(tabs, "")}
	for i := start; i < end; i++ {
		r := rows[i]
		switch r.kind {
		case rowGroup:
			left = append(left, lipgloss.NewStyle().Foreground(ui.ColMuted).Bold(true).Render("  "+r.text))
		case rowTool:
			idx := fmt.Sprintf("%2d", r.idx+1)
			if r.idx == m.cursor {
				left = append(left, "▸ "+lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true).Render(idx+" "+r.text))
			} else {
				left = append(left, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+idx+" "+r.text))
			}
		}
	}
	for len(left) < contentH {
		left = append(left, "")
	}
	leftCol := strings.Join(left[:contentH], "\n")

	// ---- 右列：输入 + 输出 ----
	var right []string
	right = append(right, lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Input"))
	if m.inputValue == "" {
		switch {
		case m.isNoInput():
			right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (no input - ctrl+r to regenerate)"))
		case m.cursor < len(m.visible) && m.visible[m.cursor].Hint != "":
			// 多参数工具（如 HMAC/AES）显示格式提示
			right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+m.visible[m.cursor].Hint))
		default:
			right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (press / to input)"))
		}
	} else {
		// 长输入分组换行显示，超 6 行截断
		inLines := strings.Split(wrapLong(m.inputValue), "\n")
		const maxInputLines = 6
		if len(inLines) > maxInputLines {
			omitted := len(inLines) - maxInputLines
			inLines = inLines[:maxInputLines]
			inLines[maxInputLines-1] += fmt.Sprintf(" …(+%d lines)", omitted)
		}
		for _, l := range inLines {
			right = append(right, lipgloss.NewStyle().Foreground(ui.ColText).Width(rightW-2).Render("  "+l))
		}
	}
	right = append(right, "")

	right = append(right, lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Output"))
	switch {
	case m.resultErr != nil:
		right = append(right, lipgloss.NewStyle().Foreground(ui.ColRed).Render("  ✗ "+m.resultErr.Error()))
	case m.inputValue == "" && !m.isNoInput():
		right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (empty)"))
	default:
		// 输出按右列宽度自动换行
		outLines := strings.Split(lipgloss.NewStyle().Width(rightW-2).Render(m.result), "\n")
		avail := contentH - len(right)
		if avail < 1 {
			avail = 1
		}
		total := len(outLines)
		if m.outScroll > total-avail {
			m.outScroll = total - avail
		}
		if m.outScroll < 0 {
			m.outScroll = 0
		}
		outEnd := m.outScroll + avail
		if outEnd > total {
			outEnd = total
		}
		if total > avail { // 超出可视区，显示滚动位置提示
			right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render(
				fmt.Sprintf("  [pgup/pgdn %d-%d/%d]", m.outScroll+1, outEnd, total)))
		}
		for _, l := range outLines[m.outScroll:outEnd] {
			right = append(right, "  "+l)
		}
	}
	for len(right) < contentH {
		right = append(right, "")
	}
	rightCol := strings.Join(right[:contentH], "\n")

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
	return ui.Card("DevTools", content, ui.ColAccent, cardWidth)
}
