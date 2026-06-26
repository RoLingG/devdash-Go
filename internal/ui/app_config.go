package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
)

// CfgChangedMsg 配置项变更消息，各模块在路径/城市变更时发送
type CfgChangedMsg struct {
	Key   string // "city" / "repo" / "logPath" / "configPath" / "savedRoutes"
	Value string
	Data  any // 复杂类型数据（如 []RouteConfig），优先级高于 Value
}

// UpdateCfgCmd 返回一个发送 CfgChangedMsg 的 Cmd（字符串值）
func UpdateCfgCmd(key, value string) tea.Cmd {
	return func() tea.Msg { return CfgChangedMsg{Key: key, Value: value} }
}

// UpdateCfgDataCmd 返回一个发送 CfgChangedMsg 的 Cmd（复杂类型数据）
func UpdateCfgDataCmd(key string, data any) tea.Cmd {
	return func() tea.Msg { return CfgChangedMsg{Key: key, Data: data} }
}

// RouteConfig 保存的路由配置条目
type RouteConfig struct {
	Dest      string `json:"dest"`       // 目标地址（如 "10.0.0.0"）
	PrefixLen uint8  `json:"prefix_len"` // 前缀长度（如 8）
	NextHop   string `json:"next_hop"`   // 网关（如 "10.18.1.1"）
	Metric    uint32 `json:"metric"`     // 跃点数
	IfIndex   uint32 `json:"if_index"`   // 接口索引
	IfName    string `json:"if_name"`    // 接口名称（仅展示用）
}

// AppConfig 应用持久化配置
type AppConfig struct {
	DefaultCity       string        `json:"default_city"`
	DefaultRepo       string        `json:"default_repo"`
	LastLogPath       string        `json:"last_log_path"`
	LastConfigPath    string        `json:"last_config_path"`
	RecentRepos       []string      `json:"recent_repos"`
	RecentLogFiles    []string      `json:"recent_log_files"`
	RecentConfigFiles []string      `json:"recent_config_files"`
	RecentCities      []string      `json:"recent_cities"`
	LinuxDoCookie     string        `json:"linuxdo_cookie,omitempty"`
	LinuxDoUserAgent  string        `json:"linuxdo_user_agent,omitempty"`
	Theme             string        `json:"theme,omitempty"`
	SavedRoutes       []RouteConfig `json:"saved_routes,omitempty"`
}

// configFileName 配置文件名
const configFileName = "devdash.json"

// GetConfigPath 获取配置文件路径（与 exe 同目录）
func GetConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		// 回退到当前目录
		return configFileName
	}
	return filepath.Join(filepath.Dir(exePath), configFileName)
}

// ConfigExists 检查配置文件是否存在
func ConfigExists() bool {
	path := GetConfigPath()
	_, err := os.Stat(path)
	return err == nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() AppConfig {
	return AppConfig{
		DefaultCity:       "Beijing",
		DefaultRepo:       "",
		LastLogPath:       "",
		LastConfigPath:    "",
		RecentRepos:       []string{},
		RecentLogFiles:    []string{},
		RecentConfigFiles: []string{},
		RecentCities:      []string{},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig() (AppConfig, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// 确保切片不为 nil
	if cfg.RecentRepos == nil {
		cfg.RecentRepos = []string{}
	}
	if cfg.RecentLogFiles == nil {
		cfg.RecentLogFiles = []string{}
	}
	if cfg.RecentConfigFiles == nil {
		cfg.RecentConfigFiles = []string{}
	}
	if cfg.RecentCities == nil {
		cfg.RecentCities = []string{}
	}

	return cfg, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(cfg AppConfig) error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AddToRecent 添加到最近列表（去重，最多保留 10 个）
func AddToRecent(list []string, item string, maxItems int) []string {
	if item == "" {
		return list
	}

	// 去重：移除已存在的
	filtered := make([]string, 0, len(list))
	for _, s := range list {
		if s != item {
			filtered = append(filtered, s)
		}
	}

	// 添加到开头
	result := append([]string{item}, filtered...)

	// 限制数量
	if len(result) > maxItems {
		result = result[:maxItems]
	}

	return result
}
