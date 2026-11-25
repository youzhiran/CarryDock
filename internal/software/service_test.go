package software

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/storage"
	"carrydock/internal/winutil"
)

type mockUserPathManager struct {
	entries []string
}

func (m *mockUserPathManager) ListUserPath() ([]string, error) {
	out := make([]string, len(m.entries))
	copy(out, m.entries)
	return out, nil
}

func (m *mockUserPathManager) AddUserPathDirs(dirs []string) ([]string, error) {
	updated, added, _ := winutil.AddDirsToPathList(m.entries, dirs)
	m.entries = updated
	return added, nil
}

func (m *mockUserPathManager) RemoveUserPathDirs(dirs []string) ([]string, error) {
	updated, removed, _ := winutil.RemoveDirsFromPathList(m.entries, dirs)
	m.entries = updated
	return removed, nil
}

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
	svc := NewService(setSvc)
	svc.SetUserPathManager(&mockUserPathManager{})
	return svc, setSvc, installDir
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

func TestResidualBackupAutoAssociationAndFallback(t *testing.T) {
	svc, _, _ := newTestService(t)
	backupDir, err := svc.ResolveBackupDirectoryPath()
	if err != nil {
		t.Fatal(err)
	}

	// 准备两个同名软件的历史残留备份（旧 + 新），以及另一个相似前缀版本的软件备份
	oldBak := filepath.Join(backupDir, "OnboardMemoryManager_2.7.4428-20261123_131205.zip")
	newBak := filepath.Join(backupDir, "OnboardMemoryManager_2.7.4428-20261123_203229.zip")
	makeZip(t, oldBak, map[string]string{"OnboardMemoryManager_2.7.4428.exe": "old"})
	makeZip(t, newBak, map[string]string{"OnboardMemoryManager_2.7.4428.exe": "new"})

	// 模拟用户删除软件后重新添加单文件 .exe（或压缩包）
	srcExe25 := filepath.Join(t.TempDir(), "OnboardMemoryManager_2.5.358.exe")
	srcExe27 := filepath.Join(t.TempDir(), "OnboardMemoryManager_2.7.4428.exe")
	if err := os.WriteFile(srcExe25, []byte("exe25"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcExe27, []byte("exe27"), 0o644); err != nil {
		t.Fatal(err)
	}

	if res := svc.AddSoftwareFromFile(srcExe25); res.Type != AddSuccess {
		t.Fatalf("add 2.5.358 failed: %v", res.Err)
	}
	res27 := svc.AddSoftwareFromFile(srcExe27)
	if res27.Type != AddDuplicate || res27.DuplicateInfo == nil || !res27.DuplicateInfo.IsVersionUpdate {
		t.Fatalf("expected version-update AddDuplicate for 2.7.4428 over 2.5.358, got %+v", res27)
	}
	// 选择「保留旧版并添加」，验证共存场景下的残留备份自动关联逻辑
	if res := svc.ResolveDuplicateAddition(*res27.DuplicateInfo, false, res27.DuplicateInfo.IntendedName); res.Type != AddSuccess {
		t.Fatalf("keep-both add 2.7.4428 failed: %v", res.Err)
	}

	// 1. 验证重新添加后，2.7.4428 自动关联最新的残留备份并写回 BackupPath，且 2.5.358 不会误关联
	managed, err := svc.LoadManagedSoftware()
	if err != nil || len(managed) != 2 {
		t.Fatalf("LoadManagedSoftware failed: len=%d err=%v", len(managed), err)
	}
	if managed[0].Name != "OnboardMemoryManager_2.5.358" || managed[0].BackupPath != "" {
		t.Fatalf("expected 2.5.358 to have empty BackupPath, got %+v", managed[0])
	}
	if managed[1].Name != "OnboardMemoryManager_2.7.4428" || managed[1].BackupPath != newBak {
		t.Fatalf("expected 2.7.4428 BackupPath=%q, got %q", newBak, managed[1].BackupPath)
	}

	list, err := svc.GetSoftwareList()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list {
		if item.Name == "OnboardMemoryManager_2.5.358" && item.IsBackupArchive {
			t.Fatalf("expected 2.5.358 IsBackupArchive=false on home list")
		}
		if item.Name == "OnboardMemoryManager_2.7.4428" && (!item.IsBackupArchive || item.BackupPath != newBak) {
			t.Fatalf("expected 2.7.4428 IsBackupArchive=true and BackupPath=%q, got %+v", newBak, item)
		}
	}

	// 2. 验证覆盖安装不会丢失已有关联的 BackupPath
	dupRes := svc.AddSoftwareFromFile(srcExe27)
	if dupRes.Type != AddDuplicate || dupRes.DuplicateInfo == nil {
		t.Fatalf("expected AddDuplicate, got %v", dupRes.Type)
	}
	overRes := svc.ResolveDuplicateAddition(*dupRes.DuplicateInfo, true, "")
	if overRes.Type != AddSuccess {
		t.Fatalf("ResolveDuplicateAddition overwrite failed: %v", overRes.Err)
	}
	managed, _ = svc.LoadManagedSoftware()
	if managed[1].BackupPath != newBak {
		t.Fatalf("expected BackupPath preserved after overwrite, got %q", managed[1].BackupPath)
	}

	// 3. 测试手动取消备份关联：取消关联后 BackupPath 清空，且后续 LoadManagedSoftware / GetSoftwareList 不会强行恢复关联
	if err := svc.UnlinkArchiveFromSoftware(newBak); err != nil {
		t.Fatalf("UnlinkArchiveFromSoftware(newBak) failed: %v", err)
	}
	managed, _ = svc.LoadManagedSoftware()
	if managed[1].BackupPath != "" || managed[1].IsBackupArchive {
		t.Fatalf("expected BackupPath cleared and IsBackupArchive=false after unlink, got %+v", managed[1])
	}
	// 重新手动关联回 newBak
	if err := svc.LinkArchiveToSoftware(managed[1].ID, newBak); err != nil {
		t.Fatalf("LinkArchiveToSoftware(newBak) failed: %v", err)
	}
	managed, _ = svc.LoadManagedSoftware()
	if managed[1].BackupPath != newBak || !managed[1].IsBackupArchive {
		t.Fatalf("expected BackupPath re-linked to %q, got %+v", newBak, managed[1])
	}

	// 4. 删除当前关联的最新备份后，应自动回退关联到剩余的旧备份 oldBak
	if err := svc.DeleteArchiveFile(newBak); err != nil {
		t.Fatalf("DeleteArchiveFile(newBak) failed: %v", err)
	}
	managed, _ = svc.LoadManagedSoftware()
	if managed[1].BackupPath != oldBak {
		t.Fatalf("expected BackupPath to fallback to %q, got %q", oldBak, managed[1].BackupPath)
	}

	// 5. 再删除最后一份旧备份后，BackupPath 应清空且主页 IsBackupArchive 为 false
	if err := svc.DeleteArchiveFile(oldBak); err != nil {
		t.Fatalf("DeleteArchiveFile(oldBak) failed: %v", err)
	}
	managed, _ = svc.LoadManagedSoftware()
	if managed[1].BackupPath != "" {
		t.Fatalf("expected BackupPath cleared after all backups deleted, got %q", managed[1].BackupPath)
	}
	list, _ = svc.GetSoftwareList()
	for _, item := range list {
		if item.Name == "OnboardMemoryManager_2.7.4428" && item.IsBackupArchive {
			t.Fatalf("expected 2.7.4428 IsBackupArchive=false after all backups deleted")
		}
	}
}

func TestCompareVersionsAndNormalizeSoftwareIdentityName(t *testing.T) {
	cmpCases := []struct {
		a, b string
		want int
	}{
		{"1.4.1.1026", "1.4.1.1024", 1},
		{"24.10", "24.9", 1},
		{"v1.2.0", "1.2", 0},
		{"1.2.0", "1.2.0-beta.1", 1},
		{"1.2.0-beta.1", "1.2.0", -1},
		{"4.5", "4.6", -1},
		{"", "1.0", 0},
	}
	for _, tc := range cmpCases {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}

	normCases := []struct {
		in      string
		want    string
		wantVer bool
	}{
		{"7-Zip", "7 zip", false},
		{"7-Zip-24.08-x64", "7 zip", true},
		{"Everything-1.4.1.1026.x64-Portable", "everything", true},
		{"rufus-4.6p", "rufus", true},
		{"MyApp-Client-1.0", "myapp client", true},
		{"MyApp-Server-1.0", "myapp server", true},
		{"Office_Tool_with_runtime_v10.26.36.0_x64", "office tool", true},
	}
	for _, tc := range normCases {
		got, hasVer := normalizeSoftwareIdentityName(tc.in)
		if got != tc.want || hasVer != tc.wantVer {
			t.Errorf("normalizeSoftwareIdentityName(%q) = (%q, %v), want (%q, %v)", tc.in, got, hasVer, tc.want, tc.wantVer)
		}
	}
}

func TestAddNewVersionArchivePromptsAndReplacesOldVersion(t *testing.T) {
	svc, _, installDir := newTestService(t)
	archDir, err := svc.ResolveArchiveDirectoryPath()
	if err != nil {
		t.Fatal(err)
	}

	// 0. 预先安装一个置顶占位软件，验证替换后原 SortOrder 保持不变
	otherZip := filepath.Join(t.TempDir(), "FirstApp.zip")
	makeZip(t, otherZip, map[string]string{"FirstApp/first.exe": "1"})
	if res := svc.AddSoftwareFromFile(otherZip); res.Type != AddSuccess {
		t.Fatalf("add FirstApp failed: %+v", res)
	}

	// 1. 安装 MyTool-1.0.0.zip
	oldZip := filepath.Join(t.TempDir(), "MyTool-1.0.0.zip")
	makeZip(t, oldZip, map[string]string{"MyTool/mytool.exe": "v1"})
	if res := svc.AddSoftwareFromFile(oldZip); res.Type != AddSuccess {
		t.Fatalf("add MyTool-1.0.0 failed: %+v", res)
	}

	// 再加第三个软件以验证 SortOrder=1 的位置保留
	thirdZip := filepath.Join(t.TempDir(), "ThirdApp.zip")
	makeZip(t, thirdZip, map[string]string{"ThirdApp/third.exe": "3"})
	if res := svc.AddSoftwareFromFile(thirdZip); res.Type != AddSuccess {
		t.Fatalf("add ThirdApp failed: %+v", res)
	}

	managedBefore, _ := svc.LoadManagedSoftware()
	var oldSw model.Software
	for _, s := range managedBefore {
		if s.Name == "MyTool-1.0.0" {
			oldSw = s
			break
		}
	}
	if oldSw.ID == "" {
		t.Fatalf("expected MyTool-1.0.0 in managed list: %+v", managedBefore)
	}

	// 2. 放入新版本 MyTool-2.0.0.zip -> 应识别为旧版本替换提示且不残留 staging 目录
	newZip := filepath.Join(t.TempDir(), "MyTool-2.0.0.zip")
	makeZip(t, newZip, map[string]string{"MyTool/mytool.exe": "v2"})
	dupRes := svc.AddSoftwareFromFile(newZip)
	if dupRes.Type != AddDuplicate || dupRes.DuplicateInfo == nil {
		t.Fatalf("expected AddDuplicate for MyTool-2.0.0.zip, got %+v", dupRes)
	}
	info := *dupRes.DuplicateInfo
	if !info.IsVersionUpdate || info.TargetPathCollides {
		t.Fatalf("expected IsVersionUpdate=true TargetPathCollides=false, got %+v", info)
	}
	if info.ExistingManagedSoftware == nil || info.ExistingManagedSoftware.ID != oldSw.ID {
		t.Fatalf("expected matched ExistingManagedSoftware ID %q, got %+v", oldSw.ID, info.ExistingManagedSoftware)
	}
	entries, _ := os.ReadDir(installDir)
	for _, ent := range entries {
		if strings.HasPrefix(ent.Name(), ".carrydock_staging_") {
			t.Fatalf("expected staging dir cleaned up after duplicate prompt, found %s", ent.Name())
		}
	}

	// 3. 模拟「备份并替换」：先为旧版创建备份，再调用 ResolveDuplicateAddition(overwrite=true)
	bakPath, err := svc.CreateBackupForSoftware(*info.ExistingManagedSoftware)
	if err != nil {
		t.Fatalf("CreateBackupForSoftware failed: %v", err)
	}
	cp := *info.ExistingManagedSoftware
	cp.BackupPath = bakPath
	info.ExistingManagedSoftware = &cp

	repRes := svc.ResolveDuplicateAddition(info, true, "")
	if repRes.Type != AddSuccess {
		t.Fatalf("expected AddSuccess after replace, got %+v", repRes)
	}

	if _, err := os.Stat(oldSw.InstallPath); !os.IsNotExist(err) {
		t.Fatalf("expected old install dir %s to be removed", oldSw.InstallPath)
	}
	if _, err := os.Stat(oldSw.ArchivePath); !os.IsNotExist(err) {
		t.Fatalf("expected old archive %s to be removed", oldSw.ArchivePath)
	}
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("expected backup %s to remain intact: %v", bakPath, err)
	}

	managedAfter, _ := svc.LoadManagedSoftware()
	var replacedSw model.Software
	for _, s := range managedAfter {
		if s.ID == oldSw.ID || s.Name == "MyTool-1.0.0" {
			t.Fatalf("old software entry should be replaced, still found %+v", s)
		}
		if s.Name == "MyTool-2.0.0" {
			replacedSw = s
		}
	}
	if replacedSw.ID == "" {
		t.Fatalf("expected MyTool-2.0.0 in managed list: %+v", managedAfter)
	}
	if replacedSw.SortOrder != oldSw.SortOrder {
		t.Fatalf("expected preserved SortOrder=%d, got %d", oldSw.SortOrder, replacedSw.SortOrder)
	}
	if replacedSw.BackupPath != bakPath {
		t.Fatalf("expected preserved BackupPath=%q, got %q", bakPath, replacedSw.BackupPath)
	}
	if replacedSw.ArchivePath != filepath.Join(archDir, "MyTool-2.0.0.zip") {
		t.Fatalf("unexpected ArchivePath: %q", replacedSw.ArchivePath)
	}

	// 4. 验证无版本号的干净目录名（CoolTool）：原地升级并自动匹配多可执行文件中的原主程序
	cleanZip := filepath.Join(t.TempDir(), "CoolTool.zip")
	makeZip(t, cleanZip, map[string]string{
		"CoolTool/cool.exe":  "v1",
		"CoolTool/extra.exe": "e1",
	})
	pendingClean := svc.AddSoftwareFromFile(cleanZip)
	if pendingClean.Type != AddNeedsSelection || pendingClean.PendingAddition == nil {
		t.Fatalf("expected AddNeedsSelection for CoolTool, got %+v", pendingClean)
	}
	var chosenCoolExe string
	for _, ex := range pendingClean.PendingAddition.ExecutablePaths {
		if strings.EqualFold(filepath.Base(ex), "cool.exe") {
			chosenCoolExe = ex
			break
		}
	}
	if err := svc.CompleteSoftwareAddition(*pendingClean.PendingAddition, chosenCoolExe); err != nil {
		t.Fatal(err)
	}

	cleanNewZip := filepath.Join(t.TempDir(), "CoolTool-2.0.0.zip")
	makeZip(t, cleanNewZip, map[string]string{
		"CoolTool-2.0.0/cool.exe":  "v2",
		"CoolTool-2.0.0/extra.exe": "e2",
	})
	dupClean := svc.AddSoftwareFromFile(cleanNewZip)
	if dupClean.Type != AddDuplicate || dupClean.DuplicateInfo == nil || !dupClean.DuplicateInfo.IsVersionUpdate {
		t.Fatalf("expected version update duplicate for CoolTool-2.0.0.zip, got %+v", dupClean)
	}
	repClean := svc.ResolveDuplicateAddition(*dupClean.DuplicateInfo, true, "")
	if repClean.Type != AddSuccess {
		t.Fatalf("expected CoolTool in-place upgrade to auto-match cool.exe and return AddSuccess, got %+v", repClean)
	}
	managedClean, _ := svc.LoadManagedSoftware()
	var coolSw model.Software
	for _, s := range managedClean {
		if s.Name == "CoolTool" {
			coolSw = s
			break
		}
	}
	if coolSw.ID == "" || !strings.EqualFold(filepath.Base(coolSw.ExecutablePath), "cool.exe") {
		t.Fatalf("expected CoolTool directory preserved with cool.exe selected, got %+v", coolSw)
	}
	content, err := os.ReadFile(coolSw.ExecutablePath)
	if err != nil || string(content) != "v2" {
		t.Fatalf("expected CoolTool/cool.exe updated to v2, got %q (err=%v)", string(content), err)
	}

	// 5. 验证「保留旧版并添加」后新旧版本共存
	bothZip := filepath.Join(t.TempDir(), "CoolTool-3.0.0.zip")
	makeZip(t, bothZip, map[string]string{"CoolTool/cool.exe": "v3"})
	dupBoth := svc.AddSoftwareFromFile(bothZip)
	if dupBoth.Type != AddDuplicate || dupBoth.DuplicateInfo == nil {
		t.Fatalf("expected AddDuplicate for CoolTool-3.0.0.zip, got %+v", dupBoth)
	}
	keepRes := svc.ResolveDuplicateAddition(*dupBoth.DuplicateInfo, false, dupBoth.DuplicateInfo.IntendedName)
	if keepRes.Type != AddSuccess {
		t.Fatalf("expected Keep Both AddSuccess, got %+v", keepRes)
	}
	finalList, _ := svc.LoadManagedSoftware()
	hasV2Clean, hasV3 := false, false
	for _, s := range finalList {
		if s.Name == "CoolTool" {
			hasV2Clean = true
		}
		if s.Name == "CoolTool-3.0.0" {
			hasV3 = true
		}
	}
	if !hasV2Clean || !hasV3 {
		t.Fatalf("expected both CoolTool and CoolTool-3.0.0 to coexist, got %+v", finalList)
	}
}

