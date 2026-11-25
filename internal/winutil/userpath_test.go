package winutil

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestUserPathHelpers(t *testing.T) {
	raw := `%USERPROFILE%\.cargo\bin;C:\Tools\Node\;; "D:\Dev\ffmpeg\bin" ;`
	entries := SplitPathEntries(raw)
	wantSplit := []string{`%USERPROFILE%\.cargo\bin`, `C:\Tools\Node\`, ` "D:\Dev\ffmpeg\bin" `}
	if !reflect.DeepEqual(entries, wantSplit) {
		t.Fatalf("SplitPathEntries got %#v, want %#v", entries, wantSplit)
	}

	// AddDirsToPathList 应原样保留现有 %VAR% 项并去重追加新目录
	updated, added, changed := AddDirsToPathList(entries, []string{
		`C:\Tools\Node`,     // 已存在（尾部斜杠不同）
		`d:\dev\ffmpeg\bin`, // Windows 下已存在（大小写与引号不同）
		`E:\Apps\Git\cmd\`,  // 新增
	})
	if !changed {
		t.Fatal("expected changed=true")
	}
	if updated[0] != `%USERPROFILE%\.cargo\bin` {
		t.Fatalf("expected existing %%VAR%% entry preserved verbatim, got %q", updated[0])
	}
	wantNew := NormalizePathEntry(`E:\Apps\Git\cmd\`)
	if runtime.GOOS == "windows" {
		if len(added) != 1 || added[0] != wantNew {
			t.Fatalf("expected added=[%q], got %#v", wantNew, added)
		}
	}

	// RemoveDirsFromPathList 应移除匹配项并保留其余项原样
	afterRm, removed, rmChanged := RemoveDirsFromPathList(updated, []string{`C:\Tools\Node`, wantNew})
	if !rmChanged || len(removed) != 2 {
		t.Fatalf("expected 2 removed, got changed=%v removed=%#v", rmChanged, removed)
	}
	if afterRm[0] != `%USERPROFILE%\.cargo\bin` {
		t.Fatalf("expected remaining %%VAR%% entry preserved verbatim, got %#v", afterRm)
	}

	// ValidatePathEntry 拒绝空值和含分号路径
	if _, err := ValidatePathEntry("   "); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := ValidatePathEntry(`C:\a;C:\b`); err == nil {
		t.Fatal("expected error for semicolon in path")
	}

	// IsPathUnderOrEqual
	root := filepath.Join("C:", "Portable", "Git")
	sub := filepath.Join("C:", "Portable", "Git", "cmd")
	other := filepath.Join("C:", "Portable", "GitHub")
	if !IsPathUnderOrEqual(root, root) {
		t.Fatal("expected root to be under/equal itself")
	}
	if !IsPathUnderOrEqual(sub, root) {
		t.Fatal("expected sub to be under root")
	}
	if IsPathUnderOrEqual(other, root) {
		t.Fatal("did not expect sibling prefix GitHub to match Git")
	}
}

func TestLaunchEnvAndCmdLineHelpers(t *testing.T) {
	text := "# comment\n\nJAVA_HOME=%INSTALL_DIR%\r\nAPP_FLAGS=--a=1 --b=2\n  "
	parsed, err := ParseEnvText(text)
	if err != nil {
		t.Fatalf("ParseEnvText failed: %v", err)
	}
	wantMap := map[string]string{
		"JAVA_HOME": "%INSTALL_DIR%",
		"APP_FLAGS": "--a=1 --b=2",
	}
	if !reflect.DeepEqual(parsed, wantMap) {
		t.Fatalf("ParseEnvText = %#v, want %#v", parsed, wantMap)
	}
	if formatted := FormatEnvText(parsed); formatted != "APP_FLAGS=--a=1 --b=2\nJAVA_HOME=%INSTALL_DIR%" {
		t.Fatalf("FormatEnvText = %q", formatted)
	}

	if _, err := ParseEnvText("OK=1\nBAD_LINE_NO_EQ"); err == nil || !strings.Contains(err.Error(), "2") {
		t.Fatalf("expected line 2 error, got: %v", err)
	}
	if _, err := ParseEnvText("BAD KEY=1"); err == nil {
		t.Fatal("expected error for whitespace in env key")
	}
	if empty, err := ParseEnvText("\n# only comments\n"); err != nil || empty != nil {
		t.Fatalf("expected nil map for empty/comment-only env text, got %#v, %v", empty, err)
	}

	base := []string{
		`Path=C:\Windows\System32`,
		`EXISTING=old`,
	}
	merged := MergeEnv(base, map[string]string{
		"JAVA_HOME": `%INSTALL_DIR%`,
		"PATH":      `%EXE_DIR%;%PATH%;100%`,
		"existing":  `new`,
		"LITERAL":   `%UNMATCHED_VAR%`,
	}, map[string]string{
		"INSTALL_DIR": `C:\Apps\jdk-21`,
		"EXE_DIR":     `C:\Apps\jdk-21\bin`,
	}, true)

	got := make(map[string]string)
	pathCount := 0
	for _, entry := range merged {
		eq := strings.IndexByte(entry, '=')
		k, v := entry[:eq], entry[eq+1:]
		got[k] = v
		if strings.EqualFold(k, "PATH") {
			pathCount++
		}
	}
	if pathCount != 1 {
		t.Fatalf("expected case-insensitive PATH replacement without duplicate, count=%d, merged=%#v", pathCount, merged)
	}
	if got["PATH"] != `C:\Apps\jdk-21\bin;C:\Windows\System32;100%` {
		t.Fatalf("unexpected merged PATH: %q", got["PATH"])
	}
	if got["JAVA_HOME"] != `C:\Apps\jdk-21` {
		t.Fatalf("unexpected JAVA_HOME: %q", got["JAVA_HOME"])
	}
	if got["existing"] != "new" {
		t.Fatalf("unexpected existing override: %q", got["existing"])
	}
	if got["LITERAL"] != `%UNMATCHED_VAR%` {
		t.Fatalf("expected unmatched placeholder preserved, got %q", got["LITERAL"])
	}

	if gotCmd := BuildExeCmdLine(`C:\Apps\Node\node.exe`, `--version "a b"`); gotCmd != `"C:\Apps\Node\node.exe" --version "a b"` {
		t.Fatalf("BuildExeCmdLine = %q", gotCmd)
	}
	if gotBat := BuildBatchCmdLineWithArgs(`C:\Windows\System32\cmd.exe`, `C:\Apps\tool\run.bat`, `--foo bar`); gotBat != `"C:\Windows\System32\cmd.exe" /c ""C:\Apps\tool\run.bat" --foo bar"` {
		t.Fatalf("BuildBatchCmdLineWithArgs = %q", gotBat)
	}
}
