package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"gopkg.in/yaml.v3"
)

// Node 配置树的一个节点
type Node struct {
	Key      string
	Value    interface{}
	Children []Node
	IsArray  bool
	Expanded bool
	Depth    int
}

// LoadMsg 配置加载完成消息
type LoadMsg struct {
	Root Node
	Err  error
}

// DirMsg 目录扫描结果
type DirMsg struct {
	Dir   string
	Files []string
}

// LoadFile 从文件加载配置（自动识别 JSON / YAML）
func LoadFile(path string) tea.Msg {
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadMsg{Err: fmt.Errorf("cannot read: %s (%v)", path, err)}
	}
	var raw interface{}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return LoadMsg{Err: fmt.Errorf("YAML parse error: %v", err)}
		}
	default:
		if err := json.Unmarshal(data, &raw); err != nil {
			return LoadMsg{Err: fmt.Errorf("JSON parse error: %v", err)}
		}
	}
	root := BuildTree("", raw, 0, true)
	return LoadMsg{Root: root}
}

// ScanDir 扫描目录下所有 JSON / YAML 文件
func ScanDir(dir string) tea.Msg {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirMsg{Dir: dir, Files: nil}
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".json" || ext == ".yaml" || ext == ".yml" {
			files = append(files, e.Name())
		}
	}
	return DirMsg{Dir: dir, Files: files}
}

// LoadFileCmd 返回一个执行 LoadFile 的 Cmd
func LoadFileCmd(path string) tea.Cmd {
	return func() tea.Msg { return LoadFile(path) }
}

// ScanDirCmd 返回一个执行 ScanDir 的 Cmd
func ScanDirCmd(dir string) tea.Cmd {
	return func() tea.Msg { return ScanDir(dir) }
}

// LoadSampleConfig 加载示例配置
func LoadSampleConfig() tea.Msg {
	sample := `{
  "name": "devdash",
  "version": "1.0.0",
  "description": "Developer terminal dashboard",
  "author": {
    "name": "You",
    "email": "you@example.com"
  },
  "dependencies": {
    "bubbletea": "^1.0.0",
    "lipgloss": "^1.0.0",
    "bubbles": "^1.0.0"
  },
  "scripts": {
    "build": "go build -o devdash .",
    "run": "go run .",
    "test": "go test ./..."
  },
  "config": {
    "theme": "dark",
    "refreshRate": 30,
    "modules": ["git", "log", "weather", "config"],
    "features": {
      "autoRefresh": true,
      "notifications": false
    }
  }
}`
	var raw interface{}
	_ = json.Unmarshal([]byte(sample), &raw)
	root := BuildTree("", raw, 0, true)
	return LoadMsg{Root: root}
}

// BuildTree 递归构建配置树
func BuildTree(key string, val interface{}, depth int, expanded bool) Node {
	node := Node{Key: key, Depth: depth, Expanded: expanded}
	switch v := val.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			node.Children = append(node.Children, BuildTree(k, v[k], depth+1, false))
		}
	case map[interface{}]interface{}:
		keys := make([]string, 0, len(v))
		keyMap := make(map[string]interface{}, len(v))
		for k, child := range v {
			ks := fmt.Sprintf("%v", k)
			keys = append(keys, ks)
			keyMap[ks] = child
		}
		sort.Strings(keys)
		for _, k := range keys {
			node.Children = append(node.Children, BuildTree(k, keyMap[k], depth+1, false))
		}
	case []interface{}:
		node.IsArray = true
		for i, child := range v {
			node.Children = append(node.Children, BuildTree(fmt.Sprintf("[%d]", i), child, depth+1, false))
		}
	default:
		node.Value = val
	}
	return node
}
