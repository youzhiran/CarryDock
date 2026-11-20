package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"carrydock/internal/archive"
	"carrydock/internal/i18n"
	"carrydock/internal/logger"
	"carrydock/internal/settings"
	"carrydock/internal/version"
	"carrydock/internal/winutil"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) loadSettingsSnapshot() {
	a.setSavedInstallPath = a.settingsSvc.GetInstallPath()
	a.setInstallPath = a.setSavedInstallPath

	a.setSavedArchivePath = a.settingsSvc.GetArchivePath()
	a.setArchivePath = a.setSavedArchivePath

	a.setConfigFilePath = a.settingsSvc.ConfigFilePath()

	a.setSavedRemoveNested = a.settingsSvc.GetRemoveNestedFoldersEnabled()
	a.setRemoveNested = a.setSavedRemoveNested

	a.setSavedMaxDepth = a.settingsSvc.GetExecutableSearchMaxDepth()
	a.setMaxDepth = a.setSavedMaxDepth

	a.setSavedExts = settings.NormalizeExtensions(a.settingsSvc.GetExecutableExtensions())
	a.setExts = slices.Clone(a.setSavedExts)

	a.fontFamily = a.settingsSvc.GetFontFamily()
	a.setFontFamily = a.fontFamily

	a.setSavedLanguage = a.settingsSvc.GetLanguage()
	a.setLanguage = a.setSavedLanguage
	a.lang = i18n.Normalize(a.setSavedLanguage)

	a.setFileLogging = a.settingsSvc.GetEnableFileLogging()
	a.setLogFilePath = a.settingsSvc.GetLogFilePath()

	a.devEnabled = a.settingsSvc.GetDeveloperOptionsEnabled()
	a.useGrid = a.settingsSvc.GetUseGridLayout()
}

func (a *App) installPathDirty() bool {
	return strings.TrimSpace(a.setInstallPath) != a.setSavedInstallPath
}

func (a *App) archivePathDirty() bool {
	return strings.TrimSpace(a.setArchivePath) != a.setSavedArchivePath
}

func (a *App) archiveHandlingDirty() bool {
	return a.setRemoveNested != a.setSavedRemoveNested
}

func (a *App) executableSettingsDirty() bool {
	return a.setMaxDepth != a.setSavedMaxDepth || !slices.Equal(a.setExts, a.setSavedExts)
}

func (a *App) languageDirty() bool {
	return a.setLanguage != a.setSavedLanguage
}

func (a *App) hasUnsavedSettings() bool {
	return a.installPathDirty() ||
		a.archivePathDirty() ||
		a.archiveHandlingDirty() ||
		a.executableSettingsDirty() ||
		a.languageDirty()
}

func (a *App) saveInstallPath() {
	val := strings.TrimSpace(a.setInstallPath)
	if err := a.settingsSvc.SaveInstallPath(val); err != nil {
		a.logAndToastError(a.tr().Error, err)
		return
	}
	a.setSavedInstallPath = val
	a.setInstallPath = val
	a.toast(a.tr().Success, a.tr().SettingsInstallPathSaved)
}

func (a *App) saveArchivePath() {
	val := strings.TrimSpace(a.setArchivePath)
	if err := a.settingsSvc.SaveArchivePath(val); err != nil {
		a.logAndToastError(a.tr().Error, err)
		return
	}
	a.setSavedArchivePath = val
	a.setArchivePath = val
	a.toast(a.tr().Success, a.tr().SettingsArchivePathSaved)
}

func (a *App) saveArchiveHandling() {
	if err := a.settingsSvc.SaveRemoveNestedFoldersEnabled(a.setRemoveNested); err != nil {
		a.logAndToastError(a.tr().Error, err)
		return
	}
	a.setSavedRemoveNested = a.setRemoveNested
	a.toast(a.tr().Success, a.tr().SettingsArchiveHandlingSaved)
}

func (a *App) saveExecutableSettings() {
	if len(a.setExts) == 0 {
		a.showInfo(a.tr().Error, a.tr().SettingsExtSelectAtLeastOne)
		return
	}
	if err := a.settingsSvc.SaveExecutableSearchMaxDepth(a.setMaxDepth); err != nil {
		a.logAndToastError(a.tr().Error, err)
		return
	}
	if err := a.settingsSvc.SaveExecutableExtensions(a.setExts); err != nil {
		a.logAndToastError(a.tr().Error, err)
		return
	}
	a.setSavedMaxDepth = a.setMaxDepth
	a.setSavedExts = slices.Clone(a.setExts)
	a.toast(a.tr().Success, a.tr().SettingsExecSettingsSaved)
}

