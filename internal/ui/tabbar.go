package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderTabBar 渲染顶部 Tab 栏
func RenderTabBar(active TabState, width int) string {
	labels := []string{"1:Git", "2:Log", "3:Weather", "4:Config", "5:System", "6:Ports", "7:LinuxDo", "8:Route"}
	var parts []string
	for i, label := range labels {
		if TabState(i) == active {
			parts = append(parts, StyleTabActive().Render(label))
		} else {
			parts = append(parts, StyleTabInactive().Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	gap := width - lipgloss.Width(bar)
	if gap > 0 {
		bar += lipgloss.NewStyle().Background(ColBgMid).Render(strings.Repeat(" ", gap))
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
	styled := StyleHelpBar().Render(h)
	gap := width - lipgloss.Width(styled)
	if gap > 0 {
		styled += StyleHelpBar().Render(strings.Repeat(" ", gap))
	}
	return styled
}

// RenderStatusBar 渲染底部双行状态栏
func RenderStatusBar(active TabState, width int) string {
	sep := "  \u2022  " // • 分隔符

	switch active {
	case TabGit:
		line1 := helpLine([]string{"↑/↓ Scroll", "/ Open", "? Help"}, width, sep)
		line2 := helpLine([]string{"^S Save", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabLog:
		line1 := helpLine([]string{"↑/↓ Scroll", "←/→ Page", "/ Open", "^P Jump", "^L Level", "^F Follow"}, width, sep)
		line2 := helpLine([]string{"Type Filter", "^U Clear", "Esc Close", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabWeather:
		line1 := helpLine([]string{"↑/↓ Scroll", "/ Open", "? Help"}, width, sep)
		line2 := helpLine([]string{"^S Save", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabConfig:
		line1 := helpLine([]string{"↑/↓ Scroll", "Enter Toggle", "/ Open", "^N/B Match", "^E/W All"}, width, sep)
		line2 := helpLine([]string{"Type Filter", "^U Clear", "Esc Close", "? Help", "^R Refresh", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabSystem:
		line1 := helpLine([]string{"↑/↓ Scroll", "Tab Switch", "/ Filter", "^R Refresh"}, width, sep)
		line2 := helpLine([]string{"^U Clear", "? Help", "1-8 Tab", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabPorts:
		line1 := helpLine([]string{"↑/↓ Scroll", "/ Add Port", "^R Refresh", "? Help"}, width, sep)
		line2 := helpLine([]string{"1-8 Tab", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2

	case TabLinuxDo:
		line1 := helpLine([]string{"↑/↓ Scroll", "^↑/↓ ±10", "Enter Open", "Esc Back"}, width, sep)
		line2 := helpLine([]string{"1-8 Tab", "^Q Quit", "/ Cookie", "^F Search", "^R Refresh"}, width, sep)
		return line1 + "\n" + line2

	case TabRoute:
		line1 := helpLine([]string{"↑/↓ Scroll", "Home/End", "^A Add Route", "^D Delete", "^S Save"}, width, sep)
		line2 := helpLine([]string{"1-8 Tab", "Tab Ifaces", "^R Refresh", "^L Load", "^Q Quit"}, width, sep)
		return line1 + "\n" + line2
	}
	return ""
}
