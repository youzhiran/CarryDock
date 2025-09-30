import 'dart:io';

import 'package:carrydock/models/software.dart';
import 'package:carrydock/services/archive_extractor.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/services/software_service.dart';
import 'package:carrydock/utils/error_handler.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:carrydock/widgets/select_executable_dialog.dart';
import 'package:carrydock/widgets/software_list_tile.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:path/path.dart' as p;
import 'package:provider/provider.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  late final SoftwareService _softwareService;
  final SettingsService _settingsService = SettingsService();
  List<Software> _managedSoftware = [];
  List<Software> _unmanagedSoftware = [];
  bool _isLoading = true;
  bool _isDragHovering = false;
  bool _useGridLayout = false;
  bool _isReorderModeEnabled = false;
  static const Duration _dragOverlayAnimationDuration = Duration(
    milliseconds: 180,
  );

  String _suggestAlternativeName(String baseName) {
    const suffix = '_副本';
    final timestamp = DateTime.now().millisecondsSinceEpoch;
    if (baseName.isEmpty) {
      return '新软件$timestamp';
    }
    if (!baseName.endsWith(suffix)) {
      return '$baseName$suffix';
    }
    return '${baseName}_$timestamp';
  }

  Future<_DuplicateResolution?> _showDuplicateResolutionDialog(
    DuplicateSoftwareInfo info,
  ) async {
    if (!mounted) return null;

    final conflicts = <String>[];
    if (info.existingManagedSoftware != null) {
      conflicts.add(
        '已托管软件：${info.existingManagedSoftware!.name} (${p.basename(info.existingManagedSoftware!.installPath)})',
      );
    }
    if (info.installDirExists) {
      conflicts.add('安装目录已存在：${info.targetInstallPath}');
    }
    if (info.archiveExists) {
      conflicts.add('归档文件已存在：${p.basename(info.targetArchivePath)}');
    }

    var renameValue = _suggestAlternativeName(info.intendedName);
    final controller = TextEditingController(text: renameValue);
    return showDialog<_DuplicateResolution>(
      context: context,
      builder: (context) {
        return StatefulBuilder(
          builder: (context, setState) {
            final canRename = renameValue.trim().isNotEmpty;
            final screenWidth = MediaQuery.of(context).size.width;
            final maxDialogWidth = screenWidth >= 600
                ? (screenWidth * 0.6).clamp(420.0, 620.0)
                : (screenWidth - 32).clamp(280.0, screenWidth);
            return ContentDialog(
              title: const Text('检测到重复软件'),
              constraints: BoxConstraints(maxWidth: maxDialogWidth),
              content: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('系统检测到“${info.intendedName}”可能已经存在，请选择如何处理：'),
                  const SizedBox(height: 12),
                  if (conflicts.isNotEmpty)
                    ...conflicts.map(
                      (conflict) => Padding(
                        padding: const EdgeInsets.only(bottom: 4),
                        child: Text('• $conflict'),
                      ),
                    ),
                  const SizedBox(height: 12),
                  const Text('如果希望以其他名称保留新软件，请在下方输入新的安装名称：'),
                  const SizedBox(height: 8),
                  TextBox(
                    controller: controller,
                    onChanged: (value) {
                      setState(() {
                        renameValue = value;
                      });
                    },
                  ),
                ],
              ),
              actions: [
                Button(
                  child: const Text('取消'),
                  onPressed: () => Navigator.of(context).pop(),
                ),
                Button(
                  onPressed: canRename
                      ? () => Navigator.of(
                          context,
                        ).pop(_DuplicateResolution.rename(renameValue.trim()))
                      : null,
                  child: const Text('重命名后添加'),
                ),
                FilledButton(
                  child: const Text('覆盖现有软件'),
                  onPressed: () => Navigator.of(
                    context,
                  ).pop(_DuplicateResolution.overwrite()),
                ),
              ],
            );
          },
        );
      },
    );
  }

  @override
  void initState() {
    super.initState();
    // 从 Provider 获取 ErrorHandler 实例并创建 SoftwareService
    _softwareService = SoftwareService(
      errorHandler: Provider.of<ErrorHandler>(context, listen: false),
    );
    _loadSoftware();
  }

  @override
  void dispose() {
    super.dispose();
  }

  Future<void> _loadSoftware() async {
    setState(() => _isLoading = true);
    try {
      final softwareList = await _softwareService.getSoftwareList();
      if (mounted) {
        setState(() {
          _managedSoftware =
              softwareList
                  .where((s) => s.status == SoftwareStatus.managed)
                  .toList()
                ..sort((a, b) {
                  final orderCompare = a.sortOrder.compareTo(b.sortOrder);
                  if (orderCompare != 0) return orderCompare;
                  return a.name.toLowerCase().compareTo(b.name.toLowerCase());
                });

          _unmanagedSoftware =
              softwareList
                  .where((s) => s.status != SoftwareStatus.managed)
                  .toList()
                ..sort((a, b) {
                  final statusCompare = a.status.index.compareTo(
                    b.status.index,
                  );
                  if (statusCompare != 0) return statusCompare;
                  return a.name.toLowerCase().compareTo(b.name.toLowerCase());
                });
        });
      }
    } finally {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  Future<void> _handleOpenInstallDirectory() async {
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        await _showMessageDialog('安装路径未设置', '请先在设置页面配置绿色软件安装目录。');
        return;
      }

      final installDir = Directory(installPath);
      if (!await installDir.exists()) {
        await _showMessageDialog('目录不存在', '当前配置的安装目录不存在：$installPath');
        return;
      }

      await Process.start('explorer.exe', [installPath], runInShell: true);
    } catch (e, s) {
      logger.e('打开软件安装目录时发生异常', error: e, stackTrace: s);
      _softwareService.errorHandler?.handleError(e, s);
      await _showMessageDialog('操作失败', '无法打开软件安装目录，请重试。');
    }
  }

  /// 触发“扫描当前目录软件”：
  /// - 询问用户确认；
  /// - 显示进度；
  /// - 调用服务层批量压缩安装目录的子目录至归档目录；
  /// - 展示结果摘要并刷新列表。
  Future<void> _handleScanAndArchive() async {
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        await _showMessageDialog('安装路径未设置', '请先在设置页面配置绿色软件安装目录。');
        return;
      }

      String? archiveDir;
      try {
        archiveDir = await _softwareService.resolveArchiveDirectoryPath();
      } catch (_) {
        archiveDir = null;
      }
      final backupDirShown = archiveDir != null ? p.join(archiveDir, 'backup') : null;

      if (!mounted) return;
      bool withArchive = true;
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) {
          return StatefulBuilder(
            builder: (context, setState) {
              return ContentDialog(
                title: const Text('扫描并归档'),
                content: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '即将扫描安装目录（$installPath）下的所有子目录。',
                    ),
                    const SizedBox(height: 8),
                    Text(
                      withArchive
                          ? '将为每个目录创建带时间戳的备份 ZIP${backupDirShown != null ? '（保存到：$backupDirShown）' : ''}。'
                          : '不会创建备份归档，仅识别并加入软件列表。',
                    ),
                    const SizedBox(height: 12),
                    ToggleSwitch(
                      content: const Text('创建备份归档'),
                      checked: withArchive,
                      onChanged: (v) => setState(() => withArchive = v),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '关闭后将不创建 ZIP（归档），仅维护软件列表。',
                      style: FluentTheme.of(context).typography.caption,
                    ),
                  ],
                ),
                actions: [
                  Button(
                    child: const Text('取消'),
                    onPressed: () => Navigator.of(context).pop(false),
                  ),
                  FilledButton(
                    child: const Text('开始'),
                    onPressed: () => Navigator.of(context).pop(true),
                  ),
                ],
              );
            },
          );
        },
      );
      if (confirmed != true) return;

      // 显示带百分比的进度弹窗
      int done = 0;
      int total = 0;
      double progress = 0.0;
      String currentName = '';
      StateSetter? dialogSetState;
      showDialog(
        context: context,
        barrierDismissible: false,
        builder: (context) {
          final theme = FluentTheme.of(context);
          return StatefulBuilder(
            builder: (context, setState) {
              dialogSetState = setState;
              final percent = (progress * 100).clamp(0, 100).toStringAsFixed(0);
              return ContentDialog(
                title: const Text('正在扫描并添加'),
                content: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const SizedBox(height: 4),
                    Container(
                      width: 360,
                      height: 10,
                      decoration: BoxDecoration(
                        color: theme.resources.controlFillColorDefault,
                        borderRadius: BorderRadius.circular(6),
                        border: Border.all(
                          color: theme.resources.controlStrokeColorDefault,
                        ),
                      ),
                      child: Align(
                        alignment: Alignment.centerLeft,
                        child: FractionallySizedBox(
                          alignment: Alignment.centerLeft,
                          widthFactor: progress.isNaN ? 0 : progress.clamp(0.0, 1.0),
                          child: Container(
                            decoration: BoxDecoration(
                              color: theme.accentColor,
                              borderRadius: BorderRadius.circular(6),
                            ),
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text('进度：$done/$total ($percent%)'),
                    const SizedBox(height: 6),
                    if (currentName.isNotEmpty)
                      Text('当前：$currentName'),
                  ],
                ),
              );
            },
          );
        },
      );

      try {
        final summary = await _softwareService.archiveInstallSubdirectories(
          overwrite: false,
          manageRecognized: true,
          onProgress: (d, t) {
            if (dialogSetState != null) {
              dialogSetState!(() {
                done = d;
                total = t;
                progress = t > 0 ? d / t : 0.0;
              });
            }
          },
          onCurrentItem: (name) {
            if (dialogSetState != null) {
              dialogSetState!(() {
                currentName = name;
              });
            }
          },
          createBackup: withArchive,
        );
        if (!mounted) return;
        final sb = StringBuffer()
          ..writeln('安装目录：${summary.installDirPath}')
          ..writeln('归档目录：${summary.archiveDirPath}')
          ..writeln('总计扫描：${summary.total}')
          ..writeln('成功归档：${summary.archived.length}')
          ..writeln('已存在跳过：${summary.skippedExisting.length}')
          ..writeln('失败：${summary.failures.length}');
        if (summary.failures.isNotEmpty) {
          for (final f in summary.failures.take(5)) {
            sb.writeln('- ${f.name}: ${f.error}');
          }
          if (summary.failures.length > 5) {
            sb.writeln('... 其余 ${summary.failures.length - 5} 项略');
          }
        }
        _popSafely();
        // 扫描完成后的“统一关联”对话框
        if (summary.suggestions.isNotEmpty) {
          final suggestions = summary.suggestions;
          // 记录每项是否关联与所选候选
          final associate = <String, bool>{};
          final selectedMap = <String, String>{};
          for (final s in suggestions) {
            associate[s.installPath] = true;
            selectedMap[s.installPath] = s.candidates.first;
          }

          final confirmedAssoc = await showDialog<bool>(
            context: context,
            builder: (context) {
              final screen = MediaQuery.of(context).size;
              final maxDialogWidth = screen.width >= 600
                  ? (screen.width * 0.7).clamp(420.0, 720.0)
                  : (screen.width - 32).clamp(280.0, screen.width);
              final maxDialogHeight = (screen.height * 0.8).clamp(320.0, 820.0);
              return StatefulBuilder(
                builder: (context, setState) {
                  return ContentDialog(
                    constraints: BoxConstraints(
                      maxWidth: maxDialogWidth.toDouble(),
                      maxHeight: maxDialogHeight.toDouble(),
                    ),
                    title: Text('发现可关联归档（${suggestions.length} 项）'),
                    content: SizedBox(
                      width: double.infinity,
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('请选择需要直接关联到现有归档的项目：'),
                          const SizedBox(height: 8),
                          ConstrainedBox(
                            constraints: BoxConstraints(
                              // 列表区域高度自适应并可滚动
                              minHeight: 120,
                              maxHeight: (maxDialogHeight.toDouble() - 200).clamp(120.0, 520.0),
                            ),
                            child: ListView.builder(
                              itemCount: suggestions.length,
                              itemBuilder: (context, index) {
                                final s = suggestions[index];
                                final checked = associate[s.installPath] ?? false;
                                final items = s.candidates
                                    .map((p0) => ComboBoxItem<String>(
                                          value: p0,
                                          child: Text(p.basename(p0), overflow: TextOverflow.ellipsis),
                                        ))
                                    .toList();
                                final selected = selectedMap[s.installPath];
                                return Padding(
                                  padding: const EdgeInsets.symmetric(vertical: 6),
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      // 第一行：开关 + 名称（名称单独渲染，避免与 ToggleSwitch 内部布局叠加）
                                      Row(
                                        children: [
                                          ToggleSwitch(
                                            content: const SizedBox.shrink(),
                                            checked: checked,
                                            onChanged: (v) => setState(() => associate[s.installPath] = v),
                                          ),
                                          const SizedBox(width: 8),
                                          // 名称占满剩余空间，单行省略
                                          Expanded(
                                            child: Tooltip(
                                              message: s.displayName,
                                              child: Text(
                                                s.displayName,
                                                overflow: TextOverflow.ellipsis,
                                                maxLines: 1,
                                                softWrap: false,
                                              ),
                                            ),
                                          ),
                                        ],
                                      ),
                                      const SizedBox(height: 8),
                                      // 第二行：归档选择下拉，单个候选时禁用
                                      ComboBox<String>(
                                        isExpanded: true,
                                        items: items,
                                        value: selected,
                                        onChanged: (checked && s.candidates.length > 1)
                                            ? (v) => setState(() {
                                                  if (v != null) selectedMap[s.installPath] = v;
                                                })
                                            : null,
                                      ),
                                    ],
                                  ),
                                );
                              },
                            ),
                          ),
                        ],
                      ),
                    ),
                    actions: [
                      Button(
                        child: const Text('跳过全部'),
                        onPressed: () => Navigator.of(context).pop(false),
                      ),
                      FilledButton(
                        child: const Text('应用关联'),
                        onPressed: () => Navigator.of(context).pop(true),
                      ),
                    ],
                  );
                },
              );
            },
          );

          if (confirmedAssoc == true) {
            final selections = <ArchiveAssociationResolution>[];
            for (final s in suggestions) {
              final doAssoc = associate[s.installPath] == true;
              final sel = doAssoc ? (selectedMap[s.installPath] ?? '') : '';
              selections.add(ArchiveAssociationResolution(
                installPath: s.installPath,
                selectedArchivePath: doAssoc && sel.isNotEmpty ? sel : null,
              ));
            }
            await _softwareService.applyArchiveAssociations(
              resolutions: selections,
              createBackupForUnselected: withArchive,
            );
          }
        }

        await _showMessageDialog('扫描完成', sb.toString());
        await _loadSoftware();
      } catch (e, s) {
        logger.e('扫描并归档失败', error: e, stackTrace: s);
        _softwareService.errorHandler?.handleError(e, s);
        _popSafely();
        await _showMessageDialog('操作失败', '扫描/归档过程中发生错误，请稍后重试。');
      }
    } catch (e, s) {
      logger.e('扫描并归档操作触发失败', error: e, stackTrace: s);
      _softwareService.errorHandler?.handleError(e, s);
      await _showMessageDialog('操作失败', '无法开始扫描，请检查安装路径设置后重试。');
    }
  }

  Future<void> _showMessageDialog(String title, String content) async {
    if (!mounted) return;
    await showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          Button(
            child: const Text('确定'),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }

  void _showProgressDialog(String content) {
    showDialog(
      context: context,
      barrierDismissible: false, // 用户不能通过点击外部来关闭
      builder: (context) => ContentDialog(
        content: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const ProgressRing(),
            const SizedBox(width: 16),
            Text(content),
          ],
        ),
      ),
    );
  }

  void _popSafely() {
    if (!mounted) return;
    // 检查当前路由是否可以被弹出
    if (Navigator.of(context).canPop()) {
      try {
        // 使用原生的 Navigator.pop，而不是 go_router 的扩展方法
        Navigator.of(context).pop();
      } catch (e) {
        logger.w('尝试关闭对话框时出错 (可能已被关闭): $e');
      }
    }
  }

  Future<void> _performAddSoftwareAction(
    Future<AddSoftwareResult> Function() addAction,
  ) async {
    _showProgressDialog('正在处理文件，请稍候...');
    bool shouldReload = false;
    try {
      final result = await addAction();

      switch (result.type) {
        case AddSoftwareResultType.success:
          shouldReload = true;
          break;

        case AddSoftwareResultType.cancelled:
          break;

        case AddSoftwareResultType.needsSelection:
          _popSafely();
          if (!mounted) return;

          final pendingAddition = result.pendingAddition!;
          final selected = await showDialog<String>(
            context: context,
            barrierDismissible: true,
            builder: (context) => SelectExecutableDialog(
              executablePaths: pendingAddition.executablePaths,
            ),
          );

          if (selected != null) {
            if (!mounted) return;
            _showProgressDialog('正在安装软件...');
            await _softwareService.completeSoftwareAddition(
              installPath: pendingAddition.installPath,
              archivePath: pendingAddition.archivePath,
              selectedExecutablePath: selected,
              preferredSortOrder: pendingAddition.preferredSortOrder,
            );
            shouldReload = true;
          } else {
            if (!mounted) return;
            _showProgressDialog('正在取消操作...');
            await _softwareService.cleanupTemporaryFiles(
              installPath: pendingAddition.installPath,
              archivePath: pendingAddition.archivePath,
            );
          }
          break;

        case AddSoftwareResultType.duplicate:
          _popSafely();
          if (!mounted) return;
          final info = result.duplicateInfo;
          if (info == null) {
            logger.w('重复软件信息缺失，无法继续处理');
            break;
          }

          final resolution = await _showDuplicateResolutionDialog(info);
          if (resolution == null) {
            break;
          }

          await _performAddSoftwareAction(
            () => _softwareService.resolveDuplicateAddition(
              info: info,
              overwriteExisting:
                  resolution.type == _DuplicateResolutionType.overwrite,
              renameTo: resolution.type == _DuplicateResolutionType.rename
                  ? resolution.rename
                  : null,
            ),
          );
          return;

        case AddSoftwareResultType.error:
          break;
      }
    } catch (e, s) {
      logger.e('处理添加软件操作时发生未捕获的错误', error: e, stackTrace: s);
      _softwareService.errorHandler?.handleError(e, s);
    } finally {
      if (shouldReload) {
        await _loadSoftware();
      }
      _popSafely();
    }
  }

  Future<void> _performRehostAction(Software software) async {
    _showProgressDialog('正在从归档恢复，请稍候...');
    bool shouldReload = false;
    try {
      final result = await _softwareService.rehostSoftware(software);
      switch (result.type) {
        case AddSoftwareResultType.success:
          shouldReload = true;
          break;
        case AddSoftwareResultType.needsSelection:
          _popSafely();
          if (!mounted) return;
          final pending = result.pendingAddition!;
          final selected = await showDialog<String>(
            context: context,
            barrierDismissible: true,
            builder: (context) => SelectExecutableDialog(
              executablePaths: pending.executablePaths,
            ),
          );
          if (selected != null) {
            if (!mounted) return;
            _showProgressDialog('正在完成重新托管...');
            await _softwareService.completeSoftwareRehost(
              existingId: pending.existingSoftwareId ?? software.id,
              installPath: pending.installPath,
              archivePath: pending.archivePath,
              selectedExecutablePath: selected,
            );
            shouldReload = true;
          } else {
            if (!mounted) return;
            _showProgressDialog('正在取消操作...');
            await _softwareService.cleanupTemporaryFiles(
              installPath: pending.installPath,
            );
          }
          break;
        case AddSoftwareResultType.cancelled:
        case AddSoftwareResultType.duplicate:
        case AddSoftwareResultType.error:
          break;
      }
    } catch (e, s) {
      logger.e('重新托管失败', error: e, stackTrace: s);
      _softwareService.errorHandler?.handleError(e, s);
    } finally {
      _popSafely();
    }
    if (shouldReload) {
      await _loadSoftware();
    }
  }

  Future<void> _handleDroppedItems(DropDoneDetails detail) async {
    final candidatePaths = <String>{};

    void collectPath(DropItem item) {
      final maybePath = item.path;
      if (maybePath.isNotEmpty) {
        candidatePaths.add(maybePath);
      }
      if (item is DropItemDirectory) {
        for (final child in item.children) {
          collectPath(child);
        }
      }
    }

    for (final item in detail.files) {
      collectPath(item);
    }

    if (candidatePaths.isEmpty) {
      logger.w('拖放操作未检测到可用文件，忽略。');
      return;
    }

    final executableExtensions = await _settingsService
        .getExecutableExtensions();
    final normalizedExecutableExtensions = executableExtensions
        .map((ext) => ext.toLowerCase())
        .toSet();

    final validPaths = <String>[];
    for (final path in candidatePaths) {
      final archiveFormat = ArchiveExtractor.detectFormat(path);
      if (archiveFormat != null) {
        if (!validPaths.contains(path)) {
          validPaths.add(path);
        }
        continue;
      }

      final extension = p.extension(path).toLowerCase();
      final normalizedExtension = extension.startsWith('.')
          ? extension.substring(1)
          : extension;
      if (normalizedExecutableExtensions.contains(normalizedExtension)) {
        if (!validPaths.contains(path)) {
          validPaths.add(path);
        }
      }
    }

    if (validPaths.isEmpty) {
      await _showMessageDialog(
        '无法识别的文件',
        '请拖放支持的压缩包（ZIP/TAR）或已在设置中允许的可执行文件类型。',
      );
      return;
    }

    logger.i('检测到拖放文件: ${validPaths.join(', ')}');

    for (final path in validPaths) {
      await _performAddSoftwareAction(
        () => _softwareService.addSoftwareFromFile(path),
      );
    }
  }

  Future<void> _handleAddSoftware() async {
    await _performAddSoftwareAction(() => _softwareService.addSoftware());
  }

  Future<void> _handleChangeExecutable(Software software) async {
    final installDir = Directory(software.installPath);
    final executables = await _softwareService.findExecutablesInDirectory(
      installDir,
    );

    if (executables.length <= 1) {
      await _showMessageDialog('无法切换主程序', '当前安装目录仅检测到一个可执行文件，无需更换主程序。');
      return;
    }

    if (!mounted) return;
    final selected = await showDialog<String>(
      context: context,
      barrierDismissible: true,
      builder: (context) =>
          SelectExecutableDialog(executablePaths: executables),
    );

    if (selected != null && selected != software.executablePath) {
      await _softwareService.updateSoftwareExecutable(software.id, selected);
      await _loadSoftware();
    }
  }

  Future<void> _handleLaunchSoftware(
    Software software, {
    String? executablePath,
  }) async {
    final targetExecutable = executablePath ?? software.executablePath;
    final installDirExists = await Directory(software.installPath).exists();
    if (!installDirExists) {
      await _showMessageDialog(
        '无法启动软件',
        software.archiveExists
            ? '安装目录不存在。您可以尝试使用“重新托管”从归档恢复。'
            : '安装目录不存在，且未找到归档文件。',
      );
      return;
    }
    if (software.status != SoftwareStatus.managed || targetExecutable.isEmpty) {
      logger.w('无法启动软件，因为状态不是 managed 或可执行路径为空: ${software.name}');
      await _showMessageDialog('无法启动软件', '请确认该软件已托管并且已经配置主程序路径后再尝试启动。');
      return;
    }

    logger.i('正在尝试启动软件: ${software.name}');
    logger.d('  可执行文件: $targetExecutable');
    logger.d('  工作目录: ${software.installPath}');

    try {
      final process = await Process.start(
        'cmd.exe',
        ['/c', 'start', '""', targetExecutable],
        workingDirectory: software.installPath,
        runInShell: true,
      );
      final exitCode = await process.exitCode;
      if (exitCode == 0) {
        logger.i('软件 ${software.name} 启动成功。');
      } else {
        logger.e('启动软件 ${software.name} 失败，退出代码: $exitCode');
        await _showMessageDialog(
          '启动失败',
          '程序返回退出代码 $exitCode。请检查该软件是否仍然可用或需要管理员权限。',
        );
      }
    } catch (e, s) {
      logger.e('启动软件 ${software.name} 时发生异常', error: e, stackTrace: s);
      _softwareService.errorHandler?.handleError(e, s);
      await _showMessageDialog('启动失败', '尝试启动软件时发生异常：${e.toString()}');
    }
  }

  void _showDeleteConfirmDialog(Software software) {
    bool deleteInstallDir = false;
    bool deleteArchive = false;

    if (software.status == SoftwareStatus.unknownInstall) {
      deleteInstallDir = true;
    }
    if (software.status == SoftwareStatus.unknownArchive) {
      deleteArchive = true;
    }

    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: Text('删除 ${software.name}'),
        content: StatefulBuilder(
          builder: (context, setState) {
            if (software.status != SoftwareStatus.managed) {
              return Text(
                '您确定要删除这个${software.status == SoftwareStatus.unknownInstall ? '未知文件夹' : '未知归档文件'}吗？此操作不可恢复。',
              );
            }
            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('您确定要删除此软件吗？此操作将从列表中移除该软件。'),
                const SizedBox(height: 16),
                Checkbox(
                  checked: deleteInstallDir,
                  onChanged: (v) =>
                      setState(() => deleteInstallDir = v ?? false),
                  content: const Text('同时删除安装目录'),
                ),
                const SizedBox(height: 8),
                Checkbox(
                  checked: deleteArchive,
                  onChanged: software.archiveExists
                      ? (v) => setState(() => deleteArchive = v ?? false)
                      : null,
                  content: Text(
                    '同时删除归档文件',
                    style: TextStyle(
                      color: software.archiveExists
                          ? null
                          : FluentTheme.of(
                              context,
                            ).resources.textFillColorDisabled,
                    ),
                  ),
                ),
              ],
            );
          },
        ),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.of(context).pop(),
          ),
          FilledButton(
            style: ButtonStyle(
              backgroundColor: WidgetStateProperty.all(Colors.red),
            ),
            child: const Text('删除'),
            onPressed: () async {
              Navigator.of(context).pop();
              await _softwareService.deleteSoftware(
                software,
                deleteInstallDir: deleteInstallDir,
                deleteArchive: deleteArchive,
              );
              await _loadSoftware();
            },
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final overlayTitleStyle =
        theme.typography.subtitle ??
        theme.typography.bodyStrong ??
        theme.typography.body;
    final overlayHintStyle = theme.typography.caption ?? theme.typography.body;

    final page = ScaffoldPage.scrollable(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
      header: PageHeader(
        title: Row(
          children: [
            Text('已托管软件', style: theme.typography.title),
            const SizedBox(width: 16),
            Expanded(
              child: CommandBar(
                key: ValueKey('${_useGridLayout}_$_isReorderModeEnabled'),
                mainAxisAlignment: MainAxisAlignment.end,
                primaryItems: _isReorderModeEnabled
                    ? [
                        CommandBarButton(
                          icon: const Icon(FluentIcons.check_mark),
                          label: const Text('完成'),
                          onPressed: () async {
                            final orderedIds = _managedSoftware
                                .map((s) => s.id)
                                .toList();
                            await _softwareService.updateManagedSoftwareOrder(
                              orderedIds,
                            );
                            setState(() {
                              _isReorderModeEnabled = false;
                            });
                            await _loadSoftware();
                          },
                        ),
                      ]
                    : [
                        CommandBarButton(
                          icon: Icon(
                            _useGridLayout
                                ? FluentIcons.bulleted_list
                                : FluentIcons.grid_view_small,
                          ),
                          label: Text(_useGridLayout ? '列表视图' : '网格视图'),
                          onPressed: () {
                            setState(() {
                              _useGridLayout = !_useGridLayout;
                            });
                          },
                        ),
                        CommandBarButton(
                          icon: const Icon(FluentIcons.add),
                          label: const Text('添加'),
                          onPressed: _handleAddSoftware,
                        ),
                        CommandBarButton(
                          icon: const Icon(FluentIcons.refresh),
                          label: const Text('刷新'),
                          onPressed: _loadSoftware,
                        ),
                        if (!_useGridLayout)
                          CommandBarButton(
                            icon: const Icon(FluentIcons.sort_lines),
                            label: const Text('调整排序'),
                            onPressed: () {
                              setState(() {
                                _isReorderModeEnabled = true;
                              });
                            },
                          ),
                        CommandBarButton(
                          icon: const Icon(FluentIcons.folder_open),
                          label: const Text('打开安装目录'),
                          onPressed: () {
                            _handleOpenInstallDirectory();
                          },
                        ),
                      ],
                // 将“扫描当前目录软件”默认放在更多（Overflow）中
                secondaryItems: [
                  CommandBarButton(
                    icon: const Icon(FluentIcons.search),
                    label: const Text('扫描当前目录软件'),
                    onPressed: _handleScanAndArchive,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
      children: [
        if (_isLoading)
          const Center(child: ProgressRing())
        else if (_managedSoftware.isEmpty && _unmanagedSoftware.isEmpty)
          const Center(child: Text('尚未添加任何软件。'))
        else if (_useGridLayout)
          GridView.builder(
            padding: EdgeInsets.zero,
            gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
              maxCrossAxisExtent: 220,
              mainAxisSpacing: 16,
              crossAxisSpacing: 16,
              childAspectRatio: 0.9,
            ),
            itemCount: _managedSoftware.length + _unmanagedSoftware.length,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemBuilder: (context, index) {
              final software = index < _managedSoftware.length
                  ? _managedSoftware[index]
                  : _unmanagedSoftware[index - _managedSoftware.length];
              return SoftwareListTile(
                key: ValueKey(software.id),
                software: software,
                onDelete: () => _showDeleteConfirmDialog(software),
                onChangeExecutable: () => _handleChangeExecutable(software),
                onLaunch: () => _handleLaunchSoftware(software),
                onLaunchAlternative: (path) =>
                    _handleLaunchSoftware(software, executablePath: path),
                onRehost: ((software.status == SoftwareStatus.unknownInstall ||
                            (software.status == SoftwareStatus.managed && !software.installExists)) &&
                        software.archiveExists)
                    ? () => _performRehostAction(software)
                    : null,
                displayStyle: SoftwareTileDisplay.grid,
              );
            },
          )
        else
          Column(
            children: [
              ReorderableListView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: _managedSoftware.length,
                // 关闭默认“两个横线”
                buildDefaultDragHandles: false,
                // 自定义拖拽时的显示效果
                proxyDecorator:
                    (Widget child, int index, Animation<double> animation) {
                      return Container(
                        decoration: BoxDecoration(
                          color: Colors.white, // 白色背景
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(
                            // 用 Fluent 的系统边框色，保证跟主题一致
                            color: theme.resources.controlStrokeColorDefault,
                          ),
                        ),
                        child: child,
                      );
                    },
                itemBuilder: (context, index) {
                  final software = _managedSoftware[index];
                  // 你的原始 tile
                  Widget tile = SoftwareListTile(
                    software: software,
                    onDelete: () => _showDeleteConfirmDialog(software),
                    onChangeExecutable: () => _handleChangeExecutable(software),
                    onLaunch: () => _handleLaunchSoftware(software),
                    onLaunchAlternative: (path) =>
                        _handleLaunchSoftware(software, executablePath: path),
                    onRehost: ((software.status == SoftwareStatus.unknownInstall ||
                                (software.status == SoftwareStatus.managed && !software.installExists)) &&
                            software.archiveExists)
                        ? () => _performRehostAction(software)
                        : null,
                    displayStyle: SoftwareTileDisplay.list,
                    isReorderMode: _isReorderModeEnabled,
                  );

                  // 在“调整排序”模式下，用 ReorderableDragStartListener 包住整个 tile
                  if (_isReorderModeEnabled) {
                    tile = ReorderableDragStartListener(
                      index: index,
                      // 可选：为了避免误触内部按钮，排序模式下让 tile 不响应子控件点击
                      child: AbsorbPointer(absorbing: true, child: tile),
                    );
                  }

                  // 注意：key 要在 ReorderableListView 的直接子节点上
                  return Container(key: ValueKey(software.id), child: tile);
                },
                onReorder: (oldIndex, newIndex) {
                  setState(() {
                    if (newIndex > oldIndex) {
                      newIndex -= 1;
                    }
                    final item = _managedSoftware.removeAt(oldIndex);
                    _managedSoftware.insert(newIndex, item);
                  });
                },
              ),
              ListView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: _unmanagedSoftware.length,
                itemBuilder: (context, index) {
                  final software = _unmanagedSoftware[index];
                  return SoftwareListTile(
                    key: ValueKey(software.id),
                    software: software,
                    onDelete: () => _showDeleteConfirmDialog(software),
                    onChangeExecutable: () => _handleChangeExecutable(software),
                    onLaunch: () => _handleLaunchSoftware(software),
                    onLaunchAlternative: (path) =>
                        _handleLaunchSoftware(software, executablePath: path),
                    onRehost: ((software.status == SoftwareStatus.unknownInstall ||
                                (software.status == SoftwareStatus.managed && !software.installExists)) &&
                            software.archiveExists)
                        ? () => _performRehostAction(software)
                        : null,
                    displayStyle: SoftwareTileDisplay.list,
                  );
                },
              ),
            ],
          ),
      ],
    );

    final overlayBorderRadius = BorderRadius.circular(12);

    return DropTarget(
      onDragEntered: (_) {
        if (!_isDragHovering) {
          setState(() => _isDragHovering = true);
        }
      },
      onDragExited: (_) {
        if (_isDragHovering) {
          setState(() => _isDragHovering = false);
        }
      },
      onDragDone: (detail) async {
        if (_isDragHovering) {
          setState(() => _isDragHovering = false);
        }
        await _handleDroppedItems(detail);
      },
      child: Stack(
        children: [
          AnimatedOpacity(
            opacity: _isDragHovering ? 0.1 : 1.0,
            duration: _dragOverlayAnimationDuration,
            curve: Curves.easeOutCubic,
            child: page,
          ),
          if (_isDragHovering)
            Positioned.fill(
              child: IgnorePointer(
                child: Container(
                  margin: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    borderRadius: overlayBorderRadius,
                    border: Border.all(color: theme.accentColor, width: 2),
                    color: theme.resources.layerFillColorDefault.withValues(
                      alpha: 0.92,
                    ),
                  ),
                  child: Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          FluentIcons.add,
                          size: 36,
                          color: theme.accentColor,
                        ),
                        const SizedBox(height: 12),
                        Text('松开以添加软件', style: overlayTitleStyle),
                        const SizedBox(height: 4),
                        Text(
                          '支持 ZIP/TAR 压缩包或已配置的可执行文件',
                          style: overlayHintStyle,
                        ),
                      ],
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

enum _DuplicateResolutionType { overwrite, rename }

class _DuplicateResolution {
  final _DuplicateResolutionType type;
  final String? rename;

  const _DuplicateResolution._(this.type, this.rename);

  factory _DuplicateResolution.overwrite() =>
      const _DuplicateResolution._(_DuplicateResolutionType.overwrite, null);

  factory _DuplicateResolution.rename(String name) =>
      _DuplicateResolution._(_DuplicateResolutionType.rename, name);
}

class _SoftwareOrderDialog extends StatefulWidget {
  final List<Software> initialOrder;

  const _SoftwareOrderDialog({required this.initialOrder});

  @override
  State<_SoftwareOrderDialog> createState() => _SoftwareOrderDialogState();
}

class _SoftwareOrderDialogState extends State<_SoftwareOrderDialog> {
  late List<Software> _orderedList;

  @override
  void initState() {
    super.initState();
    _orderedList = List<Software>.from(widget.initialOrder);
  }

  void _moveItem(int from, int to) {
    setState(() {
      final item = _orderedList.removeAt(from);
      _orderedList.insert(to, item);
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final resources = theme.resources;
    final double listHeight = (_orderedList.length * 56.0)
        .clamp(140.0, 320.0)
        .toDouble();

    return ContentDialog(
      title: const Text('调整软件顺序'),
      constraints: const BoxConstraints(maxWidth: 440),
      content: SizedBox(
        width: double.infinity,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              height: listHeight,
              child: ListView.separated(
                itemCount: _orderedList.length,
                separatorBuilder: (context, index) => const SizedBox(height: 6),
                itemBuilder: (context, index) {
                  final software = _orderedList[index];
                  final canMoveUp = index > 0;
                  final canMoveDown = index < _orderedList.length - 1;
                  return Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 10,
                    ),
                    decoration: BoxDecoration(
                      color: resources.controlFillColorSecondary,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(
                        color: resources.controlStrokeColorDefault,
                      ),
                    ),
                    child: Row(
                      children: [
                        Expanded(
                          child: Text(
                            software.name,
                            style: theme.typography.body,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 12),
                        Tooltip(
                          message: '向上移动',
                          child: IconButton(
                            icon: const Icon(FluentIcons.up),
                            onPressed: canMoveUp
                                ? () => _moveItem(index, index - 1)
                                : null,
                          ),
                        ),
                        Tooltip(
                          message: '向下移动',
                          child: IconButton(
                            icon: const Icon(FluentIcons.down),
                            onPressed: canMoveDown
                                ? () => _moveItem(index, index + 1)
                                : null,
                          ),
                        ),
                      ],
                    ),
                  );
                },
              ),
            ),
            const SizedBox(height: 12),
            Text(
              '通过点击右侧箭头可调整展示顺序，保存后将在主页生效。',
              style: theme.typography.caption ?? theme.typography.body,
            ),
          ],
        ),
      ),
      actions: [
        Button(
          child: const Text('取消'),
          onPressed: () => Navigator.of(context).pop(),
        ),
        FilledButton(
          child: const Text('保存顺序'),
          onPressed: () =>
              Navigator.of(context).pop(List<Software>.from(_orderedList)),
        ),
      ],
    );
  }
}
