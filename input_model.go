// ============================================================================
// input_model.go — 通用输入组件
//
// 从 mod_git / mod_config / mod_log 三处重复的输入逻辑中抽取而来。
// 各模块持有 input inputModel 字段，输入模式激活时将消息转发给它。
// ============================================================================

package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// inputModel 通用文本输入组件
type inputModel struct {
	active bool   // 是否处于输入模式
	prompt string // 输入提示文字，如 "Repository path:" / "Filter:"
	value  string // 当前输入内容
	cursor int    // 光标位置（rune 索引）
}

// Confirm 返回用户输入的值，并关闭输入模式
func (m *inputModel) Confirm() string {
	m.active = false
	return strings.TrimSpace(m.value)
}

// Cancel 取消输入，清空内容并关闭输入模式
func (m *inputModel) Cancel() {
	m.active = false
	m.value = ""
	m.cursor = 0
}

// Open 打开输入模式，可选预填内容
func (m *inputModel) Open(prefill string) {
	m.active = true
	m.value = prefill
	m.cursor = runeLen(prefill)
}

// Reset 完全重置输入组件状态
func (m *inputModel) Reset() {
	m.active = false
	m.value = ""
	m.cursor = 0
}

// Update 统一处理输入模式下的所有按键消息。
// Enter 确认时返回 onSubmit 回调产生的 tea.Cmd，其他情况返回 nil。
func (m *inputModel) Update(msg tea.Msg, onSubmit func(value string) tea.Cmd) tea.Cmd {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.value = runeInsert(m.value, msg.Content, m.cursor)
		m.cursor += runeLen(msg.Content)
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
			m.value = ""
			m.cursor = 0
		case "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right":
			if m.cursor < runeLen(m.value) {
				m.cursor++
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = runeLen(m.value)
		case "backspace":
			if m.cursor > 0 {
				m.value = runeDeleteAt(m.value, m.cursor-1)
				m.cursor--
			}
		case "delete":
			if m.cursor < runeLen(m.value) {
				m.value = runeDeleteAt(m.value, m.cursor)
			}
		default:
			key := msg.String()
			if len(key) == 1 && key >= " " {
				m.value = runeInsert(m.value, key, m.cursor)
				m.cursor++
			}
		}
	}
	return nil
}
