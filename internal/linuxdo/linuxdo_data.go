package linuxdo

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const baseURL = "https://linux.do"

type Category struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Color      string `json:"color"`
	TopicCount int    `json:"topic_count"`
}

type Topic struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	PostsCount int    `json:"posts_count"`
	ReplyCount int    `json:"reply_count"`
	Views      int    `json:"views"`
	CreatedAt  string `json:"created_at"`
	Pinned     bool   `json:"pinned"`
	Closed     bool   `json:"closed"`
	CategoryID int    `json:"category_id"`
}

type Post struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`       // 显示名
	UserTitle  string `json:"user_title"` // 头衔
	Cooked     string `json:"cooked"`     // HTML 内容
	PostNumber int    `json:"post_number"`
	CreatedAt  string `json:"created_at"`

	ReplyCount         int     `json:"reply_count"`          // 被回复次数
	ReactionUsersCount int     `json:"reaction_users_count"` // 点赞/emoji 互动用户总量
	Boosts             []Boost `json:"boosts"`               // 短快捷回复

	ImageURLs []string `json:"-"` // 图片链接列表
}

// Boost 帖子的短快捷回复
type Boost struct {
	ID     int    `json:"id"`
	Cooked string `json:"cooked"` // HTML 内容
	User   struct {
		Username string `json:"username"`
	} `json:"user"`
}

type CategoriesMsg struct {
	Categories []Category
	Err        error
}

type TopicsMsg struct {
	Topics   []Topic
	FullPage bool // API 返回了满页（30 条），可能还有下一页
	Err      error
}

type TopicDetailMsg struct {
	Title      string
	Posts      []Post
	Stream     []int // 剩余未加载的 post ID 列表
	TotalPosts int   // 帖子总数（API 的 posts_count）
	Err        error
}

type PostStreamMsg struct {
	TopicID   int
	Posts     []Post
	Remaining []int // 剩余未加载的 post ID（链式加载用）
	Err       error
}

// SearchResult 搜索结果条目
type SearchResult struct {
	TopicID    int
	Title      string
	Blurb      string // 匹配片段
	Username   string
	PostsCount int
	ReplyCount int
	Tags       []string
}

// SearchMsg 搜索结果消息
type SearchMsg struct {
	Results []SearchResult
	Query   string
	Page    int  // 当前页码
	More    bool // 是否有更多结果
	Err     error
}

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyURL(&url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:7890",
		}),
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

func doGet(reqURL, cookie, userAgent string) ([]byte, error) {
	return doRequest(reqURL, cookie, userAgent, false, "")
}

func doRequest(reqURL, cookie, userAgent string, isAjax bool, referer string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Sec-CH-UA", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if referer != "" {
		req.Header.Set("Referer", referer)
	} else {
		req.Header.Set("Referer", "https://linux.do/")
	}
	req.Header.Set("Origin", "https://linux.do")
	if isAjax {
		// AJAX 请求（如 /posts.json 加载更多帖子）
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("X-CSRF-Token", "undefined")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("discourse-present", "true")
	} else {
		// 普通页面请求
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 读取响应体（可能 gzip 压缩）
		var errReader io.Reader = resp.Body
		if resp.Header.Get("Content-Encoding") == "gzip" {
			if gr, err := gzip.NewReader(resp.Body); err == nil {
				defer gr.Close()
				errReader = gr
			}
		}
		bodyBytes, _ := io.ReadAll(errReader)
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyPreview)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		reader = gr
	}

	return io.ReadAll(reader)
}

// FetchCategoriesCmd 获取分类列表
func FetchCategoriesCmd(cookie, userAgent string) tea.Cmd {
	return func() tea.Msg {
		data, err := doGet(baseURL+"/categories.json", cookie, userAgent)
		if err != nil {
			return CategoriesMsg{Err: err}
		}
		var resp struct {
			CategoryList struct {
				Categories []Category `json:"categories"`
			} `json:"category_list"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return CategoriesMsg{Err: err}
		}
		return CategoriesMsg{Categories: resp.CategoryList.Categories}
	}
}

