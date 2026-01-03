import 'dart:convert';
import 'dart:io';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:http/http.dart' as http;
import 'package:archive/archive.dart';
import '../utils/logger.dart';

class UpdateService {
  final String _githubRepoUrl = 'https://api.github.com/repos/youzhiran/CarryDock/releases/latest';
  String _customUpdateUrl = '';
  
  void setCustomUpdateUrl(String url) {
    _customUpdateUrl = url;
  }
  
  // 获取当前应用版本信息
  Future<PackageInfo> getCurrentVersion() async {
    return await PackageInfo.fromPlatform();
  }
  
  // 从GitHub获取最新版本信息
  Future<Map<String, dynamic>?> getLatestVersionFromGitHub() async {
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
          logger.w('GitHub Release中没有找到Windows版本的资产');
        }
      } else {
        logger.e('从GitHub获取版本信息失败: HTTP ${response.statusCode} - ${response.body}');
      }
    } catch (e) {
      logger.e('从GitHub获取版本信息失败: $e');
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
  Future<File?> downloadUpdate(String downloadUrl, String savePath) async {
    try {
      logger.i('开始下载更新包: $downloadUrl');
      final response = await http.get(Uri.parse(downloadUrl));
      
      if (response.statusCode == 200) {
        final file = File(savePath);
        await file.writeAsBytes(response.bodyBytes);
        logger.i('更新包下载完成: $savePath');
        return file;
      }
    } catch (e) {
      logger.e('下载更新包失败: $e');
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
      
      // 删除应用目录下除配置文件外的所有文件
      logger.i('删除应用目录下除配置文件外的所有文件: $appDirectory');
      final appDir = Directory(appDirectory);
      final appFiles = appDir.listSync(recursive: true);
      
      for (final file in appFiles) {
        final filePath = file.path;
        if (filePath != configFilePath) {
          if (file is File) {
            await file.delete();
          } else if (file is Directory) {
            await file.delete(recursive: true);
          }
        }
      }
      
      // 将解压后的文件复制到应用目录
      logger.i('将解压后的文件复制到应用目录');
      final tempContents = Directory(tempPath).listSync();
      Directory? tempUpdateDir;
      
      // 查找第一个目录
      for (final entity in tempContents) {
        if (entity is Directory) {
          tempUpdateDir = entity;
          break;
        }
      }
      
      if (tempUpdateDir != null) {
        final updateFiles = tempUpdateDir.listSync(recursive: true);
        for (final file in updateFiles) {
          final relativePath = file.path.replaceFirst(tempUpdateDir.path, '').substring(1);
          final targetPath = '$appDirectory/$relativePath';
          
          if (file is File) {
            final targetFile = File(targetPath);
            targetFile.parent.createSync(recursive: true);
            await file.copy(targetFile.path);
          } else if (file is Directory) {
            final targetDir = Directory(targetPath);
            targetDir.createSync(recursive: true);
          }
        }
      }
      
      // 恢复配置文件
      logger.i('恢复配置文件: $configFilePath');
      await configFile.writeAsBytes(configContent);
      
      // 清理临时目录
      logger.i('清理临时目录: $tempPath');
      tempDir.deleteSync(recursive: true);
      
      logger.i('更新安装完成');
      return true;
    } catch (e) {
      logger.e('安装更新失败: $e');
      return false;
    }
  }
}