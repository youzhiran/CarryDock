package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func TestThemeModeAndThemeColorSettings(t *testing.T) {
	a := newTestApp(t)
	a.page = pageSettings

	var lastTheme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		a.View(c)
		lastTheme = c.Theme()
	}, 1020, 760)

	// 向下滚动设置页使「外观」卡片中的主题模式与主题颜色进入可视区域
	tt.Scroll(500, 300, 0, 560)
	tt.Frame()

	if !tt.HasText("主题模式") || !tt.HasText("主题颜色") {
		t.Fatalf("expected theme mode and theme color settings, got texts: %q", tt.Texts())
	}

	// 切换为深色模式并验证即时生效与持久化
	clickLabel(t, tt, "深色")
	if a.themeMode != settings.ThemeModeDark || a.settingsSvc.GetThemeMode() != settings.ThemeModeDark {
		t.Fatalf("expected dark theme mode saved, got state=%q stored=%q", a.themeMode, a.settingsSvc.GetThemeMode())
	}
	if lastTheme == nil || !lastTheme.Dark {
		t.Fatalf("expected rendered theme to be dark, got %+v", lastTheme)
	}

	// 切换为浅色模式并验证即时生效与持久化
	clickLabel(t, tt, "浅色")
	if a.themeMode != settings.ThemeModeLight || a.settingsSvc.GetThemeMode() != settings.ThemeModeLight {
		t.Fatalf("expected light theme mode saved, got state=%q stored=%q", a.themeMode, a.settingsSvc.GetThemeMode())
	}
	if lastTheme == nil || lastTheme.Dark {
		t.Fatalf("expected rendered theme to be light, got %+v", lastTheme)
	}

	// 选择预设主题色「紫罗兰」(#7c3aed) 并验证即时生效与持久化
	clickLabel(t, tt, "紫罗兰")
	if a.themeColor != "#7c3aed" || a.settingsSvc.GetThemeColor() != "#7c3aed" {
		t.Fatalf("expected theme color #7c3aed saved, got state=%q stored=%q", a.themeColor, a.settingsSvc.GetThemeColor())
	}
	wantAccent := ui.Hex("#7c3aed").SRGB()
	gotAccent := lastTheme.Accent.SRGB()
	if gotAccent.R != wantAccent.R || gotAccent.G != wantAccent.G || gotAccent.B != wantAccent.B {
		t.Fatalf("expected rendered theme Accent %+v, got %+v", wantAccent, gotAccent)
	}

	// 点击自定义按钮弹出 ColorPicker 并确认不会报错，再次点击关闭弹窗
	clickLabel(t, tt, "自定义")
	if !a.customColorOpen {
		t.Fatalf("expected custom color popover to open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.customColorOpen {
		t.Fatalf("expected custom color popover to close on Escape")
	}

	// 点击主题颜色区域的「跟随系统」按钮恢复默认
	var sysBtns []ui.AccessInfo
	for _, item := range tt.AccessAll() {
		if item.Name == "跟随系统" {
			sysBtns = append(sysBtns, item)
		}
	}
	if len(sysBtns) < 2 {
		t.Fatalf("expected at least 2 '跟随系统' buttons (mode + color), got %d", len(sysBtns))
	}
	colorSys := sysBtns[len(sysBtns)-1]
	tt.ClickAt(colorSys.Bounds.X+colorSys.Bounds.W/2, colorSys.Bounds.Y+colorSys.Bounds.H/2)
	tt.Frame()
	if a.themeColor != "" || a.settingsSvc.GetThemeColor() != "" {
		t.Fatalf("expected theme color reset to empty, got state=%q stored=%q", a.themeColor, a.settingsSvc.GetThemeColor())
	}
}

