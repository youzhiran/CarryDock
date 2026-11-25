package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"carrydock/internal/archive"
	"carrydock/internal/i18n"
	"carrydock/internal/logger"
	"carrydock/internal/model"
	"carrydock/internal/software"
	"carrydock/internal/winutil"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) reloadHome() {
	a.homeLoading = true
	a.homeNotConfigured = false
	a.clearSoftwareCaches()

	go func() {
		list, err := a.softwareSvc.GetSoftwareList()
		a.update(func() {
			a.homeLoading = false
			if errors.Is(err, software.ErrInstallPathNotConfigured) {
				a.homeNotConfigured = true
				a.managed = nil
				a.unmanaged = nil
				return
			}
			if err != nil {
				a.logAndToastError(a.tr().Error, err)
				return
			}
			var managed, unmanaged []model.Software
			for _, s := range list {
				switch s.Status {
				case model.StatusManaged:
					managed = append(managed, s)
				case model.StatusUnknownInstall:
					unmanaged = append(unmanaged, s)
				}
			}
			a.managed = managed
			a.unmanaged = unmanaged
		})
	}()
}

func (a *App) filteredSoftware(items []model.Software) []model.Software {
	q := strings.ToLower(strings.TrimSpace(a.searchQuery))
	if q == "" {
		return items
	}
	out := make([]model.Software, 0, len(items))
	for _, s := range items {
		name := strings.ToLower(a.displayNameFor(s))
		ver := strings.ToLower(a.versionFor(s))
		verBadge := strings.ToLower(formatVersionBadge(ver))
		if strings.Contains(name, q) ||
			strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.InstallPath), q) ||
			strings.Contains(strings.ToLower(filepath.Base(s.ExecutablePath)), q) ||
			(ver != "" && (strings.Contains(ver, q) || strings.Contains(verBadge, q))) {
			out = append(out, s)
		}
	}
	return out
}

func (a *App) homeView(c *ui.Context) {
	tr := a.tr()

	// 整个主页作为外部文件拖放接收区域
	root := ui.Box(c).Fill()
	if dropped := root.DroppedFiles(); len(dropped) > 0 {
		a.handleDroppedPaths(dropped)
	}
	isDragOver := root.FileDragOver()

	root.Children(func() {
		t := c.Theme()
		p := paletteOf(t)

		ui.Column(c).Fill().Padding(20, 24).Gap(14).Children(func() {
			// 顶部标题栏 + 主工具条
			a.homeHeaderBar(c)

			// 搜索与筛选子栏（非排序、有数据时展示）
			if !a.reorderMode && !a.homeLoading && !a.homeNotConfigured && (len(a.managed) > 0 || len(a.unmanaged) > 0) {
				a.homeFilterBar(c)
			}

			// 排序模式提示条
			if a.reorderMode {
				ui.Row(c).Padding(10, 14).Radius(10).Background(p.accentTint).
					Border(1, t.Accent.Alpha(0.35)).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconSort).FontSize(15).TextColor(t.Accent)
					ui.Text(c, tr.HomeReorderHint).FontSize(13).TextColor(t.Text).Grow(1).MinWidth(0)
				})
			}

			// 主体内容
			switch {
			case a.homeLoading:
				ui.Column(c).Grow(1).Center().Gap(12).Children(func() {
					ui.Spinner(c).Size(28, 28)
					ui.Text(c, tr.Loading).FontSize(13).TextColor(t.TextMuted)
				})

			case a.homeNotConfigured:
				ui.Column(c).Grow(1).Center().Padding(24).Children(func() {
					ui.Column(c).MaxWidth(440).Padding(32, 36).Gap(12).Radius(14).
						Background(p.card).Border(1, p.cardBorder).Shadow(0, 2, 12, 0, p.shadow).
						Center().Children(func() {
						ui.Box(c).Size(60, 60).Radius(16).Background(p.accentTint).Center().Children(func() {
							ui.Icon(c, iconFolder).FontSize(28).TextColor(t.Accent)
						})
						ui.Text(c, tr.HomeConfigureInstallFirst).FontSize(17).Bold().TextAlign(ui.Center)
						ui.Text(c, tr.HomeInstallDirNotConfigMsg).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
						ui.Row(c).Padding(4, 0, 0, 0).Children(func() {
							if ui.PrimaryButton(c, tr.HomeGoToSettings).Clicked() {
								a.requestNavigate(pageSettings)
							}
						})
					})
				})

			case len(a.managed) == 0 && len(a.unmanaged) == 0:
				ui.Column(c).Grow(1).Center().Padding(24).Children(func() {
					ui.Column(c).MaxWidth(460).Padding(32, 36).Gap(12).Radius(14).
						Background(p.card).Border(1, p.borderStrong).BorderStyle(ui.BorderDashed).
						Shadow(0, 2, 12, 0, p.shadow).Center().Children(func() {
						ui.Box(c).Size(60, 60).Radius(16).Background(p.subtleBg).Center().Children(func() {
							ui.Icon(c, iconUpload).FontSize(28).TextColor(t.Accent)
						})
						ui.Text(c, tr.HomeNoSoftware).FontSize(17).Bold().TextAlign(ui.Center)
						ui.Text(c, tr.HomeNoSoftwareHint).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
						ui.Row(c).Gap(8).Padding(6, 0, 0, 0).Children(func() {
							if ui.PrimaryButton(c, tr.Add).Clicked() {
								a.startAddSoftwarePicker()
							}
							if ui.Button(c, tr.HomeScanCurrentDir).Clicked() {
								a.openScanConfirmModal()
							}
						})
					})
				})

			case a.reorderMode:
				a.homeReorderListView(c)

			case a.useGrid:
				a.homeGridContentView(c)

			default:
				a.homeListContentView(c)
			}
		})

		// 拖放文件悬浮提示遮罩
		if isDragOver {
			ui.Column(c).Absolute().Top(12).Right(12).Bottom(12).Left(12).
				Radius(14).Background(p.card.Alpha(0.95)).Border(2, t.Accent).BorderStyle(ui.BorderDashed).
				Center().Gap(12).PassThrough().Children(func() {
				ui.Box(c).Size(64, 64).Radius(18).Background(p.accentTint).Center().Children(func() {
					ui.Icon(c, iconUpload).FontSize(32).TextColor(t.Accent)
				})
				ui.Text(c, tr.HomeDropToAdd).FontSize(18).Bold()
				ui.Text(c, tr.HomeDropHint).FontSize(13).TextColor(t.TextMuted)
			})
		}
	})
}

