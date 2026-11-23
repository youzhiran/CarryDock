package main

import (
	_ "embed"
	"log"

	"carrydock/internal/app"
	"carrydock/internal/i18n"
	"carrydock/internal/logger"
	"carrydock/internal/settings"
	"carrydock/internal/software"
	"carrydock/internal/storage"
	"carrydock/internal/updater"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed assets/icon.png
var appIconPNG []byte

func main() {
	settingsSvc := settings.NewService(storage.Default())
	if err := logger.ConfigureWithLevel(
		settingsSvc.GetEnableFileLogging(),
		settingsSvc.GetLogFilePath(),
		settingsSvc.GetLogLevel(),
	); err != nil {
		log.Printf("初始化文件日志失败: %v", err)
	}
	defer logger.Close()

	logger.Info(
		"CarryDock (MyGo 原生 UI) 启动: logLevel=%s, fileLogging=%v, logPath=%s",
		settingsSvc.GetLogLevel(),
		settingsSvc.GetEnableFileLogging(),
		settingsSvc.GetLogFilePath(),
	)
	go updater.CleanupLegacyFilesForCurrentApp()

	softwareSvc := software.NewService(settingsSvc)
	updaterSvc := updater.NewService()
	state := app.New(settingsSvc, softwareSvc, updaterSvc, appIconPNG)

	mygo.App.WhenReady(func() {
		title := i18n.For(i18n.Normalize(settingsSvc.GetLanguage())).AppName
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         title,
			Width:         1020,
			Height:        680,
			MinWidth:      720,
			MinHeight:     520,
			StateKey:      "main",
			TitleBarStyle: mygo.TitleBarHidden,
			Vibrancy:      mygo.VibrancySidebar,
			Content:       ui.View(state.View),
		})
		if len(appIconPNG) > 0 {
			if err := win.SetIcon(appIconPNG); err != nil {
				logger.Warn("设置窗口任务栏图标失败: %v", err)
			}
		}
		state.AttachWindow(win)
	})

	if err := mygo.App.Run(); err != nil {
		logger.Error("应用运行退出: %v", err)
		log.Fatal(err)
	}
}
