package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"carrydock/internal/i18n"
	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/software"
	"github.com/egoist/mygo/ui"
)

func (a *App) modalsView(c *ui.Context) {
	a.busyModalView(c)
	a.infoModalView(c)
	a.selectExecutableModalView(c)
	a.duplicateModalView(c)
	a.deleteModalView(c)
	a.renameFolderModalView(c)
	a.scanConfirmModalView(c)
	a.scanProgressModalView(c)
	a.associationModalView(c)
	a.manualLinkModalView(c)
	a.createBackupModalView(c)
	a.extensionsModalView(c)
	a.unsavedSettingsModalView(c)
	a.updateModalView(c)
	a.reinstallConfirmModalView(c)
	a.devIconTestModalView(c)
}

func (a *App) busyModalView(c *ui.Context) {
	if !a.busyOpen {
		return
	}
	if ui.Modal(c.Key("modal-busy"), &a.busyOpen, func() {
		ui.Row(c).Gap(12).Padding(6, 4).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(22, 22)
			ui.Text(c, a.busyMsg).FontSize(14)
		})
	}).Dismissed() {
		// 忙碌进度弹窗禁止外部点击或 Escape 意外关闭
		a.busyOpen = true
	}
}

func (a *App) infoModalView(c *ui.Context) {
	if !a.infoModalOpen {
		return
	}
	tr := a.tr()
	ui.Modal(c.Key("modal-info"), &a.infoModalOpen, func() {
		ui.Column(c).Width(460).Gap(12).Children(func() {
			ui.Text(c, a.infoModalTitle).FontSize(16).Bold()
			ui.Scroll(c.Key("modal-info-scroll")).MaxHeight(320).Children(func() {
				ui.Text(c, a.infoModalBody).FontSize(13).Selectable()
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c.Key("modal-info-ok"), tr.OK).Clicked() {
					a.infoModalOpen = false
				}
			})
		})
	})
}

func (a *App) selectExecutableModalView(c *ui.Context) {
	if !a.selectExeOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()

	cancelAction := func() {
		a.selectExeOpen = false
		if a.selectExeMode == "add" && a.selectExePending != nil {
			pending := *a.selectExePending
			a.selectExePending = nil
			go func() {
				a.softwareSvc.CancelPendingAddition(pending)
			}()
		} else if a.selectExeMode == "rehost" && a.selectExePending != nil {
			pending := *a.selectExePending
			a.selectExePending = nil
			// 仅清理临时解压的安装目录，不删除源归档
			pending.CreatedArchive = false
			go func() {
				a.softwareSvc.CancelPendingAddition(pending)
			}()
		}
	}

	if ui.Modal(c.Key("modal-select-exe"), &a.selectExeOpen, func() {
		ui.Column(c).Width(500).Gap(12).Children(func() {
			ui.Text(c, tr.DialogSelectExecutable).FontSize(16).Bold()
			ui.Scroll(c.Key("modal-exe-scroll")).MaxHeight(320).Children(func() {
				ui.RadioGroup(c, func() {
					for _, path := range a.selectExePaths {
						pCopy := path
						meta := a.ensureExeMeta(pCopy)
						label := filepath.Base(pCopy)
						if meta != nil && meta.desc != "" {
							label = fmt.Sprintf("%s  (%s)", meta.desc, filepath.Base(pCopy))
						}
						ui.Row(c.Key("exe-opt-"+pCopy)).Padding(6, 8).Radius(6).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Radio(c, &a.selectExeChoice, pCopy, "")
							if meta != nil && meta.icon != nil {
								ui.Image(c, meta.icon).Size(22, 22).Shrink(0).Fit(ui.Contain)
							} else {
								ui.Icon(c, iconAppDefault).FontSize(18).TextColor(t.TextMuted)
							}
							ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
								ui.Text(c, label).FontSize(13).SingleLine()
								ui.Text(c, pCopy).FontSize(11).TextColor(t.TextMuted).SingleLine()
							})
						})
					}
				})
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("exe-cancel"), tr.Cancel).Clicked() {
					cancelAction()
				}
				if ui.PrimaryButton(c.Key("exe-confirm"), tr.Confirm).Disabled(a.selectExeChoice == "").Clicked() {
					chosen := a.selectExeChoice
					mode := a.selectExeMode
					pending := a.selectExePending
					targetID := a.selectExeTargetID
					a.selectExeOpen = false
					a.selectExePending = nil
					a.busyOpen = true
					a.busyMsg = tr.HomeInstallingSoftware

					go func() {
						var err error
						switch mode {
						case "add":
							if pending != nil {
								err = a.softwareSvc.CompleteSoftwareAddition(*pending, chosen)
							}
						case "rehost":
							if pending != nil {
								err = a.softwareSvc.CompleteSoftwareRehost(pending.ExistingSoftwareID, pending.InstallPath, pending.ArchivePath, chosen)
							}
						case "change":
							err = a.softwareSvc.UpdateSoftwareExecutable(targetID, chosen)
						}
						a.update(func() {
							a.busyOpen = false
							if err != nil {
								a.showInfo(a.tr().Error, err.Error())
								return
							}
							a.toast(a.tr().Success, filepath.Base(chosen))
							if a.page == pageArchive {
								a.reloadArchives()
							} else {
								a.reloadHome()
							}
						})
					}()
				}
			})
		})
	}).Dismissed() {
		cancelAction()
	}
}