func (a *App) saveLanguageSetting() {
	if err := a.settingsSvc.SaveLanguage(a.setLanguage); err != nil {
		a.logAndToastError(a.tr().Error, err)
		return
	}
	a.setSavedLanguage = a.setLanguage
	a.lang = i18n.Normalize(a.setLanguage)
	a.syncWindowTitle()
	a.toast(a.tr().Success, a.tr().SettingsLanguageSaved)
}

func (a *App) saveAllSettings() bool {
	if a.installPathDirty() {
		a.saveInstallPath()
	}
	if a.archivePathDirty() {
		a.saveArchivePath()
	}
	if a.archiveHandlingDirty() {
		a.saveArchiveHandling()
	}
	if a.executableSettingsDirty() {
		if len(a.setExts) == 0 {
			a.showInfo(a.tr().Error, a.tr().SettingsExtSelectAtLeastOne)
			return false
		}
		a.saveExecutableSettings()
	}
	if a.languageDirty() {
		a.saveLanguageSetting()
	}
	return !a.hasUnsavedSettings()
}

func (a *App) pickDirectoryInto(target *string) {
	win := a.win
	title := a.tr().Browse
	defaultPath := strings.TrimSpace(*target)
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      win,
			Title:       title,
			DefaultPath: defaultPath,
			Directory:   true,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.update(func() {
			*target = paths[0]
		})
	}()
}

