package devtools

// devtools_data_test.go — 手写 base 算法与 Multi-Decode 的单元测试
// 全部为纯函数测试，无 IO / 无网络。

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ---- Base62 ----

func TestBase62RoundTrip(t *testing.T) {
	inputs := []string{"", "a", "hello world", "你好，世界", string([]byte{0, 1, 2, 255, 128, 64})}
	for _, in := range inputs {
		enc := base62Encode([]byte(in))
		dec, err := base62Decode(enc)
		if err != nil {
			t.Fatalf("base62Decode(%q) err: %v", enc, err)
		}
		if dec != in {
			t.Errorf("base62 roundtrip: encode(%q)=%q, decode=%q", in, enc, dec)
		}
	}
}

func TestBase62KnownVector(t *testing.T) {
	// 文档给出的已知向量
	if got := base62Encode([]byte{0x01}); got != "1" {
		t.Errorf("base62Encode({0x01}) = %q, want %q", got, "1")
	}
	// 十进制 0x3E7 = 999 → base62 "G7"（999 = 16*62 + 7 → 'G' + '7'，alphabet[16]='G'）
	if got := base62Encode([]byte{0x03, 0xE7}); got != "G7" {
		t.Errorf("base62Encode({0x03,0xE7}) = %q, want %q", got, "G7")
	}
}

func TestBase62Invalid(t *testing.T) {
	if _, err := base62Decode("a+b!"); err == nil {
		t.Error("base62Decode should reject invalid chars")
	}
}

// ---- Base91 ----

func TestBase91RoundTrip(t *testing.T) {
	inputs := []string{"", "a", "hello world", "你好，世界", "The quick brown fox jumps over the lazy dog",
		string([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 200, 255})}
	for _, in := range inputs {
		enc := base91Encode([]byte(in))
		dec, err := base91Decode(enc)
		if err != nil {
			t.Fatalf("base91Decode(%q) err: %v", enc, err)
		}
		if dec != in {
			t.Errorf("base91 roundtrip: encode(%q)=%q, decode=%q", in, enc, dec)
		}
	}
}

func TestBase91KnownVector(t *testing.T) {
	// basE91 官方已知向量（经独立 Python 实现交叉验证）：
	// encode("Hello Mother Fucker") == ">OwJh>=/zT]4|alL=7X<d,uC"
	// encode("a") == "GB"
	if got := base91Encode([]byte("Hello Mother Fucker")); got != ">OwJh>=/zT]4|alL=7X<d,uC" {
		t.Errorf("base91Encode(Hello Mother Fucker) = %q, want %q", got, ">OwJh>=/zT]4|alL=7X<d,uC")
	}
	if got := base91Encode([]byte("a")); got != "GB" {
		t.Errorf("base91Encode(a) = %q, want %q", got, "GB")
	}
	dec, err := base91Decode(">OwJh>=/zT]4|alL=7X<d,uC")
	if err != nil {
		t.Fatalf("base91Decode vector err: %v", err)
	}
	if dec != "Hello Mother Fucker" {
		t.Errorf("base91Decode = %q, want %q", dec, "Hello Mother Fucker")
	}
	// 表外字符（如空格）必须报错，防止 multiDecode 误剥层
	if _, err := base91Decode("hello world"); err == nil {
		t.Error("base91Decode should reject out-of-table chars (space)")
	}
}

// ---- Multi-Decode ----

func TestMultiDecodeLayers(t *testing.T) {
	// 构造 3 层套娃：hello → base64 → base64 → base64
	// （用同一种编码避免中间层被更靠前的解码器抢先误剥，保证稳定 3 层）
	plain := "hello devtools"
	l1 := base64EncodePlain(plain)
	l2 := base64EncodePlain(l1)
	l3 := base64EncodePlain(l2)

	steps, err := multiDecode(l3)
	if err != nil {
		t.Fatalf("multiDecode err: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 layers, got %d: %v", len(steps), steps)
	}
	// 最后一层的中间结果应还原为明文
	last := steps[len(steps)-1]
	if !strings.Contains(last, plain) {
		t.Errorf("last layer should contain plaintext %q, got %q", plain, last)
	}
}

func TestMultiDecodeNotEncoded(t *testing.T) {
	if _, err := multiDecode("plain text not encoded"); err == nil {
		t.Error("multiDecode should error on plain text")
	}
}

// 测试辅助：直接用标准库编码，避免依赖 builtinTools 里的 Tool.Run 包装
func base64EncodePlain(s string) string {
	out, _ := builtinTools[6].Run(s) // Base64 Encode
	return out
}

func base32EncodePlain(s string) string {
	out, _ := builtinTools[2].Run(s) // Base32 Encode
	return out
}

// ---- Tools 区：开发工具 ----

