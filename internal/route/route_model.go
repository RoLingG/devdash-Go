package route

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// ---- 消息类型 ----

// clearMsgMsg 用于自动清除状态提示
type clearMsgMsg struct{}

type RoutesMsg struct {
	Routes []RouteEntry
	Ifaces []IfaceInfo
	Err    error
}

type RouteActionMsg struct {
	OK    bool
	Err   error
	IsAdd bool // true=添加, false=删除
}

type viewMode int

const (
	viewRoutes viewMode = iota
	viewIfaces
)

type addField int

const (
	fieldDest addField = iota
	fieldMask
	fieldGateway
	fieldMetric
	fieldIface
)

type Model struct {
	width       int
	height      int
	loading     bool
	loaded      bool
	savedRoutes []ui.RouteConfig // 从配置加载的保存路由（只读，用于 ctrl+l 加载）
	routes      []RouteEntry
	ifaces      []IfaceInfo
	cursor      int
	scroll      int //当前视图窗口第一行在完整列表中的行号偏移
	mode        viewMode
	isAdmin     bool
	msg         string // 状态提示
	msgIsErr    bool
	input       inputLike
	addOverlay  bool
	addField    addField
	addDest     string
	addMask     string
	addGateway  string
	addMetric   string
	addIfIdx    int // 当前选中的接口索引
}

// SetSavedRoutes 从 main.go 同步已保存的路由配置
func (m *Model) SetSavedRoutes(routes []ui.RouteConfig) { m.savedRoutes = routes }

// inputLike 简单输入状态（不依赖 component.InputModel，避免冲突）
type inputLike struct {
	Active bool
	Value  string
}

func (m *Model) Init() tea.Cmd {
	m.loading = true
	m.isAdmin = IsAdmin()
	return m.loadRoutesCmd()
}

func (m *Model) loadRoutesCmd() tea.Cmd {
	return func() tea.Msg {
		routes, err := GetRoutes()
		if err != nil {
			return RoutesMsg{Err: err}
		}
		ifaces, _ := GetInterfaces()
		return RoutesMsg{Routes: routes, Ifaces: ifaces}
	}
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

func (m *Model) InputActive() bool { return m.input.Active || m.addOverlay }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RoutesMsg:
		m.loading = false
		m.loaded = true
		if msg.Err != nil {
			return m, m.err(msg.Err)
		}
		m.routes = msg.Routes
		m.ifaces = msg.Ifaces
		m.msg = ""
		return m, nil

	case clearMsgMsg:
		m.msg = ""
		return m, nil

	case RouteActionMsg:
		if msg.Err != nil {
			return m, m.err(msg.Err)
		} else if msg.IsAdd {
			m.msg = "Route added successfully"
			m.msgIsErr = false
		} else {
			m.msg = "Route deleted successfully"
			m.msgIsErr = false
		}
		// 刷新路由表
		return m, m.loadRoutesCmd()

	case tea.KeyPressMsg:
		// 添加 overlay 模式
		if m.addOverlay {
			return m.handleAddKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) err(e error) tea.Cmd {
	m.msg = e.Error()
	m.msgIsErr = true
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearMsgMsg{} })
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+r":
		m.loading = true
		return m, m.loadRoutesCmd()

	case "tab":
		if m.mode == viewRoutes {
			m.mode = viewIfaces
		} else {
			m.mode = viewRoutes
		}
		m.cursor = 0
		m.scroll = 0

	case "ctrl+a":
		if !m.isAdmin {
			return m, m.err(fmt.Errorf("需要管理员权限才能添加路由"))
		}
		m.openAddOverlay()

	case "ctrl+d":
		if !m.isAdmin {
			return m, m.err(fmt.Errorf("需要管理员权限才能删除路由"))
		}
		if m.mode == viewRoutes && len(m.routes) > 0 {
			return m, m.deleteSelected()
		}

	case "ctrl+s":
		// 保存当前静态路由到配置文件
		return m.saveRoutes()

	case "ctrl+l":
		// 从配置文件加载并应用保存的路由
		if !m.isAdmin {
			return m, m.err(fmt.Errorf("需要管理员权限才能加载路由"))
		}
		return m.loadSavedRoutes()

	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "home":
		m.cursor = 0
		m.scroll = 0
	case "end":
		total := m.totalItems()
		if total > 0 {
			m.cursor = total - 1
		}
	case "pgup":
		m.moveCursor(-10)
	case "pgdown":
		m.moveCursor(10)
	}
	return m, nil
}

