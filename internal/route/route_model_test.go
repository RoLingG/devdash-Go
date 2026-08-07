package route

import (
	"strings"
	"testing"

	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
)

// kp 构造一个 tea.KeyPressMsg，使 msg.String() 与项目代码匹配
func kp(s string) tea.KeyPressMsg {
	var mod tea.KeyMod
	if strings.HasPrefix(s, "ctrl+") {
		mod = tea.ModCtrl
		s = strings.TrimPrefix(s, "ctrl+")
	}
	code := keyCode(s)
	if mod != 0 {
		return tea.KeyPressMsg{Mod: mod, Code: code}
	}
	return tea.KeyPressMsg{Code: code, Text: s}
}

// keyCode 返回按键名对应的 rune code
func keyCode(s string) rune {
	switch s {
	case "up":
		return tea.KeyUp
	case "down":
		return tea.KeyDown
	case "enter":
		return tea.KeyEnter
	case "esc":
		return tea.KeyEsc
	case "home":
		return tea.KeyHome
	case "end":
		return tea.KeyEnd
	case "tab":
		return tea.KeyTab
	case "backspace":
		return tea.KeyBackspace
	case "pgup":
		return tea.KeyPgUp
	case "pgdown":
		return tea.KeyPgDown
	}
	return []rune(s)[0]
}

func newRouteModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

var errRoute = &routeErr{}

type routeErr struct{}

func (e *routeErr) Error() string { return "route call failed" }

func TestRouteInit(t *testing.T) {
	m := newRouteModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init 返回 nil Cmd")
	}
	if !m.loading {
		t.Error("Init 后 loading 应为 true")
	}
}

func TestRouteSetSavedRoutes(t *testing.T) {
	m := newRouteModel()
	m.SetSavedRoutes([]ui.RouteConfig{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1"}})
	if len(m.savedRoutes) != 1 || m.savedRoutes[0].Dest != "10.0.0.0" {
		t.Errorf("savedRoutes = %v", m.savedRoutes)
	}
}

func TestRouteUpdateRoutesMsg(t *testing.T) {
	// 正常数据
	m := newRouteModel()
	routes := []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1", Metric: 1, IfIndex: 1, IfName: "eth0", IsStatic: true}}
	ifaces := []IfaceInfo{{Index: 1, Name: "eth0"}}
	nm, cmd := m.Update(RoutesMsg{Routes: routes, Ifaces: ifaces})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("RoutesMsg 不应返回 Cmd")
	}
	if !m.loaded {
		t.Error("RoutesMsg 后 loaded 应为 true")
	}
	if m.loading {
		t.Error("RoutesMsg 后 loading 应为 false")
	}
	if len(m.routes) != 1 || len(m.ifaces) != 1 {
		t.Errorf("routes/ifaces = %d/%d, want 1/1", len(m.routes), len(m.ifaces))
	}

	// 错误 → 显示错误
	m2 := newRouteModel()
	_, cmd2 := m2.Update(RoutesMsg{Err: errRoute})
	if cmd2 == nil {
		t.Error("RoutesMsg 带错误时应返回 err Cmd")
	}
	if !m2.msgIsErr {
		t.Error("RoutesMsg 带错误时 msgIsErr 应为 true")
	}
	if m2.msg == "" {
		t.Error("RoutesMsg 带错误时应设置 msg")
	}
}

func TestRouteUpdateClearMsg(t *testing.T) {
	m := newRouteModel()
	m.msg = "hello"
	nm, cmd := m.Update(clearMsgMsg{})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("clearMsgMsg 不应返回 Cmd")
	}
	if m.msg != "" {
		t.Errorf("clearMsgMsg 后 msg = %q, want empty", m.msg)
	}
}

