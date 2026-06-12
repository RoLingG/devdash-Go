package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderTabBar 渲染顶部 Tab 栏
func RenderTabBar(active TabState, width int) string {
	labels := []string{"1: Git", "2: Log", "3: Weather", "4: Config"}
	var parts []string
	for i, label := range labels {
		if TabState(i) == active {
			parts = append(parts, styleTabActive.Render(label))
		} else {
			parts = append(parts, styleTabInactive.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	gap := width - lipgloss.Width(bar)
	if gap > 0 {
		bar += lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(strings.Repeat(" ", gap))
	}
	return bar
}

// helpLine 一行快捷键提示，每项固定列宽对齐后用分隔符连接
const helpColW = 16 // 每列固定宽度，确保上下两行的 • 纵向对齐

func helpLine(parts []string, width int, sep string) string {
	for i, p := range parts {
		parts[i] = PadRight(p, helpColW)
	}
	h := " " + strings.Join(parts, sep) + " "
	styled := StyleHelp.Render(h)
	gap := width - lipgloss.Width(styled)
	if gap > 0 {
		styled += StyleHelp.Render(strings.Repeat(" ", gap))
	}
	return styled
}

// RenderStatusBar 渲染底部双行状态栏（nano 风格）
// line1: 模块专属操作
// line2: 全局操作
func RenderStatusBar(active TabState, width int) string {
	sep := "  \u2022  " // • 分隔符

	switch active {
	case TabGit:
		line1 := helpLine([]string{"\u2191\u2193 scroll", "/ repo", "ctrl+r refresh", "? help"}, width, sep)
		line2 := helpLine([]string{"1/2/3/4 switch", "ctrl+s save", "ctrl+q quit"}, width, sep)
		return line1 + "\n" + line2

	case TabLog:
		line1 := helpLine([]string{"\u2191\u2193 cursor", "[ ] page", "ctrl+p jump", "ctrl+l level", "ctrl+f follow"}, width, sep)
		line2 := helpLine([]string{"type filter", "/ open", "ctrl+u clear", "Esc close", "ctrl+q quit"}, width, sep)
		return line1 + "\n" + line2

	case TabWeather:
		line1 := helpLine([]string{"\u2191\u2193 scroll", "/ city", "ctrl+r refresh", "? help"}, width, sep)
		line2 := helpLine([]string{"1/2/3/4 switch", "ctrl+s save", "ctrl+q quit"}, width, sep)
		return line1 + "\n" + line2

	case TabConfig:
		line1 := helpLine([]string{"\u2191\u2193 move", "enter toggle", "/ open", "ctrl+n/b match", "ctrl+e/w expand/collapse"}, width, sep)
		line2 := helpLine([]string{"ctrl+u clear", "Esc close", "? help", "1/2/3/4 switch", "ctrl+q quit"}, width, sep)
		return line1 + "\n" + line2
	}
	return ""
}
