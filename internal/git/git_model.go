package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	spinner spinner.Model // bubbles 加载动画

	cachedInfo  Info
	cachedMtime time.Time
	cachedPath  string
}

func (m *Model) reloadFromCache() bool {
	if m.cachedPath == "" {
		return false
	}
	indexPath := filepath.Join(m.repoPath, ".git", "index")
	info, err := os.Stat(indexPath)
	if err != nil {
		return false
	}
	// 文件未变化
	if m.repoPath == m.cachedPath && info.ModTime() == m.cachedMtime {
		m.info = m.cachedInfo
		m.loaded = true
		m.loading = false
		return true
	}
	return false
}

// SetRecent 设置最近记录列表（转发给 InputModel）
func (m *Model) SetRecent(items []string) { m.input.SetRecent(items) }

func (m *Model) Init(defaultRepo string) tea.Cmd {
	if defaultRepo == "" {
		defaultRepo = "."
	}
	m.repoPath = defaultRepo
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	return tea.Batch(LoadInfoFromDirCmd(defaultRepo), m.spinner.Tick)
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

// Update 处理消息
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case InfoMsg:
		m.info = Info(msg)
		m.loaded = true
		m.loading = false
		m.cachedInfo = m.info
		m.cachedPath = m.repoPath
		indexPath := filepath.Join(m.repoPath, ".git", "index")
		if info, err := os.Stat(indexPath); err == nil {
			m.cachedMtime = info.ModTime()
		}
		m.scroll = 0
		m.input.Active = false
		m.dirListing = false

	case DirMsg:
		if len(msg.Repos) == 0 {
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
				return m, tea.Batch(LoadInfoFromDirCmd(fullPath), ui.UpdateCfgCmd("repo", fullPath), m.spinner.Tick)
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
							if m.reloadFromCache() {
								return nil
							}
							m.loading = true
							return tea.Batch(LoadInfoFromDirCmd(path), m.spinner.Tick)
						}
						return ScanDirCmd(path)
					}
					m.repoPath = filepath.Dir(path)
					if m.reloadFromCache() {
						return nil
					}
					m.loading = true
					return tea.Batch(LoadInfoFromDirCmd(filepath.Dir(path)), m.spinner.Tick)
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
		case "home":
			m.scroll = 0
		case "end":
			m.scroll = 1 << 30 // View 中有过大检测回正
		case "/":
			m.input.Prompt = "Repository path:"
			m.input.Open(m.repoPath)
		case "ctrl+r":
			if m.repoPath != "" {
				if m.reloadFromCache() {
					return m, nil
				}
				m.loading = true
				m.err = nil
				return m, tea.Batch(func() tea.Msg { return LoadInfoFromDir(m.repoPath) }, m.spinner.Tick) // 缓存未命中
			}
		}
	}
	return m, nil
}

// View 渲染视图
func (m *Model) View() string {
	cardWidth := m.width
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
		loadingContent := lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.spinner.View() + "  Loading git info...")
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

// TODO: 现在的各个模块即便过长都有长度限制，日后如果优化成长内容显示，则需要虚拟滚动进行渲染优化了
// renderVisibleSections 虚拟滚动：只渲染可见的 section
//
//nolint:unused // 预留的虚拟滚动代码，日后可能用到
func (m *Model) renderVisibleSections(cardWidth int) []string {
	// 计算每个 section 的大致行数（用于判断可见性）
	sectionLines := m.calcSectionLines()

	// 计算总行数
	totalLines := 0
	for _, lines := range sectionLines {
		totalLines += lines
	}
	// section 之间有空行分隔
	totalLines += 3 // 4个section之间3个空行

	// 计算可见行数
	top := m.height - 3
	if top < 5 {
		top = 5
	}

	// 边界检查
	if m.scroll > totalLines-top {
		m.scroll = totalLines - top
	}
	if m.scroll < 0 {
		m.scroll = 0
	}

	start := m.scroll
	end := start + top

	// 判断哪些 section 在可见范围内
	var result []string
	currentLine := 0

	// 定义 section 渲染函数
	renderFuncs := []func() string{
		m.viewCommits,
		m.viewBranches,
		m.viewFiles,
		m.viewStats,
	}

	for i, lines := range sectionLines {
		sectionStart := currentLine
		sectionEnd := currentLine + lines

		// 检查 section 是否在可见范围内
		if sectionEnd > start && sectionStart < end {
			// 需要渲染这个 section
			rendered := renderFuncs[i]()
			renderedLines := strings.Split(rendered, "\n")

			// 计算在这个 section 内需要显示的行
			sectionVisibleStart := 0
			if start > sectionStart {
				sectionVisibleStart = start - sectionStart
			}
			sectionVisibleEnd := len(renderedLines)
			if end < sectionEnd {
				sectionVisibleEnd = end - sectionStart
			}

			// 边界检查
			if sectionVisibleStart < 0 {
				sectionVisibleStart = 0
			}
			if sectionVisibleEnd > len(renderedLines) {
				sectionVisibleEnd = len(renderedLines)
			}
			if sectionVisibleStart < sectionVisibleEnd {
				result = append(result, renderedLines[sectionVisibleStart:sectionVisibleEnd]...)
			}
		}

		currentLine = sectionEnd
		// section 之间的空行
		if i < len(sectionLines)-1 {
			currentLine++
			if currentLine > start && currentLine < end && sectionEnd > start && sectionStart < end {
				result = append(result, "")
			}
		}
	}

	return result
}

// calcSectionLines 计算每个 section 的大致行数
//
//nolint:unused // 预留的虚拟滚动代码，配合 renderVisibleSections 使用
func (m *Model) calcSectionLines() []int {
	lines := make([]int, 4)

	// 1. Commits section
	// 固定开销：标题行(1) + 路径行(1) + header行(1) + 空行(1) + 表头行(1) + 分隔线(1) + 边框上下(2) + 内边距(2) = 10行
	commitFixed := 10
	commitDataLines := len(m.info.Commits)
	if commitDataLines == 0 {
		commitDataLines = 1 // 空状态提示
	}
	lines[0] = commitFixed + commitDataLines

	// 2. Branches section
	// 固定开销：标题行(1) + 边框上下(2) + 内边距(2) = 5行
	branchFixed := 5
	branchDataLines := len(m.info.Branches)
	if branchDataLines == 0 {
		branchDataLines = 1 // 空状态提示
	}
	lines[1] = branchFixed + branchDataLines

	// 3. Files section
	// 固定开销：标题行(1) + 比例行(1) + 边框上下(2) + 内边距(2) = 6行
	fileFixed := 6
	fileDataLines := len(m.info.Files)
	if fileDataLines == 0 {
		fileDataLines = 1 // 空状态提示
	}
	lines[2] = fileFixed + fileDataLines

	// 4. Stats section
	// 固定开销：标题行(1) + 统计项(5) + 边框上下(2) + 内边距(2) = 10行
	lines[3] = 10

	return lines
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

	if len(m.info.Commits) == 0 {
		emptyMsg := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  📭 仓库暂无提交记录")
		lines = append(lines, emptyMsg)
	}

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
