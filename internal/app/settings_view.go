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
	if a.systemFontsLoaded && a.fontFamily != "" {
		if canonical, ok := a.installedFontMap[strings.ToLower(a.fontFamily)]; ok {
			a.fontFamily = canonical
		}
	}
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
	p := paletteOf(t)
	tr := a.tr()
	if !a.systemFontsLoaded && !a.systemFontsLoading && a.win != nil {
		a.loadSystemFontsAsync()
	}

	ui.Column(c).Fill().Padding(20, 24).Gap(14).Children(func() {
		// 顶部标题与「全部保存」按钮
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tr.SettingsTitle).FontSize(22).Bold().SingleLine()
			if a.hasUnsavedSettings() {
				pillBadge(c, tr.SettingsUnsavedBadge, p.warningTint, t.Warning)
			}
			ui.Spacer(c)
			if a.hasUnsavedSettings() {
				if ui.PrimaryButton(c.Key("set-save-all"), tr.SettingsSaveAll).Clicked() {
					a.saveAllSettings()
				}
			}
		})

		ui.Scroll(c.Key("settings-scroll")).TrackScroll(&a.setScrollState).Grow(1).Children(func() {
			ui.Column(c).Gap(16).Padding(2, 4, 24, 0).MaxWidth(760).Children(func() {
				// 提示条
				ui.Row(c).Padding(12, 16).Radius(11).Background(p.accentTint).
					Border(1, t.Accent.Alpha(0.28)).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(28, 28).Radius(8).Background(t.Accent.Alpha(0.14)).Center().Shrink(0).Children(func() {
						ui.Icon(c, iconInfo).FontSize(15).TextColor(t.Accent)
					})
					ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, tr.SettingsDevHint).FontSize(13).Bold()
						ui.Text(c, tr.SettingsDevHintContent).FontSize(12).TextColor(t.TextMuted)
					})
				})

				// 1. 存储设置
				settingsCard(c, iconHardDrive, tr.SettingsStorage, func() {
					// 安装路径
					ui.Column(c).Gap(6).Children(func() {
						ui.Text(c, tr.SettingsInstallPath).FontSize(13).Bold()
						ui.Row(c).Gap(8).Children(func() {
							ui.TextInput(c.Key("set-install"), &a.setInstallPath).
								Placeholder(tr.SettingsInstallPathHolder).Grow(1).MinWidth(0)
							if ui.Button(c.Key("set-install-browse"), tr.Browse).Clicked() {
								a.pickDirectoryInto(&a.setInstallPath)
							}
							if ui.PrimaryButton(c.Key("set-install-save"), tr.Save).
								Disabled(!a.installPathDirty()).Clicked() {
								a.saveInstallPath()
							}
						})
					})

					ui.Divider(c).Background(p.cardBorder)

					// 归档路径
					ui.Column(c).Gap(6).Children(func() {
						ui.Text(c, tr.SettingsArchivePath).FontSize(13).Bold()
						ui.Row(c).Gap(8).Children(func() {
							ui.TextInput(c.Key("set-archive"), &a.setArchivePath).
								Placeholder(tr.SettingsArchivePathHolder).Grow(1).MinWidth(0)
							if ui.Button(c.Key("set-archive-browse"), tr.Browse).Clicked() {
								a.pickDirectoryInto(&a.setArchivePath)
							}
							if ui.PrimaryButton(c.Key("set-archive-save"), tr.Save).
								Disabled(!a.archivePathDirty()).Clicked() {
								a.saveArchivePath()
							}
						})
					})

					ui.Divider(c).Background(p.cardBorder)

					// 配置文件路径
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, tr.SettingsConfigFilePath).FontSize(13).Bold()
							ui.Text(c, a.setConfigFilePath).FontSize(12).TextColor(t.TextMuted).Selectable()
						})
						if ui.Button(c.Key("set-open-cfg"), tr.SettingsOpenFileLocation).Clicked() {
							a.revealInExplorerAsync(a.setConfigFilePath, true)
						}
					})
				})

				// 2. 归档处理
				settingsCard(c, iconArchive, tr.SettingsArchiveHandling, func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
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

					ui.Divider(c).Background(p.cardBorder)

					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
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
				settingsCard(c, iconTerminal, tr.SettingsExecRecognition, func() {
					// 最大搜索深度
					depthOpts := make([]string, settings.MaxSearchDepth)
					for i := range depthOpts {
						depthOpts[i] = strconv.Itoa(i + 1)
					}
					depthStr := strconv.Itoa(a.setMaxDepth)
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr.SettingsMaxSearchDepth).FontSize(13).Bold().Grow(1).MinWidth(0)
						if ui.Select(c.Key("set-depth"), &depthStr, depthOpts).Width(100).Changed() {
							if d, err := strconv.Atoi(depthStr); err == nil {
								a.setMaxDepth = settings.NormalizeDepth(d)
							}
						}
					})

					ui.Divider(c).Background(p.cardBorder)

					// 扩展名标签列表
					ui.Column(c).Gap(8).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Children(func() {
							ui.Text(c, tr.SettingsExecExtensions).FontSize(13).Bold().Grow(1).MinWidth(0)
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
										Background(p.subtleBg).Border(1, p.cardBorder).Gap(6).AlignItems(ui.Center).Children(func() {
										ui.Text(c, "."+ext).FontSize(12)
										rmBtn := ui.ButtonBase(c.Key("ext-rm-"+ext)).Size(16, 16).Radius(8).Center().
											Label(tr.Remove)
										rmColor := applyHoverIconStyle(rmBtn, t, false, false, nil)
										rmBtn.Children(func() {
											ui.Icon(c, iconX).FontSize(11).TextColor(rmColor)
										})
										if rmBtn.Clicked() {
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
				settingsCard(c, iconPalette, tr.SettingsAppearance, func() {
					// 字体选择（即时生效并保存）
					a.fontPickerView(c)

					ui.Divider(c).Background(p.cardBorder)

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
						ui.Text(c, tr.SettingsLanguage).FontSize(13).Bold().Grow(1).MinWidth(0)
						if ui.Select(c.Key("set-lang"), &langChoice, []string{zhLabel, enLabel}).Width(240).Changed() {
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
				settingsCard(c, iconFileText, tr.SettingsLogSettings, func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
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

					ui.Divider(c).Background(p.cardBorder)

					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
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
				settingsCard(c, iconInfo, tr.SettingsAbout, func() {
					// 版本（3 秒内连续点击 5 次解锁开发者选项）
					verRow := ui.ButtonBase(c.Key("set-version-tap")).Padding(6, 8).Radius(8).
						AlignItems(ui.Center).Gap(10).Cursor(ui.CursorPointer).Label(tr.SettingsVersion)
					if verRow.Hovered() {
						verRow.Background(t.SurfaceHover)
					}
					verRow.Children(func() {
						if a.appIcon != nil {
							ui.Image(c, a.appIcon).Size(24, 24).Radius(6).Shrink(0).Fit(ui.Contain)
						}
						ui.Text(c, tr.SettingsVersion).FontSize(13).Bold().Grow(1).MinWidth(0)
						pillBadge(c, version.FullVersion(), p.subtleBg, t.Text)
					})
					if verRow.Clicked() {
						a.handleVersionTap()
					}

					ui.Divider(c).Background(p.cardBorder)

					ui.Row(c).Padding(4, 8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr.SettingsBuildTime).FontSize(13).Bold().Grow(1).MinWidth(0)
						ui.Text(c, version.BuildTime).FontSize(13).TextColor(t.TextMuted)
					})

					aboutLinkRow := func(key string, ico *ui.SVG, title, displayURL, targetURL string) {
						ui.Divider(c).Background(p.cardBorder)

						row := ui.ButtonBase(c.Key(key)).Padding(6, 8).Radius(8).
							Justify(ui.Start).AlignItems(ui.Center).Gap(10).
							Cursor(ui.CursorPointer).Label(title).Tooltip(targetURL)
						if row.Hovered() {
							row.Background(t.SurfaceHover)
						}
						row.Children(func() {
							ui.Icon(c, ico).FontSize(15).TextColor(t.TextMuted).Shrink(0)
							ui.Text(c, title).FontSize(13).Bold().Grow(1).MinWidth(0).SingleLine()
							ui.Text(c, displayURL).FontSize(12).TextColor(t.Accent).SingleLine().Shrink(0)
							ui.Icon(c, iconExternalLink).FontSize(13).TextColor(t.Accent).Shrink(0)
						})
						if row.Clicked() {
							c.OpenURLThen(targetURL, func(err error) {
								if err != nil {
									a.logAndToastError(a.tr().Error, err)
								}
							})
						}
					}

					aboutLinkRow("set-about-website", iconGlobe, tr.SettingsWebsite, "carrydock.devyi.com", version.WebsiteURL)
					aboutLinkRow("set-about-github", iconGitHub, tr.SettingsGitHub, "github.com/youzhiran/CarryDock", version.GitHubURL)

					ui.Divider(c).Background(p.cardBorder)

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

func (a *App) fontPickerView(c *ui.Context) {
	t := c.Theme()
	p := paletteOf(t)
	tr := a.tr()
	resolved := a.resolvedDefaultFont()

	fallbackName := resolved
	if fallbackName == "" {
		fallbackName = tr.SettingsSystemDefaultFont
	}

	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, tr.SettingsAppFont).FontSize(13).Bold()
			switch {
			case a.setFontFamily == "":
				desc := tr.SettingsFontDefaultDescPlain
				if resolved != "" {
					desc = i18n.Fmt(tr.SettingsFontDefaultDescFmt, resolved)
				}
				ui.Text(c, desc).FontSize(12).TextColor(t.TextMuted)
			case a.systemFontsLoaded && !a.isFontInstalled(a.setFontFamily):
				ui.Text(c, i18n.Fmt(tr.SettingsFontMissingWarnFmt, fallbackName)).
					FontSize(12).TextColor(t.Warning)
			}
		})

		chosen := a.setFontFamily
		sel := ui.SelectBase(c.Key("set-font"), &chosen)
		trigger := sel.Trigger
		trigger.Width(240).Padding(6, 12).Gap(8).Radius(t.Radius).
			Border(1, p.cardBorder).Justify(ui.SpaceBetween).AlignItems(ui.Center).
			Cursor(ui.CursorPointer).Label(tr.SettingsAppFont)
		switch {
		case trigger.Pressed():
			trigger.Background(t.SurfacePressed)
		case trigger.Hovered():
			trigger.Background(t.SurfaceHover).Border(1, p.borderStrong)
		default:
			trigger.Background(p.subtleBg)
		}

		triggerLabel := tr.SettingsSystemDefaultFont
		if a.setFontFamily == "" {
			if resolved != "" {
				triggerLabel = fmt.Sprintf("%s (%s)", tr.SettingsSystemDefaultFont, resolved)
			}
		} else {
			triggerLabel = a.setFontFamily
			if a.systemFontsLoaded && !a.isFontInstalled(a.setFontFamily) {
				triggerLabel = fmt.Sprintf("%s (%s)", a.setFontFamily, tr.SettingsFontUnavailable)
			}
		}
		trigger.Children(func() {
			ui.Text(c, triggerLabel).FontSize(13).SingleLine().Grow(1).MinWidth(0)
			ui.Icon(c, iconDown).FontSize(14).TextColor(t.TextMuted).Shrink(0)
		})

		sel.Popup(func(panel ui.Element) {
			panel.Margin(4, 0, 0, 0).Padding(4).Radius(t.Radius+2).
				Background(p.card).Border(1, p.cardBorder).
				Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).
				Width(300).MaxHeight(340)

			renderFontOption := func(value, label, badge string, disabled, selected bool) {
				item := sel.Item(value).Padding(6, 10).Radius(t.Radius).
					AlignItems(ui.Center).Gap(8).Label(label)
				if badge != "" {
					item.Description(badge)
				}
				if disabled {
					item.Disabled(true)
				} else {
					item.Cursor(ui.CursorPointer)
				}
				highlighted := !disabled && item.Highlighted()
				switch {
				case highlighted:
					item.Background(t.Accent)
				case selected && !disabled:
					item.Background(p.accentTint)
				}

				fg := t.Text
				subFg := t.TextMuted
				switch {
				case disabled:
					fg = t.TextMuted
				case highlighted:
					fg = t.AccentText
					subFg = t.AccentText.Alpha(0.85)
				case selected:
					fg = t.Accent
				}

				item.Children(func() {
					txt := ui.Text(c, label).FontSize(13).TextColor(fg).SingleLine().Grow(1).MinWidth(0)
					if selected && !disabled {
						txt.Bold()
					}
					if badge != "" {
						ui.Text(c, badge).FontSize(11).TextColor(subFg).Shrink(0)
					}
					if selected {
						ui.Icon(c, iconCheck).FontSize(13).TextColor(fg).Shrink(0)
					}
				})
			}

			ui.Scroll(c.Key("set-font-scroll")).Shrink(1).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					// 1. 自动默认（PingFang → 鸿蒙 → 微软雅黑）
					renderFontOption("", tr.SettingsSystemDefaultFont, resolved, false, a.setFontFamily == "")

					// 2. 预设字体（检测系统中是否可用，不可用时置灰禁用）
					ui.Text(c, tr.SettingsPresetFonts).FontSize(11).Bold().
						TextColor(t.TextMuted).Padding(6, 10, 2, 10)

					shownKeys := make(map[string]bool, len(settings.PresetFonts)+len(a.systemFonts))
					shownKeys[""] = true
					for _, preset := range settings.PresetFonts[1:] {
						key := strings.ToLower(preset)
						shownKeys[key] = true

						available := a.isFontInstalled(preset)
						val := preset
						if canonical, ok := a.installedFontMap[key]; ok {
							val = canonical
						}
						badge := ""
						if !available {
							badge = tr.SettingsFontUnavailable
						}
						selected := strings.EqualFold(a.setFontFamily, val)
						renderFontOption(val, preset, badge, !available, selected)
					}

					// 3. 收集系统已安装的其余字体（排除已列出的预设字体）
					extraFonts := make([]string, 0, len(a.systemFonts))
					for _, name := range a.systemFonts {
						key := strings.ToLower(name)
						if shownKeys[key] {
							continue
						}
						shownKeys[key] = true
						extraFonts = append(extraFonts, name)
					}

					// 若配置中保存的自定义字体未出现在预设或过滤后的系统列表（如已卸载或符号字体），仍保留可见项
					if a.setFontFamily != "" && !shownKeys[strings.ToLower(a.setFontFamily)] {
						shownKeys[strings.ToLower(a.setFontFamily)] = true
						available := a.isFontInstalled(a.setFontFamily)
						badge := ""
						if !available {
							badge = tr.SettingsFontUnavailable
						}
						renderFontOption(a.setFontFamily, a.setFontFamily, badge, !available, true)
					}

					// 4. 系统已安装的其他字体
					ui.Divider(c).Margin(4, 6)
					if !a.systemFontsLoaded && a.systemFontsLoading {
						ui.Text(c, tr.SettingsFontLoading).FontSize(12).TextColor(t.TextMuted).Padding(6, 10)
					} else if len(extraFonts) > 0 {
						ui.Text(c, fmt.Sprintf("%s (%d)", tr.SettingsSystemFonts, len(extraFonts))).
							FontSize(11).Bold().TextColor(t.TextMuted).Padding(4, 10, 2, 10)
						for _, name := range extraFonts {
							renderFontOption(name, name, "", false, strings.EqualFold(a.setFontFamily, name))
						}
					}
				})
			})
		})

		if sel.Trigger.Changed() {
			a.setFontFamily = chosen
			a.fontFamily = chosen
			go func(fn string) {
				_ = a.settingsSvc.SaveFontFamily(fn)
			}(chosen)
		}
	})
}

func settingsCard(c *ui.Context, ico *ui.SVG, title string, body func()) {
	t := c.Theme()
	p := paletteOf(t)
	ui.Column(c).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ico != nil {
				ui.Box(c).Size(24, 24).Radius(6).Background(p.accentTint).Center().Shrink(0).Children(func() {
					ui.Icon(c, ico).FontSize(13).TextColor(t.Accent)
				})
			}
			ui.Text(c, title).FontSize(14).Bold().TextColor(t.Text)
		})
		ui.Column(c).Padding(14, 16).Gap(12).Radius(12).
			Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 4, 0, p.shadow).Children(body)
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
