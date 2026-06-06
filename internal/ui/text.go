package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TrueWidth 修正 lipgloss.Width 对带变体选择器 emoji（如 ⚙️）的宽度偏差
// lipgloss.Width("⚙️") 返回 1，但终端渲染占 2 列，此函数修正该偏差
func TrueWidth(s string) int {
	w := lipgloss.Width(s)
	runes := []rune(s)
	for i, r := range runes {
		if r == '\uFE0F' && i > 0 {
			// 变体选择器 16 让前一个字符以 emoji 样式渲染（+1 列）
			w++
		}
	}
	return w
}

// ForceTruncate 按 rune 遍历强制截断到指定列宽
func ForceTruncate(s string, max int) string {
	w := 0
	for i, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > max {
			return s[:i]
		}
		w += rw
	}
	return s
}

// Truncate 截断字符串（基于可见宽度，忽略 ANSI 码）
func Truncate(s string, max int) string {
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	for i := 1; i <= len(runes); i++ {
		if lipgloss.Width(string(runes[:i]))+3 > max {
			if i > 1 {
				return string(runes[:i-1]) + "..."
			}
			return "..."
		}
	}
	return s
}

// PadRight 右侧填充空格（基于可见宽度，忽略 ANSI 码）
func PadRight(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

// ---- rune 级别字符串编辑工具 ----

// RuneToByteIdx 将 rune 索引转为 byte 索引
func RuneToByteIdx(s string, runeIdx int) int {
	i := 0
	for bi := range s {
		if i >= runeIdx {
			return bi
		}
		i++
	}
	return len(s)
}

// RuneInsert 在指定 rune 索引处插入字符串
func RuneInsert(s, insert string, runeIdx int) string {
	bi := RuneToByteIdx(s, runeIdx)
	return s[:bi] + insert + s[bi:]
}

// RuneDeleteAt 删除指定 rune 索引处的一个字符
func RuneDeleteAt(s string, runeIdx int) string {
	runes := []rune(s)
	if runeIdx < 0 || runeIdx >= len(runes) {
		return s
	}
	return string(append(runes[:runeIdx], runes[runeIdx+1:]...))
}

// RuneLen 返回字符串的 rune 数量
func RuneLen(s string) int {
	return len([]rune(s))
}

// RuneSubstr 按 rune 索引截取子串
func RuneSubstr(s string, start, end int) string {
	runes := []rune(s)
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}