func (a *App) settingsView(c *ui.Context) {
	t := c.Theme()
	tr := a.tr()

	ui.Column(c).Fill().Padding(20, 24).Gap(14).Children(func() {
		// 顶部标题与「全部保存」按钮
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tr.SettingsTitle).FontSize(22).Bold().SingleLine()
			ui.Spacer(c)
			if a.hasUnsavedSettings() {
				if ui.PrimaryButton(c.Key("set-save-all"), tr.SettingsSaveAll).Clicked() {
					a.saveAllSettings()
				}
			}
		})

		ui.Scroll(c.Key("settings-scroll")).TrackScroll(&a.setScrollState).Grow(1).Children(func() {
			ui.Column(c).Gap(20).Padding(0, 4, 24, 0).MaxWidth(760).Children(func() {
				// 提示条
				ui.Row(c).Padding(12, 14).Radius(10).Background(t.Accent.Alpha(0.08)).
					Border(1, t.Accent.Alpha(0.3)).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Gap(2).Grow(1).Children(func() {
						ui.Text(c, tr.SettingsDevHint).FontSize(13).Bold()
						ui.Text(c, tr.SettingsDevHintContent).FontSize(12).TextColor(t.TextMuted)
					})
				})

				// 1. 存储设置
				settingsCard(c, tr.SettingsStorage, func() {
					// 安装路径
					ui.Column(c).Gap(6).Children(func() {
						ui.Text(c, tr.SettingsInstallPath).FontSize(13).Bold()
						ui.Row(c).Gap(8).Children(func() {
							ui.TextInput(c.Key("set-install"), &a.setInstallPath).
								Placeholder(tr.SettingsInstallPathHolder).Grow(1)
							if ui.Button(c.Key("set-install-browse"), tr.Browse).Clicked() {
								a.pickDirectoryInto(&a.setInstallPath)
							}
							if ui.PrimaryButton(c.Key("set-install-save"), tr.Save).
								Disabled(!a.installPathDirty()).Clicked() {
								a.saveInstallPath()
							}
						})
					})

					ui.Divider(c)

					// 归档路径
					ui.Column(c).Gap(6).Children(func() {
						ui.Text(c, tr.SettingsArchivePath).FontSize(13).Bold()
						ui.Row(c).Gap(8).Children(func() {
							ui.TextInput(c.Key("set-archive"), &a.setArchivePath).
								Placeholder(tr.SettingsArchivePathHolder).Grow(1)
							if ui.Button(c.Key("set-archive-browse"), tr.Browse).Clicked() {
								a.pickDirectoryInto(&a.setArchivePath)
							}
							if ui.PrimaryButton(c.Key("set-archive-save"), tr.Save).
								Disabled(!a.archivePathDirty()).Clicked() {
								a.saveArchivePath()
							}
						})
					})

					ui.Divider(c)

					// 配置文件路径
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, tr.SettingsConfigFilePath).FontSize(13).Bold()
							ui.Text(c, a.setConfigFilePath).FontSize(12).TextColor(t.TextMuted).Selectable()
						})
						if ui.Button(c.Key("set-open-cfg"), tr.SettingsOpenFileLocation).Clicked() {
							a.revealInExplorerAsync(a.setConfigFilePath, true)
						}
					})
				})

				// 2. 归档处理
				settingsCard(c, tr.SettingsArchiveHandling, func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, tr.SettingsRemoveNested).FontSize(13).Bold()
							ui.Text(c, tr.SettingsRemoveNestedDesc).FontSize(12).TextColor(t.TextMuted)
						})
						ui.Switch(c.Key("set-nested"), &a.setRemoveNested).Label(tr.SettingsRemoveNested)
					})
					if a.archiveHandlingDirty() {
						ui.Row(c).Justify(ui.End).Children(func() {
							if ui.PrimaryButton(c.Key("set-save-nested"), tr.SettingsSaveArchiveHandling).Clicked() {
								a.saveArchiveHandling()
							}
						})
					}

					ui.Divider(c)

					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, tr.SettingsTest7Zip).FontSize(13).Bold()
							ui.Text(c, tr.SettingsTest7ZipDesc).FontSize(12).TextColor(t.TextMuted)
						})
						if ui.Button(c.Key("set-test-7z"), tr.SettingsTest7Zip).Clicked() {
							go func() {
								_, report := archive.Diagnose7Zip()
								a.update(func() {
									a.showInfo(tr.SettingsTest7ZipTitle, report)
								})
							}()
						}
					})
				})

				// 3. 可执行文件识别
				settingsCard(c, tr.SettingsExecRecognition, func() {
					// 最大搜索深度
					depthOpts := make([]string, settings.MaxSearchDepth)
					for i := range depthOpts {
						depthOpts[i] = strconv.Itoa(i + 1)
					}
					depthStr := strconv.Itoa(a.setMaxDepth)
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr.SettingsMaxSearchDepth).FontSize(13).Bold().Grow(1)
						if ui.Select(c.Key("set-depth"), &depthStr, depthOpts).Width(100).Changed() {
							if d, err := strconv.Atoi(depthStr); err == nil {
								a.setMaxDepth = settings.NormalizeDepth(d)
							}
						}
					})

					ui.Divider(c)

					// 扩展名标签列表
					ui.Column(c).Gap(8).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Children(func() {
							ui.Text(c, tr.SettingsExecExtensions).FontSize(13).Bold().Grow(1)
							if ui.Button(c.Key("set-edit-exts"), tr.SettingsSelectExtensions).Clicked() {
								a.extDraft = make(map[string]bool, len(a.setExts))
								for _, e := range a.setExts {
									a.extDraft[e] = true
								}
								a.extCustom = ""
								a.extError = ""
								a.extModalOpen = true
							}
						})
						if len(a.setExts) == 0 {
							ui.Text(c, tr.SettingsNoExtensions).FontSize(12).TextColor(t.TextMuted)
						} else {
							var removeTarget string
							ui.Row(c).Wrap().Gap(6).Children(func() {
								for _, ext := range a.setExts {
									extCopy := ext
									ui.Row(c.Key("ext-chip-"+ext)).Padding(4, 10).Radius(999).
										Background(t.Background).Border(1, t.Border).Gap(6).AlignItems(ui.Center).Children(func() {
										ui.Text(c, "."+ext).FontSize(12)
										if ui.ButtonBase(c.Key("ext-rm-"+ext)).Size(16, 16).Radius(8).Center().
											Label(tr.Remove).Children(func() {
											ui.Icon(c, iconX).FontSize(11).TextColor(t.TextMuted)
										}).Clicked() {
											removeTarget = extCopy
										}
									})
								}
							})
							if removeTarget != "" {
								var updated []string
								for _, e := range a.setExts {
									if e != removeTarget {
										updated = append(updated, e)
									}
								}
								a.setExts = updated
							}
						}
					})

					if a.executableSettingsDirty() {
						ui.Row(c).Justify(ui.End).Children(func() {
							if ui.PrimaryButton(c.Key("set-save-exec"), tr.SettingsSaveExecSettings).Clicked() {
								a.saveExecutableSettings()
							}
						})
					}
				})

				// 4. 外观与语言
				settingsCard(c, tr.SettingsAppearance, func() {
					// 字体选择（即时生效并保存）
					fontLabels := make([]string, 0, len(settings.PresetFonts))
					fontMap := make(map[string]string, len(settings.PresetFonts))
					currentFontLabel := tr.SettingsSystemDefaultFont
					for _, f := range settings.PresetFonts {
						label := f
						if label == "" {
							label = tr.SettingsSystemDefaultFont
						}
						fontLabels = append(fontLabels, label)
						fontMap[label] = f
						if f == a.setFontFamily {
							currentFontLabel = label
						}
					}
					if a.setFontFamily != "" {
						if _, exists := fontMap[a.setFontFamily]; !exists {
							fontLabels = append(fontLabels, a.setFontFamily)
							fontMap[a.setFontFamily] = a.setFontFamily
							currentFontLabel = a.setFontFamily
						}
					}

					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr.SettingsAppFont).FontSize(13).Bold().Grow(1)
						if ui.Select(c.Key("set-font"), &currentFontLabel, fontLabels).Width(220).Changed() {
							chosen := fontMap[currentFontLabel]
							a.setFontFamily = chosen
							a.fontFamily = chosen
							go func(fn string) {
								_ = a.settingsSvc.SaveFontFamily(fn)
							}(chosen)
						}
					})

					ui.Divider(c)

					// 界面语言
					zhLabel := tr.LanguageChinese
					enLabel := tr.LanguageEnglish
					if tr.TranslationByAI != "" {
						enLabel = enLabel + " " + tr.TranslationByAI
					}
					langChoice := zhLabel
					if i18n.Normalize(a.setLanguage) == i18n.LangEnUS {
						langChoice = enLabel
					}
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr.SettingsLanguage).FontSize(13).Bold().Grow(1)
						if ui.Select(c.Key("set-lang"), &langChoice, []string{zhLabel, enLabel}).Width(220).Changed() {
							if langChoice == enLabel {
								a.setLanguage = string(i18n.LangEnUS)
							} else {
								a.setLanguage = string(i18n.LangZhCN)
							}
						}
					})
					if a.languageDirty() {
						ui.Row(c).Justify(ui.End).Children(func() {
							if ui.PrimaryButton(c.Key("set-save-lang"), tr.SettingsSaveLanguage).Clicked() {
								a.saveLanguageSetting()
							}
						})
					}
				})

				// 5. 日志设置
				settingsCard(c, tr.SettingsLogSettings, func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, tr.SettingsEnableFileLogging).FontSize(13).Bold()
							ui.Text(c, tr.SettingsEnableFileLoggingMsg).FontSize(12).TextColor(t.TextMuted)
						})
						if ui.Switch(c.Key("set-logging"), &a.setFileLogging).Label(tr.SettingsEnableFileLogging).Changed() {
							enabled := a.setFileLogging
							logPath := a.setLogFilePath
							go func() {
								_ = a.settingsSvc.SaveEnableFileLogging(enabled)
								_ = logger.Configure(enabled, logPath)
								a.update(func() {
									a.toast(a.tr().Success, a.tr().SettingsLogSettingsSaved)
								})
							}()
						}
					})

					ui.Divider(c)

					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, tr.SettingsLogFilePath).FontSize(13).Bold()
							ui.Text(c, a.setLogFilePath).FontSize(12).TextColor(t.TextMuted).Selectable()
						})
						if ui.Button(c.Key("set-open-log"), tr.SettingsOpenFileLocation).Clicked() {
							ensureDir := winutil.SamePath(filepath.Dir(a.setLogFilePath), filepath.Dir(a.setConfigFilePath))
							a.revealInExplorerAsync(a.setLogFilePath, ensureDir)
						}
					})
				})

				// 6. 关于与更新
				settingsCard(c, tr.SettingsAbout, func() {
					// 版本（3 秒内连续点击 5 次解锁开发者选项）
					verRow := ui.ButtonBase(c.Key("set-version-tap")).Padding(6, 8).Radius(6).
						AlignItems(ui.Center).Label(tr.SettingsVersion)
					if verRow.Hovered() {
						verRow.Background(t.SurfaceHover)
					}
					verRow.Children(func() {
						ui.Text(c, tr.SettingsVersion).FontSize(13).Bold().Grow(1)
						ui.Text(c, version.FullVersion()).FontSize(13).TextColor(t.TextMuted)
					})
					if verRow.Clicked() {
						a.handleVersionTap()
					}

					ui.Divider(c)

					ui.Row(c).Padding(4, 8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr.SettingsBuildTime).FontSize(13).Bold().Grow(1)
						ui.Text(c, version.BuildTime).FontSize(13).TextColor(t.TextMuted)
					})

					ui.Divider(c)

					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						if ui.Button(c.Key("set-check-update"), "").Disabled(a.updateChecking).Children(func() {
							if a.updateChecking {
								ui.Spinner(c).Size(14, 14)
							} else {
								ui.Icon(c, iconRefresh).FontSize(14)
							}
							ui.Text(c, tr.SettingsCheckUpdate).SingleLine()
						}).Clicked() {
							a.startCheckForUpdate(false)
						}

						if ui.Button(c.Key("set-reinstall"), "").Disabled(a.updateChecking).Children(func() {
							ui.Icon(c, iconDownload).FontSize(14)
							ui.Text(c, tr.SettingsReinstallLatest).SingleLine()
						}).Clicked() {
							a.reinstallConfirmOpen = true
						}
					})
				})
			})
		})
	})
}

