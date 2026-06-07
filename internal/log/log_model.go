package log

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// Model 日志查看器模块状态
type Model struct {
	all      []Line
	filtered []Line
	width    int
	height   int
	scroll   int
	filter   string
	loaded   bool
	errMsg   string
	logPath  string

	input      component.InputModel
	dirListing bool
	dirPath    string
	dirList    component.ListModel
}

func (m *Model) Init(lastLogPath string) tea.Cmd {
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		return LoadFromStdin
	}
	for i, arg := range os.Args[1:] {
		if arg == "--log" {
			fileIdx := i + 2
			if fileIdx < len(os.Args) {
				logPath := os.Args[fileIdx]
				m.logPath = logPath
				return LoadFromFileCmd(logPath)
			}
		}
	}
	if lastLogPath != "" {
		m.logPath = lastLogPath
		return LoadFromFileCmd(lastLogPath)
	}
	return nil
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		if msg.Err != nil {
			m.errMsg = msg.Err.Error()
			m.input.Prompt = "Log file path:"
			m.input.Open(m.logPath)
			return m, nil
		}
		lines := msg.Lines
		if lines == nil {
			lines = SampleLogLines()
		}
		m.all = lines
		m.filtered = lines
		m.loaded = true
		m.input.Active = false
		m.dirListing = false
		m.errMsg = ""
		m.scroll = len(m.filtered) - (m.height - 9)
		if m.scroll < 0 {
			m.scroll = 0
		}

	case DirMsg:
		if msg.Files == nil || len(msg.Files) == 0 {
			m.errMsg = "No .log files found in: " + msg.Dir
			m.input.Prompt = "Log file path:"
			m.input.Open(m.dirPath)
			m.dirListing = false
			return m, nil
		}
		m.dirListing = true
		m.dirPath = msg.Dir
		m.dirList.SetItems(msg.Files)
		m.input.Active = false
		return m, nil

	case tea.PasteMsg:
		if m.input.Active {
			return m, m.input.Update(msg, nil)
		}
		m.filter += msg.Content
		m.applyFilter()
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		if m.dirListing {
			switch key {
			case "up", "k":
				m.dirList.MoveUp()
			case "down", "j":
				m.dirList.MoveDown()
			case "enter":
				selected := m.dirList.Selected()
				fullPath := filepath.Join(m.dirPath, selected)
				m.logPath = fullPath
				m.dirListing = false
				m.filter = ""
				m.errMsg = ""
				return m, tea.Batch(LoadFromFileCmd(fullPath), ui.UpdateCfgCmd("logPath", fullPath))
			case "esc":
				m.dirListing = false
				m.input.Prompt = "Log file path:"
				m.input.Open(m.dirPath)
			}
			return m, nil
		}

		if m.input.Active {
			return m, tea.Batch(
				m.input.Update(msg, func(path string) func() tea.Msg {
					if path != "" {
						m.errMsg = ""
						info, err := os.Stat(path)
						if err != nil {
							m.errMsg = "Path not found: " + path
							return nil
						}
						if info.IsDir() {
							return ScanDirCmd(path)
						}
						m.logPath = path
						m.filter = ""
						return LoadFromFileCmd(path)
					}
					return nil
				}),
				ui.UpdateCfgCmd("logPath", m.logPath),
			)
		}

		switch key {
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			maxScroll := len(m.filtered) - (m.height - 9)
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scroll < maxScroll {
				m.scroll++
			}
		case "/":
			m.input.Prompt = "Log file path:"
			m.input.Open(m.logPath)
		case "R":
			if m.logPath != "" {
				return m, func() tea.Msg { return LoadFromFile(m.logPath) }
			}
		case "enter":
			m.applyFilter()
		case "esc":
			m.filter = ""
			m.filtered = m.all
			m.scroll = 0
		case "backspace":
			if ui.RuneLen(m.filter) > 0 {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.applyFilter()
			}
		default:
			if len(key) == 1 && key >= " " {
				m.filter += key
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m *Model) applyFilter() {
	if m.filter == "" {
		m.filtered = m.all
		m.scroll = 0
		return
	}
	re, err := regexp.Compile(m.filter)
	if err != nil {
		m.errMsg = "Invalid regex"
		return
	}
	m.errMsg = ""
	var filtered []Line
	for _, line := range m.all {
		if re.MatchString(line.Raw) {
			filtered = append(filtered, line)
		}
	}
	m.filtered = filtered
	m.scroll = 0
}

func (m *Model) View() string {
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 目录文件列表模式
	if m.dirListing {
		return ui.RenderDirListCard(ui.DirListCardOpts{
			Title:     "Log Files",
			DirPath:   m.dirPath,
			DirList:   &m.dirList,
			Height:    m.height,
			CardWidth: cardWidth,
			ErrMsg:    m.errMsg,
		})
	}

	// 路径输入模式
	if m.input.Active {
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:     "Open Log",
			Prompt:    m.input.Prompt,
			Value:     m.input.Value,
			Cursor:    m.input.Cursor,
			ErrMsg:    m.errMsg,
			CardWidth: cardWidth,
		})
	}

	// 未加载
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to enter log file path")
		return ui.Card("Log Viewer", emptyContent, ui.ColMuted, cardWidth)
	}

	var lines []string
	filterInfo := ""
	if m.filter != "" {
		filterInfo = fmt.Sprintf(" Filter: %s (%d/%d lines)", m.filter, len(m.filtered), len(m.all))
	} else {
		filterInfo = fmt.Sprintf(" Showing all %d lines", len(m.filtered))
	}
	if m.errMsg != "" {
		filterInfo += " | " + lipgloss.NewStyle().Foreground(ui.ColRed).Render(m.errMsg)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColAccent).Render(filterInfo))
	lines = append(lines, "")

	viewH := m.height - 11
	if viewH < 5 {
		viewH = 5
	}
	start := m.scroll
	end := start + viewH
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for _, l := range m.filtered[start:end] {
		lines = append(lines, colorizeLog(l))
	}

	lines = append(lines, "")
	lines = append(lines, m.stats())
	//return strings.Join(lines, "\n")
	return ui.Box("Log", strings.Join(lines, "\n"), m.width)
}

func (m *Model) stats() string {
	counts := map[string]int{}
	for _, l := range m.all {
		counts[l.Level]++
	}
	return fmt.Sprintf(" %s  %s  %s  %s  %s",
		lipgloss.NewStyle().Foreground(ui.ColRed).Render(fmt.Sprintf("ERROR:%d", counts["ERROR"])),
		lipgloss.NewStyle().Foreground(ui.ColAccent).Render(fmt.Sprintf("WARN:%d", counts["WARN"])),
		lipgloss.NewStyle().Foreground(ui.ColGreen).Render(fmt.Sprintf("INFO:%d", counts["INFO"])),
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("DEBUG:%d", counts["DEBUG"])),
		fmt.Sprintf("OTHER:%d", counts["OTHER"]),
	)
}

func colorizeLog(l Line) string {
	var color lipgloss.Color
	switch l.Level {
	case "ERROR":
		color = ui.ColRed
	case "WARN":
		color = ui.ColAccent
	case "INFO":
		color = ui.ColGreen
	case "DEBUG":
		color = ui.ColMuted
	default:
		return "  " + l.Raw
	}
	return "  " + lipgloss.NewStyle().Foreground(color).Render(l.Raw)
}
