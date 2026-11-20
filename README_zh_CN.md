# CarryDock (绿驿管家)

CarryDock (绿驿管家) 是一款专为 Windows 平台设计的绿色软件（便携软件）管理器。现已使用 **Go + [MyGo](https://mygo.run/zh/docs/ui) 原生 UI** 全面重写：单文件纯 Go（无需 CGO、无需 WebView2），Direct3D 11 GPU 直接绘制界面，秒开低占用，并与原版配置文件和数据格式完全兼容。

## 核心功能

- **软件集中管理**：自动扫描并展示安装目录内的绿色软件，支持实时关键词搜索
- **拖拽添加**：拖入压缩包（ZIP / TAR / TAR.GZ / TAR.BZ2 / TAR.XZ / GZ / BZ2 / XZ / RAR / 7Z）或可执行文件即可添加
- **自动解压与归档**：流式解压到安装目录，自动保留原始归档，支持 GBK/GB18030 中文文件名与 Zip-Slip 路径穿越防护
- **主程序智能识别**：
  - 单一可执行文件时自动设为主程序
  - 多个可执行文件时弹窗展示图标与文件说明供用户选择
- **备选启动**：自动扫描已安装目录内其他可执行文件，支持快速启动备选程序
- **视图模式与排序**：列表与网格视图一键切换；列表模式支持拖拽排序与上下微调
- **未知项识别与重新托管**：识别未托管文件夹，并在安装目录丢失时一键从归档或备份重新托管
- **归档管理**：集中查看、手动关联、创建备份、还原与删除归档/备份文件
- **批量扫描与统一关联**：批量纳管现有子目录，自动匹配旧归档或创建带时间戳的 ZIP 备份
- **Windows 深度集成**：原生提取图标与文件说明、`ShellExecuteW` 启动（支持 UAC 提权与空格路径）、开始菜单快捷方式、Windows 11 Mica 材质与系统主题/强调色跟随
- **高度可配置**：自定义安装/归档目录、可执行扩展名、搜索深度、嵌套目录自动展开、字体、中英文双语、文件日志、GitHub Releases 更新（支持系统代理）
- **开发者选项**：设置页连续点击「版本」5 次开启，可测试图标与文件说明提取

## 开发与构建

环境要求：Go 1.27+

```powershell
# 运行测试与检查
go test ./...
go vet ./...

# 开发模式运行
go run -tags mygo_nowebview .
# 或带热重载：
go tool mygo dev

# 构建单文件 Windows GUI 可执行程序（无 WebView2 依赖，自动内嵌图标与版本信息）
$env:CGO_ENABLED="0"
$env:GOFLAGS="-tags=mygo_nowebview"
go tool mygo build -platform windows/amd64 -skip-installer
# 产物位于 build/windows-amd64/CarryDock.exe
```

## 数据兼容性

- **应用配置**：`可执行文件目录/data/app_data.json`（配套 `app_data.lock` 独占锁与原子替换写入）
- **托管列表**：`归档目录/software_list.json`（配套 `software_list.lock`）
- 与原版 Flutter 版本数据格式 100% 兼容，可直接平滑替换运行。
