package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/software"
	"carrydock/internal/storage"
	"carrydock/internal/updater"
	"carrydock/internal/version"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestAppNativeUIViewsAndNavigation(t *testing.T) {
	tmp := t.TempDir()
	store := storage.NewAtDir(filepath.Join(tmp, "data"))
	setSvc := settings.NewService(store)
	swSvc := software.NewService(setSvc)
	updSvc := updater.NewService()

	a := New(setSvc, swSvc, updSvc, nil)
	a.homeNotConfigured = true

	tt := ui.NewTester(a.View, 1020, 680)

	// 1. 初始主页未配置安装路径时展示引导提示
	if !tt.HasText("尚未配置绿色软件安装目录") {
		t.Fatalf("expected unconfigured prompt on home view, got texts: %q", tt.Texts())
	}

	// 2. 点击「前往设置」切换到设置页
	if err := tt.Click("前往设置"); err != nil {
		t.Fatalf("click settings CTA failed: %v", err)
	}
	if a.page != pageSettings {
		t.Fatalf("expected pageSettings, got %v", a.page)
	}
	if !tt.HasText("绿色软件安装目录") {
		t.Fatalf("expected settings view content, got texts: %q", tt.Texts())
	}

	// 3. 配置安装路径并切换到归档管理页
	installDir := filepath.Join(tmp, "Portable")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	a.setInstallPath = installDir
	a.saveInstallPath()
	tt.Frame()

	if err := tt.Click("归档管理"); err != nil {
		t.Fatalf("click Archive Manager nav failed: %v", err)
	}
	if a.page != pageArchive {
		t.Fatalf("expected pageArchive, got %v", a.page)
	}

	if err := tt.Click("备份文件"); err != nil {
		t.Fatalf("click Backups nav failed: %v", err)
	}
	if a.page != pageBackup {
		t.Fatalf("expected pageBackup, got %v", a.page)
	}

	// 4. 切回设置页，验证 5 连击版本解锁开发者选项
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click Settings nav failed: %v", err)
	}
	for range 5 {
		a.handleVersionTap()
	}
	tt.Frame()
	if !a.devEnabled || !tt.HasText("开发者选项") {
		t.Fatalf("expected developer options to be unlocked after 5 taps")
	}

	// 5. 切换进入开发者选项页
	if err := tt.Click("开发者选项"); err != nil {
		t.Fatalf("click Developer Options nav failed: %v", err)
	}
	if a.page != pageDev || !tt.HasText("图标提取测试") {
		t.Fatalf("expected developer options view, page=%v", a.page)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	setSvc := settings.NewService(store)
	swSvc := software.NewService(setSvc)
	updSvc := updater.NewService()
	return New(setSvc, swSvc, updSvc, nil)
}

func TestSidebarNavIconsLeftAligned(t *testing.T) {
	a := newTestApp(t)
	a.homeNotConfigured = true

	tt := ui.NewTester(a.View, 1020, 680)
	rHome, ok1 := tt.Find("主页")
	rArch, ok2 := tt.Find("归档管理")
	rBak, ok3 := tt.Find("备份文件")
	rSet, ok4 := tt.Find("设置")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		t.Fatalf("expected all nav labels, found home=%v arch=%v bak=%v set=%v", ok1, ok2, ok3, ok4)
	}

	if math.Abs(float64(rHome.X-rArch.X)) > 1.0 || math.Abs(float64(rHome.X-rBak.X)) > 1.0 || math.Abs(float64(rHome.X-rSet.X)) > 1.0 {
		t.Fatalf("nav items should be left-aligned at same X, got home=%.1f arch=%.1f bak=%.1f set=%.1f", rHome.X, rArch.X, rBak.X, rSet.X)
	}
	if rHome.X > 60 {
		t.Fatalf("expected nav items near left edge of sidebar, got X=%.1f", rHome.X)
	}
}

