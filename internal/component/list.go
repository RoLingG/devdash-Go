package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// ListModel 通用列表组件
type ListModel struct {
	Items  []string
	Cursor int
}

// SetItems 设置列表项并重置 cursor
func (m *ListModel) SetItems(items []string) {
	m.Items = items
	m.Cursor = 0
}

// MoveUp 向上移动光标
func (m *ListModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

// MoveDown 向下移动光标
func (m *ListModel) MoveDown() {
	if len(m.Items) > 0 && m.Cursor < len(m.Items)-1 {
		m.Cursor++
	}
}

// Selected 返回当前选中项，列表为空返回空字符串
func (m *ListModel) Selected() string {
	if len(m.Items) == 0 || m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return ""
	}
	return m.Items[m.Cursor]
}

// Len 返回列表项数量
func (m *ListModel) Len() int {
	return len(m.Items)
}

// Render 渲染列表，支持滚动窗口
func (m *ListModel) Render(maxShow int, highlightStyle lipgloss.Style, prefix string) string {
	if len(m.Items) == 0 {
		return ""
	}
	start := 0
	if m.Cursor >= maxShow {
		start = m.Cursor - maxShow + 1
	}
	end := start + maxShow
	if end > len(m.Items) {
		end = len(m.Items)
	}
	var sb strings.Builder
	for i, item := range m.Items[start:end] {
		idx := start + i
		line := prefix + item
		if idx == m.Cursor {
			line = highlightStyle.Render(line)
		}
		sb.WriteString(line)
		if i < end-start-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// RenderWithFormat 渲染列表，支持自定义格式化函数
func (m *ListModel) RenderWithFormat(maxShow int, formatter func(item string, selected bool) string) string {
	if len(m.Items) == 0 {
		return ""
	}
	start := 0
	if m.Cursor >= maxShow {
		start = m.Cursor - maxShow + 1
	}
	end := start + maxShow
	if end > len(m.Items) {
		end = len(m.Items)
	}
	var lines []string
	for i, item := range m.Items[start:end] {
		idx := start + i
		selected := idx == m.Cursor
		lines = append(lines, formatter(item, selected))
	}
	return strings.Join(lines, "\n")
}

// ScrollInfo 返回滚动信息
func (m *ListModel) ScrollInfo() string {
	if len(m.Items) == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", m.Cursor+1, len(m.Items))
}
