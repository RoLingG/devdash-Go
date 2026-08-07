package ports

import (
	"net"
	"testing"
)

// TestProbePort 测试端口探测
func TestProbePort(t *testing.T) {
	// 启动本地监听，探测应为 true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("无法启动监听: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	if !probePort(port) {
		t.Errorf("probePort(%d) = false, want true（本地监听应开放）", port)
	}

	// 未监听的端口应为 false（连接立即被拒）
	if probePort(1) {
		t.Errorf("probePort(1) = true, want false（应无服务监听）")
	}
}

// TestScanPorts 测试端口扫描（合并、排序、去重、自定义标记）
func TestScanPorts(t *testing.T) {
	msg := ScanPorts([]int{8080, 99999})
	pm, ok := msg.(PortsMsg)
	if !ok {
		t.Fatalf("ScanPorts 返回类型错误: %T", msg)
	}
	if len(pm.Ports) == 0 {
		t.Fatal("ScanPorts 结果为空")
	}

	// 按端口号升序
	for i := 1; i < len(pm.Ports); i++ {
		if pm.Ports[i].Port < pm.Ports[i-1].Port {
			t.Errorf("结果未按端口排序: %v", pm.Ports)
		}
	}

	// 预设端口带服务名
	for _, p := range pm.Ports {
		if p.Port == 22 && p.Service != "SSH" {
			t.Errorf("22 端口服务名 = %q, want SSH", p.Service)
		}
	}

	// 自定义端口并入且标记 Custom
	found := false
	for _, p := range pm.Ports {
		if p.Port == 99999 && p.Service == "Custom" {
			found = true
		}
	}
	if !found {
		t.Error("自定义端口 99999 未合并或标记错误")
	}

	// 预设 + 自定义重复端口应去重
	count := 0
	for _, p := range pm.Ports {
		if p.Port == 8080 {
			count++
		}
	}
	if count != 1 {
		t.Errorf("8080 出现 %d 次, want 1（预设+自定义应去重）", count)
	}
}
