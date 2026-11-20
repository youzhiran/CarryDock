# Repository Guidelines

## 项目结构与模块组织

本项目已采用 **Go + [MyGo](https://mygo.run/zh/docs/ui) 原生 GPU UI** 重写，纯 Go 实现（`CGO_ENABLED=0`，无需 WebView2 或前端构建）。

- `main.go`：应用入口，负责初始化日志、配置、核心服务及 MyGo 主窗口。
- `mygo.json`：MyGo 应用打包与元数据配置（图标位于 `assets/icon.png`）。
- `internal/app/`：MyGo 原生 UI 视图层与交互状态：
  - `app.go`：主窗口框架、主题、标题栏、可折叠侧边栏与异步图标/元数据缓存。
  - `home_view.go`：主页（软件列表/网格、拖拽添加、拖拽/箭头排序、右键菜单、批量扫描）。
  - `archive_view.go`：归档管理页（归档与备份列表、手动关联、还原、创建备份、删除）。
  - `settings_view.go`：设置页（存储、归档处理、7-Zip 检测、可执行识别、外观字体/语言、文件日志、关于与更新）。
  - `dev_view.go`：开发者选项页（图标与文件说明提取测试）。
  - `modals.go`：全部原生模态对话框集中管理。
  - `icons.go`：内置矢量 SVG 图标集。
- `internal/software/`：核心业务层（托管列表读写、压缩包解压与扁平化、去重冲突、重新托管、重命名、批量扫描归档）。
- `internal/archive/`：多格式流式归档解压（ZIP/TAR/GZ/BZ2/XZ/RAR/7Z）、Zip-Slip 防护、GB18030 中文文件名兼容与 7-Zip 检测。
- `internal/settings/` & `internal/storage/`：跨进程文件锁保护、原子写入的 JSON 配置存储（完全兼容旧版 `data/app_data.json`）。
- `internal/winutil/`：纯 Go Win32 API 交互（`version.dll` 文件说明、`ExtractIconExW`/`DrawIconEx`、`ShellExecuteW` 启动、`LockFileEx` 锁、开始菜单 `.lnk` 快捷方式、7-Zip 注册表查询）。
- `internal/updater/`：GitHub Releases 更新检查、可取消流式下载（支持系统代理）与安全自更新。
- `internal/i18n/`：中英文双语界面文案。
- `internal/logger/`：控制台与纯文本文件日志（支持运行时热切换）。

## 构建、测试与开发命令

- 开发运行：`go tool mygo dev` 或 `go run -tags mygo_nowebview .`
- 单元测试与 UI 测试：`go test ./...`
- 静态检查：`go vet ./...` 与 `go tool mygo vet`
- 发布构建（Windows 无 WebView 精简版，自动内嵌 PE 图标、DPI 清单与版本信息）：
  - `$env:CGO_ENABLED="0"; $env:GOFLAGS="-tags=mygo_nowebview"; go tool mygo build -platform windows/amd64 -skip-installer`
  - 输出位于 `build/windows-amd64/CarryDock.exe`

## 编码规范

- 遵循标准 `gofmt` 与 Go 惯例；避免在结构或 goroutine 中保存 `ui.Element` 临时构建值。
- 后台耗时操作（IO、解压、网络、系统对话框）一律在 goroutine 中执行，通过 `win.Update(...)` 回到主线程更新 UI 状态。
- 重要逻辑保持清晰简洁的中文注释。
