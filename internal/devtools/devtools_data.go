package devtools

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"encoding/ascii85"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"hash/fnv"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Tool 一个工具箱条目
type Tool struct {
	Section string // 功能区：Codec / Hash / Tools
	Group   string // 区内分组：Base / URL / Multi
	Name    string
	NoInput bool   // 无需输入即可运行
	Async   bool   // 异步执行：耗时操作（如 HTTP），结果通过消息返回
	Hint    string // 空输入时的格式提示（多参数工具用）
	Run     func(input string) (string, error)
}

// 功能区常量，sections 顺序即左列 Tab 顺序
const (
	SectionCodec = "Codec" // 编码：Base 全系列 + URL
	SectionHash  = "Hash"  // 哈希
	SectionTools = "Tools" // 开发工具区
)

var sections = []string{SectionCodec, SectionHash, SectionTools}

// toolsInSection 返回指定功能区的工具子集
func toolsInSection(tools []Tool, section string) []Tool {
	var out []Tool
	for _, t := range tools {
		if t.Section == section {
			out = append(out, t)
		}
	}
	return out
}

// builtinTools 内置工具列表
var builtinTools = []Tool{
	{Section: SectionCodec, Group: "Base", Name: "Base16 (Hex) Encode", Run: func(s string) (string, error) {
		return wrapLong(hex.EncodeToString([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base16 (Hex) Decode", Run: func(s string) (string, error) {
		b, err := hex.DecodeString(strings.TrimSpace(s))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base32 Encode", Run: func(s string) (string, error) {
		return wrapLong(base32.StdEncoding.EncodeToString([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base32 Decode", Run: func(s string) (string, error) {
		return base32Decode(s, base32.StdEncoding)
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base32hex Encode", Run: func(s string) (string, error) {
		return wrapLong(base32.HexEncoding.EncodeToString([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base32hex Decode", Run: func(s string) (string, error) {
		return base32Decode(s, base32.HexEncoding)
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base64 Encode", Run: func(s string) (string, error) {
		return wrapLong(base64.StdEncoding.EncodeToString([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base64 Decode", Run: func(s string) (string, error) {
		return base64Decode(s, base64.StdEncoding)
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base64 URL Encode", Run: func(s string) (string, error) {
		return wrapLong(base64.URLEncoding.EncodeToString([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base64 URL Decode", Run: func(s string) (string, error) {
		return base64Decode(s, base64.URLEncoding)
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base62 Encode", Run: func(s string) (string, error) {
		return wrapLong(base62Encode([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base62 Decode", Run: func(s string) (string, error) {
		return base62Decode(strings.TrimSpace(s))
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base85 (Ascii85) Encode", Run: func(s string) (string, error) {
		dst := make([]byte, ascii85.MaxEncodedLen(len(s)))
		n := ascii85.Encode(dst, []byte(s))
		return wrapLong(string(dst[:n])), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base85 (Ascii85) Decode", Run: func(s string) (string, error) {
		dst := make([]byte, len(s))
		n, _, err := ascii85.Decode(dst, []byte(strings.TrimSpace(s)), true)
		if err != nil {
			return "", err
		}
		return string(dst[:n]), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base91 Encode", Run: func(s string) (string, error) {
		return wrapLong(base91Encode([]byte(s))), nil
	}},
	{Section: SectionCodec, Group: "Base", Name: "Base91 Decode", Run: func(s string) (string, error) {
		return base91Decode(strings.TrimSpace(s))
	}},

	// ---- URL ----
	{Section: SectionCodec, Group: "URL", Name: "URL Encode", Run: func(s string) (string, error) {
		return wrapLong(url.QueryEscape(s)), nil
	}},
	{Section: SectionCodec, Group: "URL", Name: "URL Decode", Run: func(s string) (string, error) {
		return url.QueryUnescape(strings.TrimSpace(s))
	}},

	// ---- Hash ----
	{Section: SectionHash, Group: "Hash", Name: "MD5", Run: func(s string) (string, error) {
		return wrapLong(fmt.Sprintf("%x", md5.Sum([]byte(s)))), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA1", Run: func(s string) (string, error) {
		return wrapLong(fmt.Sprintf("%x", sha1.Sum([]byte(s)))), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA224", Run: func(s string) (string, error) {
		return wrapLong(sha224Hex(s)), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA256", Run: func(s string) (string, error) {
		return wrapLong(fmt.Sprintf("%x", sha256.Sum256([]byte(s)))), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA384", Run: func(s string) (string, error) {
		return wrapLong(sha384Hex(s)), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA512", Run: func(s string) (string, error) {
		return wrapLong(fmt.Sprintf("%x", sha512.Sum512([]byte(s)))), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA512/224", Run: func(s string) (string, error) {
		return wrapLong(sha512224Hex(s)), nil
	}},
	{Section: SectionHash, Group: "Hash", Name: "SHA512/256", Run: func(s string) (string, error) {
		return wrapLong(sha512256Hex(s)), nil
	}},

	// ---- Checksum 校验和 ----
	{Section: SectionHash, Group: "Checksum", Name: "CRC32", Run: func(s string) (string, error) {
		return wrapLong(crc32Hex(s)), nil
	}},
	{Section: SectionHash, Group: "Checksum", Name: "CRC64", Run: func(s string) (string, error) {
		return wrapLong(crc64Hex(s)), nil
	}},
	{Section: SectionHash, Group: "Checksum", Name: "FNV-1a", Run: func(s string) (string, error) {
		return wrapLong(fnv1aHex(s)), nil
	}},
	{Section: SectionHash, Group: "Checksum", Name: "Adler-32", Run: func(s string) (string, error) {
		return wrapLong(adler32Hex(s)), nil
	}},

	// ---- HMAC 带密钥消息认证 ----
	{Section: SectionHash, Group: "HMAC", Name: "HMAC-SHA256", Hint: "第一行密钥，第二行数据", Run: func(s string) (string, error) {
		key, data, err := splitKeyData(s)
		if err != nil {
			return "", err
		}
		return wrapLong(hmacSHA256(key, data)), nil
	}},

	// ---- Multi 多层解码 ----
	{Section: SectionCodec, Group: "Multi", Name: "Multi-Decode", Run: func(s string) (string, error) {
		steps, err := multiDecode(s)
		if err != nil {
			return "", err
		}
		return strings.Join(steps, "\n"), nil
	}},

	// ---- Tools 开发工具区 ----
	{Section: SectionTools, Group: "Auth", Name: "JWT Decode", Run: func(s string) (string, error) {
		return jwtDecode(s)
	}},
	{Section: SectionTools, Group: "Num", Name: "Number Convert", Run: func(s string) (string, error) {
		return numConvert(s)
	}},
	{Section: SectionTools, Group: "Time", Name: "Unix Timestamp", Run: func(s string) (string, error) {
		return unixTimeConvert(s)
	}},
	{Section: SectionTools, Group: "Text", Name: "Unicode Escape", Run: func(s string) (string, error) {
		return unicodeEscape(s), nil
	}},
	{Section: SectionTools, Group: "Text", Name: "Unicode Unescape", Run: func(s string) (string, error) {
		return unicodeUnescape(s)
	}},
	{Section: SectionTools, Group: "Text", Name: "Text Stats", Run: func(s string) (string, error) {
		return textStats(s), nil
	}},
	{Section: SectionTools, Group: "Misc", Name: "UUID v4", NoInput: true, Run: func(string) (string, error) {
		return uuidV4()
	}},

	// ---- Encrypt 对称加解密 ----
	{Section: SectionTools, Group: "Encrypt", Name: "AES-256 Encrypt", Hint: "第一行密钥，第二行数据", Run: func(s string) (string, error) {
		key, data, err := splitKeyData(s)
		if err != nil {
			return "", err
		}
		return aesEncrypt(key, []byte(data))
	}},
	{Section: SectionTools, Group: "Encrypt", Name: "AES-256 Decrypt", Hint: "第一行密钥，第二行密文(hex)", Run: func(s string) (string, error) {
		key, data, err := splitKeyData(s)
		if err != nil {
			return "", err
		}
		return aesDecrypt(key, strings.TrimSpace(data))
	}},

	// ---- HTTP 接口测试与安全检查（异步执行）----
	{Section: SectionTools, Group: "HTTP", Name: "HTTP Request", Async: true,
		Hint: "第一行 METHOD URL，随后 Header 行，空行后 Body", Run: func(s string) (string, error) {
			return httpRequest(s)
		}},
	{Section: SectionTools, Group: "HTTP", Name: "HTTP Response Audit", Async: true,
		Hint: "同 HTTP Request 格式", Run: func(s string) (string, error) {
			return httpAudit(s)
		}},

	// ---- HTTP 开发辅助（同步）----
	{Section: SectionTools, Group: "HTTP", Name: "Curl 转请求", Run: func(s string) (string, error) {
		return curlParse(s)
	}},
	{Section: SectionTools, Group: "Auth", Name: "JWT 验签", Hint: "第一行密钥，第二行 token", Run: func(s string) (string, error) {
		return jwtVerify(s)
	}},
	{Section: SectionTools, Group: "HTTP", Name: "HTTP Status Lookup", Run: func(s string) (string, error) {
		return httpStatusLookup(s)
	}},
}

// base32Decode 宽容解码，先按带 padding 的标准编码，失败后按 NoPadding 重试
func base32Decode(s string, enc *base32.Encoding) (string, error) {
	s = strings.TrimSpace(s)
	b, err := enc.DecodeString(s)
	if err == nil {
		return string(b), nil
	}
	b, err = enc.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// base64Decode 宽容解码，先按带 padding 的标准编码，失败后按 NoPadding 重试
func base64Decode(s string, enc *base64.Encoding) (string, error) {
	s = strings.TrimSpace(s)
	b, err := enc.DecodeString(s)
	if err == nil {
		return string(b), nil
	}
	b, err = enc.WithPadding(base64.NoPadding).DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ---- Base62 大数转换 ----

// base62Alphabet 标准 Base62 字符表（0-9、A-Z、a-z）
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var base62DecTable [256]int

func init() {
	for i := range base62DecTable {
		base62DecTable[i] = -1
	}
	for i, c := range base62Alphabet {
		base62DecTable[int(c)] = i
	}
}

// base62Encode 字节 → Base62，0x00 前缀输出 '0' 保证无损还原
func base62Encode(input []byte) string {
	// SetBytes 会丢弃前导零字节，单独统计并用 '0'（alphabet[0]）补回
	lead := 0
	for lead < len(input) && input[lead] == 0 {
		lead++
	}
	x := new(big.Int).SetBytes(input[lead:])
	base := big.NewInt(62)
	mod := new(big.Int)
	// 预分配容量 base62 每字符承载 log2(62) ≈ 5.954 bit，每字节 8 bit
	// 输出字符数 ≈ 输入字节数 × 8/5.954 ≈ 1.3436。138/100 = 1.38 是略大于 1.3436 的保守上界，+1 为防超长度兜底。
	result := make([]byte, 0, len(input)*138/100+1)
	for x.Sign() > 0 {
		x.DivMod(x, base, mod) // x = x÷62，余数写进 mod
		result = append(result, base62Alphabet[mod.Int64()])
	}
	// 结果当前是低位在前，需反转
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	// 前导零补 '0'
	prefix := make([]byte, lead)
	for i := range prefix {
		prefix[i] = base62Alphabet[0]
	}
	return string(append(prefix, result...))
}

// base62Decode Base62 → 字节
func base62Decode(s string) (string, error) {
	// 统计前导 '0'，解码主体后补回 0x00
	lead := 0
	for lead < len(s) && s[lead] == '0' {
		lead++
	}
	x := new(big.Int)
	base := big.NewInt(62)
	// 从左到右累加还原大整数
	for _, r := range s[lead:] {
		if r > 255 { // 查表前先防 rune 越界
			return "", fmt.Errorf("invalid base62 char %q", r)
		}
		v := base62DecTable[r]
		if v < 0 {
			return "", fmt.Errorf("invalid base62 char %q", r)
		}
		x.Mul(x, base)
		x.Add(x, big.NewInt(int64(v)))
	}
	prefix := make([]byte, lead)
	return string(append(prefix, x.Bytes()...)), nil
}

// ---- Base91 ----

// base91Alphabet base91 字符表
const base91Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&()*+,./:;<=>?@[]^_`{|}~\""

var base91DecTable [256]int

func init() {
	for i := range base91DecTable {
		base91DecTable[i] = -1
	}
	for i, c := range base91Alphabet {
		base91DecTable[int(c)] = i
	}
}

// base91Encode 字节 → Base91
func base91Encode(input []byte) string {
	var out []byte
	var b uint32
	var n int
	for _, byteVal := range input {
		b |= uint32(byteVal) << uint(n)
		n += 8
		// 一位 91 进制 ≈ 6.5 bit，装不下 8-bit 字节，必须两位一组（≈13~14 bit）拼够再切
		if n > 13 {
			var v uint32
			v = b & 8191 // 先取低 13 位
			// 两位 base91 数码最多表示 8280 = 91*90 + 90（两个码都取最大值 90，基数 91）
			// 8192 = 2^13，是第 14 位的权重
			// 低 13 位 > 88：再借第 14 位必然超过 8280 → 退回切 13 位，剩余留到下一组
			// 低 13 位 ≤ 88：即使第 14 位是 1，88+8192 = 8280 仍不越界 → 切 14 位
			if v > 88 {
				b >>= 13
				n -= 13
			} else {
				v = b & 16383 // 重取低 14 位（含第 14 位，最坏 = 88+8192 = 8280）
				b >>= 14
				n -= 14
			}
			// 这一次 out 的两个值即为 91 进制的两位数（v = 高码×91 + 低码）
			out = append(out, base91Alphabet[v%91], base91Alphabet[v/91])
		}
	}
	if n > 0 {
		out = append(out, base91Alphabet[b%91])
		if n > 7 || b > 90 {
			out = append(out, base91Alphabet[b/91])
		}
	}
	return string(out)
}

// base91Decode Base91 → 字节
func base91Decode(s string) (string, error) {
	var out []byte
	var b uint32
	var n int
	v := -1
	for _, r := range s {
		if r > 255 { // 查表前先防 rune 越界
			return "", fmt.Errorf("invalid base91 char %q", r)
		}
		c := base91DecTable[r]
		if c < 0 {
			return "", fmt.Errorf("invalid base91 char %q", r)
		}
		if v < 0 {
			// 第一个码：先存进 v，等第二个
			v = c
		} else {
			// 第二个码：v = 高码×91 + 低码（编码的 v/91, v%91 逆运算）
			v += c * 91
			b |= uint32(v) << uint(n)
			if v&8191 > 88 {
				n += 13
			} else {
				n += 14
			}
			// 累加器攒够 8 bit，就吐出一个字节
			for n > 7 {
				out = append(out, byte(b&255))
				b >>= 8
				n -= 8
			}
			v = -1
		}
	}
	if v > -1 {
		out = append(out, byte((b|uint32(v)<<uint(n))&255))
	}
	return string(out), nil
}

// ---- 多层解码 ----

// decodeStep 一种可用的解码尝试
type decodeStep struct {
	name   string
	decode func(string) (string, error)
}

// multiDecode 逐层尝试全部 base 解码器，按结果可读性择优剥层，直到无可读候选
func multiDecode(input string) (steps []string, err error) {
	decoders := []decodeStep{
		{"base64", func(s string) (string, error) { return base64Decode(s, base64.StdEncoding) }},
		{"base64url", func(s string) (string, error) { return base64Decode(s, base64.URLEncoding) }},
		{"base32", func(s string) (string, error) { return base32Decode(s, base32.StdEncoding) }},
		{"base32hex", func(s string) (string, error) { return base32Decode(s, base32.HexEncoding) }},
		{"base16", func(s string) (string, error) {
			b, err := hex.DecodeString(strings.TrimSpace(s))
			return string(b), err
		}},
		{"base62", func(s string) (string, error) { return base62Decode(strings.TrimSpace(s)) }},
		{"base85", func(s string) (string, error) {
			dst := make([]byte, len(s))
			n, _, err := ascii85.Decode(dst, []byte(strings.TrimSpace(s)), true)
			if err != nil {
				return "", err
			}
			return string(dst[:n]), nil
		}},
		{"base91", func(s string) (string, error) { return base91Decode(strings.TrimSpace(s)) }},
	}

	cur := strings.TrimSpace(input)
	const maxLayer = 10
	for layer := 1; layer <= maxLayer; layer++ {
		var bestName, bestOut string
		bestScore := -1.0
		// 贪心策略，每种编码解码尝试一遍，择优选取
		for _, d := range decoders {
			out, e := d.decode(cur)
			if e != nil || out == cur {
				continue
			}
			// 拒绝乱码候选，避免字符集重叠选错解码器、明文被继续误剥
			if score := printableRatio(out); score >= 0.5 && score > bestScore {
				bestName, bestOut, bestScore = d.name, out, score
			}
		}
		if bestName == "" {
			break
		}
		// 转义控制字符，避免污染 TUI 渲染
		steps = append(steps, fmt.Sprintf("[%d] %s → %s", layer, bestName, truncate(escapeCtl(bestOut))))
		cur = bestOut
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("未检测到可解码的 base 编码")
	}
	return steps, nil
}

// printableRatio 返回字符串中可读字节占比，用于剥层择优
func printableRatio(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	good := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= 0x20 && c <= 0x7e: // ASCII 可打印字符
			good++
		case c == '\n' || c == '\t' || c == '\r': // 文本结构符视为可读
			good++
		}
	}
	return float64(good) / float64(len(s))
}

// escapeCtl 将控制字符转义为 \xNN，避免其破坏 TUI 渲染
func escapeCtl(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\n' || c == '\t':
			b.WriteByte(c)
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&b, "\\x%02X", c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// truncate 限制展示长度，避免多层解码中间结果无限膨胀
func truncate(s string) string {
	const max = 60
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// wrapLong 按已有换行拆行，仅硬切超过 step 字符的行（幂等：重复调用不二次插换行）
func wrapLong(s string) string {
	const step = 64
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		r := []rune(l)
		if len(r) <= step {
			continue // 短行/已分段的行，原样保留
		}
		// 超长单行：按 step 个字符硬切，段间插 \n
		var b strings.Builder
		for j := 0; j < len(r); j += step {
			if j > 0 {
				b.WriteByte('\n')
			}
			end := j + step
			if end > len(r) {
				end = len(r)
			}
			b.WriteString(string(r[j:end]))
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

// ---- Tools 区 ----

// jwtDecode 解析 JWT，base64url 解码 header/payload 并格式化 JSON 展示
func jwtDecode(s string) (string, error) {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("JWT 需要 header.payload.signature 三段，实际 %d 段", len(parts))
	}
	decode := func(raw string) (string, error) {
		// 先按 RawURL（无 padding），失败补 padding 重试
		for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding} {
			if b, err := enc.DecodeString(raw); err == nil {
				return formatJSON(string(b))
			}
		}
		return "", fmt.Errorf("base64url 解码失败")
	}
	header, err := decode(parts[0])
	if err != nil {
		return "", fmt.Errorf("header: %v", err)
	}
	payload, err := decode(parts[1])
	if err != nil {
		return "", fmt.Errorf("payload: %v", err)
	}
	return "header:\n" + header + "\n\npayload:\n" + payload, nil
}

// formatJSON 缩进格式化 JSON，非法 JSON 原样返回
func formatJSON(s string) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(strings.TrimSpace(s)), "", "  "); err != nil {
		return s, nil
	}
	return buf.String(), nil
}

// numConvert 输入数字按前缀自动识别进制，输出 dec/hex/bin/oct 四行
func numConvert(s string) (string, error) {
	body := strings.TrimSpace(s)
	base, digits := 10, body
	switch {
	case strings.HasPrefix(body, "0x"), strings.HasPrefix(body, "0X"):
		base, digits = 16, body[2:]
	case strings.HasPrefix(body, "0b"), strings.HasPrefix(body, "0B"):
		base, digits = 2, body[2:]
	case strings.HasPrefix(body, "0o"), strings.HasPrefix(body, "0O"):
		base, digits = 8, body[2:]
	}
	x, ok := new(big.Int).SetString(digits, base)
	if !ok {
		return "", fmt.Errorf("invalid number %q（base %d）", body, base)
	}
	return fmt.Sprintf("dec: %s\nhex: 0x%X\nbin: 0b%b\noct: 0o%o", x, x, x, x), nil
}

// unicodeEscape 将非 ASCII 与控制字符转义为 \uXXXX（可读文本保留）
func unicodeEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r > 0x7e {
			fmt.Fprintf(&b, "\\u%04X", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// unicodeUnescape 还原 \uXXXX / \xXX 转义序列
func unicodeUnescape(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		switch s[i+1] {
		case 'u':
			if i+6 > len(s) {
				return "", fmt.Errorf("truncated \\u escape")
			}
			v, err := strconv.ParseUint(s[i+2:i+6], 16, 32)
			if err != nil {
				return "", fmt.Errorf("invalid \\u escape %q", s[i:i+6])
			}
			b.WriteRune(rune(v))
			i += 6
		case 'x':
			if i+4 > len(s) {
				return "", fmt.Errorf("truncated \\x escape")
			}
			v, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
			if err != nil {
				return "", fmt.Errorf("invalid \\x escape %q", s[i:i+4])
			}
			b.WriteByte(byte(v))
			i += 4
		case '\\':
			b.WriteByte('\\')
			i += 2
		default:
			b.WriteByte('\\')
			i++
		}
	}
	return b.String(), nil
}

// uuidV4 生成 UUID v4
func uuidV4() (string, error) {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = u[6]&0x0f | 0x40 // 版本 4
	u[8] = u[8]&0x3f | 0x80 // 变体 RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16]), nil
}

// unixTimeConvert 时间戳与日期互转，数字按位数识别秒/毫秒/微秒
func unixTimeConvert(s string) (string, error) {
	body := strings.TrimSpace(s)
	if body == "" {
		return "", fmt.Errorf("输入为空")
	}
	if n, err := strconv.ParseInt(body, 10, 64); err == nil {
		unit, t := "seconds", time.Unix(n, 0)
		switch {
		case n >= 1e15: // 16 位及以上 → 微秒
			unit, t = "microseconds", time.UnixMicro(n)
		case n >= 1e12: // 13 位 → 毫秒
			unit, t = "milliseconds", time.UnixMilli(n)
		}
		return fmt.Sprintf("unix %s: %d\nutc:   %s\nlocal: %s",
			unit, n, t.UTC().Format("2006-01-02 15:04:05"), t.Local().Format("2006-01-02 15:04:05")), nil
	}
	// 按常见格式解析为本地时间
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, body, time.Local); err == nil {
			return fmt.Sprintf("local: %s\nunix:  %d", t.Format("2006-01-02 15:04:05"), t.Unix()), nil
		}
	}
	return "", fmt.Errorf("无法识别：既非数字时间戳，也非常见日期格式")
}

// textStats 文本统计字符 / 字节 / 单词 / 行数 / 去重行数
func textStats(s string) string {
	if s == "" {
		return "chars: 0\nbytes: 0\nwords: 0\nlines: 0\nunique lines: 0"
	}
	body := strings.TrimSuffix(s, "\n") // 末尾换行不额外算一行
	chars := utf8.RuneCountInString(s)
	bytes := len(s)
	words := len(strings.Fields(s))
	lines := len(strings.Split(body, "\n"))
	seen := make(map[string]struct{})
	uniq := 0
	for _, ln := range strings.Split(body, "\n") {
		if _, ok := seen[ln]; !ok {
			seen[ln] = struct{}{}
			uniq++
		}
	}
	return fmt.Sprintf("chars: %d\nbytes: %d\nwords: %d\nlines: %d\nunique lines: %d",
		chars, bytes, words, lines, uniq)
}

// ---- Hash 与校验和辅助函数 ----

func sha224Hex(s string) string    { return fmt.Sprintf("%x", sha256.Sum224([]byte(s))) }
func sha384Hex(s string) string    { return fmt.Sprintf("%x", sha512.Sum384([]byte(s))) }
func sha512224Hex(s string) string { return fmt.Sprintf("%x", sha512.Sum512_224([]byte(s))) }
func sha512256Hex(s string) string { return fmt.Sprintf("%x", sha512.Sum512_256([]byte(s))) }
func crc32Hex(s string) string     { return fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(s))) }
func crc64Hex(s string) string {
	return fmt.Sprintf("%016x", crc64.Checksum([]byte(s), crc64.MakeTable(crc64.ECMA)))
}
func fnv1aHex(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%016x", h.Sum64())
}
func adler32Hex(s string) string { return fmt.Sprintf("%08x", adler32.Checksum([]byte(s))) }

// hmacSHA256 用密钥计算数据的 HMAC-SHA256（十六进制小写）
func hmacSHA256(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// ---- Encrypt 对称加解密 ----

// splitKeyData 拆出密钥与数据：第一行密钥，第二行起为数据
func splitKeyData(s string) (key, data string, err error) {
	lines := strings.SplitN(s, "\n", 2)
	key = strings.TrimSpace(lines[0])
	if key == "" {
		return "", "", fmt.Errorf("第一行为空，缺少密钥")
	}
	if len(lines) == 2 {
		data = lines[1]
	}
	return key, data, nil
}

// aesKey 口令 → 32 字节 AES-256 密钥（SHA-256 派生，免记 hex）
func aesKey(passphrase string) []byte {
	sum := sha256.Sum256([]byte(passphrase))
	return sum[:]
}

// aesEncrypt AES-256-GCM 加密，nonce 随机并拼在密文前，输出 hex
func aesEncrypt(passphrase string, plain []byte) (string, error) {
	block, err := aes.NewCipher(aesKey(passphrase))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)
	// nonce || ciphertext 一并输出，解密端自动拆分，用户无需手填 IV
	return wrapLong(hex.EncodeToString(append(nonce, sealed...))), nil
}

// aesDecrypt AES-256-GCM 解密，从输入 hex 自动拆出 nonce 与密文
func aesDecrypt(passphrase, in string) (string, error) {
	// 加密输出经 wrapLong 换行，容忍密文中的空白（换行/空格/tab）
	in = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', ' ', '\t':
			return -1
		}
		return r
	}, strings.TrimSpace(in))
	raw, err := hex.DecodeString(in)
	if err != nil {
		return "", fmt.Errorf("密文需为 hex：%v", err)
	}
	block, err := aes.NewCipher(aesKey(passphrase))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize()+gcm.Overhead() {
		// 过短多为把密文直接粘贴、第一行被误当密钥；提示最小长度与格式
		return "", fmt.Errorf("密文过短（至少 %d 个 hex 字符；确认第一行是密钥，第二行起是密文）", (gcm.NonceSize()+gcm.Overhead())*2)
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密钥错误或密文被篡改）：%v", err)
	}
	return string(plain), nil
}

// ---- HTTP 接口测试与安全检查 ----

// httpReq 解析后的 HTTP 请求参数
type httpReq struct {
	method  string   // GET/POST/...
	url     string   // 目标地址
	headers []string // "Key: Value" 行
	body    string   // 请求体
}

// parseHTTPInput 解析 "METHOD URL\nKey: Value\n\nbody" 格式
func parseHTTPInput(s string) (httpReq, error) {
	var req httpReq
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return req, fmt.Errorf("第一行需为 METHOD URL")
	}
	fields := strings.Fields(strings.TrimSpace(lines[0]))
	if len(fields) == 1 {
		req.method, req.url = "GET", fields[0]
	} else {
		req.method = strings.ToUpper(fields[0])
		req.url = fields[1]
	}
	if req.url == "" {
		return req, fmt.Errorf("缺少 URL")
	}
	if !validHTTPMethod(req.method) {
		return req, fmt.Errorf("不支持的请求方法 %q", req.method)
	}
	u, err := url.Parse(req.url)
	if err != nil {
		return req, fmt.Errorf("URL 无效：%v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return req, fmt.Errorf("URL 需带 http:// 或 https:// 协议前缀")
	}
	// 剩余行：含 ":" 视为 Header，其余视为 Body（宽容：漏空行也能识别 body）
	inBody := false
	for i := 1; i < len(lines); i++ {
		ln := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(ln) == "" {
			inBody = true
			continue
		}
		if !inBody && strings.Contains(ln, ":") {
			req.headers = append(req.headers, ln)
		} else {
			inBody = true
			req.body += ln + "\n"
		}
	}
	req.body = strings.TrimSuffix(req.body, "\n")
	return req, nil
}

// validHTTPMethod 判断是否为常见 HTTP 方法
func validHTTPMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return true
	}
	return false
}

// httpClient 带超时的共享客户端，避免 DNS/连接挂死阻塞事件循环
var httpClient = &http.Client{Timeout: 10 * time.Second}

// httpResponse 请求结果（响应体 + 排序后的展示头 + TLS 状态）
type httpResponse struct {
	status  string               // "200 OK"
	header  http.Header          // 原始响应头（用于 Audit 取单头）
	headers []string             // 排序后的 "K: V" 展示行
	body    string               // 响应体
	tls     *tls.ConnectionState // 仅 https 时非 nil
}

// headerValue 取单个响应头值
func (r httpResponse) headerValue(name string) string { return r.header.Get(name) }

// doHTTP 执行请求并读取完整响应（响应体限制 1MB，超长截断）
func doHTTP(req httpReq) (httpResponse, error) {
	var body io.Reader
	if req.body != "" {
		body = strings.NewReader(req.body)
	}
	hreq, err := http.NewRequest(req.method, req.url, body)
	if err != nil {
		return httpResponse{}, err
	}
	for _, h := range req.headers {
		if k, v, ok := strings.Cut(h, ":"); ok {
			hreq.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
	resp, err := httpClient.Do(hreq)
	if err != nil {
		return httpResponse{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return httpResponse{}, err
	}
	var out httpResponse
	out.status, out.header, out.body, out.tls = resp.Status, resp.Header, string(data), resp.TLS
	names := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		for _, v := range resp.Header.Values(k) {
			out.headers = append(out.headers, k+": "+v)
		}
	}
	return out, nil
}

// httpRequest 发起请求并格式化状态行 / 响应头 / 响应体
func httpRequest(s string) (string, error) {
	req, err := parseHTTPInput(s)
	if err != nil {
		return "", err
	}
	resp, err := doHTTP(req)
	if err != nil {
		return "", fmt.Errorf("请求失败：%v", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n→ %s\n", req.method, req.url, resp.status)
	for _, h := range resp.headers {
		fmt.Fprintf(&b, "%s\n", h)
	}
	if resp.body != "" {
		b.WriteString("\n")
		r := []rune(resp.body)
		if len(r) > 4000 {
			r = r[:4000]
		}
		b.WriteString(escapeCtl(wrapLong(string(r)))) // 控制字符转义 + 长内容换行
	}
	return b.String(), nil
}

// secretPatterns 响应体敏感数据泄露关键词正则
var secretPatterns = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(`(?i)password["'\s:=]+[^\s"']{4,}`), "密码字段"},
	{regexp.MustCompile(`(?i)api[_-]?key["'\s:=]+[^\s"']{8,}`), "API Key"},
	{regexp.MustCompile(`(?i)authorization["'\s:=]+[^\s"']{8,}`), "授权头"},
	{regexp.MustCompile(`(?i)bearer\s+[a-z0-9._-]{10,}`), "Bearer Token"},
	{regexp.MustCompile(`(?i)secret["'\s:=]+[^\s"']{8,}`), "机密字段"},
	{regexp.MustCompile(`(?i)token["'\s:=]+[^\s"']{8,}`), "Token"},
}

// httpAudit 发起请求并做安全检查：TLS 证书 / 安全 Header / CORS / 敏感数据泄露
func httpAudit(s string) (string, error) {
	req, err := parseHTTPInput(s)
	if err != nil {
		return "", err
	}
	resp, err := doHTTP(req)
	if err != nil {
		return "", fmt.Errorf("请求失败：%v", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s → %s\n", req.method, req.url, resp.status)

	// TLS 证书检查（仅 https）
	if resp.tls != nil && len(resp.tls.PeerCertificates) > 0 {
		cert := resp.tls.PeerCertificates[0]
		now := time.Now()
		switch {
		case now.After(cert.NotAfter):
			fmt.Fprintf(&b, "[✗] TLS 证书已过期（%s）\n", cert.NotAfter.Format("2006-01-02"))
		case now.Before(cert.NotBefore):
			fmt.Fprintf(&b, "[✗] TLS 证书尚未生效（%s）\n", cert.NotBefore.Format("2006-01-02"))
		default:
			fmt.Fprintf(&b, "[✓] TLS 证书有效（%s ~ %s，签发 %s）\n",
				cert.NotBefore.Format("2006-01-02"), cert.NotAfter.Format("2006-01-02"), cert.Issuer)
		}
	} else {
		fmt.Fprintf(&b, "[⚠] 非 HTTPS，未检查 TLS 证书\n")
	}

	// 安全响应头缺失检查
	secHeaders := []struct{ name, desc string }{
		{"Strict-Transport-Security", "HSTS 强制 HTTPS"},
		{"Content-Security-Policy", "CSP 内容安全策略"},
		{"X-Content-Type-Options", "防 MIME 嗅探"},
		{"X-Frame-Options", "防点击劫持"},
		{"Referrer-Policy", "Referrer 策略"},
	}
	for _, h := range secHeaders {
		if v := resp.headerValue(h.name); v != "" {
			fmt.Fprintf(&b, "[✓] %s: %s\n", h.name, truncate(v))
		} else {
			fmt.Fprintf(&b, "[✗] 缺失 %s（%s）\n", h.name, h.desc)
		}
	}

	// CORS 宽松警告
	if v := resp.headerValue("Access-Control-Allow-Origin"); v == "*" {
		fmt.Fprintf(&b, "[⚠] CORS 允许所有来源（Access-Control-Allow-Origin: *）\n")
	}

	// 响应体敏感数据泄露扫描（仅提示存在性，不打印泄露内容）
	for _, p := range secretPatterns {
		if m := p.re.FindString(resp.body); m != "" {
			fmt.Fprintf(&b, "[⚠] 响应体疑似泄露 %s（命中 %q）\n", p.desc, truncate(m))
		}
	}
	return b.String(), nil
}

// shellSplit 按 shell 规则切分参数，支持单/双引号与 \ 转义
func shellSplit(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	var quote byte
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			cur.WriteByte(c)
			escaped = false
			continue
		}
		switch {
		case c == '\\' && quote != '\'': // 单引号内反斜杠为字面量
			escaped = true
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur.WriteByte(c)
			}
		case c == '\'' || c == '"':
			quote = c
		case c == ' ' || c == '\t' || c == '\n':
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("引号未闭合")
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args, nil
}

// curlParse 解析 curl 命令为 METHOD URL/Header/Body 标准格式
func curlParse(s string) (string, error) {
	args, err := shellSplit(s)
	if err != nil {
		return "", err
	}
	if len(args) == 0 || args[0] != "curl" {
		return "", fmt.Errorf("输入需以 curl 开头")
	}
	var method, u, user string
	var headers, datas []string
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-X", "--request":
			if i+1 < len(args) {
				method = args[i+1]
				i++
			}
		case "-H", "--header":
			if i+1 < len(args) {
				headers = append(headers, args[i+1])
				i++
			}
		case "-d", "--data", "--data-raw", "--data-ascii", "--data-urlencode":
			if i+1 < len(args) {
				datas = append(datas, args[i+1])
				i++
			}
		case "-u", "--user":
			if i+1 < len(args) {
				user = args[i+1]
				i++
			}
		case "-o", "--output": // 后跟文件名，需跳过
			if i+1 < len(args) {
				i++
			}
		case "-O", "--remote-name", "-i", "--include", "-s", "--silent", "-S", "--show-error",
			"-v", "--verbose", "-L", "--location", "-k", "--insecure", "-g", "--globoff",
			"--compressed", "-A", "--user-agent", "-e", "--referer", "-b", "--cookie", "-c", "--cookie-jar":
			// 展示/输出/重定向类 flag，忽略
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				continue // 未知 flag 忽略
			}
			if u == "" {
				u = a
			}
		}
	}
	if u == "" {
		return "", fmt.Errorf("未找到 URL")
	}
	if method == "" {
		if len(datas) > 0 {
			method = "POST"
		} else {
			method = "GET"
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", method, u)
	for _, h := range headers {
		fmt.Fprintf(&b, "%s\n", h)
	}
	if user != "" {
		// user:pass → Authorization Basic
		fmt.Fprintf(&b, "Authorization: Basic %s\n", base64.StdEncoding.EncodeToString([]byte(user)))
	}
	if len(datas) > 0 {
		b.WriteString("\n")
		b.WriteString(strings.Join(datas, "&"))
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// jwtVerify 验签 JWT（仅支持 HS256）：第一行密钥，第二行 token
func jwtVerify(s string) (string, error) {
	key, token, err := splitKeyData(s)
	if err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("JWT 需要 header.payload.signature 三段")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("header 解码失败：%v", err)
	}
	var head struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &head); err != nil {
		return "", fmt.Errorf("header JSON 解析失败：%v", err)
	}
	if head.Alg != "HS256" {
		return "", fmt.Errorf("仅支持 HS256 验签，当前 alg=%s", head.Alg)
	}
	// 用密钥重算签名并与 token 携带的签名比较（常数时间，防时序攻击）
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("signature 解码失败：%v", err)
	}
	valid := hmac.Equal(want, got)

	headerOut, _ := formatJSON(string(headerJSON))
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("payload 解码失败：%v", err)
	}
	payloadOut, _ := formatJSON(string(payloadJSON))

	mark := "[✓] signature valid"
	if !valid {
		mark = "[✗] signature invalid"
	}
	return fmt.Sprintf("header:\n%s\n\npayload:\n%s\n\n%s", headerOut, payloadOut, mark), nil
}

// httpStatusTable 常用状态码描述
var httpStatusTable = map[int]string{
	100: "Continue", 101: "Switching Protocols",
	200: "OK", 201: "Created", 202: "Accepted", 203: "Non-Authoritative Information",
	204: "No Content", 205: "Reset Content", 206: "Partial Content",
	300: "Multiple Choices", 301: "Moved Permanently", 302: "Found", 303: "See Other",
	304: "Not Modified", 307: "Temporary Redirect", 308: "Permanent Redirect",
	400: "Bad Request", 401: "Unauthorized", 402: "Payment Required", 403: "Forbidden",
	404: "Not Found", 405: "Method Not Allowed", 406: "Not Acceptable", 407: "Proxy Authentication Required",
	408: "Request Timeout", 409: "Conflict", 410: "Gone", 411: "Length Required",
	412: "Precondition Failed", 413: "Payload Too Large", 414: "URI Too Long",
	415: "Unsupported Media Type", 416: "Range Not Satisfiable", 417: "Expectation Failed",
	418: "I'm a teapot", 421: "Misdirected Request", 422: "Unprocessable Entity",
	425: "Too Early", 426: "Upgrade Required", 428: "Precondition Required",
	429: "Too Many Requests", 431: "Request Header Fields Too Large",
	451: "Unavailable For Legal Reasons",
	500: "Internal Server Error", 501: "Not Implemented", 502: "Bad Gateway",
	503: "Service Unavailable", 504: "Gateway Timeout", 505: "HTTP Version Not Supported",
}

// httpStatusLookup 状态码 → 描述与分类
func httpStatusLookup(s string) (string, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 100 || n > 599 {
		return "", fmt.Errorf("无效状态码 %q（需为 100-599 数字）", strings.TrimSpace(s))
	}
	desc, ok := httpStatusTable[n]
	if !ok {
		if t := http.StatusText(n); t != "" {
			desc = t
		} else {
			desc = "Unknown"
		}
	}
	var category string
	switch n / 100 {
	case 1:
		category = "信息"
	case 2:
		category = "成功"
	case 3:
		category = "重定向"
	case 4:
		category = "客户端错误"
	case 5:
		category = "服务器错误"
	}
	return fmt.Sprintf("%d %s\n%dxx %s", n, desc, n/100, category), nil
}
