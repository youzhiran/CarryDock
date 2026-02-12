import 'package:carrydock/l10n/app_localizations.dart';
import 'package:carrydock/providers/update_provider.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/widgets/update_dialog.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

/// 设置屏幕各个部分的构建方法

/// 构建关于部分
Widget buildAboutSection(
  BuildContext context,
  bool developerOptionsEnabled,
  String appVersion,
  String buildTime,
  Function() onVersionTap,
) {
  final typography = FluentTheme.of(context).typography;
  final resources = FluentTheme.of(context).resources;
  final updateProvider = context.watch<UpdateProvider>();
  final settingsService = SettingsService();
  final l10n = AppLocalizations.of(context)!;

  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      if (updateProvider.isReinstalling ||
          updateProvider.isCheckingUpdate ||
          updateProvider.isDownloading ||
          updateProvider.isInstalling) ...[
        const SizedBox(height: 8),
        Text(updateProvider.updateStatus, style: typography.caption),
        const SizedBox(height: 8),
        ProgressBar(value: updateProvider.downloadProgress),
      ],
      const SizedBox(height: 12),
      Button(
        onPressed: onVersionTap,
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll(
            EdgeInsets.symmetric(horizontal: 16, vertical: 18),
          ),
          backgroundColor: WidgetStatePropertyAll(
            resources.controlFillColorSecondary,
          ),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(l10n.settingsVersion),
            Text(appVersion, style: typography.bodyStrong),
          ],
        ),
      ),
      const SizedBox(height: 12),
      Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
        decoration: BoxDecoration(
          color: resources.controlFillColorSecondary,
          borderRadius: BorderRadius.circular(4),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(l10n.settingsBuildTime),
            Text(buildTime, style: typography.bodyStrong),
          ],
        ),
      ),
      const SizedBox(height: 12),
      Button(
        onPressed: () async {
          await updateProvider.checkForUpdates();
          if (!context.mounted) return;
          if (updateProvider.hasUpdate &&
              updateProvider.latestVersionInfo != null) {
            await showDialog(
              context: context,
              builder: (dialogContext) =>
                  UpdateDialog(updateProvider: updateProvider),
            );
          } else {
            displayInfoBar(
              context,
              builder: (context, close) {
                return InfoBar(
                  title: Text(l10n.settingsSuccess),
                  content: Text(updateProvider.updateStatus),
                  action: IconButton(
                    icon: const Icon(FluentIcons.clear),
                    onPressed: close,
                  ),
                  severity: InfoBarSeverity.success,
                );
              },
            );
          }
        },
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll(
            EdgeInsets.symmetric(horizontal: 16, vertical: 18),
          ),
          backgroundColor: WidgetStatePropertyAll(
            resources.controlFillColorSecondary,
          ),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [Text(l10n.settingsCheckUpdate), const Icon(FluentIcons.refresh)],
        ),
      ),
      const SizedBox(height: 12),
      Button(
        onPressed: () async {
          // 确认重装操作
          final result = await showDialog<bool>(
            context: context,
            builder: (dialogContext) => ContentDialog(
              title: Text(l10n.settingsReinstallConfirm),
              content: Text(l10n.settingsReinstallMessage),
              actions: [
                Button(
                  child: Text(l10n.cancel),
                  onPressed: () => Navigator.of(dialogContext).pop(false),
                ),
                FilledButton(
                  child: Text(l10n.settingsReinstallConfirmButton),
                  onPressed: () => Navigator.of(dialogContext).pop(true),
                ),
              ],
            ),
          );

          if (result == true) {
            // 获取配置文件路径和应用目录
            final configFilePath = await settingsService.getConfigFilePath();
            final appDirectory = configFilePath.substring(0, configFilePath.indexOf('data'));

            // 执行重装操作（包含完整的检查更新 -> 下载 -> 安装流程）
            final success = await updateProvider.reinstallLatestVersion(
              appDirectory: appDirectory,
              configFilePath: configFilePath,
              onStatusUpdate: (status) {
                // 可以在这里添加进度显示
              },
            );

            if (!success) {
              // 显示重装失败提示
              if (!context.mounted) return;
              displayInfoBar(
                context,
                builder: (context, close) {
                  return InfoBar(
                    title: Text(l10n.settingsReinstallFailed),
                    content: Text(l10n.settingsReinstallFailedMessage(updateProvider.updateStatus)),
                    action: IconButton(
                      icon: const Icon(FluentIcons.clear),
                      onPressed: close,
                    ),
                    severity: InfoBarSeverity.error,
                  );
                },
              );
            }
            // 如果成功，installUpdate 会自动退出应用并执行更新脚本
          }
        },
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll(
            EdgeInsets.symmetric(horizontal: 16, vertical: 18),
          ),
          backgroundColor: WidgetStatePropertyAll(
            resources.controlFillColorSecondary,
          ),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [Text(l10n.settingsReinstallLatest), const Icon(FluentIcons.download)],
        ),
      ),
      const SizedBox(height: 12),
    ],
  );
}

/// 构建设置页分组标题，统一样式与信息层级
Widget buildSectionHeader(BuildContext context, String title) {
  final typography = FluentTheme.of(context).typography;
  return Text(title, style: typography.subtitle);
}

/// 构建路径操作行
Widget buildPathActionRow(
  BuildContext context, {
  required bool isDirty,
  required VoidCallback onSave,
  required VoidCallback onSelect,
  required String saveLabel,
}) {
  final l10n = AppLocalizations.of(context)!;
  return Wrap(
    spacing: 8,
    runSpacing: 8,
    children: [
      Button(onPressed: isDirty ? onSave : null, child: Text(saveLabel)),
      Button(onPressed: onSelect, child: Text(l10n.browse)),
    ],
  );
}

/// 构建部分分隔符
Widget buildSectionDivider({double top = 32, double bottom = 32}) {
  return Padding(
    padding: EdgeInsetsDirectional.only(top: top, bottom: bottom),
    child: const Divider(),
  );
}
