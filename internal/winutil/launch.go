package winutil

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// LaunchOptions 描述启动托管软件时的附加命令行参数与临时环境变量。
type LaunchOptions struct {
	Args       string
	Env        map[string]string
	InstallDir string
}

// ParseEnvText 解析多行 KEY=VALUE 环境变量文本。
// 空行与以 # 开头的注释行会被忽略；非法格式返回带 1-based 行号的错误。
func ParseEnvText(text string) (map[string]string, error) {
	lines := strings.Split(text, "\n")
	var out map[string]string
	for idx, raw := range lines {
		lineNum := idx + 1
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("第 %d 行格式无效，请使用 KEY=VALUE", lineNum)
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" || strings.ContainsAny(key, " \t\r\n\x00") {
			return nil, fmt.Errorf("第 %d 行环境变量名无效: %q", lineNum, key)
		}
		val := line[eq+1:]
		if strings.ContainsRune(val, '\x00') {
			return nil, fmt.Errorf("第 %d 行环境变量值包含非法空字符", lineNum)
		}
		if out == nil {
			out = make(map[string]string)
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// FormatEnvText 将环境变量映射格式化为按键名升序排列的 KEY=VALUE 多行文本。
func FormatEnvText(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+env[k])
	}
	return strings.Join(lines, "\n")
}

// MergeEnv 将 overrides 合并到 base 环境变量列表中，并展开 %NAME% 占位符。
// 查找占位符时优先匹配 builtins（如 INSTALL_DIR、EXE_DIR），其次匹配 base；未匹配到的 %...% 保持原样。
func MergeEnv(base []string, overrides map[string]string, builtins map[string]string, caseInsensitive bool) []string {
	merged := make([]string, 0, len(base)+len(overrides))
	merged = append(merged, base...)
	if len(overrides) == 0 {
		return merged
	}

	lookup := func(name string) (string, bool) {
		for k, v := range builtins {
			if k == name || (caseInsensitive && strings.EqualFold(k, name)) {
				return v, true
			}
		}
		for _, entry := range merged {
			eq := strings.IndexByte(entry, '=')
			if eq <= 0 {
				continue
			}
			k := entry[:eq]
			if k == name || (caseInsensitive && strings.EqualFold(k, name)) {
				return entry[eq+1:], true
			}
		}
		return "", false
	}

	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		expanded := expandPercentVars(overrides[k], lookup)
		replaced := false
		for i, entry := range merged {
			eq := strings.IndexByte(entry, '=')
			if eq <= 0 {
				continue
			}
			ek := entry[:eq]
			if ek == k || (caseInsensitive && strings.EqualFold(ek, k)) {
				merged[i] = k + "=" + expanded
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, k+"="+expanded)
		}
	}
	return merged
}

func expandPercentVars(val string, lookup func(string) (string, bool)) string {
	if !strings.ContainsRune(val, '%') {
		return val
	}
	var sb strings.Builder
	sb.Grow(len(val))
	for i := 0; i < len(val); {
		if val[i] != '%' {
			sb.WriteByte(val[i])
			i++
			continue
		}
		end := strings.IndexByte(val[i+1:], '%')
		if end < 0 {
			sb.WriteString(val[i:])
			break
		}
		if end == 0 {
			sb.WriteString("%%")
			i += 2
			continue
		}
		name := val[i+1 : i+1+end]
		if resolved, ok := lookup(name); ok {
			sb.WriteString(resolved)
		} else {
			sb.WriteString(val[i : i+1+end+1])
		}
		i = i + 1 + end + 1
	}
	return sb.String()
}

// BuildExeCmdLine 构建原生 CreateProcess 使用的命令行（带引号的程序路径 + 可选参数）。
func BuildExeCmdLine(exePath, args string) string {
	clean := filepath.Clean(strings.TrimSpace(exePath))
	trimmedArgs := strings.TrimSpace(args)
	if trimmedArgs == "" {
		return fmt.Sprintf(`"%s"`, clean)
	}
	return fmt.Sprintf(`"%s" %s`, clean, trimmedArgs)
}

// BuildBatchCmdLineWithArgs 构建用于 cmd.exe /c 的批处理启动命令行，支持附加参数。
func BuildBatchCmdLineWithArgs(cmdExe, scriptPath, args string) string {
	clean := filepath.Clean(strings.TrimSpace(scriptPath))
	trimmedArgs := strings.TrimSpace(args)
	if trimmedArgs == "" {
		return fmt.Sprintf(`"%s" /c ""%s""`, cmdExe, clean)
	}
	return fmt.Sprintf(`"%s" /c ""%s" %s"`, cmdExe, clean, trimmedArgs)
}

// BuildPowershellCmdLine 构建用于 powershell.exe -File 的脚本启动命令行，支持附加参数。
func BuildPowershellCmdLine(psExe, scriptPath, args string) string {
	clean := filepath.Clean(strings.TrimSpace(scriptPath))
	trimmedArgs := strings.TrimSpace(args)
	if trimmedArgs == "" {
		return fmt.Sprintf(`"%s" -NoProfile -ExecutionPolicy Bypass -File "%s"`, psExe, clean)
	}
	return fmt.Sprintf(`"%s" -NoProfile -ExecutionPolicy Bypass -File "%s" %s`, psExe, clean, trimmedArgs)
}
