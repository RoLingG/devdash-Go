// ============================================================================
// mod_config.go — 配置文件浏览器模块
//
// 功能：
//   - 解析 JSON / YAML 文件为树形结构
//   - 折叠/展开节点（Enter 键）
//   - ↑↓ 导航，/ 输入文件路径或搜索键名
//   - 路径输入支持目录扫描（列出 .json/.yaml/.yml 文件）
//   - 语法高亮：键名=蓝紫色、字符串=绿色、数字=黄色、bool=红色
//
// 使用方式：
//   devdash.exe --config package.json
//   devdash.exe --config config.yaml
//   devdash.exe                   # 按 / 输入路径
// ============================================================================

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// ==================== 树节点定义 ====================

// cfgNode 配置树的一个节点
type cfgNode struct {
	key      string      // 键名（根节点为空）
	value    interface{} // 原始值（叶子节点）
	children []cfgNode   // 子节点（对象/数组）
	isArray  bool        // 是否是数组元素
	expanded bool        // 是否展开
	depth    int         // 缩进层级
}

// ==================== 消息类型 ====================

// cfgLoadMsg 配置加载完成消息
type cfgLoadMsg struct {
	root cfgNode
	err  error
}

// cfgDirMsg 目录扫描结果
type cfgDirMsg struct {
	dir   string
	files []string
}

// ==================== 加载函数 ====================

// loadConfig 从文件加载配置（自动识别 JSON / YAML）
func loadConfig(path string) tea.Msg {
	data, err := os.ReadFile(path)
	if err != nil {
		return cfgLoadMsg{err: fmt.Errorf("cannot read: %s (%v)", path, err)}
	}

	var raw interface{}
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return cfgLoadMsg{err: fmt.Errorf("YAML parse error: %v", err)}
		}
	default:
		// 默认按 JSON 解析
		if err := json.Unmarshal(data, &raw); err != nil {
			return cfgLoadMsg{err: fmt.Errorf("JSON parse error: %v", err)}
		}
	}

	root := buildTree("", raw, 0, true)
	return cfgLoadMsg{root: root}
}

// scanConfigDir 扫描目录下所有 JSON / YAML 文件
func scanConfigDir(dir string) tea.Msg {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return cfgDirMsg{dir: dir, files: nil}
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".json" || ext == ".yaml" || ext == ".yml" {
			files = append(files, e.Name())
		}
	}
	return cfgDirMsg{dir: dir, files: files}
}

// loadSampleConfig 加载示例配置
func loadSampleConfig() tea.Msg {
	sample := `{
  "name": "devdash",
  "version": "1.0.0",
  "description": "Developer terminal dashboard",
  "author": {
    "name": "You",
    "email": "you@example.com"
  },
  "dependencies": {
    "bubbletea": "^1.0.0",
    "lipgloss": "^1.0.0",
    "bubbles": "^1.0.0"
  },
  "scripts": {
    "build": "go build -o devdash .",
    "run": "go run .",
    "test": "go test ./..."
  },
  "config": {
    "theme": "dark",
    "refreshRate": 30,
    "modules": ["git", "log", "weather", "config"],
    "features": {
      "autoRefresh": true,
      "notifications": false
    }
  }
}`
	var raw interface{}
	_ = json.Unmarshal([]byte(sample), &raw)
	root := buildTree("", raw, 0, true)
	return cfgLoadMsg{root: root}
}

// buildTree 递归构建配置树
func buildTree(key string, val interface{}, depth int, expanded bool) cfgNode {
	node := cfgNode{key: key, depth: depth, expanded: expanded}

	switch v := val.(type) {
	case map[string]interface{}:
		// 排序 key 保证子节点顺序稳定（Go map 遍历顺序随机）
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			node.children = append(node.children, buildTree(k, v[k], depth+1, false))
		}
	case map[interface{}]interface{}:
		// YAML 解析可能返回这种类型
		keys := make([]string, 0, len(v))
		keyMap := make(map[string]interface{}, len(v))
		for k, child := range v {
			ks := fmt.Sprintf("%v", k)
			keys = append(keys, ks)
			keyMap[ks] = child
		}
		sort.Strings(keys)
		for _, k := range keys {
			node.children = append(node.children, buildTree(k, keyMap[k], depth+1, false))
		}
	case []interface{}:
		node.isArray = true
		for i, child := range v {
			node.children = append(node.children, buildTree(fmt.Sprintf("[%d]", i), child, depth+1, false))
		}
	default:
		node.value = val
	}

	return node
}

// ==================== 渲染相关 ====================

