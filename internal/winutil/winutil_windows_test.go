//go:build windows

package winutil

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetFileDescriptionAndExtractIconOnSystemBinaries(t *testing.T) {
	winDir := os.Getenv("SystemRoot")
	if winDir == "" {
		winDir = `C:\Windows`
	}

	candidates := []string{
		filepath.Join(winDir, "System32", "notepad.exe"),
		filepath.Join(winDir, "System32", "cmd.exe"),
		filepath.Join(winDir, "regedit.exe"),
		filepath.Join(winDir, "explorer.exe"),
	}

	checked := 0
	for _, exe := range candidates {
		if _, err := os.Stat(exe); err != nil {
			continue
		}
		checked++

		desc := GetFileDescription(exe)
		if desc == "" {
			t.Errorf("expected non-empty FileDescription for %s", exe)
		}
		meta := GetFileMetadata(exe)
		ver := GetFileVersion(exe)
		if meta.Description != desc {
			t.Errorf("GetFileMetadata(%s).Description = %q, want %q", exe, meta.Description, desc)
		}
		if meta.Version == "" || ver == "" || meta.Version != ver {
			t.Errorf("expected non-empty consistent version for %s, meta=%q ver=%q", exe, meta.Version, ver)
		}

		pngBytes, err := ExtractIconPNG(exe, 64)
		if err != nil {
			t.Fatalf("ExtractIconPNG(%s, 64) failed: %v", exe, err)
		}
		img, err := png.Decode(bytes.NewReader(pngBytes))
		if err != nil {
			t.Fatalf("decoded icon for %s is not valid PNG: %v", exe, err)
		}
		b := img.Bounds()
		if b.Dx() == 0 || b.Dy() == 0 {
			t.Fatalf("invalid icon dimensions for %s: %v", exe, b)
		}

		// 确认图标中存在不透明可见像素，避免掩码/Alpha 处理异常导致图标全透明。
		visiblePixels := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a > 0 {
					visiblePixels++
				}
			}
		}
		if visiblePixels == 0 {
			t.Fatalf("icon for %s has 0 visible pixels (fully transparent)", exe)
		}
		t.Logf("%s -> desc=%q ver=%q icon=%dx%d visible=%d/%d (%d bytes)",
			filepath.Base(exe), desc, ver, b.Dx(), b.Dy(), visiblePixels, b.Dx()*b.Dy(), len(pngBytes))
	}
	if checked == 0 {
		t.Skip("no Windows system binaries found")
	}
}

