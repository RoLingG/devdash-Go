package route

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---- 数据结构 ----

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

// IfaceInfo 网络接口信息
type IfaceInfo struct {
	Index int
	Name  string
	Addrs []string
	MAC   string
	IsUp  bool
}

// ---- Win32 API 封装 ----
var (
	modiphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procCreateIpForwardEntry2 = modiphlpapi.NewProc("CreateIpForwardEntry2")
	procDeleteIpForwardEntry2 = modiphlpapi.NewProc("DeleteIpForwardEntry2")
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

// GetInterfaces 获取活跃的网络接口
func GetInterfaces() ([]IfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []IfaceInfo
	for _, iface := range ifaces {
		// 跳过 loopback 和未启用的
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		var ipList []string
		for _, addr := range addrs {
			ipList = append(ipList, addr.String())
		}

		result = append(result, IfaceInfo{
			Index: iface.Index,
			Name:  iface.Name,
			Addrs: ipList,
			MAC:   iface.HardwareAddr.String(),
			IsUp:  true,
		})
	}
	return result, nil
}

// AddRoute 添加静态路由（需要管理员权限）
func AddRoute(dest net.IP, prefixLen uint8, gateway net.IP, ifIndex uint32, metric uint32) error {
	row := windows.MibIpForwardRow2{
		InterfaceIndex: ifIndex,
		DestinationPrefix: windows.IpAddressPrefix{
			Prefix:       ipToRawSockaddrInet(dest),
			PrefixLength: prefixLen,
		},
		NextHop:              ipToRawSockaddrInet(gateway),
		Metric:               metric,
		Protocol:             windows.MIB_IPPROTO_NETMGMT,
		Loopback:             0,
		AutoconfigureAddress: 0,
		Publish:              0,
		Immortal:             1, // 持久路由
		ValidLifetime:        0xFFFFFFFF,
		PreferredLifetime:    0xFFFFFFFF,
	}

	ret, _, _ := procCreateIpForwardEntry2.Call(uintptr(unsafe.Pointer(&row)))
	if ret != 0 {
		return fmt.Errorf("CreateIpForwardEntry2 failed: errno %d", ret)
	}
	return nil
}

// DeleteRoute 删除路由（需要管理员权限）
func DeleteRoute(row *windows.MibIpForwardRow2) error {
	ret, _, _ := procDeleteIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
	if ret != 0 {
		return fmt.Errorf("DeleteIpForwardEntry2 failed: errno %d", ret)
	}
	return nil
}

// IsAdmin 检测当前进程是否以管理员运行
func IsAdmin() bool {
	// 创建 BuiltinAdministrators SID
	var sid *windows.SID
	sidAuth := windows.SECURITY_NT_AUTHORITY
	err := windows.AllocateAndInitializeSid(
		&sidAuth,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.GetCurrentProcessToken()
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// GetRawRoutes 获取原始 MibIpForwardRow2（用于删除操作）
func GetRawRoutes() ([]windows.MibIpForwardRow2, error) {
	var table *windows.MibIpForwardTable2
	err := windows.GetIpForwardTable2(windows.AF_INET, &table)
	if err != nil {
		return nil, err
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))

	rows := table.Rows()
	result := make([]windows.MibIpForwardRow2, len(rows))
	copy(result, rows)
	return result, nil
}