func (a *App) homeHeaderBar(c *ui.Context) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	row := ui.Row(c.Key("home-header-bar")).Gap(10).AlignItems(ui.Center)

	// 计算工具栏当前可用总宽度（优先取窗口实时尺寸减去侧边栏与内边距，确保首帧与缩放即时响应）
	winW, _ := c.Size()
	sideW := float32(204)
	if !a.sidebarOpen {
		sideW = 56
	}
	availW := winW - sideW - 49
	if availW <= 0 {
		availW = row.Bounds().W
	}
	if availW <= 0 {
		availW = 760
	}

	row.Children(func() {
		ui.Text(c, tr.HomeManagedSoftware).FontSize(22).Bold().SingleLine()
		if len(a.managed) > 0 {
			ui.Badge(c, fmt.Sprintf("%d", len(a.managed))).Background(p.subtleBg).TextColor(t.Text)
		}
		if len(a.unmanaged) > 0 {
			pillBadge(c, i18n.Fmt(tr.HomeCountUnmanagedFmt, len(a.unmanaged)), p.warningTint, t.Warning)
		}
		ui.Spacer(c)

		if a.reorderMode {
			if ui.Button(c.Key("reorder-cancel"), tr.Cancel).Clicked() {
				a.reorderMode = false
				a.reloadHome()
			}
			if ui.PrimaryButton(c.Key("reorder-finish"), "").Children(func() {
				ui.Icon(c, iconCheck).FontSize(14)
				ui.Text(c, tr.Finish).SingleLine()
			}).Clicked() {
				a.saveCurrentOrder()
			}
			return
		}

		setGrid := func(grid bool) {
			if a.useGrid == grid {
				return
			}
			a.useGrid = grid
			go func(v bool) {
				if err := a.settingsSvc.SaveUseGridLayout(v); err != nil {
					logger.Warn("保存布局偏好失败: %v", err)
				}
			}(grid)
		}

		// 估算左侧标题区与右侧主按钮「添加」所占宽度，动态计算可展开的工具栏按钮预算
		titleW, _ := c.MeasureText(0, ui.Span{Text: tr.HomeManagedSoftware, Size: 22, Weight: 700})
		if titleW <= 0 {
			titleW = float32(len([]rune(tr.HomeManagedSoftware))) * 22
		}
		leftW := titleW
		if len(a.managed) > 0 {
			leftW += 10 + 32
		}
		if len(a.unmanaged) > 0 {
			unmanagedText := i18n.Fmt(tr.HomeCountUnmanagedFmt, len(a.unmanaged))
			uw, _ := c.MeasureText(0, ui.Span{Text: unmanagedText, Size: 11, Weight: 700})
			if uw <= 0 {
				uw = float32(len([]rune(unmanagedText))) * 11
			}
			leftW += 10 + uw + 18
		}
		addTextW, _ := c.MeasureText(0, ui.Span{Text: tr.Add, Size: 13})
		if addTextW <= 0 {
			addTextW = float32(len([]rune(tr.Add))) * 13
		}
		addBtnW := 10 + 14 + 6 + addTextW + 28
		minSpacerW := float32(20)
		toolbarBudget := availW - leftW - minSpacerW - addBtnW

		type toolbarItem struct {
			width      float32
			render     func(hasNextInline bool)
			renderMenu func(m *ui.Menu)
		}

		var items []toolbarItem

		// 1. 列表 / 网格 视图切换器
		items = append(items, toolbarItem{
			width: 87, // 66px 分段控件 + 10px 间距 + 1px 分隔线 + 10px 间距
			render: func(hasNextInline bool) {
				ui.Row(c.Key("view-mode-seg")).Padding(3).Radius(8).
					Background(p.subtleBg).Border(1, p.cardBorder).Gap(2).AlignItems(ui.Center).Children(func() {
					segBtn := func(key string, ico *ui.SVG, tip string, active bool) bool {
						b := ui.ButtonBase(c.Key(key)).Size(28, 24).Radius(6).Center().
							Cursor(ui.CursorPointer).Label(tip).Tooltip(tip)
						fg := t.TextMuted
						switch {
						case active:
							b.Background(p.card).Shadow(0, 1, 2, 0, p.shadow)
							fg = t.Accent
						case b.Hovered():
							b.Background(t.Text.Alpha(0.06))
							fg = t.Text
						}
						b.Children(func() {
							ui.Icon(c, ico).FontSize(14).TextColor(fg)
						})
						return b.Clicked()
					}
					if segBtn("btn-view-list", iconList, tr.HomeListView, !a.useGrid) {
						setGrid(false)
					}
					if segBtn("btn-view-grid", iconGrid, tr.HomeGridView, a.useGrid) {
						setGrid(true)
					}
				})
				if hasNextInline {
					verticalSep(c)
				}
			},
			renderMenu: func(m *ui.Menu) {
				if a.useGrid {
					if m.Item(tr.HomeListView).Icon(menuIconList).Chosen() {
						setGrid(false)
					}
				} else {
					if m.Item(tr.HomeGridView).Icon(menuIconGrid).Chosen() {
						setGrid(true)
					}
				}
			},
		})

		// 2. 自定义排序（仅列表模式且有 >=2 项时可用）
		if !a.useGrid && len(a.managed) > 1 {
			items = append(items, toolbarItem{
				width: 40,
				render: func(_ bool) {
					if iconButton(c, "btn-reorder", iconSort, tr.HomeReorder, false, false, nil).Clicked() {
						a.searchQuery = ""
						a.homeFilter = homeFilterAll
						a.reorderMode = true
					}
				},
				renderMenu: func(m *ui.Menu) {
					if m.Item(tr.HomeReorder).Icon(menuIconSort).Chosen() {
						a.searchQuery = ""
						a.homeFilter = homeFilterAll
						a.reorderMode = true
					}
				},
			})
		}

		// 3. 刷新
		items = append(items, toolbarItem{
			width: 40,
			render: func(_ bool) {
				if iconButton(c, "btn-refresh", iconRefresh, tr.Refresh, false, false, nil).Clicked() {
					a.reloadHome()
				}
			},
			renderMenu: func(m *ui.Menu) {
				if m.Item(tr.Refresh).Icon(menuIconRefresh).Chosen() {
					a.reloadHome()
				}
			},
		})

		// 4. 打开安装目录
		items = append(items, toolbarItem{
			width: 40,
			render: func(_ bool) {
				if iconButton(c, "btn-open-install", iconFolder, tr.HomeOpenInstallDir, false, false, nil).Clicked() {
					a.openInstallDirectory()
				}
			},
			renderMenu: func(m *ui.Menu) {
				if m.Item(tr.HomeOpenInstallDir).Icon(menuIconFolder).Chosen() {
					a.openInstallDirectory()
				}
			},
		})

		// 5. 扫描当前目录软件
		items = append(items, toolbarItem{
			width: 40,
			render: func(_ bool) {
				if iconButton(c, "btn-scan-dir", iconSearch, tr.HomeScanCurrentDir, false, false, nil).Clicked() {
					a.openScanConfirmModal()
				}
			},
			renderMenu: func(m *ui.Menu) {
				if m.Item(tr.HomeScanCurrentDir).Icon(menuIconSearch).Chosen() {
					a.openScanConfirmModal()
				}
			},
		})

		var totalW float32
		for _, it := range items {
			totalW += it.width
		}

		visibleCount := len(items)
		if totalW > toolbarBudget {
			const moreBtnW = float32(40)
			budget := toolbarBudget - moreBtnW
			var used float32
			visibleCount = 0
			for _, it := range items {
				if used+it.width <= budget {
					used += it.width
					visibleCount++
				} else {
					break
				}
			}
		}

		overflowItems := items[visibleCount:]
		hasMoreBtn := len(overflowItems) > 0
		for i := 0; i < visibleCount; i++ {
			hasNext := i+1 < visibleCount || hasMoreBtn
			items[i].render(hasNext)
		}

		// 当空间不足导致部分工具栏按钮未完全显示时，自动收纳进「...」更多菜单
		if hasMoreBtn {
			moreBtn := ui.ButtonBase(c.Key("btn-home-more")).Size(30, 30).Radius(6).Center().
				Label(tr.HomeMoreActions).Tooltip(tr.HomeMoreActions)
			moreColor := applyHoverIconStyle(moreBtn, t, false, false, nil)
			moreBtn.Children(func() {
				ui.Icon(c, iconMore).FontSize(15).TextColor(moreColor)
			})
			moreBtn.PopoverMenuButton(func(m *ui.Menu) {
				for _, ov := range overflowItems {
					ov.renderMenu(m)
				}
			}, ui.PopoverMenuButtonOptions{
				At:   ui.AnchorBottomEnd,
				Self: ui.AnchorTopEnd,
			})
		}

		// 添加软件主按钮
		if ui.PrimaryButton(c.Key("btn-add-sw"), "").Children(func() {
			ui.Icon(c, iconPlus).FontSize(14)
			ui.Text(c, tr.Add).SingleLine()
		}).Clicked() {
			a.startAddSoftwarePicker()
		}
	})
}

