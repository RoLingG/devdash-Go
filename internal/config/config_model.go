package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// Model 配置浏览器模块状态
type Model struct {
	root       Node
	width      int
	height     int
	cursor     int
	scroll     int
	lines      []string
	filter     string
	loaded     bool
	err        error
	configPath string

	input      component.InputModel
	dirListing bool
	dirPath    string
	dirList    component.ListModel
}

// SetRecent 设置最近记录列表（转发给 InputModel）
func (m *Model) SetRecent(items []string) { m.input.SetRecent(items) }

func (m *Model) Init(lastConfigPath string) tea.Cmd {
	for i, arg := range os.Args[1:] {
		if arg == "--config" {
			fileIdx := i + 2
			if fileIdx < len(os.Args) {
				configPath := os.Args[fileIdx]
				m.configPath = configPath
				return LoadFileCmd(configPath)
			}
		}
	}
	if lastConfigPath != "" {
		m.configPath = lastConfigPath
		return LoadFileCmd(lastConfigPath)
	}
	return func() tea.Msg { return LoadSampleConfig() }
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LoadMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.input.Prompt = "Config file path:"
			m.input.Open(m.configPath)
			return m, nil
		}
		m.root = msg.Root
		m.loaded = true
		m.input.Active = false
		m.dirListing = false
		m.err = nil
		m.rebuildLines()

	case DirMsg:
		if msg.Files == nil || len(msg.Files) == 0 {
			m.err = fmt.Errorf("no config files found in: %s", msg.Dir)
			m.input.Prompt = "Config file path:"
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
		m.rebuildLines()
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
				m.configPath = fullPath
				m.dirListing = false
				m.err = nil
				return m, tea.Batch(LoadFileCmd(fullPath), ui.UpdateCfgCmd("configPath", fullPath))
			case "esc":
				m.dirListing = false
				m.input.Prompt = "Config file path:"
				m.input.Open(m.dirPath)
			}
			return m, nil
		}

		if m.input.Active {
			return m, tea.Batch(
				m.input.Update(msg, func(path string) func() tea.Msg {
					if path != "" {
						m.err = nil
						info, err := os.Stat(path)
						if err != nil {
							m.err = fmt.Errorf("path not found: %s", path)
							return nil
						}
						if info.IsDir() {
							return ScanDirCmd(path)
						}
						m.configPath = path
						return LoadFileCmd(path)
					}
					return nil
				}),
				ui.UpdateCfgCmd("configPath", m.configPath),
			)
		}

		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}
		case "down", "j":
			if m.cursor < len(m.lines)-1 {
				m.cursor++
				m.clampScroll()
			}
		case "enter":
			m.toggleNode(m.cursor)
			m.rebuildLines()
		case "ctrl+r":
			if m.configPath != "" {
				return m, LoadFileCmd(m.configPath)
			}
		case "/":
			m.input.Prompt = "Config file path:"
			m.input.Open(m.configPath)
		case "esc":
			m.filter = ""
			m.rebuildLines()
		case "backspace":
			if ui.RuneLen(m.filter) > 0 {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.rebuildLines()
			}
		default:
			if len(key) == 1 && key >= " " {
				m.filter += key
				m.rebuildLines()
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 目录文件列表模式
	if m.dirListing {
		return ui.RenderDirListCard(ui.DirListCardOpts{
			Title:     "Config Files",
			DirPath:   m.dirPath,
			DirList:   &m.dirList,
			Height:    m.height,
			CardWidth: cardWidth,
		})
	}

	// 路径输入模式
	if m.input.Active {
		errMsg := ""
		if m.err != nil {
			errMsg = m.err.Error()
		}
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       "Open Config",
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			ErrMsg:      errMsg,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	// 未加载
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to open a config file")
		return ui.Card("Config Browser", emptyContent, ui.ColMuted, cardWidth)
	}

	// 错误
	if m.err != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("  ✗ "+m.err.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to open another file")
		return ui.Card("Config Browser", errContent, ui.ColRed, cardWidth)
	}

	// 主要内容
	var sb strings.Builder
	headerLines := 0
	if m.loaded {
		nodes := countNodes(m.root)
		info := fmt.Sprintf("📂 %s • %d nodes • %d/%d", m.configPath, nodes, m.cursor+1, len(m.lines))
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  " + info))
		sb.WriteString("\n\n")
		headerLines += 2
	}
	if m.filter != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColAccent).Render("  Search: " + m.filter))
		sb.WriteString("\n\n")
		headerLines += 2
	}

	viewH := m.height - 2 - 4 - headerLines - 3
	if viewH < 3 {
		viewH = 3
	}
	start := m.scroll
	end := start + viewH
	if end > len(m.lines) {
		end = len(m.lines)
	}
	sb.WriteString("  {")
	sb.WriteString("\n")
	for i := start; i < end; i++ {
		line := m.lines[i]
		if i == m.cursor {
			line = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(" " + line + " ")
		}
		sb.WriteString("  " + line)
		sb.WriteString("\n")
	}
	sb.WriteString("  }")
	return ui.Card("Config Browser", sb.String(), ui.ColSecondary, cardWidth)
}

