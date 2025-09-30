import 'dart:io';

import 'package:carrydock/models/software.dart';
import 'package:carrydock/services/executable_info_service.dart';
import 'package:carrydock/services/software_service.dart';
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
        child: Icon(
          FluentIcons.archive,
          size: 14,
          color: accent,
        ),
      ),
    );
    if (tooltip != null && tooltip.isNotEmpty) {
      return Tooltip(message: tooltip, child: pill);
    }
    return pill;
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
    final installPath = widget.software.installPath;
    if (installPath.isEmpty) {
      _showInfoBar('提示', '该软件未配置安装目录。');
      return;
    }
    final directory = Directory(installPath);
    if (!await directory.exists()) {
      _showInfoBar('提示', '找不到安装目录：$installPath');
      return;
    }
    try {
      await Process.start('explorer.exe', [installPath], runInShell: true);
    } catch (e, s) {
      logger.e('打开安装目录失败', error: e, stackTrace: s);
      _showInfoBar('错误', '无法打开安装目录，请稍后重试。', severity: InfoBarSeverity.error);
    }
  }

  /// 打开归档文件所在目录，优先高亮具体文件。
  Future<void> _openArchiveDirectory() async {
    final archivePath = widget.software.archivePath;
    if (archivePath.isEmpty) {
      _showInfoBar('提示', '该软件未配置归档文件。');
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
        _showInfoBar('提示', '归档文件不存在，已打开归档目录。');
        return;
      }
      if (await parentDirectory.exists()) {
        await Process.start('explorer.exe', [
          parentDirectory.path,
        ], runInShell: true);
        _showInfoBar('提示', '归档文件不存在，已打开归档所在目录。');
        return;
      }
      _showInfoBar('提示', '找不到归档文件所在位置。');
    } catch (e, s) {
      logger.e('打开归档目录失败', error: e, stackTrace: s);
      _showInfoBar('错误', '无法打开归档目录，请稍后重试。', severity: InfoBarSeverity.error);
    }
  }

  /// 弹出右键菜单，提供常用目录的快捷入口。
  void _showContextMenu({Offset? position}) {
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
                text: const Text('打开软件文件夹'),
                onPressed: widget.software.installPath.isNotEmpty
                    ? _openInstallDirectory
                    : null,
              ),
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.archive),
                text: const Text('创建备份'),
                onPressed: widget.software.installPath.isNotEmpty
                    ? () async {
                        try {
                          final dir = Directory(widget.software.installPath);
                          if (!await dir.exists()) {
                            _showInfoBar('提示', '安装目录不存在，无法创建备份。');
                            return;
                          }
                          final created = await _softwareService
                              .createBackupForSoftware(widget.software);
                          _showInfoBar('成功', '已创建备份：${p.basename(created)}',
                              severity: InfoBarSeverity.success);
                        } catch (e, s) {
                          logger.e('创建备份失败', error: e, stackTrace: s);
                          _showInfoBar('错误', '创建备份失败，请稍后重试。',
                              severity: InfoBarSeverity.error);
                        }
                      }
                    : null,
              ),
              MenuFlyoutItem(
                leading: const Icon(FluentIcons.open_file),
                text: const Text('打开归档文件夹'),
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
      return FlyoutTarget(
        controller: _alternativeFlyoutController,
        child: Tooltip(
          message: '启动其他程序',
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
                if (widget.software.status == SoftwareStatus.unknownInstall) ...[
                  if (widget.software.archiveExists) ...[
                    _buildArchiveStatusPill(
                      exists: true,
                      tooltip: '归档文件存在',
                    ),
                    const SizedBox(width: controlSpacing),
                  ],
                  if (widget.onRehost != null)
                    Tooltip(
                      message: '重新托管',
                      child: IconButton(
                        icon: const Icon(FluentIcons.refresh),
                        style: ButtonStyle(
                          foregroundColor: WidgetStateProperty.all(Colors.green),
                        ),
                        onPressed: widget.onRehost,
                      ),
                    ),
                  if (widget.onRehost != null) const SizedBox(width: controlSpacing),
                ],
                if (widget.software.status == SoftwareStatus.managed) ...[
                  _buildArchiveStatusPill(
                    exists: widget.software.archiveExists,
                    tooltip: widget.software.archiveExists ? '归档文件存在' : '归档文件不存在',
                  ),
                  const SizedBox(width: controlSpacing),
                  // 备选程序入口（固定宽度插槽）
                  buildAlternativeSlot(),
                  const SizedBox(width: controlSpacing),
                  // 仅在“目录已删除且有归档文件”时，用“重新托管”替代“更改主程序”
                  if (!widget.software.installExists &&
                      widget.software.archiveExists &&
                      widget.onRehost != null)
                    buildActionControl(
                      icon: FluentIcons.refresh,
                      tooltip: '重新托管',
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
                if (widget.software.status == SoftwareStatus.unknownArchive) ...[
                  _buildArchiveStatusPill(
                    exists: true,
                    tooltip: widget.software.isBackupArchive ? '备份归档' : '归档文件',
                  ),
                  const SizedBox(width: controlSpacing),
                ],
                buildActionControl(
                  icon: FluentIcons.delete,
                  tooltip: '删除',
                  onPressed: widget.onDelete,
                  isDestructive: true,
                ),
              ],
            ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final bool useGrid = widget.displayStyle == SoftwareTileDisplay.grid;
    // 与列表视图保持一致的图标尺寸，避免缩放导致的模糊
    const double iconSize = 32;
    final Widget icon = _buildIcon(iconSize);

    String listTitle;
    String subtitleText;
    Color? titleColor;
    String? gridSupplementaryText;

    switch (widget.software.status) {
      case SoftwareStatus.managed:
        listTitle = _fileDescription ?? widget.software.name;
        if (!widget.software.installExists) {
          subtitleText = '软件目录已删除';
          titleColor = Colors.orange;
          gridSupplementaryText = '软件目录已删除';
        } else {
          subtitleText = p.basename(widget.software.executablePath);
          gridSupplementaryText = null;
        }
        break;
      case SoftwareStatus.unknownInstall:
        listTitle = widget.software.name;
        subtitleText = '未知文件夹';
        gridSupplementaryText = subtitleText;
        titleColor = Colors.orange;
        break;
      case SoftwareStatus.unknownArchive:
        listTitle = widget.software.name;
        subtitleText = '未知归档文件';
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
            const Text('未知文件夹'),
            if (widget.software.archiveExists) ...[
              const SizedBox(height: 6),
              Tooltip(
                message: '归档文件存在',
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.green.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Icon(
                    FluentIcons.archive,
                    color: Colors.green,
                  ),
                ),
              ),
            ],
            if (widget.onRehost != null) ...[
              const SizedBox(height: 8),
              Tooltip(
                message: '重新托管',
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
        changeExecutableTooltip = '正在扫描可执行程序...';
      } else if (effectiveExecutables.isEmpty) {
        changeExecutableTooltip = '未在安装目录中找到可执行程序';
      } else if (canChangeExecutable) {
        changeExecutableTooltip = '更改主程序';
      } else {
        changeExecutableTooltip = '当前仅发现一个可执行程序，无法更改主程序';
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
