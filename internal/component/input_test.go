package component

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// kp 构造一个 tea.KeyPressMsg，使 msg.String() 与项目代码匹配（如 "up"/"ctrl+f"/"/"）
// 注意：ctrl 组合键不能设置 Text，否则 String() 直接返回 Text 而忽略 Mod（丢失 "ctrl+" 前缀）
func kp(s string) tea.KeyPressMsg {
	var mod tea.KeyMod
	if strings.HasPrefix(s, "ctrl+") {
		mod = tea.ModCtrl
		s = strings.TrimPrefix(s, "ctrl+")
	}
	code := keyCode(s)
	if mod != 0 {
		// 组合键：只设置 Mod+Code，String() 走 Keystroke() → "ctrl+<code>"
		return tea.KeyPressMsg{Mod: mod, Code: code}
	}
	// 普通键：设置 Text 和 Code，String() 返回 Text
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
	case "delete":
		return tea.KeyDelete
	case "home":
		return tea.KeyHome
	case "end":
		return tea.KeyEnd
	case "tab":
		return tea.KeyTab
	case "space":
		return tea.KeySpace
	case "pgup":
		return tea.KeyPgUp
	case "pgdown":
		return tea.KeyPgDown
	}
	// 普通可打印字符（单 rune）
	return []rune(s)[0]
}

func TestConfirm(t *testing.T) {
	m := &InputModel{Active: true, Value: "  hello  "}
	got := m.Confirm()
	if got != "hello" {
		t.Errorf("Confirm() = %q, want %q", got, "hello")
	}
	if m.Active {
		t.Error("Confirm() 后 Active 应为 false")
	}
	if m.recentIdx != -1 {
		t.Errorf("Confirm() 后 recentIdx = %d, want -1", m.recentIdx)
	}
}

func TestCancel(t *testing.T) {
	m := &InputModel{Active: true, Value: "abc", Cursor: 3, recentIdx: 2}
	m.Cancel()
	if m.Active || m.Value != "" || m.Cursor != 0 || m.recentIdx != -1 {
		t.Errorf("Cancel() 未完全重置: %+v", m)
	}
}

func TestOpen(t *testing.T) {
	m := &InputModel{}
	m.Open("你好world")
	// "你好" 占 2 个字符，"world" 5 个，RuneLen 按字符数算共 7
	if !m.Active || m.Value != "你好world" {
		t.Errorf("Open() 状态错误: Active=%v Value=%q", m.Active, m.Value)
	}
	if m.Cursor != 7 {
		t.Errorf("Open() Cursor = %d, want 7", m.Cursor)
	}
}

func TestReset(t *testing.T) {
	m := &InputModel{Active: true, Value: "x", Cursor: 1, recentIdx: 0}
	m.Reset()
	if m.Active || m.Value != "" || m.Cursor != 0 || m.recentIdx != -1 {
		t.Errorf("Reset() 未完全重置: %+v", m)
	}
}

func TestRecentIdx(t *testing.T) {
	m := &InputModel{}
	if got := m.RecentIdx(); got != -1 {
		t.Errorf("空最近记录 RecentIdx() = %d, want -1", got)
	}
	m.SetRecent([]string{"a", "b"})
	// recentIdx 零值为 0（未导航过），且列表非空，RecentIdx() 直接返回该值
	if got := m.RecentIdx(); got != 0 {
		t.Errorf("未导航时 RecentIdx() = %d, want 0", got)
	}
	m.recentIdx = 1
	if got := m.RecentIdx(); got != 1 {
		t.Errorf("RecentIdx() = %d, want 1", got)
	}
}

func TestSetRecent(t *testing.T) {
	m := &InputModel{}
	m.SetRecent([]string{"x", "y", "z"})
	if len(m.RecentItems) != 3 || m.RecentItems[0] != "x" {
		t.Errorf("SetRecent 失败: %v", m.RecentItems)
	}
}

func TestUpdate_TypingAndEditing(t *testing.T) {
	m := &InputModel{Active: true}
	// 逐字输入
	for _, c := range []string{"a", "b", "c"} {
		if cmd := m.Update(kp(c), nil); cmd != nil {
			t.Errorf("输入 %q 时不应返回 cmd", c)
		}
	}
	if m.Value != "abc" || m.Cursor != 3 {
		t.Fatalf("输入后 Value=%q Cursor=%d, want abc/3", m.Value, m.Cursor)
	}

	// left / left → 光标到 1
	m.Update(kp("left"), nil)
	m.Update(kp("left"), nil)
	if m.Cursor != 1 {
		t.Errorf("left×2 后 Cursor = %d, want 1", m.Cursor)
	}

	// 在中间插入 "X" → "aXbc"
	m.Update(kp("X"), nil)
	if m.Value != "aXbc" || m.Cursor != 2 {
		t.Errorf("插入后 Value=%q Cursor=%d, want aXbc/2", m.Value, m.Cursor)
	}

	// backspace 删除光标前字符 → "abc"
	m.Update(kp("backspace"), nil)
	if m.Value != "abc" || m.Cursor != 1 {
		t.Errorf("backspace 后 Value=%q Cursor=%d, want abc/1", m.Value, m.Cursor)
	}

	// end → 光标到末尾；left 一步再 delete 删除末尾 → "ab"
	m.Update(kp("end"), nil)
	if m.Cursor != 3 {
		t.Errorf("end 后 Cursor = %d, want 3", m.Cursor)
	}
	m.Update(kp("left"), nil)
	m.Update(kp("delete"), nil)
	if m.Value != "ab" {
		t.Errorf("delete 后 Value = %q, want ab", m.Value)
	}

	// home → 光标 0
	m.Update(kp("home"), nil)
	if m.Cursor != 0 {
		t.Errorf("home 后 Cursor = %d, want 0", m.Cursor)
	}
}

