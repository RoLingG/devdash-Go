package linuxdo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-sixel"

	"cava_go/internal/component"
	"cava_go/internal/ui"
)

// viewMode 三级视图
type viewMode int

const (
	viewCategories viewMode = iota // 分类列表
	viewTopics                     // 帖子列表
	viewPosts                      // 帖子回复
	viewSearch                     // 搜索结果
	viewImage                      // 图片预览
)

// inputTarget 输入目标
type inputTarget int

const (
	inputCookie    inputTarget = iota // 正在输入 Cookie
	inputUserAgent                    // 正在输入 User-Agent
	inputSearch                       // 正在输入搜索关键词
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
	totalPosts        int   // 帖子总数
	postTopicID       int   // 当前话题 ID
	postStreamLoading bool  // 正在加载更多帖子
	postTitle         string
	postScroll        int      // 字符行偏移（↑↓ 按 1 行滚，驱动滚动窗口）
	postCursor        int      // 当前帖子索引（o 预览 + 黄色 ▸ 光标指示基于此）
	postLineRanges    [][2]int // 每条帖子 [起始行,结束行)，viewPosts 渲染时记录，用于行↔帖子换算
	postLoading       bool
	postErr           error

	// 搜索
	searchResults     []SearchResult
	searchCursor      int
	searchQuery       string
	searchLoading     bool
	searchErr         error
	searchPrevMode    viewMode // Esc 时返回的视图
	searchPage        int      // 当前已加载页码
	searchMore        bool     // 是否有更多结果
	searchLoadingMore bool     // 正在加载下一页

	// 输入框
	input       component.InputModel
	inputTarget inputTarget // 当前输入的是哪个字段

	// 图片预览
	imgURLs       []string  // 当前帖子的图片 URL 列表
	imgIndex      int       // 当前预览的图片索引
	imgLoading    bool      // 正在加载图片，加载中 View 显示 Loading 串，为 false 且 imgSixel 就绪时切空白占位串
	imgErr        error     // 图片加载错误
	imgSixel      []byte    // 预编码的 Sixel 字节
	imgFetchStart time.Time // 图片下载起点，下载过快时延迟展示让 Loading 提示停留约 1s

	spinner spinner.Model // bubbles 加载动画
}

// InImagePreview 当前是否处于图片预览模式（供主程序绕过布局直接全屏输出 Sixel）
func (m *Model) InImagePreview() bool {
	return m.mode == viewImage
}

