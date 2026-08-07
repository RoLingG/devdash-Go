package ui

import (
	"regexp"
	"strings"
	"testing"
)

// stripAnsi 去掉 ANSI 转义序列，便于对渲染结果做纯文本断言
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

// TestBarChart 测试柱状图渲染
func TestBarChart(t *testing.T) {
	t.Run("正常值", func(t *testing.T) {
		result := BarChart("commits", 50, 100, 20, ColAccent)
		// 应该包含标签和数值
		if !strings.Contains(result, "commits") {
			t.Error("result should contain label 'commits'")
		}
		if !strings.Contains(result, "50") {
			t.Error("result should contain value '50'")
		}
		// 应该包含柱状字符
		if !strings.Contains(result, "█") {
			t.Error("result should contain bar character '█'")
		}
	})

	t.Run("最大值为0时不panic", func(t *testing.T) {
		result := BarChart("test", 0, 0, 20, ColAccent)
		// maxValue=0 时 barLen 应为 1（最小值）
		if !strings.Contains(result, "█") {
			t.Error("result should contain at least one bar character")
		}
		if !strings.Contains(result, "0") {
			t.Error("result should contain value '0'")
		}
	})

	t.Run("值为0时最小柱宽", func(t *testing.T) {
		result := BarChart("empty", 0, 100, 20, ColAccent)
		// value=0 时 barLen=0，但应 clamp 到 1
		if !strings.Contains(result, "█") {
			t.Error("result should contain at least one bar character")
		}
	})

	t.Run("值等于最大值时满宽", func(t *testing.T) {
		result := BarChart("full", 100, 100, 20, ColAccent)
		// 应该包含 20 个 █
		barPart := result[strings.Index(result, "█"):]
		barPart = barPart[:strings.Index(barPart, " ")]
		count := strings.Count(barPart, "█")
		if count != 20 {
			t.Errorf("expected 20 bar chars, got %d", count)
		}
	})

	t.Run("超长标签截断到20字符", func(t *testing.T) {
		longLabel := "this_is_a_very_long_label_name_here"
		result := BarChart(longLabel, 10, 100, 20, ColAccent)
		// 标签部分应该被截断到 20 宽，Truncate 会加 "..."
		if !strings.Contains(result, "...") {
			t.Error("long label should be truncated with '...'")
		}
		// 截断后应包含标签前缀
		if !strings.Contains(result, "this_is_a") {
			t.Error("result should contain beginning of label")
		}
	})

	t.Run("短标签填充到20宽度", func(t *testing.T) {
		result := BarChart("git", 5, 10, 20, ColAccent)
		if !strings.Contains(result, "git") {
			t.Error("result should contain label 'git'")
		}
	})

	t.Run("柱状图比例正确", func(t *testing.T) {
		// 50/100 * 20 = 10
		result := BarChart("half", 50, 100, 20, ColAccent)
		barStart := strings.Index(result, "█")
		if barStart == -1 {
			t.Fatal("no bar character found")
		}
		barPart := result[barStart:]
		// 找到柱状图结束位置（遇到非█字符）
		barEnd := strings.IndexFunc(barPart, func(r rune) bool { return r != '█' })
		if barEnd == -1 {
			barEnd = len(barPart)
		}
		barLen := len([]rune(barPart[:barEnd]))
		if barLen != 10 {
			t.Errorf("expected bar length 10, got %d", barLen)
		}
	})
}

// TestBox 测试 Box 边框渲染
func TestBox(t *testing.T) {
	got := Box("Title", "line1\nline2", 40)
	for _, want := range []string{"Title", "line1", "line2"} {
		if !strings.Contains(got, want) {
			t.Errorf("Box 缺少 %q: %q", want, got)
		}
	}
	// 超长行应被截断而不 panic
	long := strings.Repeat("x", 200)
	_ = Box("T", long+"\nshort", 20)
	// 空内容不 panic
	_ = Box("", "", 0)
}

// TestCard 测试 Card 卡片渲染
func TestCard(t *testing.T) {
	got := Card("标题", "内容\n第二行", ColAccent, 40)
	for _, want := range []string{"标题", "内容", "第二行"} {
		if !strings.Contains(got, want) {
			t.Errorf("Card 缺少 %q: %q", want, got)
		}
	}
	// 含圆角边框字符
	if !strings.Contains(got, "╭") {
		t.Errorf("Card 缺少边框: %q", got)
	}
	// 空内容不 panic
	_ = Card("", "", ColSecondary, 10)
}

// TestRenderInputCard 测试输入框卡片渲染
func TestRenderInputCard(t *testing.T) {
	// 带最近记录和选中项
	got := RenderInputCard(InputCardOpts{
		Title: "输入", Prompt: "请输入路径", Value: "abc", Cursor: 2,
		CardWidth: 50, RecentItems: []string{"path1", "path2"}, RecentIdx: 1,
	})
	// 光标在 2，值渲染为 "ab|c"
	for _, want := range []string{"输入", "请输入路径", "ab|c", "path1", "path2"} {
		if !strings.Contains(stripAnsi(got), want) {
			t.Errorf("RenderInputCard 缺少 %q: %q", want, got)
		}
	}
	// 无最近记录不 panic
	_ = RenderInputCard(InputCardOpts{Title: "t", Prompt: "p", Value: "", Cursor: 0, CardWidth: 30})
	// 错误信息
	gotErr := RenderInputCard(InputCardOpts{Title: "t", Prompt: "p", Value: "v", Cursor: 1, CardWidth: 30, ErrMsg: "文件不存在"})
	if !strings.Contains(stripAnsi(gotErr), "文件不存在") {
		t.Errorf("RenderInputCard 缺少错误信息: %q", gotErr)
	}
	// 长路径截断
	gotLong := RenderInputCard(InputCardOpts{
		Title: "t", Prompt: "p", Value: "v", Cursor: 1, CardWidth: 30,
		RecentItems: []string{strings.Repeat("x", 60)}, RecentIdx: 0,
	})
	if strings.Contains(stripAnsi(gotLong), strings.Repeat("x", 60)) {
		t.Errorf("RenderInputCard 未截断长路径")
	}
}

// fakeDirList 模拟目录列表组件，实现 RenderWithFormat 接口
type fakeDirList struct{ items []string }

func (f fakeDirList) RenderWithFormat(maxShow int, formatter func(item string, selected bool) string) string {
	var sb strings.Builder
	for i, it := range f.items {
		sb.WriteString(formatter(it, i == 0))
		sb.WriteString("\n")
	}
	return sb.String()
}

// TestRenderDirListCard 测试目录列表卡片渲染
func TestRenderDirListCard(t *testing.T) {
	got := RenderDirListCard(DirListCardOpts{
		Title: "选择目录", DirPath: "/usr/local", CardWidth: 60, Height: 20,
		DirList: fakeDirList{items: []string{"a", "b"}},
	})
	for _, want := range []string{"选择目录", "/usr/local", "a", "b"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderDirListCard 缺少 %q: %q", want, got)
		}
	}
	// 小高度 maxShow clamp 到 5，不 panic
	_ = RenderDirListCard(DirListCardOpts{Title: "t", DirPath: "d", CardWidth: 40, Height: 3, DirList: fakeDirList{items: []string{"a"}}})
	// 错误信息
	gotErr := RenderDirListCard(DirListCardOpts{Title: "t", DirPath: "d", CardWidth: 40, Height: 20, DirList: fakeDirList{}, ErrMsg: "读取失败"})
	if !strings.Contains(gotErr, "读取失败") {
		t.Errorf("RenderDirListCard 缺少错误信息: %q", gotErr)
	}
}
