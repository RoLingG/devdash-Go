package linuxdo

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// kp 构造一个 tea.KeyPressMsg，使 msg.String() 与项目代码匹配
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
	case "backspace":
		return tea.KeyBackspace
	}
	return []rune(s)[0]
}

func newLinuxdoModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

var errLD = &ldErr{}

type ldErr struct{}

func (e *ldErr) Error() string { return "network failed" }

func TestLinuxDoInit(t *testing.T) {
	// 有 Cookie → 加载分类
	m := newLinuxdoModel()
	cmd := m.Init("token", "ua")
	if cmd == nil {
		t.Fatal("Init 有 cookie 应返回 Cmd")
	}
	if m.cookie != "token" || m.userAgent != "ua" {
		t.Errorf("cookie=%q userAgent=%q", m.cookie, m.userAgent)
	}
	if m.mode != viewCategories {
		t.Errorf("mode = %v, want viewCategories", m.mode)
	}
	if !m.catLoading {
		t.Error("Init 后 catLoading 应为 true")
	}

	// 无 Cookie → 无 Cmd
	m2 := newLinuxdoModel()
	if cmd2 := m2.Init("", "ua"); cmd2 != nil {
		t.Error("Init 无 cookie 不应返回 Cmd")
	}
}

func TestLinuxDoUpdateMsg(t *testing.T) {
	// CategoriesMsg 正常
	m := newLinuxdoModel()
	m.catLoading = true
	cats := []Category{{ID: 1, Name: "Dev"}, {ID: 2, Name: "Tech"}}
	nm, cmd := m.Update(CategoriesMsg{Categories: cats})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("CategoriesMsg 不应返回 Cmd")
	}
	if m.catLoading {
		t.Error("CategoriesMsg 后 catLoading 应为 false")
	}
	if len(m.categories) != 2 || m.categories[0].Name != "Dev" {
		t.Errorf("categories = %v", m.categories)
	}

	// CategoriesMsg 错误
	m2 := newLinuxdoModel()
	m2.Update(CategoriesMsg{Err: errLD})
	if m2.catErr == nil {
		t.Error("CategoriesMsg 带错误时应设置 catErr")
	}

	// TopicsMsg 首页替换
	m3 := newLinuxdoModel()
	m3.topPage = 0
	m3.topLoading = true
	m3.Update(TopicsMsg{Topics: []Topic{{ID: 1, Title: "a"}}, FullPage: true})
	if len(m3.topics) != 1 || m3.topics[0].Title != "a" {
		t.Errorf("topics = %v, want 首页替换", m3.topics)
	}
	if m3.topLoading {
		t.Error("TopicsMsg 后 topLoading 应为 false")
	}
	if !m3.topFullPage {
		t.Error("FullPage 应被保存")
	}

	// TopicsMsg 翻页追加 + 去重
	m4 := newLinuxdoModel()
	m4.topPage = 1
	m4.topics = []Topic{{ID: 1, Title: "a"}, {ID: 2, Title: "b"}}
	m4.Update(TopicsMsg{Topics: []Topic{{ID: 2, Title: "b"}, {ID: 3, Title: "c"}}})
	if len(m4.topics) != 3 || m4.topics[2].ID != 3 {
		t.Errorf("翻页去重后 topics = %v, want 1,2,3", m4.topics)
	}

	// TopicsMsg 错误
	m5 := newLinuxdoModel()
	m5.topLoading = true
	m5.Update(TopicsMsg{Err: errLD})
	if m5.topErr == nil {
		t.Error("TopicsMsg 带错误时应设置 topErr")
	}
}

func TestLinuxDoUpdateTopicDetail(t *testing.T) {
	// 无剩余 stream → 直接设置
	m := newLinuxdoModel()
	m.postTopicID = 5
	m.postLoading = true
	posts := []Post{{ID: 1, Username: "alice", PostNumber: 1, CreatedAt: "2026-08-01T12:00:00+08:00", Cooked: "<p>hi</p>"}}
	nm, cmd := m.Update(TopicDetailMsg{Title: "T", Posts: posts, Stream: nil, TotalPosts: 1})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd != nil {
		t.Error("无剩余 stream 时不应返回 Cmd")
	}
	if m.postLoading {
		t.Error("TopicDetailMsg 后 postLoading 应为 false")
	}
	if len(m.posts) != 1 || m.postTitle != "T" || m.totalPosts != 1 {
		t.Errorf("posts/title/total = %v/%q/%d", m.posts, m.postTitle, m.totalPosts)
	}

	// 有剩余 stream → 加载剩余帖子
	m2 := newLinuxdoModel()
	m2.postTopicID = 5
	m2.cookie = "c"
	m2.userAgent = "u"
	m2.Update(TopicDetailMsg{Posts: posts, Stream: []int{2, 3}, TotalPosts: 3})
	if !m2.postStreamLoading {
		t.Error("有剩余 stream 时应 postStreamLoading=true")
	}
	if len(m2.postStream) != 2 {
		t.Errorf("postStream = %v, want [2 3]", m2.postStream)
	}

	// 错误
	m3 := newLinuxdoModel()
	m3.postLoading = true
	m3.Update(TopicDetailMsg{Err: errLD})
	if m3.postErr == nil {
		t.Error("TopicDetailMsg 带错误时应设置 postErr")
	}
}

