// ============================================================================
// devdash — 开发者终端工具箱
//
// 9 个模块，数字键 1-9 切换，q 退出:
//   [1] Git 可视化 — 分支图、提交历史、热文件、贡献者排行
//   [2] 日志查看器 — 彩色高亮、正则过滤、错误统计
//   [3] 天气面板   — ASCII 天气动画、7 天预报
//   [4] 配置浏览   — JSON/YAML 折叠展开、语法高亮
//   [5] 系统监控   — CPU/内存/磁盘、进程列表
//   [6] 端口扫描   — 常用端口状态检测
//   [7] LinuxDo   — linux.do 论坛浏览
//   [8] Route     — 路由管理（静态路由增删查）
//   [9] DevTools  — 编码/哈希工具箱（Base 系列、URL、Hash、多层解码）
// ============================================================================

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"cava_go/internal/config"
	"cava_go/internal/devtools"
	"cava_go/internal/git"
	"cava_go/internal/linuxdo"
	"cava_go/internal/log"
	"cava_go/internal/ports"
	"cava_go/internal/route"
	"cava_go/internal/system"
	"cava_go/internal/ui"
	"cava_go/internal/weather"
)

// splashTickMsg 闪屏定时器消息
type splashTickMsg struct{}

// model 整个应用的顶层状态模型
// Bubble tea 的核心: Model(数据) → Update(更新) → View(渲染)
type model struct {
	state       ui.TabState     // 当前激活的 Tab
	width       int             // 终端宽度
	height      int             // 终端高度
	appCfg      *ui.AppConfig   // 应用持久化配置（与 route 模块共享）
	cfgSaved    bool            // 配置是否已保存
	helpOverlay bool            // 帮助面板是否展开
	splashDone  bool            // 闪屏是否已结束
	git         *git.Model      // Git 可视化子模块
	log         *log.Model      // 日志查看器子模块
	weather     *weather.Model  // 天气面板子模块
	config      *config.Model   // 配置浏览器子模块
	sys         *system.Model   // 系统监控子模块
	ports       *ports.Model    // 端口扫描子模块
	linuxdo     *linuxdo.Model  // LinuxDo 论坛子模块
	routeMod    *route.Model    // 路由管理子模块
	devtools    *devtools.Model // DevTools 编码/哈希工具箱子模块
}

// anyInputActive 检查是否有模块的输入框正在接收输入
func (m model) anyInputActive() bool {
	return m.git.InputActive() || m.log.InputActive() || m.weather.InputActive() ||
		m.config.InputActive() || m.sys.InputActive() || m.ports.InputActive() || m.linuxdo.InputActive() ||
		m.routeMod.InputActive() || m.devtools.InputActive()
}

// owner 专属异步消息按类型路由到归属模块，与当前 Tab 无关，防止切 Tab 后数据丢失
func (m model) owner(msg tea.Msg) ui.Module {
	switch msg.(type) {
	case git.InfoMsg, git.DirMsg:
		return m.git
	case log.LoadMsg, log.DirMsg, log.TailDataMsg:
		return m.log
	case weather.Msg:
		return m.weather
	case config.LoadMsg, config.DirMsg:
		return m.config
	case system.SysInfoMsg, system.ProcMsg:
		return m.sys
	case ports.PortsMsg:
		return m.ports
	case linuxdo.CategoriesMsg, linuxdo.TopicsMsg, linuxdo.TopicDetailMsg, linuxdo.PostStreamMsg, linuxdo.SearchMsg:
		return m.linuxdo
	case route.RoutesMsg, route.RouteActionMsg:
		return m.routeMod
	}
	return nil
}

// modules 全部子模块，广播类消息遍历发送
func (m model) modules() []ui.Module {
	return []ui.Module{m.git, m.log, m.weather, m.config, m.sys, m.ports, m.linuxdo, m.routeMod, m.devtools}
}

// active 当前活跃模块，交互消息（按键等）的归属
func (m model) active() ui.Module {
	switch m.state {
	case ui.TabGit:
		return m.git
	case ui.TabLog:
		return m.log
	case ui.TabWeather:
		return m.weather
	case ui.TabConfig:
		return m.config
	case ui.TabSystem:
		return m.sys
	case ui.TabPorts:
		return m.ports
	case ui.TabLinuxDo:
		return m.linuxdo
	case ui.TabRoute:
		return m.routeMod
	case ui.TabDevTools:
		return m.devtools
	}
	return nil
}

