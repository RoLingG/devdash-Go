# devdash - 开发者终端工具箱

一个基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 框架的终端仪表盘，集成 4 个实用模块，数字键快速切换。

## 功能模块

| 快捷键 | 模块 | 功能 |
|--------|------|------|
| `1` | Git 可视化 | 提交历史、分支列表、热文件柱状图、仓库统计、目录扫描 |
| `2` | 日志查看器 | 彩色高亮、正则过滤、错误统计、目录扫描 |
| `3` | 天气面板 | 卡片式布局、3 天预报、ASCII 图标、滚动浏览（数据来源 wttr.in） |
| `4` | 配置浏览 | JSON/YAML 树形展示、折叠/展开、搜索过滤、语法高亮 |

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
# 默认启动，显示 Git 可视化
./devdash.exe

# 查看日志文件
./devdash.exe --log app.log

# 浏览配置文件（JSON / YAML）
./devdash.exe --config package.json
./devdash.exe --config config.yaml

# 也可以不带参数启动，按 / 输入路径
```

## 全局快捷键

| 快捷键 | 功能 |
|--------|------|
| `1` `2` `3` `4` | 切换模块 |
| `Ctrl+R` | 手动刷新窗口布局（Windows 不支持自动检测窗口大小变化） |
| `Ctrl+Q` / `Ctrl+C` | 退出 |

### Git 模块 (`1`)
- `↑` `↓` / `k` `j` - 滚动
- `/` - 输入仓库路径（支持目录扫描，列出含 `.git` 的子目录）
- `R` - 刷新数据

**显示内容：**
- 提交历史（Hash、作者、日期、消息）
- 分支列表（当前分支高亮）
- 热文件（变更最频繁的文件柱状图）
- 仓库统计（总提交数、分支数、文件数、活跃天数）

### 日志模块 (`2`)
- `↑` `↓` / `k` `j` - 滚动
- `/` - 输入文件路径（支持目录扫描，列出 `.log` 文件）
- 输入字符 - 过滤日志行
- `Esc` - 清除过滤

### 天气模块 (`3`)
- `↑` `↓` / `k` `j` - 滚动内容
- `/` - 切换城市
- `R` - 刷新天气数据

**显示内容：**
- 当前天气（温度、体感温度、湿度、风速、能见度、UV 指数）
- ASCII 天气图标
- 3 天预报（一行多列紧凑布局，显示每 3 小时温度）

### 配置模块 (`4`)
- `↑` `↓` / `k` `j` - 导航
- `Enter` - 折叠/展开节点
- `/` - 输入文件路径（支持目录扫描，列出 `.json`/`.yaml`/`.yml` 文件）
- 输入字符 - 搜索键名/值
- `Esc` - 清除搜索

## 项目结构

```
cava_go/
├── main.go                        # 主入口，顶层模型与 Tab 切换，跨模块消息路由
├── internal/
│   ├── ui/                        # 样式、颜色、渲染辅助函数
│   │   ├── style.go               # 颜色常量、样式常量、TabState
│   │   ├── box.go                 # Card()、Box()、BarChart()
│   │   ├── tabbar.go              # RenderTabBar()、RenderHelp()
│   │   └── text.go                # Rune 系列函数、Truncate、PadRight
│   ├── component/                 # 可复用 UI 状态组件
│   │   ├── input.go               # InputModel（通用输入框）
│   │   └── list.go                # ListModel（通用列表）
│   ├── git/                       # Git 可视化模块
│   │   ├── data.go                # GitInfo + git 命令执行与解析
│   │   └── model.go               # gitModel + Init/Update/View
│   ├── log/                       # 日志查看器模块
│   │   ├── data.go                # 日志加载/扫描/级别检测
│   │   └── model.go               # logModel + Init/Update/View
│   ├── weather/                   # 天气面板模块
│   │   ├── data.go                # wttr.in API 请求与解析
│   │   └── model.go               # weatherModel + Init/Update/View
│   └── config/                    # 配置浏览器模块
│       ├── data.go                # JSON/YAML 解析与树构建
│       └── model.go               # configModel + Init/Update/View
├── docs/                          # 项目文档
└── example/                       # 示例代码
```

## 依赖

- [bubbletea](https://github.com/charmbracelet/bubbletea) v2 - 终端 UI 框架
- [lipgloss](https://github.com/charmbracelet/lipgloss) - 终端样式库
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析

## License

MIT
