package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager 管理控制台与可选文件日志输出，支持运行时热更新配置。
type Manager struct {
	mu                sync.Mutex
	enableFileLogging bool
	logFilePath       string
	file              *os.File
}

var defaultManager = &Manager{}

// Configure 更新全局日志配置（是否写入文件及日志文件路径）。
// 文件日志不写入 ANSI 转义色码，保持纯文本可读。
func Configure(enableFileLogging bool, logFilePath string) error {
	return defaultManager.Configure(enableFileLogging, logFilePath)
}

// Configure 更新当前 Manager 的日志输出目标。
func (m *Manager) Configure(enableFileLogging bool, logFilePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.file != nil {
		_ = m.file.Close()
		m.file = nil
	}

	m.enableFileLogging = enableFileLogging
	m.logFilePath = logFilePath

	if !enableFileLogging || logFilePath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logFilePath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	m.file = f
	return nil
}

// Close 关闭当前打开的日志文件句柄。
func Close() {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	if defaultManager.file != nil {
		_ = defaultManager.file.Close()
		defaultManager.file = nil
	}
}

func (m *Manager) logf(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	now := time.Now()
	consoleLine := fmt.Sprintf("%s [%-5s] %s\n", now.Format("15:04:05"), level, msg)
	fileLine := fmt.Sprintf("%s [%-5s] %s\n", now.Format("2006-01-02 15:04:05"), level, msg)

	m.mu.Lock()
	defer m.mu.Unlock()

	_, _ = io.WriteString(os.Stdout, consoleLine)
	if m.enableFileLogging && m.file != nil {
		_, _ = io.WriteString(m.file, fileLine)
	}
}

// Debug 记录调试级别日志。
func Debug(format string, args ...any) { defaultManager.logf("DEBUG", format, args...) }

// Info 记录普通信息日志。
func Info(format string, args ...any) { defaultManager.logf("INFO", format, args...) }

// Warn 记录警告日志。
func Warn(format string, args ...any) { defaultManager.logf("WARN", format, args...) }

// Error 记录错误日志。
func Error(format string, args ...any) { defaultManager.logf("ERROR", format, args...) }
