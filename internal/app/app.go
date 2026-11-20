package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"carrydock/internal/i18n"
	"carrydock/internal/logger"
	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/software"
	"carrydock/internal/updater"
	"carrydock/internal/winutil"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type pageID string

const (
	pageHome     pageID = "home"
	pageArchive  pageID = "archive"
	pageSettings pageID = "settings"
	pageDev      pageID = "dev"
)

type exeMeta struct {
	loading bool
	desc    string
	icon    *ui.Bitmap
}

type altExecEntry struct {
	loading bool
	paths   []string
}

// App 持有 CarryDock 原生 UI 的全部应用状态与核心服务引用。
type App struct {
	win         *mygo.Window
	appIcon     *ui.Bitmap
	settingsSvc *settings.Service
	softwareSvc *software.Service
	updaterSvc  *updater.Service

	// 全局外观与导航状态
	lang               i18n.Lang
	fontFamily         string
	systemFontsLoading bool
	systemFontsLoaded  bool
	systemFonts        []string          // 过滤并排序后的系统可用字体名列表
	installedFontMap   map[string]string // 小写字体名 -> 系统原始规范名（未过滤，用于准确检测可用性）
	devEnabled         bool
	page               pageID
	sidebarOpen        bool

	// 待发出的 Toast 队列（由后台任务安全入队，在 View 帧中派发）
	pendingToasts []ui.Toast

	// 主页状态
	managed           []model.Software
	unmanaged         []model.Software
	homeLoading       bool
	homeNotConfigured bool
	useGrid           bool
	reorderMode       bool
	searchQuery       string
	homeListState     ui.ListState
	homeScrollState   ui.ScrollState

	// 可执行文件元数据与备选程序异步缓存（按小写路径索引）
	exeCache map[string]*exeMeta
	altCache map[string]*altExecEntry

	// 归档管理状态
	archives             []model.Software
	backups              []model.Software
	linkedPaths          map[string]bool
	archiveLoading       bool
	archiveNotConfigured bool
	archiveScrollState   ui.ScrollState

	// 设置页表单状态
	setInstallPath       string
	setSavedInstallPath  string
	setArchivePath       string
	setSavedArchivePath  string
	setConfigFilePath    string
	setRemoveNested      bool
	setSavedRemoveNested bool
	setMaxDepth          int
	setSavedMaxDepth     int
	setExts              []string
	setSavedExts         []string
	setFontFamily        string
	setLanguage          string
	setSavedLanguage     string
	setFileLogging       bool
	setLogFilePath       string
	setScrollState       ui.ScrollState

	// 5 连击版本解锁开发者选项
	versionTapCount int
	firstVersionTap time.Time

	// 切换页面时未保存设置提醒
	unsavedModalOpen bool
	pendingNavPage   pageID

	// 开发者选项图标测试状态
	devIconModalOpen bool
	devIconPath      string
	devIconDesc      string
	devIconBmp       *ui.Bitmap

	// 全局进度与消息弹窗
	busyOpen       bool
	busyMsg        string
	infoModalOpen  bool
	infoModalTitle string
	infoModalBody  string

	// 选择可执行文件弹窗（添加 / 重新托管 / 更改主程序）
	selectExeOpen     bool
	selectExeMode     string // "add" | "rehost" | "change"
	selectExePending  *software.PendingAddition
	selectExeTargetID string
	selectExePaths    []string
	selectExeChoice   string

	// 重复冲突弹窗
	dupModalOpen          bool
	dupInfo               *software.DuplicateInfo
	dupRename             string
	dupFromArchiveManager bool

	// 删除确认弹窗
	deleteModalOpen    bool
	deleteTarget       model.Software
	deleteInstallDir   bool
	deleteArchive      bool
	deleteIsArchiveMgr bool
	deleteIsBackup     bool

	// 修改软件文件夹弹窗
	renameModalOpen bool
	renameTarget    model.Software
	renameInput     string

	// 批量扫描与统一关联弹窗
	scanConfirmOpen   bool
	scanCreateBackup  bool
	scanBackupDirText string
	scanProgressOpen  bool
	scanDone          int
	scanTotal         int
	scanCurrent       string
	assocModalOpen    bool
	assocSuggestions  []software.ArchiveAssociationSuggestion
	assocEnabled      map[string]bool
	assocSelected     map[string]string
	assocCreateBackup bool
	assocSummaryText  string

	// 归档管理：手动关联与创建备份弹窗
	linkModalOpen    bool
	linkTarget       model.Software
	linkIsBackup     bool
	linkManaged      []model.Software
	linkSelectedName string
	linkNameToID     map[string]string

	backupModalOpen    bool
	backupManaged      []model.Software
	backupSelectedName string
	backupNameToID     map[string]string

	// 设置页：扩展名编辑弹窗
	extModalOpen bool
	extDraft     map[string]bool
	extCustom    string
	extError     string

	// 更新与重装弹窗
	updateChecking       bool
	updateModalOpen      bool
	updateInfo           *updater.ReleaseInfo
	updateStatus         string
	updateProgress       float64
	updateDownloading    bool
	updateInstalling     bool
	updateFailed         bool
	updateCancel         context.CancelFunc
	reinstallConfirmOpen bool
}