func TestNormalizeVersionStringAndFormatFixedVersion(t *testing.T) {
	normCases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   \t\x00 ", ""},
		{"N/A", ""},
		{"none", ""},
		{"Unknown", ""},
		{"null", ""},
		{"no version", ""},
		{"0", ""},
		{"0.0", ""},
		{"0.0.0.0", ""},
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
		{"V24.08", "24.08"},
		{"1, 2, 3, 4", "1.2.3.4"},
		{"1, 0, 0, 1", "1.0.0.1"},
		{"2 . 5 . 0", "2.5.0"},
		{"1.2.3-beta.1", "1.2.3-beta.1"},
		{"10.0.22621.1 (WinBuild.160101.0800)", "10.0.22621.1 (WinBuild.160101.0800)"},
	}
	for _, tc := range normCases {
		if got := normalizeVersionString(tc.in); got != tc.want {
			t.Errorf("normalizeVersionString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	fixedCases := []struct {
		ms, ls uint32
		want   string
	}{
		{0, 0, ""},
		{(1 << 16) | 0, 0, "1.0"},
		{(24 << 16) | 8, 0, "24.8"},
		{(1 << 16) | 2, (3 << 16) | 0, "1.2.3"},
		{(1 << 16) | 2, (3 << 16) | 4, "1.2.3.4"},
		{(1 << 16) | 0, (0 << 16) | 5, "1.0.0.5"},
	}
	for _, tc := range fixedCases {
		if got := formatFixedVersion(tc.ms, tc.ls); got != tc.want {
			t.Errorf("formatFixedVersion(%#x, %#x) = %q, want %q", tc.ms, tc.ls, got, tc.want)
		}
	}
}

func TestResolveDirectoryTarget(t *testing.T) {
	tmp := t.TempDir()

	if _, err := resolveDirectoryTarget(""); err == nil {
		t.Fatal("expected error for empty directory path")
	}
	if _, err := resolveDirectoryTarget("   \t  "); err == nil {
		t.Fatal("expected error for whitespace directory path")
	}

	missingDir := filepath.Join(tmp, "missing_dir")
	if _, err := resolveDirectoryTarget(missingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist for missing dir, got: %v", err)
	}

	filePath := filepath.Join(tmp, "not_a_dir.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveDirectoryTarget(filePath); err == nil {
		t.Fatal("expected error when directory target is a file")
	}

	specialDir := filepath.Join(tmp, "My App, v1.0 (中文)")
	if err := os.MkdirAll(specialDir, 0o755); err != nil {
		t.Fatal(err)
	}
	slashed := filepath.ToSlash(specialDir) + "/."
	got, err := resolveDirectoryTarget(slashed)
	if err != nil {
		t.Fatalf("resolveDirectoryTarget(%q) failed: %v", slashed, err)
	}
	if !SamePath(got, specialDir) {
		t.Fatalf("resolved directory mismatch: got %q, want %q", got, specialDir)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}

	if err := OpenDirectory(""); err == nil {
		t.Fatal("expected OpenDirectory(\"\") to return error")
	}
	if err := OpenDirectory(missingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected OpenDirectory on missing dir to wrap os.ErrNotExist, got: %v", err)
	}
}

func TestResolveRevealTarget(t *testing.T) {
	tmp := t.TempDir()

	if _, _, err := resolveRevealTarget(""); err == nil {
		t.Fatal("expected error for empty file path")
	}

	specialDir := filepath.Join(tmp, "软件 目录, 测试")
	if err := os.MkdirAll(specialDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existingFile := filepath.Join(specialDir, "app, config (1).json")
	if err := os.WriteFile(existingFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. 已存在文件（含正斜杠与特殊字符）：应返回文件本身且 selectItem=true
	target, selectItem, err := resolveRevealTarget(filepath.ToSlash(existingFile))
	if err != nil {
		t.Fatalf("resolveRevealTarget(existingFile) failed: %v", err)
	}
	if !selectItem || !SamePath(target, existingFile) {
		t.Fatalf("got (%q, %v), want (%q, true)", target, selectItem, existingFile)
	}

	// 2. 已存在目录：应直接打开该目录自身且 selectItem=false
	target, selectItem, err = resolveRevealTarget(specialDir)
	if err != nil {
		t.Fatalf("resolveRevealTarget(specialDir) failed: %v", err)
	}
	if selectItem || !SamePath(target, specialDir) {
		t.Fatalf("got (%q, %v), want (%q, false)", target, selectItem, specialDir)
	}

	// 3. 文件不存在但父目录存在：应回退到父目录且 selectItem=false
	missingFileWithParent := filepath.Join(specialDir, "missing.log")
	target, selectItem, err = resolveRevealTarget(missingFileWithParent)
	if err != nil {
		t.Fatalf("resolveRevealTarget(missingFileWithParent) failed: %v", err)
	}
	if selectItem || !SamePath(target, specialDir) {
		t.Fatalf("got (%q, %v), want (%q, false)", target, selectItem, specialDir)
	}

	// 4. 文件与父目录均不存在：应返回包装 os.ErrNotExist 的错误
	bothMissing := filepath.Join(tmp, "no_such_dir", "no_such_file.zip")
	if _, _, err := resolveRevealTarget(bothMissing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist when file and parent both missing, got: %v", err)
	}

	if err := RevealInExplorer(""); err == nil {
		t.Fatal("expected RevealInExplorer(\"\") to return error")
	}
	if err := RevealInExplorer(bothMissing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected RevealInExplorer on missing tree to wrap os.ErrNotExist, got: %v", err)
	}
}

func TestExplorerFormattingHelpers(t *testing.T) {
	if got := ensureTrailingBackslash(`C:\Foo\Bar`); got != `C:\Foo\Bar\` {
		t.Fatalf("ensureTrailingBackslash mismatch: got %q", got)
	}
	if got := ensureTrailingBackslash(`C:\Foo\Bar\`); got != `C:\Foo\Bar\` {
		t.Fatalf("ensureTrailingBackslash should not duplicate backslash: got %q", got)
	}
	if got := ensureTrailingBackslash(""); got != "" {
		t.Fatalf("ensureTrailingBackslash(\"\") mismatch: got %q", got)
	}

	samplePath := `C:\My Apps, Portable\test app.exe`
	wantParams := `/select,"C:\My Apps, Portable\test app.exe"`
	if got := explorerSelectParams(samplePath); got != wantParams {
		t.Fatalf("explorerSelectParams mismatch: got %q, want %q", got, wantParams)
	}

	exp := explorerExePath()
	if !strings.EqualFold(filepath.Base(exp), "explorer.exe") {
		t.Fatalf("expected explorer.exe basename, got %q", exp)
	}
	if st, err := os.Stat(exp); err != nil || st.IsDir() {
		t.Fatalf("expected explorerExePath() %q to exist on Windows: %v", exp, err)
	}
}

func TestBuildBatchCmdLine(t *testing.T) {
	cmdExe := `C:\Windows\System32\cmd.exe`
	script := `C:\Users\测试 用户\AppData\Local\Temp\carrydock_update_1\update.bat`
	got := buildBatchCmdLine(cmdExe, script)
	want := `"C:\Windows\System32\cmd.exe" /c ""C:\Users\测试 用户\AppData\Local\Temp\carrydock_update_1\update.bat""`
	if got != want {
		t.Fatalf("buildBatchCmdLine mismatch:\n got: %s\nwant: %s", got, want)
	}

	if err := LaunchBatchScript(""); err == nil {
		t.Fatal("expected LaunchBatchScript(\"\") to return error")
	}
}