func TestLinuxDoUpdatePostStream(t *testing.T) {
	// 追加 + 链式加载
	m := newLinuxdoModel()
	m.posts = []Post{{ID: 1, Username: "a"}}
	m.postStream = []int{2, 3, 4}
	m.postStreamLoading = true
	nm, cmd := m.Update(PostStreamMsg{TopicID: 5, Posts: []Post{{ID: 2, Username: "b"}}, Remaining: []int{3, 4}})
	if nm != m {
		t.Error("Update 应返回同一 Model 指针")
	}
	if cmd == nil {
		t.Error("有 Remaining 时应返回链式加载 Cmd")
	}
	if len(m.posts) != 2 {
		t.Errorf("posts = %d, want 2", len(m.posts))
	}
	if len(m.postStream) != 2 || m.postStream[0] != 3 {
		t.Errorf("postStream = %v, want [3 4]", m.postStream)
	}

	// 无剩余 → 结束加载
	m2 := newLinuxdoModel()
	m2.posts = []Post{{ID: 1}}
	m2.postStreamLoading = true
	_, cmd2 := m2.Update(PostStreamMsg{TopicID: 5, Posts: []Post{{ID: 2}}, Remaining: nil})
	if cmd2 != nil {
		t.Error("无 Remaining 时不应返回 Cmd")
	}
	if m2.postStreamLoading {
		t.Error("无 Remaining 时 postStreamLoading 应为 false")
	}
	if m2.postStream != nil {
		t.Error("无 Remaining 时 postStream 应为 nil")
	}

	// 错误 → 结束加载
	m3 := newLinuxdoModel()
	m3.postStreamLoading = true
	_, cmd3 := m3.Update(PostStreamMsg{TopicID: 5, Err: errLD})
	if cmd3 != nil {
		t.Error("错误时不应返回 Cmd")
	}
	if m3.postStreamLoading {
		t.Error("错误时 postStreamLoading 应为 false")
	}
}

func TestLinuxDoUpdateSearch(t *testing.T) {
	// 首页（Page<=1）替换
	m := newLinuxdoModel()
	m.searchLoading = true
	m.Update(SearchMsg{Results: []SearchResult{{TopicID: 1, Title: "a"}}, Page: 1, More: true})
	if m.searchLoading {
		t.Error("SearchMsg 后 searchLoading 应为 false")
	}
	if len(m.searchResults) != 1 || m.searchResults[0].Title != "a" {
		t.Errorf("searchResults = %v", m.searchResults)
	}
	if m.searchPage != 1 {
		t.Errorf("searchPage = %d, want 1", m.searchPage)
	}
	if !m.searchMore {
		t.Error("More 应被保存")
	}

	// 后续页追加 + 去重
	m2 := newLinuxdoModel()
	m2.searchResults = []SearchResult{{TopicID: 1}}
	m2.Update(SearchMsg{Results: []SearchResult{{TopicID: 1}, {TopicID: 2}}, Page: 2, More: false})
	if len(m2.searchResults) != 2 || m2.searchResults[1].TopicID != 2 {
		t.Errorf("翻页去重后 searchResults = %v, want 1,2", m2.searchResults)
	}
	if m2.searchPage != 2 {
		t.Errorf("searchPage = %d, want 2", m2.searchPage)
	}
	if m2.searchMore {
		t.Error("More=false 应被保存")
	}

	// 错误
	m3 := newLinuxdoModel()
	m3.Update(SearchMsg{Err: errLD})
	if m3.searchErr == nil {
		t.Error("SearchMsg 带错误时应设置 searchErr")
	}
}

func TestLinuxDoInputActive(t *testing.T) {
	// 输入框活跃时按键转发给 input
	m := newLinuxdoModel()
	m.input.Active = true
	m.inputTarget = inputSearch
	m.input.Open("")
	m.Update(kp("a"))
	m.Update(kp("b"))
	if m.input.Value != "ab" {
		t.Errorf("input.Value = %q, want ab", m.input.Value)
	}
	if m.input.Cursor != 2 {
		t.Errorf("input.Cursor = %d, want 2", m.input.Cursor)
	}

	// PasteMsg 转发
	m2 := newLinuxdoModel()
	m2.input.Active = true
	m2.inputTarget = inputCookie
	m2.input.Open("")
	m2.Update(tea.PasteMsg{Content: "paste-token"})
	if m2.input.Value != "paste-token" {
		t.Errorf("input.Value = %q, want paste-token", m2.input.Value)
	}
}