func (a *App) homeFilterBar(c *ui.Context) {
	tr := a.tr()
	if a.homeFilter == "" {
		a.homeFilter = homeFilterAll
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
		sf := ui.SearchField(c.Key("home-search"), &a.searchQuery).
			Placeholder(tr.HomeSearchPlaceholder).Label(tr.HomeSearchPlaceholder).Width(240)
		if a.homeFocusSearch {
			a.homeFocusSearch = false
			sf.Focus()
		}

		ui.Spacer(c)

		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if filterChip(c, "home-filter-all", tr.FilterAll, len(a.managed)+len(a.unmanaged), a.homeFilter == homeFilterAll).Clicked() {
				a.homeFilter = homeFilterAll
			}
			if filterChip(c, "home-filter-managed", tr.FilterManaged, len(a.managed), a.homeFilter == homeFilterManaged).Clicked() {
				a.homeFilter = homeFilterManaged
			}
			if len(a.unmanaged) > 0 {
				if filterChip(c, "home-filter-unmanaged", tr.FilterUnmanaged, len(a.unmanaged), a.homeFilter == homeFilterUnmanaged).Clicked() {
					a.homeFilter = homeFilterUnmanaged
				}
			}
		})
	})
}

func (a *App) visibleHomeItems() (managed, unmanaged []model.Software) {
	if a.homeFilter != homeFilterUnmanaged {
		managed = a.filteredSoftware(a.managed)
	}
	if a.homeFilter != homeFilterManaged {
		unmanaged = a.filteredSoftware(a.unmanaged)
	}
	return managed, unmanaged
}

func (a *App) homeListContentView(c *ui.Context) {
	tr := a.tr()
	managed, unmanaged := a.visibleHomeItems()

	if len(managed) == 0 && len(unmanaged) == 0 {
		if emptySearchView(c, tr.NoMatchingResults, tr.NoMatchingResultsHint, tr.ClearFilter) {
			a.searchQuery = ""
			a.homeFilter = homeFilterAll
		}
		return
	}

	ui.Scroll(c.Key("home-list-scroll")).TrackScroll(&a.homeScrollState).Grow(1).Children(func() {
		ui.Column(c).Gap(8).Padding(2, 4, 16, 0).Children(func() {
			for i := range managed {
				a.softwareListRow(c, managed[i])
			}

			if len(unmanaged) > 0 {
				ui.Row(c).Padding(12, 4, 2, 4).Children(func() {
					sectionTitleRow(c, iconFolder, tr.HomeUnmanagedFolders, len(unmanaged))
				})
				for i := range unmanaged {
					a.softwareListRow(c, unmanaged[i])
				}
			}
		})
	})
}

func (a *App) homeReorderListView(c *ui.Context) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	a.homeListState.Key = func(i int) any { return a.managed[i].ID }
	a.homeListState.Label = func(i int) string { return a.displayNameFor(a.managed[i]) }
	a.homeListState.Reorder = func(rows []int, to int) {
		a.managed = reorderSlice(a.managed, rows, to)
	}

	ui.List(c.Key("home-reorder-list"), &a.homeListState, len(a.managed), func(i int) {
		sw := a.managed[i]
		canUp := i > 0
		canDown := i < len(a.managed)-1

		ui.Row(c.Key(sw.ID)).Padding(10, 14).Margin(0, 4, 6, 0).Radius(10).
			Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 3, 0, p.shadow).
			Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconGrip).FontSize(16).TextColor(t.TextMuted)
			ui.Box(c).Size(36, 36).Radius(9).Background(p.subtleBg).Center().Shrink(0).Children(func() {
				a.renderSoftwareIcon(c, sw, 26)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, a.displayNameFor(sw)).FontSize(14).Bold().SingleLine()
				ui.Text(c, sw.InstallPath).FontSize(11).TextColor(t.TextMuted).TruncateMiddle().SingleLine()
			})
			idx := i
			if iconButton(c, "up-"+sw.ID, iconUp, tr.HomeMoveUp, !canUp, false, nil).Clicked() && idx > 0 {
				a.managed[idx-1], a.managed[idx] = a.managed[idx], a.managed[idx-1]
			}
			if iconButton(c, "down-"+sw.ID, iconDown, tr.HomeMoveDown, !canDown, false, nil).Clicked() && idx < len(a.managed)-1 {
				a.managed[idx+1], a.managed[idx] = a.managed[idx], a.managed[idx+1]
			}
		})
	}).Grow(1)
}

func (a *App) homeGridContentView(c *ui.Context) {
	tr := a.tr()
	managed, unmanaged := a.visibleHomeItems()

	if len(managed) == 0 && len(unmanaged) == 0 {
		if emptySearchView(c, tr.NoMatchingResults, tr.NoMatchingResultsHint, tr.ClearFilter) {
			a.searchQuery = ""
			a.homeFilter = homeFilterAll
		}
		return
	}

	ui.Scroll(c.Key("home-grid-scroll")).TrackScroll(&a.homeScrollState).Grow(1).Children(func() {
		ui.Column(c).Gap(16).Padding(2, 4, 16, 0).Children(func() {
			if len(managed) > 0 {
				ui.Row(c).Wrap().Gap(14).Children(func() {
					for i := range managed {
						a.softwareGridCard(c, managed[i])
					}
				})
			}

			if len(unmanaged) > 0 {
				ui.Row(c).Padding(4, 2, 0, 2).Children(func() {
					sectionTitleRow(c, iconFolder, tr.HomeUnmanagedFolders, len(unmanaged))
				})
				ui.Row(c).Wrap().Gap(14).Children(func() {
					for i := range unmanaged {
						a.softwareGridCard(c, unmanaged[i])
					}
				})
			}
		})
	})
}