// FetchLatestTopicsCmd 获取最新帖子
func FetchLatestTopicsCmd(page int, cookie, userAgent string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf(baseURL+"/latest.json?page=%d", page)
		return fetchTopics(url, cookie, userAgent)
	}
}

// FetchCategoryTopicsCmd 获取分类下帖子
func FetchCategoryTopicsCmd(categoryID, page int, cookie, userAgent string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf(baseURL+"/c/%d.json?page=%d", categoryID, page)
		return fetchTopics(url, cookie, userAgent)
	}
}

func fetchTopics(url, cookie, userAgent string) tea.Msg {
	data, err := doGet(url, cookie, userAgent)
	if err != nil {
		return TopicsMsg{Err: err}
	}
	var resp struct {
		TopicList struct {
			Topics []Topic `json:"topics"`
		} `json:"topic_list"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return TopicsMsg{Err: err}
	}
	return TopicsMsg{Topics: resp.TopicList.Topics, FullPage: len(resp.TopicList.Topics) >= 30}
}

// FetchTopicDetailCmd 获取帖子详情（含回复）
func FetchTopicDetailCmd(topicID int, cookie, userAgent string) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf(baseURL+"/t/%d.json", topicID)
		data, err := doGet(url, cookie, userAgent)
		if err != nil {
			return TopicDetailMsg{Err: err}
		}
		var resp struct {
			Title      string `json:"title"`
			PostsCount int    `json:"posts_count"`
			PostStream struct {
				Posts  []Post `json:"posts"`
				Stream []int  `json:"stream"`
			} `json:"post_stream"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return TopicDetailMsg{Err: err}
		}
		// stream 包含所有 post ID（含已加载的），需要过滤掉已加载的
		loadedIDs := make(map[int]bool, len(resp.PostStream.Posts))
		for _, p := range resp.PostStream.Posts {
			loadedIDs[p.ID] = true
		}
		var remaining []int
		for _, id := range resp.PostStream.Stream {
			if !loadedIDs[id] {
				remaining = append(remaining, id)
			}
		}
		return TopicDetailMsg{
			Title:      resp.Title,
			Posts:      resp.PostStream.Posts,
			Stream:     remaining,
			TotalPosts: resp.PostsCount,
		}
	}
}

// FetchPostStreamCmd 批量加载帖子（每批最多 20 个，链式加载）
func FetchPostStreamCmd(topicID int, postIDs []int, cookie, userAgent string) tea.Cmd {
	return func() tea.Msg {
		const batchSize = 20
		batch := postIDs
		var remaining []int
		if len(postIDs) > batchSize {
			batch = postIDs[:batchSize]
			remaining = postIDs[batchSize:]
		}
		// 用 url.Values 编码参数，防止url编码问题
		values := url.Values{}
		for _, id := range batch {
			values.Add("post_ids[]", fmt.Sprintf("%d", id))
		}
		fullURL := fmt.Sprintf("%s/t/%d/posts.json?%s", baseURL, topicID, values.Encode())
		referer := fmt.Sprintf("%s/t/%d", baseURL, topicID)

		data, err := doRequest(fullURL, cookie, userAgent, true, referer)
		if err != nil {
			return PostStreamMsg{TopicID: topicID, Err: err}
		}
		var resp struct {
			PostStream struct {
				Posts []Post `json:"posts"`
			} `json:"post_stream"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return PostStreamMsg{TopicID: topicID, Err: err}
		}

		return PostStreamMsg{TopicID: topicID, Posts: resp.PostStream.Posts, Remaining: remaining}
	}
}

// FetchSearchCmd 全站搜索
func FetchSearchCmd(query, cookie, userAgent string, page int) tea.Cmd {
	return func() tea.Msg {
		searchURL := fmt.Sprintf("%s/search.json?q=%s&page=%d", baseURL, url.QueryEscape(query), page)
		data, err := doGet(searchURL, cookie, userAgent)
		if err != nil {
			return SearchMsg{Query: query, Page: page, Err: err}
		}
		var resp struct {
			Topics []struct {
				ID         int    `json:"id"`
				Title      string `json:"title"`
				PostsCount int    `json:"posts_count"`
				ReplyCount int    `json:"reply_count"`
				Tags       []struct {
					Name string `json:"name"`
				} `json:"tags"`
			} `json:"topics"`
			Posts []struct {
				TopicID  int    `json:"topic_id"`
				Username string `json:"username"`
				Blurb    string `json:"blurb"`
			} `json:"posts"`
			GroupedSearchResult struct {
				MoreFullPageResults bool `json:"more_full_page_results"`
			} `json:"grouped_search_result"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return SearchMsg{Query: query, Page: page, Err: err}
		}

		// topics 和 posts 按索引一一对应，保留 API 相关性排序
		pLen := len(resp.Posts)
		tLen := len(resp.Topics)
		cap := pLen
		if tLen > cap {
			cap = tLen
		}
		results := make([]SearchResult, 0, cap)

		for i := 0; i < pLen || i < tLen; i++ {
			var r SearchResult
			if i < tLen {
				t := resp.Topics[i]
				r.TopicID = t.ID
				r.Title = t.Title
				r.PostsCount = t.PostsCount
				r.ReplyCount = t.ReplyCount
				if len(t.Tags) > 0 {
					r.Tags = make([]string, 0, len(t.Tags))
					for _, tag := range t.Tags {
						if tag.Name != "" {
							r.Tags = append(r.Tags, tag.Name)
						}
					}
				}
			}
			if i < pLen {
				p := resp.Posts[i]
				blurb := tagRe.ReplaceAllString(p.Blurb, "")
				blurb = strings.ReplaceAll(blurb, "\n", " ")
				r.Blurb = strings.TrimSpace(blurb)
				r.Username = p.Username
				// posts 可能比 topics 多，用 topic_id 补充
				if r.TopicID == 0 {
					r.TopicID = p.TopicID
				}
			}
			results = append(results, r)
		}
		return SearchMsg{Results: results, Query: query, Page: page, More: resp.GroupedSearchResult.MoreFullPageResults}
	}
}