func TestGridCardLongTitleDoesNotOverflow(t *testing.T) {
	a := newTestApp(t)
	a.useGrid = true
	a.homeNotConfigured = false
	a.homeLoading = false
	a.managed = []model.Software{
		{
			ID:     "long-1",
			Name:   "Office_Tool_with_runtime_v10.26.36.0_x64",
			Status: model.StatusUnknownInstall,
		},
	}

	tt := ui.NewTester(a.View, 1020, 680)
	rTitle, ok := tt.Find("Office_Tool_with_runtime_v10.26.36.0_x64")
	if !ok {
		t.Fatalf("expected grid card title in texts: %q", tt.Texts())
	}
	if rTitle.W > 150 {
		t.Fatalf("grid card title overflowed 172px card inner width (148px): got rect=%+v", rTitle)
	}
}

func TestArchiveActionIconButtonHoverContrast(t *testing.T) {
	a := newTestApp(t)
	a.page = pageArchive
	a.archiveNotConfigured = false
	a.archiveLoading = false
	a.archives = []model.Software{
		{
			Name:        "demo-app.zip",
			ArchivePath: `C:\Portable\~archives\demo-app.zip`,
		},
	}

	tt := ui.NewTester(a.View, 1020, 680)
	rRestore, ok := tt.Find("还原")
	if !ok {
		t.Fatalf("expected restore icon button in archive view")
	}

	cx, cy := rRestore.X+rRestore.W/2, rRestore.Y+rRestore.H/2
	tt.Move(cx, cy)
	tt.Frame()

	if got := tt.Cursor(); got != ui.CursorPointer {
		t.Fatalf("expected CursorPointer when hovering archive action button, got %v", got)
	}

	img := tt.Image()
	// 取按钮内边角像素（避开居中图标）与按钮左侧的已悬停行背景像素对比
	btnBg := img.RGBAAt(int(rRestore.X+2), int(rRestore.Y+2))
	rowBg := img.RGBAAt(int(rRestore.X-10), int(rRestore.Y+2))
	if btnBg == rowBg {
		t.Fatalf("expected visible hover highlight on archive icon button over hovered row, both got %+v", btnBg)
	}
}

func TestFontPickerDefaultPriorityUnavailableGrayedAndSystemFonts(t *testing.T) {
	a := newTestApp(t)
	a.applyInstalledFonts([]string{
		"HarmonyOS Sans SC",
		"Microsoft YaHei UI",
		"Segoe UI",
		"JetBrains Mono",
		"Wingdings",
		"@Microsoft YaHei",
	})

	// 未安装 PingFang 时默认应优先回退到鸿蒙字体
	if got := a.resolvedDefaultFont(); got != "HarmonyOS Sans SC" {
		t.Fatalf("expected resolvedDefaultFont = HarmonyOS Sans SC, got %q", got)
	}

	a.page = pageSettings
	tt := ui.NewTester(a.View, 1020, 760)

	if !tt.HasText("优先顺序：PingFang → 鸿蒙 → 微软雅黑（当前：HarmonyOS Sans SC）") {
		t.Fatalf("expected default font priority description, got texts: %q", tt.Texts())
	}

	// 向下滚动设置页直到「应用字体」下拉触发器进入可视区域（正向 DY 表示向下滚动）
	tt.Scroll(500, 300, 0, 450)
	tt.Frame()

	trigInfo, ok := tt.AccessByRole(ui.RolePopUpButton, "应用字体")
	if !ok {
		t.Fatalf("expected font picker popup button in accessibility tree")
	}
	tt.ClickAt(trigInfo.Bounds.X+trigInfo.Bounds.W/2, trigInfo.Bounds.Y+trigInfo.Bounds.H/2)
	tt.Frame()

	// 未安装的预设字体应置灰禁用，且点击后不生效
	pfInfo, ok := tt.AccessByRole(ui.RoleOption, "PingFang SC")
	if !ok {
		t.Fatalf("expected PingFang SC option in font popup")
	}
	if !pfInfo.Disabled {
		t.Fatalf("expected uninstalled PingFang SC option to be disabled")
	}
	tt.ClickAt(pfInfo.Bounds.X+pfInfo.Bounds.W/2, pfInfo.Bounds.Y+pfInfo.Bounds.H/2)
	tt.Frame()
	if a.fontFamily != "" {
		t.Fatalf("clicking disabled font option should not change fontFamily, got %q", a.fontFamily)
	}

	// 过滤纯符号与 @ 竖排字体，保留并展示更多系统已安装字体
	if tt.HasText("Wingdings") || tt.HasText("@Microsoft YaHei") {
		t.Fatalf("expected Wingdings and @-prefixed fonts to be filtered out, got texts: %q", tt.Texts())
	}
	if !tt.HasText("JetBrains Mono") {
		t.Fatalf("expected system font JetBrains Mono in popup, got texts: %q", tt.Texts())
	}

	// 向下滚动弹窗以保证位于底部的系统字体项在视口内可见并可点击
	tt.Scroll(pfInfo.Bounds.X+40, pfInfo.Bounds.Y+40, 0, 240)
	tt.Frame()

	jbInfo, ok := tt.AccessByRole(ui.RoleOption, "JetBrains Mono")
	if !ok {
		t.Fatalf("expected JetBrains Mono option in font popup")
	}
	tt.ClickAt(jbInfo.Bounds.X+jbInfo.Bounds.W/2, jbInfo.Bounds.Y+jbInfo.Bounds.H/2)
	tt.Frame()
	if a.fontFamily != "JetBrains Mono" || a.setFontFamily != "JetBrains Mono" {
		t.Fatalf("expected fontFamily to update to JetBrains Mono, got fontFamily=%q setFontFamily=%q", a.fontFamily, a.setFontFamily)
	}
}

