package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"carrydock/internal/logger"
	"carrydock/internal/winutil"
)

const (
	dataDirName  = "data"
	dbFileName   = "app_data.json"
	lockFileName = "app_data.lock"
)

// JSONStorage 负责读写位于可执行文件旁 data/app_data.json 的持久化配置。
// 同时通过进程内互斥锁与跨进程独占文件锁防止并发覆盖，并采用原子替换写入避免断电损坏。
type JSONStorage struct {
	mu       sync.Mutex
	dbPath   string
	lockPath string
	cache    map[string]any
}

var (
	defaultOnce    sync.Once
	defaultStorage *JSONStorage
)

// Default 返回全局默认存储实例（数据目录位于当前可执行文件同级 data/ 下）。
func Default() *JSONStorage {
	defaultOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			exe = "."
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		baseDir := filepath.Dir(exe)
		defaultStorage = NewAtDir(filepath.Join(baseDir, dataDirName))
	})
	return defaultStorage
}

// NewAtDir 在指定数据目录创建 JSONStorage 实例（便于单元测试隔离）。
func NewAtDir(dataDir string) *JSONStorage {
	return &JSONStorage{
		dbPath:   filepath.Join(dataDir, dbFileName),
		lockPath: filepath.Join(dataDir, lockFileName),
	}
}

// FilePath 返回 app_data.json 的绝对路径。
func (s *JSONStorage) FilePath() string {
	return s.dbPath
}

func (s *JSONStorage) readLocked(forceReload bool) map[string]any {
	if s.cache != nil && !forceReload {
		return s.cache
	}
	data := map[string]any{}
	raw, err := os.ReadFile(s.dbPath)
	if err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			logger.Error("解析配置文件失败 (%s): %v", s.dbPath, err)
			data = map[string]any{}
		}
	}
	s.cache = data
	return s.cache
}

func (s *JSONStorage) writeLocked(data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(s.dbPath), 0o755); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	// 先写入同目录临时文件再原子重命名，避免写入中途崩溃造成配置清空。
	tmpPath := fmt.Sprintf("%s.tmp.%d", s.dbPath, os.Getpid())
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.dbPath); err != nil {
		// Windows 上若目标被短时占用，尝试直接覆盖写入并清理临时文件。
		_ = os.Remove(tmpPath)
		if writeErr := os.WriteFile(s.dbPath, encoded, 0o644); writeErr != nil {
			return writeErr
		}
	}
	s.cache = data
	return nil
}

// GetRaw 读取指定键的原始值。
func (s *JSONStorage) GetRaw(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.readLocked(false)
	v, ok := data[key]
	return v, ok
}

// GetString 读取字符串配置。
func (s *JSONStorage) GetString(key string) (string, bool) {
	v, ok := s.GetRaw(key)
	if !ok || v == nil {
		return "", false
	}
	str, ok := v.(string)
	return str, ok
}

// GetBool 读取布尔配置。
func (s *JSONStorage) GetBool(key string) (bool, bool) {
	v, ok := s.GetRaw(key)
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// GetInt 读取整型配置（兼容 JSON float64）。
func (s *JSONStorage) GetInt(key string) (int, bool) {
	v, ok := s.GetRaw(key)
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// GetStringSlice 读取字符串数组配置。
func (s *JSONStorage) GetStringSlice(key string) ([]string, bool) {
	v, ok := s.GetRaw(key)
	if !ok || v == nil {
		return nil, false
	}
	switch arr := v.(type) {
	case []string:
		cp := make([]string, len(arr))
		copy(cp, arr)
		return cp, true
	case []any:
		res := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				res = append(res, str)
			}
		}
		return res, true
	default:
		return nil, false
	}
}

// Set 在跨进程文件锁保护下更新单个键值。
func (s *JSONStorage) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return winutil.WithFileLock(s.lockPath, func() error {
		data := s.readLocked(true)
		if value == nil {
			delete(data, key)
		} else {
			data[key] = value
		}
		return s.writeLocked(data)
	})
}

// SetBatch 在单次加锁与落盘中批量更新多个键值。
func (s *JSONStorage) SetBatch(updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	return winutil.WithFileLock(s.lockPath, func() error {
		data := s.readLocked(true)
		for k, v := range updates {
			if v == nil {
				delete(data, k)
			} else {
				data[k] = v
			}
		}
		return s.writeLocked(data)
	})
}
