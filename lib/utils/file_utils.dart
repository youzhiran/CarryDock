import 'dart:ffi';
import 'dart:io';

import 'package:ffi/ffi.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:win32/win32.dart';

import 'logger.dart';

class FileUtils {
  static Future<String> get _startMenuPath async {
    final appData = await getApplicationSupportDirectory(); // 通常是 %APPDATA%
    // 开始菜单路径通常位于 %APPDATA%\Microsoft\Windows\Start Menu\Programs
    final startMenu = p.join(
      appData.parent.path,
      'Microsoft',
      'Windows',
      'Start Menu',
      'Programs',
    );
    return startMenu;
  }

  static Future<Directory> getCarryDockStartMenuDir() async {
    final startMenu = await _startMenuPath;
    final dir = Directory(p.join(startMenu, 'CarryDock'));
    if (!await dir.exists()) {
      logger.i('创建开始菜单目录: ${dir.path}');
      await dir.create(recursive: true);
    }
    return dir;
  }

  static void createShortcut(String targetPath, String shortcutPath) {
    logger.i('尝试创建快捷方式: $shortcutPath -> $targetPath');
    // 初始化 COM
    CoInitializeEx(Pointer.fromAddress(0), COINIT_APARTMENTTHREADED);

    final shellLink = ShellLink.createInstance();
    final pTargetPath = targetPath.toNativeUtf16();
    final pDescription = p.basenameWithoutExtension(targetPath).toNativeUtf16();
    final pShortcutPath = shortcutPath.toNativeUtf16();

    try {
      shellLink.setPath(pTargetPath);
      shellLink.setDescription(pDescription);

      final persistFile = IPersistFile.from(shellLink);
      persistFile.save(pShortcutPath, 1);
      persistFile.release();
      logger.i('成功创建快捷方式: $shortcutPath');
    } catch (e, s) {
      logger.e('创建快捷方式失败', error: e, stackTrace: s);
      // 重新抛出异常，以便调用者知道操作失败
      rethrow;
    } finally {
      calloc.free(pTargetPath);
      calloc.free(pDescription);
      calloc.free(pShortcutPath);
      shellLink.release();
      // 反初始化 COM
      CoUninitialize();
    }
  }
}
