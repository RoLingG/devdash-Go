package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// WttrResponse wttr.in JSON 响应结构
type WttrResponse struct {
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

// Msg 天气数据消息
type Msg struct {
	Data *WttrResponse
	Err  error
}

// FetchFromCity 从 wttr.in 获取指定城市的天气
func FetchFromCity(city string) tea.Msg {
	url := "https://wttr.in/?format=j1"
	if city != "" {
		url = fmt.Sprintf("https://wttr.in/%s?format=j1", city)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return Msg{Err: fmt.Errorf("failed to fetch weather: %w", err)}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Msg{Err: fmt.Errorf("failed to read response: %w", err)}
	}
	var data WttrResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return Msg{Err: fmt.Errorf("failed to parse weather data: %w", err)}
	}
	return Msg{Data: &data}
}

// GetWeatherIcon 根据天气描述返回 ASCII 图标
func GetWeatherIcon(desc string) string {
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