func (a *App) softwareListRow(c *ui.Context, sw model.Software) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	title := a.displayNameFor(sw)
	ver := a.versionFor(sw)
	verLabel := formatVersionBadge(ver)
	var subtitle string
	var statusBadge string
	titleColor := t.Text
	iconBoxBg := p.subtleBg

	switch sw.Status {
	case model.StatusManaged:
		if !sw.InstallExists {
			subtitle = tr.HomeSoftwareDirDeleted
			statusBadge = tr.HomeSoftwareDirDeleted
			titleColor = t.Warning
			iconBoxBg = p.warningTint
		} else {
			subtitle = filepath.Base(sw.ExecutablePath)
		}
	case model.StatusUnknownInstall:
		subtitle = tr.HomeUnknownFolder
		statusBadge = tr.HomeUnknownFolder
		titleColor = t.Warning
		iconBoxBg = p.warningTint
	case model.StatusUnknownArchive:
		subtitle = tr.HomeUnknownArchiveFile
		statusBadge = tr.HomeUnknownArchiveFile
		titleColor = t.Warning
		iconBoxBg = p.warningTint
	}

	row := ui.Row(c.Key("row-"+sw.ID)).Padding(12, 16).Radius(11).
		Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 4, 0, p.shadow).
		Gap(14).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(p.cardHover).Border(1, p.borderStrong)
	}

	// 右键菜单
	row.PopoverMenu(func(m *ui.Menu) {
		a.buildSoftwareContextMenu(m, sw)
	})

	row.Children(func() {
		// 左侧可点击启动区
		launchArea := ui.Row(c.Key("launch-" + sw.ID)).Grow(1).MinWidth(0).Gap(14).AlignItems(ui.Center).
			Cursor(ui.CursorPointer)
		launchArea.Children(func() {
			ui.Box(c).Size(42, 42).Radius(10).Background(iconBoxBg).Center().Shrink(0).Children(func() {
				a.renderSoftwareIcon(c, sw, 30)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				ui.Row(c).Gap(8).MinWidth(0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(14).Bold().TextColor(titleColor).SingleLine().MinWidth(0)
					if verLabel != "" {
						versionPillBadge(c, verLabel, i18n.Fmt(tr.HomeVersionTooltipFmt, ver))
					}
					if statusBadge != "" {
						pillBadge(c, statusBadge, p.warningTint, t.Warning)
					}
					if sw.Status == model.StatusManaged {
						ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
							// 归档状态徽章（绿/红）
							archColor := t.Danger
							archTip := tr.TileArchiveNotExists
							if sw.ArchiveExists {
								archColor = t.Success
								archTip = tr.TileArchiveExists
							}
							statusPill(c, iconArchive, archColor, archTip)

							// 备份状态徽章（蓝/灰）
							bakColor := t.TextMuted
							bakTip := tr.TileBackupNotDetected
							if sw.IsBackupArchive {
								bakColor = t.Accent
								bakTip = tr.TileBackupDetected
							}
							statusPill(c, iconRestore, bakColor, bakTip)
						})
					}
					if len(sw.PathEntries) > 0 {
						pillBadgeWithTip(c, tr.TilePathBadge, p.successTint, t.Success,
							i18n.Fmt(tr.TilePathTooltipFmt, strings.Join(sw.PathEntries, "\n")))
					}
					if strings.TrimSpace(sw.LaunchArgs) != "" {
						pillBadgeWithTip(c, tr.TileArgsBadge, p.accentTint, t.Accent,
							i18n.Fmt(tr.TileLaunchConfigTipFmt, sw.LaunchArgs))
					}
					if len(sw.LaunchEnv) > 0 {
						pillBadgeWithTip(c, tr.TileEnvBadge, p.accentTint, t.Accent,
							winutil.FormatEnvText(sw.LaunchEnv))
					}
				})
				ui.Row(c).Gap(6).MinWidth(0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).SingleLine().Shrink(0)
					if sw.InstallPath != "" {
						ui.Text(c, "·").FontSize(12).TextColor(t.TextMuted).Shrink(0)
						ui.Text(c, sw.InstallPath).FontSize(11).TextColor(t.TextMuted).TruncateMiddle().SingleLine().Grow(1).MinWidth(0)
					}
				})
			})
		})
		if launchArea.Clicked() && sw.Status == model.StatusManaged {
			a.launchSoftware(sw, "")
		}

		// 右侧快捷操作
		ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
			canRehost := (sw.Status == model.StatusUnknownInstall ||
				(sw.Status == model.StatusManaged && !sw.InstallExists)) &&
				(sw.ArchiveExists || sw.IsBackupArchive)

			if sw.Status == model.StatusManaged {
				// 备选程序启动按钮
				var alternatives []string
				altLoading := false
				if sw.InstallExists {
					if alt := a.ensureAltExecutables(sw.InstallPath); alt != nil {
						altLoading = alt.loading
						for _, p := range alt.paths {
							if sw.ExecutablePath == "" || !winutil.SamePath(p, sw.ExecutablePath) {
								alternatives = append(alternatives, p)
							}
						}
					}
				}
				if altLoading {
					ui.Box(c).Size(30, 30).Center().Children(func() {
						ui.Spinner(c).Size(14, 14)
					})
				} else if len(alternatives) > 0 {
					altBtn := ui.ButtonBase(c.Key("alt-"+sw.ID)).Size(30, 30).Radius(6).Center().
						Label(tr.TileLaunchOther).Tooltip(tr.TileLaunchOther)
					altColor := applyHoverIconStyle(altBtn, t, false, false, nil)
					altBtn.Children(func() {
						ui.Icon(c, iconTerminal).FontSize(15).TextColor(altColor)
					})
					altsCopy := slices.Clone(alternatives)
					swCopy := sw
					altBtn.PopoverMenuButton(func(m *ui.Menu) {
						for _, altPath := range altsCopy {
							if m.Item(filepath.Base(altPath)).Icon(menuIconTerminal).ID(altPath).Chosen() {
								a.launchSoftware(swCopy, altPath)
							}
						}
					}, ui.PopoverMenuButtonOptions{
						At:   ui.AnchorBottomEnd,
						Self: ui.AnchorTopEnd,
					})
				} else {
					ui.Box(c).Size(30, 30)
				}

				// 重新托管 或 更改主程序
				if canRehost {
					green := t.Success
					if iconButton(c, "rehost-"+sw.ID, iconRefresh, tr.HomeRehost, false, false, &green).Clicked() {
						a.startRehostSoftware(sw)
					}
				} else {
					canChange := !altLoading && len(alternatives) > 0
					tip := tr.HomeChangeExecutable
					if altLoading {
						tip = tr.HomeScanning
					} else if !sw.InstallExists {
						tip = tr.HomeSoftwareDirDeleted
					} else if !canChange {
						tip = tr.HomeOnlyOneExecutableFound
					}
					if iconButton(c, "edit-exe-"+sw.ID, iconEdit, tip, !canChange, false, nil).Clicked() {
						a.openChangeExecutableModal(sw)
					}
				}
			} else if canRehost {
				green := t.Success
				if iconButton(c, "rehost-"+sw.ID, iconRefresh, tr.HomeRehost, false, false, &green).Clicked() {
					a.startRehostSoftware(sw)
				}
			}

			if sw.Status == model.StatusManaged {
				if iconButton(c, "sw-set-"+sw.ID, iconSliders, tr.TileSoftwareSettings, false, false, nil).Clicked() {
					a.openSoftwareSettingsModal(sw)
				}
			}

			// 快捷更多菜单按钮（重命名文件夹 / 创建备份 / 开始菜单快捷方式等）
			rowMoreBtn := ui.ButtonBase(c.Key("row-more-"+sw.ID)).Size(30, 30).Radius(6).Center().
				Label(tr.HomeMoreActions).Tooltip(tr.HomeMoreActions)
			rowMoreColor := applyHoverIconStyle(rowMoreBtn, t, false, false, nil)
			rowMoreBtn.Children(func() {
				ui.Icon(c, iconMore).FontSize(15).TextColor(rowMoreColor)
			})
			swCopy := sw
			rowMoreBtn.PopoverMenuButton(func(m *ui.Menu) {
				a.buildSoftwareContextMenu(m, swCopy)
			}, ui.PopoverMenuButtonOptions{
				At:   ui.AnchorBottomEnd,
				Self: ui.AnchorTopEnd,
			})

			// 删除按钮
			if iconButton(c, "del-"+sw.ID, iconTrash, tr.Delete, false, true, nil).Clicked() {
				a.openDeleteSoftwareModal(sw)
			}
		})
	})
}

