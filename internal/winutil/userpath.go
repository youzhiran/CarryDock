package winutil

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
)

// ValidatePathEntry 校验并清洗要写入系统用户 Path 的单个目录路径。
func ValidatePathEntry(dir string) (string, error) {
	trimmed := strings.TrimSpace(dir)
	trimmed = stripOuterQuotes(trimmed)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", errors.New("Path 目录不能为空")
	}
	if strings.ContainsAny(trimmed, ";\r\n\x00") {
		return "", errors.New("Path 目录不能包含分号或换行符")
	}
	return NormalizePathEntry(trimmed), nil
}

// NormalizePathEntry 将单条 Path 字符串规整为统一格式，用于去重和比对。
func NormalizePathEntry(p string) string {
	trimmed := strings.TrimSpace(p)
	trimmed = stripOuterQuotes(trimmed)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	vol := filepath.VolumeName(clean)
	sep := string(filepath.Separator)
	// 保留根目录的结尾分隔符（如 C:\ 或 \ 或 /），其余路径移除末尾多余分隔符
	if clean != sep && clean != vol+sep {
		clean = strings.TrimRight(clean, `\/`)
	}
	return clean
}

func stripOuterQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// SplitPathEntries 按分号拆分系统 Path 字符串并过滤纯空项，原样保留非空条目文本。
func SplitPathEntries(raw string) []string {
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func pathEntriesEqual(a, b string) bool {
	na := NormalizePathEntry(a)
	nb := NormalizePathEntry(b)
	if na == "" || nb == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(na, nb)
	}
	return na == nb
}

// PathListContains 判断指定目录是否已存在于 Path 条目列表中。
func PathListContains(entries []string, dir string) bool {
	for _, entry := range entries {
		if pathEntriesEqual(entry, dir) {
			return true
		}
	}
	return false
}

// AddDirsToPathList 将 dirs 中尚未存在的目录追加到 existing 末尾，保留 existing 的全部原有顺序和文本。
func AddDirsToPathList(existing []string, dirs []string) (updated []string, added []string, changed bool) {
	updated = make([]string, len(existing))
	copy(updated, existing)
	for _, rawDir := range dirs {
		clean, err := ValidatePathEntry(rawDir)
		if err != nil {
			continue
		}
		if PathListContains(updated, clean) {
			continue
		}
		updated = append(updated, clean)
		added = append(added, clean)
		changed = true
	}
	return updated, added, changed
}

// RemoveDirsFromPathList 从 existing 中移除与 dirs 匹配的条目，保留其余条目原样。
func RemoveDirsFromPathList(existing []string, dirs []string) (updated []string, removed []string, changed bool) {
	if len(existing) == 0 || len(dirs) == 0 {
		updated = make([]string, len(existing))
		copy(updated, existing)
		return updated, nil, false
	}
	var cleanTargets []string
	for _, d := range dirs {
		if n := NormalizePathEntry(d); n != "" {
			cleanTargets = append(cleanTargets, n)
		}
	}
	if len(cleanTargets) == 0 {
		updated = make([]string, len(existing))
		copy(updated, existing)
		return updated, nil, false
	}
	updated = make([]string, 0, len(existing))
	for _, entry := range existing {
		matched := false
		for _, t := range cleanTargets {
			if pathEntriesEqual(entry, t) {
				matched = true
				break
			}
		}
		if matched {
			removed = append(removed, entry)
			changed = true
			continue
		}
		updated = append(updated, entry)
	}
	return updated, removed, changed
}

// IsPathUnderOrEqual 判断 target 是否等于 root 或位于 root 目录之下。
func IsPathUnderOrEqual(target, root string) bool {
	nt := NormalizePathEntry(target)
	nr := NormalizePathEntry(root)
	if nt == "" || nr == "" {
		return false
	}
	if pathEntriesEqual(nt, nr) {
		return true
	}
	prefix := nr
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(nt), strings.ToLower(prefix))
	}
	return strings.HasPrefix(nt, prefix)
}
