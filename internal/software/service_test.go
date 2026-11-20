package software

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/storage"
)

func newTestService(t *testing.T) (*Service, *settings.Service, string) {
	t.Helper()
	tmp := t.TempDir()
	store := storage.NewAtDir(filepath.Join(tmp, "data"))
	setSvc := settings.NewService(store)
	installDir := filepath.Join(tmp, "Portable")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := setSvc.SaveInstallPath(installDir); err != nil {
		t.Fatal(err)
	}
	return NewService(setSvc), setSvc, installDir
}

func makeZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFlattenSameNameAndMultiLevel(t *testing.T) {
	root := t.TempDir()
	// 经典嵌套同名目录冲突场景：root/foo/foo/app.exe + __MACOSX
	nested := filepath.Join(root, "foo", "foo")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "__MACOSX"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "app.exe"), []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !FlattenRedundantTopDirectories(root) {
		t.Fatal("expected flatten to return true")
	}
	if _, err := os.Stat(filepath.Join(root, "app.exe")); err != nil {
		t.Fatalf("expected app.exe at root after flattening: %v", err)
	}
}

func TestSanitizeSoftwareName(t *testing.T) {
	if got := SanitizeSoftwareName("CON"); got != "_CON" {
		t.Errorf("SanitizeSoftwareName(CON) = %q, want _CON", got)
	}
	if got := SanitizeSoftwareName("  app:name... "); got != "app_name" {
		t.Errorf("SanitizeSoftwareName(app:name...) = %q, want app_name", got)
	}
}

func TestAddArchiveMultiExeCancelDoesNotDeletePreexistingArchive(t *testing.T) {
	svc, _, _ := newTestService(t)
	archDir, err := svc.ResolveArchiveDirectoryPath()
	if err != nil {
		t.Fatal(err)
	}
	existingArchive := filepath.Join(archDir, "MultiApp.zip")
	makeZip(t, existingArchive, map[string]string{
		"MultiApp/a.exe": "a",
		"MultiApp/b.exe": "b",
	})

	res := svc.AddSoftwareFromFile(existingArchive)
	if res.Type != AddNeedsSelection || res.PendingAddition == nil {
		t.Fatalf("expected AddNeedsSelection, got %v (err=%v)", res.Type, res.Err)
	}
	if res.PendingAddition.CreatedArchive {
		t.Fatal("expected CreatedArchive=false when restoring from archive already in ~archives")
	}

	// 用户取消选择主程序时，源归档文件绝对不能被误删！
	svc.CancelPendingAddition(*res.PendingAddition)
	if _, err := os.Stat(existingArchive); err != nil {
		t.Fatalf("existing archive was wrongly deleted on cancel: %v", err)
	}
	if _, err := os.Stat(res.PendingAddition.InstallPath); !os.IsNotExist(err) {
		t.Fatalf("expected temporary install directory to be cleaned up")
	}
}

func TestLoadManagedSoftwareRespectsPersistedSortOrder(t *testing.T) {
	svc, _, installDir := newTestService(t)
	archDir, err := svc.ResolveArchiveDirectoryPath()
	if err != nil {
		t.Fatal(err)
	}
	rawList := []model.Software{
		{ID: "b", Name: "Beta", InstallPath: filepath.Join(installDir, "Beta"), SortOrder: 2},
		{ID: "a", Name: "Alpha", InstallPath: filepath.Join(installDir, "Alpha"), SortOrder: 0},
		{ID: "c", Name: "Gamma", InstallPath: filepath.Join(installDir, "Gamma"), SortOrder: 1},
	}
	data, _ := json.Marshal(rawList)
	if err := os.WriteFile(filepath.Join(archDir, softwareListFileName), data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := svc.LoadManagedSoftware()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 3 || loaded[0].ID != "a" || loaded[1].ID != "c" || loaded[2].ID != "b" {
		t.Fatalf("unexpected sort order: %+v", loaded)
	}
}

func TestCustomArchiveDirBasenameDoesNotHideInstallFolder(t *testing.T) {
	svc, setSvc, installDir := newTestService(t)
	customArchive := filepath.Join(t.TempDir(), "MyApps")
	if err := setSvc.SaveArchivePath(customArchive); err != nil {
		t.Fatal(err)
	}
	// 安装目录下恰好存在同名目录 MyApps
	userFolder := filepath.Join(installDir, "MyApps")
	if err := os.MkdirAll(userFolder, 0o755); err != nil {
		t.Fatal(err)
	}

	list, err := svc.GetSoftwareList()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range list {
		if s.Status == model.StatusUnknownInstall && s.Name == "MyApps" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected MyApps install folder to be discovered, got %+v", list)
	}
}

func TestAddRenameBackupRehostAndDelete(t *testing.T) {
	svc, _, installDir := newTestService(t)
	srcZip := filepath.Join(t.TempDir(), "CoolTool.zip")
	makeZip(t, srcZip, map[string]string{
		"CoolTool/cool.exe": "binary",
	})

	res := svc.AddSoftwareFromFile(srcZip)
	if res.Type != AddSuccess {
		t.Fatalf("AddSoftwareFromFile failed: type=%v err=%v", res.Type, res.Err)
	}

	list, err := svc.GetSoftwareList()
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 managed software, got %d (err=%v)", len(list), err)
	}
	sw := list[0]

	// 拒绝路径穿越重命名
	if err := svc.RenameSoftwareFolder(sw, "../escape"); err == nil {
		t.Fatal("expected path traversal rename to fail")
	}
	// 正常重命名
	if err := svc.RenameSoftwareFolder(sw, "CoolToolV2"); err != nil {
		t.Fatalf("RenameSoftwareFolder failed: %v", err)
	}

	list, _ = svc.GetSoftwareList()
	sw = list[0]
	if sw.Name != "CoolToolV2" || filepath.Base(sw.InstallPath) != "CoolToolV2" {
		t.Fatalf("unexpected software after rename: %+v", sw)
	}
	if filepath.Dir(sw.ExecutablePath) != filepath.Join(installDir, "CoolToolV2") {
		t.Fatalf("executablePath not updated after rename: %s", sw.ExecutablePath)
	}

	// 创建备份并删除安装目录与原归档 -> 验证能从 BackupPath 重新托管！
	backupZip, err := svc.CreateBackupForSoftware(sw)
	if err != nil {
		t.Fatalf("CreateBackupForSoftware failed: %v", err)
	}
	_ = os.RemoveAll(sw.InstallPath)
	_ = os.Remove(sw.ArchivePath)

	list, _ = svc.GetSoftwareList()
	sw = list[0]
	if sw.InstallExists || !sw.IsBackupArchive {
		t.Fatalf("expected missing install & backup detected, got %+v", sw)
	}

	rehostRes := svc.RehostSoftware(sw)
	if rehostRes.Type != AddSuccess {
		t.Fatalf("RehostSoftware from backup failed: type=%v err=%v", rehostRes.Type, rehostRes.Err)
	}
	if _, err := os.Stat(filepath.Join(sw.InstallPath, "cool.exe")); err != nil {
		t.Fatalf("expected cool.exe restored after rehost: %v", err)
	}

	// 删除备份文件应自动清除软件的 BackupPath 关联
	if err := svc.DeleteArchiveFile(backupZip); err != nil {
		t.Fatalf("DeleteArchiveFile failed: %v", err)
	}
	managed, _ := svc.LoadManagedSoftware()
	if managed[0].BackupPath != "" {
		t.Fatalf("expected BackupPath to be cleared, got %q", managed[0].BackupPath)
	}
}
