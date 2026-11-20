package settings

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"carrydock/internal/storage"
)

const (
	KeyInstallPath             = "install_path"
	KeyArchivePath             = "archive_path"
	KeyFontFamily              = "font_family"
	KeyMaxExecutableDepth      = "executable_search_max_depth"
	KeyExecutableExtensions    = "executable_extensions"
	KeyDeveloperOptionsEnabled = "developer_options_enabled"
	KeyRemoveNestedFolders     = "remove_nested_folders"
	KeyUseGridLayout           = "use_grid_layout"
	KeyEnableFileLogging       = "enable_file_logging"
	KeyLogFilePath             = "log_file_path"
	KeyLanguage                = "language"

	DefaultExecutableSearchMaxDepth = 3
	MinSearchDepth                  = 1
	MaxSearchDepth                  = 20
	DefaultRemoveNestedFolders      = true
	DefaultUseGridLayout            = false
	DefaultEnableFileLogging        = false
	DefaultLanguage                 = "zh_CN"
	DefaultFontFamily               = ""
)

// DefaultExecutableExtensions 为默认识别的可执行文件扩展名。
var DefaultExecutableExtensions = []string{"exe", "bat"}

// CommonExecutableExtensions 为扩展名弹窗中的常用候选列表。
var CommonExecutableExtensions = []string{
	"exe", "bat", "cmd", "com", "lnk", "msi", "ps1", "vbs",
}

// DefaultFontPriority 定义默认字体的优先使用顺序：PingFang → 鸿蒙 → 微软雅黑。
var DefaultFontPriority = []string{
	"PingFang SC",
	"PingFang",
	"HarmonyOS Sans SC",
	"HarmonyOS Sans",
	"Microsoft YaHei UI",
	"Microsoft YaHei",
}

// PresetFonts 为界面预设的推荐字体列表（空字符串代表自动默认优先级序列）。
var PresetFonts = []string{
	"", // 自动默认
	"PingFang SC",
	"HarmonyOS Sans SC",
	"Microsoft YaHei UI",
	"Segoe UI",
	"Maple Mono Normal NF CN",
	"SimSun",
	"Arial",
	"Verdana",
}

// excludedSystemFonts 为不适合作为界面文字的纯图标/符号字体（小写键）。
var excludedSystemFonts = map[string]struct{}{
	"marlett":              {},
	"symbol":               {},
	"webdings":             {},
	"wingdings":            {},
	"wingdings 2":          {},
	"wingdings 3":          {},
	"mt extra":             {},
	"hololens mdl2 assets": {},
	"segoe mdl2 assets":    {},
	"segoe fluent icons":   {},
	"segoe ui emoji":       {},
	"segoe ui symbol":      {},
	"segoe ui historic":    {},
}

// ResolveDefaultFont 根据已安装系统字体列表，按 PingFang → 鸿蒙 → 微软雅黑 顺序返回首个可用字体规范名。
func ResolveDefaultFont(installed []string) string {
	lookup := make(map[string]string, len(installed))
	for _, raw := range installed {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := lookup[key]; !exists {
			lookup[key] = name
		}
	}
	for _, cand := range DefaultFontPriority {
		if canonical, ok := lookup[strings.ToLower(cand)]; ok {
			return canonical
		}
	}
	return ""
}

// BuildFontStack 构建用于主题渲染的逗号分隔字体回退链。
// 当 configured 非空时将其置于首位，随后按默认优先顺序附加 PingFang → 鸿蒙 → 微软雅黑（去重）。
func BuildFontStack(configured string) string {
	configured = strings.TrimSpace(configured)
	seen := make(map[string]struct{}, len(DefaultFontPriority)+1)
	stack := make([]string, 0, len(DefaultFontPriority)+1)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		stack = append(stack, name)
	}
	if configured != "" {
		add(configured)
	}
	for _, f := range DefaultFontPriority {
		add(f)
	}
	return strings.Join(stack, ", ")
}

