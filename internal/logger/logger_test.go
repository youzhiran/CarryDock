package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLevelAndString(t *testing.T) {
	cases := []struct {
		in   string
		want Level
		str  string
		key  string
	}{
		{"debug", LevelDebug, "DEBUG", "debug"},
		{"  DEBUG ", LevelDebug, "DEBUG", "debug"},
		{"info", LevelInfo, "INFO", "info"},
		{"", LevelInfo, "INFO", "info"},
		{"unknown", LevelInfo, "INFO", "info"},
		{"warn", LevelWarn, "WARN", "warn"},
		{"warning", LevelWarn, "WARN", "warn"},
		{"ERROR", LevelError, "ERROR", "error"},
	}
	for _, tc := range cases {
		got := ParseLevel(tc.in)
		if got != tc.want || got.String() != tc.str || got.Key() != tc.key {
			t.Fatalf("ParseLevel(%q) = %v (%s/%s), want %v (%s/%s)", tc.in, got, got.String(), got.Key(), tc.want, tc.str, tc.key)
		}
	}
}

func TestManagerLevelFilteringAndFileLogging(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	var consoleBuf bytes.Buffer
	m := &Manager{
		level:         LevelInfo,
		consoleWriter: &consoleBuf,
	}
	if err := m.ConfigureWithLevel(true, logPath, "warn"); err != nil {
		t.Fatalf("ConfigureWithLevel failed: %v", err)
	}
	defer m.Close()

	m.logf(LevelDebug, "debug msg %d", 1)
	m.logf(LevelInfo, "info msg %d", 2)
	m.logf(LevelWarn, "warn msg %d", 3)
	m.logf(LevelError, "error msg %d", 4)

	consoleOut := consoleBuf.String()
	if strings.Contains(consoleOut, "debug msg") || strings.Contains(consoleOut, "info msg") {
		t.Fatalf("expected debug/info filtered out at warn level, got: %s", consoleOut)
	}
	if !strings.Contains(consoleOut, "[WARN ] warn msg 3") || !strings.Contains(consoleOut, "[ERROR] error msg 4") {
		t.Fatalf("expected warn/error present at warn level, got: %s", consoleOut)
	}

	// 动态切换为 debug 级别后应能记录 debug 与 info
	m.SetLevel("debug")
	if m.GetLevel() != LevelDebug {
		t.Fatalf("expected LevelDebug after SetLevel, got %v", m.GetLevel())
	}
	m.logf(LevelDebug, "debug after switch")
	m.Close()

	fileBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) failed: %v", logPath, err)
	}
	fileOut := string(fileBytes)
	if !strings.Contains(fileOut, "warn msg 3") || !strings.Contains(fileOut, "error msg 4") || !strings.Contains(fileOut, "debug after switch") {
		t.Fatalf("unexpected log file content: %s", fileOut)
	}
	if strings.Contains(fileOut, "debug msg 1") {
		t.Fatalf("expected early debug msg absent from log file: %s", fileOut)
	}
}

func TestManagerAutoRotation(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "rotate.log")
	var consoleBuf bytes.Buffer
	m := &Manager{
		level:         LevelInfo,
		consoleWriter: &consoleBuf,
	}
	if err := m.ConfigureWithLevel(true, logPath, "info"); err != nil {
		t.Fatalf("ConfigureWithLevel failed: %v", err)
	}
	defer m.Close()

	// 模拟已达到接近上限的文件大小，触发下一次写入时轮转
	m.logf(LevelInfo, "before rotate")
	m.mu.Lock()
	m.fileSize = maxLogFileBytes
	m.mu.Unlock()

	m.logf(LevelInfo, "after rotate")
	m.Close()

	currBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read current log failed: %v", err)
	}
	bakBytes, err := os.ReadFile(logPath + ".1")
	if err != nil {
		t.Fatalf("read rotated backup log (.1) failed: %v", err)
	}
	if !strings.Contains(string(bakBytes), "before rotate") {
		t.Fatalf("expected rotated .1 file to contain 'before rotate', got %q", string(bakBytes))
	}
	if !strings.Contains(string(currBytes), "after rotate") || strings.Contains(string(currBytes), "before rotate") {
		t.Fatalf("expected new log file to only contain 'after rotate', got %q", string(currBytes))
	}
}
