package ui

import (
	"strings"
	"testing"
)

// TestRenderSplash 测试闪屏渲染
func TestRenderSplash(t *testing.T) {
	// 大终端：包含版本、模块数和提示语，以及 ASCII logo
	got := RenderSplash(100, 30, "v1.2.3", 8)
	for _, want := range []string{"v1.2.3", "8 modules", "Press any key"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderSplash 缺少 %q", want)
		}
	}
	if !strings.Contains(got, "██") {
		t.Errorf("大终端应包含 ASCII logo: %q", got)
	}

	// 窄终端回退为 D E V D A S H
	narrow := RenderSplash(40, 20, "v1", 8)
	if !strings.Contains(narrow, "D E V D A S H") {
		t.Errorf("窄终端应回退为 D E V D A S H: %q", narrow)
	}

	// 高度不足（小于内容行数）时垂直填充 clamp 为 0，不 panic
	_ = RenderSplash(80, 2, "v1", 8)

	// 返回行数应 >= 内容行（垂直填充 + 6 logo 行 + 空行 + 副标题 + 空行 + 提示）
	lines := strings.Split(got, "\n")
	if len(lines) < 10 {
		t.Errorf("RenderSplash 行数过少: %d", len(lines))
	}
}
