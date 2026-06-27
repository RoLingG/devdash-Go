package route

import (
	"fmt"
	"net"
)

// RouteEntry 路由表条目
type RouteEntry struct {
	Dest      string // 目标地址（如 "10.0.0.0"）
	PrefixLen uint8  // 前缀长度（如 8）
	NextHop   string // 网关（如 "10.18.1.1"）
	Metric    uint32 // 跃点数
	IfIndex   uint32 // 接口索引
	IfName    string // 接口名称
	Proto     uint32 // 协议类型
	IsStatic  bool   // 是否静态路由
}

// IfaceAddr 网络接口的单个地址信息
type IfaceAddr struct {
	IP        string // IP 地址（如 "192.168.31.220" 或 "fe80::c91:af12:89f1:73df"）
	Netmask   string // 子网掩码（如 "255.255.255.0" 或 "/64"）
	Broadcast string // 广播地址（仅 IPv4，如 "192.168.31.255"）
	IsIPv6    bool
}

// IfaceInfo 网络接口信息
type IfaceInfo struct {
	Index   int
	Name    string
	Addrs   []IfaceAddr // 所有地址（IPv4 + IPv6）
	MAC     string
	MTU     int
	IsUp    bool
}

// GetInterfaces 获取网络接口及其完整地址信息
func GetInterfaces() ([]IfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []IfaceInfo
	for _, iface := range ifaces {
		// 跳过 loopback（127.0.0.1）
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// 跳过未启用的
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		var ifaceAddrs []IfaceAddr
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			mask := ipNet.Mask
			isIPv6 := ip.To4() == nil

			var netmask, broadcast string
			if isIPv6 {
				// IPv6 显示前缀长度
				ones, _ := mask.Size()
				netmask = fmt.Sprintf("/%d", ones)
			} else {
				// ip.To4() 确保 ip 是 4 字节，和 mask 长度一致
				v4 := ip.To4()
				if v4 == nil {
					v4 = ip
				}
				// IPv4 显示完整掩码 + 广播地址
				netmask = net.IP(mask).String()
				// 广播地址 = IP | ^Mask
				bcast := make(net.IP, len(v4))
				for i := range v4 {
					bcast[i] = v4[i] | ^mask[i]
				}
				broadcast = bcast.String()
				ip = v4 // 统一用 4 字节格式
			}

			ifaceAddrs = append(ifaceAddrs, IfaceAddr{
				IP:        ip.String(),
				Netmask:   netmask,
				Broadcast: broadcast,
				IsIPv6:    isIPv6,
			})
		}

		result = append(result, IfaceInfo{
			Index: iface.Index,
			Name:  iface.Name,
			Addrs: ifaceAddrs,
			MAC:   iface.HardwareAddr.String(),
			MTU:   iface.MTU,
			IsUp:  iface.Flags&net.FlagUp != 0,
		})
	}
	return result, nil
}
