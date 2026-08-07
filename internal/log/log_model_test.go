package log

import (
	"os"
	"path/filepath"
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
	case "left":
		return tea.KeyLeft
	case "right":
		return tea.KeyRight
	case "enter":
		return tea.KeyEnter
	case "esc":
		return tea.KeyEsc
	case "backspace":
		return tea.KeyBackspace
	case "home":
		return tea.KeyHome
	case "end":
		return tea.KeyEnd
	case "space":
		return tea.KeySpace
	}
	return []rune(s)[0]
}

func newLogModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

// sampleTestLines 构造测试日志行
func sampleTestLines() []Line {
	return []Line{
		{Raw: "INFO server started", Level: "INFO"},
		{Raw: "WARN disk low", Level: "WARN"},
		{Raw: "ERROR connection failed", Level: "ERROR"},
		{Raw: "DEBUG cache hit", Level: "DEBUG"},
	}
}

func TestLogInit(t *testing.T) {
	// lastLogPath 非空 → LoadFromFileCmd
	m := newLogModel()
	cmd := m.Init("/tmp/test.log")
	if cmd == nil {
		t.Fatal("Init(非空路径) 返回 nil")
	}
	if m.logPath != "/tmp/test.log" {
		t.Errorf("logPath = %q, want /tmp/test.log", m.logPath)
	}

	// lastLogPath 为空且无 --log 参数 → nil（测试环境 stdin 非 tty 但无参数）
	m2 := newLogModel()
	if cmd := m2.Init(""); cmd != nil {
		t.Logf("Init(\"\") 返回 cmd（stdin 非 tty 时走 LoadFromStdin）: %v", cmd != nil)
	}
}

func TestLogUpdateLoadMsg(t *testing.T) {
	m := newLogModel()
	m.logPath = "/tmp/test.log"
	lines := sampleTestLines()

	nm, _ := m.Update(LoadMsg{Lines: lines})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if !m.loaded {
		t.Error("LoadMsg 后 loaded 应为 true")
	}
	if len(m.all) != 4 {
		t.Errorf("all = %d 行, want 4", len(m.all))
	}
	if len(m.filtered) != 4 {
		t.Errorf("filtered = %d 行, want 4", len(m.filtered))
	}
	if m.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", m.errMsg)
	}
	if m.cachedPath != "/tmp/test.log" {
		t.Errorf("cachedPath = %q, want /tmp/test.log", m.cachedPath)
	}

	// 错误消息
	m2 := newLogModel()
	m2.logPath = "/tmp/not_exist.log"
	m2.input.Active = true
	nm2, _ := m2.Update(LoadMsg{Err: os.ErrNotExist})
	if nm2 != m2 {
		t.Error("Update 应返回同一 Model 指针")
	}
	if m2.errMsg == "" {
		t.Error("LoadMsg 带错误时应设置 errMsg")
	}
	if !m2.loaded {
		t.Error("LoadMsg 带错误时 loaded 应为 true")
	}

	// Lines 为 nil → 使用示例数据
	m3 := newLogModel()
	m3.logPath = "/tmp/sample.log"
	m3.Update(LoadMsg{Lines: nil})
	if len(m3.all) == 0 {
		t.Error("Lines=nil 时应使用示例数据")
	}
}

func TestLogUpdateDirMsg(t *testing.T) {
	// 有文件 → 目录列表模式
	m := newLogModel()
	nm, cmd := m.Update(DirMsg{Dir: "/logs", Files: []string{"a.log", "b.log"}})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("DirMsg 有文件时不应返回 Cmd")
	}
	if !m.dirListing {
		t.Error("应进入目录列表模式")
	}

	// 无文件 → 错误 + 打开输入框
	m2 := newLogModel()
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
	if m2.dirListing {
		t.Error("无文件时应退出目录列表模式")
	}
	if !m2.input.Active {
		t.Error("无文件时应打开输入框")
	}
}

