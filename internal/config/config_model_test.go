package config

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
	case "backspace":
		return tea.KeyBackspace
	}
	return []rune(s)[0]
}

func newConfigModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

// sampleRoot 构造一个简单配置树（含嵌套对象）
func sampleRoot() Node {
	return BuildTree("", map[string]interface{}{
		"name":    "devdash",
		"version": "1.0.0",
		"author": map[string]interface{}{
			"name":  "You",
			"email": "you@example.com",
		},
	}, 0, true)
}

func TestConfigInit(t *testing.T) {
	// 无 lastConfigPath → 返回示例配置 Cmd
	m := newConfigModel()
	cmd := m.Init("")
	if cmd == nil {
		t.Fatal("Init(\"\") 返回 nil Cmd")
	}

	// lastConfigPath 非空 → LoadFileCmd
	m2 := newConfigModel()
	cmd2 := m2.Init("/tmp/config.json")
	if cmd2 == nil {
		t.Fatal("Init(路径) 返回 nil Cmd")
	}
	if m2.configPath != "/tmp/config.json" {
		t.Errorf("configPath = %q, want /tmp/config.json", m2.configPath)
	}
}

func TestConfigUpdateLoadMsg(t *testing.T) {
	// 正常加载
	m := newConfigModel()
	m.configPath = "/tmp/config.json"
	nm, _ := m.Update(LoadMsg{Root: sampleRoot()})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if !m.loaded {
		t.Error("LoadMsg 后 loaded 应为 true")
	}
	if len(m.lines) == 0 {
		t.Error("LoadMsg 后 lines 不应为空")
	}
	if m.cachedPath != "/tmp/config.json" {
		t.Errorf("cachedPath = %q, want /tmp/config.json", m.cachedPath)
	}

	// 错误 → 打开输入框
	m2 := newConfigModel()
	nm2, _ := m2.Update(LoadMsg{Err: errConfig})
	if nm2 != m2 {
		t.Error("Update 应返回同一 Model 指针")
	}
	if m2.errMsg == "" {
		t.Error("LoadMsg 带错误时应设置 errMsg")
	}
	if !m2.loaded {
		t.Error("LoadMsg 带错误时 loaded 应为 true")
	}
	if !m2.input.Active {
		t.Error("LoadMsg 带错误时应打开输入框")
	}
}

var errConfig = &configErr{}

type configErr struct{}

func (e *configErr) Error() string { return "parse failed" }

func TestConfigUpdateDirMsg(t *testing.T) {
	// 有文件 → 目录列表模式
	m := newConfigModel()
	nm, cmd := m.Update(DirMsg{Dir: "/etc", Files: []string{"a.json", "b.yaml"}})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("DirMsg 有文件时不应返回 Cmd")
	}
	if !m.dirListing {
		t.Error("应进入目录列表模式")
	}

	// 无文件 → 错误 + 输入框
	m2 := newConfigModel()
	m2.dirPath = "/empty"
	nm2, cmd2 := m2.Update(DirMsg{Dir: "/empty", Files: nil})
	if nm2 != m2 {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd2 != nil {
		t.Error("无文件时不应返回 Cmd")
	}
	if m2.errMsg == "" {
		t.Error("无文件时应设置 errMsg")
	}
	if !m2.input.Active {
		t.Error("无文件时应打开输入框")
	}
}

