package ports

import (
	"fmt"
	"strconv"
	"strings"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// Model 端口扫描模块状态
type Model struct {
	width   int
	height  int
	loading bool
	loaded  bool
	err     error
	ports   []PortInfo
	scroll  int
	input   component.InputModel
	extra   []int // 用户自定义端口
}

func (m *Model) Init() tea.Cmd {
	m.loading = true
	return ScanPortsCmd(m.extra)
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PortsMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.ports = msg.Ports
		}
		m.loaded = true
		m.loading = false
	case tea.PasteMsg:
		if m.input.Active {
			return m, m.input.Update(msg, nil)
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.input.Active {
			return m, m.input.Update(msg, func(portStr string) func() tea.Msg {
				if portStr != "" {
					if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port <= 65535 {
						// 检查是否已存在
						exists := false
						for _, p := range m.ports {
							if p.Port == port {
								exists = true
								break
							}
						}
						if !exists {
							m.extra = append(m.extra, port)
							m.loading = true
							return ScanPortsCmd(m.extra)
						}
					}
				}
				return nil
			})
		}
		switch msg.String() {
		case "ctrl+r":
			m.loading = true
			m.err = nil
			return m, ScanPortsCmd(m.extra)
		case "/":
			m.input.Prompt = "Port:"
			m.input.Open("")
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			m.scroll++
		case "home":
			m.scroll = 0
		case "end":
			m.scroll = 1 << 30
		}
	}
	return m, nil
}

func (m *Model) View() string {
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 输入框
	if m.input.Active {
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       "Add Port",
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	// 加载中
	if m.loading {
		content := lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⏳ Scanning ports...")
		return ui.Card("Ports", content, ui.ColAccent, cardWidth)
	}

	// 错误
	if m.err != nil {
		content := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.err.Error())
		return ui.Card("Ports", content, ui.ColRed, cardWidth)
	}

	var sb strings.Builder

	// 统计
	openCount := 0
	for _, p := range m.ports {
		if p.Open {
			openCount++
		}
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render(
		fmt.Sprintf("  %d open / %d total", openCount, len(m.ports)),
	) + "\n\n")

	// 表头
	header := lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render(
		fmt.Sprintf("  %-6s %-6s %-18s", "Status", "Port", "Service"),
	)
	sb.WriteString(header + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+strings.Repeat("─", cardWidth-6)) + "\n")

	// 端口列表
	for _, p := range m.ports {
		var status string
		if p.Open {
			status = lipgloss.NewStyle().Foreground(ui.ColGreen).Render("  ✓  ")
		} else {
			status = lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  ✗  ")
		}
		portStr := fmt.Sprintf("%-6d", p.Port)
		service := p.Service
		if len(service) > 18 {
			service = service[:15] + "..."
		}
		sb.WriteString(status + portStr + " " + lipgloss.NewStyle().Foreground(ui.ColText).Render(service) + "\n")
	}

	// 滚动处理
	content := sb.String()
	lines := strings.Split(content, "\n")
	viewH := m.height - 7
	if viewH < 3 {
		viewH = 3
	}
	totalLines := len(lines)
	if m.scroll > totalLines-viewH {
		m.scroll = totalLines - viewH
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	end := m.scroll + viewH
	if end > totalLines {
		end = totalLines
	}
	visible := strings.Join(lines[m.scroll:end], "\n")
	return ui.Card("Ports", visible, ui.ColAccent, cardWidth)
}
