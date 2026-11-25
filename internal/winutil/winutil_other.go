//go:build !windows

package winutil

import (
	"errors"
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
	return LaunchExecutableWithOptions(exePath, fallbackWorkDir, LaunchOptions{})
}

// LaunchExecutableWithOptions 在非 Windows 平台按附加参数和临时环境变量启动程序。
func LaunchExecutableWithOptions(exePath, fallbackWorkDir string, opts LaunchOptions) error {
	var args []string
	if trimmed := strings.TrimSpace(opts.Args); trimmed != "" {
		args = strings.Fields(trimmed)
	}
	cmd := exec.Command(exePath, args...)
	workDir := filepath.Dir(exePath)
	if workDir == "" || workDir == "." {
		workDir = fallbackWorkDir
	}
	cmd.Dir = workDir
	if len(opts.Env) > 0 {
		builtins := map[string]string{
			"EXE_DIR": filepath.Dir(exePath),
		}
		if strings.TrimSpace(opts.InstallDir) != "" {
			builtins["INSTALL_DIR"] = filepath.Clean(strings.TrimSpace(opts.InstallDir))
		}
		cmd.Env = MergeEnv(os.Environ(), opts.Env, builtins, false)
	}
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
	return CreateShortcutWithArgs(targetPath, shortcutPath, description, "")
}

// CreateShortcutWithArgs 非 Windows 平台返回不支持错误。
func CreateShortcutWithArgs(targetPath, shortcutPath, description, args string) error {
	return fmt.Errorf("仅 Windows 平台支持创建快捷方式: %w", errors.ErrUnsupported)
}

// ListUserPath 非 Windows 平台返回不支持错误。
func ListUserPath() ([]string, error) {
	return nil, fmt.Errorf("仅 Windows 平台支持用户 Path 管理: %w", errors.ErrUnsupported)
}

// AddUserPathDirs 非 Windows 平台返回不支持错误。
func AddUserPathDirs(dirs []string) ([]string, error) {
	return nil, fmt.Errorf("仅 Windows 平台支持用户 Path 管理: %w", errors.ErrUnsupported)
}

// RemoveUserPathDirs 非 Windows 平台返回不支持错误。
func RemoveUserPathDirs(dirs []string) ([]string, error) {
	return nil, fmt.Errorf("仅 Windows 平台支持用户 Path 管理: %w", errors.ErrUnsupported)
}

// Read7ZipRegistryPaths 非 Windows 平台返回空切片。
func Read7ZipRegistryPaths() []RegistryResult { return nil }
