package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		local, remote string
		want          bool
	}{
		{"0.8.4+104", "v0.8.5+105", true},
		{"0.9.0+105", "v0.8.4+104", false},
		{"0.9.0+105", "v0.9.0+105", false},
		{"0.8.4", "v0.9.0", true},
		{"0.9.0", "v0.8.9", false},
		{"1.0.0", "v1.0.1", true},
	}
	for _, tc := range cases {
		if got := IsNewerVersion(tc.local, tc.remote); got != tc.want {
			t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", tc.local, tc.remote, got, tc.want)
		}
	}
}

func TestCleanupLegacyFiles(t *testing.T) {
	appDir := t.TempDir()

	// 模拟旧版 Flutter v0.8.4 完整目录结构与新版本文件、用户数据共存
	mustWrite := func(rel string, content []byte) {
		t.Helper()
		full := filepath.Join(appDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustExist := func(rel string) {
		t.Helper()
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s to remain, got err: %v", rel, err)
		}
	}
	mustNotExist := func(rel string) {
		t.Helper()
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be deleted, stat err=%v", rel, err)
		}
	}

	for _, name := range legacyRootFiles {
		mustWrite(name, []byte("legacy"))
	}
	for _, name := range legacyDataFiles {
		mustWrite("data/"+name, []byte("legacy-data"))
	}
	mustWrite("data/flutter_assets/AssetManifest.bin", []byte("manifest"))
	mustWrite("data/flutter_assets/fonts/MaterialIcons-Regular.otf", []byte("font"))
	mustWrite("update_script.bat", []byte("@echo off\r\necho CarryDock Auto Update Script\r\n"))
	mustWrite("config_backup.bin", []byte(`{"install_path":"D:\\Portable"}`))

	// 应保留的文件与目录
	mustWrite("CarryDock.exe", []byte("mz"))
	mustWrite("data/app_data.json", []byte(`{"install_path":"D:\\Portable"}`))
	mustWrite("data/app_data.lock", []byte(""))
	mustWrite("data/carrydock.log", []byte("log line"))
	mustWrite("~archives/demo.zip", []byte("zip"))
	mustWrite("~archives/software_list.json", []byte("[]"))
	mustWrite("custom_user_file.txt", []byte("keep"))
	mustWrite("custom_dir/keep.txt", []byte("keep"))

	removed := CleanupLegacyFiles(appDir)
	if len(removed) == 0 {
		t.Fatal("expected legacy files to be removed")
	}

	for _, name := range legacyRootFiles {
		mustNotExist(name)
	}
	for _, name := range legacyDataFiles {
		mustNotExist("data/" + name)
	}
	for _, name := range legacyDataDirs {
		mustNotExist("data/" + name)
	}
	mustNotExist("update_script.bat")
	mustNotExist("config_backup.bin")

	mustExist("CarryDock.exe")
	mustExist("data/app_data.json")
	mustExist("data/app_data.lock")
	mustExist("data/carrydock.log")
	mustExist("~archives/demo.zip")
	mustExist("~archives/software_list.json")
	mustExist("custom_user_file.txt")
	mustExist("custom_dir/keep.txt")

	// 无特征的非 CarryDock 脚本和二进制文件不得被误删
	cleanDir := t.TempDir()
	writeClean := func(rel string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(cleanDir, rel), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeClean("update_script.bat", []byte("@echo off\necho custom user script\n"))
	writeClean("config_backup.bin", []byte{0x01, 0x02, 0x03, 0x04})

	if got := CleanupLegacyFiles(cleanDir); len(got) != 0 {
		t.Fatalf("expected non-CarryDock files to be preserved, removed=%v", got)
	}
	if _, err := os.Stat(filepath.Join(cleanDir, "update_script.bat")); err != nil {
		t.Fatalf("user update_script.bat should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cleanDir, "config_backup.bin")); err != nil {
		t.Fatalf("user config_backup.bin should remain: %v", err)
	}

	// 单独存在含 CarryDock 配置键的 config_backup.bin 应被清理
	writeClean("config_backup.bin", []byte(`{"install_path":"D:\\Apps"}`))
	got := CleanupLegacyFiles(cleanDir)
	if len(got) != 1 || got[0] != "config_backup.bin" {
		t.Fatalf("expected CarryDock config_backup.bin to be cleaned, got %v", got)
	}
}

func TestBuildUpdateScript(t *testing.T) {
	script := buildUpdateScript(updateScriptParams{
		AppDir:     `D:\测试 100% 目录\CarryDock\`,
		SourceDir:  `C:\Users\中文 用户\AppData\Local\Temp\carrydock_update_1\pkg\CarryDock\`,
		OldExeName: "carrydock.exe",
		NewExeName: "CarryDock.exe",
		TempDir:    `C:\Users\中文 用户\AppData\Local\Temp\carrydock_update_1\`,
		ZipPath:    `C:\Users\中文 用户\AppData\Local\Temp\carrydock-v0.9.1+106.zip`,
		OldPID:     4242,
	})

	// 必须全部使用 CRLF
	withoutCRLF := strings.ReplaceAll(script, "\r\n", "")
	if strings.Contains(withoutCRLF, "\n") {
		t.Fatal("expected all line endings in update script to be CRLF")
	}

	required := []string{
		"@echo off",
		"chcp 65001 >nul",
		`set "appDir=D:\测试 100%% 目录\CarryDock"`,
		`set "srcDir=C:\Users\中文 用户\AppData\Local\Temp\carrydock_update_1\pkg\CarryDock"`,
		`set "oldExeName=carrydock.exe"`,
		`set "exeName=CarryDock.exe"`,
		`set "oldPid=4242"`,
		`tasklist /FI "PID eq %oldPid%" 2>nul | find "%oldPid%" >nul`,
		`move /y "%appDir%\%oldExeName%" "%appDir%\%oldExeName%.update.bak" >nul`,
		`robocopy "%srcDir%" "%appDir%" /E /XD data ~archives /R:5 /W:1 >nul`,
		`if %errorlevel% GEQ 8 (`,
		`move /y "%appDir%\%oldExeName%.update.bak" "%appDir%\%oldExeName%" >nul`,
		`if exist "%appDir%\flutter_windows.dll" del /f /q "%appDir%\flutter_windows.dll" 2>nul`,
		`if exist "%appDir%\data\app.so" del /f /q "%appDir%\data\app.so" 2>nul`,
		`if exist "%appDir%\data\flutter_assets" rmdir /s /q "%appDir%\data\flutter_assets" 2>nul`,
		`findstr /C:"CarryDock Auto Update Script" "%appDir%\update_script.bat" >nul 2>nul`,
		`start "" /D "%appDir%" "%appDir%\%exeName%"`,
		`if exist "%zipPath%" del /f /q "%zipPath%" 2>nul`,
		`start /b "" cmd.exe /c "timeout /t 2 /nobreak >nul & rmdir /s /q "%tempDir%" 2>nul"`,
	}
	for _, want := range required {
		if !strings.Contains(script, want) {
			t.Errorf("update script missing %q", want)
		}
	}

	// 不得包含删除用户数据的语句
	for _, forbidden := range []string{"app_data.json", "app_data.lock", "~archives\\"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("update script should not reference %q", forbidden)
		}
	}

	// ZipPath 为空时不生成 zipPath 删除命令
	noZipScript := buildUpdateScript(updateScriptParams{
		AppDir:     `D:\CarryDock`,
		SourceDir:  `C:\Temp\pkg`,
		OldExeName: "CarryDock.exe",
		NewExeName: "CarryDock.exe",
		TempDir:    `C:\Temp\upd`,
	})
	if strings.Contains(noZipScript, "zipPath") {
		t.Error("expected no zipPath cleanup when ZipPath is empty")
	}
}

func TestFindDirWithExe(t *testing.T) {
	// 1. 平铺结构：extractDir/CarryDock.exe
	flatRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(flatRoot, "CarryDock.exe"), []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := findDirWithExe(flatRoot); got != flatRoot {
		t.Fatalf("flat findDirWithExe = %q, want %q", got, flatRoot)
	}

	// 2. 单层子目录结构：extractDir/CarryDock/CarryDock.exe
	nestedRoot := t.TempDir()
	nestedSub := filepath.Join(nestedRoot, "CarryDock")
	if err := os.MkdirAll(nestedSub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedSub, "CarryDock.exe"), []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := findDirWithExe(nestedRoot); got != nestedSub {
		t.Fatalf("nested findDirWithExe = %q, want %q", got, nestedSub)
	}

	// 3. 无 exe 目录
	emptyRoot := t.TempDir()
	if got := findDirWithExe(emptyRoot); got != "" {
		t.Fatalf("empty findDirWithExe = %q, want empty", got)
	}
}

func TestCheckForUpdateWithMockEndpoint(t *testing.T) {
	var mode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case "newer":
			_, _ = w.Write([]byte(`{
				"tag_name": "v99.0.0+9999",
				"body": "新版本更新日志",
				"assets": [
					{"name": "CarryDock-windows-amd64.zip", "browser_download_url": "https://example.com/CarryDock.zip"}
				]
			}`))
		case "uptodate":
			_, _ = w.Write([]byte(`{
				"tag_name": "v0.0.1+0",
				"body": "旧版本",
				"assets": [
					{"name": "CarryDock-windows-amd64.zip", "browser_download_url": "https://example.com/CarryDock.zip"}
				]
			}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "internal server error"}`))
		}
	}))
	defer srv.Close()

	svc := NewServiceWithEndpoint(srv.URL, srv.Client())

	mode = "newer"
	hasUpdate, info, err := svc.CheckForUpdate(context.Background(), false)
	if err != nil {
		t.Fatalf("CheckForUpdate(newer) unexpected error: %v", err)
	}
	if !hasUpdate || info == nil || info.Version != "v99.0.0+9999" || info.Changelog != "新版本更新日志" {
		t.Fatalf("unexpected newer result: hasUpdate=%v info=%+v", hasUpdate, info)
	}

	mode = "uptodate"
	hasUpdate, info, err = svc.CheckForUpdate(context.Background(), false)
	if err != nil {
		t.Fatalf("CheckForUpdate(uptodate) unexpected error: %v", err)
	}
	if hasUpdate || info == nil {
		t.Fatalf("expected up-to-date result, got hasUpdate=%v info=%+v", hasUpdate, info)
	}

	mode = "error"
	hasUpdate, info, err = svc.CheckForUpdate(context.Background(), false)
	if err == nil || hasUpdate || info != nil {
		t.Fatalf("expected error result, got hasUpdate=%v info=%+v err=%v", hasUpdate, info, err)
	}
}