func TestHomeToolbarResponsiveOverflow(t *testing.T) {
	a := newTestApp(t)
	a.managed = []model.Software{
		{ID: "1", Name: "Alpha", Status: model.StatusManaged, InstallExists: true},
		{ID: "2", Name: "Beta", Status: model.StatusManaged, InstallExists: true},
	}

	// 1. 常规宽窗口（1020px）：全部工具栏按钮直接完整展示，不出现「...」更多按钮，且「打开安装目录」不重复
	wideTester := ui.NewTester(a.View, 1020, 720)
	for _, label := range []string{"列表视图", "网格视图", "调整排序", "刷新", "打开安装目录", "扫描当前目录软件"} {
		if _, ok := wideTester.Access(label); !ok {
			t.Fatalf("expected toolbar button %q visible in wide window", label)
		}
	}

	// 2. 窄窗口（560px）：部分工具栏按钮显示不下，自动收纳到「更多操作」菜单中
	narrowTester := ui.NewTester(a.View, 560, 720)
	if _, ok := narrowTester.Access("扫描当前目录软件"); ok {
		t.Fatalf("expected '扫描当前目录软件' collapsed into overflow menu in narrow window")
	}
	// 点击工具栏的「更多操作」按钮，验证被折叠的按钮出现在弹出菜单中
	var toolbarMore ui.AccessInfo
	foundMore := false
	for _, item := range narrowTester.AccessAll() {
		if item.Name == "更多操作" && item.Bounds.Y < 80 {
			toolbarMore = item
			foundMore = true
			break
		}
	}
	if !foundMore {
		t.Fatalf("expected toolbar overflow button '更多操作' in narrow window")
	}
	narrowTester.ClickAt(toolbarMore.Bounds.X+toolbarMore.Bounds.W/2, toolbarMore.Bounds.Y+toolbarMore.Bounds.H/2)
	narrowTester.Frame()
	if !narrowTester.HasText("扫描当前目录软件") {
		t.Fatalf("expected overflow menu to contain '扫描当前目录软件', got texts: %q", narrowTester.Texts())
	}
}

func TestSettingsLogLevelStrategySwitch(t *testing.T) {
	a := newTestApp(t)
	a.page = pageSettings

	tt := ui.NewTester(a.View, 1020, 760)
	// 向下滚动到「日志设置」卡片区域
	tt.Scroll(500, 300, 0, 1050)
	tt.Frame()

	if !tt.HasText("日志输出级别") || !tt.HasText("常规 (INFO)") {
		t.Fatalf("expected log level setting in Settings view, got texts: %q", tt.Texts())
	}

	// 切换到「调试 (DEBUG)」策略
	clickLabel(t, tt, "调试 (DEBUG)")
	if a.setLogLevel != settings.LogLevelDebug || a.settingsSvc.GetLogLevel() != settings.LogLevelDebug {
		t.Fatalf("expected log level debug, got state=%q stored=%q", a.setLogLevel, a.settingsSvc.GetLogLevel())
	}

	// 切换到「仅错误 (ERROR)」策略
	clickLabel(t, tt, "仅错误 (ERROR)")
	if a.setLogLevel != settings.LogLevelError || a.settingsSvc.GetLogLevel() != settings.LogLevelError {
		t.Fatalf("expected log level error, got state=%q stored=%q", a.setLogLevel, a.settingsSvc.GetLogLevel())
	}
}

