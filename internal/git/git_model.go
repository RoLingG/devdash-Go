package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// InfoMsg 异步加载完成后的消息类型
type InfoMsg Info

// DirMsg 目录扫描结果（含 .git 的子目录）
type DirMsg struct {
	Dir   string
	Repos []string
}

// LoadInfoFromDir 在 goroutine 中加载指定目录的 Git 数据
func LoadInfoFromDir(dir string) tea.Msg {
	return InfoMsg(FetchInfoFromDir(dir))
}

// ScanDir 扫描目录下所有含 .git 的子目录
func ScanDir(dir string) tea.Msg {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirMsg{Dir: dir, Repos: nil}
	}
	var repos []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitPath := filepath.Join(dir, e.Name(), ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			repos = append(repos, e.Name())
		}
	}
	return DirMsg{Dir: dir, Repos: repos}
}

// LoadInfoFromDirCmd 返回一个执行 LoadInfoFromDir 的 Cmd
func LoadInfoFromDirCmd(dir string) tea.Cmd {
	return func() tea.Msg { return LoadInfoFromDir(dir) }
}

// ScanDirCmd 返回一个执行 ScanDir 的 Cmd
func ScanDirCmd(dir string) tea.Cmd {
	return func() tea.Msg { return ScanDir(dir) }
}

// Model Git 模块状态
type Model struct {
	info     Info
	width    int
	height   int
	scroll   int
	loading  bool
	loaded   bool
	err      error
	repoPath string

	input      component.InputModel
	dirListing bool
	dirPath    string
	dirList    component.ListModel
}

// SetRecent 设置最近记录列表（转发给 InputModel）
func (m *Model) SetRecent(items []string) { m.input.SetRecent(items) }

func (m *Model) Init(defaultRepo string) tea.Cmd {
	if defaultRepo == "" {
		defaultRepo = "."
	}
	m.repoPath = defaultRepo
	return LoadInfoFromDirCmd(defaultRepo)
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

// Update 处理消息
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case InfoMsg:
		m.info = Info(msg)
		m.loaded = true
		m.loading = false
		m.scroll = 0
		m.input.Active = false
		m.dirListing = false

	case DirMsg:
		if msg.Repos == nil || len(msg.Repos) == 0 {
			m.err = fmt.Errorf("no git repos found in: %s", msg.Dir)
			m.input.Open(m.dirPath)
			m.dirListing = false
			return m, nil
		}
		m.dirListing = true
		m.dirPath = msg.Dir
		m.dirList.SetItems(msg.Repos)
		m.input.Active = false
		return m, nil

	case tea.PasteMsg:
		if m.input.Active {
			return m, m.input.Update(msg, nil)
		}
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		if m.dirListing {
			switch key {
			case "up", "k":
				m.dirList.MoveUp()
			case "down", "j":
				m.dirList.MoveDown()
			case "enter":
				selected := m.dirList.Selected()
				fullPath := filepath.Join(m.dirPath, selected)
				m.repoPath = fullPath
				m.dirListing = false
				m.loading = true
				m.err = nil
				return m, tea.Batch(LoadInfoFromDirCmd(fullPath), ui.UpdateCfgCmd("repo", fullPath))
			case "esc":
				m.dirListing = false
				m.input.Prompt = "Repository path:"
				m.input.Open(m.dirPath)
			}
			return m, nil
		}

		if m.input.Active {
			return m, tea.Batch(
				m.input.Update(msg, func(path string) func() tea.Msg {
					if path == "" {
						path = "."
					}
					m.err = nil
					info, err := os.Stat(path)
					if err != nil {
						m.err = fmt.Errorf("path not found: %s", path)
						return nil
					}
					if info.IsDir() {
						gitPath := filepath.Join(path, ".git")
						if gi, gErr := os.Stat(gitPath); gErr == nil && gi.IsDir() {
							m.repoPath = path
							m.loading = true
							return LoadInfoFromDirCmd(path)
						}
						return ScanDirCmd(path)
					}
					m.repoPath = filepath.Dir(path)
					m.loading = true
					return LoadInfoFromDirCmd(filepath.Dir(path))
				}),
				ui.UpdateCfgCmd("repo", m.repoPath),
			)
		}

		switch key {
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			m.scroll++
		case "/":
			m.input.Prompt = "Repository path:"
			m.input.Open(m.repoPath)
		case "ctrl+r":
			m.loading = true
			m.err = nil
			return m, func() tea.Msg { return LoadInfoFromDir(m.repoPath) }
		}
	}
	return m, nil
}

