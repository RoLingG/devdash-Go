package ui

import "github.com/charmbracelet/lipgloss"

// ---- 主题系统 ----

// Theme 包含所有可切换的颜色值
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Text      lipgloss.Color
	Muted     lipgloss.Color
	Green     lipgloss.Color
	Red       lipgloss.Color
	Blue      lipgloss.Color
	BgDark    lipgloss.Color
	BgMid     lipgloss.Color
}

var darkTheme = Theme{
	Primary: "205", Secondary: "62", Accent: "226",
	Text: "252", Muted: "243", Green: "82", Red: "196", Blue: "39",
	BgDark: "235", BgMid: "237",
}

var lightTheme = Theme{
	Primary: "133", Secondary: "68", Accent: "178",
	Text: "238", Muted: "245", Green: "28", Red: "160", Blue: "25",
	BgDark: "252", BgMid: "250",
}

var currentTheme = darkTheme

var (
	ColPrimary   = darkTheme.Primary
	ColSecondary = darkTheme.Secondary
	ColAccent    = darkTheme.Accent
	ColText      = darkTheme.Text
	ColMuted     = darkTheme.Muted
	ColGreen     = darkTheme.Green
	ColRed       = darkTheme.Red
	ColBlue      = darkTheme.Blue
	ColBgDark    = darkTheme.BgDark
	ColBgMid     = darkTheme.BgMid
)

// SetTheme 切换主题，更新所有颜色变量
func SetTheme(t Theme) {
	currentTheme = t
	ColPrimary = t.Primary
	ColSecondary = t.Secondary
	ColAccent = t.Accent
	ColText = t.Text
	ColMuted = t.Muted
	ColGreen = t.Green
	ColRed = t.Red
	ColBlue = t.Blue
	ColBgDark = t.BgDark
	ColBgMid = t.BgMid
}

// SetThemeByName 按名称切换主题
func SetThemeByName(name string) {
	if name == "light" {
		SetTheme(lightTheme)
	} else {
		SetTheme(darkTheme)
	}
}

// IsLightTheme 返回当前是否浅色主题
func IsLightTheme() bool {
	return currentTheme == lightTheme
}

// ---- Tab 类型 ----

type TabState int

const (
	TabGit TabState = iota
	TabLog
	TabWeather
	TabConfig
	TabSystem
	TabPorts
	TabLinuxDo
)

// StyleTabActive 激活状态下的 Tab 样式
func StyleTabActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColBgDark).
		Background(ColPrimary).
		Padding(0, 2)
}

// StyleTabInactive 未激活状态下的 Tab 样式
func StyleTabInactive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColMuted).
		Background(ColBgMid).
		Padding(0, 2)
}

// StyleTitle 模块标题样式
func StyleTitle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColPrimary).
		MarginBottom(1)
}

// StyleBox 卡片盒样式
func StyleBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColSecondary).
		Padding(0, 1)
}

// StyleHelpBar 帮助面板样式
func StyleHelpBar() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColMuted).
		Background(ColBgMid).
		Padding(0, 1)
}
