package ports

import (
	"strings"
	"testing"

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
	case "delete":
		return tea.KeyDelete
	}
	return []rune(s)[0]
}

func newPortsModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

func TestPortsInit(t *testing.T) {
	m := newPortsModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init 返回 nil Cmd")
	}
	if !m.loading {
		t.Error("Init 后 loading 应为 true")
	}
}

func TestPortsUpdatePortsMsg(t *testing.T) {
	m := newPortsModel()
	m.loading = true

	// 正常数据
	nm, _ := m.Update(PortsMsg{Ports: []PortInfo{{Port: 80, Service: "HTTP", Open: true}}})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if len(m.ports) != 1 {
		t.Errorf("ports = %v, want 1 项", m.ports)
	}
	if !m.loaded {
		t.Error("PortsMsg 后 loaded 应为 true")
	}
	if m.loading {
		t.Error("PortsMsg 后 loading 应为 false")
	}

	// 错误数据
	m2 := newPortsModel()
	m2.Update(PortsMsg{Err: errTest})
	if m2.err == nil {
		t.Error("PortsMsg 带错误时应设置 err")
	}
	if m2.ports != nil {
		t.Errorf("错误时 ports 应为 nil, got %v", m2.ports)
	}
}

var errTest = &customErr{}

type customErr struct{}

func (e *customErr) Error() string { return "scan failed" }

func TestPortsUpdateKeyNavigation(t *testing.T) {
	m := newPortsModel()
	m.scroll = 3
	m.Update(kp("up"))
	if m.scroll != 2 {
		t.Errorf("up 后 scroll = %d, want 2", m.scroll)
	}
	m.Update(kp("home"))
	if m.scroll != 0 {
		t.Errorf("home 后 scroll = %d, want 0", m.scroll)
	}
	m.Update(kp("end"))
	if m.scroll != 1<<30 {
		t.Errorf("end 后 scroll = %d, want 1<<30", m.scroll)
	}
	m.Update(kp("down"))
	if m.scroll != 1<<30+1 {
		t.Errorf("down 后 scroll = %d, want 1<<30+1", m.scroll)
	}

	// scroll=0 up 不再减
	m2 := newPortsModel()
	m2.Update(kp("up"))
	if m2.scroll != 0 {
		t.Errorf("scroll=0 时 up 后 = %d, want 0", m2.scroll)
	}
}

func TestPortsUpdateRefreshAndInput(t *testing.T) {
	// ctrl+r 刷新
	m := newPortsModel()
	nm, cmd := m.Update(kp("ctrl+r"))
	if nm != m {
		t.Error("ctrl+r 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("ctrl+r 应返回 ScanPortsCmd")
	}
	if !m.loading {
		t.Error("ctrl+r 后 loading 应为 true")
	}
	if m.err != nil {
		t.Errorf("ctrl+r 后 err 应为 nil, got %v", m.err)
	}

	// / 打开输入框
	m2 := newPortsModel()
	_, cmd2 := m2.Update(kp("/"))
	if cmd2 != nil {
		t.Error("按 / 不应返回 Cmd")
	}
	if !m2.input.Active {
		t.Error("按 / 后应打开输入框")
	}
	if m2.input.Prompt != "Port:" {
		t.Errorf("Prompt = %q, want %q", m2.input.Prompt, "Port:")
	}
}

func TestPortsInputActive(t *testing.T) {
	m := newPortsModel()
	if m.InputActive() {
		t.Error("初始状态 InputActive 应为 false")
	}
	m.input.Active = true
	if !m.InputActive() {
		t.Error("输入框活跃时 InputActive 应为 true")
	}
}

func TestPortsViewStates(t *testing.T) {
	// 输入框模式
	m := newPortsModel()
	m.input.Active = true
	m.input.Prompt = "Port:"
	m.input.Value = "8080"
	m.input.Cursor = 4
	v := m.View()
	if !strings.Contains(v, "Add Port") {
		t.Errorf("输入框模式视图缺少标题: %q", v)
	}

	// 加载中
	m2 := newPortsModel()
	m2.loading = true
	v2 := m2.View()
	if !strings.Contains(v2, "Scanning") {
		t.Errorf("加载中视图缺少 Scanning: %q", v2)
	}

	// 错误
	m3 := newPortsModel()
	m3.err = errTest
	v3 := m3.View()
	if !strings.Contains(v3, "scan failed") {
		t.Errorf("错误视图缺少错误信息: %q", v3)
	}

	// 主列表
	m4 := newPortsModel()
	m4.loaded = true
	m4.ports = []PortInfo{{Port: 22, Service: "SSH", Open: true}, {Port: 8080, Service: "Custom", Open: false}}
	v4 := m4.View()
	for _, want := range []string{"1 open / 2 total", "Status", "Port", "SSH", "Custom"} {
		if !strings.Contains(v4, want) {
			t.Errorf("主视图缺少 %q", want)
		}
	}

	// 空端口不 panic
	m5 := newPortsModel()
	m5.loaded = true
	if v5 := m5.View(); v5 == "" {
		t.Error("空端口视图为空")
	}

	// 小高度不 panic
	m6 := newPortsModel()
	m6.UpdateSize(60, 4)
	m6.loaded = true
	m6.ports = []PortInfo{{Port: 22, Service: "SSH", Open: true}}
	if v6 := m6.View(); v6 == "" {
		t.Error("小高度视图为空")
	}
}