func (m *Model) Init(cookie, userAgent string) tea.Cmd {
	m.cookie = cookie
	m.userAgent = userAgent
	m.mode = viewCategories
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	if cookie != "" {
		m.catLoading = true
		return tea.Batch(FetchCategoriesCmd(cookie, userAgent), m.spinner.Tick)
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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case exitImageMsg:
		// ClearScreen 已置 s.scr.clear 后才切状态，此轮 viewEquals 失效 → flush 全量重绘
		m.mode = viewPosts
		m.imgURLs = nil
		m.imgIndex = 0
		m.imgSixel = nil
		m.imgErr = nil
		m.imgLoading = false
		return m, nil

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
			// 首页则替换
			m.topics = msg.Topics
			m.topCursor = 0
			m.topErr = nil
		} else {
			// 翻页则追加（无限滚动），按 ID 去重
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
			// 预提取每篇帖子的图片 URL，供 o 键预览使用
			m.posts = make([]Post, 0, len(msg.Posts))
			for _, p := range msg.Posts {
				_, urls := HTMLToText(p.Cooked)
				p.ImageURLs = urls
				m.posts = append(m.posts, p)
			}
			m.postStream = msg.Stream
			m.totalPosts = msg.TotalPosts
			m.postTitle = msg.Title
			m.postScroll = 0
			m.postCursor = 0 // 新帖子详情，光标归位第一条
			m.postErr = nil
			// 如果有更多帖子，加载剩余的
			if len(msg.Stream) > 0 {
				m.postStreamLoading = true
				return m, tea.Batch(FetchPostStreamCmd(m.postTopicID, msg.Stream, m.cookie, m.userAgent), m.spinner.Tick)
			}
		}
		return m, nil

	case PostStreamMsg:
		if msg.Err == nil {
			// 预提取图片 URL
			for _, p := range msg.Posts {
				_, urls := HTMLToText(p.Cooked)
				p.ImageURLs = urls
				m.posts = append(m.posts, p)
			}
			// 链式加载：还有剩余则继续
			if len(msg.Remaining) > 0 {
				m.postStream = msg.Remaining
				return m, tea.Batch(FetchPostStreamCmd(msg.TopicID, msg.Remaining, m.cookie, m.userAgent), m.spinner.Tick)
			}
		}
		m.postStream = nil
		m.postStreamLoading = false
		return m, nil

	case SearchMsg:
		m.searchLoading = false
		m.searchLoadingMore = false
		if msg.Err != nil {
			m.searchErr = msg.Err
		} else {
			if msg.Page <= 1 {
				// 首页：替换结果
				m.searchResults = msg.Results
				m.searchCursor = 0
				m.searchPage = 1
			} else {
				// 后续页：追加结果，按 TopicID 去重
				existing := make(map[int]bool, len(m.searchResults))
				for _, r := range m.searchResults {
					existing[r.TopicID] = true
				}
				for _, r := range msg.Results {
					if !existing[r.TopicID] {
						m.searchResults = append(m.searchResults, r)
						existing[r.TopicID] = true
					}
				}
				m.searchPage = msg.Page
			}
			m.searchMore = msg.More
			m.searchErr = nil
		}
		return m, nil

	case ImageLoadedMsg:
		if msg.Err != nil {
			m.imgLoading = false
			m.imgErr = msg.Err
			return m, nil
		}
		if msg.Img == nil {
			m.imgLoading = false
			m.imgErr = fmt.Errorf("图片解码失败")
			return m, nil
		}
		// 先缩放到内容区像素上限，防止图片压穿 card 边框
		maxW := (m.width - 4) * cellPixelW
		maxH := (m.height - 5) * cellPixelH
		if maxW < cellPixelW {
			maxW = cellPixelW
		}
		if maxH < cellPixelH {
			maxH = cellPixelH
		}
		// Sixel 编码
		scaled := scaleImage(msg.Img, maxW, maxH)
		var buf strings.Builder
		enc := sixel.NewEncoder(&buf)
		enc.Colors = 256
		if err := enc.Encode(scaled); err != nil {
			m.imgLoading = false
			m.imgErr = err
			return m, nil
		}
		m.imgSixel = []byte(buf.String())
		m.imgErr = nil
		// 下载过快则延迟到满 minImgLoadingTime 再进入展示，让 Loading 提示停留 1s
		// imgLoading 保持 true，串不切换，渲染器比对相同跳过重绘，Loading 卡片得以保留
		if elapsed := time.Since(m.imgFetchStart); elapsed < minImgLoadingTime {
			return m, flushSixelCmd(m.imgIndex, minImgLoadingTime-elapsed)
		}
		// 已超过最小停留则直接进入展示，imgLoading 置 false 切到空白占位串
		// 本帧末渲染器重绘出足高空白 card，下一轮 showImageMsg 才叠图
		m.imgLoading = false
		return m, showImageCmd(m.imgIndex)

	case flushSixelMsg:
		// 延迟到期进入展示，切空白占位串和下一轮叠图
		// 切图/Esc 后 imgIndex 或 mode 已变，msg.Index 对不上即跳过，防旧定时器误触发
		if m.mode == viewImage && msg.Index == m.imgIndex && m.imgSixel != nil {
			m.imgLoading = false
			return m, showImageCmd(m.imgIndex)
		}
		return m, nil

	case showImageMsg:
		// card 字符层已输出完成，此时定位到内容区左上角写入 Sixel，图像呈现在边框之内
		if m.mode == viewImage && msg.Index == m.imgIndex && !m.imgLoading && m.imgSixel != nil {
			os.Stdout.Write([]byte(imgOriginSeq))
			os.Stdout.Write(m.imgSixel)
		}
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
				if m.inputTarget == inputSearch {
					// 搜索输入完成
					m.searchPrevMode = m.mode
					m.mode = viewSearch
					m.searchLoading = true
					m.searchErr = nil
					m.searchResults = nil
					m.searchCursor = 0
					m.searchQuery = val
					m.searchPage = 0
					m.searchMore = false
					m.searchLoadingMore = false
					return tea.Batch(FetchSearchCmd(val, m.cookie, m.userAgent, 1), m.spinner.Tick)
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
					m.spinner.Tick,
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
	case "ctrl+f":
		if m.cookie == "" {
			return m, nil
		}
		m.inputTarget = inputSearch
		m.input.Prompt = "Search:"
		m.input.Open("")
		return m, nil
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
	case "pgup", "ctrl+p":
		return m.movePostCursor(-1) // 上一条帖子
	case "pgdown", "ctrl+n":
		return m.movePostCursor(1) // 下一条帖子
	case "o":
		// 打开当前光标帖(postCursor)的图片预览
		// postCursor 由 ↑↓/PgUp PgDn 同步维护
		if m.mode == viewPosts && len(m.posts) > 0 {
			idx := m.postCursor
			if idx < 0 {
				idx = 0
			}
			if idx >= len(m.posts) {
				idx = len(m.posts) - 1
			}
			p := m.posts[idx]
			if len(p.ImageURLs) > 0 {
				m.imgURLs = p.ImageURLs
				m.imgIndex = 0
				m.imgLoading = true
				m.imgErr = nil
				m.imgSixel = nil
				m.imgFetchStart = time.Now()
				m.mode = viewImage
				return m, tea.Batch(FetchImageCmd(m.imgURLs[0], 0), m.spinner.Tick)
			}
		}
	case "left":
		// viewImage 中切换上一张
		if m.mode == viewImage && m.imgIndex > 0 {
			m.imgIndex--
			m.imgLoading = true
			m.imgErr = nil
			m.imgSixel = nil
			m.imgFetchStart = time.Now()
			// 切图时串变化触发全量重绘，清掉上一张的 Sixel 像素并回到加载提示
			// 新图加载完成后由二段式时序重新叠入
			return m, tea.Batch(clearScreenCmd(), FetchImageCmd(m.imgURLs[m.imgIndex], m.imgIndex), m.spinner.Tick)
		}
	case "right":
		// viewImage 中切换下一张
		if m.mode == viewImage && m.imgIndex < len(m.imgURLs)-1 {
			m.imgIndex++
			m.imgLoading = true
			m.imgErr = nil
			m.imgSixel = nil
			m.imgFetchStart = time.Now()
			// 切图时串变化触发全量重绘，清掉上一张的 Sixel 像素并回到加载提示
			// 新图加载完成后由二段式时序重新叠入
			return m, tea.Batch(clearScreenCmd(), FetchImageCmd(m.imgURLs[m.imgIndex], m.imgIndex), m.spinner.Tick)
		}
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
				return m, tea.Batch(m.fetchCurrentTopics(), m.spinner.Tick)
			}
		}
	case viewPosts:
		m.postScroll += delta
		if m.postScroll < 0 {
			m.postScroll = 0
		}
		m.syncPostCursor() // 字符行滚后同步帖子索引
	case viewSearch:
		m.searchCursor += delta
		if m.searchCursor < 0 {
			m.searchCursor = 0
		}
		if m.searchCursor >= len(m.searchResults) {
			m.searchCursor = len(m.searchResults) - 1
			// 无限滚动：到底且有更多结果，加载下一页
			if !m.searchLoadingMore && m.searchMore {
				m.searchLoadingMore = true
				return m, tea.Batch(FetchSearchCmd(m.searchQuery, m.cookie, m.userAgent, m.searchPage+1), m.spinner.Tick)
			}
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
		// pos 无上限，实际由 viewPosts 滚动裁剪处 clamp，之后同步 postCursor
		m.syncPostCursor()
	case viewSearch:
		m.searchCursor = pos
		if m.searchCursor >= len(m.searchResults) {
			m.searchCursor = len(m.searchResults) - 1
		}
		if m.searchCursor < 0 {
			m.searchCursor = 0
		}
	}
}