func (a *App) duplicateModalView(c *ui.Context) {
	if !a.dupModalOpen || a.dupInfo == nil {
		return
	}
	t := c.Theme()
	tr := a.tr()
	info := *a.dupInfo

	ui.Modal(c.Key("modal-dup"), &a.dupModalOpen, func() {
		ui.Column(c).Width(480).Gap(12).Children(func() {
			ui.Text(c, tr.DupDetected).FontSize(16).Bold()
			ui.Text(c, i18n.Fmt(tr.DupHelpFmt, info.IntendedName)).FontSize(13)

			ui.Column(c).Padding(10, 12).Radius(8).Background(t.Surface).Border(1, t.Border).Gap(4).Children(func() {
				if info.ExistingManagedSoftware != nil {
					ui.Text(c, "• "+i18n.Fmt(tr.DupExistingFmt, info.ExistingManagedSoftware.Name, filepath.Base(info.ExistingManagedSoftware.InstallPath))).FontSize(12)
				}
				if info.InstallDirExists {
					ui.Text(c, "• "+i18n.Fmt(tr.DupInstallDirFmt, info.TargetInstallPath)).FontSize(12)
				}
				if info.ArchiveExists {
					ui.Text(c, "• "+i18n.Fmt(tr.DupArchiveFmt, filepath.Base(info.TargetArchivePath))).FontSize(12)
				}
			})

			ui.Text(c, tr.DupRenameHint).FontSize(13)
			ui.TextInput(c.Key("dup-rename-input"), &a.dupRename).AutoFocus()

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("dup-cancel"), tr.Cancel).Clicked() {
					a.dupModalOpen = false
					a.dupInfo = nil
				}
				if ui.Button(c.Key("dup-rename-btn"), tr.DupRenameAdd).
					Disabled(strings.TrimSpace(a.dupRename) == "").Clicked() {
					newName := strings.TrimSpace(a.dupRename)
					fromMgr := a.dupFromArchiveManager
					a.dupModalOpen = false
					a.dupInfo = nil
					a.busyOpen = true
					a.busyMsg = tr.HomeProcessingFiles
					go func() {
						res := a.softwareSvc.ResolveDuplicateAddition(info, false, newName)
						a.update(func() {
							a.busyOpen = false
							a.handleAddResult(res, fromMgr)
						})
					}()
				}
				if info.InstallDirExists {
					if ui.Button(c.Key("dup-bak-over"), tr.DupBackupOverwrite).Clicked() {
						fromMgr := a.dupFromArchiveManager
						a.dupModalOpen = false
						a.dupInfo = nil
						a.busyOpen = true
						a.busyMsg = tr.HomeRestoring
						go func() {
							if _, err := a.softwareSvc.CreateBackupFromDirectory(info.TargetInstallPath, filepath.Base(info.TargetInstallPath)); err != nil {
								a.update(func() {
									a.busyOpen = false
									a.showInfo(a.tr().Error, err.Error())
								})
								return
							}
							res := a.softwareSvc.ResolveDuplicateAddition(info, true, "")
							a.update(func() {
								a.busyOpen = false
								a.handleAddResult(res, fromMgr)
							})
						}()
					}
				}
				if ui.PrimaryButton(c.Key("dup-overwrite"), tr.DupOverwrite).Clicked() {
					fromMgr := a.dupFromArchiveManager
					a.dupModalOpen = false
					a.dupInfo = nil
					a.busyOpen = true
					a.busyMsg = tr.HomeProcessingFiles
					go func() {
						res := a.softwareSvc.ResolveDuplicateAddition(info, true, "")
						a.update(func() {
							a.busyOpen = false
							a.handleAddResult(res, fromMgr)
						})
					}()
				}
			})
		})
	})
}