func TestLinuxDoInputComplete(t *testing.T) {
	// Cookie 输入完成 → 提示 User-Agent
	m := newLinuxdoModel()
	m.input.Active = true
	m.inputTarget = inputCookie
	m.input.Prompt = "Cookie:"
	m.input.Open("")
	m.input.Value = "mytoken"
	m.input.Cursor = 7
	_, cmd := m.Update(kp("enter"))
	if cmd == nil {
		t.Error("Cookie 提交应返回 UpdateCfgCmd")
	}
	if m.cookie != "mytoken" {
		t.Errorf("cookie = %q, want mytoken", m.cookie)
	}
	if m.inputTarget != inputUserAgent {
		t.Errorf("inputTarget = %v, want inputUserAgent", m.inputTarget)
	}
	if m.input.Prompt != "User-Agent:" {
		t.Errorf("Prompt = %q, want User-Agent:", m.input.Prompt)
	}
	if !m.input.Active {
		t.Error("Cookie 后应继续打开 User-Agent 输入")
	}

	// User-Agent 输入完成 → 加载分类
	m2 := newLinuxdoModel()
	m2.cookie = "mytoken"
	m2.input.Active = true
	m2.inputTarget = inputUserAgent
	m2.input.Prompt = "User-Agent:"
	m2.input.Open("")
	m2.input.Value = "MyUA"
	m2.input.Cursor = 4
	_, cmd2 := m2.Update(kp("enter"))
	if cmd2 == nil {
		t.Error("UA 提交应返回 Batch Cmd")
	}
	if m2.userAgent != "MyUA" {
		t.Errorf("userAgent = %q, want MyUA", m2.userAgent)
	}
	if !m2.catLoading {
		t.Error("UA 提交后 catLoading 应为 true")
	}

	// 搜索输入完成 → 进入搜索视图
	m3 := newLinuxdoModel()
	m3.cookie = "c"
	m3.mode = viewTopics
	m3.input.Active = true
	m3.inputTarget = inputSearch
	m3.input.Prompt = "Search:"
	m3.input.Open("")
	m3.input.Value = "golang"
	m3.input.Cursor = 6
	_, cmd3 := m3.Update(kp("enter"))
	if cmd3 == nil {
		t.Error("搜索提交应返回 FetchSearchCmd")
	}
	if m3.mode != viewSearch {
		t.Errorf("mode = %v, want viewSearch", m3.mode)
	}
	if m3.searchQuery != "golang" {
		t.Errorf("searchQuery = %q, want golang", m3.searchQuery)
	}
	if !m3.searchLoading {
		t.Error("搜索提交后 searchLoading 应为 true")
	}
	if m3.searchPrevMode != viewTopics {
		t.Errorf("searchPrevMode = %v, want viewTopics", m3.searchPrevMode)
	}

	// 空值提交不产生 Cmd
	m4 := newLinuxdoModel()
	m4.input.Active = true
	m4.inputTarget = inputSearch
	m4.input.Open("")
	_, cmd4 := m4.Update(kp("enter"))
	if cmd4 != nil {
		t.Error("空值提交不应返回 Cmd")
	}
}

func TestLinuxDoHandleKey(t *testing.T) {
	// ctrl+f 无 cookie → 忽略
	m := newLinuxdoModel()
	_, cmd := m.Update(kp("ctrl+f"))
	if cmd != nil {
		t.Error("无 cookie 时 ctrl+f 不应返回 Cmd")
	}
	if m.input.Active {
		t.Error("无 cookie 时 ctrl+f 不应打开输入框")
	}

	// ctrl+f 有 cookie → 搜索输入
	m2 := newLinuxdoModel()
	m2.cookie = "c"
	_, cmd2 := m2.Update(kp("ctrl+f"))
	if cmd2 != nil {
		t.Error("ctrl+f 不应返回 Cmd")
	}
	if !m2.input.Active || m2.inputTarget != inputSearch {
		t.Error("ctrl+f 应打开搜索输入框")
	}
	if m2.input.Prompt != "Search:" {
		t.Errorf("Prompt = %q, want Search:", m2.input.Prompt)
	}

	// / 打开 Cookie 输入
	m3 := newLinuxdoModel()
	m3.cookie = "old"
	_, cmd3 := m3.Update(kp("/"))
	if cmd3 != nil {
		t.Error("按 / 不应返回 Cmd")
	}
	if !m3.input.Active || m3.inputTarget != inputCookie {
		t.Error("按 / 应打开 Cookie 输入框")
	}
	if m3.input.Prompt != "Cookie:" {
		t.Errorf("Prompt = %q, want Cookie:", m3.input.Prompt)
	}
	if m3.input.Value != "old" {
		t.Errorf("Cookie 输入应预填 = %q", m3.input.Value)
	}

	// ctrl+u 清空 cookie
	m4 := newLinuxdoModel()
	m4.cookie = "x"
	_, cmd4 := m4.Update(kp("ctrl+u"))
	if cmd4 != nil {
		t.Error("ctrl+u 不应返回 Cmd")
	}
	if m4.cookie != "" {
		t.Errorf("ctrl+u 后 cookie = %q, want empty", m4.cookie)
	}
}

