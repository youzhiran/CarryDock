package settings

import (
	"path/filepath"
	"strings"
)

// IsAbsOrHasVolume 判断路径是否为绝对路径、包含盘符/UNC 卷名，或以根路径分隔符开头。
func IsAbsOrHasVolume(p string) bool {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	if filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" {
		return true
	}
	if filepath.Separator == '\\' && strings.HasPrefix(clean, `\`) {
		return true
	}
	return false
}

// ResolvePath 将原始路径解析为运行时可用的路径：
// - 空路径返回 ""；
// - 绝对路径或含卷标路径清洗后原样返回；
// - 相对路径基于 baseDir（即 CarryDock.exe 所在目录）拼接为绝对路径。
func ResolvePath(baseDir, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	base := strings.TrimSpace(baseDir)
	if IsAbsOrHasVolume(clean) || base == "" {
		return clean
	}
	return filepath.Clean(filepath.Join(filepath.Clean(base), clean))
}

// ToRelativePath 尝试将 raw 转换为相对于 baseDir 的便携式路径（如 ".\Apps"、"."、"..\Portable"）。
// 若跨盘符或无法建立相对关系，返回解析后的绝对路径与 false。
func ToRelativePath(baseDir, raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", true
	}
	base := strings.TrimSpace(baseDir)
	if base == "" {
		return filepath.Clean(filepath.FromSlash(trimmed)), false
	}

	absBase := filepath.Clean(base)
	if resolved, err := filepath.Abs(absBase); err == nil {
		absBase = resolved
	}
	absTarget := ResolvePath(absBase, trimmed)
	if resolved, err := filepath.Abs(absTarget); err == nil {
		absTarget = resolved
	}

	volBase := filepath.VolumeName(absBase)
	volTarget := filepath.VolumeName(absTarget)
	if !strings.EqualFold(volBase, volTarget) {
		return absTarget, false
	}
	// Windows 上盘符大小写不同会导致 filepath.Rel 报错，统一为基准目录盘符大小写
	if volBase != "" && volTarget != volBase && strings.HasPrefix(absTarget, volTarget) {
		absTarget = volBase + absTarget[len(volTarget):]
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return absTarget, false
	}
	if rel == "." {
		return ".", true
	}
	upPrefix := ".." + string(filepath.Separator)
	if rel == ".." || strings.HasPrefix(rel, upPrefix) {
		return rel, true
	}
	return "." + string(filepath.Separator) + rel, true
}

// CanRelativize 判断给定路径是否可转换为相对 baseDir 的路径（空路径视为可转换）。
func CanRelativize(baseDir, raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	_, ok := ToRelativePath(baseDir, raw)
	return ok
}

// FormatPathForMode 按指定模式格式化待持久化的路径。
// 当 useRelative 为 true 时尽量转为相对路径（返回 isRelative=true），跨盘符则保留绝对路径（isRelative=false）；
// 当 useRelative 为 false 时统一转为绝对路径（isRelative=false）。
func FormatPathForMode(baseDir, raw string, useRelative bool) (stored string, isRelative bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	if useRelative {
		return ToRelativePath(baseDir, trimmed)
	}
	return ResolvePath(baseDir, trimmed), false
}
