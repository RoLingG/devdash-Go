package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpItem 一个快捷键说明项
type HelpItem struct {
	Key  string
	Desc string
}

// RenderHelpOverlay 渲染帮助面板（居中卡片）
func RenderHelpOverlay(active TabState, width, height int) string {
	title := map[TabState]string{
		TabGit:     "Git 快捷键",
		TabLog:     "Log 快捷键",
		TabWeather: "Weather 快捷键",
		TabConfig:  "Config 快捷键",
		TabSystem:  "System 快捷键",
		TabPorts:   "Ports 快捷键",
		TabLinuxDo: "LinuxDo 快捷键",
		TabRoute:   "Route 快捷键",
	}

	items := map[TabState][]HelpItem{
		TabGit: {
			{"↑/↓", "scroll commits"}, {"Home/End", "first/last"}, {"/", "change repo"}, {"ctrl+r", "refresh"},
			{"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabLog: {
			{"↑↓", "cursor in page"}, {"Home/End", "first/last"}, {"[ ]", "prev/next page"},
			{"ctrl+↑/↓", "fast ±10 pages"}, {"ctrl+p", "jump to page"},
			{"/", "open path"}, {"type", "filter"}, {"ctrl+l", "level filter"},
			{"ctrl+u", "clear filter"}, {"ctrl+f", "follow mode"},
			{"ctrl+r", "refresh"}, {"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabWeather: {
			{"↑/↓", "scroll content"}, {"Home/End", "first/last"}, {"/", "change city"}, {"ctrl+r", "refresh"},
			{"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabConfig: {
			{"↑/↓", "move cursor"}, {"Home/End", "first/last"}, {"enter", "toggle node"}, {"/", "open file"},
			{"ctrl+r", "refresh"}, {"type", "search filter"}, {"ctrl+n/b", "next/prev match"},
			{"ctrl+u", "clear search"}, {"ctrl+e", "expand all"}, {"ctrl+w", "collapse all"},
			{"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabSystem: {
			{"↑/↓", "scroll"}, {"Home/End", "first/last"}, {"tab", "switch view"}, {"/", "filter process"},
			{"ctrl+r", "refresh"}, {"ctrl+u", "clear filter"},
			{"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabPorts: {
			{"↑/↓", "scroll"}, {"Home/End", "first/last"}, {"/", "add port"},
			{"ctrl+r", "rescan"},
			{"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabLinuxDo: {
			{"↑/↓", "scroll"}, {"^ctrl+↑/↓", "scroll ±10"}, {"Home/End", "first/last"}, {"enter", "open topic"},
			{"/", "set cookie"}, {"ctrl+f", "search"}, {"ctrl+r", "refresh"},
			{"ctrl+u", "clear cookie"}, {"?", "toggle help"}, {"Esc", "back/close"},
			{"1-8", "switch tab"}, {"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
		TabRoute: {
			{"↑/↓", "scroll"}, {"Home/End", "first/last"}, {"Tab", "interfaces"},
			{"ctrl+d", "delete route"}, {"ctrl+a", "add route"}, {"ctrl+r", "refresh"},
			{"ctrl+s", "save routes"}, {"ctrl+l", "load routes"},
			{"?", "toggle help"}, {"Esc", "close"}, {"1-8", "switch tab"},
			{"ctrl+t", "toggle theme"}, {"ctrl+q", "quit"},
		},
	}

	// 构建内容
	var sb strings.Builder
	keyStyle := lipgloss.NewStyle().Foreground(ColAccent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColText)
	sep := lipgloss.NewStyle().Foreground(ColMuted).Render("  →  ")

	for _, item := range items[active] {
		line := fmt.Sprintf("  %s%s%s", keyStyle.Render(PadRight(item.Key, 12)), sep, descStyle.Render(item.Desc))
		sb.WriteString(line + "\n")
	}

	// 添加关闭提示
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColMuted).Render("  Press ? to close"))

	// 计算卡片宽度（根据最长行）
	maxW := 0
	for _, line := range strings.Split(sb.String(), "\n") {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	cardWidth := maxW + 6
	if cardWidth < 36 {
		cardWidth = 36
	}
	maxCardW := width - 8
	if cardWidth > maxCardW {
		cardWidth = maxCardW
	}

	// 截断超宽行
	lines := strings.Split(sb.String(), "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > cardWidth {
			lines[i] = ForceTruncate(line, cardWidth)
		}
	}

	card := Card(title[active], strings.Join(lines, "\n"), ColAccent, cardWidth)

	// 水平垂直居中
	cardLines := strings.Split(card, "\n")
	cardHeight := len(cardLines)
	verticalPad := (height - cardHeight) / 2
	if verticalPad < 0 {
		verticalPad = 0
	}

	var centered []string
	for i := 0; i < verticalPad; i++ {
		centered = append(centered, strings.Repeat(" ", width))
	}
	for _, line := range cardLines {
		w := lipgloss.Width(line)
		leftPad := (width - w) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad := width - leftPad - w
		if rightPad < 0 {
			rightPad = 0
		}
		centered = append(centered, strings.Repeat(" ", leftPad)+line+strings.Repeat(" ", rightPad))
	}

	return strings.Join(centered, "\n")
}
