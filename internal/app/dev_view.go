package app

import (
	"carrydock/internal/winutil"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) devView(c *ui.Context) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	ui.Column(c).Fill().Padding(20, 24).Gap(16).MaxWidth(680).Children(func() {
		ui.Text(c, tr.DevTitle).FontSize(22).Bold()
		ui.Text(c, tr.DevHint).FontSize(13).TextColor(t.TextMuted)

		// 图标提取测试卡片
		ui.Column(c).Padding(16).Gap(10).Radius(12).
			Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 4, 0, p.shadow).Children(func() {
			ui.Text(c, tr.DevIconTest).FontSize(14).Bold()
			ui.Text(c, tr.DevIconTestHint).FontSize(12).TextColor(t.TextMuted)
			ui.Row(c).Gap(8).Children(func() {
				if ui.PrimaryButton(c.Key("dev-icon-test"), tr.DevIconTest).Clicked() {
					a.startDevIconTest()
				}
			})
		})

		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c.Key("dev-hide"), tr.DevHide).Clicked() {
				a.devEnabled = false
				go func() {
					_ = a.settingsSvc.SaveDeveloperOptionsEnabled(false)
				}()
				a.navigateNow(pageSettings)
			}
			ui.Button(c.Key("dev-easter"), tr.DevHiddenButton).Opacity(0.25).OnClick(func() {
				a.showInfo(tr.DevHiddenFeature, tr.DevHiddenFeatureMsg)
			})
		})
	})
}

func (a *App) startDevIconTest() {
	win := a.win
	title := a.tr().DevIconTest

	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: win,
			Title:  title,
			Filters: []mygo.FileFilter{
				{Name: "Executables & Icons", Extensions: []string{"exe", "lnk", "ico"}},
			},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		target := paths[0]
		desc := winutil.GetFileDescription(target)
		var pngBytes []byte
		if b, err := winutil.ExtractIconPNG(target, 128); err == nil && len(b) > 0 {
			pngBytes = b
		} else if b, err := mygo.Shell.FileIcon(target, 128); err == nil && len(b) > 0 {
			pngBytes = b
		}
		var bmp *ui.Bitmap
		if len(pngBytes) > 0 {
			bmp, _ = ui.DecodeBitmap(pngBytes)
		}
		a.update(func() {
			if bmp == nil {
				a.showInfo(a.tr().DevExtractFailed, a.tr().DevExtractFailedMsg)
				return
			}
			a.devIconPath = target
			a.devIconDesc = desc
			a.devIconBmp = bmp
			a.devIconModalOpen = true
		})
	}()
}
