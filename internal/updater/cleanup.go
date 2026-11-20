package updater

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"carrydock/internal/logger"
)

const (
	legacyUpdateScriptName   = "update_script.bat"
	legacyUpdateScriptMarker = "CarryDock Auto Update Script"
	legacyConfigBackupName   = "config_backup.bin"
)

// legacyRootFiles 列出旧版 Flutter (v0.8.x) 在应用根目录释放的运行时与构建残留文件。
// 严格采用精确白名单，确保不误删 CarryDock.exe、data/app_data.json、~archives 及任何用户文件。
var legacyRootFiles = []string{
	"flutter_windows.dll",
	"desktop_drop_plugin.dll",
	"url_launcher_windows_plugin.dll",
	"bitsdojo_window_windows_plugin.dll",
	"bitsdojo_window_windows_plugin.lib",
	"carrydock.exp",
	"carrydock.lib",
	"carrydock.pdb",
	"build_info.txt",
}

// legacyDataFiles 列出旧版 Flutter 在 data/ 下释放的运行时资源文件。
var legacyDataFiles = []string{
	"app.so",
	"icudtl.dat",
}

// legacyDataDirs 列出旧版 Flutter 在 data/ 下释放的资源目录。
var legacyDataDirs = []string{
	"flutter_assets",
}

var legacyConfigKnownKeys = []string{
	"install_path",
	"archive_path",
	"use_grid_layout",
	"executable_extensions",
	"language",
	"font_family",
	"executable_search_max_depth",
	"developer_options_enabled",
	"remove_nested_folders",
	"enable_file_logging",
	"log_file_path",
}

// CleanupLegacyFiles 清理 appDir 中由旧版 Flutter 客户端或旧版更新器遗留的冗余文件，
// 返回成功删除的相对路径列表（不触碰 CarryDock.exe、data/app_data.json、data/carrydock.log 与 ~archives）。
func CleanupLegacyFiles(appDir string) []string {
	trimmed := strings.TrimSpace(appDir)
	if trimmed == "" {
		return nil
	}
	cleanDir := filepath.Clean(trimmed)
	st, err := os.Stat(cleanDir)
	if err != nil || !st.IsDir() {
		return nil
	}

	var removed []string
	hasLegacyOrScript := false

	for _, name := range legacyRootFiles {
		p := filepath.Join(cleanDir, name)
		if info, err := os.Lstat(p); err == nil && !info.IsDir() {
			if os.Remove(p) == nil {
				removed = append(removed, name)
				hasLegacyOrScript = true
			}
		}
	}

	for _, name := range legacyDataFiles {
		rel := filepath.Join("data", name)
		p := filepath.Join(cleanDir, rel)
		if info, err := os.Lstat(p); err == nil && !info.IsDir() {
			if os.Remove(p) == nil {
				removed = append(removed, filepath.ToSlash(rel))
				hasLegacyOrScript = true
			}
		}
	}

	for _, name := range legacyDataDirs {
		rel := filepath.Join("data", name)
		p := filepath.Join(cleanDir, rel)
		if info, err := os.Lstat(p); err == nil && info.IsDir() {
			if os.RemoveAll(p) == nil {
				removed = append(removed, filepath.ToSlash(rel))
				hasLegacyOrScript = true
			}
		}
	}

	scriptPath := filepath.Join(cleanDir, legacyUpdateScriptName)
	if info, err := os.Lstat(scriptPath); err == nil && info.Mode().IsRegular() && info.Size() <= 64*1024 {
		if raw, err := os.ReadFile(scriptPath); err == nil && bytes.Contains(raw, []byte(legacyUpdateScriptMarker)) {
			if os.Remove(scriptPath) == nil {
				removed = append(removed, legacyUpdateScriptName)
				hasLegacyOrScript = true
			}
		}
	}

	backupPath := filepath.Join(cleanDir, legacyConfigBackupName)
	if info, err := os.Lstat(backupPath); err == nil && info.Mode().IsRegular() && info.Size() <= 4*1024*1024 {
		if hasLegacyOrScript || isLegacyConfigBackup(backupPath) {
			if os.Remove(backupPath) == nil {
				removed = append(removed, legacyConfigBackupName)
			}
		}
	}

	return removed
}

func isLegacyConfigBackup(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return false
	}
	if len(payload) == 0 {
		return true
	}
	for _, k := range legacyConfigKnownKeys {
		if _, ok := payload[k]; ok {
			return true
		}
	}
	return false
}

// CleanupLegacyFilesForCurrentApp 在应用启动时清理当前可执行文件目录中的旧版冗余文件。
// 若当前运行于系统临时目录（如 go run）或源码根目录（含 go.mod），则跳过清理。
func CleanupLegacyFilesForCurrentApp() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	appDir := filepath.Dir(exePath)
	if isPathUnderDir(appDir, os.TempDir()) {
		return
	}
	if _, err := os.Stat(filepath.Join(appDir, "go.mod")); err == nil {
		return
	}

	if removed := CleanupLegacyFiles(appDir); len(removed) > 0 {
		logger.Info("已清理 %d 个旧版冗余文件: %s", len(removed), strings.Join(removed, ", "))
	}
}

func isPathUnderDir(targetPath, baseDir string) bool {
	if strings.TrimSpace(targetPath) == "" || strings.TrimSpace(baseDir) == "" {
		return false
	}
	check := func(t, b string) bool {
		absT, errT := filepath.Abs(filepath.Clean(t))
		absB, errB := filepath.Abs(filepath.Clean(b))
		if errT != nil || errB != nil {
			return false
		}
		rel, err := filepath.Rel(strings.ToLower(absB), strings.ToLower(absT))
		if err != nil {
			return false
		}
		if rel == "." {
			return true
		}
		return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
	}
	if check(targetPath, baseDir) {
		return true
	}
	resolvedT, errT := filepath.EvalSymlinks(targetPath)
	resolvedB, errB := filepath.EvalSymlinks(baseDir)
	if errT == nil && errB == nil {
		return check(resolvedT, resolvedB)
	}
	return false
}
