// ============================================================================
// devdash — 开发者终端工具箱
//
// 4 个模块，数字键 1-4 切换，q 退出：
//   [1] Git 可视化 — 分支图、提交历史、热文件、贡献者排行
//   [2] 日志查看器 — 彩色高亮、正则过滤、错误统计
//   [3] 天气面板   — ASCII 天气动画、7 天预报
//   [4] 配置浏览   — JSON/YAML 折叠展开、语法高亮
// ============================================================================

package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// tabState 当前激活的 Tab 页索引
type tabState int

const (
	tabGit     tabState = iota // 0 → Git 可视化
	tabLog                     // 1 → 日志查看器
	tabWeather                 // 2 → 天气面板
	tabConfig                  // 3 → 配置浏览器
)

// model 整个应用的顶层状态模型
// Bubbletea 的核心：Model(数据) → Update(更新) → View(渲染)
type model struct {
	state   tabState      // 当前激活的 Tab
	width   int           // 终端宽度
	height  int           // 终端高度
	git     *gitModel     // Git 可视化子模块
	log     *logModel     // 日志查看器子模块
	weather *weatherModel // 天气面板子模块
	config  *configModel  // 配置浏览器子模块
}

// Init 应用启动时执行的初始化命令
// 返回一个 Batch 命令，同时初始化所有子模块
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.git.Init(),
		m.log.Init(),
		m.weather.Init(),
		m.config.Init(),
	)
}

// Update 处理所有消息（键盘事件、窗口大小变化、异步数据返回等）
// Bubbletea 框架会把所有事件封装为 tea.Msg 传入
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 终端大小变化，更新全局尺寸并通知所有子模块
		m.width = msg.Width
		m.height = msg.Height
		m.git.UpdateSize(msg.Width, msg.Height)
		m.log.UpdateSize(msg.Width, msg.Height)
		m.weather.UpdateSize(msg.Width, msg.Height)
		m.config.UpdateSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+q", "ctrl+c":
			return m, tea.Quit
		// ctrl+r 手动触发窗口大小检测（Windows 不支持 SIGWINCH）
		case "ctrl+r":
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		// 数字键切换 Tab（全局快捷键，在任何子模块中都能用）
		// 切换时顺便检测窗口大小，确保布局正确
		case "1":
			m.state = tabGit
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		case "2":
			m.state = tabLog
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		case "3":
			m.state = tabWeather
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		case "4":
			m.state = tabConfig
			return m, func() tea.Msg { return tea.RequestWindowSize() }
		}
	}

	// 跨模块消息：异步加载结果可能在用户切到其他 Tab 后才返回
	// 必须在顶层拦截，否则当前 Tab 的模块会丢弃这些消息
	switch msg := msg.(type) {
	case logLoadMsg, logDirMsg:
		var cmd tea.Cmd
		m.log, cmd = m.log.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case gitInfoMsg, gitDirMsg:
		var cmd tea.Cmd
		m.git, cmd = m.git.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case cfgLoadMsg, cfgDirMsg:
		var cmd tea.Cmd
		m.config, cmd = m.config.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// 把消息转发给当前激活的子模块处理
	// 每个子模块自己决定如何处理键盘、鼠标等事件
	switch m.state {
	case tabGit:
		var cmd tea.Cmd
		m.git, cmd = m.git.Update(msg)
		cmds = append(cmds, cmd)
	case tabLog:
		var cmd tea.Cmd
		m.log, cmd = m.log.Update(msg)
		cmds = append(cmds, cmd)
	case tabWeather:
		var cmd tea.Cmd
		m.weather, cmd = m.weather.Update(msg)
		cmds = append(cmds, cmd)
	case tabConfig:
		var cmd tea.Cmd
		m.config, cmd = m.config.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View 根据当前状态渲染整个界面
// 布局：Tab 栏(顶部) + 模块内容(中间) + 帮助栏(底部)
func (m model) View() tea.View {
	tabBar := renderTabBar(m.state, m.width)

	// 根据当前 Tab 渲染对应模块
	var content string
	switch m.state {
	case tabGit:
		content = m.git.View()
	case tabLog:
		content = m.log.View()
	case tabWeather:
		content = m.weather.View()
	case tabConfig:
		content = m.config.View()
	}

	help := renderHelp(m.state, m.width)

	// 计算内容区域可用高度，用空行填充让 help 栏固定在底部
	// tabBar 占 1 行，help 占 1 行，中间是内容区
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	lines := strings.Count(content, "\n")
	padding := contentHeight - lines
	if padding > 0 {
		content += strings.Repeat("\n", padding)
	}

	v := tea.NewView(tabBar + "\n" + content + help)
	v.AltScreen = true // v2: 替代 tea.WithAltScreen()
	return v
}

func main() {
	m := model{
		state:   tabGit,
		weather: &weatherModel{},
		config:  &configModel{},
		git:     &gitModel{},
		log:     &logModel{},
	}
	p := tea.NewProgram(m) // v2: AltScreen 已在 View() 中设置
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