func (a *App) softwareGridCard(c *ui.Context, sw model.Software) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()

	title := a.displayNameFor(sw)
	ver := a.versionFor(sw)
	verLabel := formatVersionBadge(ver)
	titleColor := t.Text
	iconMedBg := p.subtleBg
	var badgeText string
	if sw.Status == model.StatusManaged && !sw.InstallExists {
		titleColor = t.Warning
		iconMedBg = p.warningTint
		badgeText = tr.HomeSoftwareDirDeleted
	} else if sw.Status == model.StatusUnknownInstall {
		titleColor = t.Warning
		iconMedBg = p.warningTint
		badgeText = tr.HomeUnknownFolder
	}

	card := ui.Column(c.Key("grid-"+sw.ID)).Size(176, 174).Padding(14, 14).Radius(12).
		Background(p.card).Border(1, p.cardBorder).Shadow(0, 2, 8, 0, p.shadow).
		Center().Gap(8).Clip().Cursor(ui.CursorPointer)
	cardHovered := card.Hovered()
	if cardHovered {
		card.Background(p.cardHover).Border(1, t.Accent.Alpha(0.55)).
			Shadow(0, 6, 16, 0, p.shadowHover)
	}
	card.PopoverMenu(func(m *ui.Menu) {
		a.buildSoftwareContextMenu(m, sw)
	})
	card.Children(func() {
		ui.Box(c).Size(56, 56).Radius(14).Background(iconMedBg).Center().Shrink(0).Children(func() {
			a.renderSoftwareIcon(c, sw, 40)
		})
		ui.Column(c).FillWidth().Center().Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13).Bold().TextColor(titleColor).TextAlign(ui.Center).MaxLines(2).FillWidth()
			if badgeText == "" && verLabel != "" {
				ui.Text(c, verLabel).FontSize(11).TextColor(t.TextMuted).SingleLine().TextAlign(ui.Center).FillWidth().
					Tooltip(i18n.Fmt(tr.HomeVersionTooltipFmt, ver))
			}
		})
		if badgeText != "" {
			pillBadge(c, badgeText, p.warningTint, t.Warning)
		} else if sw.Status == model.StatusManaged {
			ui.Row(c).Gap(6).Center().Children(func() {
				archColor := t.Danger
				archTip := tr.TileArchiveNotExists
				if sw.ArchiveExists {
					archColor = t.Success
					archTip = tr.TileArchiveExists
				}
				statusMiniPill(c, iconArchive, archColor, archTip)

				bakColor := t.TextMuted
				bakTip := tr.TileBackupNotDetected
				if sw.IsBackupArchive {
					bakColor = t.Accent
					bakTip = tr.TileBackupDetected
				}
				statusMiniPill(c, iconRestore, bakColor, bakTip)

				if len(sw.PathEntries) > 0 {
					statusMiniPill(c, iconPathEnv, t.Success, i18n.Fmt(tr.TilePathTooltipFmt, strings.Join(sw.PathEntries, "\n")))
				}
			})
		}

		// 右上角快捷更多按钮：常驻于树中以免弹出菜单时卸载锚点，非悬停时隐藏
		moreBtn := ui.ButtonBase(c.Key("grid-more-"+sw.ID)).Size(24, 24).Radius(6).
			Absolute().Top(6).Right(6).Center().Label(tr.HomeMoreActions).Tooltip(tr.HomeMoreActions)
		if !cardHovered && !moreBtn.Hovered() && !moreBtn.Pressed() {
			moreBtn.Opacity(0)
		}
		moreColor := applyHoverIconStyle(moreBtn, t, false, false, nil)
		moreBtn.Children(func() {
			ui.Icon(c, iconMore).FontSize(14).TextColor(moreColor)
		})
		swCopy := sw
		moreBtn.PopoverMenuButton(func(m *ui.Menu) {
			a.buildSoftwareContextMenu(m, swCopy)
		}, ui.PopoverMenuButtonOptions{
			At:   ui.AnchorBottomEnd,
			Self: ui.AnchorTopEnd,
		})
	})
	if card.Clicked() && sw.Status == model.StatusManaged {
		a.launchSoftware(sw, "")
	}
}

func statusMiniPill(c *ui.Context, ico *ui.SVG, color ui.Color, tooltip string) {
	ui.Box(c).Size(24, 18).Radius(5).
		Background(color.Alpha(0.12)).Border(1, color.Alpha(0.32)).
		Center().Tooltip(tooltip).Children(func() {
		ui.Icon(c, ico).FontSize(11).TextColor(color)
	})
}

func statusPill(c *ui.Context, ico *ui.SVG, color ui.Color, tooltip string) {
	ui.Box(c).Height(20).Padding(0, 6).Radius(5).Shrink(0).
		Background(color.Alpha(0.12)).Border(1, color.Alpha(0.36)).
		Center().Tooltip(tooltip).Children(func() {
		ui.Icon(c, ico).FontSize(11).TextColor(color)
	})
}