func TestSoftwareNativeVersionDisplaySearchAndModals(t *testing.T) {
	// 1. 验证 formatVersionBadge 格式化与超长截断
	if got := formatVersionBadge(""); got != "" {
		t.Fatalf("formatVersionBadge(\"\") = %q, want empty", got)
	}
	if got := formatVersionBadge("24.08"); got != "v24.08" {
		t.Fatalf("formatVersionBadge(\"24.08\") = %q, want v24.08", got)
	}
	if got := formatVersionBadge("v1.2.3"); got != "v1.2.3" {
		t.Fatalf("formatVersionBadge(\"v1.2.3\") = %q, want v1.2.3", got)
	}
	longVer := "1.2.3.45678901234567890123456"
	gotLong := formatVersionBadge(longVer)
	if len([]rune(gotLong)) != 24 || !strings.HasSuffix(gotLong, "…") {
		t.Fatalf("formatVersionBadge(long) = %q (runes=%d), want 24 runes ending with …", gotLong, len([]rune(gotLong)))
	}

	a := newTestApp(t)
	exe7z := filepath.Clean(`c:\apps\7z\7zfm.exe`)
	dir7z := filepath.Clean(`c:\apps\7z`)
	exeMissing := filepath.Clean(`c:\apps\missing\app.exe`)
	dirMissing := filepath.Clean(`c:\apps\missing`)
	exeNoVer := filepath.Clean(`c:\apps\nover\tool.exe`)
	dirNoVer := filepath.Clean(`c:\apps\nover`)
	exeIncoming := filepath.Clean(`c:\downloads\7zfm_new.exe`)

	a.exeCache[strings.ToLower(exe7z)] = &exeMeta{desc: "7-Zip File Manager", version: "24.08"}
	a.altCache[strings.ToLower(dir7z)] = &altExecEntry{paths: []string{exe7z}}
	a.exeCache[strings.ToLower(exeMissing)] = &exeMeta{desc: "MissingApp", version: "9.9.9"}
	a.exeCache[strings.ToLower(exeNoVer)] = &exeMeta{desc: "NoVerTool", version: ""}
	a.altCache[strings.ToLower(dirNoVer)] = &altExecEntry{paths: []string{exeNoVer}}
	a.exeCache[strings.ToLower(exeIncoming)] = &exeMeta{desc: "7-Zip File Manager", version: "24.09"}

	a.managed = []model.Software{
		{ID: "m1", Name: "7-Zip", Status: model.StatusManaged, InstallExists: true, ExecutablePath: exe7z, InstallPath: dir7z},
		{ID: "m2", Name: "MissingApp", Status: model.StatusManaged, InstallExists: false, ExecutablePath: exeMissing, InstallPath: dirMissing},
		{ID: "m3", Name: "NoVerTool", Status: model.StatusManaged, InstallExists: true, ExecutablePath: exeNoVer, InstallPath: dirNoVer},
	}

	tt := ui.NewTester(a.View, 1020, 720)

	// 2. 列表视图展示 v24.08，且目录缺失项不展示版本号
	if !tt.HasText("v24.08") {
		t.Fatalf("expected version badge v24.08 in list view, texts: %q", tt.Texts())
	}
	if tt.HasText("v9.9.9") {
		t.Fatalf("expected missing install directory not to show version v9.9.9, texts: %q", tt.Texts())
	}

	// 3. 网格视图正确渲染版本号且不超出卡片边界
	clickLabel(t, tt, "网格视图")
	rVer, ok := tt.Find("v24.08")
	if !ok {
		t.Fatalf("expected version v24.08 in grid view, texts: %q", tt.Texts())
	}
	if rVer.W > 152 {
		t.Fatalf("grid card version overflowed card inner width: %+v", rVer)
	}

	// 4. 支持按版本号搜索（不带 v 与带 v）
	a.searchQuery = "24.08"
	tt.Frame()
	if !tt.HasText("7-Zip File Manager") || tt.HasText("NoVerTool") {
		t.Fatalf("expected search by '24.08' to match only 7-Zip File Manager, texts: %q", tt.Texts())
	}
	a.searchQuery = "v24.08"
	tt.Frame()
	if !tt.HasText("7-Zip File Manager") || tt.HasText("NoVerTool") {
		t.Fatalf("expected search by 'v24.08' to match only 7-Zip File Manager, texts: %q", tt.Texts())
	}
	a.searchQuery = ""
	tt.Frame()

	// 5. 重复覆盖弹窗：归档更新显示当前软件版本；可执行文件更新显示版本对比与相同版本提示
	a.dupModalOpen = true
	a.dupInfo = &software.DuplicateInfo{
		SourceType:              software.SourceArchive,
		SourcePath:              `c:\downloads\7z2409-x64.zip`,
		IntendedName:            "7-Zip",
		ExistingManagedSoftware: &a.managed[0],
	}
	tt.Frame()
	if !tt.HasText("当前软件版本：v24.08") {
		t.Fatalf("expected current version in archive duplicate modal, texts: %q", tt.Texts())
	}

	a.dupInfo = &software.DuplicateInfo{
		SourceType:              software.SourceExecutable,
		SourcePath:              exeIncoming,
		IntendedName:            "7-Zip",
		ExistingManagedSoftware: &a.managed[0],
	}
	tt.Frame()
	if !tt.HasText("版本对比：v24.08 → v24.09") {
		t.Fatalf("expected version comparison in executable duplicate modal, texts: %q", tt.Texts())
	}

	a.exeCache[strings.ToLower(exeIncoming)].version = "24.08"
	tt.Frame()
	if !tt.HasText("版本对比：v24.08 → v24.08（版本相同）") {
		t.Fatalf("expected same version suffix in duplicate modal, texts: %q", tt.Texts())
	}
	a.dupModalOpen = false

	// 6. 选择主程序弹窗渲染候选程序版本号
	a.selectExeOpen = true
	a.selectExeMode = "change"
	a.selectExePaths = []string{exe7z, exeNoVer}
	a.selectExeChoice = exe7z
	tt.Frame()
	if !tt.HasText("v24.08") {
		t.Fatalf("expected candidate version badge in select executable modal, texts: %q", tt.Texts())
	}
	a.selectExeOpen = false

	// 7. 开发者选项图标提取测试弹窗渲染文件版本
	imgSrc := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, imgSrc); err != nil {
		t.Fatal(err)
	}
	bmp, err := ui.DecodeBitmap(pngBuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	a.devIconPath = exe7z
	a.devIconDesc = "7-Zip File Manager"
	a.devIconVersion = "24.08"
	a.devIconBmp = bmp
	a.devIconModalOpen = true
	tt.Frame()
	if !tt.HasText("文件版本：24.08") {
		t.Fatalf("expected file version in dev icon test modal, texts: %q", tt.Texts())
	}
	a.devIconModalOpen = false

	// 8. 发现新版本替换旧版本弹窗与降级提示渲染
	a.dupModalOpen = true
	a.dupRename = "Everything-1.4.1.1026"
	a.dupInfo = &software.DuplicateInfo{
		SourceType:              software.SourceArchive,
		SourcePath:              `c:\downloads\Everything-1.4.1.1026.x64.zip`,
		IntendedName:            "Everything-1.4.1.1026",
		TargetInstallPath:       filepath.Clean(`c:\apps\Everything-1.4.1.1026`),
		InstallDirExists:        true,
		ExistingManagedSoftware: &a.managed[0],
		IsVersionUpdate:         true,
		TargetPathCollides:      false,
		CurrentVersion:          "1.4.1.1024",
		IncomingVersion:         "1.4.1.1026",
	}
	tt.Frame()
	for _, want := range []string{
		"发现新版本",
		"版本对比：v1.4.1.1024 → v1.4.1.1026",
		"替换旧版本",
		"备份并替换",
		"保留旧版并添加",
	} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in version update modal, texts: %q", want, tt.Texts())
		}
	}

	a.dupInfo.IncomingVersion = "1.4.1.1020"
	tt.Frame()
	if !tt.HasText("检测到已安装软件") || !tt.HasText("版本对比：v1.4.1.1024 → v1.4.1.1020（低于当前版本）") {
		t.Fatalf("expected downgrade prompt in replace modal, texts: %q", tt.Texts())
	}
}

