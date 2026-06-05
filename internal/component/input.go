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
}

// Confirm 返回用户输入的值，并关闭输入模式
func (m *InputModel) Confirm() string {
	m.Active = false
	return strings.TrimSpace(m.Value)
}

// Cancel 取消输入，清空内容并关闭输入模式
func (m *InputModel) Cancel() {
	m.Active = false
	m.Value = ""
	m.Cursor = 0
}

// Open 打开输入模式，可选预填内容
func (m *InputModel) Open(prefill string) {
	m.Active = true
	m.Value = prefill
	m.Cursor = ui.RuneLen(prefill)
}

// Reset 完全重置输入组件状态
func (m *InputModel) Reset() {
	m.Active = false
	m.Value = ""
	m.Cursor = 0
}

// Update 统一处理输入模式下的所有按键消息。
// onSubmit 返回 func() tea.Msg 而非 tea.Cmd，避免 component 包依赖 bubbletea。
func (m *InputModel) Update(msg tea.Msg, onSubmit func(value string) func() tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.Value = ui.RuneInsert(m.Value, msg.Content, m.Cursor)
		m.Cursor += ui.RuneLen(msg.Content)
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
			}
		case "delete":
			if m.Cursor < ui.RuneLen(m.Value) {
				m.Value = ui.RuneDeleteAt(m.Value, m.Cursor)
			}
		default:
			key := msg.String()
			if len(key) == 1 && key >= " " {
				m.Value = ui.RuneInsert(m.Value, key, m.Cursor)
				m.Cursor++
			}
		}
	}
	return nil
}
