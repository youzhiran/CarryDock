import 'dart:io';

import 'package:carrydock/providers/update_provider.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:fluent_ui/fluent_ui.dart';

/// 更新对话框，用于显示更新信息和执行更新操作
class UpdateDialog extends StatefulWidget {
  final UpdateProvider updateProvider;
  
  const UpdateDialog({required this.updateProvider});
  
  @override
  State<UpdateDialog> createState() => _UpdateDialogState();
}

class _UpdateDialogState extends State<UpdateDialog> {
  final SettingsService _settingsService = SettingsService();
  
  @override
  Widget build(BuildContext context) {
    final latestVersionInfo = widget.updateProvider.latestVersionInfo;
    if (latestVersionInfo == null) {
      return ContentDialog(
        title: const Text('更新信息'),
        content: const Text('未获取到更新信息'),
        actions: [
          Button(
            child: const Text('关闭'),
            onPressed: () => Navigator.of(context, rootNavigator: true).pop(),
          ),
        ],
      );
    }
    
    final version = latestVersionInfo['version'] as String;
    final changelog = latestVersionInfo['changelog'] as String;
    
    return ContentDialog(
      title: Text('发现新版本：$version'),
      content: SizedBox(
        width: 480,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('更新内容：'),
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
            ],
          ],
        ),
      ),
      actions: [
        Button(
          child: const Text('取消'),
          onPressed: () => Navigator.of(context, rootNavigator: true).pop(),
        ),
        if (!widget.updateProvider.isDownloading && !widget.updateProvider.isInstalling) ...[
          FilledButton(
            child: const Text('下载并更新'),
            onPressed: () async {
              // 获取配置文件路径
              final configFilePath = await _settingsService.getConfigFilePath();
              // 获取应用目录
              final appDirectory = configFilePath.substring(0, configFilePath.indexOf('data'));
              
              // 生成临时下载路径
              final tempDir = Directory.systemTemp;
              final assetName = latestVersionInfo['assetName'] as String;
              logger.i('下载更新包：$assetName');
              final tempFilePath = '${tempDir.path}\$assetName';
              
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
                      title: const Text('更新成功'),
                      content: const Text('应用已成功更新，将在下次启动时生效。'),
                      actions: [
                        Button(
                          child: const Text('关闭'),
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
    );
  }
}