func TestStartupUpdateCheckBehavior(t *testing.T) {
	newAppWithMockServer := func(t *testing.T, handler http.HandlerFunc) (*App, *settings.Service) {
		t.Helper()
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)

		store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
		setSvc := settings.NewService(store)
		swSvc := software.NewService(setSvc)
		updSvc := updater.NewServiceWithEndpoint(srv.URL, srv.Client())
		return New(setSvc, swSvc, updSvc, nil), setSvc
	}

	waitCheckDone := func(t *testing.T, ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for startup update check")
		}
	}

	// Case A: 启用且存在新版本，无弹窗阻塞时自动打开更新弹窗并展示侧边栏角标
	a, _ := newAppWithMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v99.0.0+9999",
			"body": "启动自动检查更新内容",
			"assets": [{"name": "CarryDock-windows.zip", "browser_download_url": "https://example.com/CarryDock.zip"}]
		}`))
	})
	doneA := make(chan struct{}, 2)
	a.onUpdateCheckDone = func() { doneA <- struct{}{} }
	a.checkUpdateOnStartupIfEnabled()
	waitCheckDone(t, doneA)

	if a.availableUpdate == nil || a.availableUpdate.Version != "v99.0.0+9999" {
		t.Fatalf("expected availableUpdate v99.0.0+9999, got %+v", a.availableUpdate)
	}
	if !a.updateModalOpen || a.updateInfo == nil {
		t.Fatalf("expected updateModalOpen to be true, open=%v info=%+v", a.updateModalOpen, a.updateInfo)
	}
	tt := ui.NewTester(a.View, 1020, 680)
	if !tt.HasText("发现新版本：v99.0.0+9999") || !tt.HasText("启动自动检查更新内容") {
		t.Fatalf("expected update modal text, got %q", tt.Texts())
	}

	// 关闭弹窗后侧边栏仍保留更新徽标，点击可重新打开更新弹窗
	clickLabel(t, tt, "取消")
	if a.updateModalOpen {
		t.Fatalf("expected update modal closed after clicking cancel")
	}
	if !tt.HasText("有新版本") {
		t.Fatalf("expected sidebar update badge '有新版本', texts: %q", tt.Texts())
	}
	clickLabel(t, tt, "有新版本")
	if !a.updateModalOpen {
		t.Fatalf("expected clicking sidebar badge to reopen update modal")
	}

	// Case B: 已有弹窗打开时仅发 Toast 提示，不打断当前弹窗
	b, _ := newAppWithMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v99.1.0+10000",
			"body": "后台更新",
			"assets": [{"name": "CarryDock-windows.zip", "browser_download_url": "https://example.com/CarryDock.zip"}]
		}`))
	})
	b.infoModalOpen = true
	doneB := make(chan struct{}, 1)
	b.onUpdateCheckDone = func() { doneB <- struct{}{} }
	b.checkUpdateOnStartupIfEnabled()
	waitCheckDone(t, doneB)
	if b.updateModalOpen {
		t.Fatalf("expected update modal NOT to steal focus when another modal is open")
	}
	if b.availableUpdate == nil || len(b.pendingToasts) == 0 {
		t.Fatalf("expected availableUpdate and pending toast, available=%+v toasts=%d", b.availableUpdate, len(b.pendingToasts))
	}

	// Case C: 已是最新版本时静默，不弹窗也不提示 Toast
	cApp, _ := newAppWithMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.0.1+0",
			"body": "旧版本",
			"assets": [{"name": "CarryDock-windows.zip", "browser_download_url": "https://example.com/CarryDock.zip"}]
		}`))
	})
	doneC := make(chan struct{}, 1)
	cApp.onUpdateCheckDone = func() { doneC <- struct{}{} }
	cApp.checkUpdateOnStartupIfEnabled()
	waitCheckDone(t, doneC)
	if cApp.updateModalOpen || cApp.availableUpdate != nil || len(cApp.pendingToasts) != 0 {
		t.Fatalf("expected silent no-op when up to date, modal=%v avail=%+v toasts=%d", cApp.updateModalOpen, cApp.availableUpdate, len(cApp.pendingToasts))
	}

	// Case D: 网络/HTTP 错误时静默，不弹窗也不弹出错误 Toast
	dApp, _ := newAppWithMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	doneD := make(chan struct{}, 1)
	dApp.onUpdateCheckDone = func() { doneD <- struct{}{} }
	dApp.checkUpdateOnStartupIfEnabled()
	waitCheckDone(t, doneD)
	if dApp.updateModalOpen || len(dApp.pendingToasts) != 0 {
		t.Fatalf("expected silent failure on startup check, modal=%v toasts=%d", dApp.updateModalOpen, len(dApp.pendingToasts))
	}

	// Case E: 关闭启动检查更新时不发起请求
	var disabledHits atomic.Int32
	storeE := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	setSvcE := settings.NewService(storeE)
	if err := setSvcE.SaveCheckUpdateOnStartup(false); err != nil {
		t.Fatal(err)
	}
	srvE := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		disabledHits.Add(1)
	}))
	defer srvE.Close()
	eApp := New(setSvcE, software.NewService(setSvcE), updater.NewServiceWithEndpoint(srvE.URL, srvE.Client()), nil)
	eApp.checkUpdateOnStartupIfEnabled()
	time.Sleep(50 * time.Millisecond)
	if disabledHits.Load() != 0 {
		t.Fatalf("expected 0 HTTP requests when startup update check disabled, got %d", disabledHits.Load())
	}

	// Case F: 重复调用 checkUpdateOnStartupIfEnabled 仅执行一次
	var idemHits atomic.Int32
	fApp, _ := newAppWithMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		idemHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.0.1+0",
			"body": "",
			"assets": [{"name": "CarryDock-windows.zip", "browser_download_url": "https://example.com/CarryDock.zip"}]
		}`))
	})
	doneF := make(chan struct{}, 2)
	fApp.onUpdateCheckDone = func() { doneF <- struct{}{} }
	fApp.checkUpdateOnStartupIfEnabled()
	fApp.checkUpdateOnStartupIfEnabled()
	waitCheckDone(t, doneF)
	time.Sleep(50 * time.Millisecond)
	if idemHits.Load() != 1 {
		t.Fatalf("expected exactly 1 HTTP request across repeated startup calls, got %d", idemHits.Load())
	}
}

