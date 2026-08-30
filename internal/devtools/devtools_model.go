package devtools

import (
	"fmt"
	"strings"

	"cava_go/internal/ui"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/atotto/clipboard"
)

// httpResultMsg 异步 HTTP 请求完成消息，携带 reqID 用于丢弃过期响应
type httpResultMsg struct {
	reqID  int
	result string
	err    error
}

// Model DevTools 开发工具箱模块状态
type Model struct {
	width      int
	height     int
	tools      []Tool
	visible    []Tool // 当前功能区的工具子集
	section    int    // 当前功能区索引
	cursor     int    // 工具列表光标
	input      textarea.Model
	recentIdx  int      // 输入框历史浏览索引，-1 表示未浏览
	inputValue string   // 已确认的输入文本
	recent     []string // 会话内最近输入记录
	result     string   // 当前工具计算结果
	resultErr  error    // 当前工具计算错误
	outScroll  int      // 输出滚动偏移
	copied     bool     // ctrl+p 复制成功提示
	loading    bool     // 异步工具（HTTP）请求进行中
	reqID      int      // 递增请求号，用于丢弃过期异步响应
}

// Init 初始化工具列表
func (m *Model) Init() tea.Cmd {
	if m.tools == nil {
		m.tools = builtinTools
	}
	m.visible = toolsInSection(m.tools, sections[m.section])
	m.initTextarea()
	return nil
}

// initTextarea 初始化多行输入框（bubbles textarea）
// 说明: 原 component.InputModel 仅支持单行，无法满足 HTTP/HMAC/AES 的多行输入格式
func (m *Model) initTextarea() {
	ta := textarea.New()
	ta.SetValue("")
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.Placeholder = ""
	m.input = ta
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Focused() }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case httpResultMsg:
		// 只接受最新请求的响应，过期响应（reqID 不匹配）直接丢弃
		if msg.reqID != m.reqID {
			return nil
		}
		m.loading = false
		m.result, m.resultErr = msg.result, msg.err
		return nil
	case tea.PasteMsg:
		if m.input.Focused() {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}
	case tea.KeyPressMsg:
		if m.input.Focused() {
			return m.handleInputKey(msg)
		}
		var cmd tea.Cmd
		switch msg.String() {
		case "up", "k":
			cmd = m.moveCursor(-1)
		case "down", "j":
			cmd = m.moveCursor(1)
		case "home":
			m.cursor = 0
			cmd = m.compute()
		case "end":
			m.cursor = len(m.visible) - 1
			cmd = m.compute()
		case "tab":
			cmd = m.switchSection(1)
		case "shift+tab":
			cmd = m.switchSection(-1)
		case "/":
			m.openInput("")
		case "pgup":
			m.outScroll -= 5
		case "pgdown":
			m.outScroll += 5
		case "ctrl+r":
			if m.inputValue == "" && m.isNoInput() {
				m.result, m.resultErr = "", nil // 清缓存，强制重新生成（如 UUID）
			}
			cmd = m.compute()
		case "ctrl+u":
			m.clear()
		case "ctrl+p":
			cmd = m.copyResult() // 复制当前结果到剪贴板
		}
		return cmd
	}
	return nil
}

// handleInputKey 处理输入框活跃时的按键
// ctrl+o 提交、esc 取消、ctrl+↑↓ 浏览最近记录，其余交给 textarea
func (m *Model) handleInputKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+o":
		return m.submitInput()
	case "esc":
		m.cancelInput()
		return nil
	case "ctrl+up":
		m.cycleRecent(-1)
		return nil
	case "ctrl+down":
		m.cycleRecent(1)
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

// openInput 打开多行输入框，可选预填内容
func (m *Model) openInput(prefill string) tea.Cmd {
	m.initTextarea()
	if prefill != "" {
		m.input.SetValue(prefill)
	}
	m.recentIdx = -1
	return m.input.Focus()
}

// submitInput 确认输入并重算；非空输入记入最近列表
func (m *Model) submitInput() tea.Cmd {
	value := strings.TrimSpace(m.input.Value())
	m.input.Blur()
	m.recentIdx = -1
	m.inputValue = value
	if value != "" {
		m.recent = ui.AddToRecent(m.recent, value, 10)
	}
	return m.compute()
}

// cancelInput 取消输入：关闭输入框但不改动已确认的 inputValue
func (m *Model) cancelInput() {
	m.input.Blur()
	m.recentIdx = -1
}