func TestLinuxDoMoveCursorCategories(t *testing.T) {
	m := newLinuxdoModel()
	m.mode = viewCategories
	m.categories = []Category{{ID: 1}, {ID: 2}, {ID: 3}}
	m.catCursor = 1

	m.Update(kp("down"))
	if m.catCursor != 2 {
		t.Errorf("down 后 catCursor = %d, want 2", m.catCursor)
	}
	m.Update(kp("down"))
	if m.catCursor != 3 {
		t.Errorf("down 后 catCursor = %d, want 3（上限=len）", m.catCursor)
	}
	m.Update(kp("down"))
	if m.catCursor != 3 {
		t.Errorf("越界 down 后 catCursor = %d, want 3", m.catCursor)
	}
	m.Update(kp("up"))
	if m.catCursor != 2 {
		t.Errorf("up 后 catCursor = %d, want 2", m.catCursor)
	}

	// home / end / ctrl+up / ctrl+down
	m.Update(kp("home"))
	if m.catCursor != 0 {
		t.Errorf("home 后 catCursor = %d, want 0", m.catCursor)
	}
	m.Update(kp("end"))
	if m.catCursor != 3 {
		t.Errorf("end 后 catCursor = %d, want 3", m.catCursor)
	}
	m.Update(kp("ctrl+down"))
	if m.catCursor != 3 {
		t.Errorf("ctrl+down 后 catCursor = %d, want 3", m.catCursor)
	}
	m.Update(kp("ctrl+up"))
	if m.catCursor != 0 {
		t.Errorf("ctrl+up 后 catCursor = %d, want 0", m.catCursor)
	}

	// k / j 别名
	m.Update(kp("j"))
	if m.catCursor != 1 {
		t.Errorf("j 后 catCursor = %d, want 1", m.catCursor)
	}
	m.Update(kp("k"))
	if m.catCursor != 0 {
		t.Errorf("k 后 catCursor = %d, want 0", m.catCursor)
	}
}

func TestLinuxDoMoveCursorTopicsScroll(t *testing.T) {
	// 到底且满页 → 触发无限滚动
	m := newLinuxdoModel()
	m.mode = viewTopics
	m.topics = []Topic{{ID: 1}, {ID: 2}, {ID: 3}}
	m.topCursor = 2
	m.topFullPage = true
	m.topLoading = false
	m.topPage = 0
	_, cmd := m.Update(kp("down"))
	if cmd == nil {
		t.Fatal("满页到底 down 应返回下一页 Cmd")
	}
	if m.topPage != 1 {
		t.Errorf("topPage = %d, want 1", m.topPage)
	}
	if !m.topLoading {
		t.Error("无限滚动后 topLoading 应为 true")
	}

	// 未满页 → 不加载下一页
	m2 := newLinuxdoModel()
	m2.mode = viewTopics
	m2.topics = []Topic{{ID: 1}, {ID: 2}, {ID: 3}}
	m2.topCursor = 2
	m2.topFullPage = false
	_, cmd2 := m2.Update(kp("down"))
	if cmd2 != nil {
		t.Error("未满页到底 down 不应返回 Cmd")
	}
}

func TestLinuxDoMoveCursorSearchScroll(t *testing.T) {
	// 到底且有更多 → 加载下一页
	m := newLinuxdoModel()
	m.mode = viewSearch
	m.searchResults = []SearchResult{{TopicID: 1}, {TopicID: 2}}
	m.searchCursor = 1
	m.searchMore = true
	m.searchLoadingMore = false
	m.searchPage = 1
	_, cmd := m.Update(kp("down"))
	if cmd == nil {
		t.Fatal("有更多结果到底 down 应返回下一页 Cmd")
	}
	if !m.searchLoadingMore {
		t.Error("无限滚动后 searchLoadingMore 应为 true")
	}

	// 无更多 → 不加载
	m2 := newLinuxdoModel()
	m2.mode = viewSearch
	m2.searchResults = []SearchResult{{TopicID: 1}}
	m2.searchCursor = 0
	m2.searchMore = false
	_, cmd2 := m2.Update(kp("down"))
	if cmd2 != nil {
		t.Error("无更多结果到底 down 不应返回 Cmd")
	}
}

func TestLinuxDoMoveCursorPosts(t *testing.T) {
	m := newLinuxdoModel()
	m.mode = viewPosts
	m.postScroll = 5
	m.Update(kp("up"))
	if m.postScroll != 4 {
		t.Errorf("up 后 postScroll = %d, want 4", m.postScroll)
	}
	m.Update(kp("down"))
	if m.postScroll != 5 {
		t.Errorf("down 后 postScroll = %d, want 5", m.postScroll)
	}
	m.Update(kp("home"))
	if m.postScroll != 0 {
		t.Errorf("home 后 postScroll = %d, want 0", m.postScroll)
	}
	m.Update(kp("end"))
	if m.postScroll != 1<<30 {
		t.Errorf("end 后 postScroll = %d, want 1<<30", m.postScroll)
	}
}

