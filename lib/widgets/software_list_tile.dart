import 'dart:io';

import 'package:carrydock/l10n/app_localizations.dart';
import 'package:carrydock/models/software.dart';
import 'package:carrydock/services/executable_info_service.dart';
import 'package:carrydock/services/software_service.dart';
import 'package:carrydock/utils/file_utils.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/services.dart';
import 'package:path/path.dart' as p;

/// 控制软件条目呈现形式，支持列表与网格。
enum SoftwareTileDisplay { list, grid }

class SoftwareListTile extends StatefulWidget {
  final Software software;
  final VoidCallback onDelete;
  final VoidCallback onChangeExecutable;
  final VoidCallback onLaunch;
  final ValueChanged<String> onLaunchAlternative;
  final VoidCallback? onRehost;
  final VoidCallback? onRefresh;
  final SoftwareTileDisplay displayStyle;
  final bool isReorderMode;

  const SoftwareListTile({
    super.key,
    required this.software,
    required this.onDelete,
    required this.onChangeExecutable,
    required this.onLaunch,
    required this.onLaunchAlternative,
    this.onRehost,
    this.onRefresh,
    this.displayStyle = SoftwareTileDisplay.list,
    this.isReorderMode = false,
  });

  @override
  State<SoftwareListTile> createState() => _SoftwareListTileState();
}

class _SoftwareListTileState extends State<SoftwareListTile> {
  final ExecutableInfoService _infoService = ExecutableInfoService();
  final SoftwareService _softwareService = SoftwareService();
  final FlyoutController _alternativeFlyoutController = FlyoutController();
  final FlyoutController _contextMenuController = FlyoutController();
  Uint8List? _iconData;
  String? _fileDescription;
  bool _isLoading = true;
  bool _isExecutableLoading = true;
  List<String> _availableExecutables = const [];
  bool _isContextMenuActive = false;
  bool _isGridTileHovered = false;

