//go:build !windows

package winutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RegistryResult 表示单条 7-Zip 注册表查询结果（非 Windows 占位）。
type RegistryResult struct {
	Label   string
	DirPath string
	ExePath string
	Exists  bool
	Err     string
}

// SamePath 比较两个路径是否相同。
func SamePath(a, b string) bool {
	return filepath.Clean(strings.TrimSpace(a)) == filepath.Clean(strings.TrimSpace(b))
}

// WithFileLock 在非 Windows 平台简单执行回调。
func WithFileLock(lockPath string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return err
	}
	return fn()
}

// FileMetadata 保存可执行文件的描述、产品信息与原生版本信息（非 Windows 占位）。
type FileMetadata struct {
	Description      string
	ProductName      string
	OriginalFilename string
	Version          string
}

// GetFileMetadata 非 Windows 平台返回空元信息。
func GetFileMetadata(exePath string) FileMetadata { return FileMetadata{} }

// GetFileDescription 非 Windows 平台返回空。
func GetFileDescription(exePath string) string { return "" }

// GetFileVersion 非 Windows 平台返回空。
func GetFileVersion(exePath string) string { return "" }

// ExtractIconPNG 非 Windows 平台返回不支持错误。
func ExtractIconPNG(exePath string, preferredSize int) ([]byte, error) {
	return nil, fmt.Errorf("仅 Windows 平台支持提取 EXE 图标")
}

// LaunchExecutable 在非 Windows 平台直接启动程序。
func LaunchExecutable(exePath, fallbackWorkDir string) error {
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	return cmd.Start()
}

// HideConsole 在非 Windows 平台无操作。
func HideConsole(cmd *exec.Cmd) {}

// LaunchBatchScript 非 Windows 平台返回不支持错误。
func LaunchBatchScript(scriptPath string) error {
	return fmt.Errorf("仅 Windows 平台支持启动批处理脚本: %s", scriptPath)
}

// OpenDirectory 打开目录（占位实现）。
func OpenDirectory(dirPath string) error {
	return fmt.Errorf("当前平台不支持资源管理器打开: %s", dirPath)
}

// RevealInExplorer 在文件管理器中定位文件（占位实现）。
func RevealInExplorer(filePath string) error {
	return fmt.Errorf("当前平台不支持定位文件: %s", filePath)
}

// StartMenuDir 非 Windows 平台返回不支持错误。
func StartMenuDir() (string, error) {
	return "", fmt.Errorf("仅 Windows 平台支持开始菜单")
}

// CreateShortcut 非 Windows 平台返回不支持错误。
func CreateShortcut(targetPath, shortcutPath, description string) error {
	return fmt.Errorf("仅 Windows 平台支持创建快捷方式")
}

// Read7ZipRegistryPaths 非 Windows 平台返回空切片。
func Read7ZipRegistryPaths() []RegistryResult { return nil }