// New 创建并初始化 CarryDock 应用状态。
func New(settingsSvc *settings.Service, softwareSvc *software.Service, updaterSvc *updater.Service, iconPNG []byte) *App {
	var appBmp *ui.Bitmap
	if len(iconPNG) > 0 {
		appBmp, _ = ui.DecodeBitmap(iconPNG)
	}
	a := &App{
		appIcon:          appBmp,
		settingsSvc:      settingsSvc,
		softwareSvc:      softwareSvc,
		updaterSvc:       updaterSvc,
		page:             pageHome,
		sidebarOpen:      true,
		installedFontMap: make(map[string]string),
		exeCache:         make(map[string]*exeMeta),
		altCache:         make(map[string]*altExecEntry),
		linkedPaths:      make(map[string]bool),
		extDraft:         make(map[string]bool),
	}
	a.loadSettingsSnapshot()
	return a
}

// AttachWindow 绑定主窗口并在启动时加载主页数据与系统字体。
func (a *App) AttachWindow(win *mygo.Window) {
	a.win = win
	a.syncWindowTitle()
	a.loadSystemFontsAsync()
	a.reloadHome()
}

func (a *App) loadSystemFontsAsync() {
	if a.systemFontsLoaded || a.systemFontsLoading || a.win == nil {
		return
	}
	a.systemFontsLoading = true
	go func() {
		families := ui.FontFamilies()
		names := make([]string, 0, len(families))
		for _, f := range families {
			names = append(names, f.Name)
		}
		a.update(func() {
			a.applyInstalledFonts(names)
		})
	}()
}

// applyInstalledFonts 写入已安装系统字体索引并生成过滤后的候选列表。
func (a *App) applyInstalledFonts(rawNames []string) {
	installed := make(map[string]string, len(rawNames))
	for _, raw := range rawNames {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := installed[key]; !exists {
			installed[key] = name
		}
	}
	a.installedFontMap = installed
	a.systemFonts = settings.FilterSystemFonts(rawNames)
	a.systemFontsLoaded = true
	a.systemFontsLoading = false

	// 若已保存字体与系统规范名存在大小写差异，在内存中规整为规范名以便选择器精确匹配
	if a.setFontFamily != "" {
		if canonical, ok := a.installedFontMap[strings.ToLower(strings.TrimSpace(a.setFontFamily))]; ok {
			a.setFontFamily = canonical
			a.fontFamily = canonical
		}
	}
}

