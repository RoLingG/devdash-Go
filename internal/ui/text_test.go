package ui

import (
	"testing"
)

// TestRuneLen 测试 rune 长度计算
func TestRuneLen(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"空字符串", "", 0},
		{"纯 ASCII", "hello", 5},
		{"中文", "你好世界", 4},
		{"混合", "hi你好", 4},
		{"emoji", "🎉🎊", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneLen(tt.input); got != tt.want {
				t.Errorf("RuneLen(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestRuneSubstr 测试 rune 子串截取
func TestRuneSubstr(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		start int
		end   int
		want  string
	}{
		{"正常截取", "hello", 1, 3, "el"},
		{"中文截取", "你好世界", 1, 3, "好世"},
		{"超出范围", "hello", 0, 10, "hello"},
		{"start 超出", "hello", 10, 15, ""},
		{"空字符串", "", 0, 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneSubstr(tt.s, tt.start, tt.end); got != tt.want {
				t.Errorf("RuneSubstr(%q, %d, %d) = %q, want %q", tt.s, tt.start, tt.end, got, tt.want)
			}
		})
	}
}

// TestRuneInsert 测试 rune 插入
func TestRuneInsert(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		insert string
		idx    int
		want   string
	}{
		{"开头插入", "world", "hello ", 0, "hello world"},
		{"中间插入", "helo", "l", 2, "hello"},
		{"末尾插入", "hello", " world", 5, "hello world"},
		{"中文插入", "你好", "世界", 1, "你世界好"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneInsert(tt.s, tt.insert, tt.idx); got != tt.want {
				t.Errorf("RuneInsert(%q, %q, %d) = %q, want %q", tt.s, tt.insert, tt.idx, got, tt.want)
			}
		})
	}
}

// TestRuneDeleteAt 测试 rune 删除
func TestRuneDeleteAt(t *testing.T) {
	tests := []struct {
		name string
		s    string
		idx  int
		want string
	}{
		{"删除首字符", "hello", 0, "ello"},
		{"删除中间", "hello", 2, "helo"},
		{"删除末尾", "hello", 4, "hell"},
		{"越界负数", "hello", -1, "hello"},
		{"越界正数", "hello", 10, "hello"},
		{"中文删除", "你好世", 1, "你世"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneDeleteAt(tt.s, tt.idx); got != tt.want {
				t.Errorf("RuneDeleteAt(%q, %d) = %q, want %q", tt.s, tt.idx, got, tt.want)
			}
		})
	}
}

// TestRuneToByteIdx 测试 rune 索引转 byte 索引
func TestRuneToByteIdx(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		runeIdx int
		want    int
	}{
		{"ASCII", "hello", 2, 2},
		{"中文", "你好世界", 2, 6},
		{"混合", "hi你好", 2, 2},
		{"越界", "hello", 10, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneToByteIdx(tt.s, tt.runeIdx); got != tt.want {
				t.Errorf("RuneToByteIdx(%q, %d) = %d, want %d", tt.s, tt.runeIdx, got, tt.want)
			}
		})
	}
}

// TestPadRight 测试右侧填充空格
func TestPadRight(t *testing.T) {
	tests := []struct {
		name string
		s    string
		w    int
		want string
	}{
		{"需要填充", "hi", 5, "hi   "},
		{"不需要填充", "hello", 5, "hello"},
		{"超出宽度", "hello world", 5, "hello world"},
		{"空字符串", "", 3, "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PadRight(tt.s, tt.w); got != tt.want {
				t.Errorf("PadRight(%q, %d) = %q, want %q", tt.s, tt.w, got, tt.want)
			}
		})
	}
}

// TestTruncate 测试字符串截断
func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"不需要截断", "hello", 10, "hello"},
		{"需要截断", "hello world", 8, "hello..."},
		{"宽度刚好等于max", "hi", 2, "hi"},
		{"超出1个字符", "hello", 4, "h..."},
		{"刚好够", "hello", 5, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.s, tt.max); got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

// TestForceTruncate 测试强制按可见列宽截断（中文按 2 列计算）
func TestForceTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"不需要截断", "hello", 10, "hello"},
		{"ASCII 截断", "hello", 3, "hel"},
		{"中文按列宽", "你好世界", 4, "你好"}, // 每字 2 列，4 列 = 2 字
		{"恰好截断", "hello", 5, "hello"},
		{"空字符串", "", 3, ""},
		{"max为0", "hello", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ForceTruncate(tt.s, tt.max); got != tt.want {
				t.Errorf("ForceTruncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