func TestRouteUpdateActionMsg(t *testing.T) {
	// 添加成功
	m := newRouteModel()
	_, cmd := m.Update(RouteActionMsg{OK: true, IsAdd: true})
	if cmd == nil {
		t.Fatal("RouteActionMsg 成功应返回刷新 Cmd")
	}
	if m.msg != "Route added successfully" {
		t.Errorf("msg = %q, want Route added successfully", m.msg)
	}
	if m.msgIsErr {
		t.Error("成功时 msgIsErr 应为 false")
	}

	// 删除成功
	m2 := newRouteModel()
	_, cmd2 := m2.Update(RouteActionMsg{OK: true, IsAdd: false})
	if cmd2 == nil {
		t.Fatal("RouteActionMsg 删除成功应返回刷新 Cmd")
	}
	if m2.msg != "Route deleted successfully" {
		t.Errorf("msg = %q, want Route deleted successfully", m2.msg)
	}

	// 错误
	m3 := newRouteModel()
	_, cmd3 := m3.Update(RouteActionMsg{Err: errRoute, IsAdd: true})
	if cmd3 == nil {
		t.Error("RouteActionMsg 错误应返回 err Cmd")
	}
	if !m3.msgIsErr {
		t.Error("RouteActionMsg 错误时 msgIsErr 应为 true")
	}
}

func TestRouteHandleKey(t *testing.T) {
	// ctrl+r 刷新
	m := newRouteModel()
	m.loading = false
	_, cmd := m.Update(kp("ctrl+r"))
	if cmd == nil {
		t.Fatal("ctrl+r 应返回 Cmd")
	}
	if !m.loading {
		t.Error("ctrl+r 后 loading 应为 true")
	}

	// tab 切换视图
	m2 := newRouteModel()
	m2.mode = viewRoutes
	m2.cursor = 2
	m2.Update(kp("tab"))
	if m2.mode != viewIfaces {
		t.Errorf("tab 后 mode = %v, want viewIfaces", m2.mode)
	}
	if m2.cursor != 0 {
		t.Error("tab 后 cursor 应重置为 0")
	}
	m2.Update(kp("tab"))
	if m2.mode != viewRoutes {
		t.Errorf("再次 tab 后 mode = %v, want viewRoutes", m2.mode)
	}

	// ctrl+a 管理员 → 打开添加面板
	m3 := newRouteModel()
	m3.isAdmin = true
	m3.ifaces = []IfaceInfo{{Index: 1, Name: "eth0"}}
	_, cmd3 := m3.Update(kp("ctrl+a"))
	if cmd3 != nil {
		t.Error("ctrl+a 不应返回 Cmd")
	}
	if !m3.addOverlay {
		t.Error("ctrl+a 管理员应打开添加面板")
	}

	// ctrl+a 非管理员 → 错误
	m4 := newRouteModel()
	m4.isAdmin = false
	_, cmd4 := m4.Update(kp("ctrl+a"))
	if cmd4 == nil {
		t.Fatal("ctrl+a 非管理员应返回 err Cmd")
	}
	if !m4.msgIsErr {
		t.Error("ctrl+a 非管理员 msgIsErr 应为 true")
	}

	// ctrl+d 非管理员 → 错误
	m5 := newRouteModel()
	m5.isAdmin = false
	_, cmd5 := m5.Update(kp("ctrl+d"))
	if cmd5 == nil {
		t.Fatal("ctrl+d 非管理员应返回 err Cmd")
	}

	// ctrl+d 管理员 + 静态路由 → 删除
	m6 := newRouteModel()
	m6.isAdmin = true
	m6.mode = viewRoutes
	m6.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1", IfIndex: 1, IsStatic: true}}
	m6.cursor = 0
	_, cmd6 := m6.Update(kp("ctrl+d"))
	if cmd6 == nil {
		t.Fatal("ctrl+d 静态路由应返回删除 Cmd")
	}

	// ctrl+d 管理员 + 默认路由 → 错误
	m6b := newRouteModel()
	m6b.isAdmin = true
	m6b.mode = viewRoutes
	m6b.routes = []RouteEntry{{Dest: "0.0.0.0", PrefixLen: 0}}
	m6b.cursor = 0
	_, cmd6b := m6b.Update(kp("ctrl+d"))
	if cmd6b == nil {
		t.Fatal("ctrl+d 默认路由应返回 err Cmd")
	}
	if !m6b.msgIsErr {
		t.Error("ctrl+d 默认路由 msgIsErr 应为 true")
	}

	// ctrl+s 保存静态路由
	m7 := newRouteModel()
	m7.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1", Metric: 1, IfIndex: 1, IfName: "eth0", IsStatic: true}}
	_, cmd7 := m7.Update(kp("ctrl+s"))
	if cmd7 == nil {
		t.Fatal("ctrl+s 应返回 Batch Cmd")
	}
	if len(m7.savedRoutes) != 1 {
		t.Errorf("savedRoutes = %d, want 1", len(m7.savedRoutes))
	}
	if m7.msg == "" {
		t.Error("ctrl+s 后应设置成功提示")
	}

	// ctrl+s 无静态路由 → 错误
	m7b := newRouteModel()
	m7b.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, IsStatic: false}}
	_, cmd7b := m7b.Update(kp("ctrl+s"))
	if cmd7b == nil {
		t.Fatal("ctrl+s 无静态路由应返回 err Cmd")
	}
	if !m7b.msgIsErr {
		t.Error("ctrl+s 无静态路由 msgIsErr 应为 true")
	}

	// ctrl+l 非管理员 → 错误
	m8 := newRouteModel()
	m8.isAdmin = false
	_, cmd8 := m8.Update(kp("ctrl+l"))
	if cmd8 == nil {
		t.Fatal("ctrl+l 非管理员应返回 err Cmd")
	}
	if !m8.msgIsErr {
		t.Error("ctrl+l 非管理员 msgIsErr 应为 true")
	}

	// ctrl+l 管理员 + 有保存路由 → Cmd
	m9 := newRouteModel()
	m9.isAdmin = true
	m9.savedRoutes = []ui.RouteConfig{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1"}}
	_, cmd9 := m9.Update(kp("ctrl+l"))
	if cmd9 == nil {
		t.Fatal("ctrl+l 有保存路由应返回 Cmd")
	}

	// ctrl+l 管理员 + 无保存路由 → 错误
	m9b := newRouteModel()
	m9b.isAdmin = true
	_, cmd9b := m9b.Update(kp("ctrl+l"))
	if cmd9b == nil {
		t.Fatal("ctrl+l 无保存路由应返回 err Cmd")
	}
	if !m9b.msgIsErr {
		t.Error("ctrl+l 无保存路由 msgIsErr 应为 true")
	}
}

