# devdash - 开发者终端工具箱

一个基于 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 框架的终端仪表盘，集成 7 个实用模块，数字键快速切换，nano 风格双行状态栏，`?` 键呼出帮助面板。

## 功能模块

| 快捷键 | 模块 | 功能 |
|--------|------|:-----|
| `1` | Git 可视化 | 提交历史、分支列表、ahead/behind 状态、工作区状态、热文件柱状图、仓库统计、目录扫描、最近记录、**数据缓存** |
| `2` | 日志查看器 | 分页浏览（每页 10 条）、正则过滤、级别筛选、跟随模式、彩色高亮、错误统计、目录扫描、最近记录、**数据缓存** |
| `3` | 天气面板 | 卡片式布局、3 天预报、ASCII 图标、滚动浏览、最近城市（数据来源 wttr.in） |
| `4` | 配置浏览 | JSON/YAML/TOML 树形展示、折叠/展开、搜索高亮过滤、节点路径显示、语法高亮、最近记录、**数据缓存** |
| `5` | 系统监控 | CPU 总体/每核心使用率、内存/磁盘进度条、进程列表、阈值颜色预警、Tab 切换子视图 |
| `6` | 端口扫描 | 预设开发端口扫描、自定义端口、并发扫描、状态显示 |
| `7` | LinuxDo 论坛 | 分类浏览、帖子列表、帖子详情、无限滚动、Cookie 认证 |

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

## 持久化配置

- `Ctrl+S` - 保存当前配置到 `devdash.json`
- 配置内容包括：Git 仓库路径、日志文件路径、天气城市、配置文件路径
- 自动记录最近使用过的仓库、日志文件、配置文件、城市（最多 10 条）
- 输入模式下 `↑` `↓` 可快速浏览和选择最近记录
- 启动时自动加载上次保存的配置

## 全局快捷键

| 快捷键 | 功能 |
|--------|------|
| `1` `2` `3` `4` `5` `6` `7` | 切换模块 |
| `?` | 帮助面板（显示当前模块快捷键） |
| `Ctrl+S` | 保存配置 |
| `Ctrl+Q` / `Ctrl+C` | 退出 |

### Git 模块 (`1`)
- `↑` `↓` / `k` `j` - 滚动
- `/` - 输入仓库路径（支持目录扫描，列出含 `.git` 的子目录）
- 输入模式下 `↑` `↓` - 浏览最近使用的仓库
- `Ctrl+R` - 刷新数据（智能缓存：检测 `.git/index` 变化，未变化时使用缓存）

**显示内容：**
- 提交历史（Hash、作者、日期、消息）
- 分支列表（当前分支高亮）
- **ahead/behind 状态**（header 显示 `↑3 ↓2` 徽章，表示本地领先/落后远程的提交数）
- **工作区 dirty 状态**（header 显示 `📝 3M 1A 2D 1??` 格式，分别表示 Modified/Added/Deleted/Untracked 文件数）
- 热文件（变更最频繁的文件柱状图）
- 仓库统计（总提交数、分支数、文件数、活跃天数）
- **并发加载**（6 个 git 命令并发执行，加载速度显著提升）
- **空状态提示**（无 commit 时显示 "📭 仓库暂无提交记录"）

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
- `Ctrl+R` - 刷新日志（智能缓存：检测文件 mtime 变化，未变化时使用缓存）

**显示内容：**
- 每页 10 条日志，分页浏览
- 彩色日志级别高亮（ERROR 红色、WARN 黄色、INFO 绿色、DEBUG 灰色）
- 正则表达式过滤 + 级别筛选（可叠加）
- 匹配子串黄色背景高亮
- 页码信息（当前页/总页数，条目范围）
- 底部统计（各级别日志数量）
- **空状态提示**（无匹配结果时显示 "🔍 没有找到匹配的日志行，尝试修改过滤条件"）
- **友好错误提示**（文件不存在/权限错误时显示红色错误信息 + 操作提示）
- **大文件保护**（超过 100MB 或 100 万行自动拒绝，50MB+ 显示警告）

