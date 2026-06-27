//go:build darwin

package route

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// GetRoutes 读取路由表（解析 netstat -rn 输出）
func GetRoutes() ([]RouteEntry, error) {
	out, err := exec.Command("netstat", "-rn").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat -rn: %w", err)
	}

	ifaces, _ := net.Interfaces()
	ifaceMap := make(map[int]string)
	for _, iface := range ifaces {
		ifaceMap[iface.Index] = iface.Name
	}

	var routes []RouteEntry
	inIPv4 := false

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 进入 IPv4 路由表
		if strings.HasPrefix(line, "Internet:") {
			inIPv4 = true
			continue
		}
		// 遇到 IPv6 路由表则停止
		if strings.HasPrefix(line, "Internet6:") {
			break
		}
		if !inIPv4 {
			continue
		}
		// 跳过表头行
		if strings.HasPrefix(line, "Destination") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		dest := fields[0]
		gateway := fields[1]
		flags := fields[2]
		ifName := fields[3]

		// 只处理有 G 标志的路由（有网关的）
		if !strings.Contains(flags, "G") {
			continue
		}

		// 跳过 link# 开头的网关（链路层地址）
		if strings.HasPrefix(gateway, "link#") {
			continue
		}

		// 解析目标地址和前缀长度
		var destIP string
		var prefixLen uint8
		if dest == "default" {
			destIP = "0.0.0.0"
			prefixLen = 0
		} else if strings.Contains(dest, "/") {
			// CIDR 格式：10.0.0.0/8
			parts := strings.SplitN(dest, "/", 2)
			destIP = parts[0]
			pl, err := strconv.Atoi(parts[1])
			if err != nil {
				continue
			}
			prefixLen = uint8(pl)
		} else {
			// 纯 IP，推断前缀长度
			destIP = dest
			ip := net.ParseIP(dest)
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if ip4[3] == 0 {
				if ip4[2] == 0 {
					if ip4[1] == 0 {
						prefixLen = 8
					} else {
						prefixLen = 16
					}
				} else {
					prefixLen = 24
				}
			} else {
				prefixLen = 32 // 主机路由
			}
		}

		// 判断是否静态路由
		// S 标志 = 静态, C 标志 = 连接（动态）
		isStatic := strings.Contains(flags, "S")

		// 通过接口名找索引
		var ifIndex uint32
		for idx, name := range ifaceMap {
			if name == ifName {
				ifIndex = uint32(idx)
				break
			}
		}

		routes = append(routes, RouteEntry{
			Dest:      destIP,
			PrefixLen: prefixLen,
			NextHop:   gateway,
			Metric:    0,
			IfIndex:   ifIndex,
			IfName:    ifName,
			IsStatic:  isStatic,
		})
	}
	return routes, nil
}

// AddRoute 添加静态路由（需要 root 权限）
func AddRoute(p AddRouteParams) error {
	destStr := p.Dest.String()
	gwStr := p.Gateway.String()

	var cmd *exec.Cmd
	if p.PrefixLen == 0 {
		// 默认路由
		cmd = exec.Command("route", "add", "default", gwStr)
	} else {
		cmd = exec.Command("route", "add", "-net",
			fmt.Sprintf("%s/%d", destStr, p.PrefixLen), gwStr)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("route add: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// DeleteRoute 删除路由（需要 root 权限）
func DeleteRoute(p DeleteRouteParams) error {
	var cmd *exec.Cmd
	if p.PrefixLen == 0 {
		cmd = exec.Command("route", "delete", "default")
	} else {
		cmd = exec.Command("route", "delete", "-net",
			fmt.Sprintf("%s/%d", p.Dest, p.PrefixLen))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("route delete: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// IsAdmin 检测当前进程是否以 root 运行
func IsAdmin() bool {
	return os.Getuid() == 0
}
