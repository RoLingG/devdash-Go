package linuxdo

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"cava_go/internal/component"
	"cava_go/internal/ui"
)

// viewMode 三级视图
type viewMode int

const (
	viewCategories viewMode = iota // 分类列表
	viewTopics                     // 帖子列表
	viewPosts                      // 帖子回复
)

// inputTarget 输入目标
type inputTarget int

const (
	inputCookie    inputTarget = iota // 正在输入 Cookie
	inputUserAgent                    // 正在输入 User-Agent
)

// Model Linux DO 模块状态
type Model struct {
	width     int
	height    int
	cookie    string
	userAgent string

	// 视图状态
	mode viewMode

	// 分类列表
	categories []Category
	catCursor  int
	catLoading bool
	catErr     error

	// 帖子列表
	topics      []Topic
	topCursor   int
	topLoading  bool
	topErr      error
	topTitle    string // 当前分类名或 "Latest"
	topCategory int    // 当前分类 ID，0 表示最新
	topPage     int    // 当前页码（从 0 开始）
	topFullPage bool   // 最近一次 API 返回了满页（可能还有下一页）

	// 帖子详情
	posts             []Post
	postStream        []int // 剩余未加载的 post ID
	totalPosts        int   // 帖子总数（API 的 posts_count）
	postTopicID       int   // 当前话题 ID（用于 /posts.json 请求）
	postStreamLoading bool  // 正在加载更多帖子
	postTitle         string
	postScroll        int
	postLoading       bool
	postErr           error

	// 输入框
	input       component.InputModel
	inputTarget inputTarget // 当前输入的是哪个字段
}

func (m *Model) Init(cookie, userAgent string) tea.Cmd {
	m.cookie = cookie
	m.userAgent = userAgent
	m.mode = viewCategories
	if cookie != "" {
		m.catLoading = true
		return FetchCategoriesCmd(cookie, userAgent)
	}
	return nil
}

func (m *Model) UpdateSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) InputActive() bool {
	return m.input.Active
}

func (m *Model) SetCookie(cookie string) {
	m.cookie = cookie
}