// syncPostCursor 按 postScroll（字符行）同步 postCursor（帖子索引）：把顶行映射到所属帖子
// postLineRanges 由 viewPosts 渲染时记录，未渲染时跳过仅 clamp
func (m *Model) syncPostCursor() {
	if len(m.posts) == 0 {
		m.postCursor = 0
		return
	}
	// 钳到帖子总数范围（PgDn 加载更多后帖子变多）
	if m.postCursor >= len(m.posts) {
		m.postCursor = len(m.posts) - 1
	}
	if m.postCursor < 0 {
		m.postCursor = 0
	}
	if len(m.postLineRanges) == 0 {
		return
	}
	if idx := m.postCursorByLine(m.postScroll); idx >= 0 {
		m.postCursor = idx
	}
}

// postCursorByLine 返回字符行 line 所属的帖子索引（postLineRanges 单调区间，二分快速匹配）
func (m *Model) postCursorByLine(line int) int {
	lo, hi := 0, len(m.postLineRanges)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		r := m.postLineRanges[mid]
		if line < r[0] {
			hi = mid - 1
		} else if line >= r[1] {
			lo = mid + 1
		} else {
			return mid
		}
	}
	return -1
}

// movePostCursor 按帖子条移动（PgUp/PgDn），postCursor ± delta 并把 postScroll 跳到目标帖起始行
func (m *Model) movePostCursor(delta int) (*Model, tea.Cmd) {
	if m.mode != viewPosts || len(m.posts) == 0 {
		return m, nil
	}
	m.postCursor += delta
	if m.postCursor < 0 {
		m.postCursor = 0
	}
	if m.postCursor >= len(m.posts) {
		m.postCursor = len(m.posts) - 1
		// 到底且还有未加载帖子 → 触发链式批量加载（下一批由 PostStreamMsg 续接）
		if !m.postStreamLoading && len(m.postStream) > 0 {
			m.postStreamLoading = true
			return m, tea.Batch(FetchPostStreamCmd(m.postTopicID, m.postStream, m.cookie, m.userAgent), m.spinner.Tick)
		}
	}
	// postScroll 跳到目标帖起始行，首屏渲染前 postLineRanges 为空时不跳（clamp 兜底）
	if m.postCursor < len(m.postLineRanges) {
		m.postScroll = m.postLineRanges[m.postCursor][0]
	}
	if m.postScroll < 0 {
		m.postScroll = 0
	}
	return m, nil
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
			return m, tea.Batch(FetchLatestTopicsCmd(0, m.cookie, m.userAgent), m.spinner.Tick)
		}
		cat := m.categories[m.catCursor-1]
		m.topTitle = cat.Name
		m.topCategory = cat.ID
		return m, tea.Batch(FetchCategoryTopicsCmd(cat.ID, 0, m.cookie, m.userAgent), m.spinner.Tick)
	case viewTopics:
		if len(m.topics) == 0 {
			return m, nil
		}
		topic := m.topics[m.topCursor]
		m.mode = viewPosts
		m.postTopicID = topic.ID
		m.postLoading = true
		m.postErr = nil
		return m, tea.Batch(FetchTopicDetailCmd(topic.ID, m.cookie, m.userAgent), m.spinner.Tick)
	case viewSearch:
		if len(m.searchResults) == 0 {
			return m, nil
		}
		result := m.searchResults[m.searchCursor]
		m.mode = viewPosts
		m.postTopicID = result.TopicID
		m.postLoading = true
		m.postErr = nil
		return m, tea.Batch(FetchTopicDetailCmd(result.TopicID, m.cookie, m.userAgent), m.spinner.Tick)
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
	case viewSearch:
		m.mode = m.searchPrevMode
		m.searchResults = nil
		m.searchQuery = ""
		m.searchErr = nil
	case viewImage:
		// 退出预览时先发 ClearScreen（只置 s.clear 标志，不立即清屏），随后 exitImageMsg 切回帖子
		// 利用 s.clear 跨 viewEquals 短路保留，切视图时触发全量重绘清掉 Sixel + 重画边框
		return m, tea.Sequence(clearScreenCmd(), exitImageCmd())
	}
	return m, nil
}

