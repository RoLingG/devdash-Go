//go:build windows

package route

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---- Win32 API 封装 ----
var (
	modiphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procInitializeIpForwardEntry = modiphlpapi.NewProc("InitializeIpForwardEntry")
	procCreateIpForwardEntry2    = modiphlpapi.NewProc("CreateIpForwardEntry2")
	procDeleteIpForwardEntry2    = modiphlpapi.NewProc("DeleteIpForwardEntry2")
)

// ipFromRawSockaddr 从 RawSockaddrInet 提取 net.IP
func ipFromRawSockaddr(sa *windows.RawSockaddrInet) net.IP {
	if sa.Family == windows.AF_INET {
		ip := make(net.IP, 4)
		copy(ip, (*[4]byte)(unsafe.Pointer(&sa.Data[0]))[:])
		return ip
	}
	if sa.Family == windows.AF_INET6 {
		ip := make(net.IP, 16)
		copy(ip, (*[16]byte)(unsafe.Pointer(&sa.Data[0]))[:])
		return ip
	}
	return nil
}

// ipToRawSockaddrInet 将 net.IP 转为 RawSockaddrInet
func ipToRawSockaddrInet(ip net.IP) windows.RawSockaddrInet {
	var sa windows.RawSockaddrInet
	if v4 := ip.To4(); v4 != nil {
		sa.Family = windows.AF_INET
		copy((*[4]byte)(unsafe.Pointer(&sa.Data[0]))[:], v4)
	} else {
		sa.Family = windows.AF_INET6
		copy((*[16]byte)(unsafe.Pointer(&sa.Data[0]))[:], ip)
	}
	return sa
}

// GetRoutes 读取路由表（IPv4）
func GetRoutes() ([]RouteEntry, error) {
	var table *windows.MibIpForwardTable2
	err := windows.GetIpForwardTable2(windows.AF_INET, &table)
	if err != nil {
		return nil, fmt.Errorf("GetIpForwardTable2: %w", err)
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))

	ifaces, _ := net.Interfaces()
	ifaceMap := make(map[uint32]string)
	for _, iface := range ifaces {
		if iface.Index > 0 {
			ifaceMap[uint32(iface.Index)] = iface.Name
		}
	}

	rows := table.Rows()
	routes := make([]RouteEntry, 0, len(rows))
	for _, row := range rows {
		destIP := ipFromRawSockaddr(&row.DestinationPrefix.Prefix)
		gwIP := ipFromRawSockaddr(&row.NextHop)

		proto := row.Protocol
		isStatic := proto == windows.MIB_IPPROTO_NETMGMT

		ifName := ifaceMap[row.InterfaceIndex]
		if ifName == "" {
			ifName = fmt.Sprintf("IF#%d", row.InterfaceIndex)
		}

		routes = append(routes, RouteEntry{
			Dest:      destIP.String(),
			PrefixLen: row.DestinationPrefix.PrefixLength,
			NextHop:   gwIP.String(),
			Metric:    row.Metric,
			IfIndex:   row.InterfaceIndex,
			IfName:    ifName,
			Proto:     proto,
			IsStatic:  isStatic,
		})
	}
	return routes, nil
}

// AddRoute 添加静态路由（需要管理员权限）
func AddRoute(dest net.IP, prefixLen uint8, gateway net.IP, ifIndex uint32, metric uint32) error {
	var row windows.MibIpForwardRow2
	// 微软要求：必须先调 InitializeIpForwardEntry 初始化结构体
	// 它会设置 InterfaceLuid、SitePrefixLength、ValidLifetime 等内部字段为默认值
	procInitializeIpForwardEntry.Call(uintptr(unsafe.Pointer(&row)))

	// 初始化后再覆盖我们需要的字段
	row.InterfaceIndex = ifIndex
	row.DestinationPrefix = windows.IpAddressPrefix{
		Prefix:       ipToRawSockaddrInet(dest),
		PrefixLength: prefixLen,
	}
	row.NextHop = ipToRawSockaddrInet(gateway)
	row.Metric = metric
	row.Protocol = windows.MIB_IPPROTO_NETMGMT
	row.Immortal = 1

	ret, _, _ := procCreateIpForwardEntry2.Call(uintptr(unsafe.Pointer(&row)))
	if ret != 0 {
		return fmt.Errorf("CreateIpForwardEntry2 failed: errno %d", ret)
	}
	return nil
}

// DeleteRoute 删除路由（需要 root 权限）
// 内部查找匹配的路由行后调用 DeleteIpForwardEntry2
func DeleteRoute(dest string, prefixLen uint8, nextHop string, ifIndex uint32) error {
	// 获取路由表，找到匹配的行
	var table *windows.MibIpForwardTable2
	err := windows.GetIpForwardTable2(windows.AF_INET, &table)
	if err != nil {
		return fmt.Errorf("GetIpForwardTable2: %w", err)
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))

	destIP := net.ParseIP(dest)
	gwIP := net.ParseIP(nextHop)
	if destIP == nil || gwIP == nil {
		return fmt.Errorf("invalid address: %s or %s", dest, nextHop)
	}

	rows := table.Rows()
	for i := range rows {
		row := &rows[i]
		rowDest := ipFromRawSockaddr(&row.DestinationPrefix.Prefix)
		rowGW := ipFromRawSockaddr(&row.NextHop)

		if rowDest.Equal(destIP.To4()) &&
			rowGW.Equal(gwIP.To4()) &&
			row.DestinationPrefix.PrefixLength == prefixLen &&
			row.InterfaceIndex == ifIndex {

			ret, _, _ := procDeleteIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
			if ret != 0 {
				return fmt.Errorf("DeleteIpForwardEntry2 failed: errno %d", ret)
			}
			return nil
		}
	}
	return fmt.Errorf("route not found: %s/%d via %s", dest, prefixLen, nextHop)
}

// IsAdmin 检测当前进程是否以 root 权限运行
func IsAdmin() bool {
	var sid *windows.SID
	sidAuth := windows.SECURITY_NT_AUTHORITY
	err := windows.AllocateAndInitializeSid(
		&sidAuth,                            // 权威机构 = NT Authority (5)
		2,                                   // 子权限个数 = 2
		windows.SECURITY_BUILTIN_DOMAIN_RID, // 子权限[0] = 32（内置域）
		windows.DOMAIN_ALIAS_RID_ADMINS,     // 子权限[1] = 544（管理员组）
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// 检查当前用户是否属于管理员组
	token := windows.GetCurrentProcessToken()
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}