func (a *App) deleteModalView(c *ui.Context) {
	if !a.deleteModalOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()
	target := a.deleteTarget

	ui.Modal(c.Key("modal-delete"), &a.deleteModalOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			if a.deleteIsArchiveMgr {
				title := tr.ArchiveDeleteArchive
				hint := tr.ArchiveDeleteArchiveHint
				if a.deleteIsBackup {
					title = tr.ArchiveDeleteBackup
					hint = tr.ArchiveDeleteBackupHint
				}
				ui.Text(c, title).FontSize(16).Bold()
				ui.Text(c, hint).FontSize(13)
				ui.Text(c, target.Name).FontSize(12).TextColor(t.TextMuted).Selectable()
			} else {
				ui.Text(c, i18n.Fmt(tr.DeleteSoftwareFmt, target.Name)).FontSize(16).Bold()
				if target.Status != model.StatusManaged {
					kind := tr.HomeUnknownFolder
					if target.Status == model.StatusUnknownArchive {
						kind = tr.HomeUnknownArchiveFile
					}
					ui.Text(c, i18n.Fmt(tr.DeleteUnmanagedFmt, kind)).FontSize(13)
				} else {
					ui.Text(c, tr.DeleteManagedHint).FontSize(13)
					ui.Checkbox(c.Key("del-cb-dir"), &a.deleteInstallDir, tr.DeleteInstallDir).
						Disabled(!target.InstallExists)
					ui.Checkbox(c.Key("del-cb-arch"), &a.deleteArchive, tr.DeleteArchive).
						Disabled(!target.ArchiveExists)
				}
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("del-cancel"), tr.Cancel).Clicked() {
					a.deleteModalOpen = false
				}
				delBtn := ui.PrimaryButton(c.Key("del-confirm"), tr.Delete).Background(t.Danger)
				if delBtn.Clicked() {
					isMgr := a.deleteIsArchiveMgr
					delDir := a.deleteInstallDir
					delArch := a.deleteArchive
					a.deleteModalOpen = false
					a.busyOpen = true
					a.busyMsg = tr.HomeDeleting

					go func() {
						var err error
						if isMgr {
							err = a.softwareSvc.DeleteArchiveFile(target.ArchivePath)
						} else {
							err = a.softwareSvc.DeleteSoftware(target, delDir, delArch)
						}
						a.update(func() {
							a.busyOpen = false
							if err != nil {
								a.showInfo(a.tr().Error, err.Error())
								return
							}
							a.toast(a.tr().Success, target.Name)
							if isMgr {
								a.reloadArchives()
							} else {
								a.reloadHome()
							}
						})
					}()
				}
			})
		})
	})
}

