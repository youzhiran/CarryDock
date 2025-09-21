import 'dart:typed_data';

import 'package:carrydock/providers/developer_options_provider.dart';
import 'package:carrydock/services/executable_info_service.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:file_picker/file_picker.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:path/path.dart' as p;
import 'package:provider/provider.dart';

class DeveloperOptionsScreen extends StatefulWidget {
  const DeveloperOptionsScreen({super.key});

  @override
  State<DeveloperOptionsScreen> createState() => _DeveloperOptionsScreenState();
}

class _DeveloperOptionsScreenState extends State<DeveloperOptionsScreen> {
  final ExecutableInfoService _executableInfoService = ExecutableInfoService();

  Future<void> _showIconTestDialog() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: const ['exe', 'lnk', 'ico'],
    );
    if (result == null) {
      return;
    }
    final path = result.files.single.path;
    if (path == null) {
      return;
    }

    Uint8List? iconBytes;
    try {
      iconBytes = await _executableInfoService.getIcon(path);
    } catch (e, s) {
      logger.e('提取软件图标失败', error: e, stackTrace: s);
    }

    if (!mounted) {
      return;
    }

    final resolvedBytes = iconBytes;
    if (resolvedBytes == null || resolvedBytes.isEmpty) {
      await _showMessageDialog('提取失败', '未能从所选文件中提取图标，请尝试其他程序。');
      return;
    }

    final imageProvider = MemoryImage(resolvedBytes);
    final fileName = p.basename(path);

    await showDialog(
      context: context,
      builder: (dialogContext) {
        final typography = FluentTheme.of(dialogContext).typography;
        return ContentDialog(
          title: const Text('软件图标测试'),
          content: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('文件名称：$fileName'),
                const SizedBox(height: 4),
                SelectableText(path),
                const SizedBox(height: 16),
                _buildIconPreviewTile(
                  dialogContext,
                  title: '方法一：Image.memory (32×32)',
                  description: '直接渲染 PNG 数据，适合在列表或按钮中展示标准尺寸图标。',
                  child: Image.memory(
                    resolvedBytes,
                    width: 32,
                    height: 32,
                    filterQuality: FilterQuality.high,
                  ),
                ),
                _buildIconPreviewTile(
                  dialogContext,
                  title: '方法二：Image.memory (64×64)',
                  description: '放大查看细节，便于检查透明度和边缘处理是否正常。',
                  child: Image.memory(
                    resolvedBytes,
                    width: 64,
                    height: 64,
                    filterQuality: FilterQuality.high,
                  ),
                ),
                _buildIconPreviewTile(
                  dialogContext,
                  title: '方法三：ImageIcon 与 Fluent 按钮',
                  description: '作为按钮图标或工具栏图标使用，体验交互态效果。',
                  child: Wrap(
                    spacing: 12,
                    runSpacing: 12,
                    children: [
                      IconButton(
                        icon: ImageIcon(imageProvider, color: null, size: 28),
                        onPressed: () {},
                      ),
                      FilledButton(
                        onPressed: () {},
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Image(
                              image: imageProvider,
                              width: 20,
                              height: 20,
                              filterQuality: FilterQuality.high,
                            ),
                            const SizedBox(width: 8),
                            const Text('带图标按钮示例'),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
                if (typography.caption != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    '提示：如果图标显示异常，请检查源文件是否包含标准的主图标资源。',
                    style: typography.caption,
                  ),
                ],
              ],
            ),
          ),
          actions: [
            Button(
              child: const Text('关闭'),
              onPressed: () => Navigator.of(dialogContext).pop(),
            ),
          ],
        );
      },
    );
  }

  Future<void> _showMessageDialog(String title, String content) async {
    if (!mounted) return;
    await showDialog(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          Button(
            child: const Text('确定'),
            onPressed: () => Navigator.of(dialogContext).pop(),
          ),
        ],
      ),
    );
  }

  Widget _buildIconPreviewTile(
    BuildContext context, {
    required String title,
    required String description,
    required Widget child,
  }) {
    final typography = FluentTheme.of(context).typography;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(title, style: typography.bodyStrong),
          const SizedBox(height: 4),
          Text(description, style: typography.caption),
          const SizedBox(height: 8),
          _buildIconPreviewContainer(context, child),
        ],
      ),
    );
  }

  Widget _buildIconPreviewContainer(BuildContext context, Widget child) {
    final resources = FluentTheme.of(context).resources;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: resources.controlFillColorSecondary,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: resources.controlStrokeColorSecondary),
      ),
      child: child,
    );
  }

  @override
  Widget build(BuildContext context) {
    final typography = FluentTheme.of(context).typography;
    final developerProvider = context.read<DeveloperOptionsProvider>();
    return NavigationView(
      content: ScaffoldPage.scrollable(
        children: [
          Text('开发者选项', style: typography.title),
          const SizedBox(height: 12),
          Text('这里包含一些实验性功能，用于验证桌面端实现效果。请谨慎操作。', style: typography.body),
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _showIconTestDialog,
            child: const Text('软件图标测试'),
          ),
          const SizedBox(height: 8),
          Text('选择一个 EXE、LNK 或 ICO 文件，以不同方式预览其图标。', style: typography.caption),
          const SizedBox(height: 24),
          Button(
            onPressed: () async {
              final navigator = Navigator.of(context);
              await developerProvider.setEnabled(false);
              if (!mounted) return;
              navigator.maybePop();
            },
            child: const Text('隐藏开发者选项'),
          ),
          const SizedBox(height: 12),
          Opacity(
            opacity: 0.2,
            child: Button(
              style: const ButtonStyle(
                padding: WidgetStatePropertyAll(
                  EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                ),
              ),
              onPressed: () =>
                  _showMessageDialog('隐藏功能', '恭喜发现隐藏按钮！当前没有额外操作，仅用于验证交互。'),
              child: const Text('隐藏按钮'),
            ),
          ),
        ],
      ),
    );
  }
}
