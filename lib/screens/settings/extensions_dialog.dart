import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import 'package:carrydock/utils/error_handler.dart';
import 'package:carrydock/screens/settings/utils.dart';

/// 可执行文件扩展名选择对话框

/// 打开可执行文件扩展名选择对话框
Future<void> openExecutableExtensionsDialog(BuildContext context, {
  required List<String> initialExtensions,
  required Function(List<String>) onExtensionsSelected,
}) async {
  final customController = TextEditingController();
  List<String>? result;
  try {
    result = await showDialog<List<String>>(
      context: context,
      builder: (dialogContext) {
        final navigator = Navigator.of(dialogContext);
        final localSelections = <String>{...initialExtensions};
        String? errorMessage;

        return StatefulBuilder(
          builder: (context, setState) {
            final sortedSelections = localSelections.toList()..sort();

            void toggleCommonExtension(String extension) {
              setState(() {
                if (localSelections.contains(extension)) {
                  localSelections.remove(extension);
                } else {
                  localSelections.add(extension);
                }
                errorMessage = null;
              });
            }

            void addCustomExtension() {
              final normalized = normalizeExtensionsList([
                customController.text,
              ]);
              if (normalized.isEmpty) {
                setState(() {
                  errorMessage = '请输入有效的扩展名（示例：exe）';
                });
                return;
              }
              final value = normalized.first;
              if (localSelections.contains(value)) {
                setState(() {
                  errorMessage = '该扩展名已存在';
                });
                return;
              }
              setState(() {
                localSelections.add(value);
                errorMessage = null;
                customController.clear();
              });
            }

            Widget buildSelectedExtensions() {
              if (sortedSelections.isEmpty) {
                return Text(
                  '暂未选择任何扩展名',
                  style: FluentTheme.of(context).typography.caption,
                );
              }
              return Wrap(
                spacing: 8,
                runSpacing: 8,
                children: sortedSelections.map((ext) {
                  return _buildExtensionPill(
                    context,
                    extension: ext,
                    onRemove: () {
                      setState(() {
                        localSelections.remove(ext);
                        errorMessage = null;
                      });
                    },
                  );
                }).toList(),
              );
            }

            return ContentDialog(
              title: const Text('选择可执行文件扩展名'),
              content: SizedBox(
                width: 360,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('常用扩展名'),
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: commonExecutableExtensions.map((ext) {
                        final isSelected = localSelections.contains(ext);
                        return ToggleButton(
                          checked: isSelected,
                          onChanged: (_) => toggleCommonExtension(ext),
                          child: Text(ext),
                        );
                      }).toList(),
                    ),
                    const SizedBox(height: 16),
                    const Text('自定义扩展名'),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        Expanded(
                          child: TextBox(
                            controller: customController,
                            placeholder: '输入扩展名，例如 exe',
                            onSubmitted: (_) => addCustomExtension(),
                          ),
                        ),
                        const SizedBox(width: 8),
                        FilledButton(
                          onPressed: addCustomExtension,
                          child: const Text('添加'),
                        ),
                      ],
                    ),
                    if (errorMessage != null) ...[
                      const SizedBox(height: 8),
                      Text(
                        errorMessage!,
                        style: FluentTheme.of(context).typography.caption
                            ?.copyWith(color: Colors.red) ??
                        TextStyle(color: Colors.red),
                      ),
                    ],
                    const SizedBox(height: 16),
                    const Text('已选择'),
                    const SizedBox(height: 8),
                    buildSelectedExtensions(),
                  ],
                ),
              ),
              actions: [
                Button(
                  child: const Text('取消'),
                  onPressed: () => navigator.pop(),
                ),
                FilledButton(
                  child: const Text('确定'),
                  onPressed: () {
                    if (localSelections.isEmpty) {
                      Provider.of<ErrorHandler>(
                        context,
                        listen: false,
                      ).showHint('设置错误', '请至少选择一个扩展名');
                      return;
                    }
                    navigator.pop(normalizeExtensionsList(localSelections));
                  },
                ),
              ],
            );
          },
        );
      },
    );
  } finally {
    customController.dispose();
  }

  if (result != null) {
    onExtensionsSelected(result);
  }
}

/// 构建扩展名标签
Widget buildExtensionTag(BuildContext context, {
  required String extension,
  required Function(String) onRemove,
}) {
  return _buildExtensionPill(
    context,
    extension: extension,
    onRemove: () => onRemove(extension),
  );
}

/// 构建扩展名胶囊组件
Widget _buildExtensionPill(
  BuildContext context, {
  required String extension,
  required VoidCallback onRemove,
}) {
  final resources = FluentTheme.of(context).resources;
  return Container(
    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
    decoration: BoxDecoration(
      color: resources.controlFillColorSecondary,
      borderRadius: BorderRadius.circular(16),
    ),
    child: Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(extension),
        const SizedBox(width: 6),
        IconButton(
          icon: const Icon(FluentIcons.chrome_close),
          style: const ButtonStyle(
            padding: WidgetStatePropertyAll(EdgeInsets.all(4)),
            iconSize: WidgetStatePropertyAll(12.0),
          ),
          onPressed: onRemove,
        ),
      ],
    ),
  );
}