func TestJWTDecode(t *testing.T) {
	// jwt.io 标准示例 token
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	out, err := jwtDecode(token)
	if err != nil {
		t.Fatalf("jwtDecode err: %v", err)
	}
	for _, want := range []string{`"alg": "HS256"`, `"sub": "1234567890"`, `"name": "John Doe"`} {
		if !strings.Contains(out, want) {
			t.Errorf("jwtDecode output missing %q, got:\n%s", want, out)
		}
	}
}

func TestJWTDecodeInvalid(t *testing.T) {
	if _, err := jwtDecode("only.two.parts.ok.extra"); err == nil {
		t.Error("jwtDecode should reject token with != 3 segments")
	}
	if _, err := jwtDecode("!!.."); err == nil {
		t.Error("jwtDecode should reject undecodable header")
	}
}

func TestNumConvert(t *testing.T) {
	out, err := numConvert("0x1F")
	if err != nil {
		t.Fatalf("numConvert err: %v", err)
	}
	for _, want := range []string{"dec: 31", "hex: 0x1F", "bin: 0b11111", "oct: 0o37"} {
		if !strings.Contains(out, want) {
			t.Errorf("numConvert output missing %q, got:\n%s", want, out)
		}
	}
	if _, err := numConvert("0xGG"); err == nil {
		t.Error("numConvert should reject invalid hex")
	}
}

func TestUnicodeEscapeUnescape(t *testing.T) {
	// escape：非 ASCII / 控制字符 → 字面 \uXXXX（6 字符）
	if got := unicodeEscape("中"); got != "\\u4E2D" {
		t.Errorf("unicodeEscape(中) = %q, want %q", got, "\\u4E2D")
	}
	if got := unicodeEscape("a\nb"); got != "a\\u000Ab" {
		t.Errorf("unicodeEscape(a\\nb) = %q, want %q", got, "a\\u000Ab")
	}
	// unescape：字面反斜杠序列还原
	got, err := unicodeUnescape("\\u4E2D\\x41")
	if err != nil {
		t.Fatalf("unicodeUnescape err: %v", err)
	}
	if got != "中A" {
		t.Errorf("unicodeUnescape = %q, want %q", got, "中A")
	}
	// escape → unescape roundtrip
	orig := "Hello 世界 \t \n 123"
	back, err := unicodeUnescape(unicodeEscape(orig))
	if err != nil {
		t.Fatalf("roundtrip err: %v", err)
	}
	if back != orig {
		t.Errorf("roundtrip: %q != %q", back, orig)
	}
}

func TestUUIDV4(t *testing.T) {
	u1, err := uuidV4()
	if err != nil {
		t.Fatalf("uuidV4 err: %v", err)
	}
	if len(u1) != 36 {
		t.Errorf("uuid length = %d, want 36", len(u1))
	}
	if u1[14] != '4' { // 版本位固定为 4（第 13 组字符，下标 14）
		t.Errorf("uuid version = %c, want 4", u1[14])
	}
	u2, _ := uuidV4()
	if u1 == u2 {
		t.Error("two uuidV4 calls should differ")
	}
}

func TestUnixTimeConvert(t *testing.T) {
	// 已知向量：epoch 0 → 1970-01-01（utc 行不依赖本地时区）
	out, err := unixTimeConvert("0")
	if err != nil {
		t.Fatalf("unixTimeConvert err: %v", err)
	}
	if !strings.Contains(out, "unix seconds: 0") || !strings.Contains(out, "utc:   1970-01-01 00:00:00") {
		t.Errorf("epoch 0 output wrong:\n%s", out)
	}
	// 毫秒（13 位）自动识别为 milliseconds，UTC 与同刻秒时间一致
	out, err = unixTimeConvert("1715400000000")
	if err != nil || !strings.Contains(out, "unix milliseconds: 1715400000000") {
		t.Fatalf("ms detection failed (err %v):\n%s", err, out)
	}
	wantUTC := time.Unix(1715400000, 0).UTC().Format("2006-01-02 15:04:05")
	if !strings.Contains(out, "utc:   "+wantUTC) {
		t.Errorf("ms→utc wrong:\n%s", out)
	}
	// 日期字符串 → 时间戳（本地时区解析，用同一时区算期望值）
	tm, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-05-11 08:00:00", time.Local)
	want := fmt.Sprintf("local: 2024-05-11 08:00:00\nunix:  %d", tm.Unix())
	out, err = unixTimeConvert("2024-05-11 08:00:00")
	if err != nil || out != want {
		t.Errorf("date→unix = %q, want %q (err %v)", out, want, err)
	}
	// 非法输入
	if _, err := unixTimeConvert("hello"); err == nil {
		t.Error("unixTimeConvert should reject garbage")
	}
}

func TestTextStats(t *testing.T) {
	// 9 字符 / 9 字节 / 5 词 / 3 行 / 3 去重行
	out := textStats("a b\nc d\ne")
	for _, want := range []string{"chars: 9", "bytes: 9", "words: 5", "lines: 3", "unique lines: 3"} {
		if !strings.Contains(out, want) {
			t.Errorf("textStats missing %q, got:\n%s", want, out)
		}
	}
	// 中文按 rune 计字符、按字节计字节
	if out := textStats("中文"); !strings.Contains(out, "chars: 2") || !strings.Contains(out, "bytes: 6") {
		t.Errorf("CJK stats got:\n%s", out)
	}
	// 空输入
	if out := textStats(""); !strings.Contains(out, "chars: 0") || !strings.Contains(out, "lines: 0") {
		t.Errorf("empty stats got:\n%s", out)
	}
}

