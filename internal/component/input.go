package component

import (
	"strings"

	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
)

// InputModel 通用文本输入组件
type InputModel struct {
	Active bool
	Prompt string
	Value  string
	Cursor int

	RecentItems []string
	recentIdx   int
}

// SetRecent 设置最近记录列表
func (m *InputModel) SetRecent(items []string) {
	m.RecentItems = items
}

// Confirm 返回用户输入的值，并关闭输入模式
func (m *InputModel) Confirm() string {
	m.Active = false
	m.recentIdx = -1
	return strings.TrimSpace(m.Value)
}

// Cancel 取消输入，清空内容并关闭输入模式
func (m *InputModel) Cancel() {
	m.Active = false
	m.Value = ""
	m.Cursor = 0
	m.recentIdx = -1
}

// Open 打开输入模式，可选预填内容
func (m *InputModel) Open(prefill string) {
	m.Active = true
	m.Value = prefill
	m.Cursor = ui.RuneLen(prefill)
	m.recentIdx = -1
}

// Reset 完全重置输入组件状态
func (m *InputModel) Reset() {
	m.Active = false
	m.Value = ""
	m.Cursor = 0
	m.recentIdx = -1
}

// RecentIdx 返回当前选中的最近记录索引
func (m *InputModel) RecentIdx() int {
	if len(m.RecentItems) == 0 {
		return -1
	}
	return m.recentIdx
}

// Update 统一处理输入模式下的所有按键消息。
// onSubmit 返回 func() tea.Msg 而非 tea.Cmd，避免 component 包依赖 bubbletea。
func (m *InputModel) Update(msg tea.Msg, onSubmit func(value string) func() tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.Value = ui.RuneInsert(m.Value, msg.Content, m.Cursor)
		m.Cursor += ui.RuneLen(msg.Content)
		m.recentIdx = -1
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			v := m.Confirm()
			if onSubmit != nil {
				return onSubmit(v)
			}
			return nil
		case "esc":
			m.Cancel()
		case "ctrl+u":
			m.Value = ""
			m.Cursor = 0
			m.recentIdx = -1
		case "up":
			// 有最近记录时，向下遍历
			if len(m.RecentItems) > 0 && m.recentIdx > 0 {
				m.recentIdx--
				m.Value = m.RecentItems[m.recentIdx]
				m.Cursor = ui.RuneLen(m.Value)
			} else if m.recentIdx == 0 {
				m.recentIdx = -1
				m.Value = ""
				m.Cursor = 0
			}
		case "down":
			// 有最近记录时，向上遍历
			if len(m.RecentItems) > 0 {
				if m.recentIdx < len(m.RecentItems)-1 {
					m.recentIdx++
				}
				m.Value = m.RecentItems[m.recentIdx]
				m.Cursor = ui.RuneLen(m.Value)
			}
		case "left":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "right":
			if m.Cursor < ui.RuneLen(m.Value) {
				m.Cursor++
			}
		case "home":
			m.Cursor = 0
		case "end":
			m.Cursor = ui.RuneLen(m.Value)
		case "backspace":
			if m.Cursor > 0 {
				m.Value = ui.RuneDeleteAt(m.Value, m.Cursor-1)
				m.Cursor--
				m.recentIdx = -1
			}
		case "delete":
			if m.Cursor < ui.RuneLen(m.Value) {
				m.Value = ui.RuneDeleteAt(m.Value, m.Cursor)
				m.recentIdx = -1
			}
		default:
			key := msg.String()
			if len(key) == 1 && key >= " " {
				m.Value = ui.RuneInsert(m.Value, key, m.Cursor)
				m.Cursor++
				m.recentIdx = -1
			}
		}
	}
	return nil
}