func (m *Model) handleAddKey(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.addOverlay = false
		return m, nil

	case "tab":
		m.addField = (m.addField + 1) % 5
		return m, nil

	case "enter":
		return m, m.submitAdd()

	case "backspace":
		m.backspaceAddField()

	default:
		// 输入字符到当前字段
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] <= 126 {
			m.appendToField(s)
		}
	}
	return m, nil
}

func (m *Model) totalItems() int {
	if m.mode == viewRoutes {
		return len(m.routes)
	}
	return len(m.ifaces)
}

func (m *Model) moveCursor(delta int) {
	total := m.totalItems()
	if total == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
}

func (m *Model) openAddOverlay() {
	m.addOverlay = true
	m.addField = fieldDest
	m.addDest = ""
	m.addMask = "255.0.0.0"
	m.addGateway = ""
	m.addMetric = "1"
	m.addIfIdx = 0
	if len(m.ifaces) > 0 {
		m.addIfIdx = 0
	}
}

func (m *Model) appendToField(s string) {
	switch m.addField {
	case fieldDest:
		m.addDest += s
	case fieldMask:
		m.addMask += s
	case fieldGateway:
		m.addGateway += s
	case fieldMetric:
		m.addMetric += s
	case fieldIface:
		// 左右切换接口
		if s == "," || s == "." {
			if len(m.ifaces) > 0 {
				m.addIfIdx = (m.addIfIdx + 1) % len(m.ifaces)
			}
		}
	}
}

func (m *Model) backspaceAddField() {
	switch m.addField {
	case fieldDest:
		if len(m.addDest) > 0 {
			m.addDest = m.addDest[:len(m.addDest)-1]
		}
	case fieldMask:
		if len(m.addMask) > 0 {
			m.addMask = m.addMask[:len(m.addMask)-1]
		}
	case fieldGateway:
		if len(m.addGateway) > 0 {
			m.addGateway = m.addGateway[:len(m.addGateway)-1]
		}
	case fieldMetric:
		if len(m.addMetric) > 0 {
			m.addMetric = m.addMetric[:len(m.addMetric)-1]
		}
	}
}

func (m *Model) submitAdd() tea.Cmd {
	dest := net.ParseIP(m.addDest)
	if dest == nil {
		return m.err(fmt.Errorf("无效的目标地址: %s", m.addDest))
	}
	gw := net.ParseIP(m.addGateway)
	if gw == nil {
		return m.err(fmt.Errorf("无效的网关地址: %s", m.addGateway))
	}

	// 解析子网掩码 → 前缀长度
	maskIP := net.ParseIP(m.addMask)
	if maskIP == nil {
		return m.err(fmt.Errorf("无效的子网掩码: %s", m.addMask))
	}
	mask := net.IPMask(maskIP.To4())
	if mask == nil {
		return m.err(fmt.Errorf("仅支持 IPv4 掩码"))
	}
	prefixLen, _ := mask.Size()

	metric := uint32(1)
	if m.addMetric != "" {
		fmt.Sscanf(m.addMetric, "%d", &metric)
	}

	ifIdx := uint32(0)
	if len(m.ifaces) > 0 && m.addIfIdx < len(m.ifaces) {
		ifIdx = uint32(m.ifaces[m.addIfIdx].Index)
	}

	m.addOverlay = false

	return func() tea.Msg {
		err := AddRoute(dest.To4(), uint8(prefixLen), gw.To4(), ifIdx, metric)
		return RouteActionMsg{OK: err == nil, Err: err, IsAdd: true}
	}
}

