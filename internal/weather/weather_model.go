package weather

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"cava_go/internal/component"
	"cava_go/internal/ui"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model 天气面板模块状态
type Model struct {
	data    *WttrResponse
	width   int
	height  int
	loaded  bool
	loading bool
	err     error
	city    string
	input   component.InputModel
	scroll  int
	spinner spinner.Model
}

func (m *Model) Init(defaultCity string) tea.Cmd {
	if defaultCity == "" {
		defaultCity = "Beijing"
	}
	m.city = defaultCity
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	m.loading = true
	return tea.Batch(FetchFromCityCmd(defaultCity), m.spinner.Tick)
}

func (m *Model) UpdateSize(w, h int) { m.width = w; m.height = h }

// SetRecent 设置最近记录列表（转发给 InputModel）
func (m *Model) SetRecent(items []string) { m.input.SetRecent(items) }

// InputActive 输入框是否活跃
func (m *Model) InputActive() bool { return m.input.Active }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if !m.loading {
			return nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
	case Msg:
		if msg.Err != nil {
			m.err = msg.Err
			m.loaded = true
			m.loading = false
			m.input.Active = false
			return nil
		}
		m.data = msg.Data
		m.loaded = true
		m.loading = false
		m.err = nil
		m.input.Active = false
	case tea.PasteMsg:
		if m.input.Active {
			return m.input.Update(msg, nil)
		}
		return nil
	case tea.KeyPressMsg:
		if m.input.Active {
			return tea.Batch(
				m.input.Update(msg, func(city string) func() tea.Msg {
					if city != "" {
						m.city = city
						m.loading = true
						m.err = nil
						return tea.Batch(FetchFromCityCmd(city), m.spinner.Tick)
					}
					return nil
				}),
				ui.UpdateCfgCmd("city", m.city),
			)
		}
		switch msg.String() {
		case "ctrl+r":
			if m.city != "" {
				m.loading = true
				m.err = nil
				return tea.Batch(FetchFromCityCmd(m.city), m.spinner.Tick)
			}
		case "/":
			m.input.Prompt = "City:"
			m.input.Open(m.city)
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			viewH := m.height - 7
			if viewH < 3 {
				viewH = 3
			}
			maxScroll := 50 - viewH
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scroll < maxScroll {
				m.scroll++
			}
		case "home":
			m.scroll = 0
		case "end":
			m.scroll = 1 << 30 // View 中会自动 clamp 到有效范围
		}
	}
	return nil
}

