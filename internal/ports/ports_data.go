package ports

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// PortInfo 端口信息
type PortInfo struct {
	Port    int
	Service string // 预设服务名
	Open    bool
	Process string // 占用进程名（可选，不一定能获取）
}

// PortsMsg 端口扫描结果消息
type PortsMsg struct {
	Ports []PortInfo
	Err   error
}

// 默认扫描的常用开发端口
var defaultPorts = []struct {
	Port    int
	Service string
}{
	{22, "SSH"},
	{80, "HTTP"},
	{443, "HTTPS"},
	{3000, "React/Express"},
	{3306, "MySQL"},
	{5432, "PostgreSQL"},
	{6379, "Redis"},
	{8080, "HTTP-Alt"},
	{8443, "HTTPS-Alt"},
	{27017, "MongoDB"},
	{9090, "Prometheus"},
	{5173, "Vite"},
	{8888, "Jupyter"},
	{4200, "Angular"},
	{9229, "Node Debug"},
	{5000, "Flask"},
	{8000, "Django"},
}

// ScanPorts 并发扫描端口列表
func ScanPorts(extraPorts []int) tea.Msg {
	var msg PortsMsg

	// 合并预设端口和自定义端口
	portSet := make(map[int]string)
	for _, p := range defaultPorts {
		portSet[p.Port] = p.Service
	}
	for _, p := range extraPorts {
		if _, exists := portSet[p]; !exists {
			portSet[p] = "Custom"
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for port, service := range portSet {
		wg.Add(1)
		go func(port int, service string) {
			defer wg.Done()
			open := probePort(port)
			mu.Lock()
			msg.Ports = append(msg.Ports, PortInfo{
				Port:    port,
				Service: service,
				Open:    open,
			})
			mu.Unlock()
		}(port, service)
	}

	wg.Wait()

	// 按端口号排序
	sort.Slice(msg.Ports, func(i, j int) bool {
		return msg.Ports[i].Port < msg.Ports[j].Port
	})

	return msg
}

// ScanPortsCmd 返回扫描端口的 Cmd
func ScanPortsCmd(extraPorts []int) tea.Cmd {
	return func() tea.Msg { return ScanPorts(extraPorts) }
}

// probePort 探测端口是否开放
func probePort(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
