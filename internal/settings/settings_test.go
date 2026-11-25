package settings

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"carrydock/internal/storage"
)

func TestGetFontFamilyDefaultAndCustom(t *testing.T) {
	store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	svc := NewService(store)

	if got := svc.GetFontFamily(); got != "" {
		t.Fatalf("expected empty default font family, got %q", got)
	}

	if err := svc.SaveFontFamily("HarmonyOS Sans SC"); err != nil {
		t.Fatalf("SaveFontFamily failed: %v", err)
	}
	if got := svc.GetFontFamily(); got != "HarmonyOS Sans SC" {
		t.Fatalf("expected saved font family, got %q", got)
	}
}

func TestResolveDefaultFontPriority(t *testing.T) {
	cases := []struct {
		name      string
		installed []string
		want      string
	}{
		{
			name:      "pingfang_first",
			installed: []string{"Microsoft YaHei UI", "HarmonyOS Sans SC", "PingFang SC"},
			want:      "PingFang SC",
		},
		{
			name:      "pingfang_generic_second",
			installed: []string{"Microsoft YaHei UI", "HarmonyOS Sans SC", "pingfang"},
			want:      "pingfang",
		},
		{
			name:      "harmony_before_yahei",
			installed: []string{"Microsoft YaHei UI", "harmonyos sans sc"},
			want:      "harmonyos sans sc",
		},
		{
			name:      "harmony_generic_before_yahei",
			installed: []string{"Microsoft YaHei", "HarmonyOS Sans"},
			want:      "HarmonyOS Sans",
		},
		{
			name:      "yahei_ui_fallback",
			installed: []string{"Segoe UI", "Microsoft YaHei", "Microsoft YaHei UI"},
			want:      "Microsoft YaHei UI",
		},
		{
			name:      "none_matched",
			installed: []string{"Arial", "Segoe UI"},
			want:      "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveDefaultFont(tc.installed); got != tc.want {
				t.Fatalf("ResolveDefaultFont(%v) = %q, want %q", tc.installed, got, tc.want)
			}
		})
	}
}

func TestBuildFontStack(t *testing.T) {
	wantDefault := "PingFang SC, PingFang, HarmonyOS Sans SC, HarmonyOS Sans, Microsoft YaHei UI, Microsoft YaHei"
	if got := BuildFontStack(""); got != wantDefault {
		t.Fatalf("BuildFontStack(\"\") = %q, want %q", got, wantDefault)
	}

	wantArial := "Arial, " + wantDefault
	if got := BuildFontStack("Arial"); got != wantArial {
		t.Fatalf("BuildFontStack(\"Arial\") = %q, want %q", got, wantArial)
	}

	// 已在优先级中的字体不会重复出现，且被提升到最前面
	wantHarmony := "HarmonyOS Sans SC, PingFang SC, PingFang, HarmonyOS Sans, Microsoft YaHei UI, Microsoft YaHei"
	if got := BuildFontStack("HarmonyOS Sans SC"); got != wantHarmony {
		t.Fatalf("BuildFontStack(\"HarmonyOS Sans SC\") = %q, want %q", got, wantHarmony)
	}
}

func TestFilterSystemFonts(t *testing.T) {
	raw := []string{
		"",
		"  ",
		"@Microsoft YaHei",
		"@SimSun",
		"Wingdings",
		"wingdings 2",
		"Segoe MDL2 Assets",
		"Segoe Fluent Icons",
		"Segoe UI Emoji",
		"Segoe UI Symbol",
		"Segoe UI Historic",
		"Webdings",
		"Symbol",
		"Marlett",
		"MT Extra",
		"HoloLens MDL2 Assets",
		"JetBrains Mono",
		"jetbrains mono",
		"Arial",
		"PingFang SC",
	}

	got := FilterSystemFonts(raw)
	want := []string{"Arial", "JetBrains Mono", "PingFang SC"}
	if !slices.Equal(got, want) {
		t.Fatalf("FilterSystemFonts = %v, want %v", got, want)
	}
}