func TestRouteNavigation(t *testing.T) {
	m := newRouteModel()
	m.mode = viewRoutes
	m.routes = []RouteEntry{{}, {}, {}}
	m.cursor = 1
	m.Update(kp("down"))
	if m.cursor != 2 {
		t.Errorf("down 后 cursor = %d, want 2", m.cursor)
	}
	m.Update(kp("up"))
	if m.cursor != 1 {
		t.Errorf("up 后 cursor = %d, want 1", m.cursor)
	}
	m.Update(kp("home"))
	if m.cursor != 0 {
		t.Errorf("home 后 cursor = %d, want 0", m.cursor)
	}
	m.Update(kp("end"))
	if m.cursor != 2 {
		t.Errorf("end 后 cursor = %d, want 2", m.cursor)
	}
	m.Update(kp("pgdown"))
	if m.cursor != 2 {
		t.Errorf("pgdown 后 cursor = %d, want 2（越界 clamp）", m.cursor)
	}
	m.Update(kp("pgup"))
	if m.cursor != 0 {
		t.Errorf("pgup 后 cursor = %d, want 0", m.cursor)
	}

	// k / j 别名
	m.Update(kp("j"))
	if m.cursor != 1 {
		t.Errorf("j 后 cursor = %d, want 1", m.cursor)
	}
	m.Update(kp("k"))
	if m.cursor != 0 {
		t.Errorf("k 后 cursor = %d, want 0", m.cursor)
	}

	// 无数据时 moveCursor 不动
	m2 := newRouteModel()
	m2.mode = viewRoutes
	m2.cursor = 3
	m2.Update(kp("up"))
	if m2.cursor != 3 {
		t.Errorf("无数据 up 后 cursor = %d, want 3（不变）", m2.cursor)
	}
}

