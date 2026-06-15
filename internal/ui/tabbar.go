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

const helpColW = 13

// helpLine 一行快捷键提示，每项固定列宽对齐后用分隔符连接
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

// RenderStatusBar 渲染底部双行状态栏
func RenderStatusBar(active TabState, width int) string {
	sep := "  \u2022  " // • 分隔符

	switch active {
	case TabGit:
		line1 := helpLine([]string{"\u2191\u2193 Scroll", "/ Open", "? Help"}, width, sep)
		line2 := helpLine([]string{"^S Save", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabLog:
		line1 := helpLine([]string{"\u2191\u2193 Scroll", "[ ] Page", "/ Open", "^P Jump", "^L Level", "^F Follow"}, width, sep)
		line2 := helpLine([]string{"Type Filter", "^U Clear", "Esc Close", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabWeather:
		line1 := helpLine([]string{"\u2191\u2193 Scroll", "/ Open", "? Help"}, width, sep)
		line2 := helpLine([]string{"^S Save", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabConfig:
		line1 := helpLine([]string{"\u2191\u2193 Scroll", "Enter Toggle", "/ Open", "^N/B Match", "^E/W All"}, width, sep)
		line2 := helpLine([]string{"Type Filter", "^U Clear", "Esc Close", "? Help", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2
	}
	return ""
}
