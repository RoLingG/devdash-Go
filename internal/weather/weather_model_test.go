package weather

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
	}
	return []rune(s)[0]
}

func newWeatherModel() *Model {
	m := &Model{}
	m.UpdateSize(100, 40)
	return m
}

// sampleWeather 构造一个包含当前天气 + 3 天预报的响应
func sampleWeather() *WttrResponse {
	w := &WttrResponse{}
	w.CurrentCondition = append(w.CurrentCondition, struct {
		TempC         string `json:"temp_C"`
		FeelsLikeC    string `json:"FeelsLikeC"`
		Humidity      string `json:"humidity"`
		WindspeedKmph string `json:"windspeedKmph"`
		WeatherDesc   []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
		Visibility string `json:"visibility"`
		UVIndex    string `json:"uvIndex"`
	}{TempC: "25", FeelsLikeC: "26", Humidity: "60", WindspeedKmph: "10",
		WeatherDesc: []struct {
			Value string `json:"value"`
		}{{Value: "Sunny"}}, Visibility: "10", UVIndex: "5"})

	w.NearestArea = append(w.NearestArea, struct {
		AreaName []struct {
			Value string `json:"value"`
		} `json:"areaName"`
		Country []struct {
			Value string `json:"value"`
		} `json:"country"`
	}{AreaName: []struct {
		Value string `json:"value"`
	}{{Value: "Beijing"}}, Country: []struct {
		Value string `json:"value"`
	}{{Value: "China"}}})

	w.Weather = append(w.Weather, struct {
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
	}{Date: "2026-08-06", Hourly: []struct {
		Time        string `json:"time"`
		TempC       string `json:"tempC"`
		WeatherDesc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	}{{Time: "900", TempC: "24", WeatherDesc: []struct {
		Value string `json:"value"`
	}{{Value: "Cloudy"}}}}})
	return w
}

func TestWeatherInit(t *testing.T) {
	m := newWeatherModel()
	cmd := m.Init("")
	if cmd == nil {
		t.Fatal("Init 返回 nil Cmd")
	}
	if m.city != "Beijing" {
		t.Errorf("Init(\"\") city = %q, want Beijing", m.city)
	}
	if !m.loading {
		t.Error("Init 后 loading 应为 true")
	}

	m2 := newWeatherModel()
	if m2.Init("Shanghai") == nil {
		t.Fatal("Init(Shanghai) 返回 nil Cmd")
	}
	if m2.city != "Shanghai" {
		t.Errorf("city = %q, want Shanghai", m2.city)
	}
}

func TestWeatherUpdateMsg(t *testing.T) {
	// 正常数据
	m := newWeatherModel()
	m.loading = true
	m.Update(Msg{Data: sampleWeather()})
	if !m.loaded {
		t.Error("Msg 后 loaded 应为 true")
	}
	if m.loading {
		t.Error("Msg 后 loading 应为 false")
	}
	if m.err != nil {
		t.Errorf("Msg 后 err 应为 nil, got %v", m.err)
	}
	if m.data == nil {
		t.Error("Msg 后 data 不应为 nil")
	}

	// 错误数据
	m2 := newWeatherModel()
	m2.input.Active = true
	m2.Update(Msg{Err: errWeather})
	if m2.err == nil {
		t.Error("Msg 带错误时应设置 err")
	}
	if !m2.loaded {
		t.Error("Msg 带错误时 loaded 应为 true")
	}
	if m2.loading {
		t.Error("Msg 带错误时 loading 应为 false")
	}
	if m2.input.Active {
		t.Error("Msg 后输入框应关闭")
	}
}

var errWeather = &weatherErr{}

type weatherErr struct{}

func (e *weatherErr) Error() string { return "network down" }

func TestWeatherUpdateKeyNavigation(t *testing.T) {
	m := newWeatherModel()
	m.scroll = 3
	m.Update(kp("up"))
	if m.scroll != 2 {
		t.Errorf("up 后 scroll = %d, want 2", m.scroll)
	}
	m.Update(kp("home"))
	if m.scroll != 0 {
		t.Errorf("home 后 scroll = %d, want 0", m.scroll)
	}
	m.Update(kp("end"))
	if m.scroll != 1<<30 {
		t.Errorf("end 后 scroll = %d, want 1<<30", m.scroll)
	}
	// down 受 maxScroll 上限约束（50 - viewH），已达上限后不再增加
	m.Update(kp("down"))
	if m.scroll != 1<<30 {
		t.Errorf("已达上限后 down scroll = %d, want 1<<30（不增加）", m.scroll)
	}
	// 正常范围内 down 递增
	m2 := newWeatherModel()
	m2.UpdateSize(100, 20) // viewH=13, maxScroll=37
	m2.Update(kp("down"))
	if m2.scroll != 1 {
		t.Errorf("down 后 scroll = %d, want 1", m2.scroll)
	}
	m2.Update(kp("up"))
	if m2.scroll != 0 {
		t.Errorf("up 后 scroll = %d, want 0", m2.scroll)
	}
}

