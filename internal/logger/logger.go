package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level 表示日志输出级别。
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String 返回用于日志行前缀的标准级别名称。
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Key 返回用于配置存储的小写级别标识。
func (l Level) Key() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

// ParseLevel 将字符串解析为日志级别，未知值默认回退为 LevelInfo。
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info", "":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

const maxLogFileBytes = 5 * 1024 * 1024 // 单个日志文件最大 5 MB，超出后自动轮转为 .1

// Manager 管理控制台与可选文件日志输出，支持运行时热更新级别与文件目标。
type Manager struct {
	mu                sync.Mutex
	level             Level
	enableFileLogging bool
	logFilePath       string
	file              *os.File
	fileSize          int64
	consoleWriter     io.Writer
}

var defaultManager = &Manager{
	level:         LevelInfo,
	consoleWriter: os.Stdout,
}

// Configure 更新全局日志配置（是否写入文件及日志文件路径），保持当前日志级别不变。
// 文件日志不写入 ANSI 转义色码，保持纯文本可读。
func Configure(enableFileLogging bool, logFilePath string) error {
	return defaultManager.Configure(enableFileLogging, logFilePath)
}

// ConfigureWithLevel 同时更新全局日志的文件输出配置与最低输出级别。
func ConfigureWithLevel(enableFileLogging bool, logFilePath string, level string) error {
	return defaultManager.ConfigureWithLevel(enableFileLogging, logFilePath, level)
}

// SetLevel 动态更新全局日志最低输出级别。
func SetLevel(level string) {
	defaultManager.SetLevel(level)
}

// GetLevel 返回当前全局日志最低输出级别。
func GetLevel() Level {
	return defaultManager.GetLevel()
}

// SetLevel 动态更新当前 Manager 的最低输出级别。
func (m *Manager) SetLevel(level string) {
	m.mu.Lock()
	m.level = ParseLevel(level)
	m.mu.Unlock()
}

// GetLevel 返回当前 Manager 的最低输出级别。
func (m *Manager) GetLevel() Level {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.level
}

// Configure 更新当前 Manager 的日志输出目标（保留当前日志级别）。
func (m *Manager) Configure(enableFileLogging bool, logFilePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configureLocked(enableFileLogging, logFilePath)
}

// ConfigureWithLevel 同时更新当前 Manager 的日志输出目标与最低日志级别。
func (m *Manager) ConfigureWithLevel(enableFileLogging bool, logFilePath string, level string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.level = ParseLevel(level)
	return m.configureLocked(enableFileLogging, logFilePath)
}

func (m *Manager) configureLocked(enableFileLogging bool, logFilePath string) error {
	if m.file != nil {
		_ = m.file.Close()
		m.file = nil
		m.fileSize = 0
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
	if st, statErr := f.Stat(); statErr == nil {
		m.fileSize = st.Size()
	}
	m.file = f
	return nil
}

func (m *Manager) rotateIfNeededLocked(incomingBytes int64) {
	if m.file == nil || m.logFilePath == "" || m.fileSize+incomingBytes <= maxLogFileBytes {
		return
	}
	_ = m.file.Close()
	m.file = nil
	m.fileSize = 0

	backupPath := m.logFilePath + ".1"
	_ = os.Remove(backupPath)
	_ = os.Rename(m.logFilePath, backupPath)

	if f, err := os.OpenFile(m.logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		m.file = f
	}
}

// Close 关闭当前打开的日志文件句柄。
func Close() {
	defaultManager.Close()
}

// Close 关闭当前 Manager 打开的日志文件句柄。
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.file != nil {
		_ = m.file.Close()
		m.file = nil
		m.fileSize = 0
	}
}

func (m *Manager) logf(lvl Level, format string, args ...any) {
	m.mu.Lock()
	if lvl < m.level {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	now := time.Now()
	lvlStr := lvl.String()
	consoleLine := fmt.Sprintf("%s [%-5s] %s\n", now.Format("15:04:05"), lvlStr, msg)
	fileLine := fmt.Sprintf("%s [%-5s] %s\n", now.Format("2006-01-02 15:04:05"), lvlStr, msg)

	m.mu.Lock()
	defer m.mu.Unlock()
	if lvl < m.level {
		return
	}

	cw := m.consoleWriter
	if cw == nil {
		cw = os.Stdout
	}
	_, _ = io.WriteString(cw, consoleLine)
	if m.enableFileLogging && m.file != nil {
		m.rotateIfNeededLocked(int64(len(fileLine)))
		if m.file != nil {
			if n, err := io.WriteString(m.file, fileLine); err == nil {
				m.fileSize += int64(n)
			}
		}
	}
}

// Debug 记录调试级别日志。
func Debug(format string, args ...any) { defaultManager.logf(LevelDebug, format, args...) }

// Info 记录普通信息日志。
func Info(format string, args ...any) { defaultManager.logf(LevelInfo, format, args...) }

// Warn 记录警告日志。
func Warn(format string, args ...any) { defaultManager.logf(LevelWarn, format, args...) }

// Error 记录错误日志。
func Error(format string, args ...any) { defaultManager.logf(LevelError, format, args...) }
