# devdash - 開發者終端工具箱

[简体中文](../README.md) | [English](README_EN.md) | **繁體中文**

一個基於 [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) 框架的終端儀表板，整合 8 個實用模組，數字鍵快速切換，nano 風格雙行狀態列，`?` 鍵呼出說明面板。

## 功能模組

| 快捷鍵 | 模組 | 功能 |
|--------|------|:-----|
| `1` | Git 視覺化 | 提交歷史、分支清單、ahead/behind 狀態、工作區狀態、熱門檔案柱狀圖、儲存庫統計、目錄掃描、最近記錄、**資料快取** |
| `2` | 日誌檢視器 | 分頁瀏覽（每頁 10 條）、正規表達式篩選、等級篩選、跟隨模式、彩色醒目提示、錯誤統計、目錄掃描、最近記錄、**資料快取** |
| `3` | 天氣面板 | 卡片式版面配置、3 天預報、ASCII 圖示、捲動瀏覽、最近城市（資料來源 wttr.in） |
| `4` | 設定瀏覽 | JSON/YAML/TOML 樹狀顯示、折疊/展開、搜尋醒目提示篩選、節點路徑顯示、語法醒目提示、最近記錄、**資料快取** |
| `5` | 系統監控 | CPU 整體/每核心使用率、記憶體/磁碟進度列、行程清單、閾值顏色預警、Tab 切換子檢視、**自動重新整理** |
| `6` | 埠號掃描 | 預設開發埠號掃描、自訂埠號、並行掃描、狀態顯示 |
| `7` | LinuxDo 論壇 | 分類瀏覽、主題清單、主題詳情、無限捲動、Cookie 驗證 |
| `8` | 路由管理 | 跨平台路由表（Windows/macOS）、新增/刪除靜態路由、網路介面資訊、持久化 |

## 安裝與執行

```bash
# 複製專案
git clone <repo-url>
cd cava_go

# 編譯
go build -o devdash.exe .

# 執行
./devdash.exe
```

## 使用方式

```bash
# 預設啟動，顯示設定選擇介面（首次）或上次儲存的設定
./devdash.exe

# 直接指定日誌檔案
./devdash.exe --log app.log

# 直接指定設定檔（JSON / YAML）
./devdash.exe --config package.json
./devdash.exe --config config.yaml

# 管道輸入日誌
cat app.log | ./devdash.exe
```

啟動時會顯示 TUI 設定選擇介面，可選擇 Git 儲存庫路徑、日誌檔案路徑、天氣城市、設定檔路徑。設定會自動儲存為 `devdash.json`（exe 同目錄），下次啟動自動載入上次的設定。

選擇設定後會顯示啟動閃屏（ASCII Logo + 版本資訊），任意按鍵或 2 秒後自動進入主介面。

## 持久化設定

- `Ctrl+S` - 儲存目前設定到 `devdash.json`
- 設定內容包括：Git 儲存庫路徑、日誌檔案路徑、天氣城市、設定檔路徑
- 自動記錄最近使用過的儲存庫、日誌檔案、設定檔、城市（最多 10 條）
- 輸入模式下 `↑` `↓` 可快速瀏覽和選擇最近記錄
- 啟動時自動載入上次儲存的設定

## 全域快捷鍵

| 快捷鍵 | 功能 |
|--------|------|
| `1` `2` `3` `4` `5` `6` `7` `8` | 切換模組 |
| `Ctrl+←` / `Ctrl+→` | 循環切換模組（首尾循環） |
| `?` | 說明面板（顯示目前模組快捷鍵） |
| `Ctrl+S` | 儲存設定 |
| `Ctrl+T` | 切換暗色/亮色主題 |
| `Ctrl+Q` / `Ctrl+C` | 結束 |