func (m *Model) deleteSelected() tea.Cmd {
	if m.cursor >= len(m.routes) {
		return nil
	}
	route := m.routes[m.cursor]
	if route.Dest == "0.0.0.0" && route.PrefixLen == 0 {
		return m.err(fmt.Errorf("不能删除默认路由"))
	}
	if !route.IsStatic {
		return m.err(fmt.Errorf("只能删除静态路由"))
	}

	return func() tea.Msg {
		err := DeleteRoute(route.Dest, route.PrefixLen, route.NextHop, route.IfIndex)
		return RouteActionMsg{OK: err == nil, Err: err, IsAdd: false}
	}
}

// saveRoutes 保存当前静态路由到配置文件
func (m *Model) saveRoutes() (*Model, tea.Cmd) {
	var saved []ui.RouteConfig
	for _, r := range m.routes {
		if r.IsStatic && !(r.Dest == "0.0.0.0" && r.PrefixLen == 0) {
			saved = append(saved, ui.RouteConfig{
				Dest:      r.Dest,
				PrefixLen: r.PrefixLen,
				NextHop:   r.NextHop,
				Metric:    r.Metric,
				IfIndex:   r.IfIndex,
				IfName:    r.IfName,
			})
		}
	}
	if len(saved) == 0 {
		return m, m.err(fmt.Errorf("没有可保存的静态路由"))
	}
	m.savedRoutes = saved
	m.msg = fmt.Sprintf("已保存 %d 条静态路由", len(saved))
	m.msgIsErr = false
	return m, tea.Batch(
		tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearMsgMsg{} }),
		ui.UpdateCfgDataCmd("savedRoutes", saved),
	)
}

// loadSavedRoutes 从配置文件加载并应用保存的路由
// 先获取系统当前路由，对比后只添加缺失的
func (m *Model) loadSavedRoutes() (*Model, tea.Cmd) {
	if len(m.savedRoutes) == 0 {
		return m, m.err(fmt.Errorf("没有已保存的路由配置"))
	}
	routes := m.savedRoutes
	return m, func() tea.Msg {
		// 获取系统当前路由，用于去重
		existing, err := GetRoutes()
		if err != nil {
			return RouteActionMsg{Err: fmt.Errorf("获取系统路由失败: %v", err), IsAdd: true}
		}
		// 用 map 记录已存在的路由 key: "dest/prefixLen/nextHop"
		existMap := make(map[string]bool, len(existing))
		for _, r := range existing {
			key := fmt.Sprintf("%s/%d/%s", r.Dest, r.PrefixLen, r.NextHop)
			existMap[key] = true
		}

		var added, skipped int
		var errs []string
		for _, rc := range routes {
			key := fmt.Sprintf("%s/%d/%s", rc.Dest, rc.PrefixLen, rc.NextHop)
			if existMap[key] {
				skipped++
				continue
			}
			dest := net.ParseIP(rc.Dest)
			nextHop := net.ParseIP(rc.NextHop)
			if dest == nil || nextHop == nil {
				errs = append(errs, fmt.Sprintf("%s/%d: 地址无效", rc.Dest, rc.PrefixLen))
				continue
			}
			if err := AddRoute(dest.To4(), rc.PrefixLen, nextHop.To4(), rc.IfIndex, rc.Metric); err != nil {
				errs = append(errs, fmt.Sprintf("%s/%d: %v", rc.Dest, rc.PrefixLen, err))
			} else {
				added++
			}
		}
		if len(errs) > 0 {
			return RouteActionMsg{Err: fmt.Errorf("加载完成，新增 %d 跳过 %d 失败 %d: %s", added, skipped, len(errs), strings.Join(errs, "; ")), IsAdd: true}
		}
		if added == 0 {
			return RouteActionMsg{OK: true, IsAdd: true} // 全部已存在
		}
		return RouteActionMsg{OK: true, IsAdd: true}
	}
}