func (a *App) renameFolderModalView(c *ui.Context) {
	if !a.renameModalOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()
	target := a.renameTarget

	submit := func() {
		newName := strings.TrimSpace(a.renameInput)
		if newName == "" {
			return
		}
		a.renameModalOpen = false
		a.busyOpen = true
		a.busyMsg = tr.TileMigrating

		go func() {
			err := a.softwareSvc.RenameSoftwareFolder(target, newName)
			a.update(func() {
				a.busyOpen = false
				if err != nil {
					a.showInfo(a.tr().Error, i18n.Fmt(a.tr().TileMigrateFailedFmt, err.Error()))
					return
				}
				a.toast(a.tr().Success, a.tr().TileMigrateSuccess)
				a.reloadHome()
			})
		}()
	}

	ui.Modal(c.Key("modal-rename"), &a.renameModalOpen, func() {
		ui.Column(c).Width(440).Gap(10).Children(func() {
			ui.Text(c, tr.TileChangeFolderTitle).FontSize(16).Bold()
			ui.Text(c, tr.TileCurrentPath).FontSize(12).TextColor(t.TextMuted)
			ui.Text(c, target.InstallPath).FontSize(12).Selectable()
			ui.Text(c, tr.TileNewFolderName).FontSize(13).Bold()
			if ui.TextInput(c.Key("rename-input"), &a.renameInput).
				Placeholder(tr.TileNewFolderPlaceholder).AutoFocus().Submitted() {
				submit()
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("rename-cancel"), tr.Cancel).Clicked() {
					a.renameModalOpen = false
				}
				if ui.PrimaryButton(c.Key("rename-ok"), tr.Confirm).
					Disabled(strings.TrimSpace(a.renameInput) == "").Clicked() {
					submit()
				}
			})
		})
	})
}

func (a *App) scanConfirmModalView(c *ui.Context) {
	if !a.scanConfirmOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()
	installPath := a.settingsSvc.GetInstallPath()

	ui.Modal(c.Key("modal-scan-confirm"), &a.scanConfirmOpen, func() {
		ui.Column(c).Width(460).Gap(12).Children(func() {
			ui.Text(c, tr.ScanArchiveTitle).FontSize(16).Bold()
			ui.Text(c, i18n.Fmt(tr.ScanArchiveHintFmt, installPath)).FontSize(13)
			if a.scanCreateBackup {
				ui.Text(c, i18n.Fmt(tr.ScanArchiveBackupOnFmt, a.scanBackupDirText)).FontSize(12).TextColor(t.TextMuted)
			} else {
				ui.Text(c, tr.ScanArchiveBackupOff).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Switch(c.Key("scan-toggle"), &a.scanCreateBackup).Label(tr.ScanArchiveToggle)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, tr.ScanArchiveToggle).FontSize(13).Bold()
					ui.Text(c, tr.ScanArchiveToggleHint).FontSize(11).TextColor(t.TextMuted)
				})
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("scan-cancel"), tr.Cancel).Clicked() {
					a.scanConfirmOpen = false
				}
				if ui.PrimaryButton(c.Key("scan-start"), tr.ScanArchiveStart).Clicked() {
					a.startBatchScan()
				}
			})
		})
	})
}

func (a *App) scanProgressModalView(c *ui.Context) {
	if !a.scanProgressOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()
	progress := -1.0
	percent := 0
	if a.scanTotal > 0 {
		progress = float64(a.scanDone) / float64(a.scanTotal)
		percent = int(progress * 100)
	}
	if ui.Modal(c.Key("modal-scan-progress"), &a.scanProgressOpen, func() {
		ui.Column(c).Width(400).Gap(10).Children(func() {
			ui.Text(c, tr.ScanScanning).FontSize(15).Bold()
			ui.Progress(c, progress)
			ui.Text(c, i18n.Fmt(tr.ScanProgressFmt, a.scanDone, a.scanTotal, percent)).FontSize(12).TextColor(t.TextMuted)
			if a.scanCurrent != "" {
				ui.Text(c, i18n.Fmt(tr.ScanCurrentFmt, a.scanCurrent)).FontSize(12).SingleLine()
			}
		})
	}).Dismissed() {
		a.scanProgressOpen = true
	}
}

