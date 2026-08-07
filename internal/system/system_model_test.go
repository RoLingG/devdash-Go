package system

import (
	"strings"
	"testing"
	"time"

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
	}
	return []rune(s)[0]
}

func newSystemModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

// sampleSysInfo 构造带数据的 SysInfoMsg
func sampleSysInfo() SysInfoMsg {
	return SysInfoMsg{
		CPU: CPUInfo{Overall: 42.5, PerCore: []float64{10, 20, 30, 40}},
		Mem: MemInfo{Used: 8 * 1024 * 1024 * 1024, Total: 16 * 1024 * 1024 * 1024, Percent: 50},
		Disks: []DiskInfo{
			{Mount: "C:\\", Used: 100 * 1024 * 1024 * 1024, Total: 200 * 1024 * 1024 * 1024, Percent: 50},
		},
	}
}

func TestSystemInit(t *testing.T) {
	m := newSystemModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init 返回 nil Cmd")
	}
	if !m.loading {
		t.Error("Init 后 loading 应为 true")
	}
}

func TestSystemUpdateSysInfoMsg(t *testing.T) {
	// 正常数据
	m := newSystemModel()
	m.loading = true
	nm, _ := m.Update(sampleSysInfo())
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if !m.loaded {
		t.Error("SysInfoMsg 后 loaded 应为 true")
	}
	if m.loading {
		t.Error("SysInfoMsg 后 loading 应为 false")
	}
	if m.sysInfo.CPU.Overall != 42.5 {
		t.Errorf("CPU.Overall = %v, want 42.5", m.sysInfo.CPU.Overall)
	}

	// 错误数据
	m2 := newSystemModel()
	m2.Update(SysInfoMsg{Err: errSystem})
	if m2.err == nil {
		t.Error("SysInfoMsg 带错误时应设置 err")
	}
}

var errSystem = &systemErr{}

type systemErr struct{}

func (e *systemErr) Error() string { return "system call failed" }

func TestSystemUpdateProcMsg(t *testing.T) {
	// 正常数据 → 应用过滤
	m := newSystemModel()
	procs := []ProcessInfo{
		{PID: 1, Name: "chrome.exe", CPU: 20, MemMB: 100},
		{PID: 2, Name: "go.exe", CPU: 60, MemMB: 50},
	}
	m.filter = "go"
	nm, _ := m.Update(ProcMsg{Processes: procs})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if len(m.process) != 2 {
		t.Errorf("process = %d 项, want 2", len(m.process))
	}
	// filter "go" 只匹配 go.exe（chrome 不含 "go"）
	if len(m.filtered) != 1 || m.filtered[0].Name != "go.exe" {
		t.Errorf("filtered = %v, want 仅 go.exe", m.filtered)
	}

	// 错误数据不更新
	m2 := newSystemModel()
	m2.Update(ProcMsg{Err: errSystem})
	if len(m2.process) != 0 {
		t.Errorf("ProcMsg 带错误时 process 不应更新: %v", m2.process)
	}
}

func TestSystemUpdateSysTickMsg(t *testing.T) {
	m := newSystemModel()
	nm, cmd := m.Update(SysTickMsg{})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("SysTickMsg 应返回 Batch Cmd")
	}
}

