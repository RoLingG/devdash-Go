package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// kp 构造一个 tea.KeyPressMsg，使 msg.String() 与项目代码匹配（如 "up"/"ctrl+f"/"/"）
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
	}
	return []rune(s)[0]
}

// newGitModel 构造一个尺寸合理的默认 Model
func newGitModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

func TestGitInit(t *testing.T) {
	m := newGitModel()
	cmd := m.Init(".")
	if cmd == nil {
		t.Fatal("Init 返回 nil Cmd")
	}
	if m.repoPath != "." {
		t.Errorf("repoPath = %q, want %q", m.repoPath, ".")
	}

	// Init 传入空串 → 默认 "."
	m2 := newGitModel()
	if m2.Init("") == nil {
		t.Fatal("Init(\"\") 返回 nil Cmd")
	}
	if m2.repoPath != "." {
		t.Errorf("Init(\"\") repoPath = %q, want %q", m2.repoPath, ".")
	}
}

func TestGitUpdateInfoMsg(t *testing.T) {
	m := newGitModel()
	m.repoPath = "."
	m.loading = true

	info := Info{
		Branches: []string{"main"},
		Current:  "main",
		Commits:  []Commit{{Hash: "abc", Author: "A", Date: "2026-08-01", Message: "fix"}},
	}
	m.Update(InfoMsg(info))
	if !m.loaded {
		t.Error("InfoMsg 后 loaded 应为 true")
	}
	if m.loading {
		t.Error("InfoMsg 后 loading 应为 false")
	}
	if m.info.Current != "main" {
		t.Errorf("info.Current = %q, want main", m.info.Current)
	}
	if m.cachedPath != "." {
		t.Errorf("cachedPath = %q, want %q", m.cachedPath, ".")
	}
	if m.input.Active {
		t.Error("InfoMsg 后输入框应关闭")
	}
	if m.dirListing {
		t.Error("InfoMsg 后不应处于目录列表模式")
	}
}

func TestGitUpdateDirMsg(t *testing.T) {
	m := newGitModel()

	// 有仓库 → 进入目录列表模式
	cmd := m.Update(DirMsg{Dir: "/repo", Repos: []string{"a", "b"}})
	if cmd != nil {
		t.Error("DirMsg 有仓库时不应返回 Cmd")
	}
	if !m.dirListing {
		t.Error("应进入目录列表模式")
	}
	if m.dirPath != "/repo" {
		t.Errorf("dirPath = %q, want /repo", m.dirPath)
	}

	// 无仓库 → 错误 + 重新打开输入框
	m2 := newGitModel()
	m2.input.Active = true
	m2.dirListing = true
	cmd2 := m2.Update(DirMsg{Dir: "/empty", Repos: nil})
	if cmd2 != nil {
		t.Error("无仓库时不应返回 Cmd")
	}
	if m2.err == nil {
		t.Error("无仓库时应设置 err")
	}
	if m2.dirListing {
		t.Error("无仓库时应退出目录列表模式")
	}
}

func TestGitUpdateKeyDirListing(t *testing.T) {
	m := newGitModel()
	m.dirListing = true
	m.dirPath = "/repo"
	m.dirList.SetItems([]string{"a", "b", "c"})

	// up/down 移动
	m.Update(kp("down"))
	m.Update(kp("down"))
	if got := m.dirList.Selected(); got != "c" {
		t.Errorf("down 两次后选中 = %q, want c", got)
	}
	m.Update(kp("up"))
	if got := m.dirList.Selected(); got != "b" {
		t.Errorf("up 后选中 = %q, want b", got)
	}

	// enter → 选中仓库加载
	cmd := m.Update(kp("enter"))
	if cmd == nil {
		t.Error("enter 应返回 Batch Cmd（加载 + 更新配置）")
	}
	if m.dirListing {
		t.Error("enter 后应退出目录列表模式")
	}
	if m.repoPath != filepath.Join("/repo", "b") {
		t.Errorf("repoPath = %q, want %q", m.repoPath, filepath.Join("/repo", "b"))
	}
	if !m.loading {
		t.Error("enter 后 loading 应为 true")
	}

	// esc → 退出目录列表，打开输入框
	m3 := newGitModel()
	m3.dirListing = true
	m3.dirPath = "/repo"
	m3.dirList.SetItems([]string{"a"})
	m3.Update(kp("esc"))
	if m3.dirListing {
		t.Error("esc 后应退出目录列表模式")
	}
	if !m3.input.Active {
		t.Error("esc 后应打开输入框")
	}
}

func TestGitUpdateKeyNavigation(t *testing.T) {
	// 普通模式下滚动导航
	m := newGitModel()
	m.loaded = true
	m.scroll = 2

	m.Update(kp("up"))
	if m.scroll != 1 {
		t.Errorf("up 后 scroll = %d, want 1", m.scroll)
	}
	m.Update(kp("home"))
	if m.scroll != 0 {
		t.Errorf("home 后 scroll = %d, want 0", m.scroll)
	}
	m.Update(kp("end"))
	if m.scroll != 1<<30 {
		t.Errorf("end 后 scroll = %d, want 1<<30", m.scroll)
	}
	m.Update(kp("down"))
	if m.scroll != 1<<30+1 {
		t.Errorf("down 后 scroll = %d, want 1<<30+1", m.scroll)
	}

	// scroll 已在 0，up 不再减
	m2 := newGitModel()
	m2.scroll = 0
	m2.Update(kp("up"))
	if m2.scroll != 0 {
		t.Errorf("scroll=0 时 up 后 = %d, want 0", m2.scroll)
	}
}

