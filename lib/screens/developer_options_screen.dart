import 'dart:typed_data';

import 'package:carrydock/l10n/app_localizations.dart';
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
    final l10n = AppLocalizations.of(context)!;
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
      await _showMessageDialog(l10n.devExtractFailed, l10n.devExtractFailedMessage);
      return;
    }

    final imageProvider = MemoryImage(resolvedBytes);
    final fileName = p.basename(path);

    await showDialog(
      context: context,
      builder: (dialogContext) {
        final typography = FluentTheme.of(dialogContext).typography;
        return ContentDialog(
          title: Text(l10n.devIconTest),
          content: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('${l10n.devFileName}$fileName'),
                const SizedBox(height: 4),
                SelectableText(path),
                const SizedBox(height: 16),
                _buildIconPreviewTile(
                  dialogContext,
                  title: l10n.devMethod1,
                  description: l10n.devMethod1Desc,
                  child: Image.memory(
                    resolvedBytes,
                    width: 32,
                    height: 32,
                    filterQuality: FilterQuality.high,
                  ),
                ),
                _buildIconPreviewTile(
                  dialogContext,
                  title: l10n.devMethod2,
                  description: l10n.devMethod2Desc,
                  child: Image.memory(
                    resolvedBytes,
                    width: 64,
                    height: 64,
                    filterQuality: FilterQuality.high,
                  ),
                ),
                _buildIconPreviewTile(
                  dialogContext,
                  title: l10n.devMethod3,
                  description: l10n.devMethod3Desc,
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
                            Text(l10n.devButtonWithIcon),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
                if (typography.caption != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    l10n.devIconHint,
                    style: typography.caption,
                  ),
                ],
              ],
            ),
          ),
          actions: [
            Button(
              child: Text(l10n.devClose),
              onPressed: () => Navigator.of(dialogContext).pop(),
            ),
          ],
        );
      },
    );
  }

  Future<void> _showMessageDialog(String title, String content) async {
    if (!mounted) return;
    final l10n = AppLocalizations.of(context)!;
    await showDialog(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          Button(
            child: Text(l10n.ok),
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
    final l10n = AppLocalizations.of(context)!;
    final typography = FluentTheme.of(context).typography;
    final developerProvider = context.read<DeveloperOptionsProvider>();
    return NavigationView(
      content: ScaffoldPage.scrollable(
        children: [
          Text(l10n.devTitle, style: typography.title),
          const SizedBox(height: 12),
          Text(l10n.devHint, style: typography.body),
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _showIconTestDialog,
            child: Text(l10n.devIconTest),
          ),
          const SizedBox(height: 8),
          Text(l10n.devIconTestHint, style: typography.caption),
          const SizedBox(height: 24),
          Button(
            onPressed: () async {
              final navigator = Navigator.of(context);
              await developerProvider.setEnabled(false);
              if (!mounted) return;
              navigator.maybePop();
            },
            child: Text(l10n.devHide),
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
                  _showMessageDialog(l10n.devHiddenFeature, l10n.devHiddenFeatureMessage),
              child: Text(l10n.devHiddenButton),
            ),
          ),
        ],
      ),
    );
  }
}