func TestLinuxDoEnterSelected(t *testing.T) {
	// Latest
	m := newLinuxdoModel()
	m.categories = []Category{{ID: 1, Name: "Dev"}}
	m.mode = viewCategories
	m.catCursor = 0
	_, cmd := m.Update(kp("enter"))
	if cmd == nil {
		t.Fatal("Latest enter 应返回 Cmd")
	}
	if m.mode != viewTopics || m.topTitle != "Latest" || m.topCategory != 0 {
		t.Errorf("mode/title/category = %v/%q/%d", m.mode, m.topTitle, m.topCategory)
	}
	if !m.topLoading {
		t.Error("enter 后 topLoading 应为 true")
	}

	// 具体分类
	m2 := newLinuxdoModel()
	m2.categories = []Category{{ID: 1, Name: "Dev"}}
	m2.mode = viewCategories
	m2.catCursor = 1
	_, cmd2 := m2.Update(kp("enter"))
	if cmd2 == nil {
		t.Fatal("分类 enter 应返回 Cmd")
	}
	if m2.topTitle != "Dev" || m2.topCategory != 1 {
		t.Errorf("topTitle/topCategory = %q/%d, want Dev/1", m2.topTitle, m2.topCategory)
	}

	// 空分类
	m3 := newLinuxdoModel()
	m3.mode = viewCategories
	if _, cmd3 := m3.Update(kp("enter")); cmd3 != nil {
		t.Error("空分类 enter 不应返回 Cmd")
	}

	// topics → posts
	m4 := newLinuxdoModel()
	m4.mode = viewTopics
	m4.topics = []Topic{{ID: 5, Title: "T"}}
	_, cmd4 := m4.Update(kp("enter"))
	if cmd4 == nil {
		t.Fatal("topic enter 应返回 Cmd")
	}
	if m4.mode != viewPosts || m4.postTopicID != 5 {
		t.Errorf("mode/postTopicID = %v/%d", m4.mode, m4.postTopicID)
	}
	if !m4.postLoading {
		t.Error("topic enter 后 postLoading 应为 true")
	}

	// search → posts
	m5 := newLinuxdoModel()
	m5.mode = viewSearch
	m5.searchResults = []SearchResult{{TopicID: 7}}
	_, cmd5 := m5.Update(kp("enter"))
	if cmd5 == nil {
		t.Fatal("search enter 应返回 Cmd")
	}
	if m5.mode != viewPosts || m5.postTopicID != 7 {
		t.Errorf("mode/postTopicID = %v/%d", m5.mode, m5.postTopicID)
	}

	// 空 search
	m6 := newLinuxdoModel()
	m6.mode = viewSearch
	if _, cmd6 := m6.Update(kp("enter")); cmd6 != nil {
		t.Error("空 search enter 不应返回 Cmd")
	}
}

func TestLinuxDoGoBack(t *testing.T) {
	// posts → topics
	m := newLinuxdoModel()
	m.mode = viewPosts
	m.posts = []Post{{ID: 1}}
	m.postStream = []int{2}
	m.totalPosts = 3
	m.postTopicID = 1
	m.postStreamLoading = true
	m.postTitle = "T"
	_, cmd := m.Update(kp("esc"))
	if cmd != nil {
		t.Error("esc 不应返回 Cmd")
	}
	if m.mode != viewTopics {
		t.Errorf("posts esc 后 mode = %v, want viewTopics", m.mode)
	}
	if m.posts != nil || m.postStream != nil || m.totalPosts != 0 || m.postTopicID != 0 || m.postStreamLoading {
		t.Error("posts esc 后应清空帖子状态")
	}

	// topics → categories
	m2 := newLinuxdoModel()
	m2.mode = viewTopics
	m2.topics = []Topic{{ID: 1}}
	m2.topTitle = "T"
	m2.topFullPage = true
	m2.Update(kp("esc"))
	if m2.mode != viewCategories {
		t.Errorf("topics esc 后 mode = %v, want viewCategories", m2.mode)
	}
	if m2.topics != nil || m2.topTitle != "" || m2.topFullPage {
		t.Error("topics esc 后应清空帖子列表状态")
	}

	// search → prevMode
	m3 := newLinuxdoModel()
	m3.mode = viewSearch
	m3.searchPrevMode = viewTopics
	m3.searchResults = []SearchResult{{TopicID: 1}}
	m3.searchQuery = "q"
	m3.searchErr = errLD
	m3.Update(kp("esc"))
	if m3.mode != viewTopics {
		t.Errorf("search esc 后 mode = %v, want viewTopics", m3.mode)
	}
	if m3.searchResults != nil || m3.searchQuery != "" || m3.searchErr != nil {
		t.Error("search esc 后应清空搜索状态")
	}
}

func TestLinuxDoRefresh(t *testing.T) {
	// 无 cookie → 无 Cmd
	m := newLinuxdoModel()
	if _, cmd := m.Update(kp("ctrl+r")); cmd != nil {
		t.Error("无 cookie 时 ctrl+r 不应返回 Cmd")
	}

	// categories
	m2 := newLinuxdoModel()
	m2.cookie = "c"
	_, cmd2 := m2.Update(kp("ctrl+r"))
	if cmd2 == nil {
		t.Error("categories ctrl+r 应返回 Cmd")
	}
	if !m2.catLoading {
		t.Error("categories ctrl+r 后 catLoading 应为 true")
	}

	// topics（有分类 → 分类接口）
	m3 := newLinuxdoModel()
	m3.cookie = "c"
	m3.mode = viewTopics
	m3.topCategory = 2
	m3.topPage = 1
	_, cmd3 := m3.Update(kp("ctrl+r"))
	if cmd3 == nil {
		t.Error("topics ctrl+r 应返回 Cmd")
	}
	if !m3.topLoading {
		t.Error("topics ctrl+r 后 topLoading 应为 true")
	}

	// posts
	m4 := newLinuxdoModel()
	m4.cookie = "c"
	m4.mode = viewPosts
	m4.postTopicID = 9
	_, cmd4 := m4.Update(kp("ctrl+r"))
	if cmd4 == nil {
		t.Error("posts ctrl+r 应返回 Cmd")
	}
	if !m4.postLoading {
		t.Error("posts ctrl+r 后 postLoading 应为 true")
	}

	// posts 无 topicID → 无 Cmd
	m5 := newLinuxdoModel()
	m5.cookie = "c"
	m5.mode = viewPosts
	if _, cmd5 := m5.Update(kp("ctrl+r")); cmd5 != nil {
		t.Error("posts 无 topicID 时 ctrl+r 不应返回 Cmd")
	}

	// search
	m6 := newLinuxdoModel()
	m6.cookie = "c"
	m6.mode = viewSearch
	m6.searchQuery = "q"
	_, cmd6 := m6.Update(kp("ctrl+r"))
	if cmd6 == nil {
		t.Error("search ctrl+r 应返回 Cmd")
	}
	if !m6.searchLoading {
		t.Error("search ctrl+r 后 searchLoading 应为 true")
	}
}

