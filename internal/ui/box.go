package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Box 用边框包裹内容并加标题
func Box(title, content string, width int) string {
	header := StyleTitle.Render(title)
	maxW := width - 4
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > maxW {
			lines[i] = ForceTruncate(line, maxW)
		}
	}
	return StyleBox.Width(maxW).Render(header + "\n" + strings.Join(lines, "\n"))
}

// Card 创建一个带边框的卡片
func Card(title, content string, borderColor lipgloss.Color, width int) string {
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

// BarChart 绘制水平柱状图的一行
func BarChart(label string, value, maxValue, barMaxWidth int, barColor lipgloss.Color) string {
	name := PadRight(Truncate(label, 20), 20)
	barLen := 1
	if maxValue > 0 {
		barLen = int(float64(value) / float64(maxValue) * float64(barMaxWidth))
		if barLen < 1 {
			barLen = 1
		}
	}
	bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", barLen))
	count := lipgloss.NewStyle().Foreground(ColMuted).Render(fmt.Sprintf(" %d", value))
	return fmt.Sprintf("  %s %s%s", name, bar, count)
}

// InputCardOpts 输入框卡片的渲染参数
type InputCardOpts struct {
	Title       string   // 卡片标题
	Prompt      string   // 输入提示文字
	Value       string   // 当前输入内容
	Cursor      int      // 光标位置（rune 索引）
	ErrMsg      string   // 错误信息，为空则不显示
	CardWidth   int      // 卡片宽度
	RecentItems []string // 最近记录列表
	RecentIdx   int      // 当前选中的最近记录索引，-1 表示未选择
}

// RenderInputCard 渲染统一的输入框卡片
func RenderInputCard(opts InputCardOpts) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(ColText).Render("  " + opts.Prompt))
	sb.WriteString("\n")

	before := RuneSubstr(opts.Value, 0, opts.Cursor)
	after := RuneSubstr(opts.Value, opts.Cursor, RuneLen(opts.Value))
	inputLine := "  > " + before + lipgloss.NewStyle().Foreground(ColAccent).Render("|") + after
	sb.WriteString(lipgloss.NewStyle().Foreground(ColText).Render(inputLine))
	sb.WriteString("\n")

	// 渲染最近记录列表
	if len(opts.RecentItems) > 0 {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(ColMuted).Render("  Recent:"))
		sb.WriteString("\n")
		for i, item := range opts.RecentItems {
			marker := "  "
			style := lipgloss.NewStyle().Foreground(ColMuted)
			if i == opts.RecentIdx {
				marker = lipgloss.NewStyle().Foreground(ColAccent).Render("▸ ")
				style = lipgloss.NewStyle().Foreground(ColText).Bold(true)
			} else {
				marker = "  "
			}
			// 截断过长路径
			display := item
			maxW := opts.CardWidth - 8
			if maxW > 10 && RuneLen(display) > maxW {
				display = RuneSubstr(display, 0, maxW-3) + "..."
			}
			sb.WriteString(marker + style.Render(display))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	if opts.ErrMsg != "" {
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(ColRed).Render("✗ "+opts.ErrMsg))
		sb.WriteString("\n\n")
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(ColMuted).Render("  Enter confirm  ↑↓ recent  ←→ cursor  Home/End  ctrl+u clear  Esc close"))
	return Card(opts.Title, sb.String(), ColSecondary, opts.CardWidth)
}

// DirListCardOpts 目录列表卡片的渲染参数
type DirListCardOpts struct {
	Title   string
	DirPath string
	DirList interface {
		RenderWithFormat(maxShow int, formatter func(item string, selected bool) string) string
	}
	Height    int
	CardWidth int
	ErrMsg    string
}

// RenderDirListCard 渲染统一的目录列表卡片
func RenderDirListCard(opts DirListCardOpts) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(ColSecondary).Render("  📂 " + opts.DirPath))
	sb.WriteString("\n\n")

	maxShow := opts.Height - 10
	if maxShow < 5 {
		maxShow = 5
	}

	highlightStyle := lipgloss.NewStyle().Foreground(ColAccent)
	listContent := opts.DirList.RenderWithFormat(maxShow, func(item string, selected bool) string {
		if selected {
			return highlightStyle.Render("  > " + item)
		}
		return lipgloss.NewStyle().Foreground(ColText).Render("    " + item)
	})
	sb.WriteString(listContent)
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColMuted).Render("  ↑↓ select  Enter open  Esc back"))

	if opts.ErrMsg != "" {
		sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(ColRed).Render(opts.ErrMsg))
	}

	return Card(opts.Title, sb.String(), ColSecondary, opts.CardWidth)
}