func TestRouteAddOverlayKeys(t *testing.T) {
	// esc 关闭
	m := newRouteModel()
	m.addOverlay = true
	_, cmd := m.Update(kp("esc"))
	if cmd != nil {
		t.Error("esc 不应返回 Cmd")
	}
	if m.addOverlay {
		t.Error("esc 后 addOverlay 应为 false")
	}

	// tab 循环字段
	m2 := newRouteModel()
	m2.addOverlay = true
	m2.addField = fieldIface
	m2.Update(kp("tab"))
	if m2.addField != fieldDest {
		t.Errorf("tab 后 addField = %v, want fieldDest", m2.addField)
	}

	// backspace
	m3 := newRouteModel()
	m3.addOverlay = true
	m3.addField = fieldDest
	m3.addDest = "10.0"
	m3.Update(kp("backspace"))
	if m3.addDest != "10." {
		t.Errorf("backspace 后 addDest = %q, want 10.", m3.addDest)
	}

	// 打印字符追加
	m4 := newRouteModel()
	m4.addOverlay = true
	m4.addField = fieldDest
	m4.Update(kp("1"))
	m4.Update(kp("0"))
	if m4.addDest != "10" {
		t.Errorf("输入后 addDest = %q, want 10", m4.addDest)
	}

	// 接口字段 , 切换
	m5 := newRouteModel()
	m5.addOverlay = true
	m5.addField = fieldIface
	m5.ifaces = []IfaceInfo{{Index: 1, Name: "a"}, {Index: 2, Name: "b"}}
	m5.addIfIdx = 0
	m5.Update(kp(","))
	if m5.addIfIdx != 1 {
		t.Errorf(", 后 addIfIdx = %d, want 1", m5.addIfIdx)
	}
}

func TestRouteSubmitAdd(t *testing.T) {
	// 无效目标地址 → 错误，overlay 保持
	m := newRouteModel()
	m.addOverlay = true
	m.addDest = "not-an-ip"
	_, cmd := m.Update(kp("enter"))
	if cmd == nil {
		t.Fatal("无效地址应返回 err Cmd")
	}
	if !m.msgIsErr {
		t.Error("无效地址 msgIsErr 应为 true")
	}
	if !m.addOverlay {
		t.Error("无效地址时 addOverlay 应保持 true")
	}

	// 无效掩码 → 错误
	m2 := newRouteModel()
	m2.addOverlay = true
	m2.addDest = "10.0.0.0"
	m2.addMask = "bad"
	m2.addGateway = "10.0.0.1"
	_, cmd2 := m2.Update(kp("enter"))
	if cmd2 == nil || !m2.msgIsErr {
		t.Error("无效掩码应返回 err Cmd")
	}

	// 有效输入 → 关闭 overlay + 返回 AddRoute Cmd
	m3 := newRouteModel()
	m3.addOverlay = true
	m3.addDest = "10.0.0.0"
	m3.addMask = "255.0.0.0"
	m3.addGateway = "10.0.0.1"
	m3.ifaces = []IfaceInfo{{Index: 5, Name: "eth0"}}
	m3.addIfIdx = 0
	_, cmd3 := m3.Update(kp("enter"))
	if cmd3 == nil {
		t.Fatal("有效输入应返回 AddRoute Cmd")
	}
	if m3.addOverlay {
		t.Error("有效输入后 addOverlay 应为 false")
	}
}

