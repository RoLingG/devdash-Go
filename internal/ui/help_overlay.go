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
	}

	items := map[TabState][]HelpItem{
		TabGit: {
			{"↑↓", "scroll commits"}, {"/", "change repo"}, {"R", "refresh"},
			{"?", "toggle help"}, {"1/2/3/4", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+q", "quit"},
		},
		TabLog: {
			{"↑↓", "scroll lines"}, {"/", "open path"}, {"type", "filter"},
			{"Esc", "clear filter"}, {"R", "refresh"},
			{"?", "toggle help"}, {"1/2/3/4", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+q", "quit"},
		},
		TabWeather: {
			{"↑↓", "scroll content"}, {"/", "change city"}, {"R", "refresh"},
			{"?", "toggle help"}, {"1/2/3/4", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+q", "quit"},
		},
		TabConfig: {
			{"↑↓", "move cursor"}, {"enter", "toggle node"}, {"/", "open file"},
			{"type", "search filter"}, {"Esc", "clear search"},
			{"?", "toggle help"}, {"1/2/3/4", "switch tab"},
			{"ctrl+s", "save config"}, {"ctrl+q", "quit"},
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