func (m *Model) View() string {
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	if m.loading {
		return ui.Card("Route Manager", lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⏳ Loading..."), ui.ColAccent, cardWidth)
	}

	if m.addOverlay {
		return m.viewAddOverlay(cardWidth)
	}

	if m.mode == viewIfaces {
		return m.viewIfaces(cardWidth)
	}
	return m.viewRoutes(cardWidth)
}

func (m *Model) viewRoutes(cardWidth int) string {
	platform := "Win"
	if runtime.GOOS == "darwin" {
		platform = "Mac"
	}
	header := fmt.Sprintf("Routes (%d) [%s]", len(m.routes), platform)

	viewH := (m.height - 7) / 2
	if viewH < 1 {
		viewH = 1
	}
	// 内容区总行数（用于填满卡片）
	contentLines := viewH * 2

	total := len(m.routes)
	start := m.cursor - viewH/2
	if start < 0 {
		start = 0
	}
	end := start + viewH
	if end > total {
		end = total
		start = end - viewH
		if start < 0 {
			start = 0
		}
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		r := m.routes[i]
		prefix := "  "
		if i == m.cursor {
			prefix = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸ ")
		}

		destStr := fmt.Sprintf("%-15s /%-2d", r.Dest, r.PrefixLen)
		gwStr := fmt.Sprintf("%-15s", r.NextHop)
		metricStr := fmt.Sprintf("M:%-3d", r.Metric)
		ifStr := r.IfName

		// 第一行：dest/mask gateway metric ifname
		line1 := prefix + lipgloss.NewStyle().Foreground(ui.ColText).Render(destStr) +
			"  " + lipgloss.NewStyle().Foreground(ui.ColSecondary).Render(gwStr) +
			"  " + lipgloss.NewStyle().Foreground(ui.ColMuted).Render(metricStr) +
			"  " + lipgloss.NewStyle().Foreground(ui.ColPrimary).Render(ifStr)
		sb.WriteString(line1 + "\n")

		// 第二行：协议类型标记
		protoStr := "dynamic"
		if r.IsStatic {
			protoStr = lipgloss.NewStyle().Foreground(ui.ColGreen).Render("static")
		} else {
			protoStr = lipgloss.NewStyle().Foreground(ui.ColMuted).Render(protoStr)
		}
		sb.WriteString("    " + protoStr + "\n")
	}

	// 已渲染行数
	renderedLines := (end - start) * 2

	// 状态提示占 1 行
	if m.msg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(ui.ColGreen)
		if m.msgIsErr {
			msgStyle = lipgloss.NewStyle().Foreground(ui.ColRed)
		}
		sb.WriteString(msgStyle.Render("  "+m.msg) + "\n")
		renderedLines++
	}

	// 用空行填满卡片剩余空间
	if renderedLines < contentLines {
		sb.WriteString(strings.Repeat("\n", contentLines-renderedLines))
	}

	content := strings.TrimRight(sb.String(), "\n")
	return ui.Card(header, content, ui.ColSecondary, cardWidth)
}

