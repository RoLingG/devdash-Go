package system

import (
	"context"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// CPUInfo CPU 使用信息
type CPUInfo struct {
	Overall float64   // 总体使用率 %
	PerCore []float64 // 每核心使用率 %
}

// MemInfo 内存使用信息
type MemInfo struct {
	Used    uint64  // 已用字节
	Total   uint64  // 总字节
	Percent float64 // 使用率 %
}

// DiskInfo 磁盘使用信息
type DiskInfo struct {
	Mount   string  // 挂载点
	Used    uint64  // 已用字节
	Total   uint64  // 总字节
	Percent float64 // 使用率 %
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID   int32
	Name  string
	CPU   float64 // CPU %
	MemMB float64 // 内存 MB
}

// SysInfoMsg 系统信息消息
type SysInfoMsg struct {
	CPU   CPUInfo
	Mem   MemInfo
	Disks []DiskInfo
	Err   error
}

// ProcMsg 进程列表消息
type ProcMsg struct {
	Processes []ProcessInfo
	Err       error
}

// FetchSystemInfo 获取系统信息
func FetchSystemInfo() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var info SysInfoMsg

	// CPU 使用率
	cpuPercents, err := cpu.PercentWithContext(ctx, time.Second, true)
	if err != nil {
		info.Err = err
		return info
	}
	if len(cpuPercents) > 0 {
		var total float64
		for _, p := range cpuPercents {
			total += p
		}
		info.CPU.Overall = total / float64(len(cpuPercents))
		info.CPU.PerCore = cpuPercents
	}

	// 内存
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		info.Err = err
		return info
	}
	info.Mem = MemInfo{
		Used:    vm.Used,
		Total:   vm.Total,
		Percent: vm.UsedPercent,
	}

	// 磁盘（只看主要分区）
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err == nil {
		for _, p := range partitions {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			// 跳过小分区（< 100MB）
			if usage.Total < 100*1024*1024 {
				continue
			}
			info.Disks = append(info.Disks, DiskInfo{
				Mount:   p.Mountpoint,
				Used:    usage.Used,
				Total:   usage.Total,
				Percent: usage.UsedPercent,
			})
		}
	}

	return info
}

// FetchSystemInfoCmd 返回获取系统信息的 Cmd
func FetchSystemInfoCmd() tea.Cmd {
	return FetchSystemInfo
}

// FetchProcessesCmd 返回获取进程列表的 Cmd
func FetchProcessesCmd() tea.Cmd {
	return FetchProcesses
}

// FetchProcesses 获取进程列表
func FetchProcesses() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msg ProcMsg

	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		msg.Err = err
		return msg
	}

	var procs []ProcessInfo
	for _, pid := range pids {
		p, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			continue
		}
		name, _ := p.NameWithContext(ctx)
		cpuPercent, _ := p.CPUPercentWithContext(ctx)
		memInfo, _ := p.MemoryInfoWithContext(ctx)
		var memMB float64
		if memInfo != nil {
			memMB = float64(memInfo.RSS) / 1024 / 1024
		}
		// 过滤掉系统空闲进程和自身
		if name == "" || name == "System Idle Process" {
			continue
		}
		procs = append(procs, ProcessInfo{
			PID:   pid,
			Name:  name,
			CPU:   cpuPercent,
			MemMB: memMB,
		})
	}

	// 默认按 CPU 降序
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})

	msg.Processes = procs
	return msg
}