func TestLinuxDoViewStates(t *testing.T) {
	// 输入框模式
	m := newLinuxdoModel()
	m.input.Active = true
	m.inputTarget = inputCookie
	m.input.Prompt = "Cookie:"
	m.input.Value = "x"
	m.input.Cursor = 1
	if v := m.View(); !strings.Contains(v, "Set Cookie") {
		t.Errorf("cookie 输入视图缺少标题: %q", v)
	}

	m2 := newLinuxdoModel()
	m2.input.Active = true
	m2.inputTarget = inputUserAgent
	m2.input.Prompt = "User-Agent:"
	if v := m2.View(); !strings.Contains(v, "Set User-Agent") {
		t.Errorf("UA 输入视图缺少标题: %q", v)
	}

	m3 := newLinuxdoModel()
	m3.input.Active = true
	m3.inputTarget = inputSearch
	m3.input.Prompt = "Search:"
	if v := m3.View(); !strings.Contains(v, "Search") {
		t.Errorf("搜索输入视图缺少标题: %q", v)
	}

	// 未设置 Cookie
	m4 := newLinuxdoModel()
	if v := m4.View(); !strings.Contains(v, "Press '/' to set Cookie") {
		t.Errorf("未设置 Cookie 视图缺少提示: %q", v)
	}

	// 分类加载中
	m5 := newLinuxdoModel()
	m5.cookie = "c"
	m5.catLoading = true
	if v := m5.View(); !strings.Contains(v, "Loading categories") {
		t.Errorf("分类加载视图缺少内容: %q", v)
	}

	// 分类错误
	m6 := newLinuxdoModel()
	m6.cookie = "c"
	m6.catErr = errLD
	if v := m6.View(); !strings.Contains(v, "network failed") {
		t.Errorf("分类错误视图缺少错误: %q", v)
	}

	// 分类为空
	m7 := newLinuxdoModel()
	m7.cookie = "c"
	if v := m7.View(); !strings.Contains(v, "No categories") {
		t.Errorf("空分类视图缺少提示: %q", v)
	}

	// 帖子列表加载中
	m8 := newLinuxdoModel()
	m8.cookie = "c"
	m8.mode = viewTopics
	m8.topTitle = "Dev"
	m8.topLoading = true
	if v := m8.View(); !strings.Contains(v, "Loading topics") {
		t.Errorf("帖子加载视图缺少内容: %q", v)
	}

	// 帖子列表为空
	m9 := newLinuxdoModel()
	m9.cookie = "c"
	m9.mode = viewTopics
	m9.topTitle = "Dev"
	if v := m9.View(); !strings.Contains(v, "No topics") {
		t.Errorf("空帖子视图缺少提示: %q", v)
	}

	// 帖子详情加载中
	m10 := newLinuxdoModel()
	m10.cookie = "c"
	m10.mode = viewPosts
	m10.postTitle = "T"
	m10.postLoading = true
	if v := m10.View(); !strings.Contains(v, "Loading posts") {
		t.Errorf("详情加载视图缺少内容: %q", v)
	}

	// 搜索加载中
	m11 := newLinuxdoModel()
	m11.cookie = "c"
	m11.mode = viewSearch
	m11.searchQuery = "go"
	m11.searchLoading = true
	if v := m11.View(); !strings.Contains(v, "Searching") {
		t.Errorf("搜索加载视图缺少内容: %q", v)
	}

	// 搜索为空
	m12 := newLinuxdoModel()
	m12.cookie = "c"
	m12.mode = viewSearch
	m12.searchQuery = "go"
	if v := m12.View(); !strings.Contains(v, "No results") {
		t.Errorf("空搜索结果视图缺少提示: %q", v)
	}
}

func TestLinuxDoViewCategories(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.categories = []Category{{ID: 1, Name: "Dev", TopicCount: 12}, {ID: 2, Name: "Tech", TopicCount: 5}}
	m.catCursor = 1
	v := m.View()
	for _, want := range []string{"LinuxDo", "Latest", "Dev", "Tech", "(12 topics)"} {
		if !strings.Contains(v, want) {
			t.Errorf("分类视图缺少 %q: %q", want, v)
		}
	}
}

