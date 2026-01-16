import 'dart:io';
import 'package:carrydock/l10n/app_localizations.dart';
import 'package:fluent_ui/fluent_ui.dart';

/// 7z.exe 路径测试功能

/// 测试7z.exe路径
Future<void> test7ZipPathFinding(BuildContext context) async {
  final l10n = AppLocalizations.of(context)!;
  final results = <String>[];
  String? foundPath;

  try {
    results.add('=== 7z.exe 路径测试开始 ===\n');

    // 1. 检查常见路径
    results.add('1. 检查常见安装路径：');
    final commonPaths = [
      r'C:\Program Files\7-Zip\7z.exe',
      r'C:\Program Files (x86)\7-Zip\7z.exe',
      r'C:\Apps\7-Zip\7z.exe',
      r'D:\Program Files\7-Zip\7z.exe',
      r'D:\Program Files (x86)\7-Zip\7z.exe',
    ];

    for (final path in commonPaths) {
      final exists = await File(path).exists();
      results.add('   ${exists ? '✓' : '✗'} $path');
      if (exists) {
        foundPath = path;
      }
    }
    results.add('');

    // 2. 检查系统PATH
    results.add('2. 检查系统PATH中的7z命令：');
    try {
      final pathResult = await Process.run('where', ['7z'], runInShell: true);
      if (pathResult.exitCode == 0 &&
          pathResult.stdout.toString().trim().isNotEmpty) {
        final paths = pathResult.stdout.toString().trim().split('\r\n');
        for (final path in paths) {
          final exists = await File(path).exists();
          results.add('   ${exists ? '✓' : '✗'} $path');
          if (exists && foundPath == null) {
            foundPath = path;
          }
        }
      } else {
        results.add('   ✗ 未在PATH中找到7z命令');
      }
    } catch (e) {
      results.add('   ✗ 通过where命令查找失败：${e.toString()}');
    }
    results.add('');

    // 3. 检查注册表
    results.add('3. 检查Windows注册表：');
    String? regPath;

    // 检查64位注册表
    try {
      final regResult = await Process.run('reg', [
        'query',
        r'HKLM\SOFTWARE\7-Zip',
        '/v',
        'Path',
      ], runInShell: true);
      if (regResult.exitCode == 0) {
        final output = regResult.stdout.toString();
        final match = RegExp(
          r'Path\s+REG_SZ\s+([^\r\n]+)',
        ).firstMatch(output);
        if (match != null) {
          regPath = match.group(1)!.trim();
          final exePath = regPath.endsWith('\\')
              ? '${regPath}7z.exe'
              : '$regPath\\7z.exe';
          final exists = await File(exePath).exists();
          results.add('   ✓ 从注册表（64位）获取路径：$regPath');
          results.add('     ${exists ? '✓' : '✗'} 可执行文件：$exePath');
          if (exists && foundPath == null) {
            foundPath = exePath;
          }
        } else {
          results.add('   ✗ 未找到注册表项（64位）');
        }
      } else {
        results.add('   ✗ 无法访问注册表（64位）：${regResult.stderr}');
      }
    } catch (e) {
      results.add('   ✗ 读取注册表（64位）失败：${e.toString()}');
    }

    // 检查32位注册表
    try {
      final regResult = await Process.run('reg', [
        'query',
        r'HKLM\SOFTWARE\WOW6432Node\7-Zip',
        '/v',
        'Path',
      ], runInShell: true);
      if (regResult.exitCode == 0) {
        final output = regResult.stdout.toString();
        final match = RegExp(
          r'Path\s+REG_SZ\s+([^\r\n]+)',
        ).firstMatch(output);
        if (match != null) {
          regPath = match.group(1)!.trim();
          final exePath = regPath.endsWith('\\')
              ? '${regPath}7z.exe'
              : '$regPath\\7z.exe';
          final exists = await File(exePath).exists();
          results.add('   ✓ 从注册表（32位）获取路径：$regPath');
          results.add('     ${exists ? '✓' : '✗'} 可执行文件：$exePath');
          if (exists && foundPath == null) {
            foundPath = exePath;
          }
        } else {
          results.add('   ✗ 未找到注册表项（32位）');
        }
      } else {
        results.add('   ✗ 无法访问注册表（32位）：${regResult.stderr}');
      }
    } catch (e) {
      results.add('   ✗ 读取注册表（32位）失败：${e.toString()}');
    }
    results.add('');

    // 4. 最终结果
    results.add('4. 测试结果：');
    if (foundPath != null) {
      results.add('   ✓ 成功找到7z.exe：$foundPath');
    } else {
      results.add('   ✗ 未找到7z.exe，请检查是否已安装7-Zip软件');
    }
    results.add('\n=== 测试完成 ===');

    // 显示结果
    await showDialog(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: Text(l10n.test7ZipTitle),
        content: SingleChildScrollView(
          child: SelectableText(results.join('\n')),
        ),
        actions: [
          Button(
            child: Text(l10n.test7ZipClose),
            onPressed: () => Navigator.of(dialogContext).pop(),
          ),
        ],
      ),
    );
  } catch (e) {
    await showDialog(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: Text('${l10n.test7ZipTitle} (${l10n.error})'),
        content: Text(l10n.test7ZipError(e.toString())),
        actions: [
          Button(
            child: Text(l10n.test7ZipClose),
            onPressed: () => Navigator.of(dialogContext).pop(),
          ),
        ],
      ),
    );
  }
}