func TestThemeModeAndThemeColorSettings(t *testing.T) {
	store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	svc := NewService(store)

	// 默认值校验
	if got := svc.GetThemeMode(); got != DefaultThemeMode {
		t.Fatalf("expected default theme mode %q, got %q", DefaultThemeMode, got)
	}
	if got := svc.GetThemeColor(); got != DefaultThemeColor {
		t.Fatalf("expected default theme color %q, got %q", DefaultThemeColor, got)
	}

	// 规范化与预设校验
	modeCases := map[string]string{
		"":        ThemeModeSystem,
		"system":  ThemeModeSystem,
		" LIGHT ": ThemeModeLight,
		"Dark":    ThemeModeDark,
		"invalid": ThemeModeSystem,
	}
	for in, want := range modeCases {
		if got := NormalizeThemeMode(in); got != want {
			t.Fatalf("NormalizeThemeMode(%q) = %q, want %q", in, got, want)
		}
	}

	colorCases := map[string]string{
		"":         "",
		"  ":       "",
		"system":   "",
		"SYSTEM":   "",
		"#2563EB":  "#2563eb",
		"7C3AED":   "#7c3aed",
		" #0d9488": "#0d9488",
		"#123":     "",
		"#gggggg":  "",
		"red":      "",
	}
	for in, want := range colorCases {
		if got := NormalizeThemeColor(in); got != want {
			t.Fatalf("NormalizeThemeColor(%q) = %q, want %q", in, got, want)
		}
	}

	if !IsPresetThemeColor("#7C3AED") {
		t.Fatalf("expected #7C3AED to be recognized as preset theme color")
	}
	if IsPresetThemeColor("#123456") || IsPresetThemeColor("") {
		t.Fatalf("did not expect #123456 or empty string to be preset theme color")
	}

	// 持久化读写校验
	if err := svc.SaveThemeMode("Dark"); err != nil {
		t.Fatalf("SaveThemeMode failed: %v", err)
	}
	if got := svc.GetThemeMode(); got != ThemeModeDark {
		t.Fatalf("expected saved theme mode %q, got %q", ThemeModeDark, got)
	}

	if err := svc.SaveThemeColor("7C3AED"); err != nil {
		t.Fatalf("SaveThemeColor failed: %v", err)
	}
	if got := svc.GetThemeColor(); got != "#7c3aed" {
		t.Fatalf("expected saved theme color #7c3aed, got %q", got)
	}

	if err := svc.SaveThemeColor("invalid"); err != nil {
		t.Fatalf("SaveThemeColor(invalid) failed: %v", err)
	}
	if got := svc.GetThemeColor(); got != "" {
		t.Fatalf("expected invalid color to reset to empty, got %q", got)
	}
}

func TestLogLevelSettings(t *testing.T) {
	store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	svc := NewService(store)

	if got := svc.GetLogLevel(); got != DefaultLogLevel {
		t.Fatalf("expected default log level %q, got %q", DefaultLogLevel, got)
	}

	cases := map[string]string{
		"":         LogLevelInfo,
		"debug":    LogLevelDebug,
		" DEBUG ":  LogLevelDebug,
		"INFO":     LogLevelInfo,
		"warn":     LogLevelWarn,
		"WARNING":  LogLevelWarn,
		"Error":    LogLevelError,
		"nonsense": LogLevelInfo,
	}
	for in, want := range cases {
		if got := NormalizeLogLevel(in); got != want {
			t.Fatalf("NormalizeLogLevel(%q) = %q, want %q", in, got, want)
		}
	}

	for _, lvl := range []string{"DEBUG", "warn", "error", "info"} {
		if err := svc.SaveLogLevel(lvl); err != nil {
			t.Fatalf("SaveLogLevel(%q) failed: %v", lvl, err)
		}
		if got := svc.GetLogLevel(); got != NormalizeLogLevel(lvl) {
			t.Fatalf("GetLogLevel() = %q, want %q", got, NormalizeLogLevel(lvl))
		}
	}
}

