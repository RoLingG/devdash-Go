# devdash - Developer Terminal Toolkit

[简体中文](../README.md) | **English** | [繁體中文](README_TW.md)

A terminal dashboard based on [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) framework, integrating 9 practical modules with numeric key quick switching, nano-style dual-line status bar, and `?` key help panel.

## Features

| Hotkey | Module | Features |
|--------|--------|:---------|
| `1` | Git Visualization | Commit history, branch list, ahead/behind status, working directory status, hot files bar chart, repository statistics, directory scanning, recent records, **data caching** |
| `2` | Log Viewer | Paginated browsing (10 entries per page), regex filtering, level filtering, follow mode, color highlighting, error statistics, directory scanning, recent records, **data caching** |
| `3` | Weather Panel | Card-style layout, 3-day forecast, ASCII icons, scrollable view, recent cities (data from wttr.in) |
| `4` | Config Browser | JSON/YAML/TOML tree view, collapse/expand, search highlighting filter, node path display, syntax highlighting, recent records, **data caching** |
| `5` | System Monitor | CPU overall/per-core usage, memory/disk progress bars, process list, threshold color warnings, Tab to switch sub-views, **auto-refresh** |
| `6` | Port Scanner | Preset development ports scanning, custom ports, concurrent scanning, status display |
| `7` | LinuxDo Forum | Category browsing, topic list, topic details, infinite scrolling, Cookie authentication |
| `8` | Route Manager | Cross-platform routing table (Windows/macOS), add/delete static routes, network interface info, persistence |
| `9` | DevTools Toolbox | Base16/32/64/62/85/91 encode/decode, URL encoding, MD5/SHA hashing, multi-layer decode (offline pure functions, zero dependencies) |

## Installation & Running

```bash
# Clone the repository
git clone <repo-url>
cd cava_go

# Build
go build -o devdash.exe .

# Run
./devdash.exe
```

## Usage

```bash
# Default start, shows config selection interface (first time) or last saved config
./devdash.exe

# Directly specify log file
./devdash.exe --log app.log

# Directly specify config file (JSON / YAML)
./devdash.exe --config package.json
./devdash.exe --config config.yaml

# Pipe log input
cat app.log | ./devdash.exe
```

On startup, a TUI config selection interface is displayed where you can choose Git repository path, log file path, weather city, and config file path. The configuration is automatically saved as `devdash.json` (in the same directory as the exe), and loaded automatically on next startup.

After selecting configuration, a splash screen (ASCII Logo + version info) is displayed, which automatically enters the main interface after any keypress or 2 seconds.

## Persistent Configuration

- `Ctrl+S` - Save current config to `devdash.json`
- Config content includes: Git repository path, log file path, weather city, config file path
- Automatically records recently used repositories, log files, config files, cities (up to 10 entries)
- In input mode, `↑` `↓` to quickly browse and select recent records
- Automatically loads last saved config on startup

## Global Hotkeys

| Hotkey | Function |
|--------|----------|
| `1` `2` `3` `4` `5` `6` `7` `8` `9` | Switch modules |
| `Ctrl+←` / `Ctrl+→` | Cycle through modules (wraps around) |
| `?` | Help panel (shows current module hotkeys) |
| `Ctrl+S` | Save configuration |
| `Ctrl+T` | Toggle dark/light theme |
| `Ctrl+Q` / `Ctrl+C` | Exit |

### Git Module (`1`)
- `↑` `↓` / `k` `j` - Scroll
- `/` - Input repository path (supports directory scanning, lists subdirectories containing `.git`)
- In input mode `↑` `↓` - Browse recently used repositories
- `Ctrl+R` - Refresh data (smart cache: detects `.git/index` changes, uses cache if unchanged)

### Log Module (`2`)
- `↑` `↓` / `k` `j` - Move cursor within page
- `←` `→` - Previous page / Next page
- `Ctrl+↑` `Ctrl+↓` - Quick flip 10 pages
- `Ctrl+P` - Jump to page number (enter page number then confirm with Enter)
- `/` - Input file path (supports directory scanning, lists `.log` files)
- In input mode `↑` `↓` - Browse recently used log files
- Type characters - Regex filter log lines
- `Ctrl+L` - Level filtering (multi-select: All/Info/Warn/Error/Debug, stacks with regex filter)
- `Ctrl+F` - Follow mode (auto-detect file changes and refresh, press again to cancel)
- `Ctrl+U` - Clear filter
- `Esc` - Close input box
- `Ctrl+R` - Refresh log (smart cache: detects file mtime changes, uses cache if unchanged; restarts polling in follow mode)

### Weather Module (`3`)
- `↑` `↓` / `k` `j` - Scroll content
- `/` - Switch city
- In input mode `↑` `↓` - Browse recently used cities
- `Ctrl+R` - Refresh weather data

### Config Module (`4`)
- `↑` `↓` / `k` `j` - Navigate
- `Enter` - Collapse/expand node
- `/` - Input file path (supports directory scanning, lists `.json`/`.yaml`/`.yml`/`.toml` files)
- In input mode `↑` `↓` - Browse recently used config files
- `Ctrl+R` - Refresh config file (smart cache: detects file mtime changes, uses cache if unchanged)
- Type characters - Search key names/values (matching substrings highlighted)
- `Ctrl+N` / `Ctrl+B` - Jump to next/previous match
- `Ctrl+E` - Expand all nodes
- `Ctrl+W` - Collapse all nodes
- `Ctrl+U` - Clear search
- `Esc` - Close input box

