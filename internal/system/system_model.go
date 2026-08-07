package system

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// viewMode 子视图类型

type viewMode int

const (
	viewOverview viewMode = iota // 系统概览（CPU/内存/磁盘）
	viewProcess                  // 进程列表
)

// Model 系统监控模块状态
type Model struct {
	width    int
	height   int
	loading  bool
	loaded   bool
	err      error
	sysInfo  SysInfoMsg
	process  []ProcessInfo
	filtered []ProcessInfo // 过滤后的进程列表
	view     viewMode
	scroll   int
	filter   string
	input    component.InputModel
}

func (m *Model) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(FetchSystemInfoCmd(), FetchProcessesCmd(), SysTickCmd(2*time.Second))
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

// applyFilter 应用进程名过滤
func (m *Model) applyFilter() {
	if m.filter == "" {
		m.filtered = m.process
		return
	}
	m.filtered = nil
	lower := strings.ToLower(m.filter)
	for _, p := range m.process {
		if strings.Contains(strings.ToLower(p.Name), lower) {
			m.filtered = append(m.filtered, p)
		}
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SysInfoMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.sysInfo = msg
		}
		m.loaded = true
		m.loading = false
	case SysTickMsg:
		return m, tea.Batch(FetchSystemInfoCmd(), FetchProcessesCmd(), SysTickCmd(2*time.Second))
	case ProcMsg:
		if msg.Err == nil {
			m.process = msg.Processes
			m.applyFilter()
		}
	case tea.PasteMsg:
		if m.input.Active {
			return m, m.input.Update(msg, nil)
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.input.Active {
			return m, m.input.Update(msg, func(filter string) func() tea.Msg {
				m.filter = filter
				m.applyFilter()
				m.scroll = 0
				return nil
			})
		}
		switch msg.String() {
		case "tab":
			if m.view == viewOverview {
				m.view = viewProcess
			} else {
				m.view = viewOverview
			}
			m.scroll = 0
		case "ctrl+r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(FetchSystemInfoCmd(), FetchProcessesCmd(), SysTickCmd(2*time.Second))
		case "/":
			m.input.Prompt = "Filter:"
			m.input.Open(m.filter)
		case "ctrl+u":
			m.filter = ""
			m.applyFilter()
			m.scroll = 0
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
			Title:       "Filter Process",
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
		content := lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⏳ Loading system info...")
		return ui.Card("System", content, ui.ColAccent, cardWidth)
	}

	// 错误
	if m.err != nil {
		content := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ " + m.err.Error())
		return ui.Card("System", content, ui.ColRed, cardWidth)
	}

	switch m.view {
	case viewProcess:
		return m.viewProcs(cardWidth)
	default:
		return m.viewOverview(cardWidth)
	}
}

// viewOverview 渲染系统概览视图
func (m *Model) viewOverview(cardWidth int) string {
	var sb strings.Builder

	// CPU
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render("  CPU") + "\n")
	sb.WriteString(fmt.Sprintf("  Overall: %s\n", renderPercent(m.sysInfo.CPU.Overall)))
	if len(m.sysInfo.CPU.PerCore) > 0 {
		sb.WriteString("\n")
		cols := 4
		if cardWidth < 60 {
			cols = 2
		}
		gap := 6 // 列间距
		// 每格宽度 = (可用宽度 - 左缩进 - 列间距) / 列数
		cellW := (cardWidth - 5 - gap*(cols-1)) / cols
		// 进度条长度 = cellW - 标签(3) - 空格(1) - 百分比(5) - 空格(1)
		barLen := cellW - 10
		if barLen < 3 {
			barLen = 3
		}
		row := ""
		for i, p := range m.sysInfo.CPU.PerCore {
			pctColor := colorForPercent(p)
			label := fmt.Sprintf("C%-2d", i)
			pct := fmt.Sprintf("%4.0f%%", p)
			// 核心占用进度条
			filled := int(p / 100 * float64(barLen))
			if filled > barLen {
				filled = barLen
			}
			filledStr := strings.Repeat("█", filled)
			emptyStr := strings.Repeat("░", barLen-filled)
			// 奇偶序号核心色调区分
			labelColor := ui.ColPrimary
			if i%2 == 0 {
				labelColor = ui.ColSecondary
			}
			// 分段上色
			cell := lipgloss.NewStyle().Foreground(labelColor).Render(label) + " " +
				lipgloss.NewStyle().Foreground(pctColor).Render(filledStr) +
				lipgloss.NewStyle().Foreground(ui.ColMuted).Render(emptyStr) + " " +
				lipgloss.NewStyle().Foreground(pctColor).Bold(true).Render(pct)
			row += cell
			if (i+1)%cols == 0 {
				sb.WriteString("  " + row + "\n")
				row = ""
			} else {
				row += strings.Repeat(" ", gap) // 列间距
			}
		}
		if row != "" {
			sb.WriteString("  " + row + "\n")
		}
	}

	// 内存
	sb.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render("  Memory") + "\n")
	usedGB := float64(m.sysInfo.Mem.Used) / 1024 / 1024 / 1024
	totalGB := float64(m.sysInfo.Mem.Total) / 1024 / 1024 / 1024
	sb.WriteString(fmt.Sprintf("  %s / %.1f GB\n", renderPercent(m.sysInfo.Mem.Percent), totalGB))
	sb.WriteString(renderBar(m.sysInfo.Mem.Percent, cardWidth-4) + "\n")
	_ = usedGB

	// 磁盘
	if len(m.sysInfo.Disks) > 0 {
		sb.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render("  Disk") + "\n")
		for _, d := range m.sysInfo.Disks {
			usedGB := float64(d.Used) / 1024 / 1024 / 1024
			totalGB := float64(d.Total) / 1024 / 1024 / 1024
			sb.WriteString(fmt.Sprintf("  %s  %.1f / %.1f GB\n", d.Mount, usedGB, totalGB))
			sb.WriteString(renderBar(d.Percent, cardWidth-4) + "\n")
		}
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
	return ui.Card("System", visible, ui.ColAccent, cardWidth)
}

