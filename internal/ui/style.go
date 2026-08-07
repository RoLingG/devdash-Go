package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// ---- 主题系统 ----

// Theme 包含所有可切换的颜色值
// 修改: lipgloss v2 中 Color 从字符串类型变为函数, 字段类型改用标准库 image/color.Color
type Theme struct {
	Primary   color.Color
	Secondary color.Color
	Accent    color.Color
	Text      color.Color
	Muted     color.Color
	Green     color.Color
	Red       color.Color
	Blue      color.Color
	BgDark    color.Color
	BgMid     color.Color
}

var darkTheme = Theme{
	// 修改: v2 中 color.Color 是接口类型, 需用 lipgloss.Color() 函数把字符串转成具体颜色
	Primary: lipgloss.Color("205"), Secondary: lipgloss.Color("62"), Accent: lipgloss.Color("226"),
	Text: lipgloss.Color("252"), Muted: lipgloss.Color("243"), Green: lipgloss.Color("82"), Red: lipgloss.Color("196"), Blue: lipgloss.Color("39"),
	BgDark: lipgloss.Color("235"), BgMid: lipgloss.Color("237"),
}

var lightTheme = Theme{
	Primary: lipgloss.Color("133"), Secondary: lipgloss.Color("68"), Accent: lipgloss.Color("178"),
	Text: lipgloss.Color("238"), Muted: lipgloss.Color("245"), Green: lipgloss.Color("28"), Red: lipgloss.Color("160"), Blue: lipgloss.Color("25"),
	BgDark: lipgloss.Color("252"), BgMid: lipgloss.Color("250"),
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
	TabRoute
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
