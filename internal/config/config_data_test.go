package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBuildTree 测试树构建
func TestBuildTree(t *testing.T) {
	t.Run("简单对象", func(t *testing.T) {
		raw := map[string]interface{}{
			"name":    "devdash",
			"version": "1.0.0",
		}
		root := BuildTree("", raw, 0, true)

		if root.Key != "" {
			t.Errorf("root key should be empty, got %q", root.Key)
		}
		if len(root.Children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(root.Children))
		}
		// Children should be sorted by key
		if root.Children[0].Key != "name" {
			t.Errorf("first child key should be 'name', got %q", root.Children[0].Key)
		}
		if root.Children[0].Value != "devdash" {
			t.Errorf("first child value should be 'devdash', got %v", root.Children[0].Value)
		}
		if root.Children[1].Key != "version" {
			t.Errorf("second child key should be 'version', got %q", root.Children[1].Key)
		}
	})

	t.Run("嵌套对象", func(t *testing.T) {
		raw := map[string]interface{}{
			"author": map[string]interface{}{
				"name":  "You",
				"email": "you@example.com",
			},
		}
		root := BuildTree("", raw, 0, true)

		if len(root.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(root.Children))
		}
		authorNode := root.Children[0]
		if authorNode.Key != "author" {
			t.Errorf("child key should be 'author', got %q", authorNode.Key)
		}
		if len(authorNode.Children) != 2 {
			t.Errorf("author should have 2 children, got %d", len(authorNode.Children))
		}
	})

	t.Run("数组", func(t *testing.T) {
		raw := map[string]interface{}{
			"modules": []interface{}{"git", "log", "weather"},
		}
		root := BuildTree("", raw, 0, true)

		if len(root.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(root.Children))
		}
		modulesNode := root.Children[0]
		if modulesNode.Key != "modules" {
			t.Errorf("child key should be 'modules', got %q", modulesNode.Key)
		}
		if !modulesNode.IsArray {
			t.Error("modules node should be array")
		}
		if len(modulesNode.Children) != 3 {
			t.Errorf("modules should have 3 children, got %d", len(modulesNode.Children))
		}
		if modulesNode.Children[0].Key != "[0]" {
			t.Errorf("first array element key should be '[0]', got %q", modulesNode.Children[0].Key)
		}
		if modulesNode.Children[0].Value != "git" {
			t.Errorf("first array element value should be 'git', got %v", modulesNode.Children[0].Value)
		}
	})

	t.Run("叶子节点", func(t *testing.T) {
		raw := map[string]interface{}{
			"count":  42,
			"active": true,
			"name":   "test",
		}
		root := BuildTree("", raw, 0, true)

		for _, child := range root.Children {
			if len(child.Children) != 0 {
				t.Errorf("leaf node %q should have no children", child.Key)
			}
		}
		// BuildTree 会对 keys 排序，所以顺序是 active、count、name
		if root.Children[0].Key != "active" || root.Children[0].Value != true {
			t.Errorf("first child should be active=true, got %s=%v", root.Children[0].Key, root.Children[0].Value)
		}
		if root.Children[1].Key != "count" || root.Children[1].Value != 42 {
			t.Errorf("second child should be count=42, got %s=%v", root.Children[1].Key, root.Children[1].Value)
		}
		if root.Children[2].Key != "name" || root.Children[2].Value != "test" {
			t.Errorf("third child should be name=test, got %s=%v", root.Children[2].Key, root.Children[2].Value)
		}
	})

	t.Run("深度设置", func(t *testing.T) {
		raw := map[string]interface{}{
			"a": map[string]interface{}{
				"b": "value",
			},
		}
		root := BuildTree("", raw, 0, true)

		if root.Depth != 0 {
			t.Errorf("root depth should be 0, got %d", root.Depth)
		}
		aNode := root.Children[0]
		if aNode.Depth != 1 {
			t.Errorf("a depth should be 1, got %d", aNode.Depth)
		}
		bNode := aNode.Children[0]
		if bNode.Depth != 2 {
			t.Errorf("b depth should be 2, got %d", bNode.Depth)
		}
	})

	t.Run("展开状态", func(t *testing.T) {
		raw := map[string]interface{}{
			"a": map[string]interface{}{
				"b": "value",
			},
		}
		root := BuildTree("", raw, 0, true)

		if !root.Expanded {
			t.Error("root should be expanded")
		}
		aNode := root.Children[0]
		if aNode.Expanded {
			t.Error("a should not be expanded by default")
		}
	})
}