// exitImageMsg 退出图片预览的自定义消息
// 单独成一条消息，让 ClearScreen 先于状态切换被处理，利用 s.clear 跨轮保留
type exitImageMsg struct{}

// exitImageCmd 产生 exitImageMsg 的命令
func exitImageCmd() tea.Cmd {
	return func() tea.Msg { return exitImageMsg{} }
}

// clearScreenCmd 包装 tea.ClearScreen() 为 Cmd
func clearScreenCmd() tea.Cmd {
	return func() tea.Msg { return tea.ClearScreen() }
}

// minImgLoadingTime 加载提示的最小停留时长，图片下载过快时延迟展示以保证提示可读
const minImgLoadingTime = 1 * time.Second

// 终端单元格像素尺寸无法在程序内获取，按 Windows Terminal 默认字体取典型值
// 图片缩放上限由它换算，不同终端字体下显示不适配时调整这两个值即可
const (
	cellPixelW = 10
	cellPixelH = 18
)

// imgOriginSeq 把光标定位到 card 内容区左上角，前面的上边框、标题和空行占 3 行，左侧边框与 padding 占 2 列，合起来是第 4 行第 3 列
const imgOriginSeq = "\x1b[4;3H"

// flushSixelMsg 延迟展示的定时消息，携带发起时的 imgIndex 防止切图后旧定时器误触发
type flushSixelMsg struct {
	Index int
}

