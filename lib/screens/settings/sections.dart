import 'package:carrydock/providers/update_provider.dart';
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
            const Text('版本'),
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
            const Text('编译时间'),
            Text(buildTime, style: typography.bodyStrong),
          ],
        ),
      ),
      const SizedBox(height: 12),
      Button(
        onPressed: () async {
          await updateProvider.checkForUpdates();
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
                  title: const Text('成功'),
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
        child: const Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [Text('检查更新'), Icon(FluentIcons.refresh)],
        ),
      ),
      const SizedBox(height: 12),
      Button(
        onPressed: () async {
          // 确认重装操作
          final result = await showDialog<bool>(
            context: context,
            builder: (dialogContext) => ContentDialog(
              title: const Text('确认重装'),
              content: const Text('确定要重装最新版本吗？这将覆盖当前安装的版本，但会保留配置文件。'),
              actions: [
                Button(
                  child: const Text('取消'),
                  onPressed: () => Navigator.of(dialogContext).pop(false),
                ),
                FilledButton(
                  child: const Text('确认'),
                  onPressed: () => Navigator.of(dialogContext).pop(true),
                ),
              ],
            ),
          );

          if (result == true) {
            // 执行重装操作
            // 调用checkForUpdates并设置本地版本为1，确保能检测到更新
            final hasUpdate = await updateProvider.checkForUpdates(
              customLocalBuildNumber: 1,
            );

            if (hasUpdate && updateProvider.latestVersionInfo != null) {
              // 发现新版本后，直接显示更新对话框，使用现有的更新流程
              await showDialog(
                context: context,
                builder: (dialogContext) =>
                    UpdateDialog(updateProvider: updateProvider),
              );
            } else {
              // 显示获取版本信息失败提示
              displayInfoBar(
                context,
                builder: (context, close) {
                  return InfoBar(
                    title: const Text('失败'),
                    content: Text('获取最新版本信息失败：${updateProvider.updateStatus}'),
                    action: IconButton(
                      icon: const Icon(FluentIcons.clear),
                      onPressed: close,
                    ),
                    severity: InfoBarSeverity.error,
                  );
                },
              );
            }
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
        child: const Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [Text('重装最新版本'), Icon(FluentIcons.download)],
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
Widget buildPathActionRow({
  required bool isDirty,
  required VoidCallback onSave,
  required VoidCallback onSelect,
  required String saveLabel,
}) {
  // 使用 Wrap 确保在窄屏幕下按钮可以自动换行
  return Wrap(
    spacing: 8,
    runSpacing: 8,
    children: [
      Button(onPressed: isDirty ? onSave : null, child: Text(saveLabel)),
      Button(onPressed: onSelect, child: const Text('选择目录')),
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