func TestLogUpdateTailDataMsg(t *testing.T) {
	// 正常新数据 → 追加 + 返回 receiveTailCmd
	m := newLogModel()
	m.logPath = "/tmp/tail.log"
	m.Update(LoadMsg{Lines: sampleTestLines()})
	m.tailFMode = true
	ch := make(chan []byte, 1)
	m.tailCh = ch
	nm, cmd := m.Update(TailDataMsg{Lines: []Line{{Raw: "INFO new line", Level: "INFO"}}})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("TailDataMsg 有数据应返回 receiveTailCmd")
	}
	if len(m.all) != 5 {
		t.Errorf("追加后 all = %d 行, want 5", len(m.all))
	}

	// Done=true → 停止监听
	m2 := newLogModel()
	nm2, cmd2 := m2.Update(TailDataMsg{Done: true})
	if nm2 != m2 {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd2 != nil {
		t.Error("Done=true 不应返回 Cmd")
	}
	if m2.tailFMode {
		t.Error("Done=true 后 tailFMode 应为 false")
	}

	// Err != nil → 停止监听
	m3 := newLogModel()
	m3.tailFMode = true
	m3.Update(TailDataMsg{Err: os.ErrClosed})
	if m3.tailFMode {
		t.Error("Err 非 nil 后 tailFMode 应为 false")
	}
}

func TestLogUpdatePaste(t *testing.T) {
	// 输入框活跃 → 转发给 input
	m := newLogModel()
	m.input.Active = true
	nm, cmd := m.Update(tea.PasteMsg{Content: "paste"})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("输入框活跃时 Paste 不应返回 Cmd")
	}

	// 输入框不活跃 → 追加 filter
	m2 := newLogModel()
	m2.Update(LoadMsg{Lines: sampleTestLines()})
	nm2, cmd2 := m2.Update(tea.PasteMsg{Content: "server"})
	if nm2 != m2 {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd2 != nil {
		t.Error("Paste 到 filter 不应返回 Cmd")
	}
	if m2.filter != "server" {
		t.Errorf("filter = %q, want server", m2.filter)
	}
	if len(m2.filtered) != 1 {
		t.Errorf("filtered = %d 行, want 1（server 匹配 INFO 行）", len(m2.filtered))
	}
}

func TestLogUpdateDirListingKeys(t *testing.T) {
	m := newLogModel()
	m.dirListing = true
	m.dirPath = "/logs"
	m.dirList.SetItems([]string{"a.log", "b.log", "c.log"})

	m.Update(kp("down"))
	m.Update(kp("down"))
	if got := m.dirList.Selected(); got != "c.log" {
		t.Errorf("down 两次后选中 = %q, want c.log", got)
	}
	m.Update(kp("up"))
	if got := m.dirList.Selected(); got != "b.log" {
		t.Errorf("up 后选中 = %q, want b.log", got)
	}

	// enter → 加载选中文件
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
	if m.logPath != filepath.Join("/logs", "b.log") {
		t.Errorf("logPath = %q, want %q", m.logPath, filepath.Join("/logs", "b.log"))
	}

	// esc → 退出目录列表打开输入框
	m2 := newLogModel()
	m2.dirListing = true
	m2.dirPath = "/logs"
	m2.dirList.SetItems([]string{"a.log"})
	m2.Update(kp("esc"))
	if m2.dirListing {
		t.Error("esc 后应退出目录列表模式")
	}
	if !m2.input.Active {
		t.Error("esc 后应打开输入框")
	}
}