### Git 模組 (`1`)
- `↑` `↓` / `k` `j` - 捲動
- `/` - 輸入儲存庫路徑（支援目錄掃描，列出含 `.git` 的子目錄）
- 輸入模式下 `↑` `↓` - 瀏覽最近使用的儲存庫
- `Ctrl+R` - 重新整理資料（智慧快取：偵測 `.git/index` 變化，未變化時使用快取）

### 日誌模組 (`2`)
- `↑` `↓` / `k` `j` - 頁內游標移動
- `←` `→` - 上一頁 / 下一頁
- `Ctrl+↑` `Ctrl+↓` - 快速翻 10 頁
- `Ctrl+P` - 頁碼跳轉（輸入頁碼後按 Enter 確認）
- `/` - 輸入檔案路徑（支援目錄掃描，列出 `.log` 檔案）
- 輸入模式下 `↑` `↓` - 瀏覽最近使用的日誌檔案
- 輸入字元 - 正規表達式篩選日誌行
- `Ctrl+L` - 等級篩選（多選：All/Info/Warn/Error/Debug，與正規表達式篩選疊加）
- `Ctrl+F` - 跟隨模式（自動偵測檔案變化並重新整理，再次按取消）
- `Ctrl+U` - 清除篩選
- `Esc` - 關閉輸入框
- `Ctrl+R` - 重新整理日誌（智慧快取：偵測檔案 mtime 變化，未變化時使用快取；跟隨模式下同時重啟自旋）

### 天氣模組 (`3`)
- `↑` `↓` / `k` `j` - 捲動內容
- `/` - 切換城市
- 輸入模式下 `↑` `↓` - 瀏覽最近使用過的城市
- `Ctrl+R` - 重新整理天氣資料

### 設定模組 (`4`)
- `↑` `↓` / `k` `j` - 導覽
- `Enter` - 折疊/展開節點
- `/` - 輸入檔案路徑（支援目錄掃描，列出 `.json`/`.yaml`/`.yml`/`.toml` 檔案）
- 輸入模式下 `↑` `↓` - 瀏覽最近使用的設定檔
- `Ctrl+R` - 重新整理設定檔（智慧快取：偵測檔案 mtime 變化，未變化時使用快取）
- 輸入字元 - 搜尋鍵名/值（符合子字串醒目提示顯示）
- `Ctrl+N` / `Ctrl+B` - 跳轉到下一個/上一個符合項
- `Ctrl+E` - 展開全部節點
- `Ctrl+W` - 收起全部節點
- `Ctrl+U` - 清除搜尋
- `Esc` - 關閉輸入框

### 系統監控模組 (`5`)
- `Tab` - 切換子檢視（系統概覽 ↔ 行程清單）
- `↑` `↓` / `k` `j` - 捲動
- `Home` / `End` - 跳轉首/末
- `/` - 開啟輸入（行程名稱 filter）
- `Ctrl+R` - 手動重新整理資料（同時啟動自動重新整理輪詢）
- `Ctrl+U` - 清空 filter
- `Esc` - 關閉輸入框

### 埠號掃描模組 (`6`)
- `↑` `↓` / `k` `j` - 捲動埠號清單
- `Home` / `End` - 跳轉首/末
- `/` - 新增自訂埠號
- `Ctrl+R` - 重新掃描
- `Ctrl+U` - 清空自訂埠號
- `Esc` - 關閉輸入框

### LinuxDo 論壇模組 (`7`)
- `↑` `↓` / `k` `j` - 捲動/移動選取
- `Home` / `End` - 跳轉首/末
- `Enter` - 進入下級（分類→主題清單→主題詳情）
- `Esc` - 返回上級
- `Ctrl+↑` `Ctrl+↓` - 快速捲動 ±10 條
- `/` - 設定 Cookie / User-Agent
- `Ctrl+R` - 重新整理目前檢視
- `Ctrl+U` - 清空 Cookie

