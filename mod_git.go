// ============================================================================
// mod_git.go — Git 仓库可视化模块
//
// 三个区块：提交历史 / 热文件(柱状图) / 贡献者(进度条)
// ↑↓ 滚动，/ 输入路径（支持目录扫描 .git 仓库）
// ============================================================================

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// ==================== 消息类型 ====================

// gitInfoMsg 异步加载完成后的消息类型
type gitInfoMsg GitInfo

// gitDirMsg 目录扫描结果（含 .git 的子目录）
type gitDirMsg struct {
	dir   string
	repos []string
}

// ==================== 加载函数 ====================

// loadGitInfoFromDir 在 goroutine 中加载指定目录的 Git 数据
func loadGitInfoFromDir(dir string) tea.Msg {
	return gitInfoMsg(fetchGitInfoFromDir(dir))
}

// scanGitDir 扫描目录下所有含 .git 的子目录
func scanGitDir(dir string) tea.Msg {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return gitDirMsg{dir: dir, repos: nil}
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
	return gitDirMsg{dir: dir, repos: repos}
}

// ==================== 模型 ====================

type gitModel struct {
	info     GitInfo
	width    int
	height   int
	scroll   int
	loading  bool
	loaded   bool
	err      error
	repoPath string // 当前仓库路径

	// 路径输入模式
	input inputModel // 通用输入组件

	// 目录仓库列表模式
	dirListing bool      // 是否在显示目录仓库列表
	dirPath    string    // 当前目录路径
	dirList    listModel // 通用列表组件
}

func (m *gitModel) Init() tea.Cmd {
	m.repoPath = "."
	return func() tea.Msg { return loadGitInfoFromDir(".") }
}

func (m *gitModel) UpdateSize(w, h int) { m.width = w; m.height = h }

// ==================== 更新逻辑 ====================

func (m *gitModel) Update(msg tea.Msg) (*gitModel, tea.Cmd) {
	switch msg := msg.(type) {

	// Git 数据加载完成
	case gitInfoMsg:
		m.info = GitInfo(msg)
		m.loaded = true
		m.loading = false
		m.scroll = 0
		m.input.active = false
		m.dirListing = false

	// 目录扫描完成
	case gitDirMsg:
		if msg.repos == nil || len(msg.repos) == 0 {
			m.err = fmt.Errorf("no git repos found in: %s", msg.dir)
			m.input.Open(m.dirPath)
			m.dirListing = false
			return m, nil
		}
		m.dirListing = true
		m.dirPath = msg.dir
		m.dirList.SetItems(msg.repos) // 使用 listModel
		m.input.active = false
		return m, nil

	// 粘贴
	case tea.PasteMsg:
		if m.input.active {
			return m, m.input.Update(msg, nil)
		}
		return m, nil

	// 按键
	case tea.KeyPressMsg:
		key := msg.String()

		// ---- 目录仓库列表模式 ----
		if m.dirListing {
			switch key {
			case "up", "k":
				m.dirList.MoveUp() // 使用 listModel
			case "down", "j":
				m.dirList.MoveDown() // 使用 listModel
			case "enter":
				selected := m.dirList.Selected() // 使用 listModel
				fullPath := filepath.Join(m.dirPath, selected)
				m.repoPath = fullPath
				m.dirListing = false
				m.loading = true
				m.err = nil
				return m, func() tea.Msg { return loadGitInfoFromDir(fullPath) }
			case "esc":
				m.dirListing = false
				m.input.prompt = "Repository path:"
				m.input.Open(m.dirPath)
			}
			return m, nil
		}

		// ---- 路径输入模式 ----
		if m.input.active {
			return m, m.input.Update(msg, func(path string) tea.Cmd {
				if path == "" {
					path = "."
				}
				m.err = nil
				// 检测是目录还是文件
				info, err := os.Stat(path)
				if err != nil {
					m.err = fmt.Errorf("path not found: %s", path)
					return nil
				}
				if info.IsDir() {
					// 检查是否本身就是 git 仓库
					gitPath := filepath.Join(path, ".git")
					if gi, gerr := os.Stat(gitPath); gerr == nil && gi.IsDir() {
						// 是 git 仓库，直接加载
						m.repoPath = path
						m.loading = true
						return func() tea.Msg { return loadGitInfoFromDir(path) }
					}
					// 不是 git 仓库，扫描子目录
					return func() tea.Msg { return scanGitDir(path) }
				}
				// 是文件（不太常见），用其所在目录
				m.repoPath = filepath.Dir(path)
				m.loading = true
				return func() tea.Msg { return loadGitInfoFromDir(filepath.Dir(path)) }
			})
		}

		// ---- 正常模式 ----
		switch key {
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			m.scroll++
		case "/":
			m.input.prompt = "Repository path:"
			m.input.Open(m.repoPath)
		case "R":
			m.loading = true
			m.err = nil
			return m, func() tea.Msg { return loadGitInfoFromDir(m.repoPath) }
		}
	}
	return m, nil
}