func TestLogUpdatePageInput(t *testing.T) {
	m := newLogModel()
	m.Update(LoadMsg{Lines: sampleTestLines()})
	m.pageInput = true

	// 数字输入
	m.Update(kp("1"))
	m.Update(kp("2"))
	if m.pageInputValue != "12" {
		t.Errorf("pageInputValue = %q, want 12", m.pageInputValue)
	}
	// 非数字忽略
	m.Update(kp("a"))
	if m.pageInputValue != "12" {
		t.Errorf("非数字不应追加: %q", m.pageInputValue)
	}
	// backspace 删除
	m.Update(kp("backspace"))
	if m.pageInputValue != "1" {
		t.Errorf("backspace 后 pageInputValue = %q, want 1", m.pageInputValue)
	}
	// esc 取消
	m.Update(kp("esc"))
	if m.pageInput {
		t.Error("esc 后 pageInput 应为 false")
	}
	if m.pageInputValue != "" {
		t.Errorf("esc 后 pageInputValue = %q, want empty", m.pageInputValue)
	}

	// enter 确认页码
	m2 := newLogModel()
	m2.Update(LoadMsg{Lines: sampleTestLines()})
	m2.pageInput = true
	m2.pageInputValue = "1"
	m2.page = 5
	m2.Update(kp("enter"))
	if m2.pageInput {
		t.Error("enter 后 pageInput 应为 false")
	}
	if m2.page != 0 {
		t.Errorf("enter 后 page = %d, want 0", m2.page)
	}

	// 无效页码 → clamp 到合法范围
	m3 := newLogModel()
	m3.Update(LoadMsg{Lines: sampleTestLines()})
	m3.pageInput = true
	m3.pageInputValue = "99"
	m3.Update(kp("enter"))
	if m3.page < 0 || m3.page >= m3.totalPages() {
		t.Errorf("无效页码应被 clamp: page = %d, totalPages = %d", m3.page, m3.totalPages())
	}
}

func TestLogUpdateLevelOverlay(t *testing.T) {
	m := newLogModel()
	m.Update(LoadMsg{Lines: sampleTestLines()})
	m.levelOverlay = true
	m.levelIdx = 0
	m.levelSel = map[int]bool{} // 真实流程由 ctrl+l 初始化

	// down 移动光标
	m.Update(kp("down"))
	if m.levelIdx != 1 {
		t.Errorf("down 后 levelIdx = %d, want 1", m.levelIdx)
	}
	// enter 切换 INFO 选中
	m.Update(kp("enter"))
	if !m.levelSel[1] {
		t.Error("enter 后 INFO 应被选中")
	}
	// 再按一次取消
	m.Update(kp("enter"))
	if m.levelSel[1] {
		t.Error("再按 enter 后 INFO 应取消选中")
	}
	// 选中 ERROR(idx=3)
	m.Update(kp("down"))
	m.Update(kp("down"))
	m.Update(kp("enter"))
	if !m.levelSel[3] {
		t.Error("ERROR 应被选中")
	}
	// level filter 应用后 filtered 只含 ERROR
	for _, l := range m.filtered {
		if l.Level != "ERROR" {
			t.Errorf("level 过滤后混入 %q", l.Level)
		}
	}
	// 回到 All 清除所有选中
	m.Update(kp("up"))
	m.Update(kp("up"))
	m.Update(kp("up"))
	m.Update(kp("enter"))
	if len(m.levelSel) != 0 {
		t.Errorf("All 应清除所有选中: %v", m.levelSel)
	}
	if len(m.filtered) != 4 {
		t.Errorf("清除 level 后 filtered = %d, want 4", len(m.filtered))
	}
	// esc 关闭 overlay
	m.Update(kp("esc"))
	if m.levelOverlay {
		t.Error("esc 后 levelOverlay 应为 false")
	}
}