// ---- Hash 区扩展：SHA 变体 / 校验和 / HMAC ----

func TestSHAFamily(t *testing.T) {
	// 空串的官方向量（NIST）
	cases := map[string]struct{ got, want string }{
		"SHA224":     {sha224Hex(""), "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"},
		"SHA384":     {sha384Hex(""), "38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"},
		"SHA512/224": {sha512224Hex(""), "6ed0dd02806fa89e25de060c19d3ac86cabb87d6a0ddd05c333b84f4"},
		"SHA512/256": {sha512256Hex(""), "c672b8d1ef56ed28ab87c3622c5114069bdd3ad7b8f9737498d0c01ecef0967a"},
	}
	for name, c := range cases {
		if c.got != c.want {
			t.Errorf("%s(\"\") = %q, want %q", name, c.got, c.want)
		}
	}
}

func TestChecksums(t *testing.T) {
	// 校验和官方向量：CRC32 用 IEEE check value，CRC64 用 ECMA/XZ 值，FNV 用 RFC 向量，Adler-32 用维基百科示例
	if got := crc32Hex("123456789"); got != "cbf43926" {
		t.Errorf("crc32(123456789) = %q, want %q", got, "cbf43926")
	}
	if got := crc64Hex("123456789"); got != "995dc9bbdf1939fa" {
		t.Errorf("crc64(123456789) = %q, want %q", got, "995dc9bbdf1939fa")
	}
	if got := fnv1aHex("foobar"); got != "85944171f73967e8" {
		t.Errorf("fnv1a(foobar) = %q, want %q", got, "85944171f73967e8")
	}
	if got := adler32Hex("Wikipedia"); got != "11e60398" {
		t.Errorf("adler32(Wikipedia) = %q, want %q", got, "11e60398")
	}
}

func TestHMACSHA256(t *testing.T) {
	// RFC 4231 Test Case 1：key=0x0b×20，data="Hi There"
	key := string(bytes.Repeat([]byte{0x0b}, 20))
	if got := hmacSHA256(key, "Hi There"); got != "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7" {
		t.Errorf("hmacSHA256(RFC4231 TC1) = %q, want %q", got, "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7")
	}
	// 空数据也应有固定输出
	if got := hmacSHA256("key", ""); len(got) != 64 {
		t.Errorf("hmacSHA256(key,\"\") length = %d, want 64", len(got))
	}
}

func TestSplitKeyData(t *testing.T) {
	key, data, err := splitKeyData("secret\nhello world")
	if err != nil {
		t.Fatalf("splitKeyData err: %v", err)
	}
	if key != "secret" || data != "hello world" {
		t.Errorf("splitKeyData = (%q, %q), want (secret, hello world)", key, data)
	}
	// 空密钥报错
	if _, _, err := splitKeyData("\nhello"); err == nil {
		t.Error("splitKeyData should reject empty key")
	}
	// 只有一行密钥、无数据
	key, data, _ = splitKeyData("secret")
	if data != "" {
		t.Errorf("splitKeyData single-line data = %q, want empty", data)
	}
}

// ---- Encrypt 区：AES-256-GCM ----

func TestAESRoundTrip(t *testing.T) {
	// 口令加密 → 解密还原
	plain := "Hello DevTools, 你好世界"
	ct, err := aesEncrypt("passphrase", []byte(plain))
	if err != nil {
		t.Fatalf("aesEncrypt err: %v", err)
	}
	pt, err := aesDecrypt("passphrase", ct)
	if err != nil {
		t.Fatalf("aesDecrypt err: %v", err)
	}
	if pt != plain {
		t.Errorf("AES roundtrip = %q, want %q", pt, plain)
	}
}

func TestAESWrongKey(t *testing.T) {
	ct, err := aesEncrypt("correct", []byte("secret"))
	if err != nil {
		t.Fatalf("aesEncrypt err: %v", err)
	}
	// 错误口令必须解密失败（GCM 认证），而不是吐乱码
	if _, err := aesDecrypt("wrong", ct); err == nil {
		t.Error("aesDecrypt should fail with wrong passphrase")
	}
}

func TestAESInvalidInput(t *testing.T) {
	// 非 hex 密文报错
	if _, err := aesDecrypt("key", "not hex!!"); err == nil {
		t.Error("aesDecrypt should reject non-hex ciphertext")
	}
	// 过短密文报错（不足 nonce 长度）
	if _, err := aesDecrypt("key", "abcd"); err == nil {
		t.Error("aesDecrypt should reject too-short ciphertext")
	}
}