func TestRouteDeleteSelected(t *testing.T) {
	// cursor 越界 → nil
	m := newRouteModel()
	m.routes = []RouteEntry{{IsStatic: true}}
	m.cursor = 5
	if cmd := m.deleteSelected(); cmd != nil {
		t.Error("cursor 越界应返回 nil Cmd")
	}

	// 默认路由 → 错误
	m2 := newRouteModel()
	m2.routes = []RouteEntry{{Dest: "0.0.0.0", PrefixLen: 0}}
	m2.cursor = 0
	if cmd := m2.deleteSelected(); cmd == nil || !m2.msgIsErr {
		t.Error("默认路由应返回 err Cmd")
	}

	// 非静态 → 错误
	m3 := newRouteModel()
	m3.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, IsStatic: false}}
	m3.cursor = 0
	if cmd := m3.deleteSelected(); cmd == nil || !m3.msgIsErr {
		t.Error("非静态路由应返回 err Cmd")
	}

	// 静态 → DeleteRoute Cmd
	m4 := newRouteModel()
	m4.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1", IfIndex: 1, IsStatic: true}}
	m4.cursor = 0
	if cmd := m4.deleteSelected(); cmd == nil {
		t.Error("静态路由应返回 DeleteRoute Cmd")
	}
}

func TestRouteOpenAddOverlay(t *testing.T) {
	m := newRouteModel()
	m.ifaces = []IfaceInfo{{Index: 1, Name: "eth0"}}
	m.openAddOverlay()
	if !m.addOverlay {
		t.Error("openAddOverlay 后 addOverlay 应为 true")
	}
	if m.addField != fieldDest {
		t.Errorf("addField = %v, want fieldDest", m.addField)
	}
	if m.addMask != "255.0.0.0" || m.addMetric != "1" {
		t.Errorf("addMask/addMetric = %q/%q, want 255.0.0.0/1", m.addMask, m.addMetric)
	}
}

func TestRouteInputActive(t *testing.T) {
	m := newRouteModel()
	if m.InputActive() {
		t.Error("默认 InputActive 应为 false")
	}
	m.input.Active = true
	if !m.InputActive() {
		t.Error("input.Active 时 InputActive 应为 true")
	}
	m.input.Active = false
	m.addOverlay = true
	if !m.InputActive() {
		t.Error("addOverlay 时 InputActive 应为 true")
	}
}

func TestRouteViewStates(t *testing.T) {
	// 加载中
	m := newRouteModel()
	m.loading = true
	if v := m.View(); !strings.Contains(v, "Loading") {
		t.Errorf("加载视图缺少内容: %q", v)
	}

	// 添加面板
	m2 := newRouteModel()
	m2.addOverlay = true
	if v := m2.View(); !strings.Contains(v, "Add Static Route") {
		t.Errorf("添加面板视图缺少标题: %q", v)
	}

	// 接口视图
	m3 := newRouteModel()
	m3.mode = viewIfaces
	m3.ifaces = []IfaceInfo{{Index: 1, Name: "eth0", IsUp: true}}
	if v := m3.View(); !strings.Contains(v, "Interfaces") {
		t.Errorf("接口视图缺少标题: %q", v)
	}

	// 路由视图
	m4 := newRouteModel()
	m4.mode = viewRoutes
	m4.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1", IfName: "eth0", IsStatic: true}}
	if v := m4.View(); !strings.Contains(v, "Routes") {
		t.Errorf("路由视图缺少标题: %q", v)
	}
}

func TestRouteViewRoutes(t *testing.T) {
	m := newRouteModel()
	m.mode = viewRoutes
	m.routes = []RouteEntry{
		{Dest: "10.0.0.0", PrefixLen: 8, NextHop: "10.0.0.1", Metric: 1, IfIndex: 1, IfName: "eth0", IsStatic: true},
		{Dest: "192.168.0.0", PrefixLen: 16, NextHop: "192.168.1.1", Metric: 2, IfIndex: 1, IfName: "eth0", IsStatic: false},
	}
	m.cursor = 0
	v := m.View()
	for _, want := range []string{"Routes", "(2)", "10.0.0.0", "/8", "10.0.0.1", "eth0", "static", "dynamic", "M:1"} {
		if !strings.Contains(v, want) {
			t.Errorf("路由视图缺少 %q: %q", want, v)
		}
	}

	// 状态提示（成功）
	m2 := newRouteModel()
	m2.mode = viewRoutes
	m2.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, IsStatic: true}}
	m2.msg = "Route added successfully"
	m2.msgIsErr = false
	if v := m2.View(); !strings.Contains(v, "Route added successfully") {
		t.Errorf("成功提示缺失: %q", v)
	}

	// 状态提示（错误）
	m3 := newRouteModel()
	m3.mode = viewRoutes
	m3.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, IsStatic: true}}
	m3.msg = "boom"
	m3.msgIsErr = true
	if v := m3.View(); !strings.Contains(v, "boom") {
		t.Errorf("错误提示缺失: %q", v)
	}

	// 空路由
	m4 := newRouteModel()
	m4.mode = viewRoutes
	if v := m4.View(); !strings.Contains(v, "(0)") {
		t.Errorf("空路由视图缺少计数: %q", v)
	}

	// 小高度不 panic
	m5 := newRouteModel()
	m5.UpdateSize(100, 8)
	m5.mode = viewRoutes
	m5.routes = []RouteEntry{{Dest: "10.0.0.0", PrefixLen: 8, IsStatic: true}}
	if v := m5.View(); v == "" {
		t.Error("小高度路由视图为空")
	}
}

