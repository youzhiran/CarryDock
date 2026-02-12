import 'dart:io';

import 'package:carrydock/l10n/app_localizations.dart';
import 'package:carrydock/models/software.dart';
import 'package:carrydock/services/software_service.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:carrydock/widgets/select_executable_dialog.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:path/path.dart' as p;

/// 归档管理页面：集中管理归档与备份文件。
class ArchiveManagerScreen extends StatefulWidget {
  const ArchiveManagerScreen({super.key});

  @override
  State<ArchiveManagerScreen> createState() => _ArchiveManagerScreenState();
}

class _ArchiveManagerScreenState extends State<ArchiveManagerScreen> {
  final SoftwareService _softwareService = SoftwareService();
  bool _isLoading = true;
  List<Software> _archives = [];
  List<Software> _backups = [];
  Set<String> _linkedPaths = <String>{};

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _isLoading = true);
    try {
      final items = await _softwareService.listArchivesForManager(includeBackups: true);
      final managed = await _softwareService.getSoftwareList();
      final linked = <String>{
        for (final s in managed)
          if (s.archivePath.isNotEmpty) s.archivePath,
      }..addAll([
          for (final s in managed)
            if (s.backupPath.isNotEmpty) s.backupPath,
        ]);
      final archives = <Software>[];
      final backups = <Software>[];
      for (final s in items) {
        if (s.isBackupArchive) {
          backups.add(s);
        } else {
          archives.add(s);
        }
      }
      archives.sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));
      backups.sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));
      if (!mounted) return;
      setState(() {
        _archives = archives;
        _backups = backups;
        _linkedPaths = linked;
      });
    } finally {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  Future<void> _showMessage(String title, String content) async {
    if (!mounted) return;
    final l10n = AppLocalizations.of(context)!;
    await showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          Button(child: Text(l10n.ok), onPressed: () => Navigator.of(context).pop()),
        ],
      ),
    );
  }

  Future<void> _openDirectory(String path) async {
    try {
      await Process.start('explorer.exe', [path], runInShell: true);
    } catch (e, s) {
      logger.e('打开目录失败', error: e, stackTrace: s);
      if (!mounted) return;
      final l10n = AppLocalizations.of(context)!;
      await _showMessage(l10n.archiveOperationFailed, l10n.archiveCannotOpenDirectory);
    }
  }

  Future<void> _revealInExplorer(String filePath) async {
    try {
      final file = File(filePath);
      if (await file.exists()) {
        await Process.start('explorer.exe', ['/select,', filePath], runInShell: true);
      } else {
        await _openDirectory(p.dirname(filePath));
      }
    } catch (e, s) {
      logger.e('打开资源管理器失败', error: e, stackTrace: s);
      if (!mounted) return;
      final l10n = AppLocalizations.of(context)!;
      await _showMessage(l10n.archiveOperationFailed, l10n.archiveCannotOpenExplorer);
    }
  }

  void _showProgress(String message) {
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (context) => ContentDialog(
        content: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const ProgressRing(),
            const SizedBox(width: 12),
            Text(message),
          ],
        ),
      ),
    );
  }

  void _popDialog() {
    if (!mounted) return;
    if (Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
    }
  }

  Future<void> _handleRehostFromArchive(Software item) async {
    final l10n = AppLocalizations.of(context)!;
    // 对于"未知归档"，直接走 addSoftwareFromFile 流程（会复用现有的可执行选择与重复处理）
    _showProgress(l10n.archiveInstallingFromArchive);
    bool shouldReload = false;
    try {
      final result = await _softwareService.addSoftwareFromFile(item.archivePath);
      switch (result.type) {
        case AddSoftwareResultType.success:
          shouldReload = true;
          break;
        case AddSoftwareResultType.needsSelection:
          _popDialog();
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
            _showProgress(l10n.archiveCompletingInstallation);
            await _softwareService.completeSoftwareAddition(
              installPath: pending.installPath,
              archivePath: pending.archivePath,
              selectedExecutablePath: selected,
              preferredSortOrder: pending.preferredSortOrder,
            );
            shouldReload = true;
          } else {
            _showProgress(l10n.archiveCancellingOperation);
            await _softwareService.cleanupTemporaryFiles(
              installPath: pending.installPath,
              archivePath: pending.archivePath,
            );
          }
          break;
        case AddSoftwareResultType.duplicate:
          _popDialog();
          if (!mounted) break;
          final info = result.duplicateInfo;
          if (info == null) break;
          final doBackupRestore = await showDialog<bool>(
            context: context,
            builder: (context) => ContentDialog(
              title: Text(l10n.archiveDuplicateDetected),
              content: Text(
                '${l10n.archiveDuplicateMessage}'
                '${info.targetInstallPath}\n'
                '${p.basename(info.targetArchivePath)}',
              ),
              actions: [
                Button(child: Text(l10n.cancel), onPressed: () => Navigator.of(context).pop(false)),
                FilledButton(child: Text(l10n.archiveBackupAndRestore), onPressed: () => Navigator.of(context).pop(true)),
              ],
            ),
          );
          if (doBackupRestore == true) {
            try {
              if (info.installDirExists) {
                final dir = Directory(info.targetInstallPath);
                if (await dir.exists()) {
                  await _softwareService.createBackupFromDirectory(
                    sourceDir: dir,
                    displayName: p.basename(info.targetInstallPath),
                  );
                }
              }
              _showProgress(l10n.archiveRestoring);
              final r = await _softwareService.resolveDuplicateAddition(
                info: info,
                overwriteExisting: true,
              );
              if (r.type == AddSoftwareResultType.success) {
                shouldReload = true;
              }
            } catch (e, s) {
              logger.e('备份并还原失败', error: e, stackTrace: s);
              await _showMessage(l10n.archiveOperationFailed, l10n.archiveBackupRestoreFailed);
            } finally {
              _popDialog();
            }
          }
          break;
        case AddSoftwareResultType.cancelled:
        case AddSoftwareResultType.error:
          break;
      }
    } catch (e, s) {
      logger.e('从归档安装失败', error: e, stackTrace: s);
      await _showMessage(l10n.archiveOperationFailed, l10n.archiveInstallFailed);
    } finally {
      _popDialog();
      if (shouldReload) await _loadData();
    }
  }

  Future<void> _handleDeleteArchive(Software item, {required bool isBackup}) async {
    final l10n = AppLocalizations.of(context)!;
    final confirm = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => ContentDialog(
          title: Text(isBackup ? l10n.archiveDeleteBackup : l10n.archiveDeleteArchive),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (isBackup)
                Text(l10n.archiveDeleteBackupHint)
              else
                Text(l10n.archiveDeleteArchiveHint),
              const SizedBox(height: 8),
              SelectableText(item.name),
            ],
          ),
          actions: [
            Button(child: Text(l10n.cancel), onPressed: () => Navigator.of(context).pop(false)),
            FilledButton(child: Text(l10n.delete), onPressed: () => Navigator.of(context).pop(true)),
          ],
        ),
      ),
    );
    if (confirm != true) return;
    try {
      _showProgress(isBackup ? l10n.archiveDeletingBackup : l10n.archiveDeletingArchive);
      final file = File(item.archivePath);
      if (await file.exists()) {
        await file.delete();
      }
      // 备份删除后，静默清除所有引用该备份的关联，避免遗留无效路径
      if (isBackup) {
        await _softwareService.clearBackupAssociationsForPath(item.archivePath);
      }
      _popDialog();
    } catch (e, s) {
      logger.e('删除文件失败', error: e, stackTrace: s);
      _popDialog();
      await _showMessage(l10n.archiveDeleteFailed, l10n.archiveCannotDeleteFile);
    }
    await _loadData();
  }

  Future<void> _handleManualLink(Software item, {required bool isBackup}) async {
    // 选择一个已托管软件
    final managed = (await _softwareService.getSoftwareList())
        .where((s) => s.status == SoftwareStatus.managed)
        .toList();
    if (!mounted) return;
    final l10n = AppLocalizations.of(context)!;
    if (managed.isEmpty) {
      await _showMessage(l10n.archiveNoAvailableSoftware, l10n.archiveNoAvailableSoftwareHint);
      return;
    }
    // 默认选中：若该文件已被某个软件关联（归档或备份），则默认选中该软件；否则选中首项
    String? selectedId;
    try {
      final matchedIndex = managed.indexWhere(
        (s) => p.equals(s.archivePath, item.archivePath) ||
            p.equals(s.backupPath, item.archivePath),
      );
      selectedId = matchedIndex >= 0 ? managed[matchedIndex].id : managed.first.id;
    } catch (_) {
      selectedId = managed.first.id;
    }
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => ContentDialog(
          title: Text(isBackup ? l10n.archiveLinkBackup : l10n.archiveLinkArchive),
          content: ComboBox<String>(
            isExpanded: true,
            items: [
              for (final s in managed)
                ComboBoxItem<String>(
                  value: s.id,
                  child: Tooltip(
                    message: s.name,
                    child: Text(
                      s.name,
                      overflow: TextOverflow.ellipsis,
                      maxLines: 1,
                      softWrap: false,
                    ),
                  ),
                ),
            ],
            value: selectedId,
            onChanged: (v) => setState(() => selectedId = v),
          ),
          actions: [
            Button(child: Text(l10n.cancel), onPressed: () => Navigator.of(context).pop(false)),
            FilledButton(child: Text(l10n.archiveLink), onPressed: () => Navigator.of(context).pop(true)),
          ],
        ),
      ),
    );
    if (ok == true && selectedId != null) {
      await _softwareService.linkArchiveToSoftware(
        softwareId: selectedId!,
        archivePath: item.archivePath,
      );
      await _showMessage(l10n.archiveLinkSuccess, l10n.archiveLinkSuccessHint);
    }
  }

  Future<void> _handleCreateBackup() async {
    final l10n = AppLocalizations.of(context)!;
    final managed = (await _softwareService.getSoftwareList())
        .where((s) => s.status == SoftwareStatus.managed)
        .toList();
    if (!mounted) return;
    if (managed.isEmpty) {
      await _showMessage(l10n.archiveNoAvailableSoftware, l10n.archiveNoAvailableSoftwareHint);
      return;
    }
    String? selectedId = managed.first.id;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => ContentDialog(
          title: Text(l10n.archiveCreateBackup),
          content: ComboBox<String>(
            isExpanded: true,
            items: [
              for (final s in managed)
                ComboBoxItem<String>(
                  value: s.id,
                  child: Tooltip(
                    message: s.name,
                    child: Text(
                      s.name,
                      overflow: TextOverflow.ellipsis,
                      maxLines: 1,
                      softWrap: false,
                    ),
                  ),
                ),
            ],
            value: selectedId,
            onChanged: (v) => setState(() => selectedId = v),
          ),
          actions: [
            Button(child: Text(l10n.cancel), onPressed: () => Navigator.of(context).pop(false)),
            FilledButton(child: Text(l10n.add), onPressed: () => Navigator.of(context).pop(true)),
          ],
        ),
      ),
    );
    if (ok == true && selectedId != null) {
      final s = managed.firstWhere((e) => e.id == selectedId);
      try {
        _showProgress(l10n.archiveCreatingBackup);
        final path = await _softwareService.createBackupForSoftware(s);
        _popDialog();
        await _showMessage(l10n.archiveCreateSuccess, '${l10n.archiveBackupCreated}${p.basename(path)}');
        await _loadData();
      } catch (e, s) {
        logger.e('创建备份失败', error: e, stackTrace: s);
        _popDialog();
        await _showMessage(l10n.archiveCreateFailed, l10n.archiveCreateBackupFailed);
      }
    }
  }

  Widget _buildFileTile(Software item, {required bool isBackup}) {
    final theme = FluentTheme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final isLinked = _linkedPaths.contains(item.archivePath);
    return ListTile(
      onPressed: () => _revealInExplorer(item.archivePath),
      title: Row(
        children: [
          Expanded(
            child: Text(item.name, overflow: TextOverflow.ellipsis, maxLines: 1),
          ),
          const SizedBox(width: 8),
        ],
      ),
      subtitle: Text(item.archivePath, style: theme.typography.caption),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Tooltip(
            message: l10n.archiveRestore,
            child: IconButton(
              icon: const Icon(FluentIcons.refresh),
              onPressed: () => _handleRehostFromArchive(item),
            ),
          ),
          const SizedBox(width: 6),
          if (!isBackup)
            Tooltip(
              message: l10n.archiveManualLink,
              child: IconButton(
                style: isLinked
                    ? ButtonStyle(
                        foregroundColor:
                            WidgetStateProperty.all(Colors.green),
                      )
                    : null,
                icon: const Icon(FluentIcons.link),
                onPressed: () => _handleManualLink(item, isBackup: isBackup),
              ),
            ),
          const SizedBox(width: 6),
          Tooltip(
            message: l10n.delete,
            child: IconButton(
              icon: const Icon(FluentIcons.delete),
              onPressed: () => _handleDeleteArchive(item, isBackup: isBackup),
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return NavigationView(
      content: ScaffoldPage.scrollable(
        header: PageHeader(
          title: Text(l10n.archiveTitle, style: theme.typography.title),
          commandBar: CommandBar(
            mainAxisAlignment: MainAxisAlignment.end,
            primaryItems: [
              CommandBarButton(
                icon: const Icon(FluentIcons.add),
                label: Text(l10n.archiveCreateBackup),
                onPressed: _handleCreateBackup,
              ),
              CommandBarButton(
                icon: const Icon(FluentIcons.refresh),
                label: Text(l10n.archiveRefresh),
                onPressed: _loadData,
              ),
            ],
            secondaryItems: [
              CommandBarButton(
                icon: const Icon(FluentIcons.folder_open),
                label: Text(l10n.archiveOpenArchiveDir),
                onPressed: () async {
                  final root = await _softwareService.resolveArchiveDirectoryPath();
                  await _openDirectory(root);
                },
              ),
              CommandBarButton(
                icon: const Icon(FluentIcons.open_folder_horizontal),
                label: Text(l10n.archiveOpenBackupDir),
                onPressed: () async {
                  final root = await _softwareService.resolveArchiveDirectoryPath();
                  await _openDirectory(p.join(root, 'backup'));
                },
              ),
            ],
          ),
        ),
        children: [
          if (_isLoading) const Center(child: ProgressRing()),
          if (!_isLoading) ...[
            Text(l10n.archiveFiles, style: theme.typography.subtitle),
            const SizedBox(height: 8),
            if (_archives.isEmpty)
              InfoLabel(label: l10n.archiveNoArchiveFiles, child: SizedBox.shrink())
            else
              ListView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: _archives.length,
                itemBuilder: (context, index) => _buildFileTile(
                  _archives[index],
                  isBackup: false,
                ),
              ),
            const SizedBox(height: 16),
            Text(l10n.archiveBackupFiles, style: theme.typography.subtitle),
            const SizedBox(height: 8),
            if (_backups.isEmpty)
              InfoLabel(label: l10n.archiveNoBackupFiles, child: SizedBox.shrink())
            else
              ListView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: _backups.length,
                itemBuilder: (context, index) => _buildFileTile(
                  _backups[index],
                  isBackup: true,
                ),
              ),
          ],
        ],
      ),
    );
  }
}
