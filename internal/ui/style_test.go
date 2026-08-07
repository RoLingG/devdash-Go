package ui

import (
	"image/color"
	"strings"
	"testing"
)

// restoreTheme 恢复默认暗色主题，避免污染其他测试
func restoreTheme() {
	SetTheme(darkTheme)
}

func TestSetTheme(t *testing.T) {
	defer restoreTheme()

	// 构造一个自定义主题
	custom := Theme{
		Primary: color.RGBA{1, 2, 3, 255}, Secondary: color.RGBA{4, 5, 6, 255},
		Accent: color.RGBA{7, 8, 9, 255}, Text: color.RGBA{10, 11, 12, 255},
		Muted: color.RGBA{13, 14, 15, 255}, Green: color.RGBA{16, 17, 18, 255},
		Red: color.RGBA{19, 20, 21, 255}, Blue: color.RGBA{22, 23, 24, 255},
		BgDark: color.RGBA{25, 26, 27, 255}, BgMid: color.RGBA{28, 29, 30, 255},
	}
	SetTheme(custom)

	// 全局颜色变量应跟随更新
	if ColPrimary != custom.Primary {
		t.Errorf("SetTheme 后 ColPrimary 未更新")
	}
	if ColAccent != custom.Accent || ColMuted != custom.Muted {
		t.Errorf("SetTheme 后部分颜色变量未更新")
	}
	if IsLightTheme() {
		t.Error("自定义主题不应判定为 light")
	}
}

func TestSetThemeByName(t *testing.T) {
	defer restoreTheme()

	SetThemeByName("light")
	if !IsLightTheme() {
		t.Error("SetThemeByName(\"light\") 后 IsLightTheme() 应为 true")
	}
	if ColAccent != lightTheme.Accent {
		t.Error("light 主题下 ColAccent 应使用 lightTheme 值")
	}

	// 其他名称一律回退暗色
	SetThemeByName("unknown")
	if IsLightTheme() {
		t.Error("未知主题名应回退到 dark")
	}
	if ColAccent != darkTheme.Accent {
		t.Error("dark 主题下 ColAccent 应使用 darkTheme 值")
	}

	SetThemeByName("light")
	SetThemeByName("dark")
	if IsLightTheme() {
		t.Error("\"dark\" 应切回暗色")
	}
}

func TestStyleFunctions(t *testing.T) {
	defer restoreTheme()

	// 各样式函数应返回可用的样式，Render 不会丢失内容
	// 注意：lipgloss v2 的 Render 是可变参数 Render(...string) string
	check := func(name string, s interface{ Render(...string) string }) {
		if got := s.Render("test"); !strings.Contains(got, "test") {
			t.Errorf("%s Render 丢失内容: %q", name, got)
		}
	}
	check("StyleTabActive", StyleTabActive())
	check("StyleTabInactive", StyleTabInactive())
	check("StyleTitle", StyleTitle())
	check("StyleBox", StyleBox())
	check("StyleHelpBar", StyleHelpBar())
}

func TestTabState(t *testing.T) {
	// 8 个 Tab 枚举连续递增
	if TabRoute != TabGit+7 {
		t.Errorf("TabRoute = %d, want %d", TabRoute, TabGit+7)
	}
	states := []TabState{TabGit, TabLog, TabWeather, TabConfig, TabSystem, TabPorts, TabLinuxDo, TabRoute}
	for i, s := range states {
		if s != TabState(i) {
			t.Errorf("TabState(%d) 顺序错误: got %d", i, s)
		}
	}
}