func (a *App) associationModalView(c *ui.Context) {
	if !a.assocModalOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()

	ui.Modal(c.Key("modal-assoc"), &a.assocModalOpen, func() {
		ui.Column(c).Width(540).Gap(12).Children(func() {
			ui.Text(c, i18n.Fmt(tr.AssocFoundFmt, len(a.assocSuggestions))).FontSize(16).Bold()
			ui.Text(c, tr.AssocHint).FontSize(13).TextColor(t.TextMuted)

			ui.Scroll(c.Key("assoc-scroll")).MaxHeight(340).Children(func() {
				ui.Column(c).Gap(10).Children(func() {
					for _, s := range a.assocSuggestions {
						enabled := a.assocEnabled[s.InstallPath]
						selectedPath := a.assocSelected[s.InstallPath]

						labels := make([]string, len(s.Candidates))
						labelToPath := make(map[string]string, len(s.Candidates))
						selectedLabel := ""
						for i, cand := range s.Candidates {
							lbl := filepath.Base(cand)
							labels[i] = lbl
							labelToPath[lbl] = cand
							if cand == selectedPath {
								selectedLabel = lbl
							}
						}
						if selectedLabel == "" && len(labels) > 0 {
							selectedLabel = labels[0]
						}

						ui.Column(c.Key("assoc-"+s.InstallPath)).Padding(10, 12).Radius(8).
							Background(t.Surface).Border(1, t.Border).Gap(8).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								if ui.Switch(c.Key("assoc-sw-"+s.InstallPath), &enabled).Label(s.DisplayName).Changed() {
									a.assocEnabled[s.InstallPath] = enabled
								}
								ui.Text(c, s.DisplayName).FontSize(13).Bold().Grow(1).MinWidth(0).SingleLine()
							})
							if ui.Select(c.Key("assoc-sel-"+s.InstallPath), &selectedLabel, labels).
								Disabled(!enabled || len(labels) <= 1).Changed() {
								a.assocSelected[s.InstallPath] = labelToPath[selectedLabel]
							}
						})
					}
				})
			})

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("assoc-skip"), tr.AssocSkipAll).Clicked() {
					summary := a.assocSummaryText
					a.assocModalOpen = false
					a.showInfo(tr.ScanSummaryTitle, summary)
					a.reloadHome()
				}
				if ui.PrimaryButton(c.Key("assoc-apply"), tr.AssocApply).Clicked() {
					resolutions := make([]software.ArchiveAssociationResolution, 0, len(a.assocSuggestions))
					for _, s := range a.assocSuggestions {
						sel := ""
						if a.assocEnabled[s.InstallPath] {
							sel = a.assocSelected[s.InstallPath]
						}
						resolutions = append(resolutions, software.ArchiveAssociationResolution{
							InstallPath:         s.InstallPath,
							SelectedArchivePath: sel,
						})
					}
					createBak := a.assocCreateBackup
					summary := a.assocSummaryText
					a.assocModalOpen = false
					a.busyOpen = true
					a.busyMsg = tr.HomeProcessingFiles

					go func() {
						err := a.softwareSvc.ApplyArchiveAssociations(resolutions, createBak)
						a.update(func() {
							a.busyOpen = false
							if err != nil {
								a.showInfo(a.tr().Error, err.Error())
								return
							}
							a.showInfo(a.tr().ScanSummaryTitle, summary)
							a.reloadHome()
						})
					}()
				}
			})
		})
	})
}

