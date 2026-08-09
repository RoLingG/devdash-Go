# devdash - 开发者终端工具箱

**简体中文** | [English](readme/README_EN.md) | [繁體中文](readme/README_TW.md)

一个基于 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 框架的终端仪表盘，集成 8 个实用模块，数字键快速切换，nano 风格双行状态栏，`?` 键呼出帮助面板。

## 功能模块

| 快捷键 | 模块 | 功能 |
|--------|------|:-----|
| `1` | Git 可视化 | 提交历史、分支列表、ahead/behind 状态、工作区状态、热文件柱状图、仓库统计、目录扫描、最近记录、**数据缓存** |
| `2` | 日志查看器 | 分页浏览（每页 10 条）、正则过滤、级别筛选、跟随模式、彩色高亮、错误统计、目录扫描、最近记录、**数据缓存** |
| `3` | 天气面板 | 卡片式布局、3 天预报、ASCII 图标、滚动浏览、最近城市（数据来源 wttr.in） |
| `4` | 配置浏览 | JSON/YAML/TOML 树形展示、折叠/展开、搜索高亮过滤、节点路径显示、语法高亮、最近记录、**数据缓存** |
| `5` | 系统监控 | CPU 总体/每核心使用率、内存/磁盘进度条、进程列表、阈值颜色预警、Tab 切换子视图、**自动刷新** |
| `6` | 端口扫描 | 预设开发端口扫描、自定义端口、并发扫描、状态显示 |
| `7` | LinuxDo 论坛 | 分类浏览、帖子列表、帖子详情、无限滚动、Cookie 认证 |
| `8` | 路由管理 | 跨平台路由表（Windows/macOS）、添加/删除静态路由、网卡信息、持久化 |

## 安装与运行

```bash
# 克隆项目
git clone <repo-url>
cd cava_go

# 编译
go build -o devdash.exe .

# 运行
./devdash.exe
```

## 使用方式

```bash
# 默认启动，显示配置选择界面（首次）或上次保存的配置
./devdash.exe

# 直接指定日志文件
./devdash.exe --log app.log

# 直接指定配置文件（JSON / YAML）
./devdash.exe --config package.json
./devdash.exe --config config.yaml

# 管道输入日志
cat app.log | ./devdash.exe
```

启动时会显示 TUI 配置选择界面，可选择 Git 仓库路径、日志文件路径、天气城市、配置文件路径。配置会自动保存为 `devdash.json`（exe 同目录），下次启动自动加载上次的设置。

选择配置后会显示启动闪屏（ASCII Logo + 版本信息），任意按键或 2 秒后自动进入主界面。

## 持久化配置

- `Ctrl+S` - 保存当前配置到 `devdash.json`
- 配置内容包括：Git 仓库路径、日志文件路径、天气城市、配置文件路径
- 自动记录最近使用过的仓库、日志文件、配置文件、城市（最多 10 条）
- 输入模式下 `↑` `↓` 可快速浏览和选择最近记录
- 启动时自动加载上次保存的配置

## 全局快捷键

| 快捷键 | 功能 |
|--------|------|
| `1` `2` `3` `4` `5` `6` `7` `8` | 切换模块 |
| `Ctrl+←` / `Ctrl+→` | 循环切换模块（首尾循环） |
| `?` | 帮助面板（显示当前模块快捷键） |
| `Ctrl+S` | 保存配置 |
| `Ctrl+T` | 切换暗色/亮色主题 |
| `Ctrl+Q` / `Ctrl+C` | 退出 |

### Git 模块 (`1`)
- `↑` `↓` / `k` `j` - 滚动
- `/` - 输入仓库路径（支持目录扫描，列出含 `.git` 的子目录）
- 输入模式下 `↑` `↓` - 浏览最近使用的仓库
- `Ctrl+R` - 刷新数据（智能缓存：检测 `.git/index` 变化，未变化时使用缓存）

### 日志模块 (`2`)
- `↑` `↓` / `k` `j` - 页内光标移动
- `←` `→` - 上一页 / 下一页
- `Ctrl+↑` `Ctrl+↓` - 快速翻 10 页
- `Ctrl+P` - 页码跳转（输入页码后回车确认）
- `/` - 输入文件路径（支持目录扫描，列出 `.log` 文件）
- 输入模式下 `↑` `↓` - 浏览最近使用的日志文件
- 输入字符 - 正则过滤日志行
- `Ctrl+L` - 级别筛选（多选：All/Info/Warn/Error/Debug，与正则过滤叠加）
- `Ctrl+F` - 跟随模式（自动检测文件变化并刷新，再次按取消）
- `Ctrl+U` - 清除过滤
- `Esc` - 关闭输入框
- `Ctrl+R` - 刷新日志（智能缓存：检测文件 mtime 变化，未变化时使用缓存；跟随模式下同时重启自旋）

### 天气模块 (`3`)
- `↑` `↓` / `k` `j` - 滚动内容
- `/` - 切换城市
- 输入模式下 `↑` `↓` - 浏览最近使用过的城市
- `Ctrl+R` - 刷新天气数据

### 配置模块 (`4`)
- `↑` `↓` / `k` `j` - 导航
- `Enter` - 折叠/展开节点
- `/` - 输入文件路径（支持目录扫描，列出 `.json`/`.yaml`/`.yml`/`.toml` 文件）
- 输入模式下 `↑` `↓` - 浏览最近使用的配置文件
- `Ctrl+R` - 刷新配置文件（智能缓存：检测文件 mtime 变化，未变化时使用缓存）
- 输入字符 - 搜索键名/值（匹配子串高亮显示）
- `Ctrl+N` / `Ctrl+B` - 跳转到下一个/上一个匹配项
- `Ctrl+E` - 展开全部节点
- `Ctrl+W` - 收起全部节点
- `Ctrl+U` - 清除搜索
- `Esc` - 关闭输入框

