// ============================================================================
// ui.go — Tab 栏、共享样式、帮助栏
// ============================================================================

package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---- 共享颜色 ----

var (
	colPrimary   = lipgloss.Color("205") // 粉紫色（主色调）
	colSecondary = lipgloss.Color("62")  // 蓝紫色
	colAccent    = lipgloss.Color("226") // 亮黄色（强调）
	colText      = lipgloss.Color("252") // 浅灰（正文）
	colMuted     = lipgloss.Color("243") // 深灰（次要文字）
	colGreen     = lipgloss.Color("82")  // 绿色
	colRed       = lipgloss.Color("196") // 红色
	colBlue      = lipgloss.Color("39")  // 蓝色
)

// ---- 共享样式 ----

var (
	styleTabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("235")).
		Background(colPrimary).
		Padding(0, 2)

	styleTabInactive = lipgloss.NewStyle().
		Foreground(colMuted).
		Background(lipgloss.Color("237")).
		Padding(0, 2)

	styleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colPrimary).
		MarginBottom(1)

	styleBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colSecondary).
		Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
		Foreground(colMuted).
		Background(lipgloss.Color("237")).
		Padding(0, 1)
)

// renderTabBar 渲染顶部 Tab 栏
func renderTabBar(active tabState, width int) string {
	labels := []string{"1: Git", "2: Log", "3: Weather", "4: Config"}
	var parts []string
	for i, label := range labels {
		if tabState(i) == active {
			parts = append(parts, styleTabActive.Render(label))
		} else {
			parts = append(parts, styleTabInactive.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	// 用背景色填充剩余宽度
	gap := width - lipgloss.Width(bar)
	if gap > 0 {
		bar += lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(strings.Repeat(" ", gap))
	}
	return bar
}

// renderHelp 渲染底部帮助栏
func renderHelp(active tabState, width int) string {
	helps := map[tabState]string{
		tabGit:     " ↑↓ scroll  •  / change repo  •  R refresh  •  1/2/3/4 switch  •  ctrl+q quit",
		tabLog:     " ↑↓ scroll  •  / open path  •  type to filter  •  Esc clear  •  R refresh  •  1/2/3/4 switch  •  ctrl+q quit",
		tabWeather: " ↑↓ scroll  •  R refresh  •  / change city  •  1/2/3/4 switch  •  ctrl+q quit",
		tabConfig:  " ↑↓ move  •  enter toggle  •  / open file  •  type to search  •  Esc clear  •  1/2/3/4 switch  •  ctrl+q quit",
	}
	h := styleHelp.Render(helps[active])
	gap := width - lipgloss.Width(h)
	if gap > 0 {
		h += styleHelp.Render(strings.Repeat(" ", gap))
	}
	return h
}

// box 用边框包裹内容并加标题
func box(title, content string, width int) string {
	header := styleTitle.Render(title)
	// 强制截断每行到内容区宽度，防止溢出边框3
	maxW := width - 4 // border(2) + padding(2)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > maxW {
			lines[i] = forceTruncate(line, maxW)
		}
	}
	return styleBox.Width(maxW).Render(header + "\n" + strings.Join(lines, "\n"))
}

// card 创建一个带边框的卡片
func card(title, content string, borderColor lipgloss.Color, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(borderColor).
		MarginBottom(1)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	return cardStyle.Render(titleStyle.Render(title) + "\n" + content)
}

// forceTruncate 按 rune 遍历强制截断到指定列宽
func forceTruncate(s string, max int) string {
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

// truncate 截断字符串（基于可见宽度，忽略 ANSI 码）
func truncate(s string, max int) string {
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	// 找到截断点：加上 "..." 后总宽度不超过 max
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

// padRight 右侧填充空格（基于可见宽度，忽略 ANSI 码）
func padRight(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

// barChart 绘制水平柱状图的一行
func barChart(label string, value, maxValue, barMaxWidth int, barColor lipgloss.Color) string {
	name := padRight(truncate(label, 20), 20)
	barLen := 1
	if maxValue > 0 {
		barLen = int(float64(value) / float64(maxValue) * float64(barMaxWidth))
		if barLen < 1 {
			barLen = 1
		}
	}
	bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", barLen))
	count := lipgloss.NewStyle().Foreground(colMuted).Render(fmt.Sprintf(" %d", value))
	return fmt.Sprintf("  %s %s%s", name, bar, count)
}

// ---- rune 级别字符串编辑工具 ----

// runeToByteIdx 将 rune 索引转为 byte 索引（用于字符串切片）
func runeToByteIdx(s string, runeIdx int) int {
	i := 0
	for bi := range s {
		if i >= runeIdx {
			return bi
		}
		i++
	}
	return len(s)
}

// runeInsert 在指定 rune 索引处插入字符串
func runeInsert(s, insert string, runeIdx int) string {
	bi := runeToByteIdx(s, runeIdx)
	return s[:bi] + insert + s[bi:]
}

// runeDeleteAt 删除指定 rune 索引处的一个字符
func runeDeleteAt(s string, runeIdx int) string {
	runes := []rune(s)
	if runeIdx < 0 || runeIdx >= len(runes) {
		return s
	}
	return string(append(runes[:runeIdx], runes[runeIdx+1:]...))
}

// runeLen 返回字符串的 rune 数量
func runeLen(s string) int {
	return len([]rune(s))
}

// runeSubstr 按 rune 索引截取子串
func runeSubstr(s string, start, end int) string {
	runes := []rune(s)
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}