func (a *App) manualLinkModalView(c *ui.Context) {
	if !a.linkModalOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()
	title := tr.ArchiveLinkArchive
	if a.linkIsBackup {
		title = tr.ArchiveLinkBackup
	}

	labels := make([]string, 0, len(a.linkNameToID))
	for k := range a.linkNameToID {
		labels = append(labels, k)
	}
	slices.Sort(labels)

	ui.Modal(c.Key("modal-link"), &a.linkModalOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, title).FontSize(16).Bold()
			ui.Text(c, a.linkTarget.Name).FontSize(12).TextColor(t.TextMuted).SingleLine()
			ui.Select(c.Key("link-sw-sel"), &a.linkSelectedName, labels)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("link-cancel"), tr.Cancel).Clicked() {
					a.linkModalOpen = false
				}
				if ui.PrimaryButton(c.Key("link-ok"), tr.ArchiveLinkAction).Clicked() {
					swID := a.linkNameToID[a.linkSelectedName]
					archPath := a.linkTarget.ArchivePath
					a.linkModalOpen = false
					go func() {
						err := a.softwareSvc.LinkArchiveToSoftware(swID, archPath)
						a.update(func() {
							if err != nil {
								a.showInfo(a.tr().Error, err.Error())
								return
							}
							a.toast(a.tr().Success, a.tr().ArchiveLinkSuccess)
							a.reloadArchives()
						})
					}()
				}
			})
		})
	})
}

func (a *App) createBackupModalView(c *ui.Context) {
	if !a.backupModalOpen {
		return
	}
	tr := a.tr()

	labels := make([]string, 0, len(a.backupNameToID))
	for k := range a.backupNameToID {
		labels = append(labels, k)
	}
	slices.Sort(labels)

	ui.Modal(c.Key("modal-create-backup"), &a.backupModalOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, tr.ArchiveCreateBackup).FontSize(16).Bold()
			ui.Select(c.Key("bak-sw-sel"), &a.backupSelectedName, labels)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("bak-cancel"), tr.Cancel).Clicked() {
					a.backupModalOpen = false
				}
				if ui.PrimaryButton(c.Key("bak-ok"), tr.Add).Clicked() {
					swID := a.backupNameToID[a.backupSelectedName]
					var chosen model.Software
					for _, m := range a.backupManaged {
						if m.ID == swID {
							chosen = m
							break
						}
					}
					a.backupModalOpen = false
					a.busyOpen = true
					a.busyMsg = tr.ArchiveCreatingBackup

					go func() {
						path, err := a.softwareSvc.CreateBackupForSoftware(chosen)
						a.update(func() {
							a.busyOpen = false
							if err != nil {
								a.showInfo(a.tr().Error, err.Error())
								return
							}
							a.toast(a.tr().Success, i18n.Fmt(a.tr().ArchiveBackupCreatedFmt, filepath.Base(path)))
							a.reloadArchives()
						})
					}()
				}
			})
		})
	})
}