func settingsCard(c *ui.Context, title string, body func()) {
	t := c.Theme()
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, title).FontSize(14).Bold().TextColor(t.Text)
		ui.Column(c).Padding(16).Gap(12).Radius(10).
			Background(t.Surface).Border(1, t.Border).Children(body)
	})
}

func (a *App) handleVersionTap() {
	now := time.Now()
	if a.firstVersionTap.IsZero() || now.Sub(a.firstVersionTap) > 3*time.Second {
		a.firstVersionTap = now
		a.versionTapCount = 1
	} else {
		a.versionTapCount++
	}
	if !a.devEnabled && a.versionTapCount >= 5 {
		a.devEnabled = true
		a.versionTapCount = 0
		a.firstVersionTap = time.Time{}
		go func() {
			_ = a.settingsSvc.SaveDeveloperOptionsEnabled(true)
		}()
		a.toast(a.tr().Success, a.tr().SettingsDevUnlocked)
	}
}

func (a *App) startCheckForUpdate(forceReinstall bool) {
	if a.updateChecking {
		return
	}
	a.updateChecking = true
	a.updateFailed = false

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		hasUpdate, info, err := a.updaterSvc.CheckForUpdate(ctx, forceReinstall)
		a.update(func() {
			a.updateChecking = false
			tr := a.tr()
			if err != nil {
				a.toast(tr.Error, err.Error())
				return
			}
			if !hasUpdate || info == nil {
				a.toast(tr.Success, tr.UpdateUpToDate)
				return
			}
			a.updateInfo = info
			a.updateStatus = ""
			a.updateProgress = 0
			a.updateDownloading = false
			a.updateInstalling = false
			a.updateFailed = false
			a.updateModalOpen = true
			if forceReinstall {
				a.startDownloadAndInstallUpdate()
			}
		})
	}()
}