func TestCheckUpdateOnStartupSetting(t *testing.T) {
	store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	svc := NewService(store)

	if !svc.GetCheckUpdateOnStartup() {
		t.Fatalf("expected default CheckUpdateOnStartup to be true")
	}

	if err := svc.SaveCheckUpdateOnStartup(false); err != nil {
		t.Fatalf("SaveCheckUpdateOnStartup(false) failed: %v", err)
	}
	if svc.GetCheckUpdateOnStartup() {
		t.Fatalf("expected CheckUpdateOnStartup to be false after saving false")
	}

	if err := svc.SaveCheckUpdateOnStartup(true); err != nil {
		t.Fatalf("SaveCheckUpdateOnStartup(true) failed: %v", err)
	}
	if !svc.GetCheckUpdateOnStartup() {
		t.Fatalf("expected CheckUpdateOnStartup to be true after saving true")
	}
}

func TestPortablePathHelpers(t *testing.T) {
	base := filepath.Join(t.TempDir(), "CarryDock")
	sub := filepath.Join(base, "Apps", "7-Zip")
	parent := filepath.Join(filepath.Dir(base), "SharedApps")

	if IsAbsOrHasVolume("") || IsAbsOrHasVolume(`.\Apps`) || IsAbsOrHasVolume(`..\Shared`) || IsAbsOrHasVolume(".") {
		t.Fatalf("expected relative paths not to be treated as absolute")
	}
	if !IsAbsOrHasVolume(base) {
		t.Fatalf("expected %q to be treated as absolute", base)
	}
	if runtime.GOOS == "windows" {
		if !IsAbsOrHasVolume(`D:Apps`) || !IsAbsOrHasVolume(`\RootDir`) || !IsAbsOrHasVolume(`\\server\share\dir`) {
			t.Fatalf("expected drive-relative, root-slash, and UNC paths to be treated as non-relative on Windows")
		}
	}

	if got := ResolvePath(base, `.\Apps/7-Zip`); got != sub {
		t.Fatalf("ResolvePath(%q, .\\Apps/7-Zip) = %q, want %q", base, got, sub)
	}
	if got := ResolvePath(base, "."); got != base {
		t.Fatalf("ResolvePath(%q, .) = %q, want %q", base, got, base)
	}
	if got := ResolvePath(base, sub); got != sub {
		t.Fatalf("ResolvePath absolute = %q, want %q", got, sub)
	}

	if rel, ok := ToRelativePath(base, sub); !ok || rel != "."+string(filepath.Separator)+filepath.Join("Apps", "7-Zip") {
		t.Fatalf("ToRelativePath(sub) = (%q, %v)", rel, ok)
	}
	if rel, ok := ToRelativePath(base, base); !ok || rel != "." {
		t.Fatalf("ToRelativePath(same) = (%q, %v), want (\".\", true)", rel, ok)
	}
	if rel, ok := ToRelativePath(base, parent); !ok || rel != ".."+string(filepath.Separator)+"SharedApps" {
		t.Fatalf("ToRelativePath(parent) = (%q, %v)", rel, ok)
	}

	// Windows 盘符大小写差异与跨盘符回退校验
	if runtime.GOOS == "windows" {
		vol := filepath.VolumeName(base)
		if len(vol) == 2 && vol[1] == ':' {
			altVol := strings.ToLower(vol)
			if altVol == vol {
				altVol = strings.ToUpper(vol)
			}
			altSub := altVol + sub[len(vol):]
			if rel, ok := ToRelativePath(base, altSub); !ok || rel != `.\Apps\7-Zip` {
				t.Fatalf("ToRelativePath with diff drive case = (%q, %v), want (.\\Apps\\7-Zip, true)", rel, ok)
			}

			otherDrive := "Z:"
			if strings.EqualFold(vol, "Z:") {
				otherDrive = "Y:"
			}
			cross := otherDrive + `\External\Apps`
			if CanRelativize(base, cross) {
				t.Fatalf("expected CanRelativize(%q, %q) = false", base, cross)
			}
			if stored, isRel := FormatPathForMode(base, cross, true); isRel || stored != cross {
				t.Fatalf("expected cross-drive fallback (%q, false), got (%q, %v)", cross, stored, isRel)
			}
		}
	}
}