// Init 应用启动时执行的初始化命令
// 返回一个 Batch 命令，同时初始化所有子模块
func (m model) Init() tea.Cmd {
	// 首次同步最近记录到各模块
	m.git.SetRecent(m.appCfg.RecentRepos)
	m.log.SetRecent(m.appCfg.RecentLogFiles)
	m.config.SetRecent(m.appCfg.RecentConfigFiles)
	m.weather.SetRecent(m.appCfg.RecentCities)
	m.routeMod.SetSavedRoutes(m.appCfg.SavedRoutes)

	return tea.Batch(
		tea.Tick(5*time.Second, func(time.Time) tea.Msg { return splashTickMsg{} }),
		m.git.Init(m.appCfg.DefaultRepo),
		m.log.Init(m.appCfg.LastLogPath),
		m.weather.Init(m.appCfg.DefaultCity),
		m.config.Init(m.appCfg.LastConfigPath),
		m.sys.Init(),
		m.ports.Init(),
		m.linuxdo.Init(m.appCfg.LinuxDoCookie, m.appCfg.LinuxDoUserAgent),
		m.routeMod.Init(),
		m.devtools.Init(), // 纯同步模块，Init 只初始化工具列表
	)
}

// Update 处理所有消息（键盘事件、窗口大小变化、异步数据返回等）
// Bubbletea 框架会把所有事件封装为 tea.Msg 传入
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 配置变更消息: 各模块在路径/城市变更时发送，此处同步到持久化配置
	switch msg := msg.(type) {
	case ui.CfgChangedMsg:
		switch msg.Key {
		case "city":
			m.appCfg.DefaultCity = msg.Value
			m.appCfg.RecentCities = ui.AddToRecent(m.appCfg.RecentCities, msg.Value, 10)
			m.weather.SetRecent(m.appCfg.RecentCities)
		case "repo":
			m.appCfg.DefaultRepo = msg.Value
			m.appCfg.RecentRepos = ui.AddToRecent(m.appCfg.RecentRepos, msg.Value, 10)
			m.git.SetRecent(m.appCfg.RecentRepos)
		case "logPath":
			m.appCfg.LastLogPath = msg.Value
			m.appCfg.RecentLogFiles = ui.AddToRecent(m.appCfg.RecentLogFiles, msg.Value, 10)
			m.log.SetRecent(m.appCfg.RecentLogFiles)
		case "configPath":
			m.appCfg.LastConfigPath = msg.Value
			m.appCfg.RecentConfigFiles = ui.AddToRecent(m.appCfg.RecentConfigFiles, msg.Value, 10)
			m.config.SetRecent(m.appCfg.RecentConfigFiles)
		case "linuxdoCookie":
			m.appCfg.LinuxDoCookie = msg.Value
			m.linuxdo.SetCookie(msg.Value)
		case "linuxdoUserAgent":
			m.appCfg.LinuxDoUserAgent = msg.Value
			m.linuxdo.SetUserAgent(msg.Value)
		case "savedRoutes":
			if routes, ok := msg.Data.([]ui.RouteConfig); ok {
				m.appCfg.SavedRoutes = routes
				m.routeMod.SetSavedRoutes(routes)
			}
		}
		ui.SaveConfig(*m.appCfg)
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 终端大小变化，更新全局尺寸并通知所有子模块
		m.width = msg.Width
		m.height = msg.Height
		m.git.UpdateSize(msg.Width, msg.Height)
		m.log.UpdateSize(msg.Width, msg.Height)
		m.weather.UpdateSize(msg.Width, msg.Height)
		m.config.UpdateSize(msg.Width, msg.Height)
		m.sys.UpdateSize(msg.Width, msg.Height)
		m.ports.UpdateSize(msg.Width, msg.Height)
		m.linuxdo.UpdateSize(msg.Width, msg.Height)
		m.routeMod.UpdateSize(msg.Width, msg.Height)
		m.devtools.UpdateSize(msg.Width, msg.Height)
		return m, nil

	case splashTickMsg:
		// 闪屏定时器到期，进入主界面
		m.splashDone = true
		return m, nil

	case tea.KeyPressMsg:
		// 闪屏期间任意按键跳过
		if !m.splashDone {
			m.splashDone = true
			return m, nil
		}
		// 帮助面板显示时，只响应 ? 或 Esc 关闭，其他按键全部拦截
		if m.helpOverlay {
			if msg.String() == "?" || msg.String() == "esc" {
				m.helpOverlay = false
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+q", "ctrl+c":
			return m, tea.Quit
		// ctrl+r 手动触发窗口大小检测 + 透传给当前模块
		case "ctrl+r":
			cmds = append(cmds, func() tea.Msg { return tea.RequestWindowSize() })
		// ctrl+s 保存当前配置（Route 模块内由模块自行处理保存路由）
		case "ctrl+s":
			if m.state == ui.TabRoute && !m.routeMod.InputActive() {
				break // 透传给 Route 模块处理
			}
			if err := ui.SaveConfig(*m.appCfg); err != nil {
				// 保存失败，可以通过状态提示（暂时忽略）
			} else {
				m.cfgSaved = true
			}
			return m, nil
		// ctrl+t 切换主题
		case "ctrl+t":
			if ui.IsLightTheme() {
				m.appCfg.Theme = "dark"
			} else {
				m.appCfg.Theme = "light"
			}
			ui.SetThemeByName(m.appCfg.Theme)
			ui.SaveConfig(*m.appCfg)
			return m, nil
		// ? 切换帮助面板
		case "?":
			if m.anyInputActive() {
				// 输入框活跃时，? 透传给模块
				break
			}
			m.helpOverlay = true
			return m, nil
		// 数字键切换 Tab，输入框活跃时透传给当前模块
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.anyInputActive() {
				break // 透传给模块处理
			}
			switch msg.String() {
			case "1":
				m.state = ui.TabGit
			case "2":
				m.state = ui.TabLog
			case "3":
				m.state = ui.TabWeather
			case "4":
				m.state = ui.TabConfig
			case "5":
				m.state = ui.TabSystem
			case "6":
				m.state = ui.TabPorts
			case "7":
				m.state = ui.TabLinuxDo
			case "8":
				m.state = ui.TabRoute
			case "9":
				m.state = ui.TabDevTools
			}
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		case "ctrl+right":
			if m.anyInputActive() {
				break
			}
			m.state = m.state.Next()
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		case "ctrl+left":
			if m.anyInputActive() {
				break
			}
			m.state = m.state.Prev()
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		}
	}

	// 专属异步消息按类型路由到归属模块，与当前 Tab 无关，防止切 Tab 后数据丢失
	if mod := m.owner(msg); mod != nil {
		if cmd := mod.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// spinner 心跳广播给全部模块，未实现 spinner 的模块自然忽略
	if _, ok := msg.(spinner.TickMsg); ok {
		for _, mod := range m.modules() {
			if cmd := mod.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	// 其余消息交给当前活跃模块
	if mod := m.active(); mod != nil {
		if cmd := mod.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

// View 根据当前状态渲染整个界面
// 布局: Tab 栏(顶部) + 模块内容(中间) + 状态栏(底部 2 行)
func (m model) View() tea.View {
	// 闪屏期间只渲染闪屏画面
	if !m.splashDone {
		v := tea.NewView(ui.RenderSplash(m.width, m.height, "v0.2.0", 9))
		v.AltScreen = true
		return v
	}

	// 图片预览模式 Sixel 是像素级协议，必须绕过 tab/status 布局直接全屏输出
	// 否则会被布局管线当普通文本处理导致像素数据被冲垮
	if m.state == ui.TabLinuxDo && m.linuxdo.InImagePreview() {
		v := tea.NewView(m.linuxdo.View())
		v.AltScreen = true
		return v
	}

	tabBar := ui.RenderTabBar(m.state, m.width)

	// 渲染当前活跃模块
	var content string
	if mod := m.active(); mod != nil {
		content = mod.View()
	}

	// 帮助面板覆盖在主内容上
	if m.helpOverlay {
		content = ui.RenderHelpOverlay(m.state, m.width, m.height-3)
	}

	statusBar := ui.RenderStatusBar(m.state, m.width)

	// 计算内容区域可用高度，用空行填充让状态栏固定在底部
	// tabBar 占 1 行，statusBar 占 2 行，中间是内容区
	contentHeight := m.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	lines := strings.Count(content, "\n")
	padding := contentHeight - lines
	if padding > 0 {
		content += strings.Repeat("\n", padding)
	}

	v := tea.NewView(tabBar + "\n" + content + statusBar)
	v.AltScreen = true
	return v
}

func main() {
	// 配置选择界面先使用默认尺寸；启动后会通过 WindowSizeMsg 更新真实尺寸
	width, height := 80, 24

	// 配置选择流程
	var appCfg ui.AppConfig
	if ui.ConfigExists() {
		// 显示配置选择界面
		cfgSelect := ui.NewConfigSelectModel(width, height)
		p2 := tea.NewProgram(cfgSelect)
		result, err := p2.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if cm, ok := result.(ui.ConfigSelectModel); ok {
			switch cm.GetChoice() {
			case 0: // 默认配置
				appCfg = ui.DefaultConfig()
			case 1: // 加载持久化配置
				appCfg = cm.GetConfig()
			case 2: // 自定义配置（暂时也用默认，后续可扩展输入界面）
				appCfg = ui.DefaultConfig() // TODO: 出现系统文件选择框选择配置文件
			}
		}
	} else {
		// 没有配置文件，使用默认配置
		appCfg = ui.DefaultConfig()
	}

	// 启动主界面
	ui.SetThemeByName(appCfg.Theme)

	m := model{
		state:    ui.TabGit,
		appCfg:   &appCfg,
		weather:  &weather.Model{},
		config:   &config.Model{},
		git:      &git.Model{},
		log:      &log.Model{},
		sys:      &system.Model{},
		ports:    &ports.Model{},
		linuxdo:  &linuxdo.Model{},
		routeMod: &route.Model{},
		devtools: &devtools.Model{},
	}
	p3 := tea.NewProgram(m)
	if _, err := p3.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
