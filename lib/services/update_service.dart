import 'dart:convert';
import 'dart:io';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:http/http.dart' as http;
import 'package:archive/archive.dart';
import '../utils/logger.dart';

class UpdateService {
  final String _githubRepoUrl = 'https://api.github.com/repos/youzhiran/CarryDock/releases/latest';
  String _customUpdateUrl = '';
  String _lastError = '';

  String get lastError => _lastError;

  void _clearError() {
    _lastError = '';
  }

  void _setError(String error) {
    _lastError = error;
  }

  void setCustomUpdateUrl(String url) {
    _customUpdateUrl = url;
  }
  
  // 获取当前应用版本信息
  Future<PackageInfo> getCurrentVersion() async {
    return await PackageInfo.fromPlatform();
  }
  
  // 从GitHub获取最新版本信息
  Future<Map<String, dynamic>?> getLatestVersionFromGitHub() async {
    _clearError();
    try {
      logger.i('从GitHub获取最新版本信息');
      final response = await http.get(Uri.parse(_githubRepoUrl));

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        final tagName = data['tag_name'] as String;
        final body = data['body'] as String;
        final assets = data['assets'] as List;

        logger.i('GitHub Release资产列表: ${assets.map((asset) => asset['name']).toList()}');

        // 查找Windows版本的资产
        final windowsAsset = assets.firstWhere(
          (asset) {
            final assetName = asset['name'] as String;
            // 修改判断逻辑，接受所有.zip文件作为Windows资产，因为GitHub Release中可能只有一个通用资产
            final isWindowsAsset = assetName.toLowerCase().endsWith('.zip') || assetName.toLowerCase().contains('windows');
            logger.i('检查资产: $assetName, 是否为Windows资产: $isWindowsAsset');
            return isWindowsAsset;
          },
          orElse: () {
            logger.w("未找到包含'windows'或.zip扩展名的资产");
            return null;
          },
        );

        if (windowsAsset != null) {
          logger.i('找到Windows资产: ${windowsAsset['name']}');
          return {
            'version': tagName,
            'changelog': body,
            'downloadUrl': windowsAsset['browser_download_url'] as String,
            'assetName': windowsAsset['name'] as String,
          };
        } else {
          final error = 'GitHub Release中没有找到Windows版本的资产';
          logger.w(error);
          _setError(error);
        }
      } else {
        // 对所有HTTP错误都尝试解析详细错误信息
        String errorMsg = '从GitHub获取版本信息失败: HTTP ${response.statusCode}';
        try {
          final errorData = json.decode(response.body);
          if (errorData['message'] != null) {
            errorMsg += ' - ${errorData['message']}';
            // 如果有文档链接，也添加进去
            if (errorData['documentation_url'] != null) {
              errorMsg += '\n详情: ${errorData['documentation_url']}';
            }
          } else {
            errorMsg += ' - ${response.body}';
          }
        } catch (_) {
          errorMsg += ' - ${response.body}';
        }
        logger.e(errorMsg);
        _setError(errorMsg);
      }
    } catch (e) {
      final error = '从GitHub获取版本信息失败: $e';
      logger.e(error);
      _setError(error);
    }
    return null;
  }
  
  // 从自定义地址获取最新版本信息（预留接口）
  Future<Map<String, dynamic>?> getLatestVersionFromCustom() async {
    if (_customUpdateUrl.isEmpty) {
      return null;
    }
    
    try {
      logger.i('从自定义地址获取最新版本信息: $_customUpdateUrl');
      final response = await http.get(Uri.parse(_customUpdateUrl));
      
      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        // 这里假设自定义接口返回的格式与GitHub类似
        return {
          'version': data['version'] as String,
          'changelog': data['changelog'] as String,
          'downloadUrl': data['downloadUrl'] as String,
          'assetName': data['assetName'] as String,
        };
      }
    } catch (e) {
      logger.e('从自定义地址获取版本信息失败: $e');
    }
    return null;
  }
  
  // 获取最新版本信息（优先从GitHub获取，失败则尝试自定义地址）
  Future<Map<String, dynamic>?> getLatestVersion() async {
    var versionInfo = await getLatestVersionFromGitHub();
    if (versionInfo == null && _customUpdateUrl.isNotEmpty) {
      versionInfo = await getLatestVersionFromCustom();
    }
    return versionInfo;
  }
  
  // 比较版本号，判断是否需要更新
  Future<bool> isUpdateAvailable({int? customLocalBuildNumber}) async {
    final currentVersion = await getCurrentVersion();
    final latestVersionInfo = await getLatestVersion();
    
    if (latestVersionInfo == null) {
      logger.i('检查更新：未获取到远程版本信息');
      return false;
    }
    
    // 使用自定义本地版本号（用于测试或重装），否则使用真实本地版本号
    final currentBuildNumber = customLocalBuildNumber ?? int.tryParse(currentVersion.buildNumber) ?? 0;
    final latestVersion = latestVersionInfo['version'] as String;
    
    // 解析最新版本号的buildNumber，格式为v0.8.2+102
    final latestBuildNumberMatch = RegExp(r'\+(\d+)$').firstMatch(latestVersion);
    if (latestBuildNumberMatch != null) {
      final latestBuildNumber = int.parse(latestBuildNumberMatch.group(1)!);
      
      logger.i('检查更新：本地版本 - ${currentVersion.version}+${customLocalBuildNumber ?? currentVersion.buildNumber} (build: $currentBuildNumber)');
      logger.i('检查更新：远程版本 - $latestVersion (build: $latestBuildNumber)');
      
      final hasUpdate = latestBuildNumber > currentBuildNumber;
      logger.i('检查更新：是否需要更新 - $hasUpdate');
      
      return hasUpdate;
    }
    
    logger.i('检查更新：无法解析远程版本号的buildNumber - $latestVersion');
    return false;
  }
  
  // 下载更新包
  Future<File?> downloadUpdate(String downloadUrl, String savePath, {void Function(double progress)? onProgress, Function()? onCancel}) async {
    final client = http.Client();
    
    try {
      logger.i('开始下载更新包: $downloadUrl');
      final request = http.Request('GET', Uri.parse(downloadUrl));
      final response = await client.send(request);
      
      if (response.statusCode == 200) {
        final contentLength = response.contentLength;
        final file = File(savePath);
        final sink = file.openWrite();
        
        var receivedLength = 0;
        
        // 使用Stream.forEach下载文件，简化实现
        await response.stream.forEach((chunk) {
          sink.add(chunk);
          receivedLength += chunk.length;
          
          if (contentLength != null && onProgress != null) {
            final progress = receivedLength / contentLength;
            // 确保进度值在0-1之间
            final clampedProgress = progress.clamp(0.0, 1.0);
            onProgress(clampedProgress);
          }
        });
        
        await sink.close();
        logger.i('更新包下载完成: $savePath');
        return file;
      }
    } catch (e) {
      // 如果是取消下载，记录日志但不报错
      if (e.toString().contains('cancel')) {
        logger.i('下载已取消');
        // 尝试删除未完成的下载文件
        try {
          final file = File(savePath);
          if (await file.exists()) {
            await file.delete();
          }
        } catch (deleteError) {
          logger.w('删除未完成的下载文件失败: $deleteError');
        }
      } else {
        logger.e('下载更新包失败: $e');
      }
    } finally {
      client.close();
    }
    return null;
  }
  
  // 安装更新（保留配置文件，替换其他文件）
  Future<bool> installUpdate(String updateFilePath, String appDirectory, String configFilePath) async {
    try {
      logger.i('开始安装更新');
      
      // 读取配置文件内容
      final configFile = File(configFilePath);
      final configContent = await configFile.readAsBytes();
      
      // 解压更新包到临时目录
      final tempDir = Directory.systemTemp.createTempSync('carrydock_update_');
      final tempPath = tempDir.path;
      
      logger.i('解压更新包到临时目录: $tempPath');
      final bytes = await File(updateFilePath).readAsBytes();
      final archive = ZipDecoder().decodeBytes(bytes);
      
      for (final file in archive) {
        final filePath = '$tempPath/${file.name}';
        if (file.isFile) {
          final fileContent = file.content as List<int>;
          final outputFile = File(filePath);
          outputFile.parent.createSync(recursive: true);
          await outputFile.writeAsBytes(fileContent);
        }
      }

      // 递归查找包含可执行文件的目录
      Directory? findExecutableDirectory(Directory dir) {
        try {
          final contents = dir.listSync();
          // 先检查当前目录是否包含 .exe 文件
          if (contents.any((f) => f.path.endsWith('.exe'))) {
            return dir;
          }
          // 递归检查子目录
          for (final entity in contents) {
            if (entity is Directory) {
              final result = findExecutableDirectory(entity);
              if (result != null) {
                return result;
              }
            }
          }
        } catch (e) {
          logger.w('无法检查目录 ${dir.path}: $e');
        }
        return null;
      }

      // 查找包含可执行文件的目录（真正的应用目录）
      Directory? tempUpdateDir = findExecutableDirectory(Directory(tempPath));

      if (tempUpdateDir == null) {
        // 记录临时目录结构以便调试
        logger.e('更新包中未找到包含可执行文件的目录');
        try {
          final tempContents = Directory(tempPath).listSync(recursive: true);
          logger.i('临时目录内容:');
          for (final entity in tempContents) {
            logger.i('  ${entity.path}');
          }
        } catch (e) {
          logger.w('无法列出目录内容: $e');
        }
        return false;
      }

      logger.i('找到包含可执行文件的更新目录: ${tempUpdateDir.path}');
      
      // 创建更新脚本
      final scriptPath = '$tempPath/update_script.bat';

      // 确保路径使用反斜杠，并且正确处理空格
      // 移除路径末尾的分隔符，避免BAT脚本中的路径问题
      final formattedAppDir = appDirectory.replaceAll('/', '\\').replaceAll(RegExp(r'\\+$'), '');
      final formattedTempUpdateDir = tempUpdateDir.path.replaceAll('/', '\\').replaceAll(RegExp(r'\\+$'), '');
      final formattedConfigFile = configFilePath.replaceAll('/', '\\');
      final formattedConfigContent = '$tempPath\\config_backup.bin';

      // 查找可执行文件（.exe）
      String exeFileName = 'CarryDock.exe';
      try {
        final appDirFiles = Directory(appDirectory).listSync();
        for (final file in appDirFiles) {
          if (file.path.endsWith('.exe')) {
            exeFileName = file.path.split('\\').last;
            break;
          }
        }
      } catch (e) {
        logger.w('无法查找可执行文件，使用默认名称: $e');
      }

      // 使用 ASCII 字符避免编码问题
      final scriptContent = '''@echo off
if "%~1" neq "NEWWINDOW" (
    start "CarryDock Update" cmd /k ""%~f0" NEWWINDOW"
    exit /b
)

title CarryDock Update...
echo ============================================
echo CarryDock Auto Update Script
echo ============================================
echo.

set appDir=$formattedAppDir
set tempUpdateDir=$formattedTempUpdateDir
set configFile=$formattedConfigFile
set configContent=$formattedConfigContent
set exeFile=$exeFileName

echo App Directory: %appDir%
echo Update Source: %tempUpdateDir%
echo Config File: %configFile%
echo Executable: %exeFile%
echo.

:: Check if paths exist
if not exist "%appDir%" (
    echo ERROR: App directory does not exist!
    pause
    exit /b 1
)

if not exist "%tempUpdateDir%" (
    echo ERROR: Update source directory does not exist!
    pause
    exit /b 1
)

:: Wait for app to exit
echo Waiting for app to exit...
timeout /t 2 /nobreak >nul

:: Delete all files in app directory except config file
echo.
echo [1/5] Deleting old files in app directory...
for /r "%appDir%" %%f in (*) do (
    if not "%%f" == "%configFile%" (
        del /f /q "%%f" 2>nul
    )
)

:: Delete all empty directories in app directory
echo [2/5] Deleting empty directories...
for /f "delims=" %%d in ('dir /ad /b /s "%appDir%" ^| sort /r') do (
    rd "%%d" 2>nul
)

:: Copy extracted files to app directory (only content, not directory itself)
echo [3/5] Copying update files...
xcopy "%tempUpdateDir%\\*" "%appDir%" /e /y /i
if errorlevel 1 (
    echo ERROR: Failed to copy files!
    pause
    exit /b 1
)

:: Restore config file
echo [4/5] Restoring config file...
copy /y "%configContent%" "%configFile%"
if errorlevel 1 (
    echo WARNING: Failed to restore config file, but continuing...
)

:: Clean up temporary files
echo [5/5] Cleaning up temporary files...
cd /d "%TEMP%"
for /d %%d in (carrydock_update_*) do rd /s /q "%%d" 2>nul

echo.
echo ============================================
echo Update completed! Restarting application...
echo ============================================

:: Restart application
start "" "%appDir%\\%exeFile%"

echo.
echo ============================================
echo Press any key to close this window...
echo ============================================
pause
''';
      
      // 保存配置文件备份
      final configBackupFile = File('$tempPath/config_backup.bin');
      await configBackupFile.writeAsBytes(configContent);
      
      // 写入更新脚本
      final scriptFile = File(scriptPath);
      await scriptFile.writeAsString(scriptContent);

      logger.i('创建更新脚本: $scriptPath');

      // 运行更新脚本并退出应用
      // 使用 runInShell: true 确保在新窗口中运行
      logger.i('启动更新脚本并退出应用');

      await Process.start(
        scriptPath,
        [],
        runInShell: true,
        mode: ProcessStartMode.detached,
      );
      
      // 退出当前应用
      exit(0);
    } catch (e) {
      logger.e('安装更新失败: $e');
      return false;
    }
  }
}