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
		{"图片替换", `<img src="x.png" alt="pic"> end`, []string{"[image] end"}},
		{"块引用前缀", "<blockquote>quote</blockquote>", []string{"> quote"}},
		{"HTML 实体解码", "a &amp; b &lt; c", []string{"a & b < c"}},
		{"空白清理", "  lots   of   spaces  ", []string{"lots of spaces"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTMLToText(tt.html)
			for _, w := range tt.want {
				if !containsStr(got, w) {
					t.Errorf("HTMLToText(%q) = %q, 缺少 %q", tt.html, got, w)
				}
			}
		})
	}

	// 综合示例：strong 文本保留、链接文本提取、图片替换
	got := HTMLToText(`<p>Hello <strong>world</strong>!</p>
<a href="https://linux.do">link</a>
<img src="i.png">`)
	for _, w := range []string{"Hello world", "link", "[image]"} {
		if !containsStr(got, w) {
			t.Errorf("HTMLToText 综合示例缺少 %q: %q", w, got)
		}
	}
}