// cycleRecent 用 ctrl+↑↓ 循环浏览最近记录并预填输入框
func (m *Model) cycleRecent(delta int) {
	n := len(m.recent)
	if n == 0 {
		return
	}
	m.recentIdx += delta
	if m.recentIdx < 0 {
		m.recentIdx = n - 1
	}
	if m.recentIdx >= n {
		m.recentIdx = 0
	}
	m.input.SetValue(m.recent[m.recentIdx])
}

// compute 用当前输入重算当前工具结果；对异步工具返回启动命令
func (m *Model) compute() tea.Cmd {
	m.outScroll = 0
	m.copied = false // 结果已变化，清除复制提示
	if m.cursor >= len(m.visible) {
		m.result, m.resultErr = "", nil
		return nil
	}
	t := m.visible[m.cursor]
	if m.inputValue == "" {
		if !t.NoInput { // 普通工具空输入不计算
			m.result, m.resultErr = "", nil
			return nil
		}
		if m.result != "" { // NoInput 工具结果已生成则跳过，避免移动光标反复生成
			return nil
		}
	}
	// 异步工具：登记 loading + 递增请求号，在 goroutine 中执行，完成后发 httpResultMsg
	if t.Async {
		m.loading = true
		m.result, m.resultErr = "", nil
		m.reqID++
		id := m.reqID
		input := m.inputValue
		return func() tea.Msg {
			res, err := t.Run(input)
			return httpResultMsg{reqID: id, result: res, err: err}
		}
	}
	m.result, m.resultErr = t.Run(m.inputValue)
	return nil
}

// clear 清空输入与结果
func (m *Model) clear() {
	m.inputValue = ""
	m.result, m.resultErr = "", nil
	m.outScroll = 0
	m.loading = false // 异步请求进行中则一并取消（过期响应会被 reqID 丢弃）
	m.copied = false  // 清空时清除复制提示
}

// copyResult 复制当前工具结果到剪贴板（ctrl+p）
// 复制内存中的 m.result，不受屏幕换行/布局影响；请求中或空结果时静默跳过
func (m *Model) copyResult() tea.Cmd {
	m.copied = false
	if m.loading || m.result == "" {
		return nil
	}
	if err := clipboard.WriteAll(m.result); err == nil {
		m.copied = true
	}
	return nil
}

// switchSection 切换功能区并重置光标
func (m *Model) switchSection(delta int) tea.Cmd {
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
	return m.compute()
}

// moveCursor 移动工具光标并自动重算
func (m *Model) moveCursor(delta int) tea.Cmd {
	total := len(m.visible)
	if total == 0 {
		return nil
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	return m.compute()
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

	// 输入框打开时渲染多行输入卡片（textarea 自带光标/滚动/多行编辑）
	if m.input.Focused() {
		m.input.SetWidth(cardWidth - 4) // 卡片边框 2 + 左右 padding 2
		taH := m.height - 7
		if taH < 3 {
			taH = 3
		}
		m.input.SetHeight(taH - 2) // 底部预留快捷键提示行
		content := m.input.View()
		hint := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  ctrl+o 提交 · esc 取消 · ctrl+↑↓ 最近记录")
		return ui.Card("DevTools", content+"\n"+hint, ui.ColAccent, cardWidth)
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

	// Output 标题附加 ctrl+p 复制提示；复制成功后显示绿色"✓ 已复制"
	outTitle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Output")
	if m.copied {
		outTitle += lipgloss.NewStyle().Foreground(ui.ColGreen).Render("  ✓ 已复制")
	} else {
		outTitle += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  ctrl+p 复制")
	}
	right = append(right, outTitle)
	switch {
	case m.loading: // 异步工具请求进行中
		right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  ⏳ sending…"))
	case m.inputValue == "" && !m.isNoInput():
		right = append(right, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (empty)"))
	default:
		outText := m.result
		outColor := ui.ColText
		if m.resultErr != nil {
			outText, outColor = m.resultErr.Error(), ui.ColRed
		}
		// 幂等 wrapLong：只硬切超宽单行，避免对 Run 端已分段结果二次切出空行
		outLines := strings.Split(lipgloss.NewStyle().Width(rightW-2).Render(wrapLong(outText)), "\n")
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
		for i, l := range outLines[m.outScroll:outEnd] {
			pre := "  "
			if m.resultErr != nil {
				if m.outScroll+i == 0 {
					pre = "  ✗ " // 错误只标文本第一行
				} else {
					pre = "    " // 续行对齐
				}
			}
			right = append(right, lipgloss.NewStyle().Foreground(outColor).Render(pre+l))
		}
	}
	for len(right) < contentH {
		right = append(right, "")
	}
	rightCol := strings.Join(right[:contentH], "\n")

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
	return ui.Card("DevTools", content, ui.ColAccent, cardWidth)
}