func TestApplyRelativePathsModeAndAlwaysResolveOnRead(t *testing.T) {
	base := filepath.Join(t.TempDir(), "CarryDock")
	store := storage.NewAtBaseDir(base)
	svc := NewService(store)

	absInstall := filepath.Join(base, "Portable")
	absArchive := filepath.Join(base, "Archives")
	absLog := filepath.Join(base, "logs", "custom.log")
	if err := svc.SaveInstallPath(absInstall); err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveArchivePath(absArchive); err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveLogFilePath(absLog); err != nil {
		t.Fatal(err)
	}

	// 开启相对路径模式，自动转换已设置路径
	mig, err := svc.ApplyRelativePathsMode(true)
	if err != nil {
		t.Fatalf("ApplyRelativePathsMode(true) failed: %v", err)
	}
	if mig.Converted != 3 || len(mig.Unconvertible) != 0 {
		t.Fatalf("unexpected migration result: %+v", mig)
	}
	if got := svc.GetRawInstallPath(); got != "."+string(filepath.Separator)+"Portable" {
		t.Fatalf("GetRawInstallPath() = %q", got)
	}
	if got := svc.GetInstallPath(); got != absInstall {
		t.Fatalf("GetInstallPath() = %q, want %q", got, absInstall)
	}
	if got := svc.GetRawArchivePath(); got != "."+string(filepath.Separator)+"Archives" {
		t.Fatalf("GetRawArchivePath() = %q", got)
	}
	if got := svc.GetArchivePath(); got != absArchive {
		t.Fatalf("GetArchivePath() = %q, want %q", got, absArchive)
	}
	if got := svc.GetRawLogFilePath(); got != "."+string(filepath.Separator)+filepath.Join("logs", "custom.log") {
		t.Fatalf("GetRawLogFilePath() = %q", got)
	}
	if got := svc.GetLogFilePath(); got != absLog {
		t.Fatalf("GetLogFilePath() = %q, want %q", got, absLog)
	}

	// 即使关闭开关，已有相对路径在读取时依然会被正确解析为绝对路径
	if err := svc.SaveUseRelativePaths(false); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetInstallPath(); got != absInstall {
		t.Fatalf("expected GetInstallPath to resolve relative path even when toggle is off, got %q", got)
	}

	// 完整关闭并还原为绝对路径，同时确认未设置的选填路径不会凭空写入
	store2 := storage.NewAtBaseDir(filepath.Join(t.TempDir(), "CarryDock2"))
	svc2 := NewService(store2)
	_ = svc2.SaveInstallPath(filepath.Join(store2.BaseDir(), "Apps"))
	if _, err := svc2.ApplyRelativePathsMode(true); err != nil {
		t.Fatal(err)
	}
	if svc2.GetRawArchivePath() != "" || svc2.GetRawLogFilePath() != "" {
		t.Fatalf("expected unset archive/log paths to stay empty")
	}

	migOff, err := svc.ApplyRelativePathsMode(false)
	if err != nil {
		t.Fatalf("ApplyRelativePathsMode(false) failed: %v", err)
	}
	if migOff.Converted != 3 {
		t.Fatalf("expected 3 restored paths, got %+v", migOff)
	}
	if got := svc.GetRawInstallPath(); got != absInstall {
		t.Fatalf("expected restored absolute install path %q, got %q", absInstall, got)
	}
}