func (a *App) extensionsModalView(c *ui.Context) {
	if !a.extModalOpen {
		return
	}
	t := c.Theme()
	tr := a.tr()

	addCustom := func() {
		norm := settings.NormalizeExtensions([]string{a.extCustom})
		if len(norm) == 0 {
			a.extError = tr.SettingsExtInvalid
			return
		}
		v := norm[0]
		if a.extDraft[v] {
			a.extError = tr.SettingsExtExists
			return
		}
		a.extDraft[v] = true
		a.extCustom = ""
		a.extError = ""
	}

	ui.Modal(c.Key("modal-exts"), &a.extModalOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, tr.SettingsSelectExtensions).FontSize(16).Bold()

			ui.Text(c, tr.SettingsCommonExtensions).FontSize(13).Bold()
			ui.Row(c).Wrap().Gap(6).Children(func() {
				for _, ext := range settings.CommonExecutableExtensions {
					extCopy := ext
					checked := a.extDraft[extCopy]
					if ui.Toggle(c.Key("ext-tog-"+extCopy), &checked, "."+extCopy).Changed() {
						if checked {
							a.extDraft[extCopy] = true
						} else {
							delete(a.extDraft, extCopy)
						}
						a.extError = ""
					}
				}
			})

			ui.Text(c, tr.SettingsCustomExtensions).FontSize(13).Bold()
			ui.Row(c).Gap(8).Children(func() {
				if ui.TextInput(c.Key("ext-custom-in"), &a.extCustom).
					Placeholder(tr.SettingsExtPlaceholder).Grow(1).Submitted() {
					addCustom()
				}
				if ui.Button(c.Key("ext-custom-add"), tr.Add).Clicked() {
					addCustom()
				}
			})
			if a.extError != "" {
				ui.Text(c, a.extError).FontSize(12).TextColor(t.Danger)
			}

			// 已选列表
			selected := make([]string, 0, len(a.extDraft))
			for k, on := range a.extDraft {
				if on {
					selected = append(selected, k)
				}
			}
			slices.Sort(selected)
			ui.Row(c).Wrap().Gap(6).Children(func() {
				for _, ext := range selected {
					extCopy := ext
					ui.Row(c.Key("ext-sel-"+extCopy)).Padding(4, 10).Radius(999).
						Background(t.Surface).Border(1, t.Border).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "."+extCopy).FontSize(12)
						rmBtn := ui.ButtonBase(c.Key("ext-del-"+extCopy)).Size(16, 16).Radius(8).Center().
							Label(tr.Remove)
						rmColor := applyHoverIconStyle(rmBtn, t, false, false, nil)
						rmBtn.Children(func() {
							ui.Icon(c, iconX).FontSize(11).TextColor(rmColor)
						})
						if rmBtn.Clicked() {
							delete(a.extDraft, extCopy)
						}
					})
				}
			})

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("ext-cancel"), tr.Cancel).Clicked() {
					a.extModalOpen = false
				}
				if ui.PrimaryButton(c.Key("ext-ok"), tr.Confirm).Clicked() {
					if len(selected) == 0 {
						a.extError = tr.SettingsExtSelectAtLeastOne
						return
					}
					a.setExts = settings.NormalizeExtensions(selected)
					a.extModalOpen = false
				}
			})
		})
	})
}

func (a *App) unsavedSettingsModalView(c *ui.Context) {
	if !a.unsavedModalOpen {
		return
	}
	tr := a.tr()
	ui.Modal(c.Key("modal-unsaved"), &a.unsavedModalOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, tr.SettingsUnsavedTitle).FontSize(16).Bold()
			ui.Text(c, tr.SettingsUnsavedMessage).FontSize(13)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("unsaved-cancel"), tr.Cancel).Clicked() {
					a.unsavedModalOpen = false
				}
				if ui.Button(c.Key("unsaved-discard"), tr.SettingsDiscardChanges).Clicked() {
					a.unsavedModalOpen = false
					a.loadSettingsSnapshot()
					a.navigateNow(a.pendingNavPage)
				}
				if ui.PrimaryButton(c.Key("unsaved-save"), tr.SettingsSaveAndContinue).Clicked() {
					if a.saveAllSettings() {
						a.unsavedModalOpen = false
						a.navigateNow(a.pendingNavPage)
					}
				}
			})
		})
	})
}

func (a *App) updateModalView(c *ui.Context) {
	if !a.updateModalOpen || a.updateInfo == nil {
		return
	}
	t := c.Theme()
	tr := a.tr()
	info := a.updateInfo

	ui.Modal(c.Key("modal-update"), &a.updateModalOpen, func() {
		ui.Column(c).Width(480).Gap(12).Children(func() {
			ui.Text(c, i18n.Fmt(tr.UpdateNewVersionFmt, info.Version)).FontSize(16).Bold()
			ui.Text(c, tr.UpdateContent).FontSize(13).Bold()
			ui.Scroll(c.Key("update-notes-scroll")).Height(170).Padding(10).Radius(8).
				Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, info.Changelog).FontSize(12).Selectable()
			})

			if a.updateDownloading || a.updateInstalling || a.updateStatus != "" {
				ui.Column(c).Gap(6).Children(func() {
					statusColor := t.TextMuted
					if a.updateFailed {
						statusColor = t.Danger
					}
					ui.Text(c, a.updateStatus).FontSize(12).TextColor(statusColor)
					if a.updateDownloading || a.updateInstalling {
						ui.Progress(c, a.updateProgress)
						ui.Row(c).Justify(ui.End).Children(func() {
							ui.Text(c, fmtPercent(a.updateProgress)).FontSize(11).TextColor(t.TextMuted)
						})
					}
				})
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("upd-cancel"), tr.Cancel).Clicked() {
					if a.updateCancel != nil {
						a.updateCancel()
						a.updateCancel = nil
					}
					a.updateModalOpen = false
				}
				if !a.updateDownloading && !a.updateInstalling {
					btnText := tr.UpdateDownloadBtn
					if a.updateFailed {
						btnText = tr.Retry
					}
					if ui.PrimaryButton(c.Key("upd-start"), btnText).Clicked() {
						a.startDownloadAndInstallUpdate()
					}
				}
			})
		})
	})
}