func TestWeatherUpdateRefreshAndInput(t *testing.T) {
	// ctrl+r 刷新
	m := newWeatherModel()
	m.city = "Beijing"
	cmd := m.Update(kp("ctrl+r"))
	if cmd == nil {
		t.Error("ctrl+r 应返回 FetchFromCityCmd")
	}
	if !m.loading {
		t.Error("ctrl+r 后 loading 应为 true")
	}

	// city 为空时 ctrl+r 无 Cmd
	m2 := newWeatherModel()
	if cmd := m2.Update(kp("ctrl+r")); cmd != nil {
		t.Error("city 为空时 ctrl+r 不应返回 Cmd")
	}

	// / 打开输入框
	m3 := newWeatherModel()
	m3.city = "Beijing"
	cmd3 := m3.Update(kp("/"))
	if cmd3 != nil {
		t.Error("按 / 不应返回 Cmd")
	}
	if !m3.input.Active {
		t.Error("按 / 后应打开输入框")
	}
	if m3.input.Prompt != "City:" {
		t.Errorf("Prompt = %q, want City:", m3.input.Prompt)
	}
}

func TestWeatherSetRecent(t *testing.T) {
	m := newWeatherModel()
	m.SetRecent([]string{"Beijing"})
	if len(m.input.RecentItems) != 1 {
		t.Errorf("RecentItems = %v, want 1 项", m.input.RecentItems)
	}
}

func TestWeatherViewStates(t *testing.T) {
	// 输入框模式
	m := newWeatherModel()
	m.input.Active = true
	m.input.Prompt = "City:"
	m.input.Value = "Beijing"
	m.input.Cursor = 7
	v := m.View()
	if !strings.Contains(v, "Change City") {
		t.Errorf("输入框视图缺少标题: %q", v)
	}

	// 加载中
	m2 := newWeatherModel()
	m2.loading = true
	m2.city = "Beijing"
	v2 := m2.View()
	if !strings.Contains(v2, "Beijing") || !strings.Contains(v2, "Fetching") {
		t.Errorf("加载中视图缺少内容: %q", v2)
	}

	// 错误
	m3 := newWeatherModel()
	m3.err = errWeather
	v3 := m3.View()
	if !strings.Contains(v3, "network down") {
		t.Errorf("错误视图缺少错误信息: %q", v3)
	}

	// 未加载
	m4 := newWeatherModel()
	v4 := m4.View()
	if !strings.Contains(v4, "Press") {
		t.Errorf("未加载视图缺少提示: %q", v4)
	}
}

func TestWeatherViewMainContent(t *testing.T) {
	m := newWeatherModel()
	m.city = "Beijing"
	m.loaded = true
	m.data = sampleWeather()
	v := m.View()
	for _, want := range []string{"Weather", "Beijing", "25°C", "Sunny", "Humidity", "3-Day Forecast", "Today"} {
		if !strings.Contains(v, want) {
			t.Errorf("主视图缺少 %q", want)
		}
	}

	// 空数据不 panic
	m2 := newWeatherModel()
	m2.loaded = true
	m2.city = "Beijing"
	m2.data = &WttrResponse{}
	if v2 := m2.View(); v2 == "" {
		t.Error("空数据视图为空")
	}

	// 小高度不 panic
	m3 := newWeatherModel()
	m3.UpdateSize(100, 8)
	m3.loaded = true
	m3.data = sampleWeather()
	if v3 := m3.View(); v3 == "" {
		t.Error("小高度视图为空")
	}

	// 窄宽度不 panic
	m4 := newWeatherModel()
	m4.UpdateSize(50, 40)
	m4.loaded = true
	m4.data = sampleWeather()
	if v4 := m4.View(); v4 == "" {
		t.Error("窄宽度视图为空")
	}
}
