package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// RenderSplash 渲染闪屏内容
func RenderSplash(width, height int, version string, modules int) string {
	// ASCII Art Logo
	logoLines := []string{
		"██████╗ ███████╗██╗   ██╗██████╗  █████╗ ███████╗██╗  ██╗",
		"██╔══██╗██╔════╝██║   ██║██╔══██╗██╔══██╗██╔════╝██║  ██║",
		"██║  ██║█████╗  ██║   ██║██║  ██║███████║███████╗███████║",
		"██║  ██║██╔══╝  ╚██╗ ██╔╝██║  ██║██╔══██║╚════██║██╔══██║",
		"██████╔╝███████╗ ╚████╔╝ ██████╔╝██║  ██║███████║██║  ██║",
		"╚═════╝ ╚══════╝  ╚═══╝  ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝",
	}

	// 窄终端回退
	if width < 62 {
		logoLines = []string{
			lipgloss.NewStyle().Bold(true).Foreground(ColAccent).Render("D E V D A S H"),
		}
	}

	// 构建内容行
	styledLogo := make([]string, len(logoLines))
	for i, line := range logoLines {
		styledLogo[i] = lipgloss.NewStyle().Foreground(ColAccent).Render(line)
	}

	subtitle := lipgloss.NewStyle().Foreground(ColMuted).Render(
		fmt.Sprintf("%s | %d modules", version, modules))
	hint := lipgloss.NewStyle().Foreground(ColMuted).Render("Press any key to continue")

	contentLines := append(styledLogo, "")
	contentLines = append(contentLines, subtitle)
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, hint)

	// 垂直居中
	totalLines := len(contentLines)
	verticalPad := (height - totalLines) / 2
	if verticalPad < 0 {
		verticalPad = 0
	}

	var centered []string
	for i := 0; i < verticalPad; i++ {
		centered = append(centered, strings.Repeat(" ", width))
	}

	// 每行水平居中
	for _, line := range contentLines {
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