func TestAddNewVersionExecutablePromptsAndReplacesOldVersion(t *testing.T) {
	svc, _, _ := newTestService(t)
	tmpDir := t.TempDir()
	oldExe := filepath.Join(tmpDir, "rufus-4.5.exe")
	newExe := filepath.Join(tmpDir, "rufus-4.6.exe")
	if err := os.WriteFile(oldExe, []byte("exe-4.5"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newExe, []byte("exe-4.6"), 0o644); err != nil {
		t.Fatal(err)
	}

	if res := svc.AddSoftwareFromFile(oldExe); res.Type != AddSuccess {
		t.Fatalf("add rufus-4.5.exe failed: %+v", res)
	}
	managedBefore, _ := svc.LoadManagedSoftware()
	if len(managedBefore) != 1 {
		t.Fatalf("expected 1 managed item, got %d", len(managedBefore))
	}
	oldSw := managedBefore[0]

	dupRes := svc.AddSoftwareFromFile(newExe)
	if dupRes.Type != AddDuplicate || dupRes.DuplicateInfo == nil {
		t.Fatalf("expected AddDuplicate for rufus-4.6.exe, got %+v", dupRes)
	}
	if !dupRes.DuplicateInfo.IsVersionUpdate || dupRes.DuplicateInfo.TargetPathCollides {
		t.Fatalf("expected IsVersionUpdate=true TargetPathCollides=false, got %+v", dupRes.DuplicateInfo)
	}
	if dupRes.DuplicateInfo.ExistingManagedSoftware == nil || dupRes.DuplicateInfo.ExistingManagedSoftware.ID != oldSw.ID {
		t.Fatalf("expected matched old rufus, got %+v", dupRes.DuplicateInfo.ExistingManagedSoftware)
	}

	repRes := svc.ResolveDuplicateAddition(*dupRes.DuplicateInfo, true, "")
	if repRes.Type != AddSuccess {
		t.Fatalf("expected replace success for rufus-4.6.exe, got %+v", repRes)
	}
	if _, err := os.Stat(oldSw.InstallPath); !os.IsNotExist(err) {
		t.Fatalf("expected old rufus-4.5 install dir removed")
	}
	if _, err := os.Stat(oldSw.ArchivePath); !os.IsNotExist(err) {
		t.Fatalf("expected old rufus-4.5 archive removed")
	}
	managedAfter, _ := svc.LoadManagedSoftware()
	if len(managedAfter) != 1 || managedAfter[0].Name != "rufus-4.6" {
		t.Fatalf("expected only rufus-4.6 in managed list, got %+v", managedAfter)
	}
}

func copyDirForTest(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyDirForTest(%s -> %s) failed: %v", src, dst, err)
	}
}