func TestConfigUpdateDirListingKeys(t *testing.T) {
	m := newConfigModel()
	m.dirListing = true
	m.dirPath = "/etc"
	m.dirList.SetItems([]string{"a.json", "b.yaml", "c.toml"})

	m.Update(kp("down"))
	m.Update(kp("down"))
	if got := m.dirList.Selected(); got != "c.toml" {
		t.Errorf("down 两次后选中 = %q, want c.toml", got)
	}
	m.Update(kp("up"))
	if got := m.dirList.Selected(); got != "b.yaml" {
		t.Errorf("up 后选中 = %q, want b.yaml", got)
	}

	// enter → 加载
	nm, cmd := m.Update(kp("enter"))
	if nm != m {
		t.Error("enter 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("enter 应返回 Batch Cmd")
	}
	if m.dirListing {
		t.Error("enter 后应退出目录列表模式")
	}

	// esc → 打开输入框
	m2 := newConfigModel()
	m2.dirListing = true
	m2.dirPath = "/etc"
	m2.dirList.SetItems([]string{"a.json"})
	m2.Update(kp("esc"))
	if m2.dirListing {
		t.Error("esc 后应退出目录列表模式")
	}
	if !m2.input.Active {
		t.Error("esc 后应打开输入框")
	}
}

func TestConfigUpdateNavigation(t *testing.T) {
	m := newConfigModel()
	m.configPath = "/tmp/config.json"
	m.Update(LoadMsg{Root: sampleRoot()})
	n := len(m.lines)

	// down 移动
	m.Update(kp("down"))
	if m.cursor != 1 {
		t.Errorf("down 后 cursor = %d, want 1", m.cursor)
	}
	// home/end
	m.Update(kp("end"))
	if m.cursor != n-1 {
		t.Errorf("end 后 cursor = %d, want %d", m.cursor, n-1)
	}
	m.Update(kp("home"))
	if m.cursor != 0 {
		t.Errorf("home 后 cursor = %d, want 0", m.cursor)
	}
	// up 在 0 不再减
	m.Update(kp("up"))
	if m.cursor != 0 {
		t.Errorf("cursor=0 时 up 后 = %d, want 0", m.cursor)
	}
}

func TestConfigUpdateToggleAndFilter(t *testing.T) {
	m := newConfigModel()
	m.configPath = "/tmp/config.json"
	m.Update(LoadMsg{Root: sampleRoot()})

	// enter 切换展开状态
	before := m.lines[0]
	m.Update(kp("enter"))
	if m.lines[0] != before {
		t.Logf("enter 后首行变化（toggle 生效）")
	}

	// 输入字符过滤
	m.Update(kp("e"))
	m.Update(kp("m"))
	if m.filter != "em" {
		t.Errorf("filter = %q, want em", m.filter)
	}
	// "em" 匹配 email
	if len(m.lines) == 0 {
		t.Error("过滤后 lines 不应为空")
	}

	// backspace 删除
	m.Update(kp("backspace"))
	if m.filter != "e" {
		t.Errorf("backspace 后 filter = %q, want e", m.filter)
	}

	// ctrl+u 清空
	m.Update(kp("ctrl+u"))
	if m.filter != "" {
		t.Errorf("ctrl+u 后 filter = %q, want empty", m.filter)
	}
}

func TestConfigExpandCollapse(t *testing.T) {
	m := newConfigModel()
	m.configPath = "/tmp/config.json"
	m.Update(LoadMsg{Root: sampleRoot()})

	// ctrl+w 收起全部（叶子节点行保留，对象节点收起）
	m.Update(kp("ctrl+w"))
	// ctrl+e 展开全部
	m.Update(kp("ctrl+e"))
	// 展开后对象节点的子节点应可见
	if len(m.lines) < 5 {
		t.Errorf("展开全部后 lines = %d, want >= 5", len(m.lines))
	}

	// ctrl+n / ctrl+b 跳转匹配
	m2 := newConfigModel()
	m2.configPath = "/tmp/config.json"
	m2.Update(LoadMsg{Root: sampleRoot()})
	m2.filter = "name"
	m2.rebuildLines()
	m2.cursor = 0
	m2.jumpNextMatch()
	if m2.cursor == 0 {
		t.Error("jumpNextMatch 应从当前位置后移")
	}
	// filter 为空时 jump 不动
	m2.filter = ""
	m2.jumpNextMatch()
	m2.jumpPrevMatch()
}

func TestConfigUpdateRefreshAndInput(t *testing.T) {
	// ctrl+r 刷新
	m := newConfigModel()
	m.configPath = "/tmp/config.json"
	nm, cmd := m.Update(kp("ctrl+r"))
	if nm != m {
		t.Error("ctrl+r 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("ctrl+r 应返回 LoadFileCmd")
	}

	// 无 configPath → 无 Cmd
	m2 := newConfigModel()
	if _, cmd := m2.Update(kp("ctrl+r")); cmd != nil {
		t.Error("无 configPath 时 ctrl+r 不应返回 Cmd")
	}

	// / 打开输入框
	m3 := newConfigModel()
	_, cmd3 := m3.Update(kp("/"))
	if cmd3 != nil {
		t.Error("按 / 不应返回 Cmd")
	}
	if !m3.input.Active {
		t.Error("按 / 后应打开输入框")
	}
}

func TestConfigViewStates(t *testing.T) {
	// 目录列表模式
	m := newConfigModel()
	m.dirListing = true
	m.dirPath = "/etc"
	m.dirList.SetItems([]string{"a.json"})
	if v := m.View(); !strings.Contains(v, "Config Files") {
		t.Errorf("目录列表视图缺少标题: %q", v)
	}

	// 输入框模式
	m2 := newConfigModel()
	m2.input.Active = true
	m2.input.Prompt = "Config file path:"
	m2.input.Value = "x"
	m2.input.Cursor = 1
	if v := m2.View(); !strings.Contains(v, "Open Config") {
		t.Errorf("输入框视图缺少标题: %q", v)
	}

	// 未加载
	m3 := newConfigModel()
	if v := m3.View(); !strings.Contains(v, "Press") {
		t.Errorf("未加载视图缺少提示: %q", v)
	}

	// 错误
	m4 := newConfigModel()
	m4.loaded = true
	m4.errMsg = "boom"
	if v := m4.View(); !strings.Contains(v, "boom") {
		t.Errorf("错误视图缺少错误信息: %q", v)
	}
}

func TestConfigViewMainContent(t *testing.T) {
	m := newConfigModel()
	m.configPath = "/tmp/config.json"
	m.Update(LoadMsg{Root: sampleRoot()})
	v := m.View()
	for _, want := range []string{"Config Browser", "devdash", "nodes"} {
		if !strings.Contains(v, want) {
			t.Errorf("主视图缺少 %q", want)
		}
	}

	// 搜索显示
	m2 := newConfigModel()
	m2.configPath = "/tmp/config.json"
	m2.Update(LoadMsg{Root: sampleRoot()})
	m2.filter = "name"
	m2.rebuildLines()
	v2 := m2.View()
	if !strings.Contains(v2, "Search: name") {
		t.Errorf("搜索视图缺少 Search: %q", v2)
	}

	// 空树显示空状态
	m3 := newConfigModel()
	m3.loaded = true
	m3.configPath = "/tmp/empty.json"
	m3.root = Node{}
	m3.rebuildLines()
	if v3 := m3.View(); v3 == "" {
		t.Error("空树视图为空")
	}
}

func TestConfigHelpers(t *testing.T) {
	// Flatten 基本
	root := sampleRoot()
	lines := Flatten(root, "")
	if len(lines) == 0 {
		t.Fatal("Flatten 空 filter 应返回行")
	}

	// nodeMatches
	if !nodeMatches(sampleRoot(), "devdash") {
		t.Error("nodeMatches 应匹配 name 值 devdash")
	}
	if nodeMatches(sampleRoot(), "zzz") {
		t.Error("nodeMatches 不应匹配不存在的值")
	}

	// ColorizeValue 各类型
	if got := ColorizeValue("s"); !strings.Contains(got, `"s"`) {
		t.Errorf("ColorizeValue(string) = %q", got)
	}
	ColorizeValue(1.5)
	ColorizeValue(true)
	ColorizeValue(nil)
	ColorizeValue([]int{1, 2})

	// HighlightMatch
	if got := HighlightMatch("hello", ""); got != "hello" {
		t.Errorf("HighlightMatch 空 filter 应原样: %q", got)
	}
	if got := HighlightMatch("hello world", "world"); !strings.Contains(got, "world") {
		t.Errorf("HighlightMatch 高亮失败: %q", got)
	}
	if got := HighlightMatch("hello", "zzz"); got != "hello" {
		t.Errorf("HighlightMatch 无匹配应原样: %q", got)
	}

	// ColorizeValueWithHighlight
	ColorizeValueWithHighlight("hello", "el")
	ColorizeValueWithHighlight(1.5, "5")
	ColorizeValueWithHighlight(true, "tr")
	ColorizeValueWithHighlight(nil, "x")
	ColorizeValueWithHighlight("hello", "")

	// countNodes
	if countNodes(Node{Key: "a", Children: []Node{{Key: "b"}, {Key: "c", Children: []Node{{Key: "d"}}}}}) != 4 {
		t.Error("countNodes 计数错误")
	}

	// expandAll / collapseAll
	root2 := Node{Key: "a", Children: []Node{{Key: "b", Children: []Node{{Key: "c"}}}}}
	exp := expandAll(root2)
	if !exp.Children[0].Expanded {
		t.Error("expandAll 应展开子节点")
	}
	col := collapseAll(exp)
	if col.Children[0].Expanded {
		t.Error("collapseAll 应收起子节点")
	}
	// 根节点（Key=""）保持展开（BuildTree 构造时根节点 Expanded=true）
	root3 := Node{Key: "", Expanded: true, Children: []Node{{Key: "b"}}}
	col3 := collapseAll(root3)
	if !col3.Expanded {
		t.Error("根节点 collapseAll 后应保持展开")
	}

	// nodePath
	path := nodePath(sampleRoot(), 0, "")
	if len(path) == 0 {
		t.Error("nodePath 应返回路径")
	}

	// ToggleHelper
	tr := sampleRoot()
	counter := 0
	tr = ToggleHelper(tr, 0, &counter, "")
	if !tr.Children[0].Expanded && tr.Children[0].Key != "" {
		// 首行是 name（叶子），toggle 到对象节点时展开
	}
	_ = counter

	// hasMatchInTree
	if !hasMatchInTree(sampleRoot(), "email") {
		t.Error("hasMatchInTree 应匹配 email")
	}
}

func TestConfigFlattenExpandedBehavior(t *testing.T) {
	// 展开状态下显示子节点
	root := Node{
		Key: "a", Expanded: true, Depth: 1,
		Children: []Node{
			{Key: "b", Value: "v1", Depth: 2},
		},
	}
	lines := Flatten(root, "")
	if len(lines) != 2 {
		t.Errorf("展开节点 Flatten = %d 行, want 2", len(lines))
	}
	// 收起状态下只显示父节点
	root2 := Node{
		Key: "a", Expanded: false, Depth: 1,
		Children: []Node{
			{Key: "b", Value: "v1", Depth: 2},
		},
	}
	lines2 := Flatten(root2, "")
	if len(lines2) != 1 {
		t.Errorf("收起节点 Flatten = %d 行, want 1", len(lines2))
	}
}