// View 渲染视图
func (m *Model) View() string {
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 目录仓库列表模式
	if m.dirListing {
		return ui.RenderDirListCard(ui.DirListCardOpts{
			Title:     "Git Repositories",
			DirPath:   m.dirPath,
			DirList:   &m.dirList,
			Height:    m.height,
			CardWidth: cardWidth,
		})
	}

	// 路径输入模式
	if m.input.Active {
		errMsg := ""
		if m.err != nil {
			errMsg = m.err.Error()
		}
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       "Change Repository",
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			ErrMsg:      errMsg,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	// 加载中
	if m.loading {
		loadingContent := lipgloss.NewStyle().Foreground(ui.ColAccent).Render("  ⏳ Loading git info...")
		if m.repoPath != "" && m.repoPath != "." {
			loadingContent += "\n\n" + lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  📂 "+m.repoPath)
		}
		return ui.Card("Git", loadingContent, ui.ColAccent, cardWidth)
	}

	// 未加载
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to enter repository path")
		return ui.Card("Git", emptyContent, ui.ColMuted, cardWidth)
	}

	// 错误
	if m.info.Err != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("  ✗ "+m.info.Err.Error()) + "\n\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to change path, 'R' to retry")
		return ui.Card("Git", errContent, ui.ColRed, cardWidth)
	}

	// 主要内容
	var sec []string
	sec = append(sec, m.viewCommits())
	sec = append(sec, m.viewBranches())
	sec = append(sec, m.viewFiles())
	sec = append(sec, m.viewStats())

	full := strings.Join(sec, "\n\n")
	lines := strings.Split(full, "\n")

	top := m.height - 3
	if top < 5 {
		top = 5
	}
	if m.scroll > len(lines)-top {
		m.scroll = len(lines) - top
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	start := m.scroll
	end := start + top
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

func (m *Model) viewCommits() string {
	repoPath := m.repoPath
	if repoPath == "" || repoPath == "." {
		repoPath = "current directory"
	}
	pathLine := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  📂 " + repoPath)
	branch := lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.info.Current)
	header := fmt.Sprintf("  Branch: %s (%d total)", branch, len(m.info.Branches))

	// ahead/behind 状态
	if m.info.Ahead > 0 || m.info.Behind > 0 {
		aheadStyle := lipgloss.NewStyle().Foreground(ui.ColGreen)
		behindStyle := lipgloss.NewStyle().Foreground(ui.ColRed)
		var badges []string
		if m.info.Ahead > 0 {
			badges = append(badges, aheadStyle.Render(fmt.Sprintf("↑%d", m.info.Ahead)))
		}
		if m.info.Behind > 0 {
			badges = append(badges, behindStyle.Render(fmt.Sprintf("↓%d", m.info.Behind)))
		}
		header += "  " + strings.Join(badges, " ")
	}

	// 工作区 dirty 状态
	if m.info.Modified > 0 || m.info.Added > 0 || m.info.Deleted > 0 || m.info.Untracked > 0 {
		var parts []string
		if m.info.Modified > 0 {
			style := lipgloss.NewStyle().Foreground(ui.ColAccent)
			parts = append(parts, style.Render(fmt.Sprintf("%dM", m.info.Modified)))
		}
		if m.info.Added > 0 {
			style := lipgloss.NewStyle().Foreground(ui.ColGreen)
			parts = append(parts, style.Render(fmt.Sprintf("%dA", m.info.Added)))
		}
		if m.info.Deleted > 0 {
			style := lipgloss.NewStyle().Foreground(ui.ColRed)
			parts = append(parts, style.Render(fmt.Sprintf("%dD", m.info.Deleted)))
		}
		if m.info.Untracked > 0 {
			style := lipgloss.NewStyle().Foreground(ui.ColMuted)
			parts = append(parts, style.Render(fmt.Sprintf("%d??", m.info.Untracked)))
		}
		header += "  📝 " + strings.Join(parts, " ")
	}

	var lines []string
	lines = append(lines, pathLine, header, "")

	hw, aw, dw := 14, 14, 14
	gaps := 8
	mw := m.width - 4 - hw - aw - dw - gaps
	if mw > 60 {
		mw = 60
	}
	if mw < 10 {
		mw = 10
	}
	lines = append(lines, fmt.Sprintf("  %s  %s  %s  %s",
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render(ui.PadRight("Hash", hw)),
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render(ui.PadRight("Author", aw)),
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render(ui.PadRight("Date", dw)),
		lipgloss.NewStyle().Foreground(ui.ColMuted).Render("Message"),
	))
	lines = append(lines, "  "+strings.Repeat("─", m.width-10))

	for _, c := range m.info.Commits {
		hash := lipgloss.NewStyle().Foreground(ui.ColAccent).Render(ui.PadRight(c.Hash, hw))
		line := fmt.Sprintf("  %s  %s  %s  %s", hash, ui.PadRight(ui.Truncate(c.Author, aw), aw), ui.PadRight(c.Date, dw), ui.Truncate(c.Message, mw))
		lines = append(lines, line)
	}
	return ui.Box("Git Commits", strings.Join(lines, "\n"), m.width)
}