func (a *App) isFontInstalled(name string) bool {
	if !a.systemFontsLoaded {
		return true
	}
	_, ok := a.installedFontMap[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func (a *App) resolvedDefaultFont() string {
	if !a.systemFontsLoaded {
		return ""
	}
	for _, cand := range settings.DefaultFontPriority {
		if canonical, ok := a.installedFontMap[strings.ToLower(cand)]; ok {
			return canonical
		}
	}
	return ""
}

func (a *App) tr() i18n.Strings {
	return i18n.For(a.lang)
}

func (a *App) syncWindowTitle() {
	if a.win != nil {
		a.win.SetTitle(a.tr().AppName)
	}
}

// update 安全地在主线程更新状态并请求重绘；在无窗口单元测试环境下直接执行。
func (a *App) update(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
		return
	}
	fn()
}

func (a *App) toast(title, desc string) {
	a.pendingToasts = append(a.pendingToasts, ui.Toast{
		Title:       title,
		Description: desc,
		Timeout:     3500 * time.Millisecond,
	})
}

func (a *App) showInfo(title, body string) {
	a.infoModalTitle = title
	a.infoModalBody = body
	a.infoModalOpen = true
}

func (a *App) requestNavigate(target pageID) {
	if target == a.page {
		return
	}
	if a.page == pageSettings && a.hasUnsavedSettings() {
		a.pendingNavPage = target
		a.unsavedModalOpen = true
		return
	}
	a.navigateNow(target)
}

func (a *App) navigateNow(target pageID) {
	a.page = target
	a.reorderMode = false
	switch target {
	case pageHome:
		a.reloadHome()
	case pageArchive:
		a.reloadArchives()
	case pageSettings:
		a.loadSettingsSnapshot()
	}
}

// View 是 MyGo 原生 UI 主视图构建函数。
func (a *App) View(c *ui.Context) {
	// 1. 应用主题（跟随系统深浅色与强调色，并应用优先字体回退链）
	base := *c.Theme()
	base.Radius = 8
	base.Font = settings.BuildFontStack(a.fontFamily)
	c.SetTheme(&base)
	c.SetMenuPresentation(ui.PopoverMenus)

	// 2. 派发后台入队的 Toast
	if len(a.pendingToasts) > 0 {
		for _, t := range a.pendingToasts {
			c.AddToast(t)
		}
		a.pendingToasts = a.pendingToasts[:0]
	}

	// 3. 全局快捷键：F5 / Ctrl+R 刷新当前页
	if c.Shortcut(0, ui.KeyF5) || c.Shortcut(ui.Cmd, ui.KeyR) {
		switch a.page {
		case pageHome:
			a.reloadHome()
		case pageArchive:
			a.reloadArchives()
		}
	}

	t := c.Theme()
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}

	ui.Column(c).Fill().Children(func() {
		a.titleBarView(c)

		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			a.sidebarView(c)

			// 主内容卡片容器：在 Mica 材质上呈现 Windows 11 风格圆角工作区
			content := ui.Column(c).Grow(1).Background(t.Background).
				BorderWidth(1, 0, 0, 1).BorderColor(t.Border)
			if c.Vibrancy() {
				content.Radius(10)
			}
			content.Children(func() {
				switch a.page {
				case pageHome:
					a.homeView(c)
				case pageArchive:
					a.archiveView(c)
				case pageSettings:
					a.settingsView(c)
				case pageDev:
					a.devView(c)
				}
			})
		})
	})

	// 4. 渲染所有活跃的模态弹窗
	a.modalsView(c)
}

func (a *App) titleBarView(c *ui.Context) {
	t := c.Theme()
	tr := a.tr()
	bar := c.TitleBar()
	h := max(bar.Height, 38)

	row := ui.Row(c).Height(h).Padding(0, bar.Right+12, 0, bar.Left+12).
		AlignItems(ui.Center).Gap(8).DragWindow()
	if !c.Vibrancy() {
		row.Background(t.Surface)
	}
	row.Children(func() {
		toggleTooltip := tr.NavToggleTooltip
		if !a.sidebarOpen {
			toggleTooltip = tr.NavExpandTooltip
		}
		toggleBtn := ui.ButtonBase(c.Key("nav-toggle")).Size(30, 28).Radius(6).Center().
			Label(toggleTooltip).Tooltip(toggleTooltip)
		toggleColor := applyHoverIconStyle(toggleBtn, t, false, false, nil)
		toggleBtn.Children(func() {
			ui.Icon(c, iconPanelLeft).FontSize(15).TextColor(toggleColor)
		})
		if toggleBtn.Clicked() {
			a.sidebarOpen = !a.sidebarOpen
		}

		if a.appIcon != nil {
			ui.Image(c, a.appIcon).Size(20, 20).Shrink(0).Fit(ui.Contain)
		}
		ui.Text(c, tr.AppName).FontSize(13).Bold().SingleLine()
		ui.Spacer(c)
	})
}