func TestGitUpdateOpenInput(t *testing.T) {
	m := newGitModel()
	m.repoPath = "/repo"
	cmd := m.Update(kp("/"))
	if cmd != nil {
		t.Error("打开输入框不应返回 Cmd")
	}
	if !m.input.Active {
		t.Error("按 / 后应打开输入框")
	}
	if m.input.Prompt == "" {
		t.Error("输入框 Prompt 不应为空")
	}
}

func TestGitUpdateRefresh(t *testing.T) {
	// ctrl+r 刷新（repoPath 非空且缓存未命中 → 返回加载 Cmd）
	m := newGitModel()
	m.repoPath = "."
	cmd := m.Update(kp("ctrl+r"))
	if cmd == nil {
		t.Error("ctrl+r 应返回加载 Cmd")
	}
	if !m.loading {
		t.Error("ctrl+r 后 loading 应为 true")
	}
	if m.err != nil {
		t.Errorf("ctrl+r 后 err 应为 nil, got %v", m.err)
	}

	// repoPath 为空 → 不返回 Cmd
	m2 := newGitModel()
	if cmd := m2.Update(kp("ctrl+r")); cmd != nil {
		t.Error("repoPath 为空时 ctrl+r 不应返回 Cmd")
	}
}

func TestGitViewStates(t *testing.T) {
	// 目录列表模式
	m := newGitModel()
	m.dirListing = true
	m.dirPath = "/repo"
	m.dirList.SetItems([]string{"a", "b"})
	v := m.View()
	if !strings.Contains(v, "Git Repositories") {
		t.Errorf("目录列表模式视图缺少标题: %q", v)
	}

	// 输入框模式
	m2 := newGitModel()
	m2.input.Active = true
	m2.input.Prompt = "Repository path:"
	m2.input.Value = "abc"
	m2.input.Cursor = 3
	v2 := m2.View()
	if !strings.Contains(v2, "Change Repository") {
		t.Errorf("输入框模式视图缺少标题: %q", v2)
	}

	// 加载中
	m3 := newGitModel()
	m3.loading = true
	m3.repoPath = "/repo"
	v3 := m3.View()
	if !strings.Contains(v3, "Loading") {
		t.Errorf("加载中视图缺少 Loading: %q", v3)
	}

	// 未加载
	m4 := newGitModel()
	v4 := m4.View()
	if !strings.Contains(v4, "Press") {
		t.Errorf("未加载视图缺少提示: %q", v4)
	}

	// 错误
	m5 := newGitModel()
	m5.loaded = true
	m5.info.Err = os.ErrNotExist
	v5 := m5.View()
	if !strings.Contains(v5, "Press") {
		t.Errorf("错误视图缺少提示: %q", v5)
	}
}

func TestGitViewMainContent(t *testing.T) {
	m := newGitModel()
	m.loaded = true
	m.repoPath = "/repo"
	m.info = Info{
		Branches:  []string{"main", "dev"},
		Current:   "main",
		Commits:   []Commit{{Hash: "abc123", Author: "Alice", Date: "2026-08-01", Message: "fix bug"}},
		Files:     []FileChange{{File: "main.go", Changes: 5}},
		Ahead:     2,
		Behind:    1,
		Modified:  3,
		Untracked: 2,
	}
	v := m.View()
	for _, want := range []string{"Git Commits", "abc123", "Branches", "main", "Hot Files", "Statistics", "↑2", "↓1", "3M", "2??"} {
		if !strings.Contains(v, want) {
			t.Errorf("主视图缺少 %q", want)
		}
	}

	// 空数据不 panic，显示 No data
	m2 := newGitModel()
	m2.loaded = true
	m2.repoPath = "/repo"
	m2.info = Info{Current: "main"}
	v2 := m2.View()
	if !strings.Contains(v2, "No data") {
		t.Errorf("空数据视图缺少 No data: %q", v2)
	}

	// 小高度（边界）不 panic
	m3 := newGitModel()
	m3.UpdateSize(100, 4)
	m3.loaded = true
	m3.info = Info{Current: "main"}
	if v3 := m3.View(); v3 == "" {
		t.Error("小高度视图为空")
	}
}

func TestGitScanDirCmd(t *testing.T) {
	if ScanDirCmd(".") == nil {
		t.Error("ScanDirCmd 返回 nil")
	}
	if LoadInfoFromDirCmd(".") == nil {
		t.Error("LoadInfoFromDirCmd 返回 nil")
	}
}

func TestGitSetRecent(t *testing.T) {
	m := newGitModel()
	m.SetRecent([]string{"a", "b"})
	if len(m.input.RecentItems) != 2 {
		t.Errorf("RecentItems = %v, want 2 项", m.input.RecentItems)
	}
}
