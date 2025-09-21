import 'dart:typed_data';

import 'package:carrydock/services/executable_info_service.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:path/path.dart' as p;

class SelectExecutableDialog extends StatefulWidget {
  final List<String> executablePaths;

  const SelectExecutableDialog({super.key, required this.executablePaths});

  @override
  State<SelectExecutableDialog> createState() => _SelectExecutableDialogState();
}

class _SelectExecutableDialogState extends State<SelectExecutableDialog> {
  String? _selectedPath;

  @override
  void initState() {
    super.initState();
    if (widget.executablePaths.isNotEmpty) {
      _selectedPath = widget.executablePaths.first;
    }
  }

  @override
  Widget build(BuildContext context) {
    return ContentDialog(
      title: const Text('选择主程序'),
      content: SizedBox(
        height: 300,
        width: 500,
        child: ListView.builder(
          itemCount: widget.executablePaths.length,
          itemBuilder: (context, index) {
            final path = widget.executablePaths[index];
            return ListTile(
              title: _ExecutableInfoTile(exePath: path),
              leading: RadioButton(
                checked: _selectedPath == path,
                onChanged: (value) {
                  setState(() {
                    _selectedPath = path;
                  });
                },
              ),
              onPressed: () {
                setState(() {
                  _selectedPath = path;
                });
              },
            );
          },
        ),
      ),
      actions: [
        Button(
          child: const Text('取消'),
          onPressed: () => Navigator.of(context).pop(),
        ),
        FilledButton(
          onPressed: _selectedPath == null
              ? null
              : () => Navigator.of(context).pop(_selectedPath),
          child: const Text('确定'),
        ),
      ],
    );
  }
}

class _ExecutableInfoTile extends StatefulWidget {
  final String exePath;

  const _ExecutableInfoTile({required this.exePath});

  @override
  State<_ExecutableInfoTile> createState() => _ExecutableInfoTileState();
}

class _ExecutableInfoTileState extends State<_ExecutableInfoTile> {
  final ExecutableInfoService _infoService = ExecutableInfoService();
  Uint8List? _iconData;
  String? _fileDescription;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadInfo();
  }

  Future<void> _loadInfo() async {
    final icon = await _infoService.getIcon(widget.exePath);
    final desc = await _infoService.getFileDescription(widget.exePath);
    if (mounted) {
      setState(() {
        _iconData = icon;
        _fileDescription = desc;
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    Widget leading;
    if (_isLoading) {
      leading = const ProgressRing(value: null, strokeWidth: 2);
    } else if (_iconData != null) {
      leading = Image.memory(_iconData!, width: 24, height: 24);
    } else {
      leading = const Icon(FluentIcons.app_icon_default, size: 24);
    }

    return Row(
      children: [
        leading,
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(_fileDescription ?? p.basename(widget.exePath)),
              Text(
                p.basename(widget.exePath),
                style: FluentTheme.of(context).typography.caption,
              ),
            ],
          ),
        ),
      ],
    );
  }
}
