import 'package:carrydock/l10n/app_localizations.dart';
import 'package:carrydock/providers/developer_options_provider.dart';
import 'package:carrydock/providers/language_provider.dart';
import 'package:carrydock/providers/theme_provider.dart';
import 'package:carrydock/screens/settings/7zip_test.dart';
import 'package:carrydock/screens/settings/extensions_dialog.dart';
import 'package:carrydock/screens/settings/sections.dart';
import 'package:carrydock/screens/settings/utils.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/utils/error_handler.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:file_picker/file_picker.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/foundation.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:path/path.dart' as p;
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final TextEditingController _installPathController = TextEditingController();
  final TextEditingController _archivePathController = TextEditingController();
  final SettingsService _settingsService = SettingsService();

  String _savedInstallPath = '';
  String _savedArchivePath = '';
  String _configFilePath = '';
  int _savedMaxSearchDepth = SettingsService.defaultExecutableSearchMaxDepth;
  List<String> _savedExecutableExtensions = List<String>.from(
    SettingsService.defaultExecutableExtensions,
  );
  List<String> _selectedExecutableExtensions = List<String>.from(
    SettingsService.defaultExecutableExtensions,
  );
  bool _savedRemoveNestedFoldersEnabled =
      SettingsService.defaultRemoveNestedFoldersEnabled;
  bool _removeNestedFoldersEnabled =
      SettingsService.defaultRemoveNestedFoldersEnabled;

  int _versionTapCount = 0;
  DateTime? _firstVersionTap;
  String _appVersion = '';
  String _buildTime = '';

  bool _installPathDirty = false;
  bool _archivePathDirty = false;
  bool _executableSettingsDirty = false;
  bool _archiveHandlingSettingsDirty = false;
  bool _logSettingsDirty = false;
  bool _hasUnsavedChanges = false;
  int _selectedMaxSearchDepth = SettingsService.defaultExecutableSearchMaxDepth;
  bool _enableFileLogging = SettingsService.defaultEnableFileLogging;
  bool _savedEnableFileLogging = SettingsService.defaultEnableFileLogging;
  String _logFilePath = '';

  // 语言设置
  String _savedLanguage = 'zh_CN';
  String _selectedLanguage = 'zh_CN';
  bool _languageDirty = false;

  @override
  void initState() {
    super.initState();
    _savedExecutableExtensions = normalizeExtensionsList(
      _savedExecutableExtensions,
    );
    _installPathController.addListener(_refreshDirtyStates);
    _archivePathController.addListener(_refreshDirtyStates);
    _loadInstallPath();
    _loadArchivePath();
    _loadArchiveHandlingSettings();
    _loadExecutableSettings();
    _loadLogSettings();
    _loadLanguageSetting();
    // 延迟加载版本信息，等待 context 可用
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        final l10n = AppLocalizations.of(context)!;
        setState(() {
          _appVersion = l10n.loading;
          _buildTime = l10n.unknown;
        });
      }
      _loadAppVersion();
      _loadBuildTime();
    });
    _loadConfigFilePath();
  }

  @override
  void dispose() {
    _installPathController.removeListener(_refreshDirtyStates);
    _archivePathController.removeListener(_refreshDirtyStates);
    _installPathController.dispose();
    _archivePathController.dispose();
    super.dispose();
  }

  Future<void> _loadConfigFilePath() async {
    final path = await _settingsService.getConfigFilePath();
    if (mounted) {
      setState(() {
        _configFilePath = path;
      });
    }
  }

  Future<void> _loadInstallPath() async {
    final path = await _settingsService.getInstallPath();
    final normalizedPath = path?.trim() ?? '';
    if (!mounted) {
      return;
    }
    _savedInstallPath = normalizedPath;
    _installPathController.text = _savedInstallPath;
    _refreshDirtyStates();
  }

  Future<void> _loadArchivePath() async {
    final path = await _settingsService.getArchivePath();
    final normalizedPath = path?.trim() ?? '';
    if (!mounted) {
      return;
    }
    _savedArchivePath = normalizedPath;
    _archivePathController.text = _savedArchivePath;
    _refreshDirtyStates();
  }

  Future<void> _loadArchiveHandlingSettings() async {
    final removeNested = await _settingsService.getRemoveNestedFoldersEnabled();
    if (!mounted) {
      return;
    }
    setState(() {
      _savedRemoveNestedFoldersEnabled = removeNested;
      _removeNestedFoldersEnabled = removeNested;
    });
    _refreshDirtyStates();
  }

  Future<void> _loadExecutableSettings() async {
    final maxDepth = await _settingsService.getExecutableSearchMaxDepth();
    final extensions = await _settingsService.getExecutableExtensions();
    _savedExecutableExtensions = normalizeExtensionsList(extensions);
    if (!mounted) return;
    final normalizedDepth = normalizeDepth(maxDepth);
    _savedMaxSearchDepth = normalizedDepth;
    setState(() {
      _selectedMaxSearchDepth = normalizedDepth;
      _selectedExecutableExtensions = List<String>.from(
        _savedExecutableExtensions,
      );
    });
    _refreshDirtyStates();
  }

  Future<void> _loadAppVersion() async {
    if (!mounted) return;
    final l10n = AppLocalizations.of(context)!;
    try {
      final info = await PackageInfo.fromPlatform();
      final buildNumber = info.buildNumber.trim();
      final versionText = buildNumber.isEmpty
          ? info.version
          : '${info.version}+$buildNumber';
      if (!mounted) return;
      setState(() {
        _appVersion = versionText;
      });
    } catch (e, s) {
      logger.w('Failed to get app version', error: e, stackTrace: s);
      if (!mounted) return;
      setState(() {
        _appVersion = l10n.unknown;
      });
    }
  }

  Future<void> _loadBuildTime() async {
    if (!mounted) return;
    final l10n = AppLocalizations.of(context)!;
    try {
      // 尝试从生成的构建信息文件中读取
      final buildInfo = await readBuildInfo();
      if (!mounted) return;
      setState(() {
        _buildTime = buildInfo;
      });
    } catch (e, s) {
      logger.w('Failed to get build time', error: e, stackTrace: s);
      if (!mounted) return;
      setState(() {
        _buildTime = l10n.unknown;
      });
    }
  }

  Future<void> _loadLogSettings() async {
    final enableFileLogging = await _settingsService.getEnableFileLogging();
    final logFilePath = await _settingsService.getLogFilePath();
    if (!mounted) return;
    setState(() {
      _enableFileLogging = enableFileLogging;
      _savedEnableFileLogging = enableFileLogging;
      _logFilePath = logFilePath;
    });
    _refreshDirtyStates();
  }

  Future<void> _loadLanguageSetting() async {
    final language = await _settingsService.getLanguage();
    if (!mounted) return;
    setState(() {
      _savedLanguage = language;
      _selectedLanguage = language;
    });
    _refreshDirtyStates();
  }

  Future<void> _saveLanguageSetting() async {
    final l10n = AppLocalizations.of(context)!;
    final languageProvider = Provider.of<LanguageProvider>(
      context,
      listen: false,
    );
    await languageProvider.setLanguage(_selectedLanguage);
    if (!mounted) return;
    _savedLanguage = _selectedLanguage;
    _showSuccessMessage(l10n.settingsLanguageSaved);
    _refreshDirtyStates();
  }

  Future<void> _saveInstallPath() async {
    final l10n = AppLocalizations.of(context)!;
    final value = _installPathController.text.trim();
    await _settingsService.saveInstallPath(value);
    if (!mounted) return;
    _savedInstallPath = value;
    _showSuccessMessage(l10n.settingsInstallPathSaved);
    _refreshDirtyStates();
  }

  Future<void> _saveArchivePath() async {
    final l10n = AppLocalizations.of(context)!;
    final value = _archivePathController.text.trim();
    await _settingsService.saveArchivePath(value);
    if (!mounted) return;
    _savedArchivePath = value;
    _showSuccessMessage(l10n.settingsArchivePathSaved);
    _refreshDirtyStates();
  }

  Future<void> _saveArchiveHandlingSettings() async {
    final l10n = AppLocalizations.of(context)!;
    await _settingsService.saveRemoveNestedFoldersEnabled(
      _removeNestedFoldersEnabled,
    );
    if (!mounted) return;
    _savedRemoveNestedFoldersEnabled = _removeNestedFoldersEnabled;
    _showSuccessMessage(l10n.settingsArchiveHandlingSaved);
    _refreshDirtyStates();
  }

  Future<void> _saveExecutableSettings() async {
    final l10n = AppLocalizations.of(context)!;
    final depth = _selectedMaxSearchDepth;
    if (depth < maxSearchDepthOptions.first ||
        depth > maxSearchDepthOptions.last) {
      if (!mounted) return;
      Provider.of<ErrorHandler>(
        context,
        listen: false,
      ).showHint(l10n.settingsError, l10n.settingsInvalidSearchDepth);
      return;
    }

    final extensions = _selectedExecutableExtensions;
    if (extensions.isEmpty) {
      if (!mounted) return;
      Provider.of<ErrorHandler>(
        context,
        listen: false,
      ).showHint(l10n.settingsError, l10n.settingsInvalidExtensions);
      return;
    }

    await _settingsService.saveExecutableSearchMaxDepth(depth);
    await _settingsService.saveExecutableExtensions(extensions);
    if (!mounted) return;
    _savedMaxSearchDepth = depth;
    _savedExecutableExtensions = List<String>.from(extensions);
    _showSuccessMessage(l10n.settingsExecutableSettingsSaved);
    _selectedMaxSearchDepth = depth;
    _refreshDirtyStates();
  }

  Future<void> _saveLogSettings() async {
    final l10n = AppLocalizations.of(context)!;
    final enableFileLogging = _enableFileLogging;

    // 保存日志设置
    await _settingsService.saveEnableFileLogging(enableFileLogging);

    // 获取当前日志文件路径
    final logFilePath = await _settingsService.getLogFilePath();

    // 立即更新日志管理器配置，无需重启应用
    await LogManager().updateConfiguration(
      enableFileLogging: enableFileLogging,
      logFilePath: logFilePath,
    );

    if (!mounted) return;
    _savedEnableFileLogging = enableFileLogging;
    _showSuccessMessage(l10n.settingsLogSettingsSaved);
    _refreshDirtyStates();
  }

  Future<void> _saveAllChanges() async {
    if (_installPathDirty) {
      await _saveInstallPath();
      if (!mounted) return;
    }
    if (_archivePathDirty) {
      await _saveArchivePath();
      if (!mounted) return;
    }
    if (_archiveHandlingSettingsDirty) {
      await _saveArchiveHandlingSettings();
      if (!mounted) return;
    }
    if (_executableSettingsDirty) {
      await _saveExecutableSettings();
      if (!mounted) return;
    }
    if (_logSettingsDirty) {
      await _saveLogSettings();
    }
    if (_languageDirty) {
      await _saveLanguageSetting();
    }
  }

  Future<void> _pickDirectory(TextEditingController controller) async {
    final result = await FilePicker.platform.getDirectoryPath();
    if (result != null) {
      controller.text = result;
    }
  }

  void _showSuccessMessage(String message) {
    final l10n = AppLocalizations.of(context)!;
    displayInfoBar(
      context,
      builder: (context, close) {
        return InfoBar(
          title: Text(l10n.success),
          content: Text(message),
          action: IconButton(
            icon: const Icon(FluentIcons.clear),
            onPressed: close,
          ),
          severity: InfoBarSeverity.success,
        );
      },
    );
  }

  void _refreshDirtyStates() {
    if (!mounted) {
      return;
    }
    final installValue = _installPathController.text.trim();
    final archiveValue = _archivePathController.text.trim();
    final extensions = _selectedExecutableExtensions;

    final installDirty = installValue != _savedInstallPath;
    final archiveDirty = archiveValue != _savedArchivePath;
    final depthDirty = _selectedMaxSearchDepth != _savedMaxSearchDepth;
    final extensionsDirty = !listEquals(extensions, _savedExecutableExtensions);
    final removeNestedDirty =
        _removeNestedFoldersEnabled != _savedRemoveNestedFoldersEnabled;
    final enableFileLoggingDirty =
        _enableFileLogging != _savedEnableFileLogging;
    final languageDirty = _selectedLanguage != _savedLanguage;

    final executableDirty = depthDirty || extensionsDirty;
    final logDirty = enableFileLoggingDirty;
    final hasChanges =
        installDirty ||
        archiveDirty ||
        executableDirty ||
        removeNestedDirty ||
        logDirty ||
        languageDirty;

    if (installDirty != _installPathDirty ||
        archiveDirty != _archivePathDirty ||
        executableDirty != _executableSettingsDirty ||
        removeNestedDirty != _archiveHandlingSettingsDirty ||
        logDirty != _logSettingsDirty ||
        languageDirty != _languageDirty ||
        hasChanges != _hasUnsavedChanges) {
      setState(() {
        _installPathDirty = installDirty;
        _archivePathDirty = archiveDirty;
        _executableSettingsDirty = executableDirty;
        _archiveHandlingSettingsDirty = removeNestedDirty;
        _logSettingsDirty = logDirty;
        _languageDirty = languageDirty;
        _hasUnsavedChanges = hasChanges;
      });
    }
  }

  void _applyExecutableExtensions(List<String> extensions) {
    final normalized = normalizeExtensionsList(extensions);
    setState(() {
      _selectedExecutableExtensions = normalized;
    });
    _refreshDirtyStates();
  }

  void _removeExecutableExtension(String extension) {
    final updated = List<String>.from(_selectedExecutableExtensions)
      ..remove(extension);
    _applyExecutableExtensions(updated);
  }

  void _handleVersionTap() {
    final now = DateTime.now();
    if (_firstVersionTap == null ||
        now.difference(_firstVersionTap!) > const Duration(seconds: 3)) {
      _firstVersionTap = now;
      _versionTapCount = 1;
    } else {
      _versionTapCount += 1;
    }

    final developerProvider = Provider.of<DeveloperOptionsProvider>(
      context,
      listen: false,
    );

    if (!developerProvider.enabled && _versionTapCount >= 5) {
      developerProvider.setEnabled(true);
      _versionTapCount = 0;
      _firstVersionTap = null;
    }
  }

  Future<bool> _onWillPop() async {
    if (!_hasUnsavedChanges) {
      return true;
    }

    final l10n = AppLocalizations.of(context)!;
    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) {
        final navigator = Navigator.of(dialogContext);
        return ContentDialog(
          title: Text(l10n.settingsUnsavedChanges),
          content: Text(l10n.settingsUnsavedChangesMessage),
          actions: [
            Button(
              child: Text(l10n.cancel),
              onPressed: () => navigator.pop(false),
            ),
            Button(
              child: Text(l10n.settingsDiscardChanges),
              onPressed: () => navigator.pop(true),
            ),
            FilledButton(
              child: Text(l10n.settingsSaveAndExit),
              onPressed: () async {
                await _saveAllChanges();
                if (!mounted) {
                  return;
                }
                if (!_hasUnsavedChanges) {
                  navigator.pop(true);
                }
              },
            ),
          ],
        );
      },
    );

    return result ?? false;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final themeProvider = Provider.of<ThemeProvider>(context);
    final developerOptionsEnabled = context
        .watch<DeveloperOptionsProvider>()
        .enabled;

    final typography = FluentTheme.of(context).typography;

    final page = ScaffoldPage.scrollable(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
      header: PageHeader(title: Text(l10n.settingsTitle, style: typography.title)),
      children: [
        // 开发阶段提示：如遇异常请删除配置文件
        InfoBar(
          title: Text(l10n.settingsDevHint),
          content: Text(l10n.settingsDevHintContent),
          severity: InfoBarSeverity.warning,
        ),
        const SizedBox(height: 12),
        buildSectionHeader(context, l10n.settingsStorage),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsInstallPath,
          child: TextBox(
            controller: _installPathController,
            placeholder: l10n.settingsInstallPathPlaceholder,
          ),
        ),
        const SizedBox(height: 12),
        buildPathActionRow(
          context,
          isDirty: _installPathDirty,
          onSave: _saveInstallPath,
          onSelect: () => _pickDirectory(_installPathController),
          saveLabel: l10n.settingsSaveInstallPath,
        ),
        const SizedBox(height: 32),
        InfoLabel(
          label: l10n.settingsArchivePath,
          child: TextBox(
            controller: _archivePathController,
            placeholder: l10n.settingsArchivePathPlaceholder,
          ),
        ),
        const SizedBox(height: 12),
        buildPathActionRow(
          context,
          isDirty: _archivePathDirty,
          onSave: _saveArchivePath,
          onSelect: () => _pickDirectory(_archivePathController),
          saveLabel: l10n.settingsSaveArchivePath,
        ),
        const SizedBox(height: 32),
        InfoLabel(
          label: l10n.settingsConfigFilePath,
          child: Row(
            children: [
              Expanded(
                child: SelectableText(
                  _configFilePath,
                  style: FluentTheme.of(context).typography.body,
                ),
              ),
              const SizedBox(width: 8),
              Button(
                child: Text(l10n.settingsOpenFileLocation),
                onPressed: () async {
                  if (_configFilePath.isEmpty) {
                    if (!context.mounted) return;
                    Provider.of<ErrorHandler>(
                      context,
                      listen: false,
                    ).showHint(l10n.settingsError, l10n.settingsInvalidConfigPath);
                    return;
                  }
                  try {
                    final uri = Uri.file(p.dirname(_configFilePath));
                    await launchUrl(uri);
                  } catch (e, s) {
                    if (!context.mounted) return;
                    Provider.of<ErrorHandler>(
                      context,
                      listen: false,
                    ).handleError(e, s);
                  }
                },
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),
        buildSectionHeader(context, l10n.settingsArchiveHandling),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsRemoveNestedFolders,
          child: ToggleSwitch(
            checked: _removeNestedFoldersEnabled,
            onChanged: (value) {
              setState(() {
                _removeNestedFoldersEnabled = value;
              });
              _refreshDirtyStates();
            },
            content: Text(l10n.settingsRemoveNestedFoldersDesc),
          ),
        ),
        const SizedBox(height: 8),
        Button(
          onPressed: _archiveHandlingSettingsDirty
              ? _saveArchiveHandlingSettings
              : null,
          child: Text(l10n.settingsSaveArchiveHandling),
        ),
        const SizedBox(height: 24),
        Button(
          onPressed: () => test7ZipPathFinding(context),
          child: Text(l10n.settingsTest7Zip),
        ),
        const SizedBox(height: 4),
        Text(
          l10n.settingsTest7ZipDesc,
          style: FluentTheme.of(context).typography.caption,
        ),
        buildSectionDivider(top: 24, bottom: 32),
        buildSectionHeader(context, l10n.settingsExecutableRecognition),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsMaxSearchDepth,
          child: SizedBox(
            width: 120,
            child: ComboBox<int>(
              value: _selectedMaxSearchDepth,
              items: maxSearchDepthOptions.map((depth) {
                return ComboBoxItem(
                  value: depth,
                  child: Text(depth.toString()),
                );
              }).toList(),
              onChanged: (depth) {
                if (depth == null) {
                  return;
                }
                setState(() {
                  _selectedMaxSearchDepth = depth;
                });
                _refreshDirtyStates();
              },
            ),
          ),
        ),
        const SizedBox(height: 16),
        InfoLabel(
          label: l10n.settingsExecutableExtensions,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (_selectedExecutableExtensions.isEmpty)
                Text(l10n.settingsNoExtensionsSelected, style: typography.caption)
              else
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: _selectedExecutableExtensions
                      .map(
                        (ext) => buildExtensionTag(
                          context,
                          extension: ext,
                          onRemove: _removeExecutableExtension,
                        ),
                      )
                      .toList(),
                ),
              const SizedBox(height: 8),
              Button(
                onPressed: () => openExecutableExtensionsDialog(
                  context,
                  initialExtensions: _selectedExecutableExtensions,
                  onExtensionsSelected: _applyExecutableExtensions,
                ),
                child: Text(l10n.settingsSelectExtensions),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        Button(
          onPressed: _executableSettingsDirty ? _saveExecutableSettings : null,
          child: Text(l10n.settingsSaveExecutableSettings),
        ),
        buildSectionDivider(),
        buildSectionHeader(context, l10n.settingsAppearance),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsAppFont,
          child: ComboBox<String>(
            value: themeProvider.fontFamily,
            items: availableFonts.map((font) {
              return ComboBoxItem(value: font, child: Text(font));
            }).toList(),
            onChanged: (font) {
              if (font != null) {
                themeProvider.updateFontFamily(font);
              }
            },
          ),
        ),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsLanguage,
          child: ComboBox<String>(
            value: _selectedLanguage,
            items: [
              ComboBoxItem(
                value: 'zh_CN',
                child: Text(l10n.languageChinese),
              ),
              ComboBoxItem(
                value: 'en_US',
                child: Text('${l10n.languageEnglish} ${l10n.translationByAI}'),
              ),
            ],
            onChanged: (language) {
              if (language != null) {
                setState(() {
                  _selectedLanguage = language;
                });
                _refreshDirtyStates();
              }
            },
          ),
        ),
        const SizedBox(height: 8),
        Button(
          onPressed: _languageDirty ? _saveLanguageSetting : null,
          child: Text(l10n.settingsSaveLanguage),
        ),
        buildSectionDivider(),
        buildSectionHeader(context, l10n.settingsLogSettings),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsEnableFileLogging,
          child: ToggleSwitch(
            checked: _enableFileLogging,
            onChanged: (value) async {
              setState(() {
                _enableFileLogging = value;
              });
              _refreshDirtyStates();

              // 自动保存日志设置，无需手动点击保存按钮
              await _saveLogSettings();
            },
            content: Text(l10n.settingsEnableFileLoggingDesc),
          ),
        ),
        const SizedBox(height: 12),
        InfoLabel(
          label: l10n.settingsLogFilePath,
          child: Row(
            children: [
              Expanded(
                child: SelectableText(
                  _logFilePath.replaceAll('\\', '/'),
                  style: FluentTheme.of(context).typography.body,
                ),
              ),
              const SizedBox(width: 8),
              Button(
                child: Text(l10n.settingsOpenFileLocation),
                onPressed: () async {
                  if (_logFilePath.isEmpty) {
                    if (!context.mounted) return;
                    Provider.of<ErrorHandler>(
                      context,
                      listen: false,
                    ).showHint(l10n.settingsError, l10n.settingsInvalidLogPath);
                    return;
                  }
                  try {
                    final uri = Uri.file(p.dirname(_logFilePath));
                    await launchUrl(uri);
                  } catch (e, s) {
                    if (!context.mounted) return;
                    Provider.of<ErrorHandler>(
                      context,
                      listen: false,
                    ).handleError(e, s);
                  }
                },
              ),
            ],
          ),
        ),
        buildSectionDivider(),
        buildSectionHeader(context, l10n.settingsAbout),
        const SizedBox(height: 12),
        buildAboutSection(
          context,
          developerOptionsEnabled,
          _appVersion,
          _buildTime,
          _handleVersionTap,
        ),

        if (_hasUnsavedChanges) const SizedBox(height: 96),
      ],
    );

    return PopScope(
      canPop: !_hasUnsavedChanges,
      onPopInvokedWithResult: (didPop, result) async {
        if (didPop) {
          return;
        }
        final navigator = Navigator.of(context);
        final shouldPop = await _onWillPop();
        if (shouldPop && mounted) {
          navigator.pop(result);
        }
      },
      child: Stack(
        children: [
          Positioned.fill(child: page),
          Positioned(
            right: 24,
            bottom: 24,
            child: AnimatedSlide(
              duration: saveAllAnimationDuration,
              curve: Curves.easeOutCubic,
              offset: _hasUnsavedChanges ? Offset.zero : const Offset(0, 1),
              child: AnimatedOpacity(
                duration: saveAllAnimationDuration,
                curve: Curves.easeOutCubic,
                opacity: _hasUnsavedChanges ? 1 : 0,
                child: IgnorePointer(
                  ignoring: !_hasUnsavedChanges,
                  child: FilledButton(
                    onPressed: _saveAllChanges,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(FluentIcons.save),
                        const SizedBox(width: 8),
                        Text(l10n.settingsSaveAll),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
