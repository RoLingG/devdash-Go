// ============================================================================
// mod_weather.go — 终端天气面板
//
// 功能：
//   - 从 wttr.in 获取天气数据（免费、无需 API Key）
//   - 支持用户输入城市名
//   - ASCII 天气动画
//   - 温度、湿度、风速信息
//   - r 键刷新
// ============================================================================

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// wttrResponse wttr.in JSON 响应结构
type wttrResponse struct {
	CurrentCondition []struct {
		TempC         string `json:"temp_C"`
		FeelsLikeC    string `json:"FeelsLikeC"`
		Humidity      string `json:"humidity"`
		WindspeedKmph string `json:"windspeedKmph"`
		WeatherDesc   []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
		Visibility string `json:"visibility"`
		UVIndex    string `json:"uvIndex"`
	} `json:"current_condition"`
	NearestArea []struct {
		AreaName []struct {
			Value string `json:"value"`
		} `json:"areaName"`
		Country []struct {
			Value string `json:"value"`
		} `json:"country"`
	} `json:"nearest_area"`
	Weather []struct {
		Date     string `json:"date"`
		MaxtempC string `json:"maxtempC"`
		MintempC string `json:"mintempC"`
		Hourly   []struct {
			Time        string `json:"time"`
			TempC       string `json:"tempC"`
			WeatherDesc []struct {
				Value string `json:"value"`
			} `json:"weatherDesc"`
		} `json:"hourly"`
	} `json:"weather"`
}

// weatherMsg 天气数据消息
type weatherMsg struct {
	data *wttrResponse
	err  error
}

// fetchWeatherFromCity 从 wttr.in 获取指定城市的天气（JSON 格式）
func fetchWeatherFromCity(city string) tea.Msg {
	url := "https://wttr.in/?format=j1"
	if city != "" {
		url = fmt.Sprintf("https://wttr.in/%s?format=j1", city)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return weatherMsg{err: fmt.Errorf("failed to fetch weather: %w", err)}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return weatherMsg{err: fmt.Errorf("failed to read response: %w", err)}
	}
	var data wttrResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return weatherMsg{err: fmt.Errorf("failed to parse weather data: %w", err)}
	}
	return weatherMsg{data: &data}
}

type weatherModel struct {
	data    *wttrResponse
	width   int
	height  int
	loaded  bool
	loading bool
	err     error
	city    string     // 用户输入的城市名
	input   inputModel // 通用输入组件
	scroll  int        // 滚动偏移
}

func (m *weatherModel) Init() tea.Cmd {
	// 默认使用 Beijing 作为初始城市
	m.city = "Beijing"
	m.loading = true
	return func() tea.Msg { return fetchWeatherFromCity("Beijing") }
}

func (m *weatherModel) UpdateSize(w, h int) { m.width = w; m.height = h }

func (m *weatherModel) Update(msg tea.Msg) (*weatherModel, tea.Cmd) {
	switch msg := msg.(type) {
	case weatherMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loaded = true
			m.loading = false
			m.input.active = false
			return m, nil
		}
		m.data = msg.data
		m.loaded = true
		m.loading = false
		m.err = nil
		m.input.active = false
	case tea.PasteMsg:
		// v2: 粘贴内容通过 PasteMsg 传递，不再是 KeyPressMsg
		if m.input.active {
			return m, m.input.Update(msg, nil)
		}
		return m, nil
	case tea.KeyPressMsg:
		// 输入模式下的按键处理
		if m.input.active {
			return m, m.input.Update(msg, func(city string) tea.Cmd {
				if city != "" {
					m.city = city
					m.loading = true
					m.err = nil
					return func() tea.Msg { return fetchWeatherFromCity(city) }
				}
				return nil
			})
		}

		// 非输入模式下的按键处理
		switch msg.String() {
		case "R":
			// 刷新天气
			if m.city != "" {
				m.loading = true
				m.err = nil
				return m, func() tea.Msg { return fetchWeatherFromCity(m.city) }
			}
		case "/":
			// 进入输入模式
			m.input.prompt = "City:"
			m.input.Open(m.city)
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			// 最大滚动 = 内容总行数 - 可视高度
			// 天气内容约 30 行，留余量用 50
			viewH := m.height - 6 // tab(1) + help(1) + 卡片开销(4)
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
		}
	}
	return m, nil
}

