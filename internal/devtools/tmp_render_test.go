package devtools

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"cava_go/internal/ui"
)

// 临时诊断：对比短输出(MD5)与长输出(SHA512)的卡片高度，验证"换行是否导致卡片增行"
func TestTmpRenderRows(t *testing.T) {
	// 短输出工具 vs 长输出工具
	tests := []struct {
		name string
		idx  int
	}{
		{"MD5(短输出,1行)", 18},    // Hash 分组第 1 个：MD5
		{"SHA512(长输出,折多行)", 21}, // Hash 分组第 4 个：SHA512
	}
	for _, tt := range tests {
		h := 24
		w := 110
		m := &Model{}
		m.UpdateSize(w, h)
		m.Init()
		m.inputValue = "yeschief"
		m.cursor = tt.idx
		m.compute()

		card := m.View()
		cardRows := strings.Count(card, "\n") + 1
		t.Logf("== %s  result=%d字符  outScroll=%d  cardRows=%d (contentH=%d)", tt.name, len(m.result), m.outScroll, cardRows, h-7)

		tabBar := ui.RenderTabBar(ui.TabDevTools, w)
		statusBar := ui.RenderStatusBar(ui.TabDevTools, w)
		contentHeight := h - 3
		lines := strings.Count(card, "\n")
		padding := contentHeight - lines
		if padding < 0 {
			padding = 0
		}
		content := card + strings.Repeat("\n", padding)
		full := tabBar + "\n" + content + statusBar
		total := strings.Count(full, "\n") + 1
		t.Logf("  main.go padding=%d totalRows=%d (terminal=%d 溢出=%d)", padding, total, h, total-h)
		parts := strings.Split(full, "\n")
		t.Logf("  倒数第2片段: %q", parts[len(parts)-2])
		t.Logf("  倒数第1片段: %q", parts[len(parts)-1])

		// 探针：逐行 dump 卡片 + 内容行数，定位多出的行
		cardLines := strings.Split(card, "\n")
		t.Logf("  [卡片行数=%d 内容=?]", len(cardLines))
		for i, l := range cardLines {
			t.Logf("  [卡片行%02d] %q", i, l)
		}
	}

	// 探针：隔离 Card/JoinHorizontal 行数行为
	card17 := ui.Card("DevTools", strings.Join(make([]string, 17), "\n"), ui.ColAccent, 110)
	card18 := ui.Card("DevTools", strings.Join(make([]string, 18), "\n"), ui.ColAccent, 110)
	t.Logf("Card(17行内容)=%d行  Card(18行内容)=%d行", strings.Count(card17, "\n")+1, strings.Count(card18, "\n")+1)
	col17 := strings.Join(make([]string, 17), "\n")
	joined := lipgloss.JoinHorizontal(lipgloss.Top, col17, "  ", col17)
	t.Logf("JoinHorizontal(17,17)=%d行", strings.Count(joined, "\n")+1)

	// 探针：重建右列，数 output 换行产生的行数
	m2 := &Model{}
	m2.UpdateSize(110, 24)
	m2.Init()
	m2.inputValue = "yeschief"
	m2.cursor = 21
	m2.compute()
	contentH := m2.height - 7
	innerW := 110 - 4
	rightW := innerW - 26 - 2
	outLines := strings.Split(lipgloss.NewStyle().Width(rightW-2).Render(m2.result), "\n")
	t.Logf("SHA512重建: contentH=%d rightW=%d outLines=%d result=%d字符", contentH, rightW, len(outLines), len(m2.result))
	var right []string
	right = append(right, "Input", "yeschief", "", "Output")
	for _, l := range outLines {
		right = append(right, "  "+l)
	}
	t.Logf("  right切片(加输出后)=%d", len(right))
	for len(right) < contentH {
		right = append(right, "")
	}
	rightCol := strings.Join(right[:contentH], "\n")
	t.Logf("  rightCol=%d行  right[:17]长度=%d", strings.Count(rightCol, "\n")+1, len(right[:contentH]))

	// 探针：重建左列，数 SHA512 光标窗口下的行数
	m3 := &Model{}
	m3.UpdateSize(110, 24)
	m3.Init()
	m3.inputValue = "yeschief"
	m3.cursor = 21
	m3.compute()
	contentH3 := m3.height - 7
	// 按 View 里的 rows 构造逻辑重建：Group 标题行 + 工具行
	type rowKind int
	const (
		rowGroup rowKind = iota
		rowTool
	)
	type displayRow struct {
		kind rowKind
		text string
		idx  int
	}
	var rows []displayRow
	var curGroup string
	for i, t := range m3.tools {
		if t.Group != curGroup {
			curGroup = t.Group
			rows = append(rows, displayRow{kind: rowGroup, text: t.Group})
		}
		rows = append(rows, displayRow{kind: rowTool, text: t.Name, idx: i})
	}
	cursorRow := 0
	for i, r := range rows {
		if r.kind == rowTool && r.idx == m3.cursor {
			cursorRow = i
			break
		}
	}
	window := contentH3 - 1
	start := 0
	if cursorRow >= window {
		start = cursorRow - window + 1
	}
	end := start + window
	if end > len(rows) {
		end = len(rows)
	}
	// 左列 = 标题行 + start..end 的行，随后 slice 到 contentH
	leftCol := []string{"◀ Tools ▸"}
	for i := start; i < end; i++ {
		r := rows[i]
		if r.kind == rowGroup {
			leftCol = append(leftCol, "  "+r.text)
		} else {
			leftCol = append(leftCol, "  "+r.text)
		}
	}
	t.Logf("  rows总数=%d  cursorRow=%d  window=%d  start=%d  end=%d  左列内容行=%d  left[:17]=%d",
		len(rows), cursorRow, window, start, end, len(leftCol), len(leftCol[:contentH3]))
	t.Logf("  左列 slice 后 =%d 行", len(leftCol[:contentH3]))

	// 探针：用与 View() 完全相同的样式重建左右两列，JoinHorizontal 后数行数
	// 验证 content 究竟 17 还是 18 行（定位多出的那一行来自哪一列）
	m4 := &Model{}
	m4.UpdateSize(110, 24)
	m4.Init()
	m4.inputValue = "yeschief"
	m4.cursor = 21
	m4.compute()
	ch := m4.height - 7
	innerW4 := 110 - 4
	rW := innerW4 - 26 - 2
	if rW < 20 {
		rW = 20
	}
	// 右列：与 View() 逐行一致
	right4 := []string{lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Input")}
	right4 = append(right4, lipgloss.NewStyle().Foreground(ui.ColText).Render("  "+ui.Truncate(m4.inputValue, rW-2)))
	right4 = append(right4, "")
	right4 = append(right4, lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Output"))
	outLines4 := strings.Split(lipgloss.NewStyle().Width(rW-2).Render(m4.result), "\n")
	for _, l := range outLines4 {
		right4 = append(right4, "  "+l)
	}
	for len(right4) < ch {
		right4 = append(right4, "")
	}
	rightCol4 := strings.Join(right4[:ch], "\n")
	// 左列：直接复用 View() 同款行构造
	left4 := []string{lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render("◀ Tools ▸")}
	for i := start; i < end; i++ {
		r := rows[i]
		if r.kind == rowGroup {
			left4 = append(left4, lipgloss.NewStyle().Foreground(ui.ColMuted).Bold(true).Render("  "+r.text))
		} else {
			left4 = append(left4, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+r.text))
		}
	}
	for len(left4) < ch {
		left4 = append(left4, "")
	}
	leftCol4 := strings.Join(left4[:ch], "\n")
	content4 := lipgloss.JoinHorizontal(lipgloss.Top, leftCol4, "  ", rightCol4)
	t.Logf("View()同款重建: 左col=%d行 右col=%d行  join后=%d行  →Card应为%d行",
		strings.Count(leftCol4, "\n")+1, strings.Count(rightCol4, "\n")+1,
		strings.Count(content4, "\n")+1, strings.Count(content4, "\n")+5)
	t.Logf("  outLines4=%d条  (rightW-2=%d)", len(outLines4), rW-2)

	// 探针：剥离 ANSI 后测量真实渲染卡片的每一行视觉宽度，验证是否超宽被 Card 重折
	for _, tt := range tests {
		mm := &Model{}
		mm.UpdateSize(110, 24)
		mm.Init()
		mm.inputValue = "yeschief"
		mm.cursor = tt.idx
		mm.compute()
		card := mm.View()
		// 剥离 ANSI
		cleaned := ""
		ink := false
		for _, r := range card {
			if r == '\x1b' {
				ink = true
				continue
			}
			if ink {
				if r == 'm' {
					ink = false
				}
				continue
			}
			cleaned += string(r)
		}
		lines := strings.Split(cleaned, "\n")
		maxW := 0
		for _, l := range lines {
			w := lipgloss.Width(l)
			if w > maxW {
				maxW = w
			}
		}
		t.Logf("[ANSI剥离] %s: 卡%d行 最大行宽=%d (Card inner=106)", tt.name, len(lines), maxW)
		// 打印内容区每一行(去掉边框首尾)的可视宽度和截断内容，定位多出的行
		for i, l := range lines {
			if i == 0 || i == len(lines)-1 {
				continue
			}
			inner := strings.TrimRight(l, " ")
			w := lipgloss.Width(inner)
			disp := inner
			if len(disp) > 50 {
				disp = disp[:50] + "…"
			}
			t.Logf("  [%s 行%02d] 宽=%d %q", tt.name, i, w, disp)
		}
	}

	// 探针：逐字复制 View()，拦截传给 Card 的 content，数它的行数与各列行数
	mv := &Model{}
	mv.UpdateSize(110, 24)
	mv.Init()
	mv.inputValue = "yeschief"
	mv.cursor = 21
	mv.compute()

	contentHv := mv.height - 7
	innerWv := 110 - 4
	leftWv := 26
	rightWv := innerWv - leftWv - 2
	if rightWv < 20 {
		rightWv = 20
	}
	// 左列重建(带样式)
	type rk int
	const (
		rkGroup rk = iota
		rkTool
	)
	type dr struct {
		kind rk
		text string
		idx  int
	}
	var rsv []dr
	var cg string
	for i, t := range mv.tools {
		if t.Group != cg {
			cg = t.Group
			rsv = append(rsv, dr{kind: rkGroup, text: t.Group})
		}
		rsv = append(rsv, dr{kind: rkTool, text: t.Name, idx: i})
	}
	cursorRowv := 0
	for i, r := range rsv {
		if r.kind == rkTool && r.idx == mv.cursor {
			cursorRowv = i
			break
		}
	}
	windowv := contentHv - 1
	startv := 0
	if cursorRowv >= windowv {
		startv = cursorRowv - windowv + 1
	}
	endv := startv + windowv
	if endv > len(rsv) {
		endv = len(rsv)
	}
	leftv := []string{lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render("◀ Tools ▸")}
	for i := startv; i < endv; i++ {
		r := rsv[i]
		switch r.kind {
		case rkGroup:
			leftv = append(leftv, lipgloss.NewStyle().Foreground(ui.ColMuted).Bold(true).Render("  "+r.text))
		case rkTool:
			idx := fmt.Sprintf("%2d", r.idx+1)
			if r.idx == mv.cursor {
				leftv = append(leftv, "▸ "+lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true).Render(idx+" "+r.text))
			} else {
				leftv = append(leftv, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  "+idx+" "+r.text))
			}
		}
	}
	for len(leftv) < contentHv {
		leftv = append(leftv, "")
	}
	leftColv := strings.Join(leftv[:contentHv], "\n")
	// 右列重建
	rightv := []string{lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Input")}
	if mv.inputValue == "" {
		rightv = append(rightv, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (press / to input)"))
	} else {
		rightv = append(rightv, lipgloss.NewStyle().Foreground(ui.ColText).Render("  "+ui.Truncate(mv.inputValue, rightWv-2)))
	}
	rightv = append(rightv, "")
	rightv = append(rightv, lipgloss.NewStyle().Bold(true).Foreground(ui.ColPrimary).Render("Output"))
	if mv.resultErr != nil {
		rightv = append(rightv, lipgloss.NewStyle().Foreground(ui.ColRed).Render("  ✗ "+mv.resultErr.Error()))
	} else if mv.inputValue == "" {
		rightv = append(rightv, lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  (empty)"))
	} else {
		outLinesv := strings.Split(lipgloss.NewStyle().Width(rightWv-2).Render(mv.result), "\n")
		availv := contentHv - len(rightv)
		if availv < 1 {
			availv = 1
		}
		totalv := len(outLinesv)
		if mv.outScroll > totalv-availv {
			mv.outScroll = totalv - availv
		}
		if mv.outScroll < 0 {
			mv.outScroll = 0
		}
		outEndv := mv.outScroll + availv
		if outEndv > totalv {
			outEndv = totalv
		}
		if totalv > availv {
			rightv = append(rightv, lipgloss.NewStyle().Foreground(ui.ColMuted).Render(fmt.Sprintf("  [pgup/pgdn %d-%d/%d]", mv.outScroll+1, outEndv, totalv)))
		}
		for _, l := range outLinesv[mv.outScroll:outEndv] {
			rightv = append(rightv, "  "+l)
		}
	}
	for len(rightv) < contentHv {
		rightv = append(rightv, "")
	}
	rightColv := strings.Join(rightv[:contentHv], "\n")
	contentv := lipgloss.JoinHorizontal(lipgloss.Top, leftColv, "  ", rightColv)
	t.Logf("View()逐字复制: left切片=%d right切片=%d  content行=%d", len(leftv[:contentHv]), len(rightv[:contentHv]), strings.Count(contentv, "\n")+1)
	// 剥离 ANSI 打印 content 每行宽度
	cvClean := ""
	inkv := false
	for _, r := range contentv {
		if r == '\x1b' {
			inkv = true
			continue
		}
		if inkv {
			if r == 'm' {
				inkv = false
			}
			continue
		}
		cvClean += string(r)
	}
	for i, l := range strings.Split(cvClean, "\n") {
		w := lipgloss.Width(l)
		disp := strings.TrimRight(l, " ")
		if len(disp) > 40 {
			disp = disp[:40] + "…"
		}
		t.Logf("  [content行%02d] 宽=%d %q", i, w, disp)
	}
}
