package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"carrydock/internal/archive"
	"carrydock/internal/model"
	"carrydock/internal/software"
	"carrydock/internal/winutil"
	"github.com/egoist/mygo/ui"
)

func (a *App) reloadArchives() {
	a.archiveLoading = true
	a.archiveNotConfigured = false

	go func() {
		items, err := a.softwareSvc.ListArchivesForManager(true)
		if errors.Is(err, software.ErrInstallPathNotConfigured) {
			a.update(func() {
				a.archiveLoading = false
				a.archiveNotConfigured = true
				a.archives = nil
				a.backups = nil
				a.linkedPaths = make(map[string]bool)
				a.linkedSoftware = make(map[string]model.Software)
			})
			return
		}
		if err != nil {
			a.update(func() {
				a.archiveLoading = false
				a.logAndToastError(a.tr().Error, err)
			})
			return
		}

		managed, _ := a.softwareSvc.LoadManagedSoftware()
		linked := make(map[string]bool)
		linkedSw := make(map[string]model.Software)
		for _, m := range managed {
			if m.ArchivePath != "" {
				k := normalizePathKey(m.ArchivePath)
				linked[k] = true
				linkedSw[k] = m
			}
			if m.BackupPath != "" {
				k := normalizePathKey(m.BackupPath)
				linked[k] = true
				linkedSw[k] = m
			}
		}

		allowedExts := a.settingsSvc.GetExecutableExtensions()
		var archives, backups []model.Software
		for _, it := range items {
			it.ExecutablePath = resolveArchiveExePath(it, linkedSw, managed, allowedExts, true)
			if it.IsBackupArchive {
				backups = append(backups, it)
			} else {
				archives = append(archives, it)
			}
		}
		sort.Slice(archives, func(i, j int) bool {
			return strings.ToLower(archives[i].Name) < strings.ToLower(archives[j].Name)
		})
		sort.Slice(backups, func(i, j int) bool {
			return strings.ToLower(backups[i].Name) > strings.ToLower(backups[j].Name)
		})

		a.update(func() {
			a.archiveLoading = false
			a.archives = archives
			a.backups = backups
			a.linkedPaths = linked
			a.linkedSoftware = linkedSw
		})
	}()
}

func normalizePathKey(p string) string {
	clean := filepath.Clean(strings.TrimSpace(p))
	if abs, err := filepath.Abs(clean); err == nil {
		clean = abs
	}
	return strings.ToLower(clean)
}

func (a *App) isArchiveLinked(archivePath string) bool {
	k := normalizePathKey(archivePath)
	if a.linkedPaths[k] {
		return true
	}
	_, ok := a.linkedSoftware[k]
	return ok
}

func isExecutableArchiveFile(filePath string, allowedExts []string) bool {
	if filePath == "" || archive.DetectFormat(filePath) != archive.FormatUnknown {
		return false
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), ".")
	return ext != "" && (slices.Contains(allowedExts, ext) || ext == "exe" || ext == "ico" || ext == "lnk")
}

