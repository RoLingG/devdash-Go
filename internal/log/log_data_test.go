package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDetectLevel 测试日志级别检测
func TestDetectLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ERROR 大写", "ERROR something failed", "ERROR"},
		{"error 小写", "error something failed", "ERROR"},
		{"FATAL", "FATAL system crash", "ERROR"},
		{"PANIC", "PANIC goroutine error", "ERROR"},
		{"WARN", "WARN disk space low", "WARN"},
		{"warn 小写", "warn disk space low", "WARN"},
		{"INFO", "INFO server started", "INFO"},
		{"info 小写", "info server started", "INFO"},
		{"DEBUG", "DEBUG cache hit", "DEBUG"},
		{"TRACE", "TRACE function call", "DEBUG"},
		{"普通文本", "nothing special here", "OTHER"},
		{"空字符串", "", "OTHER"},
		{"混合大小写", "Error in module", "ERROR"},
		{"多个关键字", "ERROR: WARN level changed", "ERROR"}, // ERROR 在前，优先匹配
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectLevel(tt.input); got != tt.want {
				t.Errorf("detectLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestLoadFromFile 测试文件加载
func TestLoadFromFile(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	t.Run("正常文件", func(t *testing.T) {
		// 创建测试文件
		content := "INFO server started\nWARN disk low\nERROR connection failed\n"
		path := filepath.Join(tmpDir, "test.log")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err != nil {
			t.Fatalf("unexpected error: %v", loadMsg.Err)
		}
		if len(loadMsg.Lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(loadMsg.Lines))
		}
		// 检查级别检测
		if loadMsg.Lines[0].Level != "INFO" {
			t.Errorf("line 0: expected INFO, got %s", loadMsg.Lines[0].Level)
		}
		if loadMsg.Lines[1].Level != "WARN" {
			t.Errorf("line 1: expected WARN, got %s", loadMsg.Lines[1].Level)
		}
		if loadMsg.Lines[2].Level != "ERROR" {
			t.Errorf("line 2: expected ERROR, got %s", loadMsg.Lines[2].Level)
		}
	})

	t.Run("空文件", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty.log")
		_ = os.WriteFile(path, []byte(""), 0644)

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err != nil {
			t.Fatalf("unexpected error: %v", loadMsg.Err)
		}
		if len(loadMsg.Lines) != 0 {
			t.Errorf("expected 0 lines, got %d", len(loadMsg.Lines))
		}
	})

	t.Run("文件不存在", func(t *testing.T) {
		path := filepath.Join(tmpDir, "not_exist.log")

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(loadMsg.Err.Error(), "cannot open") {
			t.Errorf("expected 'cannot open' in error, got: %v", loadMsg.Err)
		}
	})

	t.Run("空行被保留", func(t *testing.T) {
		content := "line1\n\nline3\n"
		path := filepath.Join(tmpDir, "with_empty.log")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		// 注意：当前实现不跳过空行，空行会被保留
		if len(loadMsg.Lines) != 3 {
			t.Errorf("expected 3 lines (including empty), got %d", len(loadMsg.Lines))
		}
	})
}

// TestLoadFromFile_BigFileProtection 测试大文件保护
func TestLoadFromFile_BigFileProtection(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("超过行数限制", func(t *testing.T) {
		// 创建一个超过 MaxLines 行的文件
		path := filepath.Join(tmpDir, "many_lines.log")
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		for i := 0; i < MaxLines+100; i++ {
			_, _ = f.WriteString("INFO log line\n")
		}
		f.Close()

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err != nil {
			t.Fatalf("unexpected error: %v", loadMsg.Err)
		}
		// 应该被截断到 MaxLines
		if len(loadMsg.Lines) != MaxLines {
			t.Errorf("expected %d lines, got %d", MaxLines, len(loadMsg.Lines))
		}
		if !loadMsg.Trunc {
			t.Error("expected Trunc=true")
		}
		if loadMsg.Warn == "" {
			t.Error("expected warning message")
		}
	})

	t.Run("刚好等于行数限制", func(t *testing.T) {
		path := filepath.Join(tmpDir, "exact_lines.log")
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		for i := 0; i < MaxLines; i++ {
			_, _ = f.WriteString("INFO log line\n")
		}
		f.Close()

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if len(loadMsg.Lines) != MaxLines {
			t.Errorf("expected %d lines, got %d", MaxLines, len(loadMsg.Lines))
		}
		// 注意：当前实现是保守策略，读取到 MaxLines 行时无法预知后面是否还有更多行
		// 所以即使刚好等于 MaxLines，也会标记为截断
		if !loadMsg.Trunc {
			t.Error("expected Trunc=true (conservative strategy)")
		}
	})

	t.Run("超过文件大小限制", func(t *testing.T) {
		// 创建一个超过 MaxFileSize 的文件（只写元数据，不实际填充内容）
		path := filepath.Join(tmpDir, "huge.log")
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		_, _ = f.Seek(MaxFileSize, 0)
		_, _ = f.WriteString("end")
		f.Close()

		msg := LoadFromFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err == nil {
			t.Fatal("expected error for oversized file")
		}
		if !strings.Contains(loadMsg.Err.Error(), "文件过大") {
			t.Errorf("expected '文件过大' in error, got: %v", loadMsg.Err)
		}
	})
}