// flushSixelCmd 间隔 d 后发 flushSixelMsg，用于加载提示最小停留
func flushSixelCmd(index int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return flushSixelMsg{Index: index} })
}

// showImageMsg 通知叠图，此时空白占位 card 已由渲染器画好，可以放心直写 Sixel 进内容区
type showImageMsg struct {
	Index int
}

// showImageCmd 延迟发 showImageMsg，渲染器按帧率批量输出，立即发送会让图像先于
// 空白 card 的重绘落盘，其后的字符输出会遮挡图像，因此预留一个输出周期的窗口
func showImageCmd(index int) tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return showImageMsg{Index: index} })
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
		return m, tea.Batch(FetchCategoriesCmd(m.cookie, m.userAgent), m.spinner.Tick)
	case viewTopics:
		m.topLoading = true
		m.topErr = nil
		return m, tea.Batch(m.fetchCurrentTopics(), m.spinner.Tick)
	case viewPosts:
		if m.postTopicID > 0 {
			m.postLoading = true
			m.postErr = nil
			return m, tea.Batch(FetchTopicDetailCmd(m.postTopicID, m.cookie, m.userAgent), m.spinner.Tick)
		}
	case viewSearch:
		if m.searchQuery != "" {
			m.searchLoading = true
			m.searchErr = nil
			m.searchPage = 0
			m.searchMore = false
			m.searchLoadingMore = false
			return m, tea.Batch(FetchSearchCmd(m.searchQuery, m.cookie, m.userAgent, 1), m.spinner.Tick)
		}
	}
	return m, nil
}

