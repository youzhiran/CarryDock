import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

import 'logger.dart';

class FileUtils {
  static Future<String> get _startMenuPath async {
    // 使用环境变量获取系统 AppData 目录，而不是应用程序特定目录
    final appData = Platform.environment['APPDATA'] ?? '';
    if (appData.isEmpty) {
      // 如果环境变量获取失败，使用备用方法
      final appSupport = await getApplicationSupportDirectory();
      // 尝试从应用支持目录向上导航到 AppData 根目录
      return p.join(
        appSupport.path
            .split('\\')
            .takeWhile((part) => part != 'com.devyi')
            .join('\\'),
        'Microsoft',
        'Windows',
        'Start Menu',
        'Programs',
      );
    }
    // 开始菜单路径通常位于 %APPDATA%\Microsoft\Windows\Start Menu\Programs
    final startMenu = p.join(
      appData,
      'Microsoft',
      'Windows',
      'Start Menu',
      'Programs',
    );
    return startMenu;
  }

  static Future<Directory> getCarryDockStartMenuDir() async {
    final startMenu = await _startMenuPath;
    final dir = Directory(p.join(startMenu, '绿驿管家'));
    if (!await dir.exists()) {
      logger.i('创建开始菜单目录: ${dir.path}');
      await dir.create(recursive: true);
    }
    return dir;
  }

  static Future<void> createShortcut(
    String targetPath,
    String shortcutPath, [
    String? description,
  ]) async {
    logger.i('=== 开始创建快捷方式 ===');
    logger.i('目标路径: $targetPath');
    logger.i('快捷方式路径: $shortcutPath');
    logger.i('描述: $description');

    try {
      // 使用提供的描述或从目标路径提取
      final shortcutDescription =
          description ?? p.basenameWithoutExtension(targetPath);

      // 使用 PowerShell 单行命令创建快捷方式，更安全可靠
      final psCommand =
          "\$WshShell = New-Object -ComObject WScript.Shell; \$Shortcut = \$WshShell.CreateShortcut('$shortcutPath'); \$Shortcut.TargetPath = '$targetPath'; \$Shortcut.Description = '$shortcutDescription'; \$Shortcut.Save();";

      logger.i('1. 执行 PowerShell 命令创建快捷方式');
      logger.i('   命令: $psCommand');

      // 使用 Process.run 执行 PowerShell 命令
      final result = await Process.run('powershell.exe', [
        '-ExecutionPolicy',
        'Bypass',
        '-Command',
        psCommand,
      ], runInShell: true);

      logger.i('2. PowerShell 命令执行完成');
      logger.i('   退出代码: ${result.exitCode}');
      if (result.stdout.isNotEmpty) {
        logger.i('   标准输出: ${result.stdout}');
      }
      if (result.stderr.isNotEmpty) {
        logger.i('   标准错误: ${result.stderr}');
      }

      if (result.exitCode == 0) {
        logger.i('=== 快捷方式创建成功 ===');
      } else {
        logger.e('创建快捷方式失败，PowerShell 命令执行失败');
      }
    } catch (e, s) {
      logger.e('创建快捷方式失败', error: e, stackTrace: s);
      // 不再重新抛出异常，防止应用崩溃
    }
  }
}
