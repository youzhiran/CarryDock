package app

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// appPalette 封装浅色/深色模式下更具层次感的语义配色，并自适应系统强调色。
type appPalette struct {
	canvas       ui.Color // 主工作区画布底色
	sidebarBg    ui.Color // 无 Mica 时的侧栏与标题栏底色
	card         ui.Color // 提升的卡片与列表行底色
	cardHover    ui.Color // 卡片或行悬停底色
	subtleBg     ui.Color // 次级控件、分段底槽与图标底衬
	cardBorder   ui.Color // 柔和卡片边框
	borderStrong ui.Color // 强调边框
	shadow       ui.Color // 卡片常驻柔和阴影
	shadowHover  ui.Color // 卡片悬停阴影
	accentTint   ui.Color // 强调色浅底
	successTint  ui.Color // 成功态浅底
	warningTint  ui.Color // 警告态浅底
	dangerTint   ui.Color // 危险态浅底
}

// paletteOf 根据当前主题构建分层配色方案。
func paletteOf(t *ui.Theme) appPalette {
	if t.Dark {
		return appPalette{
			canvas:       ui.Hex("#141519"),
			sidebarBg:    ui.Hex("#1b1d22"),
			card:         ui.Hex("#1e2026"),
			cardHover:    ui.Hex("#262932"),
			subtleBg:     ui.Hex("#252830"),
			cardBorder:   ui.RGBA(255, 255, 255, 0.08),
			borderStrong: ui.RGBA(255, 255, 255, 0.16),
			shadow:       ui.RGBA(0, 0, 0, 0.22),
			shadowHover:  ui.RGBA(0, 0, 0, 0.36),
			accentTint:   t.Accent.Alpha(0.18),
			successTint:  t.Success.Alpha(0.18),
			warningTint:  t.Warning.Alpha(0.18),
			dangerTint:   t.Danger.Alpha(0.18),
		}
	}
	return appPalette{
		canvas:       ui.Hex("#f5f7fa"),
		sidebarBg:    ui.Hex("#eef1f5"),
		card:         ui.Hex("#ffffff"),
		cardHover:    ui.Hex("#fafbfd"),
		subtleBg:     ui.Hex("#eef1f5"),
		cardBorder:   ui.RGBA(15, 23, 42, 0.08),
		borderStrong: ui.RGBA(15, 23, 42, 0.16),
		shadow:       ui.RGBA(15, 23, 42, 0.04),
		shadowHover:  ui.RGBA(15, 23, 42, 0.10),
		accentTint:   t.Accent.Alpha(0.11),
		successTint:  t.Success.Alpha(0.12),
		warningTint:  t.Warning.Alpha(0.12),
		dangerTint:   t.Danger.Alpha(0.12),
	}
}

// applyHoverIconStyle 为无背景图标按钮应用悬停/按下高亮与鼠标手型，并返回当前图标前景色。
func applyHoverIconStyle(btn ui.Element, t *ui.Theme, disabled, danger bool, accent *ui.Color) ui.Color {
	if !disabled {
		btn.Cursor(ui.CursorPointer)
	}
	hovered := !disabled && btn.Hovered()
	pressed := !disabled && btn.Pressed()
	// 使用带透明度的前景色叠加，确保在已悬停行上依然具有清晰可见的悬停反馈
	switch {
	case pressed && danger:
		btn.Background(t.Danger.Alpha(0.24))
	case hovered && danger:
		btn.Background(t.Danger.Alpha(0.16))
	case pressed && accent != nil:
		btn.Background(accent.Alpha(0.24))
	case hovered && accent != nil:
		btn.Background(accent.Alpha(0.16))
	case pressed:
		btn.Background(t.Text.Alpha(0.16))
	case hovered:
		btn.Background(t.Text.Alpha(0.10))
	}
	if danger {
		return t.Danger
	}
	if accent != nil {
		return *accent
	}
	if hovered || pressed {
		return t.Text
	}
	return t.TextMuted
}

// iconButton 渲染紧凑的无背景图标按钮（悬停/按下高亮，支持禁用与危险色）。
func iconButton(c *ui.Context, key string, ico *ui.SVG, tooltip string, disabled, danger bool, accent *ui.Color) ui.Element {
	t := c.Theme()
	btn := ui.ButtonBase(c.Key(key)).Size(30, 30).Radius(6).Center().
		Disabled(disabled).Label(tooltip).Tooltip(tooltip)
	color := applyHoverIconStyle(btn, t, disabled, danger, accent)
	btn.Children(func() {
		ui.Icon(c, ico).FontSize(15).TextColor(color)
	})
	return btn
}