func TestSystemUpdateKeys(t *testing.T) {
	// tab 切换子视图
	m := newSystemModel()
	m.Update(kp("tab"))
	if m.view != viewProcess {
		t.Errorf("tab 后 view = %v, want viewProcess", m.view)
	}
	m.Update(kp("tab"))
	if m.view != viewOverview {
		t.Errorf("再次 tab 后 view = %v, want viewOverview", m.view)
	}

	// ctrl+r 刷新
	m2 := newSystemModel()
	nm, cmd := m2.Update(kp("ctrl+r"))
	if nm != m2 {
		t.Error("ctrl+r 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("ctrl+r 应返回 Batch Cmd")
	}
	if !m2.loading {
		t.Error("ctrl+r 后 loading 应为 true")
	}

	// / 打开过滤输入框
	m3 := newSystemModel()
	_, cmd3 := m3.Update(kp("/"))
	if cmd3 != nil {
		t.Error("按 / 不应返回 Cmd")
	}
	if !m3.input.Active {
		t.Error("按 / 后应打开输入框")
	}
	if m3.input.Prompt != "Filter:" {
		t.Errorf("Prompt = %q, want Filter:", m3.input.Prompt)
	}

	// ctrl+u 清空过滤
	m4 := newSystemModel()
	m4.process = []ProcessInfo{{PID: 1, Name: "a.exe", CPU: 10, MemMB: 10}}
	m4.filter = "b"
	m4.applyFilter()
	m4.Update(kp("ctrl+u"))
	if m4.filter != "" {
		t.Errorf("ctrl+u 后 filter = %q, want empty", m4.filter)
	}
	if len(m4.filtered) != 1 {
		t.Errorf("ctrl+u 后 filtered = %d, want 1", len(m4.filtered))
	}

	// 滚动导航
	m5 := newSystemModel()
	m5.scroll = 3
	m5.Update(kp("up"))
	if m5.scroll != 2 {
		t.Errorf("up 后 scroll = %d, want 2", m5.scroll)
	}
	m5.Update(kp("home"))
	if m5.scroll != 0 {
		t.Errorf("home 后 scroll = %d, want 0", m5.scroll)
	}
	m5.Update(kp("end"))
	if m5.scroll != 1<<30 {
		t.Errorf("end 后 scroll = %d, want 1<<30", m5.scroll)
	}
}

func TestSystemApplyFilter(t *testing.T) {
	m := newSystemModel()
	m.process = []ProcessInfo{
		{PID: 1, Name: "chrome.exe", CPU: 20, MemMB: 100},
		{PID: 2, Name: "goland.exe", CPU: 60, MemMB: 50},
	}
	// 空 filter → 全部
	m.applyFilter()
	if len(m.filtered) != 2 {
		t.Errorf("空 filter filtered = %d, want 2", len(m.filtered))
	}
	// 大小写不敏感
	m.filter = "CHROME"
	m.applyFilter()
	if len(m.filtered) != 1 || m.filtered[0].Name != "chrome.exe" {
		t.Errorf("大小写不敏感过滤失败: %v", m.filtered)
	}
	// 无匹配 → 空
	m.filter = "zzz"
	m.applyFilter()
	if len(m.filtered) != 0 {
		t.Errorf("无匹配 filtered = %d, want 0", len(m.filtered))
	}
}

func TestSystemViewStates(t *testing.T) {
	// 输入框模式
	m := newSystemModel()
	m.input.Active = true
	m.input.Prompt = "Filter:"
	m.input.Value = "go"
	m.input.Cursor = 2
	v := m.View()
	if !strings.Contains(v, "Filter Process") {
		t.Errorf("输入框视图缺少标题: %q", v)
	}

	// 加载中
	m2 := newSystemModel()
	m2.loading = true
	v2 := m2.View()
	if !strings.Contains(v2, "Loading") {
		t.Errorf("加载中视图缺少内容: %q", v2)
	}

	// 错误
	m3 := newSystemModel()
	m3.err = errSystem
	v3 := m3.View()
	if !strings.Contains(v3, "system call failed") {
		t.Errorf("错误视图缺少错误信息: %q", v3)
	}
}

func TestSystemViewOverview(t *testing.T) {
	m := newSystemModel()
	m.loaded = true
	m.sysInfo = sampleSysInfo()
	v := m.View()
	for _, want := range []string{"CPU", "Memory", "Disk", "C:\\"} {
		if !strings.Contains(v, want) {
			t.Errorf("概览视图缺少 %q", want)
		}
	}

	// 窄宽度（2 列核心网格）
	m2 := newSystemModel()
	m2.UpdateSize(50, 40)
	m2.loaded = true
	m2.sysInfo = sampleSysInfo()
	if v2 := m2.View(); v2 == "" {
		t.Error("窄宽度概览视图为空")
	}
}

func TestSystemViewProcs(t *testing.T) {
	m := newSystemModel()
	m.loaded = true
	m.view = viewProcess
	m.process = []ProcessInfo{
		{PID: 1, Name: "chrome.exe", CPU: 20, MemMB: 100},
		{PID: 2, Name: "go.exe", CPU: 60, MemMB: 50},
	}
	m.applyFilter()
	v := m.View()
	for _, want := range []string{"PID", "Name", "chrome.exe", "go.exe"} {
		if !strings.Contains(v, want) {
			t.Errorf("进程视图缺少 %q", want)
		}
	}

	// 空进程列表显示提示
	m2 := newSystemModel()
	m2.loaded = true
	m2.view = viewProcess
	v2 := m2.View()
	if !strings.Contains(v2, "No processes") {
		t.Errorf("空进程视图缺少提示: %q", v2)
	}

	// 过滤提示显示
	m3 := newSystemModel()
	m3.loaded = true
	m3.view = viewProcess
	m3.process = []ProcessInfo{{PID: 1, Name: "a.exe", CPU: 10, MemMB: 10}}
	m3.filter = "a"
	m3.applyFilter()
	v3 := m3.View()
	if !strings.Contains(v3, "Filter: a") {
		t.Errorf("过滤提示缺失: %q", v3)
	}

	// 长进程名截断不 panic
	m4 := newSystemModel()
	m4.loaded = true
	m4.view = viewProcess
	m4.process = []ProcessInfo{{PID: 1, Name: strings.Repeat("x", 40), CPU: 10, MemMB: 10}}
	m4.applyFilter()
	if v4 := m4.View(); v4 == "" {
		t.Error("长名进程视图为空")
	}
}

func TestSystemRenderHelpers(t *testing.T) {
	// renderPercent
	if !strings.Contains(renderPercent(42.5), "42.5%") {
		t.Errorf("renderPercent 缺少百分比: %q", renderPercent(42.5))
	}
	// colorForPercent 阈值分支（不 panic 即可）
	for _, pct := range []float64{95, 75, 50} {
		_ = colorForPercent(pct)
	}
	// renderBar
	if !strings.Contains(renderBar(50, 30), "50.0%") {
		t.Errorf("renderBar 缺少百分比: %q", renderBar(50, 30))
	}
	// renderBar 小宽度边界
	if !strings.Contains(renderBar(120, 5), "120.0%") {
		t.Errorf("renderBar 超 100%% 时缺少百分比: %q", renderBar(120, 5))
	}
	// 负百分比
	if !strings.Contains(renderBar(-5, 30), "-5.0%") {
		t.Errorf("renderBar 负百分比时缺少百分比: %q", renderBar(-5, 30))
	}
}

func TestSystemRefreshTick(t *testing.T) {
	// Init 的 SysTickCmd 延迟触发 SysTickMsg
	cmd := SysTickCmd(5 * time.Millisecond)
	if cmd == nil {
		t.Fatal("SysTickCmd 返回 nil")
	}
	if _, ok := cmd().(SysTickMsg); !ok {
		t.Errorf("SysTickCmd 消息类型错误")
	}
}