func (a *App) reinstallConfirmModalView(c *ui.Context) {
	if !a.reinstallConfirmOpen {
		return
	}
	tr := a.tr()
	switch ui.AlertDialog(c.Key("modal-reinstall"), &a.reinstallConfirmOpen,
		tr.SettingsReinstallConfirm, tr.SettingsReinstallMessage,
		tr.Cancel, tr.SettingsReinstallButton) {
	case 1:
		a.startCheckForUpdate(true)
	}
}

func (a *App) devIconTestModalView(c *ui.Context) {
	if !a.devIconModalOpen || a.devIconBmp == nil {
		return
	}
	t := c.Theme()
	tr := a.tr()

	ui.Modal(c.Key("modal-dev-icon"), &a.devIconModalOpen, func() {
		ui.Column(c).Width(460).Gap(12).Children(func() {
			ui.Text(c, tr.DevIconTest).FontSize(16).Bold()
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, tr.DevFileName+filepath.Base(a.devIconPath)).FontSize(13).Bold()
				if a.devIconDesc != "" {
					ui.Text(c, tr.DevFileDesc+a.devIconDesc).FontSize(12)
				}
				ui.Text(c, a.devIconPath).FontSize(11).TextColor(t.TextMuted).Selectable()
			})

			ui.Row(c).Gap(12).Children(func() {
				ui.Column(c).Grow(1).Padding(12).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Gap(6).Children(func() {
					ui.Image(c, a.devIconBmp).Size(32, 32).Fit(ui.Contain)
					ui.Text(c, tr.DevMethod1).FontSize(12).Bold()
					ui.Text(c, tr.DevMethod1Desc).FontSize(11).TextColor(t.TextMuted)
				})
				ui.Column(c).Grow(1).Padding(12).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Gap(6).Children(func() {
					ui.Image(c, a.devIconBmp).Size(64, 64).Fit(ui.Contain)
					ui.Text(c, tr.DevMethod2).FontSize(12).Bold()
					ui.Text(c, tr.DevMethod2Desc).FontSize(11).TextColor(t.TextMuted)
				})
			})

			ui.Column(c).Padding(12).Radius(8).Background(t.Surface).Border(1, t.Border).Gap(8).Children(func() {
				ui.Text(c, tr.DevMethod3).FontSize(12).Bold()
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.PrimaryButton(c.Key("dev-preview-btn"), "").Children(func() {
						ui.Image(c, a.devIconBmp).Size(18, 18).Fit(ui.Contain)
						ui.Text(c, tr.DevButtonWithIcon).SingleLine()
					})
					ui.Button(c.Key("dev-preview-btn2"), "").Children(func() {
						ui.Image(c, a.devIconBmp).Size(18, 18).Fit(ui.Contain)
						ui.Text(c, tr.DevButtonWithIcon).SingleLine()
					})
				})
			})

			ui.Text(c, tr.DevIconHint).FontSize(11).TextColor(t.TextMuted)
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c.Key("dev-icon-close"), tr.Close).Clicked() {
					a.devIconModalOpen = false
				}
			})
		})
	})
}
