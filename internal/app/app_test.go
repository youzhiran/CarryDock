package app

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/software"
	"carrydock/internal/storage"
	"carrydock/internal/updater"
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
	rSet, ok3 := tt.Find("设置")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("expected all nav labels, found home=%v arch=%v set=%v", ok1, ok2, ok3)
	}

	if math.Abs(float64(rHome.X-rArch.X)) > 1.0 || math.Abs(float64(rHome.X-rSet.X)) > 1.0 {
		t.Fatalf("nav items should be left-aligned at same X, got home=%.1f arch=%.1f set=%.1f", rHome.X, rArch.X, rSet.X)
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