var (
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	spaceRe = regexp.MustCompile(`[ \t]+`)
)

// HTMLToText 将 Discourse cooked HTML 转换为终端可读纯文本
func HTMLToText(html string) (string, []string) {
	s := html

	// 块级元素换行
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "<p>", "\n") // p 开标签视为段落分隔
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "</div>", "\n")
	s = strings.ReplaceAll(s, "</li>", "\n")
	s = strings.ReplaceAll(s, "</blockquote>", "\n")

	// 引用块前缀
	s = strings.ReplaceAll(s, "<blockquote>", "> ")

	// 图片 → [image]
	imgRe := regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*>`)
	var imageURLs []string
	s = imgRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := imgRe.FindStringSubmatch(match)
		if len(sub) < 2 || sub[1] == "" {
			return "[img]"
		}
		// 过滤 Discourse 内置 emoji 图片
		// 这类是表情符号，不是内容图片，不纳入预览
		if strings.Contains(sub[1], "/images/emoji/") {
			return ""
		}
		imageURLs = append(imageURLs, sub[1])
		return fmt.Sprintf("[img:%d]", len(imageURLs))
	})

	// 链接 → 提取文本
	linkRe := regexp.MustCompile(`<a[^>]*>(.*?)</a>`)
	s = linkRe.ReplaceAllString(s, "$1")

	// 去除所有剩余标签
	s = tagRe.ReplaceAllString(s, "")

	// HTML 实体解码
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&#x2F;", "/")

	// 清理多余空白
	s = spaceRe.ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)

	return s, imageURLs
}

// ImageLoadedMsg 图片下载+解码完成消息
type ImageLoadedMsg struct {
	Index int
	Img   image.Image
	Err   error
}

// FetchImageCmd 下载并解码图片
func FetchImageCmd(imgURL string, index int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", imgURL, nil)
		if err != nil {
			return ImageLoadedMsg{Index: index, Err: err}
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://linux.do/")

		resp, err := httpClient.Do(req)
		if err != nil {
			return ImageLoadedMsg{Index: index, Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return ImageLoadedMsg{Index: index, Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
		}

		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return ImageLoadedMsg{Index: index, Err: err}
		}
		return ImageLoadedMsg{Index: index, Img: img}
	}
}
