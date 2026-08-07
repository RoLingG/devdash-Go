package ui

import (
	"strings"
	"testing"
)

func TestRenderHelpOverlay(t *testing.T) {
	// 每个 Tab 的帮助面板都应渲染标题和内容
	for _, ts := range []TabState{TabGit, TabLog, TabWeather, TabConfig, TabSystem, TabPorts, TabLinuxDo, TabRoute} {
		got := RenderHelpOverlay(ts, 80, 30)
		if got == "" {
			t.Errorf("Tab %d 帮助面板为空", ts)
		}
		// 内容应包含若干快捷键项
		if !strings.Contains(got, "→") {
			t.Errorf("Tab %d 帮助面板缺少快捷键分隔符", ts)
		}
	}

	// 大终端：内容应完整包含标题
	got := RenderHelpOverlay(TabGit, 100, 50)
	if !strings.Contains(got, "Git 快捷键") {
		t.Errorf("帮助面板缺少标题: %q", got)
	}
	if !strings.Contains(got, "scroll commits") {
		t.Errorf("帮助面板缺少快捷键描述: %q", got)
	}

	// 小终端：不 panic，宽度自适应
	_ = RenderHelpOverlay(TabLog, 20, 10)
	_ = RenderHelpOverlay(TabRoute, 20, 5)

	// 高度不足以容纳卡片时不 panic（verticalPad clamp 为 0）
	_ = RenderHelpOverlay(TabSystem, 40, 1)
}