func (m *Model) SetUserAgent(ua string) {
	m.userAgent = ua
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CategoriesMsg:
		m.catLoading = false
		if msg.Err != nil {
			m.catErr = msg.Err
		} else {
			m.categories = msg.Categories
			m.catCursor = 0
			m.catErr = nil
		}
		return m, nil

	case TopicsMsg:
		m.topLoading = false
		if msg.Err != nil {
			m.topErr = msg.Err
		} else if m.topPage == 0 {
			// 首页：替换
			m.topics = msg.Topics
			m.topCursor = 0
			m.topErr = nil
		} else {
			// 翻页：追加（无限滚动），按 ID 去重
			existing := make(map[int]bool, len(m.topics))
			for _, t := range m.topics {
				existing[t.ID] = true
			}
			for _, t := range msg.Topics {
				if !existing[t.ID] {
					m.topics = append(m.topics, t)
				}
			}
			m.topErr = nil
		}
		m.topFullPage = msg.FullPage
		return m, nil

	case TopicDetailMsg:
		m.postLoading = false
		if msg.Err != nil {
			m.postErr = msg.Err
		} else {
			m.posts = msg.Posts
			m.postStream = msg.Stream
			m.totalPosts = msg.TotalPosts
			m.postTitle = msg.Title
			m.postScroll = 0
			m.postErr = nil
			// 如果有更多帖子，加载剩余的
			if len(msg.Stream) > 0 {
				m.postStreamLoading = true
				return m, FetchPostStreamCmd(m.postTopicID, msg.Stream, m.cookie, m.userAgent)
			}
		}
		return m, nil

	case PostStreamMsg:
		if msg.Err == nil {
			m.posts = append(m.posts, msg.Posts...)
			// 链式加载：还有剩余则继续
			if len(msg.Remaining) > 0 {
				m.postStream = msg.Remaining
				return m, FetchPostStreamCmd(msg.TopicID, msg.Remaining, m.cookie, m.userAgent)
			}
		}
		m.postStream = nil
		m.postStreamLoading = false
		return m, nil
	}

	// 输入框活跃时，所有按键交给 input 处理
	if m.input.Active {
		switch msg.(type) {
		case tea.KeyPressMsg:
			cmd := m.input.Update(msg, func(val string) func() tea.Msg {
				if val == "" {
					return nil
				}
				if m.inputTarget == inputCookie {
					// Cookie 输入完成，接着提示 User-Agent
					m.cookie = val
					m.inputTarget = inputUserAgent
					m.input.Prompt = "User-Agent:"
					m.input.Open(m.userAgent)
					return ui.UpdateCfgCmd("linuxdoCookie", val)
				}
				// User-Agent 输入完成，开始加载
				m.userAgent = val
				m.catLoading = true
				m.catErr = nil
				return tea.Batch(
					FetchCategoriesCmd(m.cookie, m.userAgent),
					ui.UpdateCfgCmd("linuxdoUserAgent", val),
				)
			})
			return m, cmd
		case tea.PasteMsg:
			return m, m.input.Update(msg, nil)
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.inputTarget = inputCookie
		m.input.Prompt = "Cookie:"
		m.input.Open(m.cookie)
		return m, nil
	case "ctrl+u":
		m.cookie = ""
		return m, nil
	case "ctrl+r":
		return m.refresh()
	case "enter":
		return m.enterSelected()
	case "esc":
		return m.goBack()
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "home":
		m.setCursor(0)
	case "end":
		m.setCursor(1 << 30)
	case "ctrl+up":
		return m.moveCursor(-10)
	case "ctrl+down":
		return m.moveCursor(10)
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) (*Model, tea.Cmd) {
	switch m.mode {
	case viewCategories:
		// 范围 0..len(categories)，0=Latest, 1+=categories
		m.catCursor += delta
		if m.catCursor < 0 {
			m.catCursor = 0
		}
		if m.catCursor > len(m.categories) {
			m.catCursor = len(m.categories)
		}
	case viewTopics:
		m.topCursor += delta
		if m.topCursor < 0 {
			m.topCursor = 0
		}
		if m.topCursor >= len(m.topics) {
			m.topCursor = len(m.topics) - 1
			// 无限滚动：到底且 API 返回过满页，说明可能还有下一页
			if !m.topLoading && m.topFullPage {
				m.topPage++
				m.topLoading = true
				m.topErr = nil
				return m, m.fetchCurrentTopics()
			}
		}
	case viewPosts:
		m.postScroll += delta
		if m.postScroll < 0 {
			m.postScroll = 0
		}
	}
	return m, nil
}

func (m *Model) setCursor(pos int) {
	switch m.mode {
	case viewCategories:
		m.catCursor = pos
		if m.catCursor > len(m.categories) {
			m.catCursor = len(m.categories)
		}
		if m.catCursor < 0 {
			m.catCursor = 0
		}
	case viewTopics:
		m.topCursor = pos
		if m.topCursor >= len(m.topics) {
			m.topCursor = len(m.topics) - 1
		}
		if m.topCursor < 0 {
			m.topCursor = 0
		}
	case viewPosts:
		m.postScroll = pos
		if m.postScroll < 0 {
			m.postScroll = 0
		}
		// 无上限，View 里会 clamp
	}
}

func (m *Model) enterSelected() (*Model, tea.Cmd) {
	switch m.mode {
	case viewCategories:
		if len(m.categories) == 0 {
			return m, nil
		}
		m.mode = viewTopics
		m.topPage = 0
		m.topCursor = 0
		m.topLoading = true
		m.topErr = nil
		m.topFullPage = false
		// cursor 0 = Latest, cursor 1+ = categories[cursor-1]
		if m.catCursor == 0 {
			m.topTitle = "Latest"
			m.topCategory = 0
			return m, FetchLatestTopicsCmd(0, m.cookie, m.userAgent)
		}
		cat := m.categories[m.catCursor-1]
		m.topTitle = cat.Name
		m.topCategory = cat.ID
		return m, FetchCategoryTopicsCmd(cat.ID, 0, m.cookie, m.userAgent)
	case viewTopics:
		if len(m.topics) == 0 {
			return m, nil
		}
		topic := m.topics[m.topCursor]
		m.mode = viewPosts
		m.postTopicID = topic.ID
		m.postLoading = true
		m.postErr = nil
		return m, FetchTopicDetailCmd(topic.ID, m.cookie, m.userAgent)
	}
	return m, nil
}

func (m *Model) goBack() (*Model, tea.Cmd) {
	switch m.mode {
	case viewPosts:
		m.mode = viewTopics
		m.posts = nil
		m.postStream = nil
		m.totalPosts = 0
		m.postTopicID = 0
		m.postStreamLoading = false
		m.postTitle = ""
	case viewTopics:
		m.mode = viewCategories
		m.topics = nil
		m.topTitle = ""
		m.topFullPage = false
	}
	return m, nil
}

func (m *Model) fetchCurrentTopics() tea.Cmd {
	if m.topCategory > 0 {
		return FetchCategoryTopicsCmd(m.topCategory, m.topPage, m.cookie, m.userAgent)
	}
	return FetchLatestTopicsCmd(m.topPage, m.cookie, m.userAgent)
}

func (m *Model) refresh() (*Model, tea.Cmd) {
	if m.cookie == "" {
		return m, nil
	}
	switch m.mode {
	case viewCategories:
		m.catLoading = true
		m.catErr = nil
		return m, FetchCategoriesCmd(m.cookie, m.userAgent)
	case viewTopics:
		m.topLoading = true
		m.topErr = nil
		return m, m.fetchCurrentTopics()
	case viewPosts:
		if m.postTopicID > 0 {
			m.postLoading = true
			m.postErr = nil
			return m, FetchTopicDetailCmd(m.postTopicID, m.cookie, m.userAgent)
		}
	}
	return m, nil
}

func (m *Model) View() string {
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 输入框
	if m.input.Active {
		title := "Set Cookie"
		if m.inputTarget == inputUserAgent {
			title = "Set User-Agent"
		}
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       title,
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	// 未设置 Cookie
	if m.cookie == "" {
		content := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to set Cookie & User-Agent") + "\n"
		content += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (Browser → F12 → Network → Request Headers)")
		return ui.Card("LinuxDo", content, ui.ColMuted, cardWidth)
	}

	switch m.mode {
	case viewCategories:
		return m.viewCategories(cardWidth)
	case viewTopics:
		return m.viewTopics(cardWidth)
	case viewPosts:
		return m.viewPosts(cardWidth)
	}
	return ""
}

func (m *Model) viewCategories(cardWidth int) string {
	if m.catLoading {
		return ui.Card("LinuxDo", lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⏳ Loading categories..."), ui.ColAccent, cardWidth)
	}
	if m.catErr != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.catErr.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("'R' retry, '/' set cookie")
		return ui.Card("LinuxDo", errContent, ui.ColRed, cardWidth)
	}
	if len(m.categories) == 0 {
		return ui.Card("LinuxDo", lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No categories"), ui.ColMuted, cardWidth)
	}

	viewH := m.height - 7
	if viewH < 3 {
		viewH = 3
	}

	// total = Latest(1) + categories
	total := len(m.categories) + 1
	start := m.catCursor - viewH/2
	if start < 0 {
		start = 0
	}
	end := start + viewH
	if end > total {
		end = total
		start = end - viewH
		if start < 0 {
			start = 0
		}
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.catCursor {
			prefix = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸  ")
		}
		if i == 0 {
			// 📌 Latest 全站最新
			name := lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true).Render("📌 Latest")
			sb.WriteString(prefix + name + "\n")
		} else {
			cat := m.categories[i-1]
			name := lipgloss.NewStyle().Foreground(ui.ColSecondary).Render(cat.Name)
			count := lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf(" (%d topics)", cat.TopicCount))
			sb.WriteString(prefix + name + count + "\n")
		}
	}

	title := fmt.Sprintf("LinuxDo (%d categories)", len(m.categories))
	return ui.Card(title, sb.String(), ui.ColAccent, cardWidth)
}

func (m *Model) viewTopics(cardWidth int) string {
	if m.topLoading {
		return ui.Card(m.topTitle, lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⏳ Loading topics..."), ui.ColAccent, cardWidth)
	}
	if m.topErr != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.topErr.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("'R' retry, Esc back")
		return ui.Card(m.topTitle, errContent, ui.ColRed, cardWidth)
	}
	if len(m.topics) == 0 {
		return ui.Card(m.topTitle, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No topics"), ui.ColMuted, cardWidth)
	}

	viewH := m.height - 7
	if viewH < 3 {
		viewH = 3
	}

	total := len(m.topics)
	start := m.topCursor - viewH/2
	if start < 0 {
		start = 0
	}
	end := start + viewH
	if end > total {
		end = total
		start = end - viewH
		if start < 0 {
			start = 0
		}
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		t := m.topics[i]
		prefix := "  "
		if i == m.topCursor {
			prefix = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸  ")
		}

		// 标题（截断到卡片宽度）
		titleText := t.Title
		maxTitleW := cardWidth - 20
		if maxTitleW < 10 {
			maxTitleW = 10
		}
		if ui.RuneLen(titleText) > maxTitleW {
			titleText = ui.Truncate(titleText, maxTitleW-1) + "…"
		}
		title := lipgloss.NewStyle().Foreground(ui.ColText).Render(titleText)

		// 元信息
		meta := lipgloss.NewStyle().Foreground(ui.ColMuted).Render(
			fmt.Sprintf("  💬 %d  👀 %d", t.PostsCount, t.Views))

		// 置顶标记
		pin := ""
		if t.Pinned {
			pin = lipgloss.NewStyle().Foreground(ui.ColAccent).Render(" 📌")
		}

		sb.WriteString(prefix + title + pin + meta + "\n")
	}

	title := fmt.Sprintf("%s (%d)", m.topTitle, total)
	return ui.Card(title, strings.TrimRight(sb.String(), "\n"), ui.ColAccent, cardWidth)
}

func (m *Model) viewPosts(cardWidth int) string {
	if m.postLoading {
		return ui.Card(m.postTitle, lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⏳ Loading posts..."), ui.ColAccent, cardWidth)
	}
	if m.postErr != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.postErr.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("'R' retry, Esc back")
		return ui.Card(m.postTitle, errContent, ui.ColRed, cardWidth)
	}
	if len(m.posts) == 0 {
		return ui.Card(m.postTitle, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No posts"), ui.ColMuted, cardWidth)
	}

	// 将所有回复拼接成可滚动内容
	var sb strings.Builder
	for _, p := range m.posts {
		// 用户名 + 时间
		header := lipgloss.NewStyle().Foreground(ui.ColSecondary).Bold(true).Render("@"+p.Username) +
			lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("  #%d", p.PostNumber))
		if t, err := time.Parse(time.RFC3339, p.CreatedAt); err == nil {
			header += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  " + t.Format("01-02 15:04"))
		}
		sb.WriteString(header + "\n")

		// 内容（HTML → 纯文本）
		text := HTMLToText(p.Cooked)
		contentLines := strings.Split(text, "\n")
		for _, line := range contentLines {
			// 长行截断
			if ui.RuneLen(line) > cardWidth-6 {
				line = ui.Truncate(line, cardWidth-9) + "..."
			}
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(ui.ColMuted).Render(strings.Repeat("─", cardWidth-6)) + "\n\n")
	}

	// 滚动裁剪
	contentLines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	viewH := m.height - 7
	if viewH < 3 {
		viewH = 3
	}
	total := len(contentLines)
	if m.postScroll > total-viewH {
		m.postScroll = total - viewH
	}
	if m.postScroll < 0 {
		m.postScroll = 0
	}
	end := m.postScroll + viewH
	if end > total {
		end = total
	}
	visible := strings.Join(contentLines[m.postScroll:end], "\n")

	title := fmt.Sprintf("%s (%d posts)", m.postTitle, m.totalPosts)
	if m.postStreamLoading {
		title = fmt.Sprintf("%s (%d/%d posts)", m.postTitle, len(m.posts), m.totalPosts)
	}
	return ui.Card(title, visible, ui.ColAccent, cardWidth)
}