func (a *App) startDownloadAndInstallUpdate() {
	if a.updateInfo == nil || a.updateDownloading || a.updateInstalling {
		return
	}
	info := *a.updateInfo
	ctx, cancel := context.WithCancel(context.Background())
	a.updateCancel = cancel
	a.updateDownloading = true
	a.updateInstalling = false
	a.updateFailed = false
	a.updateProgress = 0
	a.updateStatus = a.tr().UpdateDownloading

	go func() {
		savePath := filepath.Join(os.TempDir(), info.AssetName)
		err := a.updaterSvc.DownloadUpdate(ctx, info.DownloadURL, savePath, func(p float64) {
			a.update(func() {
				a.updateProgress = p
			})
		})
		if err != nil {
			a.update(func() {
				a.updateDownloading = false
				a.updateCancel = nil
				if ctx.Err() != nil {
					a.updateStatus = a.tr().UpdateCancelled
				} else {
					a.updateFailed = true
					a.updateStatus = i18n.Fmt(a.tr().UpdateFailedFmt, err.Error())
				}
			})
			return
		}

		a.update(func() {
			a.updateDownloading = false
			a.updateInstalling = true
			a.updateProgress = 1
			a.updateStatus = a.tr().UpdateInstalling
		})

		if err := a.updaterSvc.InstallUpdate(savePath); err != nil {
			a.update(func() {
				a.updateInstalling = false
				a.updateFailed = true
				a.updateStatus = i18n.Fmt(a.tr().UpdateFailedFmt, err.Error())
			})
		}
	}()
}

func fmtPercent(p float64) string {
	return fmt.Sprintf("%.1f%%", p*100)
}
