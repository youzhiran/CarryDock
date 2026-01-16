import 'dart:io';

import 'package:carrydock/l10n/app_localizations.dart';
import 'package:carrydock/providers/update_provider.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:fluent_ui/fluent_ui.dart';

/// 更新对话框，用于显示更新信息和执行更新操作
class UpdateDialog extends StatefulWidget {
  final UpdateProvider updateProvider;

  const UpdateDialog({super.key, required this.updateProvider});

  @override
  State<UpdateDialog> createState() => _UpdateDialogState();
}

class _UpdateDialogState extends State<UpdateDialog> {
  final SettingsService _settingsService = SettingsService();

  // 保存监听器函数引用，以便在dispose时正确移除
  void _updateListener() {
    if (mounted) {
      setState(() {});
    }
  }

  @override
  void initState() {
    super.initState();
    // 添加监听器，确保对话框能响应updateProvider的状态变化
    widget.updateProvider.addListener(_updateListener);
  }

  @override
  void dispose() {
    // 移除监听器
    widget.updateProvider.removeListener(_updateListener);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final latestVersionInfo = widget.updateProvider.latestVersionInfo;
    if (latestVersionInfo == null) {
      return ContentDialog(
        title: Text(l10n.dialogUpdateInfo),
        content: Text(l10n.dialogNoUpdateInfo),
        actions: [
          Button(
            child: Text(l10n.dialogClose),
            onPressed: () => Navigator.of(context, rootNavigator: true).pop(),
          ),
        ],
      );
    }

    final version = latestVersionInfo['version'] as String;
    final changelog = latestVersionInfo['changelog'] as String;

    return ContentDialog(
      title: Text(l10n.dialogNewVersion(version)),
      content: SizedBox(
        width: 480,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.dialogUpdateContent),
            const SizedBox(height: 8),
            Container(
              height: 180,
              width: double.infinity,
              decoration: BoxDecoration(
                color: FluentTheme.of(context).resources.controlFillColorSecondary,
                borderRadius: BorderRadius.circular(4),
              ),
              padding: const EdgeInsets.all(12),
              child: SingleChildScrollView(
                child: SelectableText(changelog),
              ),
            ),
            const SizedBox(height: 12),
            if (widget.updateProvider.isDownloading || widget.updateProvider.isInstalling) ...[
              Text(widget.updateProvider.updateStatus),
              const SizedBox(height: 8),
              ProgressBar(value: widget.updateProvider.downloadProgress),
              const SizedBox(height: 8),
              Align(
                alignment: Alignment.centerRight,
                child: Text(
                  '${(widget.updateProvider.downloadProgress * 100).toStringAsFixed(2)}%',
                  style: FluentTheme.of(context).typography.caption,
                ),
              ),
            ],
          ],
        ),
      ),
      actions: [
        Button(
          child: Text(l10n.dialogCancel),
          onPressed: () {
            if (widget.updateProvider.isDownloading) {
              // 取消下载
              widget.updateProvider.cancelDownload();
            }
            Navigator.of(context, rootNavigator: true).pop();
          },
        ),
        if (!widget.updateProvider.isDownloading && !widget.updateProvider.isInstalling) ...[
          if (widget.updateProvider.updateStatus.contains('失败')) ...[
            Button(
              child: Text(l10n.dialogRetry),
              onPressed: () async {
                // 获取配置文件路径
                final configFilePath = await _settingsService.getConfigFilePath();
                // 获取应用目录
                final appDirectory = configFilePath.substring(0, configFilePath.indexOf('data'));

                // 生成临时下载路径
                final tempDir = Directory.systemTemp;
                final assetName = latestVersionInfo['assetName'] as String;
                logger.i('重新下载更新包：$assetName');
                final tempFilePath = '${tempDir.path}\\$assetName';

                // 下载更新
                final updateFile = await widget.updateProvider.downloadUpdate(tempFilePath);
                if (updateFile != null) {
                  // 安装更新
                  final success = await widget.updateProvider.installUpdate(
                    updateFile,
                    appDirectory,
                    configFilePath,
                  );
                  if (success) {
                    // 显示更新成功提示
                    await showDialog(
                      context: context,
                      builder: (dialogContext) => ContentDialog(
                        title: Text(l10n.dialogUpdateSuccess),
                        content: Text(l10n.dialogUpdateSuccessMessage),
                        actions: [
                          Button(
                            child: Text(l10n.dialogClose),
                            onPressed: () {
                              Navigator.of(dialogContext).pop();
                              Navigator.of(context, rootNavigator: true).pop();
                            },
                          ),
                        ],
                      ),
                    );
                  }
                }
              },
            ),
          ] else ...[
            FilledButton(
              child: Text(l10n.dialogDownloadAndUpdate),
              onPressed: () async {
                // 获取配置文件路径
                final configFilePath = await _settingsService.getConfigFilePath();
                // 获取应用目录
                final appDirectory = configFilePath.substring(0, configFilePath.indexOf('data'));

                // 生成临时下载路径
                final tempDir = Directory.systemTemp;
                final assetName = latestVersionInfo['assetName'] as String;
                logger.i('下载更新包：$assetName');
                final tempFilePath = '${tempDir.path}\\$assetName';

                // 下载更新
                final updateFile = await widget.updateProvider.downloadUpdate(tempFilePath);
                if (updateFile != null) {
                  // 安装更新
                  final success = await widget.updateProvider.installUpdate(
                    updateFile,
                    appDirectory,
                    configFilePath,
                  );
                  if (success) {
                    // 显示更新成功提示
                    await showDialog(
                      context: context,
                      builder: (dialogContext) => ContentDialog(
                        title: Text(l10n.dialogUpdateSuccess),
                        content: Text(l10n.dialogUpdateSuccessMessage),
                        actions: [
                          Button(
                            child: Text(l10n.dialogClose),
                            onPressed: () {
                              Navigator.of(dialogContext).pop();
                              Navigator.of(context, rootNavigator: true).pop();
                            },
                          ),
                        ],
                      ),
                    );
                  }
                }
              },
            ),
          ],
        ],
      ],
    );
  }
}