### 路由管理模組 (`8`)
- `↑` `↓` / `k` `j` - 捲動
- `Home` / `End` - 跳轉首/末
- `Tab` - 切換路由表/介面檢視
- `Ctrl+A` - 新增靜態路由（overlay 表單）
- `Ctrl+D` - 刪除選取路由
- `Ctrl+S` - 儲存目前靜態路由到設定檔
- `Ctrl+L` - 從設定檔載入並套用路由（增量，跳過已存在的）
- `Ctrl+R` - 重新整理
- `Esc` - 關閉 overlay

## 專案結構

```
cava_go/
├── main.go                        # 主入口，頂層模型與 Tab 切換，跨模組訊息路由
├── devdash.json                   # 持久化設定檔（自動產生）
├── internal/
│   ├── ui/                        # 樣式、顏色、渲染輔助函式
│   │   ├── style.go               # 顏色常數、樣式常數、TabState
│   │   ├── box.go                 # Card()、Box()、BarChart()、RenderInputCard()、RenderDirListCard()
│   │   ├── tabbar.go              # RenderTabBar()、RenderStatusBar()
│   │   ├── help_overlay.go        # RenderHelpOverlay()（置中說明面板）
│   │   ├── text.go                # Rune 系列函式、Truncate、PadRight、TrueWidth
│   │   ├── app_config.go          # AppConfig 結構、Load/Save、CfgChangedMsg
│   │   └── config_select.go       # 啟動時 TUI 設定選擇介面
│   ├── component/                 # 可複用 UI 狀態元件
│   │   ├── input.go               # InputModel（通用輸入框）
│   │   └── list.go                # ListModel（通用清單）
│   ├── git/                       # Git 視覺化模組
│   │   ├── git_data.go            # GitInfo + git 命令執行與解析
│   │   └── git_model.go           # Model + Init/Update/View
│   ├── log/                       # 日誌檢視器模組
│   │   ├── log_data.go            # 日誌載入/掃描/等級偵測
│   │   └── log_model.go           # Model + Init/Update/View
│   ├── weather/                   # 天氣面板模組
│   │   ├── weather_data.go        # wttr.in API 請求與解析
│   │   └── weather_model.go       # Model + Init/Update/View
│   ├── config/                    # 設定瀏覽器模組
│   │   ├── config_data.go         # JSON/YAML 解析與樹建構
│   │   └── config_model.go        # Model + Init/Update/View
│   ├── system/                    # 系統監控模組
│   │   ├── system_data.go         # gopsutil 系統資訊採集（CPU/記憶體/磁碟/行程）
│   │   └── system_model.go        # Model + Init/Update/View
│   ├── ports/                     # 埠號掃描模組
│   │   ├── ports_data.go          # 埠號掃描邏輯（並行 + 逾時）
│   │   └── ports_model.go         # Model + Init/Update/View
│   ├── linuxdo/                   # LinuxDo 論壇模組
│   │   ├── linuxdo_data.go        # 資料類型、Discourse API 請求、HTML→純文字
│   │   └── linuxdo_model.go       # Model + Init/Update/View（三級檢視）
│   └── route/                     # 路由管理模組
│       ├── route_types.go         # 共享類型（RouteEntry、IfaceInfo、IfaceAddr）+ GetInterfaces()
│       ├── route_model.go         # Model + Init/Update/View（跨平台共享）
│       ├── route_data_windows.go  # Win32 API 實作（//go:build windows）
│       └── route_data_darwin.go   # netstat + route 實作（//go:build darwin）
└──
```

## 相依套件

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) - 終端 UI 框架（`charm.land/bubbletea/v2`）
- [lipgloss](https://github.com/charmbracelet/lipgloss) - 終端樣式庫
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析
- [BurntSushi/toml](https://github.com/BurntSushi/toml) - TOML 解析
- [gopsutil/v4](https://github.com/shirou/gopsutil) - 跨平台系統資訊採集（CPU/記憶體/磁碟/行程）
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) - 系統呼叫（Windows API 封裝，路由管理模組使用）

## License

MIT LICENSE