### 天气模块 (`3`)
- `↑` `↓` / `k` `j` - 滚动内容
- `/` - 切换城市
- 输入模式下 `↑` `↓` - 浏览最近使用过的城市
- `Ctrl+R` - 刷新天气数据

**显示内容：**
- 当前天气（温度、体感温度、湿度、风速、能见度、UV 指数）
- ASCII 天气图标
- 3 天预报（一行多列紧凑布局，显示每 3 小时温度）

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

**显示内容：**
- 支持 JSON/YAML/TOML 格式
- 树形结构展示，折叠/展开节点
- 搜索关键词高亮，快速跳转匹配项
- 当前节点完整路径显示（header 区域 `📍 key1 / key2 / key3`）
- 语法高亮（字符串、数字、布尔值、null）
- **空状态提示**（空配置文件时显示 "📄 配置文件为空"）
- **友好错误提示**（文件解析失败时显示红色错误信息 + 操作提示）

### 系统监控模块 (`5`)
- `Tab` - 切换子视图（系统概览 ↔ 进程列表）
- `↑` `↓` / `k` `j` - 滚动
- `Home` / `End` - 跳转首/末
- `/` - 打开输入（进程名 filter）
- `Ctrl+R` - 刷新数据
- `Ctrl+U` - 清空 filter
- `Esc` - 关闭输入框

**显示内容：**
- CPU 总体使用率 + 每核心使用率（4 列网格平铺，窄终端自动 2 列）
- 内存使用（已用/总量 + 进度条）
- 磁盘使用（各分区已用/总量 + 进程条）
- 进程列表（PID、名称、CPU%、内存 MB，按 CPU 降序）
- **阈值颜色系统**（<70% 灰色、>=70% 黄色、>=90% 红色）
- **CPU 核心奇偶标签色**（偶数蓝色、奇数粉色，便于区分）

### 端口扫描模块 (`6`)
- `↑` `↓` / `k` `j` - 滚动端口列表
- `Home` / `End` - 跳转首/末
- `/` - 添加自定义端口
- `Ctrl+R` - 重新扫描
- `Ctrl+U` - 清空自定义端口
- `Esc` - 关闭输入框

**显示内容：**
- 17 个预设开发端口（SSH、HTTP、HTTPS、MySQL、PostgreSQL、Redis 等）
- 并发扫描（500ms 超时），开放端口绿色 ✓、关闭端口灰色 ✗
- 用户自定义端口支持

### LinuxDo 论坛模块 (`7`)
- `↑` `↓` / `k` `j` - 滚动/移动选中
- `Home` / `End` - 跳转首/末
- `Enter` - 进入下级（分类→帖子列表→帖子详情）
- `Esc` - 返回上级
- `Ctrl+↑` `Ctrl+↓` - 快速滚动 ±10 条
- `/` - 设置 Cookie / User-Agent
- `Ctrl+R` - 刷新当前视图
- `Ctrl+U` - 清空 Cookie

**显示内容：**
- 三级导航：分类列表 → 帖子列表 → 帖子详情
- 分类列表（首项 "📌 Latest" 显示全站最新帖子）
- 帖子列表（标题、回复数、浏览量，无限滚动自动加载下一页）
- 帖子详情（HTML→纯文本转换，无限滚动自动加载全部回复）
- 帖子总数统计（header 显示 `(已加载/总数)` 格式）

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
│   └── ports/                     # 端口扫描模块
│       ├── ports_data.go          # 端口扫描逻辑（并发 + 超时）
│       └── ports_model.go         # Model + Init/Update/View
│   └── linuxdo/                   # LinuxDo 论坛模块
│       ├── linuxdo_data.go        # 数据类型、Discourse API 请求、HTML→纯文本
│       └── linuxdo_model.go       # Model + Init/Update/View（三级视图）
└──
```

## 依赖

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) - 终端 UI 框架（`charm.land/bubbletea/v2`）
- [lipgloss](https://github.com/charmbracelet/lipgloss) - 终端样式库
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析
- [BurntSushi/toml](https://github.com/BurntSushi/toml) - TOML 解析
- [gopsutil/v4](https://github.com/shirou/gopsutil) - 跨平台系统信息采集（CPU/内存/磁盘/进程）

## License

MIT LICENSE