// getWeatherIcon 根据天气描述返回 ASCII 图标
func getWeatherIcon(desc string) string {
	desc = strings.ToLower(desc)
	switch {
	case strings.Contains(desc, "sunny") || strings.Contains(desc, "clear"):
		return `    \   /
     .-.
  ‒ (   ) ‒
     '-'
    /   \`
	case strings.Contains(desc, "partly cloudy"):
		return `   \  /
 _ /"".-.
   \_(   ).
   /(___(__)`
	case strings.Contains(desc, "cloudy") || strings.Contains(desc, "overcast"):
		return `     .-.
  .-(   ).
(___.__)__)`
	case strings.Contains(desc, "rain") || strings.Contains(desc, "drizzle"):
		return `     .-.
  .-(   ).
(___.__)__)
  ' ' ' '
 ' ' ' '`
	case strings.Contains(desc, "snow"):
		return `     .-.
  .-(   ).
(___.__)__)
  * * * *
 * * * *`
	case strings.Contains(desc, "thunder") || strings.Contains(desc, "storm"):
		return `     .-.
  .-(   ).
(___.__)__)
  ⚡' '⚡
 ' ' ' '`
	case strings.Contains(desc, "fog") || strings.Contains(desc, "mist"):
		return ` _ - _ - _
  _ - _ -
 _ - _ - _`
	default:
		return `     .-.
  .-(   ).