// flatten 把树拍平为可见的行（只包含展开的节点）
// filter 非空时，只显示键名或值包含 filter 的节点（递归保留父级路径）
func flatten(node cfgNode, filter string) []string {
	var lines []string
	indent := strings.Repeat("  ", node.depth)
	f := strings.ToLower(filter)

	if node.key != "" {
		keyLower := strings.ToLower(node.key)

		if len(node.children) > 0 {
			// 对象/数组节点
			childLines := flattenChildren(node, filter)
			matchesSelf := f == "" || strings.Contains(keyLower, f)
			matchesChild := len(childLines) > 0

			if f == "" || matchesSelf || matchesChild {
				icon := "▼"
				if !node.expanded {
					icon = "▶"
				}
				count := fmt.Sprintf(" [%d]", len(node.children))
				suffix := lipgloss.NewStyle().Foreground(colMuted).Render(count)
				lines = append(lines, indent+icon+" "+lipgloss.NewStyle().Foreground(colSecondary).Render(node.key)+": "+suffix)

				if node.expanded || (f != "" && matchesChild) {
					lines = append(lines, childLines...)
				}
			}
		} else {
			// 叶子节点
			valStr := colorizeValue(node.value)
			valLower := strings.ToLower(fmt.Sprintf("%v", node.value))
			if f == "" || strings.Contains(keyLower, f) || strings.Contains(valLower, f) {
				lines = append(lines, indent+"  "+lipgloss.NewStyle().Foreground(colSecondary).Render(node.key)+": "+valStr)
			}
		}
	} else {
		// 根节点：只输出子节点，不输出 { }（避免行号与 toggleHelper 错位）
		if len(node.children) > 0 {
			lines = append(lines, flattenChildren(node, filter)...)
		}
	}
	return lines
}

// flattenChildren 递归拍平子节点（折叠时不生成子节点行）
func flattenChildren(node cfgNode, filter string) []string {
	f := strings.ToLower(filter)

	if f != "" && !node.expanded {
		// 父节点自身匹配 filter → 返回空切片（matchesSelf 已保证节点显示，不需要占位信号）
		if strings.Contains(strings.ToLower(node.key), f) {
			return []string{}
		}
		// 父节点不匹配：只检查子节点是否有匹配（返回占位信号）
		for _, child := range node.children {
			if nodeMatches(child, f) {
				return []string{""}
			}
		}
		return nil
	}
	if !node.expanded {
		return nil
	}
	// 父节点匹配 filter 时，子节点不过滤
	childFilter := filter
	if f != "" && strings.Contains(strings.ToLower(node.key), f) {
		childFilter = ""
	}
	var lines []string
	for _, child := range node.children {
		lines = append(lines, flatten(child, childFilter)...)
	}
	return lines
}

// nodeMatches 检查节点自身或子树是否匹配 filter
func nodeMatches(node cfgNode, f string) bool {
	if strings.Contains(strings.ToLower(node.key), f) {
		return true
	}
	if node.value != nil {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%v", node.value)), f) {
			return true
		}
	}
	for _, child := range node.children {
		if nodeMatches(child, f) {
			return true
		}
	}
	return false
}

// colorizeValue 根据值类型着色
func colorizeValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return lipgloss.NewStyle().Foreground(colGreen).Render(`"` + v + `"`)
	case float64:
		return lipgloss.NewStyle().Foreground(colAccent).Render(fmt.Sprintf("%g", v))
	case bool:
		return lipgloss.NewStyle().Foreground(colRed).Render(fmt.Sprintf("%t", v))
	case nil:
		return lipgloss.NewStyle().Foreground(colMuted).Render("null")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ==================== 模型 ====================

type configModel struct {
	root       cfgNode
	width      int
	height     int
	cursor     int      // 当前光标位置（行号）
	scroll     int      // 滚动偏移
	lines      []string // 拍平后的可见行
	filter     string
	loaded     bool
	err        error
	configPath string // 当前配置文件路径

	// 路径输入模式
	input inputModel // 通用输入组件

	// 目录文件列表模式
	dirListing bool      // 是否在显示目录文件列表
	dirPath    string    // 当前目录路径
	dirList    listModel // 通用列表组件
}

func (m *configModel) Init() tea.Cmd {
	// 命令行参数：--config <file>
	for i, arg := range os.Args[1:] {
		if arg == "--config" {
			fileIdx := i + 2
			if fileIdx < len(os.Args) {
				configPath := os.Args[fileIdx]
				return func() tea.Msg { return loadConfig(configPath) }
			}
		}
	}
	// 默认：显示示例配置
	return func() tea.Msg { return loadSampleConfig() }
}