func (a *App) buildSoftwareContextMenu(m *ui.Menu, sw model.Software) {
	tr := a.tr()
	swCopy := sw

	if swCopy.Status == model.StatusManaged && swCopy.InstallExists && swCopy.ExecutablePath != "" {
		if m.Item(tr.Open).Icon(menuIconPlay).Chosen() {
			a.launchSoftware(swCopy, "")
		}
		m.Separator()
	}

	if m.Item(tr.TileOpenFolder).Icon(menuIconFolder).Disabled(swCopy.InstallPath == "").Chosen() {
		a.openSoftwareInstallDir(swCopy)
	}
	if swCopy.Status == model.StatusManaged {
		if m.Item(tr.TileSoftwareSettings).Icon(menuIconSliders).Chosen() {
			a.openSoftwareSettingsModal(swCopy)
		}
		if len(swCopy.PathEntries) > 0 {
			if m.Item(tr.TileRemoveFromPath).Icon(menuIconPathEnv).Chosen() {
				a.quickRemoveFromUserPath(swCopy)
			}
		} else {
			if m.Item(tr.TileAddToPath).Icon(menuIconPathEnv).Disabled(!swCopy.InstallExists).Chosen() {
				a.quickAddToUserPath(swCopy)
			}
		}
		m.Separator()
		if m.Item(tr.TileChangeFolder).Icon(menuIconEdit).Disabled(!swCopy.InstallExists).Chosen() {
			a.openRenameFolderModal(swCopy)
		}
		if m.Item(tr.TileCreateBackup).Icon(menuIconRestore).Disabled(!swCopy.InstallExists).Chosen() {
			a.runCreateBackupForSoftware(swCopy)
		}
		if m.Item(tr.TileAddToStartMenu).Icon(menuIconPin).Disabled(swCopy.ExecutablePath == "" || !swCopy.InstallExists).Chosen() {
			a.runAddToStartMenu(swCopy)
		}
		if m.Item(tr.TileOpenArchiveFolder).Icon(menuIconArchive).Disabled(swCopy.ArchivePath == "").Chosen() {
			a.openSoftwareArchiveDir(swCopy)
		}
		if !swCopy.InstallExists && (swCopy.ArchiveExists || swCopy.IsBackupArchive) {
			if m.Item(tr.HomeRehost).Icon(menuIconRefresh).Chosen() {
				a.startRehostSoftware(swCopy)
			}
		} else if swCopy.InstallExists {
			if m.Item(tr.HomeChangeExecutable).Icon(menuIconTerminal).Chosen() {
				a.openChangeExecutableModal(swCopy)
			}
		}
	}
	m.Separator()
	if m.Item(tr.Delete).Icon(menuIconTrash).Danger().Chosen() {
		a.openDeleteSoftwareModal(swCopy)
	}
}

// --- 主页动作方法 ---

func (a *App) startAddSoftwarePicker() {
	if a.settingsSvc.GetInstallPath() == "" {
		a.showInfo(a.tr().HomeInstallDirNotConfig, a.tr().HomeInstallDirNotConfigMsg)
		return
	}
	exts := append(slices.Clone(archive.PickerExtensions), a.settingsSvc.GetExecutableExtensions()...)
	win := a.win
	title := a.tr().Add

	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   win,
			Title:    title,
			Multiple: true,
			Filters: []mygo.FileFilter{
				{Name: "Archives & Executables", Extensions: exts},
			},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.update(func() {
			a.addFilesSequentially(paths)
		})
	}()
}

func (a *App) handleDroppedPaths(paths []string) {
	if a.settingsSvc.GetInstallPath() == "" {
		a.showInfo(a.tr().HomeInstallDirNotConfig, a.tr().HomeInstallDirNotConfigMsg)
		return
	}
	allowedExecs := a.settingsSvc.GetExecutableExtensions()
	var valid []string
	seen := make(map[string]bool)

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}
		key := strings.ToLower(filepath.Clean(p))
		if seen[key] {
			continue
		}
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
		if archive.DetectFormat(p) != archive.FormatUnknown || slices.Contains(allowedExecs, ext) {
			seen[key] = true
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		a.showInfo(a.tr().HomeUnrecognizedDropTitle, a.tr().HomeUnrecognizedDropMsg)
		return
	}
	a.addFilesSequentially(valid)
}

func (a *App) addFilesSequentially(paths []string) {
	if len(paths) == 0 {
		return
	}
	// 先处理首个文件（如遇多可执行文件/重复冲突将弹出交互弹窗）
	a.runAddFile(paths[0], false)
}

func (a *App) runAddFile(filePath string, fromArchiveManager bool) {
	a.busyOpen = true
	a.busyMsg = a.tr().HomeProcessingFiles

	go func() {
		res := a.softwareSvc.AddSoftwareFromFile(filePath)
		a.update(func() {
			a.busyOpen = false
			a.handleAddResult(res, fromArchiveManager)
		})
	}()
}

func (a *App) handleAddResult(res software.AddResult, fromArchiveManager bool) {
	tr := a.tr()
	switch res.Type {
	case software.AddSuccess:
		a.toast(tr.Success, "")
		if fromArchiveManager {
			a.reloadArchives()
		} else {
			a.reloadHome()
		}
	case software.AddNeedsSelection:
		if res.PendingAddition == nil {
			return
		}
		a.selectExeMode = "add"
		a.selectExePending = res.PendingAddition
		a.selectExePaths = res.PendingAddition.ExecutablePaths
		if len(a.selectExePaths) > 0 {
			a.selectExeChoice = a.selectExePaths[0]
		}
		a.selectExeOpen = true
	case software.AddDuplicate:
		if res.DuplicateInfo == nil {
			return
		}
		if res.DuplicateInfo.SourceType == software.SourceExecutable && res.DuplicateInfo.SourcePath != "" {
			delete(a.exeCache, strings.ToLower(filepath.Clean(res.DuplicateInfo.SourcePath)))
		}
		a.dupInfo = res.DuplicateInfo
		if res.DuplicateInfo.IsVersionUpdate && !res.DuplicateInfo.TargetPathCollides {
			a.dupRename = res.DuplicateInfo.IntendedName
		} else {
			a.dupRename = a.suggestAlternativeName(res.DuplicateInfo.IntendedName)
		}
		a.dupFromArchiveManager = fromArchiveManager
		a.dupModalOpen = true
	case software.AddError:
		if errors.Is(res.Err, software.ErrInstallPathNotConfigured) {
			a.showInfo(tr.HomeInstallDirNotConfig, tr.HomeInstallDirNotConfigMsg)
		} else if res.Err != nil {
			a.showInfo(tr.Error, res.Err.Error())
		}
	}
}

func (a *App) suggestAlternativeName(baseName string) string {
	tr := a.tr()
	if strings.TrimSpace(baseName) == "" {
		return fmt.Sprintf("%s%d", tr.HomeNewSoftware, time.Now().UnixMilli())
	}
	if !strings.HasSuffix(baseName, tr.HomeCopySuffix) {
		return baseName + tr.HomeCopySuffix
	}
	return fmt.Sprintf("%s_%d", baseName, time.Now().Unix()%10000)
}

