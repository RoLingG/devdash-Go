package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
		// 配置选择界面启动后更新真实终端尺寸
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
			m.cursor = 0
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

	// 说明
	sb.WriteString(lipgloss.NewStyle().
		Foreground(ColMuted).
		Render("  检测到配置文件，请选择启动方式："))
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
		var preview strings.Builder
		if m.cfg.DefaultCity != "" {
			preview.WriteString(fmt.Sprintf("  默认城市: %s\n", m.cfg.DefaultCity))
		}
		if len(m.cfg.RecentRepos) > 0 {
			preview.WriteString(fmt.Sprintf("  最近仓库: %d 个\n", len(m.cfg.RecentRepos)))
		}
		if len(m.cfg.RecentLogFiles) > 0 {
			preview.WriteString(fmt.Sprintf("  最近日志: %d 个\n", len(m.cfg.RecentLogFiles)))
		}
		if len(m.cfg.RecentConfigFiles) > 0 {
			preview.WriteString(fmt.Sprintf("  最近配置: %d 个\n", len(m.cfg.RecentConfigFiles)))
		}
		sb.WriteString("\n")
		previewCard := Card("配置预览", preview.String(), ColMuted, 34)
		sb.WriteString(previewCard)
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().
		Foreground(ColMuted).
		Render("↑↓ 选择  Enter 确认"))

	// 根据内容宽度自适应卡片大小
	contentLines := strings.Split(sb.String(), "\n")
	maxContentWidth := 0
	for _, line := range contentLines {
		if w := lipgloss.Width(line); w > maxContentWidth {
			maxContentWidth = w
		}
	}
	cardWidth := maxContentWidth + 6
	maxWidth := m.width - 4
	if cardWidth > maxWidth {
		cardWidth = maxWidth
	}
	if cardWidth < 30 {
		cardWidth = 30
	}
	// 截断超宽行，防止溢出右边框
	for i, line := range contentLines {
		if lipgloss.Width(line) > cardWidth {
			contentLines[i] = ForceTruncate(line, cardWidth)
		}
	}
	content := strings.Join(contentLines, "\n")
	card := Card("📋 DevDash 配置选择", content, ColSecondary, cardWidth)

	// 水平垂直居中
	lines := strings.Split(card, "\n")
	cardHeight := len(lines)
	verticalPad := (m.height - cardHeight) / 2
	if verticalPad < 0 {
		verticalPad = 0
	}

	var centered []string
	// 上方空行
	for i := 0; i < verticalPad; i++ {
		centered = append(centered, strings.Repeat(" ", m.width))
	}
	// 每行水平居中
	for _, line := range lines {
		w := lipgloss.Width(line)
		leftPad := (m.width - w) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad := m.width - leftPad - w
		if rightPad < 0 {
			rightPad = 0
		}
		centered = append(centered, strings.Repeat(" ", leftPad)+line+strings.Repeat(" ", rightPad))
	}

	v := tea.NewView(strings.Join(centered, "\n"))
	v.AltScreen = true
	return v
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
