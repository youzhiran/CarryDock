package app

import (
	"os"
	"path/filepath"
	"testing"

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
