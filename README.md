# devdash - 开发者终端工具箱

一个基于 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 框架的终端仪表盘，集成 4 个实用模块，数字键快速切换，nano 风格双行状态栏，`?` 键呼出帮助面板。

## 功能模块

| 快捷键 | 模块 | 功能 |
|--------|------|------|
| `1` | Git 可视化 | 提交历史、分支列表、热文件柱状图、仓库统计、目录扫描、最近记录 |
| `2` | 日志查看器 | 分页浏览（每页 10 条）、正则过滤、级别筛选、跟随模式、彩色高亮、错误统计、目录扫描、最近记录 |
| `3` | 天气面板 | 卡片式布局、3 天预报、ASCII 图标、滚动浏览、最近城市（数据来源 wttr.in） |
| `4` | 配置浏览 | JSON/YAML 树形展示、折叠/展开、搜索高亮过滤、语法高亮、最近记录 |

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
| `1` `2` `3` `4` | 切换模块 |
| `?` | 帮助面板（显示当前模块快捷键） |
| `Ctrl+S` | 保存配置 |
| `Ctrl+Q` / `Ctrl+C` | 退出 |

### Git 模块 (`1`)
- `↑` `↓` / `k` `j` - 滚动
- `/` - 输入仓库路径（支持目录扫描，列出含 `.git` 的子目录）
- 输入模式下 `↑` `↓` - 浏览最近使用的仓库
- `Ctrl+R` - 刷新数据

**显示内容：**
- 提交历史（Hash、作者、日期、消息）
- 分支列表（当前分支高亮）
- 热文件（变更最频繁的文件柱状图）
- 仓库统计（总提交数、分支数、文件数、活跃天数）

### 日志模块 (`2`)
- `↑` `↓` / `k` `j` - 页内光标移动
- `[` `]` - 上一页 / 下一页
- `Ctrl+↑` `Ctrl+↓` - 快速翻 10 页
- `Ctrl+P` - 页码跳转（输入页码后回车确认）
- `/` - 输入文件路径（支持目录扫描，列出 `.log` 文件）
- 输入模式下 `↑` `↓` - 浏览最近使用的日志文件
- 输入字符 - 正则过滤日志行
- `Ctrl+L` - 级别筛选（多选：All/Info/Warn/Error/Debug，与正则过滤叠加）
- `Ctrl+F` - 跟随模式（自动检测文件变化并刷新，再次按取消）
- `Enter` - 确认过滤
- `Esc` - 清除过滤
- `Ctrl+R` - 刷新日志

**显示内容：**
- 每页 10 条日志，分页浏览
- 彩色日志级别高亮（ERROR 红色、WARN 黄色、INFO 绿色、DEBUG 灰色）
- 正则表达式过滤 + 级别筛选（可叠加）
- 页码信息（当前页/总页数，条目范围）
- 底部统计（各级别日志数量）

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
- `/` - 输入文件路径（支持目录扫描，列出 `.json`/`.yaml`/`.yml` 文件）
- 输入模式下 `↑` `↓` - 浏览最近使用的配置文件
- `Ctrl+R` - 刷新配置文件
- 输入字符 - 搜索键名/值（匹配子串高亮显示）
- `Ctrl+N` / `Ctrl+B` - 跳转到下一个/上一个匹配项
- `Esc` - 清除搜索

## 项目结构

```
cava_go/
├── main.go                        # 主入口，顶层模型与 Tab 切换，跨模块消息路由
├── devdash.json                   # 持久化配置文件（自动生成）
├── internal/
│   ├── ui/                        # 样式、颜色、渲染辅助函数
│   │   ├── style.go               # 颜色常量、样式常量、TabState
│   │   ├── box.go                 # Card()、Box()、BarChart()、RenderInputCard()、RenderDirListCard()
│   │   ├── tabbar.go              # RenderTabBar()、RenderStatusBar()（双行 nano 风格状态栏）
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
│   │   └── log_model.go           # Model + Init/Update/View（分页渲染）
│   ├── weather/                   # 天气面板模块
│   │   ├── weather_data.go        # wttr.in API 请求与解析
│   │   └── weather_model.go       # Model + Init/Update/View
│   └── config/                    # 配置浏览器模块
│       ├── config_data.go         # JSON/YAML 解析与树构建
│       └── config_model.go        # Model + Init/Update/View
├── docs/                          # 项目文档
└── example/                       # 示例代码
```

## 依赖

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) - 终端 UI 框架（`charm.land/bubbletea/v2`）
- [lipgloss](https://github.com/charmbracelet/lipgloss) - 终端样式库
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析

## License

MIT