func (m *Model) rebuildLines() {
	m.lines = Flatten(m.root, m.filter)
	if m.cursor >= len(m.lines) {
		m.cursor = len(m.lines) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.clampScroll()
}

func (m *Model) clampScroll() {
	viewH := m.height - 11
	if viewH < 3 {
		viewH = 3
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+viewH {
		m.scroll = m.cursor - viewH + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *Model) toggleNode(idx int) {
	if idx < 0 || idx >= len(m.lines) {
		return
	}
	counter := 0
	m.root = ToggleHelper(m.root, idx, &counter, m.filter)
}

// ---- 树渲染辅助函数 ----

// Flatten 把树拍平为可见的行
func Flatten(node Node, filter string) []string {
	var lines []string
	indent := strings.Repeat("  ", node.Depth)
	f := strings.ToLower(filter)

	if node.Key != "" {
		keyLower := strings.ToLower(node.Key)
		if len(node.Children) > 0 {
			childLines := flattenChildren(node, filter)
			matchesSelf := f == "" || strings.Contains(keyLower, f)
			matchesChild := len(childLines) > 0
			if f == "" || matchesSelf || matchesChild {
				icon := "▼"
				if !node.Expanded {
					icon = "▶"
				}
				count := fmt.Sprintf(" [%d]", len(node.Children))
				suffix := lipgloss.NewStyle().Foreground(ui.ColMuted).Render(count)
				lines = append(lines, indent+icon+" "+lipgloss.NewStyle().Foreground(ui.ColSecondary).Render(node.Key)+": "+suffix)
				if node.Expanded || (f != "" && matchesChild) {
					lines = append(lines, childLines...)
				}
			}
		} else {
			valStr := ColorizeValue(node.Value)
			valLower := strings.ToLower(fmt.Sprintf("%v", node.Value))
			if f == "" || strings.Contains(keyLower, f) || strings.Contains(valLower, f) {
				lines = append(lines, indent+"  "+lipgloss.NewStyle().Foreground(ui.ColSecondary).Render(node.Key)+": "+valStr)
			}
		}
	} else {
		if len(node.Children) > 0 {
			lines = append(lines, flattenChildren(node, filter)...)
		}
	}
	return lines
}

func flattenChildren(node Node, filter string) []string {
	f := strings.ToLower(filter)
	if f != "" && !node.Expanded {
		if strings.Contains(strings.ToLower(node.Key), f) {
			return []string{}
		}
		for _, child := range node.Children {
			if nodeMatches(child, f) {
				return []string{""}
			}
		}
		return nil
	}
	if !node.Expanded {
		return nil
	}
	childFilter := filter
	if f != "" && strings.Contains(strings.ToLower(node.Key), f) {
		childFilter = ""
	}
	var lines []string
	for _, child := range node.Children {
		lines = append(lines, Flatten(child, childFilter)...)
	}
	return lines
}

func nodeMatches(node Node, f string) bool {
	if strings.Contains(strings.ToLower(node.Key), f) {
		return true
	}
	if node.Value != nil {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%v", node.Value)), f) {
			return true
		}
	}
	for _, child := range node.Children {
		if nodeMatches(child, f) {
			return true
		}
	}
	return false
}

// ColorizeValue 根据值类型着色
func ColorizeValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return lipgloss.NewStyle().Foreground(ui.ColGreen).Render(`"` + v + `"`)
	case float64:
		return lipgloss.NewStyle().Foreground(ui.ColAccent).Render(fmt.Sprintf("%g", v))
	case bool:
		return lipgloss.NewStyle().Foreground(ui.ColRed).Render(fmt.Sprintf("%t", v))
	case nil:
		return lipgloss.NewStyle().Foreground(ui.ColMuted).Render("null")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToggleHelper 切换第 target 行对应节点的展开状态
func ToggleHelper(node Node, target int, counter *int, filter string) Node {
	f := strings.ToLower(filter)
	if node.Key != "" {
		keyLower := strings.ToLower(node.Key)
		if len(node.Children) > 0 {
			matchesSelf := f == "" || strings.Contains(keyLower, f)
			matchesChild := f != "" && hasMatchInTree(node, f)
			if f == "" || matchesSelf || matchesChild {
				if *counter == target {
					node.Expanded = !node.Expanded
					*counter++
					return node
				}
				*counter++
				if node.Expanded || matchesChild {
					for i := range node.Children {
						node.Children[i] = ToggleHelper(node.Children[i], target, counter, filter)
					}
				}
			}
		} else {
			valLower := strings.ToLower(fmt.Sprintf("%v", node.Value))
			if f == "" || strings.Contains(keyLower, f) || strings.Contains(valLower, f) {
				*counter++
			}
		}
	} else {
		for i := range node.Children {
			node.Children[i] = ToggleHelper(node.Children[i], target, counter, filter)
		}
	}
	return node
}

func hasMatchInTree(node Node, f string) bool {
	for _, child := range node.Children {
		if strings.Contains(strings.ToLower(child.Key), f) {
			return true
		}
		if child.Value != nil {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", child.Value)), f) {
				return true
			}
		}
		if len(child.Children) > 0 && hasMatchInTree(child, f) {
			return true
		}
	}
	return false
}

func countNodes(node Node) int {
	count := 0
	if node.Key != "" {
		count++
	}
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}