func TestSettingsCheckUpdateOnStartupToggle(t *testing.T) {
	a := newTestApp(t)
	a.page = pageSettings

	if !a.setCheckUpdateOnStartup {
		t.Fatalf("expected startup update check enabled by default")
	}

	tt := ui.NewTester(a.View, 1020, 760)
	// 向下滚动设置页直到「关于」卡片中的开关进入可视区域
	tt.Scroll(500, 300, 0, 1400)
	tt.Frame()

	if !tt.HasText("启动时检查更新") {
		t.Fatalf("expected '启动时检查更新' setting in Settings view, texts: %q", tt.Texts())
	}

	sw, ok := tt.AccessByRole(ui.RoleSwitch, "启动时检查更新")
	if !ok {
		t.Fatalf("expected RoleSwitch '启动时检查更新' in accessibility tree")
	}

	waitPersisted := func(want bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if a.settingsSvc.GetCheckUpdateOnStartup() == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for persisted CheckUpdateOnStartup=%v, got %v", want, a.settingsSvc.GetCheckUpdateOnStartup())
	}

	tt.ClickAt(sw.Bounds.X+sw.Bounds.W/2, sw.Bounds.Y+sw.Bounds.H/2)
	tt.Frame()
	if a.setCheckUpdateOnStartup {
		t.Fatalf("expected setCheckUpdateOnStartup to toggle off")
	}
	waitPersisted(false)

	tt.ClickAt(sw.Bounds.X+sw.Bounds.W/2, sw.Bounds.Y+sw.Bounds.H/2)
	tt.Frame()
	if !a.setCheckUpdateOnStartup {
		t.Fatalf("expected setCheckUpdateOnStartup to toggle back on")
	}
	waitPersisted(true)
}