// FilterSystemFonts 过滤不可用于界面排版的 @ 竖排字体及纯符号/图标字体，去重并按名称不区分大小写排序。
func FilterSystemFonts(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		name := strings.TrimSpace(item)
		if name == "" || strings.HasPrefix(name, "@") {
			continue
		}
		key := strings.ToLower(name)
		if _, excluded := excludedSystemFonts[key]; excluded {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	slices.SortFunc(out, func(a, b string) int {
		return cmp.Or(strings.Compare(strings.ToLower(a), strings.ToLower(b)), strings.Compare(a, b))
	})
	return out
}

// Service 封装应用各项设置的读取与持久化逻辑，与原版 app_data.json 字段保持一致。
type Service struct {
	store *storage.JSONStorage
}

// NewService 使用指定存储实例创建配置服务；若传入 nil 则使用全局默认存储。
func NewService(store *storage.JSONStorage) *Service {
	if store == nil {
		store = storage.Default()
	}
	return &Service{store: store}
}

// NormalizeExtensions 清洗、去重并排序扩展名列表（不含前导点号）。
func NormalizeExtensions(exts []string) []string {
	seen := make(map[string]struct{}, len(exts))
	out := make([]string, 0, len(exts))
	for _, raw := range exts {
		s := strings.ToLower(strings.TrimSpace(raw))
		s = strings.TrimPrefix(s, ".")
		s = strings.TrimSpace(s)
		if s == "" || strings.ContainsAny(s, `\/:*?"<>| `) {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

// NormalizeDepth 将搜索深度限制在合法范围内。
func NormalizeDepth(depth int) int {
	if depth < MinSearchDepth {
		return MinSearchDepth
	}
	if depth > MaxSearchDepth {
		return MaxSearchDepth
	}
	return depth
}

// ConfigFilePath 返回配置文件绝对路径。
func (s *Service) ConfigFilePath() string {
	return s.store.FilePath()
}

func (s *Service) GetInstallPath() string {
	v, _ := s.store.GetString(KeyInstallPath)
	return strings.TrimSpace(v)
}

func (s *Service) SaveInstallPath(path string) error {
	return s.store.Set(KeyInstallPath, strings.TrimSpace(path))
}

func (s *Service) GetArchivePath() string {
	v, _ := s.store.GetString(KeyArchivePath)
	return strings.TrimSpace(v)
}

func (s *Service) SaveArchivePath(path string) error {
	return s.store.Set(KeyArchivePath, strings.TrimSpace(path))
}

func (s *Service) GetFontFamily() string {
	v, ok := s.store.GetString(KeyFontFamily)
	if !ok {
		return DefaultFontFamily
	}
	return strings.TrimSpace(v)
}

func (s *Service) SaveFontFamily(font string) error {
	return s.store.Set(KeyFontFamily, strings.TrimSpace(font))
}

func (s *Service) GetExecutableSearchMaxDepth() int {
	v, ok := s.store.GetInt(KeyMaxExecutableDepth)
	if !ok || v < MinSearchDepth {
		return DefaultExecutableSearchMaxDepth
	}
	return NormalizeDepth(v)
}

func (s *Service) SaveExecutableSearchMaxDepth(depth int) error {
	return s.store.Set(KeyMaxExecutableDepth, NormalizeDepth(depth))
}

func (s *Service) GetExecutableExtensions() []string {
	raw, ok := s.store.GetStringSlice(KeyExecutableExtensions)
	if !ok {
		return slices.Clone(DefaultExecutableExtensions)
	}
	norm := NormalizeExtensions(raw)
	if len(norm) == 0 {
		return slices.Clone(DefaultExecutableExtensions)
	}
	return norm
}

func (s *Service) SaveExecutableExtensions(exts []string) error {
	norm := NormalizeExtensions(exts)
	if len(norm) == 0 {
		norm = slices.Clone(DefaultExecutableExtensions)
	}
	return s.store.Set(KeyExecutableExtensions, norm)
}

func (s *Service) GetDeveloperOptionsEnabled() bool {
	v, ok := s.store.GetBool(KeyDeveloperOptionsEnabled)
	if !ok {
		return false
	}
	return v
}

func (s *Service) SaveDeveloperOptionsEnabled(enabled bool) error {
	return s.store.Set(KeyDeveloperOptionsEnabled, enabled)
}

func (s *Service) GetRemoveNestedFoldersEnabled() bool {
	v, ok := s.store.GetBool(KeyRemoveNestedFolders)
	if !ok {
		return DefaultRemoveNestedFolders
	}
	return v
}

func (s *Service) SaveRemoveNestedFoldersEnabled(enabled bool) error {
	return s.store.Set(KeyRemoveNestedFolders, enabled)
}

func (s *Service) GetUseGridLayout() bool {
	v, ok := s.store.GetBool(KeyUseGridLayout)
	if !ok {
		return DefaultUseGridLayout
	}
	return v
}

func (s *Service) SaveUseGridLayout(useGrid bool) error {
	return s.store.Set(KeyUseGridLayout, useGrid)
}

func (s *Service) GetEnableFileLogging() bool {
	v, ok := s.store.GetBool(KeyEnableFileLogging)
	if !ok {
		return DefaultEnableFileLogging
	}
	return v
}

func (s *Service) SaveEnableFileLogging(enabled bool) error {
	return s.store.Set(KeyEnableFileLogging, enabled)
}

func (s *Service) GetLogFilePath() string {
	v, ok := s.store.GetString(KeyLogFilePath)
	if ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return filepath.Join(filepath.Dir(s.store.FilePath()), "carrydock.log")
}

func (s *Service) SaveLogFilePath(path string) error {
	return s.store.Set(KeyLogFilePath, strings.TrimSpace(path))
}

func (s *Service) GetLanguage() string {
	v, ok := s.store.GetString(KeyLanguage)
	if !ok || strings.TrimSpace(v) == "" {
		return DefaultLanguage
	}
	return strings.TrimSpace(v)
}

func (s *Service) SaveLanguage(lang string) error {
	return s.store.Set(KeyLanguage, strings.TrimSpace(lang))
}