// View 渲染视图
func (m *Model) View() string {
	cardWidth := m.width
	if cardWidth < 40 {
		cardWidth = 40
	}

	// 输入框卡片
	if m.input.Active {
		return ui.RenderInputCard(ui.InputCardOpts{
			Title:       "Change City",
			Prompt:      m.input.Prompt,
			Value:       m.input.Value,
			Cursor:      m.input.Cursor,
			CardWidth:   cardWidth,
			RecentItems: m.input.RecentItems,
			RecentIdx:   m.input.RecentIdx(),
		})
	}

	// 加载中
	if m.loading {
		loadingContent := lipgloss.NewStyle().Foreground(ui.ColAccent).Render(m.spinner.View() + " Fetching...")
		if m.city != "" {
			loadingContent = lipgloss.NewStyle().Foreground(ui.ColSecondary).Render("📍 "+m.city) + "\n\n" + loadingContent
		}
		return ui.Card("Weather", loadingContent, ui.ColAccent, cardWidth)
	}

	// 错误
	if m.err != nil {
		errContent := lipgloss.NewStyle().Foreground(ui.ColRed).Render("✗ "+m.err.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(ui.ColMuted).Render("'R' retry, '/' city")
		return ui.Card("Weather", errContent, ui.ColRed, cardWidth)
	}

	// 未加载
	if !m.loaded || m.data == nil {
		emptyContent := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("  Press '/' to enter city")
		return ui.Card("Weather", emptyContent, ui.ColMuted, cardWidth)
	}

	var sb strings.Builder

	// 当前天气
	if len(m.data.CurrentCondition) > 0 {
		cc := m.data.CurrentCondition[0]
		desc := ""
		if len(cc.WeatherDesc) > 0 {
			desc = cc.WeatherDesc[0].Value
		}
		location := m.city
		if len(m.data.NearestArea) > 0 {
			area := m.data.NearestArea[0]
			if len(area.AreaName) > 0 && area.AreaName[0].Value != "" {
				location = area.AreaName[0].Value
			}
			if len(area.Country) > 0 && area.Country[0].Value != "" {
				location += ", " + area.Country[0].Value
			}
		}

		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ui.ColAccent).Render("  📍 "+location) + "\n\n")

		icon := GetWeatherIcon(desc)
		iconLines := strings.Split(icon, "\n")
		tempLine := lipgloss.NewStyle().Foreground(ui.ColGreen).Bold(true).Render(cc.TempC + "°C")
		feelsLike := lipgloss.NewStyle().Foreground(ui.ColMuted).Render("Feels like " + cc.FeelsLikeC + "°C")
		descLine := lipgloss.NewStyle().Foreground(ui.ColSecondary).Bold(true).Render(desc)
		detailStyle := lipgloss.NewStyle().Foreground(ui.ColMuted)
		details := []string{
			detailStyle.Render("  💧 Humidity:  " + cc.Humidity + "%"),
			detailStyle.Render("  💨 Wind:     " + cc.WindspeedKmph + " km/h"),
			detailStyle.Render("  👁  Visibility: " + cc.Visibility + " km"),
			detailStyle.Render("  ☀  UV Index:  " + cc.UVIndex),
		}
		leftSide := lipgloss.NewStyle().Foreground(ui.ColAccent).Render(strings.Join(iconLines, "\n"))
		rightSide := descLine + "\n\n" + tempLine + "\n" + feelsLike + "\n\n" + strings.Join(details, "\n")
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftSide, "  ", rightSide))
	}

	// 预报
	if len(m.data.Weather) > 0 {
		sb.WriteString("\n\n" + lipgloss.NewStyle().Foreground(ui.ColAccent).Bold(true).Render("  ─── 3-Day Forecast ───") + "\n\n")

		getIcon := func(desc string) string {
			descLower := strings.ToLower(desc)
			if strings.Contains(descLower, "cloud") || strings.Contains(descLower, "overcast") {
				return "☁"
			} else if strings.Contains(descLower, "rain") || strings.Contains(descLower, "drizzle") {
				return "🌧"
			} else if strings.Contains(descLower, "snow") {
				return "❄"
			} else if strings.Contains(descLower, "thunder") {
				return "⛈"
			} else if strings.Contains(descLower, "fog") || strings.Contains(descLower, "mist") {
				return "🌫"
			}
			return "☀"
		}

		getTempColor := func(temp int) color.Color {
			if temp >= 30 {
				return ui.ColRed
			} else if temp >= 20 {
				return ui.ColAccent
			} else if temp >= 10 {
				return ui.ColGreen
			}
			return ui.ColBlue
		}

		for i, day := range m.data.Weather {
			if i >= 3 {
				break
			}
			dateStr := day.Date
			if i == 0 {
				dateStr = "Today"
			} else if i == 1 {
				dateStr = "Tomorrow"
			} else {
				if t, err := time.Parse("2006-01-02", day.Date); err == nil {
					dateStr = t.Weekday().String()[:3]
				}
			}
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ui.ColSecondary).Render("  "+dateStr) + "\n")

			timeRow := "  "
			tempRow := "  "
			iconRow := "  "
			for _, h := range day.Hourly {
				timeStr := h.Time
				hour := 0
				if len(timeStr) >= 3 {
					hour, _ = strconv.Atoi(timeStr[:len(timeStr)-2])
				} else {
					hour, _ = strconv.Atoi(timeStr)
				}
				timeStr = fmt.Sprintf("%02d:00", hour)
				temp, _ := strconv.Atoi(h.TempC)
				desc := ""
				if len(h.WeatherDesc) > 0 {
					desc = h.WeatherDesc[0].Value
				}
				timeRow += fmt.Sprintf("%-8s", timeStr)
				tempRow += lipgloss.NewStyle().Foreground(getTempColor(temp)).Render(fmt.Sprintf("%-8s", fmt.Sprintf("%d°C", temp)))
				iconRow += fmt.Sprintf("%-8s", getIcon(desc))
			}
			sb.WriteString(timeRow + "\n")
			sb.WriteString(tempRow + "\n")
			sb.WriteString(iconRow + "\n")
		}
	}

	// 滚动
	fullContent := sb.String()
	contentLines := strings.Split(fullContent, "\n")
	// Tab 栏：1 行; 状态栏：2 行; Card 边框 + 标题：4 行
	viewH := m.height - 7
	if viewH < 3 {
		viewH = 3
	}
	totalLines := len(contentLines)
	// 防止过多滚动导致视图溢出
	if m.scroll > totalLines-viewH {
		m.scroll = totalLines - viewH
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	end := m.scroll + viewH
	if end > totalLines {
		end = totalLines
	}
	visible := strings.Join(contentLines[m.scroll:end], "\n")
	return ui.Card("Weather", visible, ui.ColAccent, cardWidth)
}
