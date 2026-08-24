package linuxdo

import (
	"strings"
	"testing"
)

func containsStr(s, sub string) bool { return strings.Contains(s, sub) }

// TestHTMLToText 测试 Discourse cooked HTML → 纯文本转换
func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string // 期望包含的子串
	}{
		{"纯文本", "Hello world", []string{"Hello world"}},
		{"段落换行", "<p>First</p><p>Second</p>", []string{"First", "Second"}},
		{"br 换行", "Line1<br>Line2", []string{"Line1", "Line2"}},
		{"链接提取文本", `Visit <a href="https://x.com">the site</a> now`, []string{"Visit the site now"}},
		{"图片占位符", `<img src="x.png" alt="pic"> end`, []string{"[img:1] end"}},
		{"块引用前缀", "<blockquote>quote</blockquote>", []string{"> quote"}},
		{"HTML 实体解码", "a &amp; b &lt; c", []string{"a & b < c"}},
		{"空白清理", "  lots   of   spaces  ", []string{"lots of spaces"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := HTMLToText(tt.html)
			for _, w := range tt.want {
				if !containsStr(got, w) {
					t.Errorf("HTMLToText(%q) = %q, 缺少 %q", tt.html, got, w)
				}
			}
		})
	}

	// strong 文本保留、链接文本提取、图片占位符
	got, imgs := HTMLToText(`<p>Hello <strong>world</strong>!</p>
<a href="https://linux.do">link</a>
<img src="i.png">`)
	for _, w := range []string{"Hello world", "link", "[img:1]"} {
		if !containsStr(got, w) {
			t.Errorf("HTMLToText 综合示例缺少 %q: %q", w, got)
		}
	}
	// 验证图片 URL 提取
	if len(imgs) != 1 || imgs[0] != "i.png" {
		t.Errorf("图片 URL 提取失败: got %v", imgs)
	}

	// 多图片编号测试
	text, urls := HTMLToText(`<img src="a.jpg"><img src="b.png"><img src="c.gif">`)
	if len(urls) != 3 {
		t.Errorf("期望 3 个 URL，got %d", len(urls))
	}
	for _, w := range []string{"[img:1]", "[img:2]", "[img:3]"} {
		if !strings.Contains(text, w) {
			t.Errorf("缺少占位符 %s: %q", w, text)
		}
	}
	if urls[0] != "a.jpg" || urls[1] != "b.png" || urls[2] != "c.gif" {
		t.Errorf("URL 顺序错误: %v", urls)
	}
}