func TestUpdate_CursorBounds(t *testing.T) {
	m := &InputModel{Active: true, Value: "ab", Cursor: 0}
	// 光标在 0 时 left/backspace 不应越界
	m.Update(kp("left"), nil)
	if m.Cursor != 0 {
		t.Errorf("left 越界 Cursor = %d, want 0", m.Cursor)
	}
	m.Update(kp("backspace"), nil)
	if m.Value != "ab" {
		t.Errorf("backspace 越界 Value = %q, want ab", m.Value)
	}
	// 光标在末尾时 right/delete 不应越界
	m.Cursor = 2
	m.Update(kp("right"), nil)
	if m.Cursor != 2 {
		t.Errorf("right 越界 Cursor = %d, want 2", m.Cursor)
	}
	m.Update(kp("delete"), nil)
	if m.Value != "ab" {
		t.Errorf("delete 越界 Value = %q, want ab", m.Value)
	}
}

func TestUpdate_Enter(t *testing.T) {
	m := &InputModel{Active: true, Value: "  123  "}
	submitted := ""
	var cmd tea.Cmd
	cmd = m.Update(kp("enter"), func(v string) func() tea.Msg {
		submitted = v
		return func() tea.Msg { return nil }
	})
	if submitted != "123" {
		t.Errorf("enter 提交值 = %q, want 123", submitted)
	}
	if m.Active {
		t.Error("enter 后 Active 应为 false")
	}
	if cmd == nil {
		t.Error("enter 应返回 onSubmit 产生的 cmd")
	} else {
		cmd() // 不应 panic
	}
}

func TestUpdate_Esc(t *testing.T) {
	m := &InputModel{Active: true, Value: "abc", Cursor: 3}
	if cmd := m.Update(kp("esc"), nil); cmd != nil {
		t.Errorf("esc 不应返回 cmd")
	}
	if m.Active || m.Value != "" || m.Cursor != 0 {
		t.Errorf("esc 未取消: %+v", m)
	}
}

func TestUpdate_CtrlU(t *testing.T) {
	m := &InputModel{Active: true, Value: "abc", Cursor: 3, recentIdx: 1}
	m.Update(kp("ctrl+u"), nil)
	if m.Value != "" || m.Cursor != 0 || m.recentIdx != -1 {
		t.Errorf("ctrl+u 未清空: %+v", m)
	}
}

func TestUpdate_RecentNavigation(t *testing.T) {
	m := &InputModel{Active: true}
	m.SetRecent([]string{"cat", "dog", "fox"})
	// 模拟真实使用：Open() 会把 recentIdx 置为 -1
	m.Open("")

	// down → 选中第 1 条
	m.Update(kp("down"), nil)
	if m.Value != "cat" || m.Cursor != 3 || m.recentIdx != 0 {
		t.Errorf("down 后 Value=%q recentIdx=%d, want cat/0", m.Value, m.recentIdx)
	}

	// down ×2 → 到末尾，再 down 不越界
	m.Update(kp("down"), nil)
	m.Update(kp("down"), nil)
	if m.recentIdx != 2 || m.Value != "fox" {
		t.Errorf("down 到底后 recentIdx=%d Value=%q, want 2/fox", m.recentIdx, m.Value)
	}
	m.Update(kp("down"), nil)
	if m.recentIdx != 2 {
		t.Errorf("down 越界 recentIdx = %d, want 2", m.recentIdx)
	}

	// up → 回到第 2 条
	m.Update(kp("up"), nil)
	if m.recentIdx != 1 || m.Value != "dog" {
		t.Errorf("up 后 recentIdx=%d Value=%q, want 1/dog", m.recentIdx, m.Value)
	}

	// up ×2 → 回到底部空值（recentIdx=-1）
	m.Update(kp("up"), nil)
	m.Update(kp("up"), nil)
	if m.recentIdx != -1 || m.Value != "" {
		t.Errorf("up 回顶后 recentIdx=%d Value=%q, want -1/空", m.recentIdx, m.Value)
	}

	// 无最近记录时 up/down 不应 panic
	m2 := &InputModel{Active: true}
	m2.Update(kp("up"), nil)
	m2.Update(kp("down"), nil)
}

func TestUpdate_Paste(t *testing.T) {
	m := &InputModel{Active: true, Value: "ab", Cursor: 1}
	m.Update(tea.PasteMsg{Content: "XY"}, nil)
	if m.Value != "aXYb" || m.Cursor != 3 {
		t.Errorf("Paste 后 Value=%q Cursor=%d, want aXYb/3", m.Value, m.Cursor)
	}
	if m.recentIdx != -1 {
		t.Errorf("Paste 后 recentIdx = %d, want -1", m.recentIdx)
	}
}

func TestUpdate_NonPrintableIgnored(t *testing.T) {
	m := &InputModel{Active: true, Value: "ab", Cursor: 2}
	// 特殊键但不匹配任何 case（如 F5），不应修改内容
	m.Update(tea.KeyPressMsg{Code: tea.KeyF5}, nil)
	if m.Value != "ab" {
		t.Errorf("F5 不应修改 Value = %q", m.Value)
	}
}
