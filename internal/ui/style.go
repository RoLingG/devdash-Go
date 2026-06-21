package ui

import "github.com/charmbracelet/lipgloss"

// ---- 共享颜色 ----

var (
	ColPrimary   = lipgloss.Color("205")
	ColSecondary = lipgloss.Color("62")
	ColAccent    = lipgloss.Color("226")
	ColText      = lipgloss.Color("252")
	ColMuted     = lipgloss.Color("243")
	ColGreen     = lipgloss.Color("82")
	ColRed       = lipgloss.Color("196")
	ColBlue      = lipgloss.Color("39")
)

// ---- Tab 类型 ----

type TabState int

const (
	TabGit     TabState = iota
	TabLog
	TabWeather
	TabConfig
	TabSystem
	TabPorts
	TabLinuxDo
)

// ---- 共享样式 ----

var (
	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("235")).
			Background(ColPrimary).
			Padding(0, 2)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(ColMuted).
				Background(lipgloss.Color("237")).
				Padding(0, 2)

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColPrimary).
			MarginBottom(1)

	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColSecondary).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColMuted).
			Background(lipgloss.Color("237")).
			Padding(0, 1)
)