func (m *configModel) UpdateSize(w, h int) { m.width = w; m.height = h }

// ==================== 更新逻辑 ====================

func (m *configModel) Update(msg tea.Msg) (*configModel, tea.Cmd) {
	switch msg := msg.(type) {

	// 配置加载完成
	case cfgLoadMsg:
		if msg.err != nil {
			m.err = msg.err
			m.input.prompt = "Config file path:"
			m.input.Open(m.configPath)
			return m, nil
		}
		m.root = msg.root
		m.loaded = true
		m.input.active = false
		m.dirListing = false
		m.err = nil
		m.rebuildLines()

	// 目录扫描完成
	case cfgDirMsg:
		if msg.files == nil || len(msg.files) == 0 {
			m.err = fmt.Errorf("no config files found in: %s", msg.dir)
			m.input.prompt = "Config file path:"
			m.input.Open(m.dirPath)
			m.dirListing = false
			return m, nil
		}
		m.dirListing = true
		m.dirPath = msg.dir
		m.dirList.SetItems(msg.files)
		m.input.active = false
		return m, nil

	// 粘贴
	case tea.PasteMsg:
		if m.input.active {
			return m, m.input.Update(msg, nil)
		}
		m.filter += msg.Content
		m.rebuildLines()
		return m, nil

	// 按键
	case tea.KeyPressMsg:
		key := msg.String()

		// ---- 目录文件列表模式 ----
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
				return m, func() tea.Msg { return loadConfig(fullPath) }
			case "esc":
				m.dirListing = false
				m.input.prompt = "Config file path:"
				m.input.Open(m.dirPath)
			}
			return m, nil
		}

		// ---- 路径输入模式 ----
		if m.input.active {
			return m, m.input.Update(msg, func(path string) tea.Cmd {
				if path != "" {
					m.err = nil
					info, err := os.Stat(path)
					if err != nil {
						m.err = fmt.Errorf("path not found: %s", path)
						return nil
					}
					if info.IsDir() {
						return func() tea.Msg { return scanConfigDir(path) }
					}
					m.configPath = path
					return func() tea.Msg { return loadConfig(path) }
				}
				return nil
			})
		}

		// ---- 正常模式 ----
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
		case "/":
			m.input.prompt = "Config file path:"
			m.input.Open(m.configPath)
		case "esc":
			m.filter = ""
			m.rebuildLines()
		case "backspace":
			if runeLen(m.filter) > 0 {
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

// ==================== 视图 ====================

func (m *configModel) View() string {
	// 卡片宽度
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// ---- 目录文件列表模式 ----
	if m.dirListing {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colSecondary).Render("  📂 " + m.dirPath))
		sb.WriteString("\n\n")

		maxShow := m.height - 10
		if maxShow < 5 {
			maxShow = 5
		}

		// 使用 listModel 渲染
		highlightStyle := lipgloss.NewStyle().Foreground(colAccent)
		listContent := m.dirList.RenderWithFormat(maxShow, func(item string, selected bool) string {
			if selected {
				return highlightStyle.Render("  > " + item)
			}
			return lipgloss.NewStyle().Foreground(colText).Render("    " + item)
		})

		sb.WriteString(listContent)
		sb.WriteString("\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  ↑↓ select  Enter open  Esc back"))
		return card("Config Files", sb.String(), colSecondary, cardWidth)
	}

	// ---- 路径输入模式 ----
	if m.input.active {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render("  " + m.input.prompt))
		sb.WriteString("\n")

		before := runeSubstr(m.input.value, 0, m.input.cursor)
		after := runeSubstr(m.input.value, m.input.cursor, runeLen(m.input.value))
		inputLine := "  > " + before + lipgloss.NewStyle().Foreground(colAccent).Render("|") + after
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render(inputLine))
		sb.WriteString("\n\n")

		if m.err != nil {
			sb.WriteString("  " + lipgloss.NewStyle().Foreground(colRed).Render("✗ "+m.err.Error()))
			sb.WriteString("\n\n")
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  Enter confirm  ←→ cursor  Home/End  Esc cancel"))
		return card("Open Config", sb.String(), colSecondary, cardWidth)
	}

	// ---- 加载中 / 未加载 ----
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(colMuted).Render("  Press '/' to open a config file")
		return card("Config Browser", emptyContent, colMuted, cardWidth)
	}

	// ---- 错误 ----
	if m.err != nil {
		errContent := lipgloss.NewStyle().Foreground(colRed).Render("  ✗ "+m.err.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(colMuted).Render("  Press '/' to open another file")
		return card("Config Browser", errContent, colRed, cardWidth)
	}

	// ---- 主要内容 ----
	var sb strings.Builder

	// 头部信息：文件路径 + 节点数 + 光标位置
	headerLines := 0
	if m.loaded {
		nodes := countNodes(m.root)
		info := fmt.Sprintf("📂 %s  •  %d nodes  •  %d/%d", m.configPath, nodes, m.cursor+1, len(m.lines))
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  " + info))
		sb.WriteString("\n\n")
		headerLines += 2
	}

	// 过滤状态
	if m.filter != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(colAccent).Render("  Search: " + m.filter))
		sb.WriteString("\n\n")
		headerLines += 2
	}

	// 可用高度：终端高度 - tab栏(1) - help栏(1) - 卡片开销(4) - 头部 - { }(2)
	viewH := m.height - 2 - 4 - headerLines - 2
	if viewH < 3 {
		viewH = 3
	}

	start := m.scroll
	end := start + viewH
	if end > len(m.lines) {
		end = len(m.lines)
	}

	// 固定的 JSON 开括号
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

	// 固定的 JSON 闭括号
	sb.WriteString("  }")

	return card("Config Browser", sb.String(), colSecondary, cardWidth)
}

