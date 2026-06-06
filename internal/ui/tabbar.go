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

// RenderHelp 渲染底部帮助栏
func RenderHelp(active TabState, width int) string {
	helps := map[TabState]string{
		TabGit:     " ↑↓ scroll  •  / change repo  •  R refresh  •  1/2/3/4 switch  •  ctrl+s save  •  ctrl+q quit",
		TabLog:     " ↑↓ scroll  •  / open path  •  type to filter  •  Esc clear  •  R refresh  •  1/2/3/4 switch  •  ctrl+s save  •  ctrl+q quit",
		TabWeather: " ↑↓ scroll  •  R refresh  •  / change city  •  1/2/3/4 switch  •  ctrl+s save  •  ctrl+q quit",
		TabConfig:  " ↑↓ move  •  enter toggle  •  / open file  •  type to search  •  Esc clear  •  1/2/3/4 switch  •  ctrl+s save  •  ctrl+q quit",
	}
	h := StyleHelp.Render(helps[active])
	gap := width - lipgloss.Width(h)
	if gap > 0 {
		h += StyleHelp.Render(strings.Repeat(" ", gap))
	}
	return h
}