func (a *App) startRehostSoftware(sw model.Software) {
	a.busyOpen = true
	a.busyMsg = a.tr().HomeRestoring

	go func() {
		res := a.softwareSvc.RehostSoftware(sw)
		a.update(func() {
			a.busyOpen = false
			switch res.Type {
			case software.AddSuccess:
				a.toast(a.tr().Success, sw.Name)
				a.reloadHome()
			case software.AddNeedsSelection:
				if res.PendingAddition != nil {
					a.selectExeMode = "rehost"
					a.selectExePending = res.PendingAddition
					a.selectExePaths = res.PendingAddition.ExecutablePaths
					if len(a.selectExePaths) > 0 {
						a.selectExeChoice = a.selectExePaths[0]
					}
					a.selectExeOpen = true
				}
			case software.AddError:
				if res.Err != nil {
					a.showInfo(a.tr().Error, res.Err.Error())
				}
			}
		})
	}()
}

func (a *App) openChangeExecutableModal(sw model.Software) {
	execs := a.softwareSvc.FindExecutablesInDirectory(sw.InstallPath)
	if len(execs) <= 1 {
		a.showInfo(a.tr().HomeChangeExecutable, a.tr().HomeOnlyOneExecutableFound)
		return
	}
	a.selectExeMode = "change"
	a.selectExeTargetID = sw.ID
	a.selectExePaths = execs
	a.selectExeChoice = sw.ExecutablePath
	if a.selectExeChoice == "" {
		a.selectExeChoice = execs[0]
	}
	a.selectExeOpen = true
}

func (a *App) launchSoftware(sw model.Software, altExePath string) {
	tr := a.tr()
	target := sw.ExecutablePath
	if altExePath != "" {
		target = altExePath
	}
	if st, err := os.Stat(sw.InstallPath); err != nil || !st.IsDir() {
		if sw.ArchiveExists || sw.IsBackupArchive {
			a.showInfo(tr.HomeCannotLaunchTitle, tr.HomeCannotLaunchMissingDir)
		} else {
			a.showInfo(tr.HomeCannotLaunchTitle, tr.HomeCannotLaunchNoArchive)
		}
		return
	}
	if sw.Status != model.StatusManaged || target == "" {
		a.showInfo(tr.HomeCannotLaunchTitle, tr.HomeCannotLaunchNoExe)
		return
	}
	logger.Info("启动软件: %s -> %s (参数=%q, 环境变量=%d项)", sw.Name, target, sw.LaunchArgs, len(sw.LaunchEnv))
	opts := winutil.LaunchOptions{
		InstallDir: sw.InstallPath,
		Env:        sw.LaunchEnv,
	}
	if altExePath == "" {
		opts.Args = sw.LaunchArgs
	}
	if err := winutil.LaunchExecutableWithOptions(target, sw.InstallPath, opts); err != nil {
		logger.Error("启动软件失败 (%s): %v", sw.Name, err)
		a.showInfo(tr.HomeLaunchFailedTitle, err.Error())
	}
}

func (a *App) openInstallDirectory() {
	tr := a.tr()
	installPath := a.settingsSvc.GetInstallPath()
	if installPath == "" {
		a.showInfo(tr.HomeInstallDirNotConfig, tr.HomeInstallDirNotConfigMsg)
		return
	}
	if st, err := os.Stat(installPath); err != nil || !st.IsDir() {
		a.showInfo(tr.HomeDirNotExistTitle, installPath)
		return
	}
	a.openDirectoryAsync(installPath)
}

func (a *App) openSoftwareInstallDir(sw model.Software) {
	tr := a.tr()
	if sw.InstallPath == "" {
		a.toast(tr.Warning, tr.TileNoInstallDir)
		return
	}
	if st, err := os.Stat(sw.InstallPath); err != nil || !st.IsDir() {
		a.toast(tr.Warning, sw.InstallPath)
		return
	}
	a.openDirectoryAsync(sw.InstallPath)
}

func (a *App) openSoftwareArchiveDir(sw model.Software) {
	tr := a.tr()
	if sw.ArchivePath == "" {
		a.toast(tr.Warning, tr.TileNoArchive)
		return
	}
	go func(target string) {
		if err := winutil.RevealInExplorer(target); err != nil {
			logger.Warn("在资源管理器中显示归档失败 (%s): %v", target, err)
			a.update(func() {
				if errors.Is(err, os.ErrNotExist) {
					a.toast(a.tr().Warning, a.tr().TileArchiveNotFound)
					return
				}
				a.logAndToastError(a.tr().Error, err)
			})
		}
	}(sw.ArchivePath)
}

func (a *App) openRenameFolderModal(sw model.Software) {
	a.renameTarget = sw
	a.renameInput = filepath.Base(sw.InstallPath)
	a.renameModalOpen = true
}

func (a *App) openSoftwareSettingsModal(sw model.Software) {
	candidates, defaults := a.softwareSvc.SuggestPathCandidates(sw)
	checked := make(map[string]bool, len(candidates))
	for _, d := range defaults {
		norm := winutil.NormalizePathEntry(d)
		if norm != "" {
			checked[strings.ToLower(norm)] = true
		}
	}
	a.swSetTarget = sw
	a.swSetArgs = sw.LaunchArgs
	a.swSetEnvText = winutil.FormatEnvText(sw.LaunchEnv)
	a.swSetEnvError = ""
	a.swSetPathCandidates = candidates
	a.swSetPathChecked = checked
	a.swSetPathInSystem = make(map[string]bool)
	a.swSetPathBusy = false
	a.swSettingsModalOpen = true
	a.refreshSoftwareUserPathPresence()
}

func (a *App) refreshSoftwareUserPathPresence() {
	dirs := slices.Clone(a.swSetPathCandidates)
	if len(dirs) == 0 {
		a.swSetPathLoading = false
		return
	}
	a.swSetPathLoading = true
	targetID := a.swSetTarget.ID
	go func() {
		presence, err := a.softwareSvc.QuerySoftwareUserPathPresence(dirs)
		a.update(func() {
			if a.swSetTarget.ID != targetID {
				return
			}
			a.swSetPathLoading = false
			if err == nil && presence != nil {
				a.swSetPathInSystem = presence
			}
		})
	}()
}

func (a *App) quickAddToUserPath(sw model.Software) {
	_, defaultDirs := a.softwareSvc.SuggestPathCandidates(sw)
	if len(defaultDirs) == 0 {
		a.openSoftwareSettingsModal(sw)
		return
	}
	go func() {
		synced, err := a.softwareSvc.RegisterSoftwarePath(sw.ID, defaultDirs)
		a.update(func() {
			tr := a.tr()
			if err != nil {
				a.showInfo(tr.Error, err.Error())
				return
			}
			a.toast(tr.Success, i18n.Fmt(tr.SwSetPathRegisteredFmt, len(synced)))
			a.reloadHome()
		})
	}()
}

func (a *App) quickRemoveFromUserPath(sw model.Software) {
	go func() {
		_, err := a.softwareSvc.UnregisterSoftwarePath(sw.ID)
		a.update(func() {
			tr := a.tr()
			if err != nil {
				a.showInfo(tr.Error, err.Error())
				return
			}
			a.toast(tr.Success, i18n.Fmt(tr.SwSetPathRemovedFmt, a.displayNameFor(sw)))
			a.reloadHome()
		})
	}()
}

