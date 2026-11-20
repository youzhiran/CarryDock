package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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
		for _, m := range managed {
			if m.ArchivePath != "" {
				linked[normalizePathKey(m.ArchivePath)] = true
			}
			if m.BackupPath != "" {
				linked[normalizePathKey(m.BackupPath)] = true
			}
		}

		var archives, backups []model.Software
		for _, it := range items {
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

func (a *App) filteredArchives(items []model.Software) []model.Software {
	q := strings.ToLower(strings.TrimSpace(a.archiveSearch))
	filter := a.archiveFilter
	if filter == "" {
		filter = archiveFilterAll
	}
	out := make([]model.Software, 0, len(items))
	for _, it := range items {
		isLinked := a.linkedPaths[normalizePathKey(it.ArchivePath)]
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
		if a.linkedPaths[normalizePathKey(it.ArchivePath)] {
			linked++
		} else {
			unlinked++
		}
	}
	return linked, unlinked
}

func (a *App) archiveView(c *ui.Context) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	ui.Column(c).Fill().Padding(20, 24).Gap(14).Children(func() {
		// 顶部标题与操作栏
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tr.ArchiveTitle).FontSize(22).Bold().SingleLine()
			total := len(a.archives) + len(a.backups)
			if total > 0 {
				ui.Badge(c, fmt.Sprintf("%d", total)).Background(p.subtleBg).TextColor(t.Text)
			}
			ui.Spacer(c)

			if iconButton(c, "arch-refresh", iconRefresh, tr.Refresh, false, false, nil).Clicked() {
				a.reloadArchives()
			}
			if iconButton(c, "arch-open-dir", iconFolder, tr.ArchiveOpenArchiveDir, false, false, nil).Clicked() {
				if root, err := a.softwareSvc.ResolveArchiveDirectoryPath(); err == nil {
					a.openDirectoryAsync(root)
				} else {
					a.showInfo(tr.HomeInstallDirNotConfig, tr.HomeInstallDirNotConfigMsg)
				}
			}
			if ui.Button(c.Key("arch-open-backup"), tr.ArchiveOpenBackupDir).Clicked() {
				if dir, err := a.softwareSvc.ResolveBackupDirectoryPath(); err == nil {
					a.openDirectoryAsync(dir)
				} else {
					a.showInfo(tr.HomeInstallDirNotConfig, tr.HomeInstallDirNotConfigMsg)
				}
			}
			if ui.PrimaryButton(c.Key("arch-create-backup"), "").Children(func() {
				ui.Icon(c, iconPlus).FontSize(14)
				ui.Text(c, tr.ArchiveCreateBackup).SingleLine()
			}).Clicked() {
				a.openCreateBackupModal()
			}
		})

		if !a.archiveLoading && !a.archiveNotConfigured && (len(a.archives) > 0 || len(a.backups) > 0) {
			if a.archiveFilter == "" {
				a.archiveFilter = archiveFilterAll
			}
			archLinked, archUnlinked := a.countLinkedArchives(a.archives)
			bakLinked, bakUnlinked := a.countLinkedArchives(a.backups)
			totalAll := len(a.archives) + len(a.backups)
			totalLinked := archLinked + bakLinked
			totalUnlinked := archUnlinked + bakUnlinked

			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				sf := ui.SearchField(c.Key("arch-search"), &a.archiveSearch).
					Placeholder(tr.ArchiveSearchPlaceholder).Label(tr.ArchiveSearchPlaceholder).Width(240)
				if a.archiveFocusFind {
					a.archiveFocusFind = false
					sf.Focus()
				}
				ui.Spacer(c)
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					if filterChip(c, "arch-filter-all", tr.FilterAll, totalAll, a.archiveFilter == archiveFilterAll).Clicked() {
						a.archiveFilter = archiveFilterAll
					}
					if filterChip(c, "arch-filter-linked", tr.FilterLinked, totalLinked, a.archiveFilter == archiveFilterLinked).Clicked() {
						a.archiveFilter = archiveFilterLinked
					}
					if filterChip(c, "arch-filter-unlinked", tr.FilterUnlinked, totalUnlinked, a.archiveFilter == archiveFilterUnlinked).Clicked() {
						a.archiveFilter = archiveFilterUnlinked
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
						ui.Icon(c, iconArchive).FontSize(28).TextColor(t.Accent)
					})
					ui.Text(c, tr.HomeConfigureInstallFirst).FontSize(17).Bold().TextAlign(ui.Center)
					ui.Row(c).Padding(4, 0, 0, 0).Children(func() {
						if ui.PrimaryButton(c, tr.HomeGoToSettings).Clicked() {
							a.requestNavigate(pageSettings)
						}
					})
				})
			})

		default:
			filteredArchives := a.filteredArchives(a.archives)
			filteredBackups := a.filteredArchives(a.backups)
			hasFilter := strings.TrimSpace(a.archiveSearch) != "" ||
				(a.archiveFilter != "" && a.archiveFilter != archiveFilterAll)

			if hasFilter && len(filteredArchives) == 0 && len(filteredBackups) == 0 {
				if emptySearchView(c, tr.NoMatchingResults, tr.NoMatchingResultsHint, tr.ClearFilter) {
					a.archiveSearch = ""
					a.archiveFilter = archiveFilterAll
				}
				return
			}

			ui.Scroll(c.Key("archive-scroll")).TrackScroll(&a.archiveScrollState).Grow(1).Children(func() {
				ui.Column(c).Gap(18).Padding(2, 4, 16, 0).Children(func() {
					// 归档文件区块
					ui.Column(c).Gap(8).Children(func() {
						sectionTitleRow(c, iconArchive, tr.ArchiveFiles, len(filteredArchives))
						if len(filteredArchives) == 0 {
							ui.Box(c).Padding(16).Radius(10).Background(p.card).
								Border(1, p.cardBorder).Shadow(0, 1, 3, 0, p.shadow).Children(func() {
								ui.Text(c, tr.ArchiveNoArchiveFiles).FontSize(13).TextColor(t.TextMuted)
							})
						} else {
							for i := range filteredArchives {
								a.archiveFileRow(c, filteredArchives[i], false)
							}
						}
					})

					// 备份文件区块
					ui.Column(c).Gap(8).Children(func() {
						sectionTitleRow(c, iconRestore, tr.ArchiveBackupFiles, len(filteredBackups))
						if len(filteredBackups) == 0 {
							ui.Box(c).Padding(16).Radius(10).Background(p.card).
								Border(1, p.cardBorder).Shadow(0, 1, 3, 0, p.shadow).Children(func() {
								ui.Text(c, tr.ArchiveNoBackupFiles).FontSize(13).TextColor(t.TextMuted)
							})
						} else {
							for i := range filteredBackups {
								a.archiveFileRow(c, filteredBackups[i], true)
							}
						}
					})
				})
			})
		}
	})
}

func (a *App) archiveFileRow(c *ui.Context, item model.Software, isBackup bool) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()
	isLinked := a.linkedPaths[normalizePathKey(item.ArchivePath)]

	row := ui.Row(c.Key("arch-row-"+item.ArchivePath)).Padding(12, 16).Radius(11).
		Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 4, 0, p.shadow).
		Gap(14).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(p.cardHover).Border(1, p.borderStrong)
	}
	row.Children(func() {
		left := ui.Row(c.Key("arch-reveal-" + item.ArchivePath)).Grow(1).MinWidth(0).Gap(12).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Tooltip(tr.ArchiveRevealInExplorer)
		left.Children(func() {
			ico := iconArchive
			icoColor := t.TextMuted
			icoBg := p.subtleBg
			if isBackup {
				ico = iconRestore
				icoColor = t.Accent
				icoBg = p.accentTint
			}
			ui.Box(c).Size(36, 36).Shrink(0).Radius(9).Background(icoBg).Center().Children(func() {
				ui.Icon(c, ico).FontSize(17).TextColor(icoColor)
			})
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