// TestLoadFile 测试文件加载
func TestLoadFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("JSON 文件", func(t *testing.T) {
		content := `{"name": "test", "version": "1.0.0"}`
		path := filepath.Join(tmpDir, "test.json")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err != nil {
			t.Fatalf("unexpected error: %v", loadMsg.Err)
		}
		if len(loadMsg.Root.Children) != 2 {
			t.Errorf("expected 2 children, got %d", len(loadMsg.Root.Children))
		}
	})

	t.Run("YAML 文件", func(t *testing.T) {
		content := "name: test\nversion: \"1.0.0\"\n"
		path := filepath.Join(tmpDir, "test.yaml")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err != nil {
			t.Fatalf("unexpected error: %v", loadMsg.Err)
		}
		if len(loadMsg.Root.Children) != 2 {
			t.Errorf("expected 2 children, got %d", len(loadMsg.Root.Children))
		}
	})

	t.Run("TOML 文件", func(t *testing.T) {
		content := "name = \"test\"\nversion = \"1.0.0\"\n"
		path := filepath.Join(tmpDir, "test.toml")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err != nil {
			t.Fatalf("unexpected error: %v", loadMsg.Err)
		}
		if len(loadMsg.Root.Children) != 2 {
			t.Errorf("expected 2 children, got %d", len(loadMsg.Root.Children))
		}
	})

	t.Run("文件不存在", func(t *testing.T) {
		path := filepath.Join(tmpDir, "not_exist.json")

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("无效 JSON", func(t *testing.T) {
		content := `{invalid json}`
		path := filepath.Join(tmpDir, "invalid.json")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("无效 YAML", func(t *testing.T) {
		content := ":\n  :\n    invalid: [yaml"
		path := filepath.Join(tmpDir, "invalid.yaml")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("无效 TOML", func(t *testing.T) {
		content := "[invalid\ntoml = ["
		path := filepath.Join(tmpDir, "invalid.toml")
		_ = os.WriteFile(path, []byte(content), 0644)

		msg := LoadFile(path)
		loadMsg, ok := msg.(LoadMsg)
		if !ok {
			t.Fatalf("expected LoadMsg, got %T", msg)
		}
		if loadMsg.Err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// TestScanDir 测试目录扫描
func TestScanDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("扫描配置文件", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644)
		_ = os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(""), 0644)
		_ = os.WriteFile(filepath.Join(tmpDir, "config.yml"), []byte(""), 0644)
		_ = os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(""), 0644)
		_ = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(""), 0644)
		_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

		msg := ScanDir(tmpDir)
		dirMsg, ok := msg.(DirMsg)
		if !ok {
			t.Fatalf("expected DirMsg, got %T", msg)
		}
		if len(dirMsg.Files) != 4 {
			t.Errorf("expected 4 files, got %d: %v", len(dirMsg.Files), dirMsg.Files)
		}
	})

	t.Run("空目录", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		_ = os.Mkdir(emptyDir, 0755)

		msg := ScanDir(emptyDir)
		dirMsg, ok := msg.(DirMsg)
		if !ok {
			t.Fatalf("expected DirMsg, got %T", msg)
		}
		if len(dirMsg.Files) != 0 {
			t.Errorf("expected 0 files, got %d", len(dirMsg.Files))
		}
	})

	t.Run("目录不存在", func(t *testing.T) {
		msg := ScanDir("/nonexistent/path")
		dirMsg, ok := msg.(DirMsg)
		if !ok {
			t.Fatalf("expected DirMsg, got %T", msg)
		}
		if dirMsg.Files != nil {
			t.Errorf("expected nil files, got %v", dirMsg.Files)
		}
	})
}

// TestLoadSampleConfig 测试示例配置加载
func TestLoadSampleConfig(t *testing.T) {
	msg := LoadSampleConfig()
	loadMsg, ok := msg.(LoadMsg)
	if !ok {
		t.Fatalf("expected LoadMsg, got %T", msg)
	}
	if loadMsg.Err != nil {
		t.Fatalf("unexpected error: %v", loadMsg.Err)
	}
	if loadMsg.Root.Key != "" {
		t.Errorf("root key should be empty, got %q", loadMsg.Root.Key)
	}
	if len(loadMsg.Root.Children) == 0 {
		t.Error("sample config should have children")
	}
}

// TestBuildTreeTOMLMap 测试 TOML 解析出的 map[interface{}]interface{} 类型
func TestBuildTreeTOMLMap(t *testing.T) {
	root := BuildTree("", map[interface{}]interface{}{"k": "v", "num": 42}, 0, true)
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}
	// Children 按 key 排序：k、num
	if root.Children[0].Key != "k" || root.Children[0].Value != "v" {
		t.Errorf("child[0] should be k=v, got %s=%v", root.Children[0].Key, root.Children[0].Value)
	}
	if root.Children[1].Key != "num" || root.Children[1].Value != 42 {
		t.Errorf("child[1] should be num=42, got %s=%v", root.Children[1].Key, root.Children[1].Value)
	}
}

// TestConfigCmds 测试 Cmd 构造
func TestConfigCmds(t *testing.T) {
	if LoadFileCmd("x") == nil {
		t.Error("LoadFileCmd 返回 nil")
	}
	if ScanDirCmd("x") == nil {
		t.Error("ScanDirCmd 返回 nil")
	}
}
