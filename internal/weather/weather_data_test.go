package weather

import "testing"

// TestGetWeatherIcon 测试天气描述 → ASCII 图标映射
func TestGetWeatherIcon(t *testing.T) {
	tests := []struct {
		name string
		desc string
	}{
		{"sunny", "Sunny"},
		{"clear", "Clear"},
		{"partly cloudy", "Partly cloudy"},
		{"cloudy", "Cloudy"},
		{"overcast", "Overcast"},
		{"rain", "Light rain"},
		{"drizzle", "Drizzle"},
		{"snow", "Heavy snow"},
		{"thunder", "Thunder storm"},
		{"storm", "Storm"},
		{"fog", "Fog"},
		{"mist", "Mist"},
		{"unknown", "Wacky weather"}, // 落入默认分支
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetWeatherIcon(tt.desc)
			if icon == "" {
				t.Errorf("GetWeatherIcon(%q) 为空", tt.desc)
			}
		})
	}
	// 不同类别应返回不同图标（用晴天与雪天对比）
	if GetWeatherIcon("Sunny") == GetWeatherIcon("Heavy snow") {
		t.Error("晴天与雪天图标不应相同")
	}
	// 默认分支（未知描述）也应有内容
	if def := GetWeatherIcon("???"); def == "" {
		t.Error("默认图标为空")
	}
}

// TestFetchFromCityCmd 测试 Cmd 构造（不实际发起网络请求）
func TestFetchFromCityCmd(t *testing.T) {
	cmd := FetchFromCityCmd("Beijing")
	if cmd == nil {
		t.Fatal("FetchFromCityCmd 返回 nil")
	}
}
