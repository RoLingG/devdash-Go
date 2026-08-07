package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderTabBar(t *testing.T) {
	// 激活的 Tab 应包含其标签
	got := RenderTabBar(TabLog, 100)
	for _, label := range []string{"1:Git", "2:Log", "3:Weather", "4:Config", "5:System", "6:Ports", "7:LinuxDo", "8:Route"} {
		if !strings.Contains(got, label) {
			t.Errorf("Tab 栏缺少 %q: %q", label, got)
		}
	}

	// 窄宽度下仍应包含所有标签（不足部分填充背景）
	gotNarrow := RenderTabBar(TabGit, 30)
	for _, label := range []string{"1:Git", "8:Route"} {
		if !strings.Contains(gotNarrow, label) {
			t.Errorf("窄 Tab 栏缺少 %q", label)
		}
	}

	// 宽度应 >= 传入宽度
	if lipgloss.Width(got) < 100 {
		t.Errorf("RenderTabBar 输出宽度 %d < 100", lipgloss.Width(got))
	}
}

func TestRenderStatusBar(t *testing.T) {
	// 每个 Tab 都应渲染出双行帮助栏
	for _, ts := range []TabState{TabGit, TabLog, TabWeather, TabConfig, TabSystem, TabPorts, TabLinuxDo, TabRoute} {
		got := RenderStatusBar(ts, 80)
		lines := strings.Split(got, "\n")
		if len(lines) != 2 {
			t.Errorf("Tab %d 状态栏应为 2 行，实际 %d 行: %q", ts, len(lines), got)
		}
		if !strings.Contains(got, "^Q Quit") && ts != TabSystem && ts != TabPorts {
			t.Errorf("Tab %d 状态栏应包含 ^Q Quit", ts)
		}
		if !strings.Contains(lines[0], "•") {
			t.Errorf("Tab %d 状态栏第一行应含分隔符", ts)
		}
	}

	// 未知 Tab 返回空字符串
	if got := RenderStatusBar(TabState(99), 80); got != "" {
		t.Errorf("未知 Tab 状态栏应为空，实际 %q", got)
	}
}

func TestHelpLine(t *testing.T) {
	// 多部分提示拼接后每项固定列宽 13，用分隔符连接
	got := helpLine([]string{"↑/↓", "Open", "Help"}, 60, "  •  ")
	if !strings.Contains(got, "↑/↓") || !strings.Contains(got, "Open") || !strings.Contains(got, "Help") {
		t.Errorf("helpLine 内容缺失: %q", got)
	}
	if lipgloss.Width(got) < 60 {
		t.Errorf("helpLine 输出宽度 %d < 60", lipgloss.Width(got))
	}

	// 宽度为 0 时不应 panic（gap 为负数时跳过填充）
	_ = helpLine([]string{"a", "b"}, 0, "  •  ")
}