// resolveArchiveExePath 为归档或备份条目解析用于提取软件图标的可执行文件路径。
func resolveArchiveExePath(
	item model.Software,
	linkedSw map[string]model.Software,
	managed []model.Software,
	allowedExts []string,
	checkDisk bool,
) string {
	existsOK := func(p string) bool {
		if p == "" {
			return false
		}
		if !checkDisk {
			return true
		}
		info, err := os.Stat(p)
		return err == nil && !info.IsDir()
	}

	// 1. 直接关联的已托管软件主程序
	if sw, ok := linkedSw[normalizePathKey(item.ArchivePath)]; ok && existsOK(sw.ExecutablePath) {
		return sw.ExecutablePath
	}

	// 2. 独立可执行文件归档本身（如单文件 .exe）
	if isExecutableArchiveFile(item.ArchivePath, allowedExts) && existsOK(item.ArchivePath) {
		return item.ArchivePath
	}

	// 3. 按名称匹配已托管软件（使未关联的历史时间戳备份或同名归档也能显示对应软件图标）
	if item.IsBackupArchive {
		lowerName := strings.ToLower(item.Name)
		bestLen := 0
		bestExe := ""
		for _, m := range managed {
			if !existsOK(m.ExecutablePath) {
				continue
			}
			var prefixes []string
			if san := software.SanitizeSoftwareName(m.Name); san != "" {
				prefixes = append(prefixes, strings.ToLower(san)+"-")
			}
			if m.InstallPath != "" {
				if baseSan := software.SanitizeSoftwareName(filepath.Base(m.InstallPath)); baseSan != "" {
					prefixes = append(prefixes, strings.ToLower(baseSan)+"-")
				}
			}
			for _, prefix := range prefixes {
				if strings.HasPrefix(lowerName, prefix) && len(prefix) > bestLen {
					bestLen = len(prefix)
					bestExe = m.ExecutablePath
				}
			}
		}
		return bestExe
	}

	stem := strings.ToLower(archive.StripArchiveExtension(item.ArchivePath))
	if stem != "" {
		for _, m := range managed {
			if !existsOK(m.ExecutablePath) {
				continue
			}
			if strings.ToLower(software.SanitizeSoftwareName(m.Name)) == stem {
				return m.ExecutablePath
			}
			if m.InstallPath != "" && strings.ToLower(filepath.Base(m.InstallPath)) == stem {
				return m.ExecutablePath
			}
		}
	}
	return ""
}

func (a *App) archiveIconExePath(item model.Software, isBackup bool) string {
	if item.ExecutablePath != "" {
		return item.ExecutablePath
	}
	item.IsBackupArchive = isBackup
	return resolveArchiveExePath(item, a.linkedSoftware, a.managed, a.settingsSvc.GetExecutableExtensions(), false)
}