// TestSampleLogLines 测试示例数据
func TestSampleLogLines(t *testing.T) {
	lines := SampleLogLines()
	if len(lines) == 0 {
		t.Fatal("expected non-empty sample lines")
	}
	// 检查每行都有 Level
	for i, line := range lines {
		if line.Level == "" {
			t.Errorf("line %d: expected non-empty Level", i)
		}
		if line.Raw == "" {
			t.Errorf("line %d: expected non-empty Raw", i)
		}
	}
}

// TestScanDir 测试扫描目录下的 .log 文件（大小写不敏感）
func TestScanDir(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.log", "b.LOG", "c.txt", "d.log"} {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644)
	}
	dm, ok := ScanDir(dir).(DirMsg)
	if !ok {
		t.Fatalf("ScanDir 返回类型错误: %T", dm)
	}
	if len(dm.Files) != 3 {
		t.Errorf("Files = %v, want 3 个 log 文件", dm.Files)
	}

	// 不存在的目录 → Files nil
	dm, _ = ScanDir(filepath.Join(dir, "nope")).(DirMsg)
	if dm.Files != nil {
		t.Errorf("不存在目录应返回 nil Files: %v", dm.Files)
	}
}

// TestReceiveTailCmd 测试 tail 数据接收 Cmd
func TestReceiveTailCmd(t *testing.T) {
	// 正常数据：分行 + 级别识别
	ch := make(chan []byte, 1)
	ch <- []byte("ERROR one\nINFO two\n")
	close(ch)
	tm, ok := receiveTailCmd(ch)().(TailDataMsg)
	if !ok {
		t.Fatal("receiveTailCmd 返回类型错误")
	}
	if tm.Done {
		t.Error("有数据时 Done 应为 false")
	}
	if len(tm.Lines) != 2 {
		t.Fatalf("Lines = %d, want 2", len(tm.Lines))
	}
	if tm.Lines[0].Level != "ERROR" || tm.Lines[1].Level != "INFO" {
		t.Errorf("级别检测错误: %+v", tm.Lines)
	}

	// 空行跳过
	ch2 := make(chan []byte, 1)
	ch2 <- []byte("a\n\nb\n")
	close(ch2)
	tm2 := receiveTailCmd(ch2)().(TailDataMsg)
	if len(tm2.Lines) != 2 {
		t.Errorf("空行未跳过: %+v", tm2.Lines)
	}

	// channel 已关闭（无数据）→ Done=true
	closed := make(chan []byte)
	close(closed)
	tm3 := receiveTailCmd(closed)().(TailDataMsg)
	if !tm3.Done {
		t.Error("channel 关闭后 Done 应为 true")
	}
}

// TestWatchFile 测试文件监听 goroutine（读取已有内容）
func TestWatchFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tail.log")
	content := "line one\nline two\n"
	os.WriteFile(path, []byte(content), 0644)

	done := make(chan struct{})
	ch := watchFile(path, 0, done)
	defer close(done)

	// 应读取到已有全部内容
	select {
	case data := <-ch:
		if string(data) != content {
			t.Errorf("读取内容 = %q, want %q", string(data), content)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watchFile 超时未返回内容")
	}
}

// TestLogCmds 测试 Cmd 构造
func TestLogCmds(t *testing.T) {
	if LoadFromFileCmd("x") == nil {
		t.Error("LoadFromFileCmd 返回 nil")
	}
	if ScanDirCmd("x") == nil {
		t.Error("ScanDirCmd 返回 nil")
	}
}