func TestRelativePathsUSBMigrationEndToEnd(t *testing.T) {
	rootA := filepath.Join(t.TempDir(), "OldPC", "CarryDock")
	storeA := storage.NewAtBaseDir(rootA)
	setSvcA := settings.NewService(storeA)
	installDirA := filepath.Join(rootA, "Apps")
	if err := os.MkdirAll(installDirA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := setSvcA.SaveInstallPath(installDirA); err != nil {
		t.Fatal(err)
	}
	svcA := NewService(setSvcA)
	svcA.SetUserPathManager(&mockUserPathManager{})

	srcZip := filepath.Join(t.TempDir(), "NotePad.zip")
	makeZip(t, srcZip, map[string]string{
		"NotePad/notepad.exe": "bin",
	})
	if res := svcA.AddSoftwareFromFile(srcZip); res.Type != AddSuccess {
		t.Fatalf("AddSoftwareFromFile failed: %+v", res)
	}
	listA, err := svcA.LoadManagedSoftware()
	if err != nil || len(listA) != 1 {
		t.Fatalf("LoadManagedSoftware failed: %v, len=%d", err, len(listA))
	}
	if _, err := svcA.CreateBackupForSoftware(listA[0]); err != nil {
		t.Fatalf("CreateBackupForSoftware failed: %v", err)
	}

	// 开启相对路径模式，转换配置和 software_list.json
	summary, err := svcA.SetRelativePathsMode(true)
	if err != nil {
		t.Fatalf("SetRelativePathsMode(true) failed: %v", err)
	}
	if summary.SettingsConverted != 1 || summary.SoftwareCount != 1 || summary.SoftwareConverted < 4 || summary.SoftwareSkipped != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	rawCfg, err := os.ReadFile(storeA.FilePath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawCfg), rootA) {
		t.Fatalf("expected app_data.json to contain no absolute root %q, got:\n%s", rootA, string(rawCfg))
	}
	archDirA, err := svcA.ResolveArchiveDirectoryPath()
	if err != nil {
		t.Fatal(err)
	}
	rawListA, err := os.ReadFile(filepath.Join(archDirA, softwareListFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawListA), rootA) {
		t.Fatalf("expected software_list.json to contain no absolute root %q, got:\n%s", rootA, string(rawListA))
	}

	// 模拟将整个 CarryDock 文件夹拷入新电脑/U 盘路径 rootB
	rootB := filepath.Join(t.TempDir(), "MovedUSB", "CarryDock")
	copyDirForTest(t, rootA, rootB)

	storeB := storage.NewAtBaseDir(rootB)
	setSvcB := settings.NewService(storeB)
	svcB := NewService(setSvcB)
	svcB.SetUserPathManager(&mockUserPathManager{})

	wantInstallB := filepath.Join(rootB, "Apps")
	if got := setSvcB.GetInstallPath(); got != wantInstallB {
		t.Fatalf("GetInstallPath after move = %q, want %q", got, wantInstallB)
	}

	listB, err := svcB.LoadManagedSoftware()
	if err != nil || len(listB) != 1 {
		t.Fatalf("LoadManagedSoftware after move failed: %v, len=%d", err, len(listB))
	}
	swB := listB[0]
	if !swB.InstallExists || !swB.ArchiveExists || !swB.IsBackupArchive {
		t.Fatalf("expected moved software artifacts to exist under rootB, got %+v", swB)
	}
	if !strings.HasPrefix(swB.InstallPath, rootB) || !strings.HasPrefix(swB.ExecutablePath, rootB) ||
		!strings.HasPrefix(swB.ArchivePath, rootB) || !strings.HasPrefix(swB.BackupPath, rootB) {
		t.Fatalf("expected all runtime paths resolved under rootB %q, got %+v", rootB, swB)
	}

	// 在新位置新增软件与重命名目录，仍应以相对路径落盘
	extraExe := filepath.Join(t.TempDir(), "QuickTool.exe")
	if err := os.WriteFile(extraExe, []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := svcB.AddSoftwareFromFile(extraExe); res.Type != AddSuccess {
		t.Fatalf("AddSoftwareFromFile after move failed: %+v", res)
	}
	if err := svcB.RenameSoftwareFolder(swB, "NotePadPro"); err != nil {
		t.Fatalf("RenameSoftwareFolder after move failed: %v", err)
	}

	archDirB, _ := svcB.ResolveArchiveDirectoryPath()
	rawListB, err := os.ReadFile(filepath.Join(archDirB, softwareListFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawListB), rootB) {
		t.Fatalf("expected newly saved software_list.json under rootB to stay relative, got:\n%s", string(rawListB))
	}

	// 关闭相对路径模式，恢复为当前机器绝对路径
	offSummary, err := svcB.SetRelativePathsMode(false)
	if err != nil {
		t.Fatalf("SetRelativePathsMode(false) failed: %v", err)
	}
	if offSummary.SettingsConverted != 1 || offSummary.SoftwareConverted == 0 {
		t.Fatalf("unexpected disable summary: %+v", offSummary)
	}
	rawListBOff, _ := os.ReadFile(filepath.Join(archDirB, softwareListFileName))
	if !strings.Contains(strings.ReplaceAll(string(rawListBOff), `\\`, `\`), rootB) {
		t.Fatalf("expected software_list.json to contain rootB after disabling relative mode, got:\n%s", string(rawListBOff))
	}
}

func TestInstallPathDotExcludesDataDirectory(t *testing.T) {
	base := filepath.Join(t.TempDir(), "CarryDockPortable")
	store := storage.NewAtBaseDir(base)
	setSvc := settings.NewService(store)
	if _, err := setSvc.ApplyRelativePathsMode(true); err != nil {
		t.Fatal(err)
	}
	if err := setSvc.SaveInstallPath("."); err != nil {
		t.Fatal(err)
	}
	// 创建 CarryDock 自身的 data/ 目录及一个绿色软件目录
	if err := os.MkdirAll(filepath.Join(base, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "MyApp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "MyApp", "myapp.exe"), []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(setSvc)
	all, err := svc.GetSoftwareList()
	if err != nil {
		t.Fatalf("GetSoftwareList failed: %v", err)
	}
	for _, sw := range all {
		if strings.EqualFold(sw.Name, "data") {
			t.Fatalf("expected CarryDock data directory to be excluded from software list, got %+v", all)
		}
	}

	summary, err := svc.ArchiveInstallSubdirectories(false, nil)
	if err != nil {
		t.Fatalf("ArchiveInstallSubdirectories failed: %v", err)
	}
	for _, name := range summary.Archived {
		if strings.EqualFold(name, "data") {
			t.Fatalf("expected CarryDock data directory not to be archived as software")
		}
	}
}

func TestDeveloperLaunchConfigAndUserPathLifecycle(t *testing.T) {
	svc, _, installDir := newTestService(t)
	mockPath := &mockUserPathManager{
		entries: []string{`%USERPROFILE%\.cargo\bin`},
	}
	svc.SetUserPathManager(mockPath)

	// 1. 准备 JDK 风格（主程序在 bin\java.exe）与 Git 风格（cmd\git.exe）软件
	jdkZip := filepath.Join(t.TempDir(), "OpenJDK21.zip")
	makeZip(t, jdkZip, map[string]string{
		"OpenJDK21/bin/java.exe": "bin",
		"OpenJDK21/lib/rt.jar":   "lib",
	})
	if res := svc.AddSoftwareFromFile(jdkZip); res.Type != AddSuccess {
		t.Fatalf("AddSoftwareFromFile(jdk) failed: %+v", res)
	}
	gitZip := filepath.Join(t.TempDir(), "PortableGit.zip")
	makeZip(t, gitZip, map[string]string{
		"PortableGit/cmd/git.exe":     "bin",
		"PortableGit/usr/bin/awk.exe": "bin",
	})
	if res := svc.AddSoftwareFromFile(gitZip); res.Type != AddNeedsSelection {
		t.Fatalf("expected AddNeedsSelection for git, got %+v", res)
	} else if err := svc.CompleteSoftwareAddition(*res.PendingAddition, res.PendingAddition.ExecutablePaths[0]); err != nil {
		t.Fatalf("CompleteSoftwareAddition(git) failed: %v", err)
	}

	list, err := svc.LoadManagedSoftware()
	if err != nil || len(list) != 2 {
		t.Fatalf("LoadManagedSoftware = %v, len=%d", err, len(list))
	}
	var jdk, git model.Software
	for _, item := range list {
		if strings.Contains(item.Name, "OpenJDK") {
			jdk = item
		} else {
			git = item
		}
	}

	// 2. 验证 SuggestPathCandidates 推荐 bin / cmd
	jdkCands, jdkDef := svc.SuggestPathCandidates(jdk)
	if len(jdkCands) < 2 || len(jdkDef) != 1 || !winutil.SamePath(jdkDef[0], filepath.Join(jdk.InstallPath, "bin")) {
		t.Fatalf("unexpected JDK path suggestion: cands=%v def=%v", jdkCands, jdkDef)
	}
	gitCands, gitDef := svc.SuggestPathCandidates(git)
	if len(gitCands) < 3 || len(gitDef) != 1 || !winutil.SamePath(gitDef[0], filepath.Join(git.InstallPath, "cmd")) {
		t.Fatalf("unexpected Git path suggestion: cands=%v def=%v", gitCands, gitDef)
	}

	// 3. 保存启动参数与环境变量
	if err := svc.UpdateSoftwareLaunchConfig(jdk.ID, "-Xmx2g -jar app.jar", map[string]string{
		"JAVA_HOME": "%INSTALL_DIR%",
	}); err != nil {
		t.Fatalf("UpdateSoftwareLaunchConfig failed: %v", err)
	}
	// 注册 Path
	synced, err := svc.RegisterSoftwarePath(jdk.ID, []string{filepath.Join(jdk.InstallPath, "bin")})
	if err != nil || len(synced) != 1 {
		t.Fatalf("RegisterSoftwarePath failed: synced=%v err=%v", synced, err)
	}
	presence, err := svc.QuerySoftwareUserPathPresence(synced)
	if err != nil || !presence[strings.ToLower(winutil.NormalizePathEntry(synced[0]))] {
		t.Fatalf("expected registered dir present in mock user path, presence=%v err=%v", presence, err)
	}

	// 4. 重命名文件夹应自动迁移 PathEntries 与系统用户 Path
	if err := svc.RenameSoftwareFolder(jdk, "JDK21"); err != nil {
		t.Fatalf("RenameSoftwareFolder failed: %v", err)
	}
	newBin := filepath.Join(installDir, "JDK21", "bin")
	if winutil.PathListContains(mockPath.entries, filepath.Join(installDir, "OpenJDK21", "bin")) {
		t.Fatalf("expected old bin path removed after rename, got %v", mockPath.entries)
	}
	if !winutil.PathListContains(mockPath.entries, newBin) {
		t.Fatalf("expected new bin path %q registered after rename, got %v", newBin, mockPath.entries)
	}

	// 5. 开启相对路径模式时 PathEntries 应转为相对路径并可还原解析
	if _, err := svc.SetRelativePathsMode(true); err != nil {
		t.Fatalf("SetRelativePathsMode(true) failed: %v", err)
	}
	reloaded, err := svc.LoadManagedSoftware()
	if err != nil {
		t.Fatal(err)
	}
	var jdkReloaded model.Software
	for _, item := range reloaded {
		if item.ID == jdk.ID {
			jdkReloaded = item
		}
	}
	if jdkReloaded.LaunchArgs != "-Xmx2g -jar app.jar" || jdkReloaded.LaunchEnv["JAVA_HOME"] != "%INSTALL_DIR%" {
		t.Fatalf("launch config lost after reload: %+v", jdkReloaded)
	}
	if len(jdkReloaded.PathEntries) != 1 || !winutil.SamePath(jdkReloaded.PathEntries[0], newBin) {
		t.Fatalf("expected resolved PathEntries [%q], got %v", newBin, jdkReloaded.PathEntries)
	}

	// 6. 移除托管（不删除安装目录）时，仍应自动从用户 Path 清理该软件注册的目录，并保留原有 %USERPROFILE% 条目
	if err := svc.DeleteSoftware(jdkReloaded, false, false); err != nil {
		t.Fatalf("DeleteSoftware failed: %v", err)
	}
	if winutil.PathListContains(mockPath.entries, newBin) {
		t.Fatalf("expected user Path cleaned up on DeleteSoftware, got %v", mockPath.entries)
	}
	if len(mockPath.entries) != 1 || mockPath.entries[0] != `%USERPROFILE%\.cargo\bin` {
		t.Fatalf("expected unrelated user Path entries intact, got %v", mockPath.entries)
	}
}

func TestOverridePreservesLaunchConfigAndUpdatesPath(t *testing.T) {
	svc, _, installDir := newTestService(t)
	mockPath := &mockUserPathManager{}
	svc.SetUserPathManager(mockPath)

	v1Zip := filepath.Join(t.TempDir(), "Node-v20.0.zip")
	makeZip(t, v1Zip, map[string]string{
		"Node-v20.0/node.exe": "bin",
	})
	if res := svc.AddSoftwareFromFile(v1Zip); res.Type != AddSuccess {
		t.Fatalf("add v1 failed: %+v", res)
	}
	list, _ := svc.LoadManagedSoftware()
	v1 := list[0]
	if err := svc.UpdateSoftwareLaunchConfig(v1.ID, "--inspect", map[string]string{"NODE_ENV": "development"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RegisterSoftwarePath(v1.ID, []string{v1.InstallPath}); err != nil {
		t.Fatal(err)
	}

	v2Zip := filepath.Join(t.TempDir(), "Node-v22.0.zip")
	makeZip(t, v2Zip, map[string]string{
		"Node-v22.0/node.exe": "bin",
	})
	dup := svc.AddSoftwareFromFile(v2Zip)
	if dup.Type != AddDuplicate || dup.DuplicateInfo == nil {
		t.Fatalf("expected AddDuplicate for v2, got %+v", dup)
	}
	if res := svc.ResolveDuplicateAddition(*dup.DuplicateInfo, true, ""); res.Type != AddSuccess {
		t.Fatalf("ResolveDuplicateAddition overwrite failed: %+v", res)
	}

	after, err := svc.LoadManagedSoftware()
	if err != nil || len(after) != 1 {
		t.Fatalf("LoadManagedSoftware after override failed: %v len=%d", err, len(after))
	}
	v2 := after[0]
	wantV2Dir := filepath.Join(installDir, "Node-v22.0")
	if v2.LaunchArgs != "--inspect" || v2.LaunchEnv["NODE_ENV"] != "development" {
		t.Fatalf("expected launch config preserved on override, got %+v", v2)
	}
	if len(v2.PathEntries) != 1 || !winutil.SamePath(v2.PathEntries[0], wantV2Dir) {
		t.Fatalf("expected PathEntries remapped to %q, got %v", wantV2Dir, v2.PathEntries)
	}
	if winutil.PathListContains(mockPath.entries, v1.InstallPath) || !winutil.PathListContains(mockPath.entries, wantV2Dir) {
		t.Fatalf("expected mock user Path updated to %q, got %v", wantV2Dir, mockPath.entries)
	}
}