// ==================== 视图 ====================

func (m *gitModel) View() string {
	// 卡片宽度（自适应终端宽度）
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// ---- 目录仓库列表模式 ----
	if m.dirListing {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colSecondary).Render("  📂 " + m.dirPath))
		sb.WriteString("\n\n")

		maxShow := m.height - 10
		if maxShow < 5 {
			maxShow = 5
		}

		// 使用 listModel 渲染
		highlightStyle := lipgloss.NewStyle().Foreground(colAccent)
		listContent := m.dirList.RenderWithFormat(maxShow, func(item string, selected bool) string {
			if selected {
				return highlightStyle.Render("  > " + item)
			}
			return lipgloss.NewStyle().Foreground(colText).Render("    " + item)
		})

		sb.WriteString(listContent)
		sb.WriteString("\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  ↑↓ select  Enter open  Esc back"))
		return card("Git Repositories", sb.String(), colSecondary, cardWidth)
	}

	// ---- 路径输入模式 ----
	if m.input.active {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render("  " + m.input.prompt))
		sb.WriteString("\n")

		// 绘制输入行，光标位置用 | 标记
		before := runeSubstr(m.input.value, 0, m.input.cursor)
		after := runeSubstr(m.input.value, m.input.cursor, runeLen(m.input.value))
		inputLine := "  > " + before + lipgloss.NewStyle().Foreground(colAccent).Render("|") + after
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render(inputLine))
		sb.WriteString("\n\n")

		if m.err != nil {
			sb.WriteString("  " + lipgloss.NewStyle().Foreground(colRed).Render("✗ "+m.err.Error()))
			sb.WriteString("\n\n")
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  Enter confirm  ←→ cursor  Home/End  Esc cancel"))
		return card("Change Repository", sb.String(), colSecondary, cardWidth)
	}

	// ---- 加载中 ----
	if m.loading {
		loadingContent := lipgloss.NewStyle().Foreground(colAccent).Render("  ⏳ Loading git info...")
		if m.repoPath != "" && m.repoPath != "." {
			loadingContent += "\n\n" + lipgloss.NewStyle().Foreground(colMuted).Render("  📂 "+m.repoPath)
		}
		return card("Git", loadingContent, colAccent, cardWidth)
	}

	// ---- 未加载 ----
	if !m.loaded {
		emptyContent := lipgloss.NewStyle().Foreground(colMuted).Render("  Press '/' to enter repository path")
		return card("Git", emptyContent, colMuted, cardWidth)
	}

	// ---- 错误 ----
	if m.info.Err != nil {
		errContent := lipgloss.NewStyle().Foreground(colRed).Render("  ✗ "+m.info.Err.Error()) + "\n\n"
		errContent += lipgloss.NewStyle().Foreground(colMuted).Render("  Press '/' to change path, 'r' to retry")
		return card("Git", errContent, colRed, cardWidth)
	}

	// ---- 主要内容 ----
	var sec []string
	sec = append(sec, m.viewCommits())
	sec = append(sec, m.viewBranches())
	sec = append(sec, m.viewFiles())
	sec = append(sec, m.viewStats())

	full := strings.Join(sec, "\n\n")
	lines := strings.Split(full, "\n")

	// 可滚动视口
	top := m.height - 6
	if top < 5 {
		top = 5
	}
	start := m.scroll
	if start > len(lines)-top {
		start = len(lines) - top
	}
	if start < 0 {
		start = 0
	}
	end := start + top
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// viewCommits 提交历史
func (m *gitModel) viewCommits() string {
	repoPath := m.repoPath
	if repoPath == "" || repoPath == "." {
		repoPath = "current directory"
	}
	pathLine := lipgloss.NewStyle().Foreground(colMuted).Render("  📂 " + repoPath)

	branch := lipgloss.NewStyle().Foreground(colAccent).Render(m.info.Current)
	header := fmt.Sprintf("  Branch: %s (%d total)", branch, len(m.info.Branches))

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
		lipgloss.NewStyle().Foreground(colMuted).Render(padRight("Hash", hw)),
		lipgloss.NewStyle().Foreground(colMuted).Render(padRight("Author", aw)),
		lipgloss.NewStyle().Foreground(colMuted).Render(padRight("Date", dw)),
		lipgloss.NewStyle().Foreground(colMuted).Render("Message"),
	))
	lines = append(lines, "  "+strings.Repeat("─", m.width-10))

	for _, c := range m.info.Commits {
		hash := lipgloss.NewStyle().Foreground(colAccent).Render(padRight(c.Hash, hw))
		line := fmt.Sprintf("  %s  %s  %s  %s", hash, padRight(truncate(c.Author, aw), aw), padRight(c.Date, dw), truncate(c.Message, mw))
		lines = append(lines, line)
	}
	return box("Git Commits", strings.Join(lines, "\n"), m.width)
}

