package devtools

// devtools_model_test.go — 最近输入记录单元测试
// 测试 submitInput 的记录逻辑，不依赖 UI 渲染。

import (
	"reflect"
	"testing"
)

func TestRecentRecord(t *testing.T) {
	m := &Model{}
	m.initTextarea()
	m.input.SetValue("hello")
	m.submitInput()
	if !reflect.DeepEqual(m.recent, []string{"hello"}) {
		t.Errorf("after first input recent = %v, want [hello]", m.recent)
	}

	// 新输入置顶
	m.input.SetValue("world")
	m.submitInput()
	if !reflect.DeepEqual(m.recent, []string{"world", "hello"}) {
		t.Errorf("recent = %v, want [world hello]", m.recent)
	}

	// 重复输入去重并移到最前
	m.input.SetValue("hello")
	m.submitInput()
	if !reflect.DeepEqual(m.recent, []string{"hello", "world"}) {
		t.Errorf("dedup recent = %v, want [hello world]", m.recent)
	}

	// 空输入不记录
	m.input.SetValue("")
	m.submitInput()
	if !reflect.DeepEqual(m.recent, []string{"hello", "world"}) {
		t.Errorf("empty input recent = %v, want unchanged [hello world]", m.recent)
	}

	// inputValue 应固定为确认值（含空）
	if m.inputValue != "" {
		t.Errorf("inputValue = %q, want \"\" (empty confirmed)", m.inputValue)
	}
	m.input.SetValue("abc")
	m.submitInput()
	if m.inputValue != "abc" {
		t.Errorf("inputValue = %q, want abc", m.inputValue)
	}
}

func TestRecentMaxCap(t *testing.T) {
	m := &Model{}
	m.initTextarea()
	for i := 0; i < 15; i++ {
		m.input.SetValue(string(rune('a'+i%26)) + "xx")
		m.submitInput()
	}
	if len(m.recent) != 10 {
		t.Fatalf("recent len = %d, want 10", len(m.recent))
	}
	// 最新输入应在最前（i=14 → 'a'+14='o'）
	if m.recent[0] != "oxx" {
		t.Errorf("newest = %q, want oxx", m.recent[0])
	}
}