func TestLinuxDoViewTopics(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.mode = viewTopics
	m.topTitle = "Dev"
	m.topics = []Topic{
		{ID: 1, Title: "Hello World", PostsCount: 3, Views: 10, Pinned: true},
		{ID: 2, Title: "Go Tips", PostsCount: 7, Views: 99},
	}
	m.topCursor = 0
	v := m.View()
	for _, want := range []string{"Dev", "Hello World", "Go Tips", "💬 3", "👀 10", "📌"} {
		if !strings.Contains(v, want) {
			t.Errorf("帖子视图缺少 %q: %q", want, v)
		}
	}
}

func TestLinuxDoViewPosts(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.mode = viewPosts
	m.postTitle = "Topic Title"
	m.totalPosts = 3
	m.posts = []Post{
		// 设 Name(与 Username 不同)以触发 @username 渲染分支，测试断言 @alice 才成立
		{ID: 1, Name: "Alice", Username: "alice", PostNumber: 1, CreatedAt: "2026-08-01T12:00:00+08:00", Cooked: "<p>Hello there</p>"},
	}
	v := m.View()
	for _, want := range []string{"Topic Title", "(3 posts)", "@alice", "#1", "Hello there"} {
		if !strings.Contains(v, want) {
			t.Errorf("帖子详情视图缺少 %q: %q", want, v)
		}
	}

	// 空帖子
	m2 := newLinuxdoModel()
	m2.cookie = "c"
	m2.mode = viewPosts
	m2.postTitle = "T"
	if v := m2.View(); !strings.Contains(v, "No posts") {
		t.Errorf("空帖子详情视图缺少提示: %q", v)
	}

	// 无效时间不 panic
	m3 := newLinuxdoModel()
	m3.cookie = "c"
	m3.mode = viewPosts
	m3.postTitle = "T"
	m3.posts = []Post{{ID: 1, Username: "bob", PostNumber: 1, CreatedAt: "bad-time", Cooked: "plain"}}
	if v := m3.View(); v == "" {
		t.Error("无效时间帖子详情视图为空")
	}
}

func TestLinuxDoViewSearch(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.mode = viewSearch
	m.searchQuery = "golang"
	m.searchResults = []SearchResult{
		{TopicID: 1, Title: "How to Go", Blurb: "golang blurb text", PostsCount: 2, ReplyCount: 1, Tags: []string{"go", "dev"}},
	}
	m.searchCursor = 0
	v := m.View()
	for _, want := range []string{"Search: golang", "How to Go", "💬 2", "🔁 1", "🏷️ go dev", "golang blurb text"} {
		if !strings.Contains(v, want) {
			t.Errorf("搜索结果视图缺少 %q: %q", want, v)
		}
	}

	// searchMore + Loading more
	m2 := newLinuxdoModel()
	m2.cookie = "c"
	m2.mode = viewSearch
	m2.searchQuery = "q"
	m2.searchMore = true
	m2.searchLoadingMore = true
	m2.searchResults = []SearchResult{{TopicID: 1, Title: "T", Tags: []string{}}}
	if v := m2.View(); !strings.Contains(v, "Loading more") {
		t.Errorf("加载更多提示缺失: %q", v)
	}

	// 空标题 fallback
	m3 := newLinuxdoModel()
	m3.cookie = "c"
	m3.mode = viewSearch
	m3.searchQuery = "q"
	m3.searchResults = []SearchResult{{TopicID: 1}}
	if v := m3.View(); !strings.Contains(v, "(no title)") {
		t.Errorf("空标题 fallback 缺失: %q", v)
	}
}

// TestLinuxDoPostLineRanges 验证 viewPosts 渲染后 postLineRanges 被正确填充，
// 且 postCursorByLine 能把字符行号映射回帖子索引（含隔断行归属上帖）。
func TestLinuxDoPostLineRanges(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.mode = viewPosts
	m.postTitle = "T"
	m.posts = []Post{
		{ID: 1, Name: "A", Username: "a", PostNumber: 1, CreatedAt: "2026-08-01T12:00:00+08:00", Cooked: "<p>line1</p>"},
		{ID: 2, Name: "B", Username: "b", PostNumber: 2, CreatedAt: "2026-08-01T12:01:00+08:00", Cooked: "<p>line2</p>"},
	}
	_ = m.View() // 触发 viewPosts 渲染，填充 postLineRanges

	if len(m.postLineRanges) != len(m.posts) {
		t.Fatalf("postLineRanges 长度 = %d, want %d", len(m.postLineRanges), len(m.posts))
	}
	// 首帖必须从第 0 行开始
	if m.postLineRanges[0][0] != 0 {
		t.Errorf("首帖起始行 = %d, want 0", m.postLineRanges[0][0])
	}
	// 区间单调递增且首尾相接
	for i := 1; i < len(m.postLineRanges); i++ {
		if m.postLineRanges[i][0] != m.postLineRanges[i-1][1] {
			t.Errorf("帖子 %d 起始行 %d 与上帖结束行 %d 不衔接", i, m.postLineRanges[i][0], m.postLineRanges[i-1][1])
		}
		if m.postLineRanges[i][1] <= m.postLineRanges[i][0] {
			t.Errorf("帖子 %d 区间非正长度: %v", i, m.postLineRanges[i])
		}
	}

	// postCursorByLine: 每帖内部任意行 → 该帖索引
	for i, r := range m.postLineRanges {
		for line := r[0]; line < r[1]; line++ {
			if got := m.postCursorByLine(line); got != i {
				t.Errorf("postCursorByLine(%d) = %d, want %d", line, got, i)
			}
		}
	}
	// 越界行（总行数）返回 -1
	if m.postCursorByLine(m.postLineRanges[len(m.postLineRanges)-1][1]) != -1 {
		t.Error("超出最后一帖的行号应返回 -1")
	}
}