func (a *App) sidebarView(c *ui.Context) {
	t := c.Theme()
	tr := a.tr()

	w := float32(204)
	if !a.sidebarOpen {
		w = 56
	}
	side := ui.Column(c).Width(w).Padding(8, 8).Gap(4)
	if !c.Vibrancy() {
		side.Background(t.Surface)
	}
	side.Children(func() {
		a.navButton(c, pageHome, iconHome, tr.NavHome)
		a.navButton(c, pageArchive, iconArchive, tr.NavArchive)

		ui.Spacer(c)

		if a.devEnabled {
			a.navButton(c, pageDev, iconCode, tr.NavDeveloperOptions)
		}
		a.navButton(c, pageSettings, iconSettings, tr.NavSettings)
	})
}

func (a *App) navButton(c *ui.Context, id pageID, ico *ui.SVG, label string) {
	t := c.Theme()
	active := a.page == id

	btn := ui.ButtonBase(c.Key("nav-" + string(id))).Height(36).Radius(8).
		AlignItems(ui.Center).Gap(10).Cursor(ui.CursorPointer).Label(label)
	if a.sidebarOpen {
		// ButtonBase 默认设置 Justify(Center)，展开侧边栏时显式重置为左对齐
		btn.Justify(ui.Start).Padding(0, 12)
	} else {
		btn.Center().Tooltip(label)
	}

	switch {
	case active:
		btn.Background(t.Accent.Alpha(0.14))
	case btn.Hovered():
		btn.Background(t.SurfaceHover)
	}

	btn.Children(func() {
		iconColor := t.TextMuted
		textColor := t.Text
		if active {
			iconColor = t.Accent
			textColor = t.Accent
		}
		ui.Icon(c, ico).FontSize(17).TextColor(iconColor).Shrink(0)
		if a.sidebarOpen {
			txt := ui.Text(c, label).FontSize(13).TextColor(textColor).SingleLine().MinWidth(0)
			if active {
				txt.Bold()
			}
		}
	})
	if btn.Clicked() {
		a.requestNavigate(id)
	}
}

// ensureExeMeta 异步提取并缓存指定可执行文件的图标和 FileDescription。
func (a *App) ensureExeMeta(exePath string) *exeMeta {
	if exePath == "" {
		return nil
	}
	key := strings.ToLower(filepath.Clean(exePath))
	if m, ok := a.exeCache[key]; ok {
		return m
	}
	m := &exeMeta{loading: true}
	a.exeCache[key] = m

	go func(p, cacheKey string) {
		desc := winutil.GetFileDescription(p)
		var pngBytes []byte
		if b, err := winutil.ExtractIconPNG(p, 64); err == nil && len(b) > 0 {
			pngBytes = b
		} else if b, err := mygo.Shell.FileIcon(p, 64); err == nil && len(b) > 0 {
			pngBytes = b
		}
		var bmp *ui.Bitmap
		if len(pngBytes) > 0 {
			bmp, _ = ui.DecodeBitmap(pngBytes)
		}
		a.update(func() {
			entry, ok := a.exeCache[cacheKey]
			if !ok {
				entry = &exeMeta{}
				a.exeCache[cacheKey] = entry
			}
			entry.loading = false
			entry.desc = desc
			entry.icon = bmp
		})
	}(exePath, key)

	return m
}

