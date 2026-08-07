package component

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestList_SetItems(t *testing.T) {
	m := &ListModel{Cursor: 5}
	m.SetItems([]string{"a", "b", "c"})
	if len(m.Items) != 3 || m.Cursor != 0 {
		t.Errorf("SetItems 后 Items=%v Cursor=%d, want len 3 / Cursor 0", m.Items, m.Cursor)
	}
}

func TestList_Move(t *testing.T) {
	m := &ListModel{}
	m.SetItems([]string{"a", "b", "c"})

	// 空光标时 MoveUp 不越界
	m.MoveUp()
	if m.Cursor != 0 {
		t.Errorf("MoveUp 越界 Cursor = %d, want 0", m.Cursor)
	}

	m.MoveDown()
	m.MoveDown()
	if m.Cursor != 2 {
		t.Errorf("MoveDown×2 Cursor = %d, want 2", m.Cursor)
	}
	m.MoveDown() // 到底后不再移动
	if m.Cursor != 2 {
		t.Errorf("MoveDown 越界 Cursor = %d, want 2", m.Cursor)
	}

	m.MoveUp()
	if m.Cursor != 1 {
		t.Errorf("MoveUp Cursor = %d, want 1", m.Cursor)
	}
}

func TestList_Selected(t *testing.T) {
	m := &ListModel{}
	if got := m.Selected(); got != "" {
		t.Errorf("空列表 Selected() = %q, want 空", got)
	}
	m.SetItems([]string{"x", "y"})
	m.Cursor = 1
	if got := m.Selected(); got != "y" {
		t.Errorf("Selected() = %q, want y", got)
	}
	m.Cursor = 99
	if got := m.Selected(); got != "" {
		t.Errorf("越界 Selected() = %q, want 空", got)
	}
}

func TestList_Len(t *testing.T) {
	m := &ListModel{}
	if got := m.Len(); got != 0 {
		t.Errorf("空列表 Len() = %d, want 0", got)
	}
	m.SetItems([]string{"a", "b"})
	if got := m.Len(); got != 2 {
		t.Errorf("Len() = %d, want 2", got)
	}
}

func TestList_ScrollInfo(t *testing.T) {
	m := &ListModel{}
	if got := m.ScrollInfo(); got != "0/0" {
		t.Errorf("空列表 ScrollInfo() = %q, want 0/0", got)
	}
	m.SetItems([]string{"a", "b", "c"})
	m.Cursor = 2
	if got := m.ScrollInfo(); got != "3/3" {
		t.Errorf("ScrollInfo() = %q, want 3/3", got)
	}
}

func TestList_Render(t *testing.T) {
	m := &ListModel{}
	if got := m.Render(5, lipgloss.NewStyle(), "  "); got != "" {
		t.Errorf("空列表 Render() = %q, want 空", got)
	}

	m.SetItems([]string{"a", "b", "c"})
	// 全部可见时直接渲染全部，每行带 prefix
	got := m.Render(5, lipgloss.NewStyle(), "  ")
	if !strings.Contains(got, "  a") || !strings.Contains(got, "  b") || !strings.Contains(got, "  c") {
		t.Errorf("Render() 缺少项: %q", got)
	}
	// 换行连接
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Errorf("Render() 行数 = %d, want 3: %q", len(lines), got)
	}

	// 窗口滚动：cursor=8，maxShow=4 → 窗口 start=5, 显示 f/g/h/i
	var items []string
	for i := 0; i < 10; i++ {
		items = append(items, string(rune('a'+i)))
	}
	m.SetItems(items)
	m.Cursor = 8
	got = m.Render(4, lipgloss.NewStyle(), "")
	lines = strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Errorf("滚动窗口 Render() 行数 = %d, want 4: %q", len(lines), got)
	}
	// 窗口首行应为 f（index 5），末行应为 i（index 8，光标行）
	if lines[0] != "f" || lines[len(lines)-1] != "i" {
		t.Errorf("滚动窗口内容错误: %q", got)
	}

	// 高亮样式渲染：光标行用 style 包裹后仍包含原文
	hi := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	got = m.Render(4, hi, "")
	if !strings.Contains(got, "i") {
		t.Errorf("高亮 Render() 应含光标项: %q", got)
	}
}

func TestList_RenderWithFormat(t *testing.T) {
	m := &ListModel{}
	if got := m.RenderWithFormat(5, func(item string, selected bool) string { return item }); got != "" {
		t.Errorf("空列表 RenderWithFormat() = %q, want 空", got)
	}

	m.SetItems([]string{"a", "b"})
	m.Cursor = 1
	got := m.RenderWithFormat(5, func(item string, selected bool) string {
		if selected {
			return ">" + item
		}
		return " " + item
	})
	want := " a\n>b"
	if got != want {
		t.Errorf("RenderWithFormat() = %q, want %q", got, want)
	}
}