func (m *Model) View() string {
	cardWidth := m.width
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 输入框
	if m.input.Active {
		title := "Set Cookie"
		switch m.inputTarget {
		case inputUserAgent:
			title = "Set User-Agent"
		case inputSearch:
			title = "Search"
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
	case viewSearch:
		return m.viewSearch(cardWidth)
	case viewImage:
		return m.viewImage()
	}
	return ""
}

func (m *Model) viewCategories(cardWidth int) string {
	if m.catLoading {
		return ui.Card("LinuxDo", lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.spinner.View()+"  Loading categories..."), ui.ColAccent, cardWidth)
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
		return ui.Card(m.topTitle, lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.spinner.View()+"  Loading topics..."), ui.ColAccent, cardWidth)
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
		return ui.Card(m.postTitle, lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.spinner.View()+"  Loading posts..."), ui.ColAccent, cardWidth)
	}
	if m.postErr != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.postErr.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("'R' retry, Esc back")
		return ui.Card(m.postTitle, errContent, ui.ColRed, cardWidth)
	}
	if len(m.posts) == 0 {
		return ui.Card(m.postTitle, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No posts"), ui.ColMuted, cardWidth)
	}

	// 将所有回复拼接成可滚动内容，边构建边记每帖字符行范围 postLineRanges，供 ↑↓ 行 ↔ o 帖子预览换算
	lines := make([]string, 0, 64)
	m.postLineRanges = m.postLineRanges[:0] // 每帧重建，换行随宽度变化，旧范围不可复用
	for i, p := range m.posts {
		startLen := len(lines) // 该帖起始行号

		// 当前 postCursor 帖行标识 ▸ ，其余帖 2 空格对齐
		cursorMark := "  "
		if i == m.postCursor {
			cursorMark = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸ ")
		}

		// 帖子头：显示名 + @用户名 + 头衔 + 楼号 + 时间
		displayName := p.Name
		if displayName == "" {
			displayName = p.Username
		}
		header := lipgloss.NewStyle().Foreground(ui.ColSecondary).Bold(true).Render(displayName)
		if p.Name != "" && p.Name != p.Username {
			header += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  @" + p.Username)
		}
		if p.UserTitle != "" {
			header += lipgloss.NewStyle().Foreground(ui.ColAccent).Render("  [" + p.UserTitle + "]")
		}
		header += lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("  #%d", p.PostNumber))
		if t, err := time.Parse(time.RFC3339, p.CreatedAt); err == nil {
			header += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  " + t.Format("01-02 15:04"))
		}
		lines = append(lines, cursorMark+header)

		// 内容（HTML → 纯文本）
		text, _ := HTMLToText(p.Cooked)
		for _, line := range strings.Split(text, "\n") {
			// 超长行按显示宽度换行
			for _, wl := range wrapLine(line, cardWidth-6) {
				lines = append(lines, "  "+wl)
			}
		}

		if len(p.ImageURLs) > 0 {
			lines = append(lines, "") // 图片提示前空行，与上文隔开
			lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColMuted).Render(
				fmt.Sprintf("  📷 %d张图片，按 o 预览", len(p.ImageURLs))))
			lines = append(lines, "") // 图片提示后空行，与 boosts 隔开
		}

		// boosts 短快捷回复
		for _, b := range p.Boosts {
			name := b.User.Username
			if name == "" {
				name = "匿名"
			}
			head := "    " + lipgloss.NewStyle().Foreground(ui.ColAccent).Render("⚡ ") +
				lipgloss.NewStyle().Foreground(ui.ColSecondary).Render("@"+name) + " "
			headW := lipgloss.Width(head) // head 可见宽度，忽略 ANSI 码
			boostMaxW := cardWidth - 4 - headW
			if boostMaxW < 5 {
				boostMaxW = 5
			}
			first := true
			boostText, _ := HTMLToText(b.Cooked)
			for _, line := range strings.Split(boostText, "\n") {
				for _, wl := range wrapLine(line, boostMaxW) {
					if first {
						lines = append(lines, head+lipgloss.NewStyle().Foreground(ui.ColMuted).Render(wl))
						first = false
					} else {
						lines = append(lines, strings.Repeat(" ", headW)+lipgloss.NewStyle().Foreground(ui.ColMuted).Render(wl))
					}
				}
			}
		}

		// 互动统计 + 隔断
		lines = append(lines, "\n"+lipgloss.NewStyle().Foreground(ui.ColMuted).Render(
			fmt.Sprintf("  ❤ %d    💬 %d", p.ReactionUsersCount, p.ReplyCount)))
		lines = append(lines, "\n"+lipgloss.NewStyle().Foreground(ui.ColMuted).Render(strings.Repeat("─", cardWidth-6))+"\n")

		m.postLineRanges = append(m.postLineRanges, [2]int{startLen, len(lines)})
	}

	// 滚动裁剪
	viewH := m.height - 7
	if viewH < 3 {
		viewH = 3
	}
	total := len(lines)
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
	visible := strings.Join(lines[m.postScroll:end], "\n")

	title := fmt.Sprintf("%s (%d posts)", m.postTitle, m.totalPosts)
	if m.postStreamLoading {
		// 链式批量加载更多帖子时，标题显示加载动画 + 已加载数/总数
		title = fmt.Sprintf("%s %s (%d/%d posts)", m.postTitle, m.spinner.View(), len(m.posts), m.totalPosts)
	}
	return ui.Card(title, visible, ui.ColAccent, cardWidth)
}