func TestRouteViewIfaces(t *testing.T) {
	m := newRouteModel()
	m.mode = viewIfaces
	m.ifaces = []IfaceInfo{
		{
			Index: 1, Name: "eth0", IsUp: true, MAC: "aa:bb:cc:dd:ee:ff", MTU: 1500,
			Addrs: []IfaceAddr{
				{IP: "10.0.0.2", Netmask: "255.255.255.0", Broadcast: "10.0.0.255"},
				{IP: "fe80::1", Netmask: "/64", IsIPv6: true},
			},
		},
	}
	m.cursor = 0
	v := m.View()
	for _, want := range []string{"Interfaces", "(1)", "eth0", "up", "mtu=1500", "10.0.0.2", "bcast 10.0.0.255", "fe80::1", "mac aa:bb:cc:dd:ee:ff"} {
		if !strings.Contains(v, want) {
			t.Errorf("接口视图缺少 %q: %q", want, v)
		}
	}

	// down 状态
	m2 := newRouteModel()
	m2.mode = viewIfaces
	m2.ifaces = []IfaceInfo{{Index: 2, Name: "eth1", IsUp: false, Addrs: []IfaceAddr{{IP: "0.0.0.0", Netmask: "/0"}}}}
	if v := m2.View(); !strings.Contains(v, "down") {
		t.Errorf("down 接口缺少标记: %q", v)
	}

	// 空列表
	m3 := newRouteModel()
	m3.mode = viewIfaces
	if v := m3.View(); !strings.Contains(v, "No interfaces") {
		t.Errorf("空接口视图缺少提示: %q", v)
	}

	// 小高度不 panic
	m4 := newRouteModel()
	m4.UpdateSize(100, 8)
	m4.mode = viewIfaces
	m4.ifaces = []IfaceInfo{{Index: 1, Name: "eth0", Addrs: []IfaceAddr{{IP: "1.2.3.4", Netmask: "/24", Broadcast: "1.2.3.255"}}}}
	if v := m4.View(); v == "" {
		t.Error("小高度接口视图为空")
	}
}

func TestRouteViewAddOverlay(t *testing.T) {
	m := newRouteModel()
	m.addOverlay = true
	m.addField = fieldDest
	m.addDest = "10.0.0.0"
	m.addMask = "255.0.0.0"
	m.addGateway = "10.0.0.1"
	m.addMetric = "1"
	m.ifaces = []IfaceInfo{{Index: 1, Name: "eth0"}}
	m.addIfIdx = 0
	v := m.View()
	for _, want := range []string{"Add Static Route", "Destination", "Subnet Mask", "Gateway", "Metric", "Interface", "eth0", "Tab:switch field"} {
		if !strings.Contains(v, want) {
			t.Errorf("添加面板缺少 %q: %q", want, v)
		}
	}

	// 无接口 → [none]
	m2 := newRouteModel()
	m2.addOverlay = true
	if v := m2.View(); !strings.Contains(v, "[none]") {
		t.Errorf("无接口时应显示 [none]: %q", v)
	}
}
