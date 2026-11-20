<div>

[**简体中文**](README_zh_CN.md)

</div>

# CarryDock

CarryDock is a portable software manager built for Windows. It has been rewritten in pure Go with **[MyGo](https://mygo.run/docs/ui) Native UI** — rendered directly on the GPU (Direct3D 11 on Windows) with zero CGO, zero WebView2 dependency, instant startup, tiny memory footprint, and full backward compatibility with existing CarryDock configuration and software list files.

## Key Features

- **Centralized Portable App Management**: Manage portable apps with instant search, list & grid views, and drag-and-drop reordering
- **Drag & Drop Import**: Drop archives (`ZIP`, `TAR`, `TAR.GZ`, `TAR.BZ2`, `TAR.XZ`, `GZ`, `BZ2`, `XZ`, `RAR`, `7Z`) or executables directly into the window
- **Safe Streaming Extraction & Archiving**: Path traversal (Zip-Slip) protection, GB18030/UTF-8 filename compatibility, and multi-level redundant folder flattening
- **Smart Executable Detection & Alternative Launch**: Auto-selects single executable or prompts with icons & file descriptions; quick launch menu for secondary executables
- **Archive & Backup Manager**: Centralized management of archives and timestamped ZIP backups, manual linking, backup-and-restore, and automatic stale-link cleanup
- **Batch Scan & Unified Association**: Batch adopt existing portable directories and link them to existing archives or create backups
- **Native Windows Integration**: High-DPI icon & `FileDescription` extraction, `ShellExecuteW` launch (UAC & special-character safe), Start Menu `.lnk` creation, Windows 11 Mica vibrancy, and system dark/light/accent theme sync
- **Bilingual & Configurable**: Chinese & English UI, custom fonts, configurable search depth & executable extensions, live file logging, and proxy-aware GitHub Release updater

## Development & Build

Requires Go 1.27+.

```powershell
# Run tests and vet
go test ./...
go vet ./...

# Run locally
go run -tags mygo_nowebview .
# Or with MyGo live reload:
go tool mygo dev

# Build standalone Windows GUI binary (without WebView2)
$env:CGO_ENABLED="0"
go build -tags mygo_nowebview -ldflags "-H=windowsgui -s -w" -o CarryDock.exe .
```