func clickLabel(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("expected element %q in view, texts: %q", label, tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
}

func TestHomeFilterChipsAndEmptySearchState(t *testing.T) {
	a := newTestApp(t)
	exe1 := filepath.Clean(`c:\apps\7z\7zfm.exe`)
	dir1 := filepath.Clean(`c:\apps\7z`)
	exe2 := filepath.Clean(`c:\apps\everything\everything.exe`)
	dir2 := filepath.Clean(`c:\apps\everything`)
	a.exeCache[exe1] = &exeMeta{desc: "7-Zip"}
	a.altCache[dir1] = &altExecEntry{}
	a.exeCache[exe2] = &exeMeta{desc: "Everything"}
	a.altCache[dir2] = &altExecEntry{}

	a.managed = []model.Software{
		{ID: "m1", Name: "7-Zip", Status: model.StatusManaged, InstallExists: true, ExecutablePath: exe1, InstallPath: dir1},
		{ID: "m2", Name: "Everything", Status: model.StatusManaged, InstallExists: true, ExecutablePath: exe2, InstallPath: dir2},
	}
	a.unmanaged = []model.Software{
		{ID: "u1", Name: "PortableUnknown", Status: model.StatusUnknownInstall, InstallPath: filepath.Clean(`c:\apps\unknown`)},
	}

	tt := ui.NewTester(a.View, 1020, 720)

	// 1. 分段切换器：列表 -> 网格 -> 列表
	clickLabel(t, tt, "网格视图")
	if !a.useGrid || !tt.HasText("7-Zip") || !tt.HasText("PortableUnknown") {
		t.Fatalf("expected grid mode to stay visible with managed & unmanaged cards, useGrid=%v", a.useGrid)
	}
	clickLabel(t, tt, "列表视图")
	if a.useGrid {
		t.Fatalf("expected list mode after clicking list segment")
	}

	// 2. 状态筛选胶囊：仅未托管 / 仅已托管
	clickLabel(t, tt, "未托管")
	if !tt.HasText("PortableUnknown") || tt.HasText("7-Zip") {
		t.Fatalf("expected only unmanaged items under unmanaged filter, texts: %q", tt.Texts())
	}

	clickLabel(t, tt, "已托管")
	if !tt.HasText("7-Zip") || !tt.HasText("Everything") || tt.HasText("PortableUnknown") {
		t.Fatalf("expected only managed items under managed filter, texts: %q", tt.Texts())
	}

	// 3. 无匹配结果空状态 + 点击「清除筛选」还原
	a.searchQuery = "nonexistent-tool-xyz"
	tt.Frame()
	if !tt.HasText("没有匹配的项目") {
		t.Fatalf("expected empty search state message, got texts: %q", tt.Texts())
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("click clear filter failed: %v", err)
	}
	tt.Frame()
	if a.searchQuery != "" || a.homeFilter != homeFilterAll {
		t.Fatalf("expected filter & search reset, got search=%q filter=%q", a.searchQuery, a.homeFilter)
	}
	if !tt.HasText("7-Zip") || !tt.HasText("PortableUnknown") {
		t.Fatalf("expected all items restored after clearing filter, texts: %q", tt.Texts())
	}
}

func TestArchiveSearchAndLinkedFilter(t *testing.T) {
	a := newTestApp(t)
	a.page = pageArchive
	a.archives = []model.Software{
		{Name: "Alpha.zip", ArchivePath: `C:\Arch\Alpha.zip`},
		{Name: "Beta.7z", ArchivePath: `C:\Arch\Beta.7z`},
	}
	a.backups = []model.Software{
		{Name: "Alpha_bak.zip", ArchivePath: `C:\Bak\Alpha_bak.zip`, IsBackupArchive: true},
		{Name: "Linked_bak.zip", ArchivePath: `C:\Bak\Linked_bak.zip`, IsBackupArchive: true},
	}
	a.linkedPaths = map[string]bool{
		normalizePathKey(`C:\Arch\Alpha.zip`):     true,
		normalizePathKey(`C:\Bak\Linked_bak.zip`): true,
	}

	tt := ui.NewTester(a.View, 1020, 720)

	// 0. 归档管理页仅显示归档文件，不混入备份文件
	if !tt.HasText("Alpha.zip") || !tt.HasText("Beta.7z") || tt.HasText("Alpha_bak.zip") {
		t.Fatalf("expected only archives on pageArchive, texts: %q", tt.Texts())
	}

	// 1. 归档页「已关联」筛选
	clickLabel(t, tt, "已关联")
	if !tt.HasText("Alpha.zip") || tt.HasText("Beta.7z") {
		t.Fatalf("expected only linked archives, texts: %q", tt.Texts())
	}

	// 2. 归档页「未关联」筛选
	clickLabel(t, tt, "未关联")
	if !tt.HasText("Beta.7z") || tt.HasText("Alpha.zip") || tt.HasText("Alpha_bak.zip") {
		t.Fatalf("expected only unlinked archives, texts: %q", tt.Texts())
	}

	// 3. 归档页搜索无结果 -> 清除筛选还原
	a.archiveSearch = "no-such-archive"
	tt.Frame()
	if !tt.HasText("没有匹配的项目") {
		t.Fatalf("expected archive empty search prompt, got texts: %q", tt.Texts())
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("click clear archive filter failed: %v", err)
	}
	tt.Frame()
	if a.archiveSearch != "" || a.archiveFilter != archiveFilterAll {
		t.Fatalf("expected archive search & filter reset, got search=%q filter=%q", a.archiveSearch, a.archiveFilter)
	}
	if !tt.HasText("Alpha.zip") || !tt.HasText("Beta.7z") {
		t.Fatalf("expected all archives visible after clear, texts: %q", tt.Texts())
	}

	// 4. 切换到备份文件页，验证仅显示备份文件及独立搜索/筛选状态
	a.page = pageBackup
	tt.Frame()
	if !tt.HasText("Alpha_bak.zip") || !tt.HasText("Linked_bak.zip") || tt.HasText("Alpha.zip") {
		t.Fatalf("expected only backups on pageBackup, texts: %q", tt.Texts())
	}

	clickLabel(t, tt, "已关联")
	if !tt.HasText("Linked_bak.zip") || tt.HasText("Alpha_bak.zip") {
		t.Fatalf("expected only linked backups, texts: %q", tt.Texts())
	}

	clickLabel(t, tt, "未关联")
	if !tt.HasText("Alpha_bak.zip") || tt.HasText("Linked_bak.zip") {
		t.Fatalf("expected only unlinked backups, texts: %q", tt.Texts())
	}

	a.backupSearch = "no-such-backup"
	tt.Frame()
	if !tt.HasText("没有匹配的项目") {
		t.Fatalf("expected backup empty search prompt, got texts: %q", tt.Texts())
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("click clear backup filter failed: %v", err)
	}
	tt.Frame()
	if a.backupSearch != "" || a.backupFilter != archiveFilterAll {
		t.Fatalf("expected backup search & filter reset, got search=%q filter=%q", a.backupSearch, a.backupFilter)
	}
	if !tt.HasText("Alpha_bak.zip") || !tt.HasText("Linked_bak.zip") {
		t.Fatalf("expected all backups visible after clear, texts: %q", tt.Texts())
	}
}

func TestGlobalKeyboardShortcuts(t *testing.T) {
	a := newTestApp(t)
	a.managed = []model.Software{
		{ID: "m1", Name: "ToolOne", Status: model.StatusManaged, InstallExists: false, InstallPath: `c:\apps\toolone`},
	}
	a.backups = []model.Software{
		{Name: "ToolOne-20261123_131205.zip", ArchivePath: `C:\Bak\ToolOne-20261123_131205.zip`, IsBackupArchive: true},
	}

	tt := ui.NewTester(a.View, 1020, 720)

	// Ctrl+B 收起/展开侧边栏
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	if a.sidebarOpen {
		t.Fatalf("expected Ctrl+B to collapse sidebar")
	}
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	if !a.sidebarOpen {
		t.Fatalf("expected second Ctrl+B to expand sidebar")
	}

	// Ctrl+F 聚焦主页搜索框
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	if !tt.Focused("搜索软件名称或路径...") {
		t.Fatalf("expected Ctrl+F to focus home search box")
	}

	// 备份文件页 Ctrl+F 聚焦备份搜索框
	a.page = pageBackup
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Frame()
	if !tt.Focused("搜索备份文件名...") {
		t.Fatalf("expected Ctrl+F to focus backup search box")
	}

	// Ctrl+1 / Ctrl+2 / Ctrl+3 / Ctrl+4 切换页面
	tt.Key(ui.Cmd, ui.Key1)
	tt.Frame()
	if a.page != pageHome {
		t.Fatalf("expected Ctrl+1 to navigate to pageHome, got %v", a.page)
	}

	tt.Key(ui.Cmd, ui.Key2)
	tt.Frame()
	if a.page != pageArchive {
		t.Fatalf("expected Ctrl+2 to navigate to pageArchive, got %v", a.page)
	}

	tt.Key(ui.Cmd, ui.Key3)
	tt.Frame()
	if a.page != pageBackup {
		t.Fatalf("expected Ctrl+3 to navigate to pageBackup, got %v", a.page)
	}

	tt.Key(ui.Cmd, ui.Key4)
	tt.Frame()
	if a.page != pageSettings {
		t.Fatalf("expected Ctrl+4 to navigate to pageSettings, got %v", a.page)
	}
}

func TestSettingsAboutLinks(t *testing.T) {
	a := newTestApp(t)
	a.page = pageSettings

	tt := ui.NewTester(a.View, 1020, 760)
	// 向下滚动到底部的「关于」卡片区域（正向 DY 表示向下滚动）
	tt.Scroll(500, 300, 0, 1400)
	tt.Frame()

	if !tt.HasText("carrydock.devyi.com") || !tt.HasText("github.com/youzhiran/CarryDock") {
		t.Fatalf("expected website & GitHub display URLs in Settings About card, got texts: %q", tt.Texts())
	}

	clickLabel(t, tt, "官方网站")
	clickLabel(t, tt, "GitHub 仓库")

	want := []string{version.WebsiteURL, version.GitHubURL}
	if got := tt.OpenedURLs(); !slices.Equal(got, want) {
		t.Fatalf("expected opened URLs %v, got %v", want, got)
	}
}

func TestSoftwareContextMenuIcons(t *testing.T) {
	menuIcons := []struct {
		name string
		icon *mygo.MenuIcon
	}{
		{"menuIconPlay", menuIconPlay},
		{"menuIconFolder", menuIconFolder},
		{"menuIconEdit", menuIconEdit},
		{"menuIconRestore", menuIconRestore},
		{"menuIconPin", menuIconPin},
		{"menuIconArchive", menuIconArchive},
		{"menuIconRefresh", menuIconRefresh},
		{"menuIconTerminal", menuIconTerminal},
		{"menuIconTrash", menuIconTrash},
		{"menuIconSearch", menuIconSearch},
	}
	for _, tc := range menuIcons {
		if tc.icon == nil {
			t.Fatalf("%s should not be nil", tc.name)
		}
		pm := tc.icon.PlatformMenuIcon()
		if pm == nil || !pm.Template || len(pm.PNG) == 0 {
			t.Fatalf("%s expected non-empty template PlatformMenuIcon, got %+v", tc.name, pm)
		}
	}

	a := newTestApp(t)
	exe1 := filepath.Clean(`c:\apps\7z\7zfm.exe`)
	dir1 := filepath.Clean(`c:\apps\7z`)
	a.exeCache[exe1] = &exeMeta{desc: "7-Zip"}
	a.altCache[dir1] = &altExecEntry{}
	a.managed = []model.Software{
		{
			ID:             "m1",
			Name:           "7-Zip",
			Status:         model.StatusManaged,
			InstallExists:  true,
			ExecutablePath: exe1,
			InstallPath:    dir1,
			ArchivePath:    filepath.Clean(`c:\apps\~archives\7z.zip`),
		},
	}

	tt := ui.NewTester(a.View, 1020, 720)
	if err := tt.RightClick("7-Zip"); err != nil {
		t.Fatalf("right click software row failed: %v", err)
	}
	tt.Frame()

	wantItems := []string{
		"打开",
		"打开软件文件夹",
		"修改软件文件夹",
		"创建备份",
		"添加到开始菜单",
		"打开归档文件夹",
		"更改主程序",
		"删除",
	}
	for _, label := range wantItems {
		if !tt.HasText(label) {
			t.Fatalf("expected context menu item %q, texts: %q", label, tt.Texts())
		}
	}

	folderItem, ok := tt.AccessByRole(ui.RoleMenuItem, "打开软件文件夹")
	if !ok {
		t.Fatalf("expected 打开软件文件夹 menu item in accessibility tree")
	}
	delItem, ok := tt.AccessByRole(ui.RoleMenuItem, "删除")
	if !ok {
		t.Fatalf("expected 删除 menu item in accessibility tree")
	}

	img := tt.Image()
	scanHasInk := func(b ui.Rect, pred func(r, g, b uint8) bool) bool {
		minX := max(0, int(b.X))
		maxX := max(minX+1, int(b.X+min(b.W, 44)))
		minY := max(0, int(b.Y))
		maxY := max(minY+1, int(b.Y+b.H))
		for y := minY; y < maxY; y++ {
			for x := minX; x < maxX; x++ {
				px := img.RGBAAt(x, y)
				if pred(px.R, px.G, px.B) {
					return true
				}
			}
		}
		return false
	}

	// 验证普通菜单项左侧图标区域绘制了非背景前景像素
	folderBg := img.RGBAAt(max(0, int(folderItem.Bounds.X+4)), max(0, int(folderItem.Bounds.Y+4)))
	if !scanHasInk(folderItem.Bounds, func(r, g, b uint8) bool {
		return r != folderBg.R || g != folderBg.G || b != folderBg.B
	}) {
		t.Fatalf("expected rendered icon pixels inside 打开软件文件夹 item %+v", folderItem.Bounds)
	}

	// 验证危险操作「删除」左侧图标区域绘制了红色危险着色像素
	if !scanHasInk(delItem.Bounds, func(r, g, b uint8) bool {
		return int(r) > int(g)+20 && int(r) > int(b)+20
	}) {
		t.Fatalf("expected red danger trash icon pixels inside 删除 item %+v", delItem.Bounds)
	}
}

func TestArchiveAndBackupSoftwareIconAndCornerBadge(t *testing.T) {
	a := newTestApp(t)
	exeTool := filepath.Clean(`C:\Portable\ToolOne\ToolOne.exe`)
	archTool := filepath.Clean(`C:\Portable\~archives\ToolOne.zip`)
	bakLinked := filepath.Clean(`C:\Portable\~archives\backup\ToolOne-20261123_131205.zip`)
	bakOld := filepath.Clean(`C:\Portable\~archives\backup\ToolOne-20260101_000000.zip`)
	standaloneExe := filepath.Clean(`C:\Portable\~archives\Standalone.exe`)
	unlinkedNamedArch := filepath.Clean(`C:\Portable\~archives\ToolOne.7z`)
	unrelatedArch := filepath.Clean(`C:\Portable\~archives\RandomOther.zip`)

	managed := []model.Software{
		{
			ID:             "sw-1",
			Name:           "ToolOne",
			InstallPath:    filepath.Clean(`C:\Portable\ToolOne`),
			ExecutablePath: exeTool,
			ArchivePath:    archTool,
			BackupPath:     bakLinked,
			Status:         model.StatusManaged,
		},
	}
	linkedSw := map[string]model.Software{
		normalizePathKey(archTool):  managed[0],
		normalizePathKey(bakLinked): managed[0],
	}
	allowedExts := []string{"exe", "bat"}

	// 1. 单元测试 resolveArchiveExePath 覆盖全部匹配场景
	if got := resolveArchiveExePath(model.Software{Name: "ToolOne.zip", ArchivePath: archTool}, linkedSw, managed, allowedExts, false); got != exeTool {
		t.Fatalf("linked archive exe = %q, want %q", got, exeTool)
	}
	if got := resolveArchiveExePath(model.Software{Name: "ToolOne-20261123_131205.zip", ArchivePath: bakLinked, IsBackupArchive: true}, linkedSw, managed, allowedExts, false); got != exeTool {
		t.Fatalf("linked backup exe = %q, want %q", got, exeTool)
	}
	if got := resolveArchiveExePath(model.Software{Name: "ToolOne-20260101_000000.zip", ArchivePath: bakOld, IsBackupArchive: true}, linkedSw, managed, allowedExts, false); got != exeTool {
		t.Fatalf("unlinked older backup exe = %q, want %q", got, exeTool)
	}
	if got := resolveArchiveExePath(model.Software{Name: "Standalone.exe", ArchivePath: standaloneExe}, linkedSw, managed, allowedExts, false); got != standaloneExe {
		t.Fatalf("standalone exe archive = %q, want %q", got, standaloneExe)
	}
	if got := resolveArchiveExePath(model.Software{Name: "ToolOne.7z", ArchivePath: unlinkedNamedArch}, linkedSw, managed, allowedExts, false); got != exeTool {
		t.Fatalf("unlinked named archive exe = %q, want %q", got, exeTool)
	}
	if got := resolveArchiveExePath(model.Software{Name: "RandomOther.zip", ArchivePath: unrelatedArch}, linkedSw, managed, allowedExts, false); got != "" {
		t.Fatalf("unrelated archive exe = %q, want empty", got)
	}

	// 2. 构造纯绿色 32x32 软件图标位图并注入 exeCache，验证归档与备份列表均渲染软件图标及右下角角标
	imgSrc := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			imgSrc.SetNRGBA(x, y, color.NRGBA{R: 0, G: 204, B: 68, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, imgSrc); err != nil {
		t.Fatal(err)
	}
	bmp, err := ui.DecodeBitmap(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	a.exeCache[strings.ToLower(exeTool)] = &exeMeta{icon: bmp}
	a.managed = managed
	a.linkedSoftware = linkedSw
	a.linkedPaths = map[string]bool{
		normalizePathKey(archTool):  true,
		normalizePathKey(bakLinked): true,
	}
	a.archives = []model.Software{
		{Name: "ToolOne.zip", ArchivePath: archTool, ExecutablePath: exeTool},
	}
	a.backups = []model.Software{
		{Name: "ToolOne-20261123_131205.zip", ArchivePath: bakLinked, ExecutablePath: exeTool, IsBackupArchive: true},
	}

	hasPixelInRect := func(img *image.RGBA, x0, y0, x1, y1 int, pred func(r, g, b uint8) bool) bool {
		x0 = max(0, x0)
		y0 = max(0, y0)
		x1 = min(img.Bounds().Dx(), x1)
		y1 = min(img.Bounds().Dy(), y1)
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				px := img.RGBAAt(x, y)
				if pred(px.R, px.G, px.B) {
					return true
				}
			}
		}
		return false
	}

	// 归档管理页：验证绿色软件图标 + 灰蓝归档角标
	a.page = pageArchive
	tt := ui.NewTester(a.View, 1020, 720)
	rArchTitle, ok := tt.Find("ToolOne.zip")
	if !ok {
		t.Fatalf("expected ToolOne.zip in archive view, texts: %q", tt.Texts())
	}
	archImg := tt.Image()
	iconX0, iconY0 := int(rArchTitle.X)-62, int(rArchTitle.Y)-10
	iconX1, iconY1 := int(rArchTitle.X)-6, int(rArchTitle.Y)+42
	if !hasPixelInRect(archImg, iconX0, iconY0, iconX1, iconY1, func(r, g, b uint8) bool {
		return int(g) > int(r)+80 && int(g) > int(b)+40
	}) {
		t.Fatalf("expected green software icon pixels left of archive title %+v", rArchTitle)
	}
	// 右下角归档角标 (#64748b: b > r+20, g > r+8)
	if !hasPixelInRect(archImg, iconX0+20, iconY0+20, iconX1, iconY1, func(r, g, b uint8) bool {
		return int(b) > int(r)+20 && int(g) > int(r)+8
	}) {
		t.Fatalf("expected slate archive corner badge pixels at bottom-right of archive icon")
	}

	// 备份文件页：验证绿色软件图标 + 强调色备份角标
	a.page = pageBackup
	tt.Frame()
	rBakTitle, ok := tt.Find("ToolOne-20261123_131205.zip")
	if !ok {
		t.Fatalf("expected backup title in backup view, texts: %q", tt.Texts())
	}
	bakImg := tt.Image()
	bX0, bY0 := int(rBakTitle.X)-62, int(rBakTitle.Y)-10
	bX1, bY1 := int(rBakTitle.X)-6, int(rBakTitle.Y)+42
	if !hasPixelInRect(bakImg, bX0, bY0, bX1, bY1, func(r, g, b uint8) bool {
		return int(g) > int(r)+80 && int(g) > int(b)+40
	}) {
		t.Fatalf("expected green software icon pixels left of backup title %+v", rBakTitle)
	}
	// 右下角备份角标 (Accent 蓝色: b > r+40)
	if !hasPixelInRect(bakImg, bX0+20, bY0+20, bX1, bY1, func(r, g, b uint8) bool {
		return int(b) > int(r)+40
	}) {
		t.Fatalf("expected accent blue backup corner badge pixels at bottom-right of backup icon")
	}
}
