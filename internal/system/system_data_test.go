package system

import (
	"testing"
	"time"
)

// TestSysTickCmd 测试定时刷新 Cmd（延迟后产生 SysTickMsg）
func TestSysTickCmd(t *testing.T) {
	cmd := SysTickCmd(time.Millisecond)
	if cmd == nil {
		t.Fatal("SysTickCmd 返回 nil")
	}
	msg := cmd()
	if _, ok := msg.(SysTickMsg); !ok {
		t.Errorf("SysTickCmd 消息类型 = %T, want SysTickMsg", msg)
	}
}

// TestFetchCmds 测试 Cmd 构造
func TestFetchCmds(t *testing.T) {
	if FetchSystemInfoCmd() == nil {
		t.Error("FetchSystemInfoCmd 返回 nil")
	}
	if FetchProcessesCmd() == nil {
		t.Error("FetchProcessesCmd 返回 nil")
	}
}

// TestFetchSystemInfo 测试真实获取系统信息（真实机器上应返回数据）
func TestFetchSystemInfo(t *testing.T) {
	msg := FetchSystemInfo()
	info, ok := msg.(SysInfoMsg)
	if !ok {
		t.Fatalf("FetchSystemInfo 返回类型错误: %T", msg)
	}
	// 机器环境允许失败（如受限 CI），但调用应返回 SysInfoMsg 而不 panic
	_ = info
}