  /// 构建归档存在状态的小徽章，统一高度并居中，确保与右侧图标按钮上下对齐。
  Widget _buildArchiveStatusPill({required bool exists, String? tooltip}) {
    // 备份归档使用蓝色强调（无论条目状态为何），否则按照存在状态使用绿色/红色
    final bool isBackup = widget.software.isBackupArchive;
    final Color accent = isBackup
        ? Colors.blue
        : (exists ? Colors.green : Colors.red);
    final pill = ConstrainedBox(
      constraints: const BoxConstraints(minHeight: 28),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8),
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: accent.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Icon(FluentIcons.archive, size: 14, color: accent),
      ),
    );
    if (tooltip != null && tooltip.isNotEmpty) {
      return Tooltip(message: tooltip, child: pill);
    }
    return pill;
  }

  /// 构建小徽章（图标+颜色），用于显示归档/备份状态。
  Widget _buildStatusBadge({
    required IconData icon,
    required Color color,
    String? tooltip,
  }) {
    final pill = ConstrainedBox(
      constraints: const BoxConstraints(minHeight: 28),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8),
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: color),
        ),
        child: Icon(icon, size: 14, color: color),
      ),
    );
    if (tooltip != null && tooltip.isNotEmpty) {
      return Tooltip(message: tooltip, child: pill);
    }
    return pill;
  }

  /// 同时展示"归档状态（绿/红）"与"备份状态（蓝/灰）"。
  Widget _buildArchiveAndBackupBadges(Software s) {
    final l10n = AppLocalizations.of(context)!;
    final hasArchive = s.archiveExists;
    // 按最新约定：仅根据服务层扫描结果判断"备份存在"
    final hasBackup = s.isBackupArchive;
    final widgets = <Widget>[];
    widgets.add(
      _buildStatusBadge(
        icon: FluentIcons.archive,
        color: hasArchive ? Colors.green : Colors.red,
        tooltip: hasArchive ? l10n.tileArchiveExists : l10n.tileArchiveNotExists,
      ),
    );
    widgets.add(const SizedBox(width: 6));
    widgets.add(
      _buildStatusBadge(
        icon: FluentIcons.update_restore,
        color: hasBackup ? Colors.blue : Colors.grey,
        tooltip: hasBackup ? l10n.tileBackupDetected : l10n.tileBackupNotDetected,
      ),
    );
    return Row(mainAxisSize: MainAxisSize.min, children: widgets);
  }

  @override
  void initState() {
    super.initState();
    _loadExeInfo();
    _loadExecutableOptions();
    _contextMenuController.addListener(_handleContextMenuStateChange);
  }

  Future<void> _loadExeInfo() async {
    if (widget.software.status != SoftwareStatus.managed ||
        widget.software.executablePath.isEmpty) {
      setState(() => _isLoading = false);
      return;
    }

    try {
      final icon = await _infoService.getIcon(widget.software.executablePath);
      final desc = await _infoService.getFileDescription(
        widget.software.executablePath,
      );
      if (mounted) {
        setState(() {
          _iconData = icon;
          _fileDescription = desc;
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  Future<void> _loadExecutableOptions() async {
    if (widget.software.status != SoftwareStatus.managed ||
        widget.software.installPath.isEmpty) {
      setState(() {
        _availableExecutables = const [];
        _isExecutableLoading = false;
      });
      return;
    }

    final installDir = Directory(widget.software.installPath);
    if (!await installDir.exists()) {
      setState(() {
        _availableExecutables = const [];
        _isExecutableLoading = false;
      });
      return;
    }

    try {
      final executables = await _softwareService.findExecutablesInDirectory(
        installDir,
      );
      if (!mounted) return;
      setState(() {
        _availableExecutables = executables;
        _isExecutableLoading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _availableExecutables = const [];
        _isExecutableLoading = false;
      });
    }
  }

  @override
  void didUpdateWidget(covariant SoftwareListTile oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.software.id != widget.software.id ||
        oldWidget.software.executablePath != widget.software.executablePath ||
        oldWidget.software.installPath != widget.software.installPath) {
      setState(() {
        _isLoading = true;
        _isExecutableLoading = true;
        _availableExecutables = const [];
      });
      _loadExeInfo();
      _loadExecutableOptions();
    }
  }

  @override
  void dispose() {
    _alternativeFlyoutController.dispose();
    _contextMenuController.removeListener(_handleContextMenuStateChange);
    _contextMenuController.dispose();
    super.dispose();
  }

  void _handleContextMenuStateChange() {
    final shouldHighlight = _contextMenuController.isOpen;
    if (shouldHighlight != _isContextMenuActive) {
      setState(() {
        _isContextMenuActive = shouldHighlight;
      });
    }
  }

  void _setGridHovering(bool hovering) {
    if (_isGridTileHovered == hovering) {
      return;
    }
    setState(() {
      _isGridTileHovered = hovering;
    });
  }

  /// 根据指定尺寸构建图标或加载中的占位。
  Widget _buildIcon(double size) {
    if (_isLoading) {
      return SizedBox(
        width: size,
        height: size,
        child: ProgressRing(strokeWidth: size >= 48 ? 4 : 2),
      );
    }
    if (_iconData != null) {
      return Image.memory(
        _iconData!,
        width: size,
        height: size,
        fit: BoxFit.contain,
      );
    }
    // 备份归档：使用蓝色归档图标以示区分
    if (widget.software.status == SoftwareStatus.unknownArchive &&
        widget.software.isBackupArchive) {
      return Icon(FluentIcons.archive, size: size, color: Colors.blue);
    }
    return Icon(FluentIcons.app_icon_default, size: size);
  }

  /// 在页面右上角弹出信息条提示用户。
  void _showInfoBar(
    String title,
    String message, {
    InfoBarSeverity severity = InfoBarSeverity.warning,
  }) {
    if (!mounted) {
      return;
    }
    displayInfoBar(
      context,
      builder: (context, close) {
        return InfoBar(
          title: Text(title),
          content: Text(message),
          action: IconButton(
            icon: const Icon(FluentIcons.clear),
            onPressed: close,
          ),
          severity: severity,
        );
      },
    );
  }

  /// 打开当前软件的安装目录，不存在时提供友好提示。
  Future<void> _openInstallDirectory() async {
    final l10n = AppLocalizations.of(context)!;
    final installPath = widget.software.installPath;
    if (installPath.isEmpty) {
      _showInfoBar(l10n.tileHintNoInstallDir, l10n.tileHintNoInstallDirMessage);
      return;
    }
    final directory = Directory(installPath);
    if (!await directory.exists()) {
      _showInfoBar(l10n.tileHintNoInstallDir, l10n.tileHintCannotOpenDir(installPath));
      return;
    }
    try {
      await Process.start('explorer.exe', [installPath], runInShell: true);
    } catch (e, s) {
      logger.e('打开安装目录失败', error: e, stackTrace: s);
      _showInfoBar(l10n.tileError, l10n.tileErrorCannotOpenDirMessage, severity: InfoBarSeverity.error);
    }
  }

  /// 修改软件文件夹：允许用户直接输入新的安装目录路径
  Future<void> _changeSoftwareFolder() async {
    final l10n = AppLocalizations.of(context)!;
    try {
      // 获取当前软件名称作为默认新目录名
      final currentDirName = p.basename(widget.software.installPath);
      final parentDir = p.dirname(widget.software.installPath);

      // 弹出对话框让用户输入新路径
      final newPathResult = await showDialog<String>(
        context: context,
        builder: (context) {
          TextEditingController controller = TextEditingController(
            text: currentDirName,
          );

          return ContentDialog(
            title: Text(l10n.tileChangeSoftwareFolder),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(l10n.tileCurrentPath),
                Text(
                  widget.software.installPath,
                  style: FluentTheme.of(context).typography.caption,
                ),
                const SizedBox(height: 16),
                Text(l10n.tileNewFolderName),
                const SizedBox(height: 8),
                TextBox(
                  controller: controller,
                  placeholder: l10n.tileNewFolderPlaceholder,
                  autofocus: true,
                ),
                const SizedBox(height: 8),
                Text(
                  l10n.tileNewPathHint(p.join(parentDir, '')),
                  style: FluentTheme.of(context).typography.caption,
                ),
              ],
            ),
            actions: [
              Button(
                child: Text(l10n.tileRenameCancel),
                onPressed: () => Navigator.pop(context, null),
              ),
              Button(
                onPressed: () => Navigator.pop(context, controller.text.trim()),
                style: ButtonStyle(
                  backgroundColor: WidgetStateProperty.resolveWith((states) {
                    if (states.contains(WidgetState.pressed)) {
                      return FluentTheme.of(
                        context,
                      ).accentColor.withValues(alpha: 0.8);
                    }
                    return FluentTheme.of(context).accentColor;
                  }),
                  foregroundColor: WidgetStateProperty.all(Colors.white),
                ),
                child: Text(l10n.tileRenameConfirm),
              ),
            ],
          );
        },
      );

      if (newPathResult == null || newPathResult.isEmpty) {
        // 用户取消或输入为空
        return;
      }

      // 构建新的完整路径
      final newInstallPath = p.join(parentDir, newPathResult);
      final oldInstallPath = widget.software.installPath;

      // 如果路径相同，不需要更新
      if (p.equals(newInstallPath, oldInstallPath)) {
        _showInfoBar(l10n.tileHintSamePath, l10n.tileHintSamePathMessage);
        return;
      }

      // 检查旧目录是否存在
      final oldDir = Directory(oldInstallPath);
      if (!await oldDir.exists()) {
        _showInfoBar(l10n.tileError, l10n.tileErrorFolderNotExist, severity: InfoBarSeverity.error);
        return;
      }

      // 检查新目录是否已存在
      final newDir = Directory(newInstallPath);
      if (await newDir.exists()) {
        _showInfoBar(l10n.tileError, l10n.tileErrorFolderExists, severity: InfoBarSeverity.error);
        return;
      }

      // 显示迁移进度对话框
      if (!mounted) return;
      showDialog(
        context: context,
        barrierDismissible: false,
        builder: (dialogContext) => ContentDialog(
          content: Row(
            mainAxisSize: MainAxisSize.min,
            children: [ProgressRing(), SizedBox(width: 12), Text(l10n.tileMigrating)],
          ),
        ),
      );

      // 创建新目录
      await newDir.create(recursive: true);

      // 迁移文件：复制旧目录所有内容到新目录
      await _copyDirectory(oldDir, newDir);

      // 更新可执行文件路径
      String? newExecutablePath;
      if (widget.software.executablePath.isNotEmpty) {
        final relativeExecPath = p.relative(
          widget.software.executablePath,
          from: oldInstallPath,
        );
        newExecutablePath = p.join(newInstallPath, relativeExecPath);
      }

      // 调用服务更新软件路径
      await _softwareService.updateSoftwarePath(
        softwareId: widget.software.id,
        newInstallPath: newInstallPath,
        newExecutablePath: newExecutablePath,
      );

      // 删除旧目录
      await oldDir.delete(recursive: true);

      // 关闭进度对话框
      if (mounted && Navigator.of(context).canPop()) {
        Navigator.of(context).pop();
      }

      _showInfoBar(l10n.tileSuccess, l10n.tileMigrateSuccess, severity: InfoBarSeverity.success);

      // 刷新界面：重新加载软件信息
      await _loadExeInfo();
      await _loadExecutableOptions();

      // 通知父组件重新加载软件列表
      if (widget.onRefresh != null) {
        widget.onRefresh!();
      }
    } catch (e, s) {
      logger.e('修改软件文件夹失败', error: e, stackTrace: s);

      // 关闭可能存在的进度对话框
      if (mounted && Navigator.of(context).canPop()) {
        Navigator.of(context).pop();
      }

      _showInfoBar(l10n.tileError, l10n.tileErrorMigrateFailed('$e'), severity: InfoBarSeverity.error);
    }
  }

  /// 复制目录及其所有内容
  Future<void> _copyDirectory(Directory source, Directory destination) async {
    await for (final entity in source.list(recursive: false)) {
      if (entity is Directory) {
        final newDir = Directory(
          p.join(destination.path, p.basename(entity.path)),
        );
        await newDir.create();
        await _copyDirectory(entity, newDir);
      } else if (entity is File) {
        await entity.copy(p.join(destination.path, p.basename(entity.path)));
      }
    }
  }

  /// 打开归档文件所在目录，优先高亮具体文件。
  Future<void> _openArchiveDirectory() async {
    final l10n = AppLocalizations.of(context)!;
    final archivePath = widget.software.archivePath;
    if (archivePath.isEmpty) {
      _showInfoBar(l10n.tileHintNoArchive, l10n.tileHintNoArchiveMessage);
      return;
    }
    final archiveFile = File(archivePath);
    final archiveDirectory = Directory(archivePath);
    final parentDirectory = Directory(p.dirname(archivePath));
    try {
      if (await archiveFile.exists()) {
        await Process.start('explorer.exe', [
          '/select,',
          archivePath,
        ], runInShell: true);
        return;
      }
      if (await archiveDirectory.exists()) {
        await Process.start('explorer.exe', [
          archiveDirectory.path,
        ], runInShell: true);
        _showInfoBar(l10n.tileHintNoArchive, l10n.tileHintArchiveNotExists);
        return;
      }
      if (await parentDirectory.exists()) {
        await Process.start('explorer.exe', [
          parentDirectory.path,
        ], runInShell: true);
        _showInfoBar(l10n.tileHintNoArchive, l10n.tileHintParentOpened);
        return;
      }
      _showInfoBar(l10n.tileHintNoArchive, l10n.tileHintArchiveNotFound);
    } catch (e, s) {
      logger.e('打开归档目录失败', error: e, stackTrace: s);
      _showInfoBar(l10n.tileError, l10n.tileErrorCannotOpenArchive, severity: InfoBarSeverity.error);
    }
  }

  /// 获取软件的显示名称，与列表显示保持一致
  String _getSoftwareDisplayName() {
    switch (widget.software.status) {
      case SoftwareStatus.managed:
        return _fileDescription ?? widget.software.name;
      case SoftwareStatus.unknownInstall:
      case SoftwareStatus.unknownArchive:
        return widget.software.name;
    }
  }
  
  /// 检查并添加绿驿管家自身的快捷方式
  Future<void> _checkAndAddCarryDockShortcut(Directory startMenuDir) async {
    final l10n = AppLocalizations.of(context)!;
    try {
      // 获取绿驿管家自身的可执行文件路径
      final appExePath = Platform.resolvedExecutable;
      final appName = l10n.tileCarryDock;
      final appShortcutPath = p.join(startMenuDir.path, '$appName.lnk');

      // 检查快捷方式是否已存在
      final appShortcutFile = File(appShortcutPath);
      if (!await appShortcutFile.exists()) {
        logger.i(l10n.tileCarryDockShortcutAdding);
        await FileUtils.createShortcut(appExePath, appShortcutPath, appName);
        logger.i(l10n.tileCarryDockShortcutAdded);
      }
    } catch (e, s) {
      logger.e('添加绿驿管家快捷方式失败', error: e, stackTrace: s);
      // 不影响主流程，继续执行
    }
  }

  /// 将软件添加到开始菜单。
  Future<void> _addToStartMenu() async {
    final l10n = AppLocalizations.of(context)!;
    final exePath = widget.software.executablePath;
    if (exePath.isEmpty) {
      _showInfoBar(l10n.tileHintNoExecutable, l10n.tileHintNoExecutableMessage);
      return;
    }

    try {
      // 获取 CarryDock 开始菜单目录
      final startMenuDir = await FileUtils.getCarryDockStartMenuDir();

      // 检查并添加绿驿管家自身的快捷方式
      await _checkAndAddCarryDockShortcut(startMenuDir);

      // 获取统一的名称
      final displayName = _getSoftwareDisplayName();
      // 使用显示名称作为快捷方式文件名称（去除空格和特殊字符）
      final sanitizedName = displayName.replaceAll(RegExp(r'[\/:*?"<>|]'), '_').trim();
      final shortcutPath = p.join(startMenuDir.path, '$sanitizedName.lnk');

      // 创建快捷方式
      await FileUtils.createShortcut(exePath, shortcutPath, displayName);

      _showInfoBar(
        l10n.tileSuccess,
        l10n.tileAddedToStartMenu(displayName),
        severity: InfoBarSeverity.success,
      );
    } catch (e, s) {
      logger.e('添加到开始菜单失败', error: e, stackTrace: s);
      _showInfoBar(
        l10n.tileError,
        l10n.tileErrorAddToStartMenu('$e'),
        severity: InfoBarSeverity.error,
      );
    }
  }

  /// 弹出右键菜单，提供常用目录的快捷入口。
  void _showContextMenu({Offset? position}) {
    final l10n = AppLocalizations.of(context)!;
    if (_contextMenuController.isAttached && _contextMenuController.isOpen) {
      _contextMenuController.close();
    }
    setState(() {
      _isContextMenuActive = true;
    });
    _contextMenuController.showFlyout(
      position: position,
      barrierColor: Colors.transparent,
      builder: (context) {
        final theme = FluentTheme.of(context);
        return DisableAcrylic(
          child: MenuFlyout(
            color: theme.cardColor,
            shadowColor: theme.shadowColor.withValues(alpha: 0.16),
            items: [
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.folder_open),
                text: Text(l10n.tileOpenFolder),
                onPressed: widget.software.installPath.isNotEmpty
                    ? _openInstallDirectory
                    : null,
              ),
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.edit),
                text: Text(l10n.tileChangeFolder),
                onPressed: widget.software.installPath.isNotEmpty
                    ? _changeSoftwareFolder
                    : null,
              ),
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.archive),
                text: Text(l10n.tileCreateBackup),
                onPressed: widget.software.installPath.isNotEmpty
                    ? () async {
                        final dir = Directory(widget.software.installPath);
                        if (!await dir.exists()) {
                          _showInfoBar(l10n.tileHintNoInstallDir, l10n.tileHintInstallDirNotExists);
                          return;
                        }
                        // 显示进度（非阻塞等待）
                        // 注意：不要 await showDialog，以免阻塞后续逻辑。
                        // 由后续逻辑中关闭对话框。
                        if (!context.mounted) return;
                        showDialog(
                          context: context,
                          barrierDismissible: false,
                          builder: (dialogContext) => ContentDialog(
                            content: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                const ProgressRing(),
                                const SizedBox(width: 12),
                                Text(l10n.tileCreatingBackup),
                              ],
                            ),
                          ),
                        );
                        try {
                          final created = await _softwareService
                              .createBackupForSoftware(widget.software);
                          if (mounted && Navigator.of(this.context).canPop()) {
                            Navigator.of(this.context).pop();
                          }
                          _showInfoBar(
                            l10n.tileSuccess,
                            l10n.tileBackupCreated(p.basename(created)),
                            severity: InfoBarSeverity.success,
                          );
                        } catch (e, s) {
                          logger.e('创建备份失败', error: e, stackTrace: s);
                          if (mounted && Navigator.of(this.context).canPop()) {
                            Navigator.of(this.context).pop();
                          }
                          _showInfoBar(
                            l10n.tileError,
                            l10n.tileErrorCreateBackup,
                            severity: InfoBarSeverity.error,
                          );
                        }
                      }
                    : null,
              ),
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.pin),
                text: Text(l10n.tileAddToStartMenu),
                onPressed: widget.software.executablePath.isNotEmpty
                    ? _addToStartMenu
                    : null,
              ),
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.open_file),
                text: Text(l10n.tileOpenArchiveFolder),
                onPressed: widget.software.archivePath.isNotEmpty
                    ? _openArchiveDirectory
                    : null,
              ),
            ],
          ),
        );
      },
    );
  }

  /// 包裹右键菜单监听逻辑，避免重复编写监听器。
  Widget _wrapWithContextMenu(Widget child) {
    return FlyoutTarget(
      controller: _contextMenuController,
      child: Listener(
        behavior: HitTestBehavior.deferToChild,
        onPointerDown: (event) {
          if (event.kind == PointerDeviceKind.mouse &&
              (event.buttons & kSecondaryMouseButton) != 0) {
            _showContextMenu(position: event.position);
          }
        },
        child: child,
      ),
    );
  }

  /// 构建网格布局下的卡片样式。
  Widget _buildGridTile({
    required BuildContext context,
    required Widget icon,
    required String title,
    required Color? titleColor,
    Widget? supplementary,
  }) {
    final theme = FluentTheme.of(context);
    final resources = theme.resources;
    final bool isContextMenuActive = _isContextMenuActive;
    final bool isHovering = _isGridTileHovered;
    final Color baseBackground = resources.controlFillColorSecondary;
    final Color hoverOverlay = resources.subtleFillColorSecondary;
    final Color backgroundColor;
    if (isContextMenuActive) {
      backgroundColor = theme.accentColor.withValues(alpha: 0.14);
    } else if (isHovering) {
      // 叠加列表项悬停使用的浅灰色，确保网格悬停效果与列表一致
      backgroundColor = Color.alphaBlend(hoverOverlay, baseBackground);
    } else {
      backgroundColor = baseBackground;
    }
    final Color borderColor;
    if (isContextMenuActive) {
      borderColor = theme.accentColor;
    } else if (isHovering) {
      borderColor = resources.controlStrokeColorDefault;
    } else {
      borderColor = Colors.transparent;
    }

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => _setGridHovering(true),
      onExit: (_) => _setGridHovering(false),
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: widget.onLaunch,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          curve: Curves.easeOutCubic,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 20),
          decoration: BoxDecoration(
            color: backgroundColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: borderColor,
              width: isContextMenuActive ? 1.2 : 0.0,
            ),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              SizedBox(width: 64, height: 64, child: Center(child: icon)),
              const SizedBox(height: 12),
              Text(
                title,
                textAlign: TextAlign.center,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style:
                    (FluentTheme.of(context).typography.bodyStrong ??
                            const TextStyle())
                        .copyWith(color: titleColor),
              ),
              if (supplementary != null) ...[
                const SizedBox(height: 8),
                DefaultTextStyle.merge(
                  style:
                      (FluentTheme.of(context).typography.caption ??
                              const TextStyle())
                          .copyWith(color: titleColor),
                  textAlign: TextAlign.center,
                  child: supplementary,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  /// 构建列表布局下的详细条目。
  Widget _buildListTile({
    required BuildContext context,
    required Widget icon,
    required String title,
    required Widget subtitle,
    required Color? titleColor,
    required List<String> alternativeExecutables,
    required bool hasAlternativeExecutables,
    required bool canChangeExecutable,
    required String changeExecutableTooltip,
  }) {
    final l10n = AppLocalizations.of(context)!;
    const double controlSpacing = 8;
    const double alternativeSlotWidth = 36;

    Widget buildActionControl({
      required IconData icon,
      required String tooltip,
      required VoidCallback? onPressed,
      bool isDestructive = false,
      Color? color,
    }) {
      final Color? accentColor = isDestructive ? Colors.red : color;
      return Tooltip(
        message: tooltip,
        child: IconButton(
          style: isDestructive && onPressed != null
              ? ButtonStyle(
                  foregroundColor: WidgetStateProperty.all(Colors.red),
                )
              : null,
          icon: Icon(icon, color: onPressed != null ? accentColor : null),
          onPressed: onPressed,
        ),
      );
    }

    Widget buildAlternativeLaunchButton(List<String> alternatives) {
      final l10n = AppLocalizations.of(context)!;
      return FlyoutTarget(
        controller: _alternativeFlyoutController,
        child: Tooltip(
          message: l10n.tileLaunchOther,
          child: IconButton(
            icon: const Icon(FluentIcons.custom_activity),
            onPressed: () {
              if (alternatives.isEmpty) {
                if (_alternativeFlyoutController.isAttached &&
                    _alternativeFlyoutController.isOpen) {
                  _alternativeFlyoutController.close();
                }
                return;
              }
              _alternativeFlyoutController.showFlyout(
                builder: (context) => MenuFlyout(
                  items: alternatives
                      .map(
                        (path) => MenuFlyoutItem(
                          text: Text(p.basename(path)),
                          onPressed: () => widget.onLaunchAlternative(path),
                        ),
                      )
                      .toList(),
                ),
              );
            },
          ),
        ),
      );
    }

    Widget buildAlternativeSlot() {
      if (_isExecutableLoading) {
        return SizedBox(
          width: alternativeSlotWidth,
          child: Center(
            child: SizedBox(
              width: 22,
              height: 22,
              child: const ProgressRing(strokeWidth: 2),
            ),
          ),
        );
      }
      if (hasAlternativeExecutables) {
        return SizedBox(
          width: alternativeSlotWidth,
          child: Align(
            alignment: Alignment.center,
            child: buildAlternativeLaunchButton(alternativeExecutables),
          ),
        );
      }
      if (_alternativeFlyoutController.isAttached &&
          _alternativeFlyoutController.isOpen) {
        _alternativeFlyoutController.close();
      }
      return const SizedBox(width: alternativeSlotWidth);
    }

    return ListTile.selectable(
      onPressed: widget.onLaunch,
      selected: _isContextMenuActive,
      leading: Padding(
        padding: const EdgeInsets.symmetric(vertical: 8.0),
        child: icon,
      ),
      title: Text(title, style: TextStyle(color: titleColor)),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          subtitle,
          Text(
            widget.software.installPath,
            style: FluentTheme.of(context).typography.caption,
          ),
        ],
      ),
      trailing: widget.isReorderMode
          ? const Icon(FluentIcons.drag_object)
          : Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (widget.software.status ==
                    SoftwareStatus.unknownInstall) ...[
                  if (widget.software.archiveExists) ...[
                    _buildArchiveStatusPill(exists: true, tooltip: l10n.tileArchiveExists),
                    const SizedBox(width: controlSpacing),
                  ],
                  if (widget.onRehost != null)
                    Tooltip(
                      message: l10n.homeRehost,
                      child: IconButton(
                        icon: const Icon(FluentIcons.refresh),
                        style: ButtonStyle(
                          foregroundColor: WidgetStateProperty.all(
                            Colors.green,
                          ),
                        ),
                        onPressed: widget.onRehost,
                      ),
                    ),
                  if (widget.onRehost != null)
                    const SizedBox(width: controlSpacing),
                ],
                if (widget.software.status == SoftwareStatus.managed) ...[
                  _buildArchiveAndBackupBadges(widget.software),
                  const SizedBox(width: controlSpacing),
                  // 备选程序入口（固定宽度插槽）
                  buildAlternativeSlot(),
                  const SizedBox(width: controlSpacing),
                  // 仅在"目录已删除且有归档文件"时，用"重新托管"替代"更改主程序"
                  if (!widget.software.installExists &&
                      widget.software.archiveExists &&
                      widget.onRehost != null)
                    buildActionControl(
                      icon: FluentIcons.refresh,
                      tooltip: l10n.homeRehost,
                      onPressed: widget.onRehost,
                      color: Colors.green,
                    )
                  else
                    buildActionControl(
                      icon: FluentIcons.edit,
                      tooltip: changeExecutableTooltip,
                      onPressed: canChangeExecutable
                          ? widget.onChangeExecutable
                          : null,
                    ),
                  const SizedBox(width: controlSpacing),
                ],
                if (widget.software.status ==
                    SoftwareStatus.unknownArchive) ...[
                  _buildArchiveStatusPill(
                    exists: true,
                    tooltip: widget.software.isBackupArchive ? l10n.homeBackupArchive : l10n.homeArchiveFile,
                  ),
                  const SizedBox(width: controlSpacing),
                ],
                buildActionControl(
                  icon: FluentIcons.delete,
                  tooltip: l10n.delete,
                  onPressed: widget.onDelete,
                  isDestructive: true,
                ),
              ],
            ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final bool useGrid = widget.displayStyle == SoftwareTileDisplay.grid;
    // 与列表视图保持一致的图标尺寸，避免缩放导致的模糊
    const double iconSize = 32;
    final Widget icon = _buildIcon(iconSize);

    String listTitle;
    String subtitleText;
    Color? titleColor;
    String? gridSupplementaryText;

    // 使用统一的显示名称逻辑
    listTitle = _getSoftwareDisplayName();

    switch (widget.software.status) {
      case SoftwareStatus.managed:
        if (!widget.software.installExists) {
          subtitleText = l10n.homeSoftwareDirDeleted;
          titleColor = Colors.orange;
          gridSupplementaryText = l10n.homeSoftwareDirDeleted;
        } else {
          subtitleText = p.basename(widget.software.executablePath);
          gridSupplementaryText = null;
        }
        break;
      case SoftwareStatus.unknownInstall:
        subtitleText = l10n.homeUnknownFolder;
        gridSupplementaryText = subtitleText;
        titleColor = Colors.orange;
        break;
      case SoftwareStatus.unknownArchive:
        subtitleText = l10n.homeUnknownArchiveFile;
        gridSupplementaryText = subtitleText;
        titleColor = Colors.orange;
        break;
    }

    Widget content;

    if (useGrid) {
      Widget? supplementary;
      if (widget.software.status == SoftwareStatus.unknownInstall) {
        supplementary = Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.homeUnknownFolder),
            if (widget.software.archiveExists) ...[
              const SizedBox(height: 6),
              Tooltip(
                message: l10n.tileArchiveExists,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.green.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Icon(FluentIcons.archive, color: Colors.green),
                ),
              ),
            ],
            if (widget.onRehost != null) ...[
              const SizedBox(height: 8),
              Tooltip(
                message: l10n.homeRehost,
                child: IconButton(
                  icon: const Icon(FluentIcons.refresh),
                  style: ButtonStyle(
                    foregroundColor: WidgetStateProperty.all(Colors.green),
                  ),
                  onPressed: widget.onRehost,
                ),
              ),
            ],
          ],
        );
      } else if (gridSupplementaryText != null) {
        supplementary = Text(
          gridSupplementaryText,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        );
      }

      content = _buildGridTile(
        context: context,
        icon: icon,
        // 网格视图标题与列表保持一致，优先显示可执行文件描述
        title: listTitle,
        titleColor: titleColor,
        supplementary: supplementary,
      );
    } else {
      final List<String> effectiveExecutables = _isExecutableLoading
          ? const []
          : _availableExecutables;
      final List<String> alternativeExecutables = effectiveExecutables
          .where(
            (path) => widget.software.executablePath.isEmpty
                ? true
                : !p.equals(path, widget.software.executablePath),
          )
          .toList();
      final bool hasAlternativeExecutables = alternativeExecutables.isNotEmpty;
      final bool canChangeExecutable =
          !_isExecutableLoading && hasAlternativeExecutables;
      final String changeExecutableTooltip;
      if (_isExecutableLoading) {
        changeExecutableTooltip = l10n.homeScanning;
      } else if (effectiveExecutables.isEmpty) {
        changeExecutableTooltip = l10n.homeNoExecutableFound;
      } else if (canChangeExecutable) {
        changeExecutableTooltip = l10n.homeChangeExecutable;
      } else {
        changeExecutableTooltip = l10n.homeOnlyOneExecutableFound;
      }

      content = _buildListTile(
        context: context,
        icon: icon,
        title: listTitle,
        subtitle: Text(subtitleText),
        titleColor: titleColor,
        alternativeExecutables: alternativeExecutables,
        hasAlternativeExecutables: hasAlternativeExecutables,
        canChangeExecutable: canChangeExecutable,
        changeExecutableTooltip: changeExecutableTooltip,
      );
    }

    return _wrapWithContextMenu(content);
  }
}