func (a *App) filteredArchives(items []model.Software, search string, filter archiveFilterMode) []model.Software {
	q := strings.ToLower(strings.TrimSpace(search))
	if filter == "" {
		filter = archiveFilterAll
	}
	out := make([]model.Software, 0, len(items))
	for _, it := range items {
		isLinked := a.isArchiveLinked(it.ArchivePath)
		switch filter {
		case archiveFilterLinked:
			if !isLinked {
				continue
			}
		case archiveFilterUnlinked:
			if isLinked {
				continue
			}
		}
		if q != "" &&
			!strings.Contains(strings.ToLower(it.Name), q) &&
			!strings.Contains(strings.ToLower(it.ArchivePath), q) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func (a *App) countLinkedArchives(items []model.Software) (linked, unlinked int) {
	for _, it := range items {
		if a.isArchiveLinked(it.ArchivePath) {
			linked++
		} else {
			unlinked++
		}
	}
	return linked, unlinked
}

func (a *App) archiveView(c *ui.Context) {
	a.archiveCollectionView(c, false)
}

func (a *App) backupView(c *ui.Context) {
	a.archiveCollectionView(c, true)
}

func (a *App) archiveCollectionView(c *ui.Context, isBackup bool) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	items := a.archives
	search := &a.archiveSearch
	filter := &a.archiveFilter
	focusFind := &a.archiveFocusFind
	scrollState := &a.archiveScrollState
	keyPrefix := "arch"
	title := tr.ArchiveTitle
	searchPlaceholder := tr.ArchiveSearchPlaceholder
	emptyIcon := iconArchive
	emptyTitle := tr.ArchiveNoArchiveFiles
	emptyHint := tr.ArchiveNoArchiveHint
	if isBackup {
		items = a.backups
		search = &a.backupSearch
		filter = &a.backupFilter
		focusFind = &a.backupFocusFind
		scrollState = &a.backupScrollState
		keyPrefix = "bak"
		title = tr.BackupTitle
		searchPlaceholder = tr.BackupSearchPlaceholder
		emptyIcon = iconRestore
		emptyTitle = tr.ArchiveNoBackupFiles
		emptyHint = tr.BackupNoBackupHint
	}

	ui.Column(c).Fill().Padding(20, 24).Gap(14).Children(func() {
		// 顶部标题与操作栏
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(22).Bold().SingleLine()
			if len(items) > 0 {
				ui.Badge(c, fmt.Sprintf("%d", len(items))).Background(p.subtleBg).TextColor(t.Text)
			}
			ui.Spacer(c)

			if iconButton(c, keyPrefix+"-refresh", iconRefresh, tr.Refresh, false, false, nil).Clicked() {
				a.reloadArchives()
			}
			if !isBackup {
				if ui.Button(c.Key("arch-open-dir"), tr.ArchiveOpenArchiveDir).Clicked() {
					if root, err := a.softwareSvc.ResolveArchiveDirectoryPath(); err == nil {
						a.openDirectoryAsync(root)
					} else {
						a.showInfo(tr.HomeInstallDirNotConfig, tr.HomeInstallDirNotConfigMsg)
					}
				}
			} else {
				if ui.Button(c.Key("bak-open-dir"), tr.ArchiveOpenBackupDir).Clicked() {
					if dir, err := a.softwareSvc.ResolveBackupDirectoryPath(); err == nil {
						a.openDirectoryAsync(dir)
					} else {
						a.showInfo(tr.HomeInstallDirNotConfig, tr.HomeInstallDirNotConfigMsg)
					}
				}
				if ui.PrimaryButton(c.Key("bak-create-backup"), "").Children(func() {
					ui.Icon(c, iconPlus).FontSize(14)
					ui.Text(c, tr.ArchiveCreateBackup).SingleLine()
				}).Clicked() {
					a.openCreateBackupModal()
				}
			}
		})

		if !a.archiveLoading && !a.archiveNotConfigured && len(items) > 0 {
			if *filter == "" {
				*filter = archiveFilterAll
			}
			linkedCount, unlinkedCount := a.countLinkedArchives(items)

			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				sf := ui.SearchField(c.Key(keyPrefix+"-search"), search).
					Placeholder(searchPlaceholder).Label(searchPlaceholder).Width(240)
				if *focusFind {
					*focusFind = false
					sf.Focus()
				}
				ui.Spacer(c)
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					if filterChip(c, keyPrefix+"-filter-all", tr.FilterAll, len(items), *filter == archiveFilterAll).Clicked() {
						*filter = archiveFilterAll
					}
					if filterChip(c, keyPrefix+"-filter-linked", tr.FilterLinked, linkedCount, *filter == archiveFilterLinked).Clicked() {
						*filter = archiveFilterLinked
					}
					if filterChip(c, keyPrefix+"-filter-unlinked", tr.FilterUnlinked, unlinkedCount, *filter == archiveFilterUnlinked).Clicked() {
						*filter = archiveFilterUnlinked
					}
				})
			})
		}

		switch {
		case a.archiveLoading:
			ui.Column(c).Grow(1).Center().Gap(12).Children(func() {
				ui.Spinner(c).Size(28, 28)
				ui.Text(c, tr.Loading).FontSize(13).TextColor(t.TextMuted)
			})

		case a.archiveNotConfigured:
			ui.Column(c).Grow(1).Center().Padding(24).Children(func() {
				ui.Column(c).MaxWidth(440).Padding(32, 36).Gap(12).Radius(14).
					Background(p.card).Border(1, p.cardBorder).Shadow(0, 2, 12, 0, p.shadow).
					Center().Children(func() {
					ui.Box(c).Size(60, 60).Radius(16).Background(p.accentTint).Center().Children(func() {
						ui.Icon(c, emptyIcon).FontSize(28).TextColor(t.Accent)
					})
					ui.Text(c, tr.HomeConfigureInstallFirst).FontSize(17).Bold().TextAlign(ui.Center)
					ui.Row(c).Padding(4, 0, 0, 0).Children(func() {
						if ui.PrimaryButton(c, tr.HomeGoToSettings).Clicked() {
							a.requestNavigate(pageSettings)
						}
					})
				})
			})

		case len(items) == 0:
			ui.Column(c).Grow(1).Center().Padding(24).Children(func() {
				ui.Column(c).MaxWidth(440).Padding(32, 36).Gap(12).Radius(14).
					Background(p.card).Border(1, p.cardBorder).Shadow(0, 2, 12, 0, p.shadow).
					Center().Children(func() {
					ui.Box(c).Size(60, 60).Radius(16).Background(p.subtleBg).Center().Children(func() {
						ui.Icon(c, emptyIcon).FontSize(28).TextColor(t.TextMuted)
					})
					ui.Text(c, emptyTitle).FontSize(17).Bold().TextAlign(ui.Center)
					if emptyHint != "" {
						ui.Text(c, emptyHint).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
					}
					if isBackup {
						ui.Row(c).Padding(4, 0, 0, 0).Children(func() {
							if ui.PrimaryButton(c.Key("bak-empty-create"), tr.ArchiveCreateBackup).Clicked() {
								a.openCreateBackupModal()
							}
						})
					}
				})
			})

		default:
			filtered := a.filteredArchives(items, *search, *filter)
			hasFilter := strings.TrimSpace(*search) != "" ||
				(*filter != "" && *filter != archiveFilterAll)

			if hasFilter && len(filtered) == 0 {
				if emptySearchView(c, tr.NoMatchingResults, tr.NoMatchingResultsHint, tr.ClearFilter) {
					*search = ""
					*filter = archiveFilterAll
				}
				return
			}

			ui.Scroll(c.Key(keyPrefix + "-scroll")).TrackScroll(scrollState).Grow(1).Children(func() {
				ui.Column(c).Gap(8).Padding(2, 4, 16, 0).Children(func() {
					for i := range filtered {
						a.archiveFileRow(c, filtered[i], isBackup)
					}
				})
			})
		}
	})
}