func (a *App) openDeleteSoftwareModal(sw model.Software) {
	a.deleteTarget = sw
	a.deleteIsArchiveMgr = false
	a.deleteIsBackup = false
	a.deleteInstallDir = sw.Status == model.StatusUnknownInstall
	a.deleteArchive = sw.Status == model.StatusUnknownArchive
	a.deleteModalOpen = true
}

func (a *App) runCreateBackupForSoftware(sw model.Software) {
	tr := a.tr()
	a.busyOpen = true
	a.busyMsg = tr.TileCreatingBackup

	go func() {
		created, err := a.softwareSvc.CreateBackupForSoftware(sw)
		a.update(func() {
			a.busyOpen = false
			if err != nil {
				a.showInfo(tr.Error, err.Error())
				return
			}
			a.toast(tr.Success, i18n.Fmt(tr.TileBackupCreatedFmt, filepath.Base(created)))
			a.reloadHome()
			a.reloadArchives()
		})
	}()
}

func (a *App) runAddToStartMenu(sw model.Software) {
	tr := a.tr()
	displayName := a.displayNameFor(sw)
	if sw.ExecutablePath == "" {
		a.toast(tr.Warning, tr.TileNoExecutable)
		return
	}

	go func() {
		startMenuDir, err := winutil.StartMenuDir()
		if err != nil {
			a.update(func() { a.showInfo(tr.Error, i18n.Fmt(tr.TileAddToStartMenuErrFmt, err.Error())) })
			return
		}
		// 若开始菜单内尚无绿驿管家自身快捷方式，一并补建
		if selfExe, exeErr := os.Executable(); exeErr == nil {
			selfLnk := filepath.Join(startMenuDir, tr.AppName+".lnk")
			if _, statErr := os.Stat(selfLnk); errors.Is(statErr, os.ErrNotExist) {
				_ = winutil.CreateShortcut(selfExe, selfLnk, tr.AppName)
			}
		}

		safeName := software.SanitizeSoftwareName(displayName)
		lnkPath := filepath.Join(startMenuDir, safeName+".lnk")
		err = winutil.CreateShortcutWithArgs(sw.ExecutablePath, lnkPath, displayName, sw.LaunchArgs)
		a.update(func() {
			if err != nil {
				a.showInfo(tr.Error, i18n.Fmt(tr.TileAddToStartMenuErrFmt, err.Error()))
				return
			}
			a.toast(tr.Success, i18n.Fmt(tr.TileAddedToStartMenuFmt, displayName))
		})
	}()
}

func (a *App) saveCurrentOrder() {
	ids := make([]string, len(a.managed))
	for i, s := range a.managed {
		ids[i] = s.ID
	}
	go func() {
		err := a.softwareSvc.UpdateManagedSoftwareOrder(ids)
		a.update(func() {
			a.reorderMode = false
			if err != nil {
				a.logAndToastError(a.tr().Error, err)
				return
			}
			a.toast(a.tr().Success, a.tr().HomeReorderSave)
			a.reloadHome()
		})
	}()
}

func (a *App) openScanConfirmModal() {
	if a.settingsSvc.GetInstallPath() == "" {
		a.showInfo(a.tr().HomeInstallDirNotConfig, a.tr().HomeInstallDirNotConfigMsg)
		return
	}
	backupDir, _ := a.softwareSvc.ResolveBackupDirectoryPath()
	a.scanBackupDirText = backupDir
	a.scanCreateBackup = true
	a.scanConfirmOpen = true
}

func (a *App) startBatchScan() {
	createBackup := a.scanCreateBackup
	a.scanConfirmOpen = false
	a.scanProgressOpen = true
	a.scanDone = 0
	a.scanTotal = 0
	a.scanCurrent = ""

	go func() {
		summary, err := a.softwareSvc.ArchiveInstallSubdirectories(createBackup, func(done, total int, current string) {
			a.update(func() {
				a.scanDone = done
				a.scanTotal = total
				a.scanCurrent = current
			})
		})
		a.update(func() {
			a.scanProgressOpen = false
			if err != nil {
				a.showInfo(a.tr().Error, err.Error())
				return
			}
			tr := a.tr()
			var sb strings.Builder
			sb.WriteString(i18n.Fmt(tr.ScanSummaryInstallFmt, summary.InstallDirPath) + "\n")
			sb.WriteString(i18n.Fmt(tr.ScanSummaryArchiveFmt, summary.ArchiveDirPath) + "\n")
			sb.WriteString(i18n.Fmt(tr.ScanSummaryTotalFmt, summary.Total) + "\n")
			sb.WriteString(i18n.Fmt(tr.ScanSummaryArchivedFmt, len(summary.Archived)) + "\n")
			sb.WriteString(i18n.Fmt(tr.ScanSummarySkippedFmt, len(summary.SkippedExisting)) + "\n")
			sb.WriteString(i18n.Fmt(tr.ScanSummaryFailedFmt, len(summary.Failures)))
			if len(summary.Failures) > 0 {
				sb.WriteString("\n")
				limit := min(5, len(summary.Failures))
				for i := range limit {
					f := summary.Failures[i]
					sb.WriteString(fmt.Sprintf("- %s: %s\n", f.Name, f.Error))
				}
				if len(summary.Failures) > 5 {
					sb.WriteString(i18n.Fmt(tr.ScanSummaryMoreFmt, len(summary.Failures)-5))
				}
			}

			if len(summary.Suggestions) > 0 {
				a.assocSuggestions = summary.Suggestions
				a.assocEnabled = make(map[string]bool, len(summary.Suggestions))
				a.assocSelected = make(map[string]string, len(summary.Suggestions))
				for _, s := range summary.Suggestions {
					a.assocEnabled[s.InstallPath] = true
					if len(s.Candidates) > 0 {
						a.assocSelected[s.InstallPath] = s.Candidates[0]
					}
				}
				a.assocCreateBackup = createBackup
				a.assocSummaryText = sb.String()
				a.assocModalOpen = true
			} else {
				a.showInfo(tr.ScanSummaryTitle, sb.String())
				a.reloadHome()
			}
		})
	}()
}

// reorderSlice 将 rows 中的元素按原相对顺序移动到位置 to 之前（兼容 MyGo ListState.Reorder 语义）。
func reorderSlice[T any](items []T, rows []int, to int) []T {
	if len(rows) == 0 || len(items) == 0 {
		return items
	}
	inRows := make(map[int]bool, len(rows))
	for _, r := range rows {
		inRows[r] = true
	}
	var moved, rest []T
	insertAt := to
	for i, item := range items {
		if inRows[i] {
			moved = append(moved, item)
			if i < to {
				insertAt--
			}
		} else {
			rest = append(rest, item)
		}
	}
	if insertAt < 0 {
		insertAt = 0
	}
	if insertAt > len(rest) {
		insertAt = len(rest)
	}
	out := make([]T, 0, len(items))
	out = append(out, rest[:insertAt]...)
	out = append(out, moved...)
	out = append(out, rest[insertAt:]...)
	return out
}