### System Monitor Module (`5`)
- `Tab` - Switch sub-view (System overview ↔ Process list)
- `↑` `↓` / `k` `j` - Scroll
- `Home` / `End` - Jump to first/last
- `/` - Open input (process name filter)
- `Ctrl+R` - Manually refresh data (also starts auto-refresh polling)
- `Ctrl+U` - Clear filter
- `Esc` - Close input box

### Port Scanner Module (`6`)
- `↑` `↓` / `k` `j` - Scroll port list
- `Home` / `End` - Jump to first/last
- `/` - Add custom port
- `Ctrl+R` - Re-scan
- `Ctrl+U` - Clear custom ports
- `Esc` - Close input box

### LinuxDo Forum Module (`7`)
- `↑` `↓` / `k` `j` - Scroll/move selection
- `Home` / `End` - Jump to first/last
- `Enter` - Enter next level (category → topic list → topic details)
- `Esc` - Return to previous level
- `Ctrl+↑` `Ctrl+↓` - Quick scroll ±10 entries
- `/` - Set Cookie / User-Agent
- `Ctrl+R` - Refresh current view
- `Ctrl+U` - Clear Cookie

### Route Manager Module (`8`)
- `↑` `↓` / `k` `j` - Scroll
- `Home` / `End` - Jump to first/last
- `Tab` - Switch routing table/interface view
- `Ctrl+A` - Add static route (overlay form)
- `Ctrl+D` - Delete selected route
- `Ctrl+S` - Save current static routes to config file
- `Ctrl+L` - Load and apply routes from config file (incremental, skips existing)
- `Ctrl+R` - Refresh
- `Esc` - Close overlay

### DevTools Toolbox Module (`9`)
- `↑` `↓` / `k` `j` - Select tool
- `Home` / `End` - Jump to first/last tool
- `/` - Input text to process
- `PgUp` / `PgDn` - Scroll output result
- `Ctrl+R` - Recalculate current tool
- `Ctrl+U` - Clear input/result
- `Esc` - Close input box

## Project Structure

```
cava_go/
├── main.go                        # Main entry, top-level model & Tab switching, cross-module message routing
├── devdash.json                   # Persistent config file (auto-generated)
├── internal/
│   ├── ui/                        # Styles, colors, rendering helper functions
│   │   ├── style.go               # Color constants, style constants, TabState
│   │   ├── box.go                 # Card(), Box(), BarChart(), RenderInputCard(), RenderDirListCard()
│   │   ├── tabbar.go              # RenderTabBar(), RenderStatusBar()
│   │   ├── help_overlay.go        # RenderHelpOverlay() (centered help panel)
│   │   ├── text.go                # Rune series functions, Truncate, PadRight, TrueWidth
│   │   ├── app_config.go          # AppConfig struct, Load/Save, CfgChangedMsg
│   │   └── config_select.go       # Startup TUI config selection interface
│   ├── component/                 # Reusable UI state components
│   │   ├── input.go               # InputModel (generic input box)
│   │   └── list.go                # ListModel (generic list)
│   ├── git/                       # Git visualization module
│   │   ├── git_data.go            # GitInfo + git command execution & parsing
│   │   └── git_model.go           # Model + Init/Update/View
│   ├── log/                       # Log viewer module
│   │   ├── log_data.go            # Log loading/scanning/level detection
│   │   └── log_model.go           # Model + Init/Update/View
│   ├── weather/                   # Weather panel module
│   │   ├── weather_data.go        # wttr.in API request & parsing
│   │   └── weather_model.go       # Model + Init/Update/View
│   ├── config/                    # Config browser module
│   │   ├── config_data.go         # JSON/YAML parsing & tree building
│   │   └── config_model.go        # Model + Init/Update/View
│   ├── system/                    # System monitor module
│   │   ├── system_data.go         # gopsutil system info collection (CPU/memory/disk/process)
│   │   └── system_model.go        # Model + Init/Update/View
│   ├── ports/                     # Port scanner module
│   │   ├── ports_data.go          # Port scanning logic (concurrent + timeout)
│   │   └── ports_model.go         # Model + Init/Update/View
│   ├── linuxdo/                   # LinuxDo forum module
│   │   ├── linuxdo_data.go        # Data types, Discourse API requests, HTML→plain text
│   │   └── linuxdo_model.go       # Model + Init/Update/View (three-level view)
│   ├── route/                     # Route manager module
│       ├── route_types.go         # Shared types (RouteEntry, IfaceInfo, IfaceAddr) + GetInterfaces()
│       ├── route_model.go         # Model + Init/Update/View (cross-platform shared)
│       ├── route_data_windows.go  # Win32 API implementation (//go:build windows)
│       └── route_data_darwin.go   # netstat + route implementation (//go:build darwin)
│   └── devtools/                  # DevTools Toolbox
│       ├── devtools_data.go       # 23 built-in tools (Base/URL/Hash/Multi) + hand-written base62/base91
│       └── devtools_model.go      # Model + Init/Update/View (pure sync, two-column layout)
└──
```

## Dependencies

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) - Terminal UI framework (`charm.land/bubbletea/v2`)
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling library
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing
- [BurntSushi/toml](https://github.com/BurntSushi/toml) - TOML parsing
- [gopsutil/v4](https://github.com/shirou/gopsutil) - Cross-platform system info collection (CPU/memory/disk/process)
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) - System calls (Windows API wrapper, used by route manager module)

## License

MIT LICENSE