func (a *App) archiveFileRow(c *ui.Context, item model.Software, isBackup bool) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()
	isLinked := a.isArchiveLinked(item.ArchivePath)

	row := ui.Row(c.Key("arch-row-"+item.ArchivePath)).Padding(12, 16).Radius(11).
		Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 4, 0, p.shadow).
		Gap(14).AlignItems(ui.Center)
	rowHovered := row.Hovered()
	if rowHovered {
		row.Background(p.cardHover).Border(1, p.borderStrong)
	}
	row.Children(func() {
		left := ui.Row(c.Key("arch-reveal-" + item.ArchivePath)).Grow(1).MinWidth(0).Gap(12).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Tooltip(tr.ArchiveRevealInExplorer)
		left.Children(func() {
			a.renderArchiveItemIcon(c, item, isBackup, rowHovered)
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				ui.Row(c).Gap(8).MinWidth(0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.Name).FontSize(14).Bold().SingleLine().MinWidth(0)
					if isLinked {
						pillBadge(c, tr.ArchiveLinked, p.successTint, t.Success)
					} else {
						pillBadge(c, tr.FilterUnlinked, p.subtleBg, t.TextMuted)
					}
				})
				ui.Text(c, item.ArchivePath).FontSize(11).TextColor(t.TextMuted).TruncateMiddle().SingleLine().MinWidth(0)
			})
		})
		if left.Clicked() {
			a.revealInExplorerAsync(item.ArchivePath, false)
		}

		// 右侧操作：还原 / 手动关联 / 删除
		ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
			if iconButton(c, "arch-restore-"+item.ArchivePath, iconRefresh, tr.ArchiveRestore, false, false, nil).Clicked() {
				a.runAddFile(item.ArchivePath, true)
			}
			var linkColor *ui.Color
			if isLinked {
				green := t.Success
				linkColor = &green
			}
			if iconButton(c, "arch-link-"+item.ArchivePath, iconLink, tr.ArchiveManualLink, false, false, linkColor).Clicked() {
				a.openManualLinkModal(item, isBackup)
			}
			if iconButton(c, "arch-del-"+item.ArchivePath, iconTrash, tr.Delete, false, true, nil).Clicked() {
				a.deleteTarget = item
				a.deleteIsArchiveMgr = true
				a.deleteIsBackup = isBackup
				a.deleteModalOpen = true
			}
		})
	})
}