// viewFiles 热文件柱状图
var heatLimits = []struct {
	threshold int
	limit     int
}{
	{50, 50},
	{100, 100},
	{500, 500},
	{1000, 1000},
}

func getHeatLimit(maxVal int) int {
	for _, hl := range heatLimits {
		if maxVal <= hl.threshold {
			return hl.limit
		}
	}
	return maxVal
}

func (m *gitModel) viewFiles() string {
	if len(m.info.Files) == 0 {
		return box("Hot Files", "  No data", m.width)
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
		lines = append(lines, barChart(f.File, f.Changes, limit, barW, colPrimary))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(colMuted).Render(fmt.Sprintf("  Scale: max %d", limit)))
	return box("Hot Files (most changed)", strings.Join(lines, "\n"), m.width)
}

// viewBranches 分支列表
func (m *gitModel) viewBranches() string {
	if len(m.info.Branches) == 0 {
		return box("Branches", "  No data", m.width)
	}

	var lines []string
	for _, branch := range m.info.Branches {
		if branch == m.info.Current {
			lines = append(lines, lipgloss.NewStyle().Foreground(colGreen).Bold(true).Render("  * "+branch))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(colMuted).Render("    "+branch))
		}
	}
	return box("Branches", strings.Join(lines, "\n"), m.width)
}

// viewStats 仓库统计
func (m *gitModel) viewStats() string {
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

	statStyle := lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(colMuted)

	var lines []string
	lines = append(lines, fmt.Sprintf("  %s %s",
		statStyle.Render(fmt.Sprintf("%d", totalCommits)),
		labelStyle.Render("commits")))
	lines = append(lines, fmt.Sprintf("  %s %s",
		statStyle.Render(fmt.Sprintf("%d", totalBranches)),
		labelStyle.Render("branches")))
	lines = append(lines, fmt.Sprintf("  %s %s",
		statStyle.Render(fmt.Sprintf("%d", totalFiles)),
		labelStyle.Render("files changed")))
	lines = append(lines, fmt.Sprintf("  %s %s",
		statStyle.Render(fmt.Sprintf("%d", activeDays)),
		labelStyle.Render("active days")))
	lines = append(lines, fmt.Sprintf("  %s %s",
		statStyle.Render(fmt.Sprintf("%d", totalChanges)),
		labelStyle.Render("total changes")))

	return box("Statistics", strings.Join(lines, "\n"), m.width)
}