// 热文件柱状图
var heatLimits = []struct {
	threshold int
	limit     int
}{
	{50, 50}, {100, 100}, {500, 500}, {1000, 1000},
}

func getHeatLimit(maxVal int) int {
	for _, hl := range heatLimits {
		if maxVal <= hl.threshold {
			return hl.limit
		}
	}
	return maxVal
}

func (m *Model) viewFiles() string {
	if len(m.info.Files) == 0 {
		return ui.Box("Hot Files", "  No data", m.width)
	}
	maxVal := 0
	for _, f := range m.info.Files {
		if f.Changes > maxVal {
			maxVal = f.Changes
		}
	}
	limit := getHeatLimit(maxVal)
	barW := m.width - 4 - 30
	if barW < 10 {
		barW = 10
	}
	var lines []string
	for _, f := range m.info.Files {
		lines = append(lines, ui.BarChart(f.File, f.Changes, limit, barW, ui.ColPrimary))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("  Scale: max %d", limit)))
	return ui.Box("Hot Files (most changed)", strings.Join(lines, "\n"), m.width)
}

func (m *Model) viewBranches() string {
	if len(m.info.Branches) == 0 {
		return ui.Box("Branches", "  No data", m.width)
	}
	var lines []string
	for _, branch := range m.info.Branches {
		if branch == m.info.Current {
			lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColGreen).Bold(true).Render("  * "+branch))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("    "+branch))
		}
	}
	return ui.Box("Branches", strings.Join(lines, "\n"), m.width)
}

func (m *Model) viewStats() string {
	totalCommits := len(m.info.Commits)
	totalFiles := len(m.info.Files)
	totalBranches := len(m.info.Branches)
	dates := make(map[string]bool)
	for _, c := range m.info.Commits {
		dates[c.Date] = true
	}
	activeDays := len(dates)
	totalChanges := 0
	for _, f := range m.info.Files {
		totalChanges += f.Changes
	}

	statStyle := lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ui.ColMuted)

	var lines []string
	lines = append(lines, fmt.Sprintf("  %s %s", statStyle.Render(fmt.Sprintf("%d", totalCommits)), labelStyle.Render("commits")))
	lines = append(lines, fmt.Sprintf("  %s %s", statStyle.Render(fmt.Sprintf("%d", totalBranches)), labelStyle.Render("branches")))
	lines = append(lines, fmt.Sprintf("  %s %s", statStyle.Render(fmt.Sprintf("%d", totalFiles)), labelStyle.Render("files changed")))
	lines = append(lines, fmt.Sprintf("  %s %s", statStyle.Render(fmt.Sprintf("%d", activeDays)), labelStyle.Render("active days")))
	lines = append(lines, fmt.Sprintf("  %s %s", statStyle.Render(fmt.Sprintf("%d", totalChanges)), labelStyle.Render("total changes")))
	return ui.Box("Statistics", strings.Join(lines, "\n"), m.width)
}