// rebuildLines 从根节点重建可见行列表
func (m *configModel) rebuildLines() {
	m.lines = flatten(m.root, m.filter)
	if m.cursor >= len(m.lines) {
		m.cursor = len(m.lines) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.clampScroll()
}

// clampScroll 确保光标在可见滚动范围内
func (m *configModel) clampScroll() {
	viewH := m.height - 10 // 预留头部和底部空间
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

// toggleNode 切换第 idx 行对应节点的展开状态
func (m *configModel) toggleNode(idx int) {
	if idx < 0 || idx >= len(m.lines) {
		return
	}
	counter := 0
	m.root = toggleHelper(m.root, idx, &counter, m.filter)
}

func toggleHelper(node cfgNode, target int, counter *int, filter string) cfgNode {
	f := strings.ToLower(filter)

	if node.key != "" {
		keyLower := strings.ToLower(node.key)

		if len(node.children) > 0 {
			// 对象/数组节点：和 flatten 保持一致的可见性判断
			matchesSelf := f == "" || strings.Contains(keyLower, f)
			matchesChild := f != "" && hasMatchInTree(node, f)

			if f == "" || matchesSelf || matchesChild {
				if *counter == target {
					node.expanded = !node.expanded
					*counter++ // 必须递增，否则后续节点 counter 重复
					return node
				}
				*counter++

				// 递归子节点（和 flatten 条件一致）
				if node.expanded || matchesChild {
					for i := range node.children {
						node.children[i] = toggleHelper(node.children[i], target, counter, filter)
					}
				}
			}
		} else {
			// 叶子节点
			valLower := strings.ToLower(fmt.Sprintf("%v", node.value))
			if f == "" || strings.Contains(keyLower, f) || strings.Contains(valLower, f) {
				*counter++
			}
		}
	} else {
		// 根节点
		for i := range node.children {
			node.children[i] = toggleHelper(node.children[i], target, counter, filter)
		}
	}
	return node
}

// hasMatchInTree 检查节点子树中是否有匹配 filter 的节点（不含自身）
func hasMatchInTree(node cfgNode, f string) bool {
	for _, child := range node.children {
		if strings.Contains(strings.ToLower(child.key), f) {
			return true
		}
		if child.value != nil {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", child.value)), f) {
				return true
			}
		}
		if len(child.children) > 0 && hasMatchInTree(child, f) {
			return true
		}
	}
	return false
}

// countMatches 统计节点及其子树中匹配 filter 的节点数
func countMatches(node cfgNode, filter string) int {
	f := strings.ToLower(filter)
	if f == "" {
		return 1
	}
	count := 0
	if strings.Contains(strings.ToLower(node.key), f) {
		count++
	}
	if node.value != nil {
		valLower := strings.ToLower(fmt.Sprintf("%v", node.value))
		if strings.Contains(valLower, f) {
			count++
		}
	}
	for _, child := range node.children {
		count += countMatches(child, filter)
	}
	return count
}

// countNodes 统计树中节点总数
func countNodes(node cfgNode) int {
	count := 0
	if node.key != "" {
		count++
	}
	for _, child := range node.children {
		count += countNodes(child)
	}
	return count
}