func TestLogUpdateNavigation(t *testing.T) {
	m := newLogModel()
	// 构造 >20 行以便翻页
	var lines []Line
	for i := 0; i < 25; i++ {
		lines = append(lines, Line{Raw: "INFO line", Level: "INFO"})
	}
	m.Update(LoadMsg{Lines: lines})
	if m.page != 2 {
		t.Errorf("加载后 page = %d, want 2（最后一页）", m.page)
	}

	// up/down 页内光标
	m.Update(kp("up"))
	if m.cursor != 0 {
		t.Errorf("cursor=0 时 up 后 = %d, want 0", m.cursor)
	}
	m.Update(kp("down"))
	if m.cursor != 1 {
		t.Errorf("down 后 cursor = %d, want 1", m.cursor)
	}

	// left/right 翻页
	m.Update(kp("left"))
	if m.page != 1 {
		t.Errorf("left 后 page = %d, want 1", m.page)
	}
	m.Update(kp("right"))
	if m.page != 2 {
		t.Errorf("right 后 page = %d, want 2", m.page)
	}

	// home/end
	m.Update(kp("home"))
	if m.page != 0 {
		t.Errorf("home 后 page = %d, want 0", m.page)
	}
	m.Update(kp("end"))
	if m.page != 2 {
		t.Errorf("end 后 page = %d, want 2", m.page)
	}

	// ctrl+up / ctrl+down 快速翻页
	m.Update(kp("ctrl+up"))
	if m.page != 0 {
		t.Errorf("ctrl+up 后 page = %d, want 0", m.page)
	}
	m.Update(kp("ctrl+down"))
	if m.page != 2 {
		t.Errorf("ctrl+down 后 page = %d, want 2", m.page)
	}

	// ctrl+p 打开页码输入
	m.Update(kp("ctrl+p"))
	if !m.pageInput {
		t.Error("ctrl+p 后 pageInput 应为 true")
	}
}

func TestLogUpdateFilter(t *testing.T) {
	m := newLogModel()
	m.Update(LoadMsg{Lines: sampleTestLines()})

	// 输入字符过滤
	m.Update(kp("c"))
	m.Update(kp("o"))
	m.Update(kp("n"))
	m.Update(kp("n"))
	if m.filter != "conn" {
		t.Errorf("filter = %q, want conn", m.filter)
	}
	// "conn" 正则只匹配 ERROR 行的 connection
	if len(m.filtered) != 1 {
		t.Errorf("filtered = %d 行, want 1", len(m.filtered))
	}
	if m.filtered[0].Level != "ERROR" {
		t.Errorf("匹配行应为 ERROR, got %s", m.filtered[0].Level)
	}

	// backspace 删除 filter
	m.Update(kp("backspace"))
	if m.filter != "con" {
		t.Errorf("backspace 后 filter = %q, want con", m.filter)
	}

	// 无效正则 → errMsg
	m2 := newLogModel()
	m2.Update(LoadMsg{Lines: sampleTestLines()})
	m2.Update(kp("["))
	m2.Update(kp("("))
	if m2.errMsg == "" {
		t.Error("无效正则应设置 errMsg")
	}

	// ctrl+u 清空 filter
	m3 := newLogModel()
	m3.Update(LoadMsg{Lines: sampleTestLines()})
	m3.filter = "abc"
	m3.Update(kp("ctrl+u"))
	if m3.filter != "" {
		t.Errorf("ctrl+u 后 filter = %q, want empty", m3.filter)
	}
	if len(m3.filtered) != 4 {
		t.Errorf("ctrl+u 后 filtered = %d, want 4", len(m3.filtered))
	}
}