(___.__)__)`
	}
}

func (m *weatherModel) View() string {
	// 计算卡片宽度（自适应终端宽度，留出边距）
	cardWidth := m.width - 2
	if cardWidth < 40 {
		cardWidth = 40
	}
	//if cardWidth > 100 {
	//	cardWidth = 100
	//}

	// 输入框卡片
	if m.input.active {
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render("  " + m.input.prompt))
		sb.WriteString("\n")

		before := runeSubstr(m.input.value, 0, m.input.cursor)
		after := runeSubstr(m.input.value, m.input.cursor, runeLen(m.input.value))
		inputLine := "  > " + before + lipgloss.NewStyle().Foreground(colAccent).Render("|") + after
		sb.WriteString(lipgloss.NewStyle().Foreground(colText).Render(inputLine))
		sb.WriteString("\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("  Enter confirm  ←→ cursor  Home/End  Esc cancel"))
		return card("Change City", sb.String(), colSecondary, cardWidth)
	}

	// 加载中
	if m.loading {
		loadingContent := lipgloss.NewStyle().Foreground(colAccent).Render("⏳ Fetching...")
		if m.city != "" {
			loadingContent = lipgloss.NewStyle().Foreground(colSecondary).Render("📍 "+m.city) + "\n\n" + loadingContent
		}
		return card("Weather", loadingContent, colAccent, cardWidth)
	}

	// 错误
	if m.err != nil {
		errContent := lipgloss.NewStyle().Foreground(colRed).Render("✗ "+m.err.Error()) + "\n"
		errContent += lipgloss.NewStyle().Foreground(colMuted).Render("'r' retry, '/' city")
		return card("Weather", errContent, colRed, cardWidth)
	}

	// 未加载
	if !m.loaded || m.data == nil {
		emptyContent := lipgloss.NewStyle().Foreground(colMuted).Render("  Press '/' to enter city")
		return card("Weather", emptyContent, colMuted, cardWidth)
	}

	// ---- 主要内容 ----
	var sb strings.Builder

	// 当前天气
	if len(m.data.CurrentCondition) > 0 {
		cc := m.data.CurrentCondition[0]
		desc := ""
		if len(cc.WeatherDesc) > 0 {
			desc = cc.WeatherDesc[0].Value
		}

		// 城市名和位置
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

		// 标题行
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colAccent).Render("  📍 "+location) + "\n\n")

		// 左侧：图标，右侧：温度和详情
		icon := getWeatherIcon(desc)
		iconLines := strings.Split(icon, "\n")

		// 温度大字
		tempLine := lipgloss.NewStyle().Foreground(colGreen).Bold(true).Render(cc.TempC + "°C")
		feelsLike := lipgloss.NewStyle().Foreground(colMuted).Render("Feels like " + cc.FeelsLikeC + "°C")

		// 天气描述
		descLine := lipgloss.NewStyle().Foreground(colSecondary).Bold(true).Render(desc)

		// 详细信息
		detailStyle := lipgloss.NewStyle().Foreground(colMuted)
		details := []string{
			detailStyle.Render("  💧 Humidity:  " + cc.Humidity + "%"),
			detailStyle.Render("  💨 Wind:     " + cc.WindspeedKmph + " km/h"),
			detailStyle.Render("  👁  Visibility: " + cc.Visibility + " km"),
			detailStyle.Render("  ☀  UV Index:  " + cc.UVIndex),
		}

		// 组合左侧（图标）
		leftSide := lipgloss.NewStyle().Foreground(colAccent).Render(strings.Join(iconLines, "\n"))

		// 组合右侧（信息）
		rightSide := descLine + "\n\n" + tempLine + "\n" + feelsLike + "\n\n" + strings.Join(details, "\n")

		// 水平排列
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftSide, "  ", rightSide))
	}

	// 预报（一行多列紧凑布局）
	if len(m.data.Weather) > 0 {
		sb.WriteString("\n\n" + lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render("  ─── 3-Day Forecast ───") + "\n\n")

		// 获取天气图标的辅助函数
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

		// 获取温度颜色的辅助函数
		getTempColor := func(temp int) lipgloss.Color {
			if temp >= 30 {
				return colRed
			} else if temp >= 20 {
				return colAccent
			} else if temp >= 10 {
				return colGreen
			}
			return colBlue
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

			// 日期标题
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colSecondary).Render("  "+dateStr) + "\n")

			// 时间行（一行显示所有时间点）
			timeRow := "  "
			tempRow := "  "
			iconRow := "  "

			for _, h := range day.Hourly {
				// 时间（wttr.in 返回的是 "0", "300", "600", "900" 等格式）
				timeStr := h.Time
				hour := 0
				if len(timeStr) >= 3 {
					hour, _ = strconv.Atoi(timeStr[:len(timeStr)-2])
				} else {
					hour, _ = strconv.Atoi(timeStr)
				}
				timeStr = fmt.Sprintf("%02d:00", hour)

				// 温度
				temp, _ := strconv.Atoi(h.TempC)

				// 天气描述
				desc := ""
				if len(h.WeatherDesc) > 0 {
					desc = h.WeatherDesc[0].Value
				}

				// 格式化为固定宽度（每个字段占 8 个字符）
				timeRow += fmt.Sprintf("%-8s", timeStr)
				tempRow += lipgloss.NewStyle().Foreground(getTempColor(temp)).Render(fmt.Sprintf("%-8s", fmt.Sprintf("%d°C", temp)))
				iconRow += fmt.Sprintf("%-8s", getIcon(desc))
			}

			sb.WriteString(timeRow + "\n")
			sb.WriteString(tempRow + "\n")
			sb.WriteString(iconRow + "\n")
		}
	}

	// 按行切分，应用滚动
	fullContent := sb.String()
	contentLines := strings.Split(fullContent, "\n")

	// 可用高度：卡片内部高度 - 卡片开销(title + border + padding ≈ 4)
	viewH := m.height - 2 - 4
	if viewH < 3 {
		viewH = 3
	}

	// 钳制滚动范围
	totalLines := len(contentLines)
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

	// 卡片包裹
	return card("Weather", visible, colAccent, cardWidth)
}