func (m *Model) viewIfaces(cardWidth int) string {
	header := fmt.Sprintf("Interfaces (%d)", len(m.ifaces))

	contentH := m.height - 7 // Card 开销 4 + tabBar 1 + statusBar 2
	if contentH < 3 {
		contentH = 3
	}

	total := len(m.ifaces)
	if total == 0 {
		return ui.Card(header, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No interfaces"), ui.ColAccent, cardWidth)
	}

	// 计算每个接口占几行（1行名称 + N行地址 + 1行MAC）
	ifaceLines := make([]int, total)
	for i, iface := range m.ifaces {
		lines := 1 + len(iface.Addrs) // 名称行 + 地址行
		if iface.MAC != "" {
			lines++
		}
		ifaceLines[i] = lines
	}

	// 计算总行数和 cursor 所在接口的起始行
	totalLines := 0
	cursorLineStart := 0
	for i := 0; i < total; i++ {
		if i == m.cursor {
			cursorLineStart = totalLines
		}
		totalLines += ifaceLines[i]
	}

	// 确保光标所在接口可见：调整 scroll
	if cursorLineStart < m.scroll {
		m.scroll = cursorLineStart
	}
	cursorLineEnd := cursorLineStart + ifaceLines[m.cursor] - 1
	if cursorLineEnd >= m.scroll+contentH {
		m.scroll = cursorLineEnd - contentH + 1
	}
	if m.scroll > totalLines-contentH {
		m.scroll = totalLines - contentH
	}
	if m.scroll < 0 {
		m.scroll = 0
	}

	// 从 scroll 位置开始渲染
	var sb strings.Builder
	renderedLines := 0
	currentLine := 0
	for i, iface := range m.ifaces {
		if renderedLines >= contentH {
			break
		}

		// 跳过 scroll 之前的行
		if currentLine+ifaceLines[i] <= m.scroll {
			currentLine += ifaceLines[i]
			continue
		}

		// 名称行
		if currentLine >= m.scroll && renderedLines < contentH {
			prefix := "  "
			if i == m.cursor {
				prefix = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸ ")
			}
			upStr := "up"
			upColor := ui.ColGreen
			if !iface.IsUp {
				upStr = "down"
				upColor = ui.ColMuted
			}
			nameStr := fmt.Sprintf("#%-3d %s", iface.Index, iface.Name)
			infoStr := fmt.Sprintf("  mtu=%d", iface.MTU)
			sb.WriteString(prefix +
				lipgloss.NewStyle().Foreground(ui.ColText).Render(nameStr) +
				lipgloss.NewStyle().Foreground(ui.ColMuted).Render(infoStr) +
				"  " + lipgloss.NewStyle().Foreground(upColor).Render(upStr) +
				"\n")
			renderedLines++
		}
		currentLine++

		// 地址行
		for _, addr := range iface.Addrs {
			if currentLine >= m.scroll && renderedLines < contentH {
				var parts string
				if addr.IsIPv6 {
					parts = fmt.Sprintf("%-40s %s", addr.IP, addr.Netmask)
					sb.WriteString("      " + lipgloss.NewStyle().Foreground(ui.ColMuted).Render(parts) + "\n")
				} else {
					parts = fmt.Sprintf("%-15s  mask %-15s  bcast %s", addr.IP, addr.Netmask, addr.Broadcast)
					sb.WriteString("      " + lipgloss.NewStyle().Foreground(ui.ColSecondary).Render(parts) + "\n")
				}
				renderedLines++
			}
			currentLine++
		}
		// MAC
		if iface.MAC != "" {
			if currentLine >= m.scroll && renderedLines < contentH {
				sb.WriteString("      " + lipgloss.NewStyle().Foreground(ui.ColMuted).Render("mac "+iface.MAC) + "\n")
				renderedLines++
			}
			currentLine++
		}
	}

	// 用空行填满
	if renderedLines < contentH {
		sb.WriteString(strings.Repeat("\n", contentH-renderedLines))
	}

	content := strings.TrimRight(sb.String(), "\n")
	return ui.Card(header, content, ui.ColAccent, cardWidth)
}

func (m *Model) viewAddOverlay(cardWidth int) string {
	var sb strings.Builder

	fields := []struct {
		label string
		value string
		field addField
	}{
		{"Destination", m.addDest, fieldDest},
		{"Subnet Mask", m.addMask, fieldMask},
		{"Gateway", m.addGateway, fieldGateway},
		{"Metric", m.addMetric, fieldMetric},
	}

	for _, f := range fields {
		cursor := " "
		if m.addField == f.field {
			cursor = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸")
		}
		label := lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("  %-14s", f.label))
		value := lipgloss.NewStyle().Foreground(ui.ColText).Render("[" + f.value + "]")
		sb.WriteString(cursor + label + value + "\n")
	}

	// 接口选择
	cursor := " "
	if m.addField == fieldIface {
		cursor = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸")
	}
	ifLabel := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Interface     ")
	if m.addIfIdx < len(m.ifaces) {
		iface := m.ifaces[m.addIfIdx]
		ifName := fmt.Sprintf("[%s]", iface.Name)
		sb.WriteString(cursor + ifLabel + lipgloss.NewStyle().Foreground(ui.ColText).Render(ifName) + "\n")
	} else {
		sb.WriteString(cursor + ifLabel + lipgloss.NewStyle().Foreground(ui.ColMuted).Render("[none]") + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Tab:switch field  / ,:switch iface  Enter:confirm  Esc:close"))

	content := sb.String()
	title := lipgloss.NewStyle().Foreground(ui.ColAccent).Render("Add Static Route")
	return ui.Card(title, content, ui.ColAccent, cardWidth)
}
