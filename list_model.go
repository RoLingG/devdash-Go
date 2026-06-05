// ============================================================================
// list_model.go — 通用列表组件
//
// 功能：
//   - 统一的列表状态管理（items、cursor）
//   - 上下移动、边界钳制
//   - 滚动窗口渲染（支持长列表）
//   - 高亮当前选中项
//
// 用于：
//   - Git 模块的仓库列表
//   - Log 模块的日志文件列表
//   - Config 模块的配置文件列表
// ============================================================================

package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// listModel 通用列表组件
type listModel struct {
	items  []string // 列表项
	cursor int      // 当前选中索引
}

// SetItems 设置列表项并重置 cursor
func (m *listModel) SetItems(items []string) {
	m.items = items
	m.cursor = 0
}

// MoveUp 向上移动光标
func (m *listModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown 向下移动光标
func (m *listModel) MoveDown() {
	if len(m.items) > 0 && m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

// Selected 返回当前选中项，列表为空返回空字符串
func (m *listModel) Selected() string {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return ""
	}
	return m.items[m.cursor]
}

// Len 返回列表项数量
func (m *listModel) Len() int {
	return len(m.items)
}

// Render 渲染列表，支持滚动窗口
// maxShow: 最大可见行数
// highlightStyle: 高亮当前选中行的样式
// prefix: 每行前缀（如 "  " 缩进）
func (m *listModel) Render(maxShow int, highlightStyle lipgloss.Style, prefix string) string {
	if len(m.items) == 0 {
		return ""
	}

	// 计算滚动窗口范围
	start := 0
	if m.cursor >= maxShow {
		start = m.cursor - maxShow + 1
	}
	end := start + maxShow
	if end > len(m.items) {
		end = len(m.items)
	}

	var sb strings.Builder
	for i, item := range m.items[start:end] {
		idx := start + i
		line := prefix + item
		if idx == m.cursor {
			// 高亮当前选中行
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
// formatter: 格式化每一项的函数，返回渲染后的字符串
func (m *listModel) RenderWithFormat(maxShow int, formatter func(item string, selected bool) string) string {
	if len(m.items) == 0 {
		return ""
	}

	// 计算滚动窗口范围
	start := 0
	if m.cursor >= maxShow {
		start = m.cursor - maxShow + 1
	}
	end := start + maxShow
	if end > len(m.items) {
		end = len(m.items)
	}

	var lines []string
	for i, item := range m.items[start:end] {
		idx := start + i
		selected := idx == m.cursor
		lines = append(lines, formatter(item, selected))
	}
	return strings.Join(lines, "\n")
}

// ScrollInfo 返回滚动信息，用于调试或状态栏显示
func (m *listModel) ScrollInfo() string {
	if len(m.items) == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", m.cursor+1, len(m.items))
}