// pillBadge 渲染语义胶囊标签（如状态标记、关联状态、未保存提示）。
func pillBadge(c *ui.Context, label string, bg, fg ui.Color) {
	pillBadgeWithTip(c, label, bg, fg, "")
}

// pillBadgeWithTip 渲染带悬停提示的语义胶囊标签。
func pillBadgeWithTip(c *ui.Context, label string, bg, fg ui.Color, tooltip string) {
	badge := ui.Row(c).Padding(2, 8).Radius(999).Background(bg).Center().Shrink(0)
	if tooltip != "" {
		badge.Tooltip(tooltip)
	}
	badge.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(fg).SingleLine()
	})
}

// versionPillBadge 渲染紧凑的软件原生版本胶囊标签，支持悬停显示完整版本号。
func versionPillBadge(c *ui.Context, label, tooltip string) {
	t := c.Theme()
	p := paletteOf(t)
	badge := ui.Row(c).Padding(1, 7).Radius(999).Background(p.subtleBg).
		Border(1, p.cardBorder).Center().Shrink(0)
	if tooltip != "" {
		badge.Tooltip(tooltip)
	}
	badge.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).SingleLine()
	})
}

// filterChip 渲染筛选栏胶囊按钮（支持激活态高亮与可选计数徽章）。
func filterChip(c *ui.Context, key, label string, count int, active bool) ui.Element {
	t := c.Theme()
	p := paletteOf(t)
	btn := ui.ButtonBase(c.Key(key)).Height(28).Padding(0, 10).Radius(999).
		Center().Gap(6).Cursor(ui.CursorPointer).Label(label)
	switch {
	case active:
		btn.Background(p.accentTint).Border(1, t.Accent.Alpha(0.42))
	case btn.Hovered():
		btn.Background(t.SurfaceHover).Border(1, p.cardBorder)
	default:
		btn.Background(p.card).Border(1, p.cardBorder)
	}
	btn.Children(func() {
		fg := t.TextMuted
		if active {
			fg = t.Accent
		} else if btn.Hovered() {
			fg = t.Text
		}
		txt := ui.Text(c, label).FontSize(12).TextColor(fg).SingleLine()
		if active {
			txt.Bold()
		}
		if count >= 0 {
			badgeBg := p.subtleBg
			badgeFg := t.TextMuted
			if active {
				badgeBg = t.Accent.Alpha(0.18)
				badgeFg = t.Accent
			}
			ui.Row(c).Padding(0, 6).Radius(999).Background(badgeBg).Center().Children(func() {
				ui.Text(c, strconv.Itoa(count)).FontSize(11).TextColor(badgeFg).SingleLine()
			})
		}
	})
	return btn
}

// sectionTitleRow 渲染带图标和数量徽章的分组标题行。
func sectionTitleRow(c *ui.Context, ico *ui.SVG, title string, badgeCount int) {
	t := c.Theme()
	p := paletteOf(t)
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		if ico != nil {
			ui.Icon(c, ico).FontSize(14).TextColor(t.TextMuted).Shrink(0)
		}
		ui.Text(c, title).FontSize(13).Bold().TextColor(t.Text)
		if badgeCount >= 0 {
			ui.Badge(c, strconv.Itoa(badgeCount)).Background(p.subtleBg).TextColor(t.TextMuted)
		}
	})
}

// emptySearchView 渲染搜索或筛选无匹配时的引导卡片与重置按钮，返回是否点击了重置按钮。
func emptySearchView(c *ui.Context, title, hint, resetLabel string) bool {
	t := c.Theme()
	p := paletteOf(t)
	var resetClicked bool
	ui.Column(c).Grow(1).Center().Padding(24).Children(func() {
		ui.Column(c).MaxWidth(420).Padding(28, 32).Gap(12).Radius(12).
			Background(p.card).Border(1, p.cardBorder).Shadow(0, 2, 10, 0, p.shadow).
			Center().Children(func() {
			ui.Box(c).Size(52, 52).Radius(14).Background(p.subtleBg).Center().Children(func() {
				ui.Icon(c, iconSearch).FontSize(24).TextColor(t.TextMuted)
			})
			ui.Text(c, title).FontSize(15).Bold().TextColor(t.Text).TextAlign(ui.Center)
			if hint != "" {
				ui.Text(c, hint).FontSize(12).TextColor(t.TextMuted).TextAlign(ui.Center)
			}
			if ui.Button(c, resetLabel).Clicked() {
				resetClicked = true
			}
		})
	})
	return resetClicked
}

// verticalSep 渲染工具栏轻量垂直分割线。
func verticalSep(c *ui.Context) {
	p := paletteOf(c.Theme())
	ui.Box(c).Size(1, 18).Background(p.borderStrong).Shrink(0)
}