// viewProcs 渲染进程列表视图
func (m *Model) viewProcs(cardWidth int) string {
	var sb strings.Builder

	// 过滤提示
	if m.filter != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Filter: "+m.filter) + "\n\n")
	}

	// 表头
	header := lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render(
		fmt.Sprintf("  %-8s %-25s %8s %10s", "PID", "Name", "CPU%", "Mem(MB)"),
	)
	sb.WriteString(header + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+strings.Repeat("─", cardWidth-6)) + "\n")

	// 进程列表
	if len(m.filtered) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No processes found") + "\n")
	} else {
		for _, p := range m.filtered {
			cpuColor := ui.ColGreen
			if p.CPU > 50 {
				cpuColor = ui.ColRed
			} else if p.CPU > 10 {
				cpuColor = ui.ColAccent
			}
			name := p.Name
			if len(name) > 25 {
				name = name[:22] + "..."
			}
			cpuStr := lipgloss.NewStyle().Foreground(cpuColor).Render(fmt.Sprintf("%7.1f%%", p.CPU))
			memStr := fmt.Sprintf("%9.1f", p.MemMB)
			line := fmt.Sprintf("  %-8d %-25s %s %s", p.PID, name, cpuStr, memStr)
			sb.WriteString(line + "\n")
		}
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
	return ui.Card("System", visible, ui.ColAccent, cardWidth)
}

// renderPercent 渲染百分比文字（带颜色）
func renderPercent(pct float64) string {
	color := colorForPercent(pct)
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%.1f%%", pct))
}

// colorForPercent 根据百分比返回颜色
// 修改: lipgloss v2 中颜色类型改为标准库 image/color.Color
func colorForPercent(pct float64) color.Color {
	if pct >= 90 {
		return ui.ColRed
	} else if pct >= 70 {
		return ui.ColAccent
	}
	return ui.ColMuted
}

// renderBar 渲染一个进度条
func renderBar(pct float64, width int) string {
	if width < 10 {
		width = 10
	}
	barWidth := width - 10 // 留空间给百分比文字
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	empty := barWidth - filled
	color := colorForPercent(pct)
	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	bar += lipgloss.NewStyle().Foreground(ui.ColMuted).Render(strings.Repeat("░", empty))
	pctColor := colorForPercent(pct)
	return "  " + bar + lipgloss.NewStyle().Foreground(pctColor).Render(fmt.Sprintf(" %.1f%%", pct))
}