func TestLogUpdateRefresh(t *testing.T) {
	// 有 logPath → 返回 LoadFromFileCmd
	m := newLogModel()
	m.logPath = "/tmp/x.log"
	nm, cmd := m.Update(kp("ctrl+r"))
	if nm != m {
		t.Error("ctrl+r 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("ctrl+r 应返回 LoadFromFileCmd")
	}

	// 无 logPath → 无 Cmd
	m2 := newLogModel()
	if _, cmd := m2.Update(kp("ctrl+r")); cmd != nil {
		t.Error("无 logPath 时 ctrl+r 不应返回 Cmd")
	}
}

func TestLogUpdateTailF(t *testing.T) {
	// 构造一个真实临时文件测试 ctrl+f
	dir := t.TempDir()
	path := filepath.Join(dir, "tail.log")
	os.WriteFile(path, []byte("existing\n"), 0644)

	m := newLogModel()
	m.logPath = path
	m.Update(LoadMsg{Lines: sampleTestLines()})

	// 打开 tail 模式 → 返回 receiveTailCmd
	nm, cmd := m.Update(kp("ctrl+f"))
	if nm != m {
		t.Error("ctrl+f 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("开启 tail 模式应返回 Cmd")
	}
	if !m.tailFMode {
		t.Error("ctrl+f 后 tailFMode 应为 true")
	}
	if m.tailCh == nil {
		t.Error("tailCh 不应为 nil")
	}

	// 关闭 tail 模式 → close done
	nm2, cmd2 := m.Update(kp("ctrl+f"))
	if nm2 != m {
		t.Error("再次 ctrl+f 应返回同一 Model 指针")
	}
	if cmd2 != nil {
		t.Error("关闭 tail 模式不应返回 Cmd")
	}
	if m.tailFMode {
		t.Error("再次 ctrl+f 后 tailFMode 应为 false")
	}
}

func TestLogViewStates(t *testing.T) {
	// 目录列表模式
	m := newLogModel()
	m.dirListing = true
	m.dirPath = "/logs"
	m.dirList.SetItems([]string{"a.log"})
	if v := m.View(); !strings.Contains(v, "Log Files") {
		t.Errorf("目录列表视图缺少标题: %q", v)
	}

	// 输入框模式
	m2 := newLogModel()
	m2.input.Active = true
	m2.input.Prompt = "Log file path:"
	m2.input.Value = "x"
	m2.input.Cursor = 1
	if v := m2.View(); !strings.Contains(v, "Open Log") {
		t.Errorf("输入框视图缺少标题: %q", v)
	}

	// 页码输入模式
	m3 := newLogModel()
	m3.pageInput = true
	m3.pageInputValue = "3"
	if v := m3.View(); !strings.Contains(v, "Go to page") {
		t.Errorf("页码输入视图缺少内容: %q", v)
	}

	// level filter 模式
	m4 := newLogModel()
	m4.levelOverlay = true
	if v := m4.View(); !strings.Contains(v, "Level Filter") {
		t.Errorf("level filter 视图缺少标题: %q", v)
	}

	// 未加载
	m5 := newLogModel()
	if v := m5.View(); !strings.Contains(v, "Press") {
		t.Errorf("未加载视图缺少提示: %q", v)
	}

	// 加载失败
	m6 := newLogModel()
	m6.loaded = true
	m6.errMsg = "boom"
	if v := m6.View(); !strings.Contains(v, "boom") {
		t.Errorf("错误视图缺少错误信息: %q", v)
	}
}

func TestLogViewMainContent(t *testing.T) {
	m := newLogModel()
	m.logPath = "/tmp/x.log"
	m.Update(LoadMsg{Lines: sampleTestLines()})
	v := m.View()
	for _, want := range []string{"Log", "INFO", "WARN", "ERROR", "DEBUG", "Page 1/1", "entries"} {
		if !strings.Contains(v, want) {
			t.Errorf("主视图缺少 %q", want)
		}
	}

	// 空 filtered 显示空状态
	m2 := newLogModel()
	m2.Update(LoadMsg{Lines: sampleTestLines()})
	m2.filter = "zzzz_no_match"
	m2.applyFilter()
	if v2 := m2.View(); !strings.Contains(v2, "没有找到匹配") {
		t.Errorf("空结果视图缺少提示: %q", v2)
	}

	// 长行 wrap 不 panic
	m3 := newLogModel()
	long := strings.Repeat("x", 200)
	m3.Update(LoadMsg{Lines: []Line{{Raw: long, Level: "INFO"}}})
	if v3 := m3.View(); !strings.Contains(v3, "xx") {
		t.Errorf("长行视图缺少内容: %q", v3)
	}

	// 小高度不 panic
	m4 := newLogModel()
	m4.UpdateSize(100, 8)
	m4.Update(LoadMsg{Lines: sampleTestLines()})
	if v4 := m4.View(); v4 == "" {
		t.Error("小高度视图为空")
	}
}

func TestLogHelpers(t *testing.T) {
	// totalPages / clampPage
	m := newLogModel()
	if m.totalPages() != 1 {
		t.Errorf("空数据 totalPages = %d, want 1", m.totalPages())
	}
	m.Update(LoadMsg{Lines: sampleTestLines()})
	if m.totalPages() != 1 {
		t.Errorf("4 行 totalPages = %d, want 1", m.totalPages())
	}

	// clampPage
	m2 := newLogModel()
	m2.Update(LoadMsg{Lines: sampleTestLines()})
	m2.page = 99
	m2.clampPage()
	if m2.page != 0 {
		t.Errorf("clampPage 后 page = %d, want 0", m2.page)
	}
	m2.page = -5
	m2.clampPage()
	if m2.page != 0 {
		t.Errorf("负 page clamp 后 = %d, want 0", m2.page)
	}

	// selectedLevels
	m3 := newLogModel()
	if len(m3.selectedLevels()) != 0 {
		t.Error("未选中时 selectedLevels 应为空")
	}
	m3.levelSel = map[int]bool{1: true, 3: true}
	levels := m3.selectedLevels()
	if len(levels) != 2 || levels[0] != "INFO" || levels[1] != "ERROR" {
		t.Errorf("selectedLevels = %v, want [INFO ERROR]", levels)
	}
}

func TestLogWrapLineAndColorize(t *testing.T) {
	// wrapLine 短行不切
	if got := wrapLine("short", 50); len(got) != 1 || got[0] != "short" {
		t.Errorf("wrapLine 短行 = %v", got)
	}
	// wrapLine 长行按宽度切（ASCII）
	got := wrapLine("abcdefgh", 4)
	if len(got) != 2 {
		t.Errorf("wrapLine 长行 = %v, want 2 段", got)
	}
	// wrapLine 中文按列宽（每字 2 列）
	gotCn := wrapLine("你好世界", 4)
	if len(gotCn) != 2 {
		t.Errorf("wrapLine 中文 = %v, want 2 段", gotCn)
	}

	// colorizeLog 各级别
	for _, lvl := range []string{"ERROR", "WARN", "INFO", "DEBUG"} {
		if got := colorizeLog(Line{Level: lvl, Raw: "msg"}, ""); !strings.Contains(got, "msg") {
			t.Errorf("colorizeLog(%s) 缺少内容", lvl)
		}
	}
	// 未知级别 → 原样返回
	if got := colorizeLog(Line{Level: "OTHER", Raw: "raw"}, ""); got != "  raw" {
		t.Errorf("colorizeLog(OTHER) = %q, want \"  raw\"", got)
	}
	// 高亮匹配
	hl := colorizeLog(Line{Level: "INFO", Raw: "hello world"}, "hello")
	if !strings.Contains(hl, "hello") {
		t.Errorf("高亮匹配缺少内容: %q", hl)
	}
}

func TestLogStats(t *testing.T) {
	m := newLogModel()
	m.all = sampleTestLines()
	s := m.stats()
	for _, want := range []string{"ERROR:1", "WARN:1", "INFO:1", "DEBUG:1", "OTHER:0"} {
		if !strings.Contains(s, want) {
			t.Errorf("stats 缺少 %q: %q", want, s)
		}
	}
}

func TestLogSelectedLevelFilterView(t *testing.T) {
	// View 中 level 徽标
	m := newLogModel()
	m.logPath = "/tmp/x.log"
	m.Update(LoadMsg{Lines: sampleTestLines()})
	m.levelSel = map[int]bool{1: true} // INFO
	m.applyFilter()
	v := m.View()
	if !strings.Contains(v, "Level:") {
		t.Errorf("level 徽标视图缺少 Level: %q", v)
	}
}
