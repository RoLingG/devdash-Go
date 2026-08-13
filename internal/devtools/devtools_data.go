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
	"math/big"
	"net/url"
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
			b.WriteByte(c) // 换行/制表是正常文本结构，保留
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

// wrapLong 长字符串按固定字符数换行，防超宽折行覆盖状态栏
func wrapLong(s string) string {
	const step = 64
	r := []rune(s)
	var b strings.Builder
	for i := 0; i < len(r); i += step {
		if i > 0 {
			b.WriteByte('\n')
		}
		end := i + step
		if end > len(r) {
			end = len(r)
		}
		b.WriteString(string(r[i:end]))
	}
	return b.String()
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
	// 纯数字 → 当作 Unix 时间戳，按位数自动识别秒/毫秒/微秒
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
	// 日期字符串 → 按常见格式解析为本地时间
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
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("密文过短")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密钥错误或密文被篡改）：%v", err)
	}
	return string(plain), nil
}