// renderArchiveItemIcon 渲染归档/备份条目图标，并在右下角叠加归档或备份小角标。
func (a *App) renderArchiveItemIcon(c *ui.Context, item model.Software, isBackup bool, rowHovered bool) {
	t := c.Theme()
	p := paletteOf(t)

	ui.Box(c).Size(44, 44).Shrink(0).Children(func() {
		ui.Box(c).Size(40, 40).Radius(10).Background(p.subtleBg).Center().Children(func() {
			exePath := a.archiveIconExePath(item, isBackup)
			if exePath != "" {
				meta := a.ensureExeMeta(exePath)
				switch {
				case meta != nil && meta.loading:
					ui.Spinner(c).Size(16, 16)
				case meta != nil && meta.icon != nil:
					ui.Image(c, meta.icon).Size(28, 28).Shrink(0).Fit(ui.Contain)
				default:
					ui.Icon(c, iconAppDefault).FontSize(22).TextColor(t.TextMuted)
				}
			} else {
				ui.Icon(c, iconAppDefault).FontSize(22).TextColor(t.TextMuted)
			}
		})

		// 右下角不透明小角标 + 卡片底色描边隔离，避免下方软件图标透色
		ringColor := p.card
		if rowHovered {
			ringColor = p.cardHover
		}
		badgeIcon := iconArchive
		badgeBg := ui.Hex("#64748b")
		if t.Dark {
			badgeBg = ui.Hex("#4b5563")
		}
		if isBackup {
			badgeIcon = iconRestore
			badgeBg = t.Accent
		}
		badgeFg := ui.Hex("#ffffff")

		ui.Box(c).Size(18, 18).Radius(99).Absolute().Bottom(0).Right(0).
			Background(badgeBg).Border(1.5, ringColor).Shadow(0, 1, 2, 0, p.shadow).
			Center().PassThrough().Children(func() {
			ui.Icon(c, badgeIcon).FontSize(10).TextColor(badgeFg)
		})
	})
}

func (a *App) openManualLinkModal(item model.Software, isBackup bool) {
	tr := a.tr()
	managed, err := a.softwareSvc.LoadManagedSoftware()
	if err != nil {
		a.showInfo(tr.Error, err.Error())
		return
	}
	if len(managed) == 0 {
		a.showInfo(tr.ArchiveNoAvailableSw, tr.ArchiveNoAvailableSwHint)
		return
	}
	a.linkTarget = item
	a.linkIsBackup = isBackup
	a.linkManaged = managed
	a.linkNameToID = make(map[string]string, len(managed))
	a.linkSelectedName = ""

	for i, m := range managed {
		label := fmt.Sprintf("%s (%s)", m.Name, filepath.Base(m.InstallPath))
		if _, dup := a.linkNameToID[label]; dup {
			label = fmt.Sprintf("%s #%d", label, i+1)
		}
		a.linkNameToID[label] = m.ID
		if winutil.SamePath(m.ArchivePath, item.ArchivePath) || winutil.SamePath(m.BackupPath, item.ArchivePath) {
			a.linkSelectedName = label
		}
		if a.linkSelectedName == "" && i == 0 {
			a.linkSelectedName = label
		}
	}
	a.linkModalOpen = true
}

func (a *App) openCreateBackupModal() {
	tr := a.tr()
	managed, err := a.softwareSvc.LoadManagedSoftware()
	if err != nil {
		a.showInfo(tr.Error, err.Error())
		return
	}
	if len(managed) == 0 {
		a.showInfo(tr.ArchiveNoAvailableSw, tr.ArchiveNoAvailableSwHint)
		return
	}
	a.backupManaged = managed
	a.backupNameToID = make(map[string]string, len(managed))
	a.backupSelectedName = ""
	for i, m := range managed {
		label := fmt.Sprintf("%s (%s)", m.Name, filepath.Base(m.InstallPath))
		if _, dup := a.backupNameToID[label]; dup {
			label = fmt.Sprintf("%s #%d", label, i+1)
		}
		a.backupNameToID[label] = m.ID
		if i == 0 {
			a.backupSelectedName = label
		}
	}
	a.backupModalOpen = true
}