// TestLinuxDoSyncPostCursor 验证 ↑↓ 字符行滚动后，postCursor 跟着行号所属帖子走。
func TestLinuxDoSyncPostCursor(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.mode = viewPosts
	m.postTitle = "T"
	m.posts = []Post{
		{ID: 1, Name: "A", Username: "a", PostNumber: 1, CreatedAt: "2026-08-01T12:00:00+08:00", Cooked: "<p>first</p>"},
		{ID: 2, Name: "B", Username: "b", PostNumber: 2, CreatedAt: "2026-08-01T12:01:00+08:00", Cooked: "<p>second</p>"},
	}
	_ = m.View()
	if len(m.postLineRanges) < 2 {
		t.Fatal("需至少 2 帖以测同步")
	}
	// 把光标滚到第二帖起始行，syncPostCursor 应把 postCursor 指向 1
	m.postScroll = m.postLineRanges[1][0]
	m.syncPostCursor()
	if m.postCursor != 1 {
		t.Errorf("滚到第二帖起始行后 postCursor = %d, want 1", m.postCursor)
	}
	// 滚回第一帖内部某行
	m.postScroll = m.postLineRanges[0][0]
	m.syncPostCursor()
	if m.postCursor != 0 {
		t.Errorf("滚回第一帖后 postCursor = %d, want 0", m.postCursor)
	}
}

// TestLinuxDoMovePostCursorLoadMore 验证 PgDn 到底且还有未加载帖子时触发链式批量加载。
func TestLinuxDoMovePostCursorLoadMore(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.userAgent = "u"
	m.mode = viewPosts
	m.postTitle = "T"
	m.postTopicID = 5
	m.posts = []Post{{ID: 1, Name: "A", Username: "a", PostNumber: 1, CreatedAt: "2026-08-01T12:00:00+08:00", Cooked: "<p>x</p>"}}
	m.postStream = []int{2, 3} // 还有 2 条未加载
	m.postStreamLoading = false
	_ = m.View() // 填充 postLineRanges

	// PgDn：postCursor 0→1，超界钳到最后一条(idx0)，且检测到有剩余 stream → 触发加载
	nm, cmd := m.movePostCursor(1)
	if cmd == nil {
		t.Error("到底且有剩余 stream 时 movePostCursor 应返回加载 Cmd")
	}
	if !nm.postStreamLoading {
		t.Error("触发加载后 postStreamLoading 应为 true")
	}
	if m.postCursor != 0 {
		t.Errorf("单帖时 PgDn 后 postCursor 应钳到 0, got %d", m.postCursor)
	}

	// 无剩余 stream 时 PgDn 不应触发加载
	m2 := newLinuxdoModel()
	m2.cookie = "c"
	m2.mode = viewPosts
	m2.posts = []Post{{ID: 1, Name: "A", Username: "a", PostNumber: 1, Cooked: "<p>x</p>"}}
	m2.postStream = nil
	_ = m2.View()
	_, cmd2 := m2.movePostCursor(1)
	if cmd2 != nil {
		t.Error("无剩余 stream 时 movePostCursor 不应返回 Cmd")
	}
}

// TestLinuxDoViewPostsCursorMark 验证当前 postCursor 帖子行首有黄色 ▸ 光标标记。
func TestLinuxDoViewPostsCursorMark(t *testing.T) {
	m := newLinuxdoModel()
	m.cookie = "c"
	m.mode = viewPosts
	m.postTitle = "T"
	m.posts = []Post{
		{ID: 1, Name: "A", Username: "a", PostNumber: 1, CreatedAt: "2026-08-01T12:00:00+08:00", Cooked: "<p>p1</p>"},
		{ID: 2, Name: "B", Username: "b", PostNumber: 2, CreatedAt: "2026-08-01T12:01:00+08:00", Cooked: "<p>p2</p>"},
	}
	m.postCursor = 1 // 光标停在第二帖
	v := m.View()
	// ▸ 光标标记应在视图中出现（ColAccent=黄色，ANSI 226）
	if !strings.Contains(v, "▸") {
		t.Errorf("当前帖子应显示 ▸ 光标标记: %q", v)
	}
	// 光标 ▸ 应只出现一次（仅当前帖）
	if strings.Count(v, "▸") != 1 {
		t.Errorf("▸ 应只出现 1 次(仅当前帖), got %d", strings.Count(v, "▸"))
	}
}