### 系统监控模块 (`5`)
- `Tab` - 切换子视图（系统概览 ↔ 进程列表）
- `↑` `↓` / `k` `j` - 滚动
- `Home` / `End` - 跳转首/末
- `/` - 打开输入（进程名 filter）
- `Ctrl+R` - 手动刷新数据（同时启动自动刷新轮询）
- `Ctrl+U` - 清空 filter
- `Esc` - 关闭输入框

### 端口扫描模块 (`6`)
- `↑` `↓` / `k` `j` - 滚动端口列表
- `Home` / `End` - 跳转首/末
- `/` - 添加自定义端口
- `Ctrl+R` - 重新扫描
- `Ctrl+U` - 清空自定义端口
- `Esc` - 关闭输入框

### LinuxDo 论坛模块 (`7`)
- `↑` `↓` / `k` `j` - 滚动/移动选中
- `Home` / `End` - 跳转首/末
- `Enter` - 进入下级（分类→帖子列表→帖子详情）
- `Esc` - 返回上级
- `Ctrl+↑` `Ctrl+↓` - 快速滚动 ±10 条
- `/` - 设置 Cookie / User-Agent
- `Ctrl+R` - 刷新当前视图
- `Ctrl+U` - 清空 Cookie

### 路由管理模块 (`8`)
- `↑` `↓` / `k` `j` - 滚动
- `Home` / `End` - 跳转首/末
- `Tab` - 切换路由表/接口视图
- `Ctrl+A` - 添加静态路由（overlay 表单）
- `Ctrl+D` - 删除选中路由
- `Ctrl+S` - 保存当前静态路由到配置文件
- `Ctrl+L` - 从配置文件加载并应用路由（增量，跳过已存在的）
- `Ctrl+R` - 刷新
- `Esc` - 关闭 overlay

## 项目结构

```
cava_go/
├── main.go                        # 主入口，顶层模型与 Tab 切换，跨模块消息路由
├── devdash.json                   # 持久化配置文件（自动生成）
├── internal/
│   ├── ui/                        # 样式、颜色、渲染辅助函数
│   │   ├── style.go               # 颜色常量、样式常量、TabState
│   │   ├── box.go                 # Card()、Box()、BarChart()、RenderInputCard()、RenderDirListCard()
│   │   ├── tabbar.go              # RenderTabBar()、RenderStatusBar()
│   │   ├── help_overlay.go        # RenderHelpOverlay()（居中帮助面板）
│   │   ├── text.go                # Rune 系列函数、Truncate、PadRight、TrueWidth
│   │   ├── app_config.go          # AppConfig 结构、Load/Save、CfgChangedMsg
│   │   └── config_select.go       # 启动时 TUI 配置选择界面
│   ├── component/                 # 可复用 UI 状态组件
│   │   ├── input.go               # InputModel（通用输入框）
│   │   └── list.go                # ListModel（通用列表）
│   ├── git/                       # Git 可视化模块
│   │   ├── git_data.go            # GitInfo + git 命令执行与解析
│   │   └── git_model.go           # Model + Init/Update/View
│   ├── log/                       # 日志查看器模块
│   │   ├── log_data.go            # 日志加载/扫描/级别检测
│   │   └── log_model.go           # Model + Init/Update/View
│   ├── weather/                   # 天气面板模块
│   │   ├── weather_data.go        # wttr.in API 请求与解析
│   │   └── weather_model.go       # Model + Init/Update/View
│   ├── config/                    # 配置浏览器模块
│   │   ├── config_data.go         # JSON/YAML 解析与树构建
│   │   └── config_model.go        # Model + Init/Update/View
│   ├── system/                    # 系统监控模块
│   │   ├── system_data.go         # gopsutil 系统信息采集（CPU/内存/磁盘/进程）
│   │   └── system_model.go        # Model + Init/Update/View
│   ├── ports/                     # 端口扫描模块
│   │   ├── ports_data.go          # 端口扫描逻辑（并发 + 超时）
│   │   └── ports_model.go         # Model + Init/Update/View
│   ├── linuxdo/                   # LinuxDo 论坛模块
│   │   ├── linuxdo_data.go        # 数据类型、Discourse API 请求、HTML→纯文本
│   │   └── linuxdo_model.go       # Model + Init/Update/View（三级视图）
│   └── route/                     # 路由管理模块
│       ├── route_types.go         # 共享类型（RouteEntry、IfaceInfo、IfaceAddr）+ GetInterfaces()
│       ├── route_model.go         # Model + Init/Update/View（跨平台共享）
│       ├── route_data_windows.go  # Win32 API 实现（//go:build windows）
│       └── route_data_darwin.go   # netstat + route 实现（//go:build darwin）
└──
```

## 依赖

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) - 终端 UI 框架（`charm.land/bubbletea/v2`）
- [lipgloss](https://github.com/charmbracelet/lipgloss) - 终端样式库
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析
- [BurntSushi/toml](https://github.com/BurntSushi/toml) - TOML 解析
- [gopsutil/v4](https://github.com/shirou/gopsutil) - 跨平台系统信息采集（CPU/内存/磁盘/进程）
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) - 系统调用（Windows API 封装，路由管理模块使用）

## License

MIT LICENSE
