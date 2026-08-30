package ui

import tea "charm.land/bubbletea/v2"

// Module 主程序统一分发的子模块接口，实现方以指针接收者原地变更状态，Update 只返回命令
type Module interface {
	// Update 原地更新模块状态，返回产生的异步命令，无命令时返回 nil
	Update(msg tea.Msg) tea.Cmd
	// View 渲染当前视图为可拼接的字符串，不是 bubbletea 的画布
	View() string
	// UpdateSize 终端尺寸变化时同步给模块
	UpdateSize(w, h int)
	// InputActive 输入框是否处于活跃状态，顶层按键拦截判断用
	InputActive() bool
}
