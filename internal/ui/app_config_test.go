package ui

import (
	"fmt"
	"os"
	"testing"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultCity != "Beijing" {
		t.Errorf("DefaultCity = %q, want Beijing", cfg.DefaultCity)
	}
	// 所有切片应为非 nil 空切片
	if cfg.RecentRepos == nil || cfg.RecentLogFiles == nil || cfg.RecentConfigFiles == nil || cfg.RecentCities == nil {
		t.Error("默认配置切片不应为 nil")
	}
}

// TestAddToRecent 测试最近列表添加逻辑
func TestAddToRecent(t *testing.T) {
	// 空项直接返回原列表
	got := AddToRecent([]string{"a"}, "", 10)
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("空项应返回原列表: %v", got)
	}
	// 新项插入开头
	got = AddToRecent([]string{"a", "b"}, "c", 10)
	if got[0] != "c" || len(got) != 3 {
		t.Errorf("AddToRecent 插入开头失败: %v", got)
	}
	// 去重：重复项移除并提到开头
	got = AddToRecent([]string{"a", "b"}, "a", 10)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("AddToRecent 去重失败: %v", got)
	}
	// 超限截断到 maxItems
	list := make([]string, 0, 15)
	for i := 0; i < 15; i++ {
		list = append(list, fmt.Sprintf("item%d", i))
	}
	got = AddToRecent(list, "new", 10)
	if len(got) != 10 {
		t.Errorf("AddToRecent 超限截断失败: len=%d, want 10", len(got))
	}
	if got[0] != "new" {
		t.Errorf("AddToRecent 新项应在最前: %v", got)
	}
}

// TestConfigCmd 测试配置变更 Cmd 消息构造
func TestConfigCmd(t *testing.T) {
	cmd := UpdateCfgCmd("city", "Shanghai")
	if cmd == nil {
		t.Fatal("UpdateCfgCmd 返回 nil")
	}
	msg := cmd()
	cm, ok := msg.(CfgChangedMsg)
	if !ok || cm.Key != "city" || cm.Value != "Shanghai" {
		t.Errorf("UpdateCfgCmd 消息错误: %+v", msg)
	}

	// 复杂类型数据
	cmdData := UpdateCfgDataCmd("savedRoutes", []RouteConfig{{Dest: "10.0.0.0"}})
	msgData := cmdData()
	cm2, ok := msgData.(CfgChangedMsg)
	if !ok || cm2.Key != "savedRoutes" {
		t.Errorf("UpdateCfgDataCmd 消息错误: %+v", msgData)
	}
	routes, ok := cm2.Data.([]RouteConfig)
	if !ok || len(routes) != 1 || routes[0].Dest != "10.0.0.0" {
		t.Errorf("UpdateCfgDataCmd Data 错误: %+v", cm2.Data)
	}
}

// TestSaveLoadConfig 测试配置保存与读取的往返
// 注意：SaveConfig/LoadConfig 写入的是测试二进制同目录（go test 的临时目录），不会污染项目目录
func TestSaveLoadConfig(t *testing.T) {
	cfgPath := GetConfigPath()
	// 清理可能残留的配置文件
	os.Remove(cfgPath)

	// 文件不存在时 LoadConfig 返回默认 + 错误
	cfg, err := LoadConfig()
	if err == nil {
		t.Error("文件不存在时 LoadConfig 应返回错误")
	}
	if cfg.DefaultCity != "Beijing" {
		t.Errorf("回退配置 DefaultCity = %q, want Beijing", cfg.DefaultCity)
	}

	// 保存自定义配置
	want := DefaultConfig()
	want.DefaultCity = "Shenzhen"
	want.RecentRepos = []string{"repo1", "repo2"}
	if err := SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig 错误: %v", err)
	}
	if !ConfigExists() {
		t.Error("SaveConfig 后 ConfigExists 应为 true")
	}

	// 加载回读
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig 错误: %v", err)
	}
	if got.DefaultCity != "Shenzhen" {
		t.Errorf("回读 DefaultCity = %q, want Shenzhen", got.DefaultCity)
	}
	if len(got.RecentRepos) != 2 || got.RecentRepos[0] != "repo1" {
		t.Errorf("回读 RecentRepos = %v", got.RecentRepos)
	}

	// 清理
	os.Remove(cfgPath)
}
