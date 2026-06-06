package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// ConfigSelectModel 配置选择界面模型
type ConfigSelectModel struct {
	choices  []string
	cursor   int
	selected bool
	cfg      AppConfig
	hasCfg   bool
	width    int
	height   int
}

// NewConfigSelectModel 创建配置选择界面
func NewConfigSelectModel(width, height int) ConfigSelectModel {
	hasCfg := ConfigExists()
	var cfg AppConfig
	if hasCfg {
		cfg, _ = LoadConfig()
	}

	choices := []string{
		"使用默认配置（临时）",
	}
	if hasCfg {
		choices = append(choices, "加载持久化配置")
	}
	choices = append(choices, "自定义配置")

	return ConfigSelectModel{
		choices: choices,
		cfg:     cfg,
		hasCfg:  hasCfg,
		width:   width,
		height:  height,
	}
}

// Init 初始化
func (m ConfigSelectModel) Init() tea.Cmd {
	return nil
}

// Update 更新
func (m ConfigSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 配置选择界面启动后更新真实终端尺寸，避免 main.go 额外启动空 model。
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = true
			return m, tea.Quit
		case "q", "esc":
			m.selected = true
			m.cursor = 0 // 默认选择第一个
			return m, tea.Quit
		}
	}
	return m, nil
}

// View 渲染
func (m ConfigSelectModel) View() tea.View {
	if m.selected {
		return tea.NewView("")
	}

	var sb strings.Builder

	// 标题
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColAccent).
		Render("⚙️  DevDash 配置选择")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// 说明
	sb.WriteString(lipgloss.NewStyle().
		Foreground(ColMuted).
		Render("检测到配置文件，请选择启动方式："))
	sb.WriteString("\n\n")

	// 选项列表
	for i, choice := range m.choices {
		prefix := "  "
		if i == m.cursor {
			prefix = lipgloss.NewStyle().Foreground(ColAccent).Render("▸ ")
			choice = lipgloss.NewStyle().Foreground(ColText).Render(choice)
		} else {
			choice = lipgloss.NewStyle().Foreground(ColMuted).Render(choice)
		}
		sb.WriteString(prefix + choice + "\n")
	}

	// 显示已有配置预览
	if m.hasCfg && m.cursor == 1 {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().
			Foreground(ColSecondary).
			Render("┌─ 配置预览 ─────────────────────────"))
		sb.WriteString("\n")
		if m.cfg.DefaultCity != "" {
			sb.WriteString(fmt.Sprintf("  默认城市: %s\n", m.cfg.DefaultCity))
		}
		if len(m.cfg.RecentRepos) > 0 {
			sb.WriteString(fmt.Sprintf("  最近仓库: %d 个\n", len(m.cfg.RecentRepos)))
		}
		if len(m.cfg.RecentLogFiles) > 0 {
			sb.WriteString(fmt.Sprintf("  最近日志: %d 个\n", len(m.cfg.RecentLogFiles)))
		}
		if len(m.cfg.RecentConfigFiles) > 0 {
			sb.WriteString(fmt.Sprintf("  最近配置: %d 个\n", len(m.cfg.RecentConfigFiles)))
		}
		sb.WriteString(lipgloss.NewStyle().
			Foreground(ColSecondary).
			Render("└──────────────────────────────────────"))
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().
		Foreground(ColMuted).
		Render("↑↓ 选择  Enter 确认"))

	// 居中显示
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColSecondary).
		Padding(1, 3).
		Render(sb.String())

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card))
}

// GetChoice 获取用户选择
func (m ConfigSelectModel) GetChoice() int {
	return m.cursor
}

// GetConfig 获取配置
func (m ConfigSelectModel) GetConfig() AppConfig {
	return m.cfg
}

// HasConfig 是否有配置文件
func (m ConfigSelectModel) HasConfig() bool {
	return m.hasCfg
}