// ensureAltExecutables 异步扫描并缓存软件安装目录下的所有可执行文件列表。
func (a *App) ensureAltExecutables(installPath string) *altExecEntry {
	if installPath == "" {
		return nil
	}
	key := strings.ToLower(filepath.Clean(installPath))
	if e, ok := a.altCache[key]; ok {
		return e
	}
	e := &altExecEntry{loading: true}
	a.altCache[key] = e

	go func(dir, cacheKey string) {
		paths := a.softwareSvc.FindExecutablesInDirectory(dir)
		a.update(func() {
			entry, ok := a.altCache[cacheKey]
			if !ok {
				entry = &altExecEntry{}
				a.altCache[cacheKey] = entry
			}
			entry.loading = false
			entry.paths = paths
		})
	}(installPath, key)

	return e
}

func (a *App) clearSoftwareCaches() {
	a.exeCache = make(map[string]*exeMeta)
	a.altCache = make(map[string]*altExecEntry)
}

func (a *App) displayNameFor(sw model.Software) string {
	if sw.Status == model.StatusManaged && sw.ExecutablePath != "" {
		if meta := a.ensureExeMeta(sw.ExecutablePath); meta != nil && meta.desc != "" {
			return meta.desc
		}
	}
	return sw.Name
}

func (a *App) renderSoftwareIcon(c *ui.Context, sw model.Software, size float32) {
	t := c.Theme()
	if sw.Status == model.StatusManaged && sw.ExecutablePath != "" {
		meta := a.ensureExeMeta(sw.ExecutablePath)
		if meta != nil && meta.loading {
			ui.Box(c).Size(size, size).Shrink(0).Center().Children(func() {
				ui.Spinner(c).Size(min(size*0.6, 20), min(size*0.6, 20))
			})
			return
		}
		if meta != nil && meta.icon != nil {
			ui.Image(c, meta.icon).Size(size, size).Shrink(0).Fit(ui.Contain)
			return
		}
	}
	if sw.Status == model.StatusUnknownArchive && sw.IsBackupArchive {
		ui.Box(c).Size(size, size).Shrink(0).Center().Children(func() {
			ui.Icon(c, iconArchive).FontSize(size * 0.72).TextColor(t.Accent)
		})
		return
	}
	ui.Box(c).Size(size, size).Shrink(0).Center().Children(func() {
		ui.Icon(c, iconAppDefault).FontSize(size * 0.72).TextColor(t.TextMuted)
	})
}

// applyHoverIconStyle 为无背景图标按钮应用悬停/按下高亮与鼠标手型，并返回当前图标前景色。
func applyHoverIconStyle(btn ui.Element, t *ui.Theme, disabled, danger bool, accent *ui.Color) ui.Color {
	if !disabled {
		btn.Cursor(ui.CursorPointer)
	}
	hovered := !disabled && btn.Hovered()
	pressed := !disabled && btn.Pressed()
	// 使用带透明度的前景色叠加，确保在已悬停行（t.SurfaceHover）上依然具有清晰可见的悬停反馈
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

func (a *App) logAndToastError(title string, err error) {
	if err == nil {
		return
	}
	logger.Error("%s: %v", title, err)
	a.toast(title, err.Error())
}

func (a *App) openDirectoryAsync(dirPath string) {
	go func(p string) {
		if err := winutil.OpenDirectory(p); err != nil {
			a.update(func() {
				a.logAndToastError(a.tr().Error, err)
			})
		}
	}(dirPath)
}

func (a *App) revealInExplorerAsync(filePath string, ensureParentDir bool) {
	go func(p string, ensureParent bool) {
		if ensureParent && strings.TrimSpace(p) != "" {
			_ = os.MkdirAll(filepath.Dir(p), 0o755)
		}
		if err := winutil.RevealInExplorer(p); err != nil {
			a.update(func() {
				a.logAndToastError(a.tr().Error, err)
			})
		}
	}(filePath, ensureParentDir)
}