func (m *Model) viewSearch(cardWidth int) string {
	title := fmt.Sprintf("Search: %s", m.searchQuery)

	if m.searchLoading {
		return ui.Card(title, lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.spinner.View()+"  Searching..."), ui.ColAccent, cardWidth)
	}
	if m.searchErr != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.searchErr.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("'R' retry, Esc back")
		return ui.Card(title, errContent, ui.ColRed, cardWidth)
	}
	if len(m.searchResults) == 0 {
		return ui.Card(title, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  No results"), ui.ColMuted, cardWidth)
	}

	viewH := (m.height - 7) / 3
	if viewH < 1 {
		viewH = 1
	}

	total := len(m.searchResults)
	start := m.searchCursor - viewH/2
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
		r := m.searchResults[i]
		prefix := "  "
		if i == m.searchCursor {
			prefix = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("▸  ")
		}

		// 标题
		titleText := r.Title
		if titleText == "" {
			titleText = "(no title)"
		}
		// margin 24 = 边框+padding(4) + 前缀(3) + meta中emoji宽度(💬👀各2列) + 余量
		maxTitleW := cardWidth - 24
		if maxTitleW < 10 {
			maxTitleW = 10
		}
		if lipgloss.Width(titleText) > maxTitleW {
			titleText = ui.Truncate(titleText, maxTitleW-1) + "…"
		}
		renderedTitle := lipgloss.NewStyle().Foreground(ui.ColText).Render(titleText)

		// 元信息
		meta := lipgloss.NewStyle().Foreground(ui.ColMuted).Render(
			fmt.Sprintf("  💬 %d  🔁 %d", r.PostsCount, r.ReplyCount))

		sb.WriteString(prefix + renderedTitle + meta + "\n")

		// tags 标签（显示在标题行下方）
		tagsStr := "    🏷️ " + strings.Join(r.Tags, " ")
		maxTagsW := cardWidth - 8
		if maxTagsW > 0 && lipgloss.Width(tagsStr) > maxTagsW {
			tagsStr = ui.Truncate(tagsStr, maxTagsW)
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColMuted).Render(tagsStr) + "\n")

		// blurb 摘要
		if r.Blurb != "" {
			blurb := strings.ReplaceAll(r.Blurb, "\n", " ")
			maxBlurbW := cardWidth - 12
			if maxBlurbW < 10 {
				maxBlurbW = 10
			}
			if lipgloss.Width(blurb) > maxBlurbW {
				blurb = ui.Truncate(blurb, maxBlurbW)
			}
			sb.WriteString("    " + lipgloss.NewStyle().Foreground(ui.ColMuted).Render(blurb) + "\n")
		}
	}

	cardTitle := fmt.Sprintf("%s (%d)", title, total)
	if m.searchMore {
		cardTitle += "+"
	}
	content := strings.TrimRight(sb.String(), "\n")
	if m.searchLoadingMore {
		content += "\n\n    " + lipgloss.NewStyle().Foreground(ui.ColMuted).Render(m.spinner.View()+" Loading more...")
	}
	return ui.Card(cardTitle, content, ui.ColAccent, cardWidth)
}

// viewImage Sixel 全屏图片预览（主程序会绕过 tab/status 布局直接全屏输出）
// card 与其他模块一致固定贴满整屏，加载中显示 Loading，加载完内容区留空白行由图像层叠入
func (m *Model) viewImage() string {
	// 加载失败：返回固定错误卡片
	if m.imgErr != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.imgErr.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("Esc back, ← → navigate")
		return ui.Card("Image Preview", errContent, ui.ColRed, m.width)
	}
	title := fmt.Sprintf("Image Preview [%d/%d]", m.imgIndex+1, len(m.imgURLs))
	// 内容区固定 m.height-4 行（减上边框/标题/MarginBottom 空行/下边框），card 贴满整屏
	innerRows := m.height - 4
	if innerRows < 3 {
		innerRows = 3
	}
	var content string
	if m.imgLoading || m.imgSixel == nil {
		// 加载中内容为提示 3 行，再补空行撑满内容区
		content = lipgloss.NewStyle().Foreground(ui.ColAccent).Render("📷 Loading...") + "\n\n" +
			lipgloss.NewStyle().Foreground(ui.ColAccent).Render("Esc back, ← → navigate")
		if pad := innerRows - 3; pad > 0 {
			content += strings.Repeat("\n", pad)
		}
	} else {
		// 已加载后内容区全部留白（Repeat 尾部会多出一行，因此行数减 1），图片由图像层叠入
		content = strings.Repeat("\n", innerRows-1)
	}
	return ui.Card(title, content, ui.ColAccent, m.width)
}

// wrapLine 按显示宽度将长行切成多行（中文/emoji 计 2 列），用于回复内容完整换行
func wrapLine(s string, maxW int) []string {
	if maxW < 1 || lipgloss.Width(s) <= maxW {
		return []string{s}
	}
	var result []string
	runes := []rune(s)
	w, start := 0, 0
	for i, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > maxW {
			result = append(result, string(runes[start:i]))
			start, w = i, rw
		} else {
			w += rw
		}
	}
	if start < len(runes) {
		result = append(result, string(runes[start:]))
	}
	return result
}
