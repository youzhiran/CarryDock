import 'dart:async';
import 'dart:io';
import 'package:flutter/material.dart';
import '../services/update_service.dart';

class UpdateProvider with ChangeNotifier {
  final UpdateService _updateService = UpdateService();
  bool _isCheckingUpdate = false;
  bool _hasUpdate = false;
  Map<String, dynamic>? _latestVersionInfo;
  String _updateStatus = '';
  double _downloadProgress = 0.0;
  bool _isDownloading = false;
  bool _isInstalling = false;
  bool _isReinstalling = false;
  
  // 用于取消下载的Completer
  Completer<bool>? _downloadCancelCompleter;
  
  bool get isCheckingUpdate => _isCheckingUpdate;
  bool get hasUpdate => _hasUpdate;
  Map<String, dynamic>? get latestVersionInfo => _latestVersionInfo;
  String get updateStatus => _updateStatus;
  double get downloadProgress => _downloadProgress;
  bool get isDownloading => _isDownloading;
  bool get isInstalling => _isInstalling;
  bool get isReinstalling => _isReinstalling;
  
  // 检查更新
  Future<bool> checkForUpdates({int? customLocalBuildNumber}) async {
    _isCheckingUpdate = true;
    _updateStatus = '正在检查更新...';
    notifyListeners();

    try {
      final isUpdateAvailable = await _updateService.isUpdateAvailable(customLocalBuildNumber: customLocalBuildNumber);
      if (isUpdateAvailable) {
        _latestVersionInfo = await _updateService.getLatestVersion();
        _hasUpdate = true;
        _updateStatus = '发现新版本！';
      } else {
        _hasUpdate = false;
        // 检查是否有错误信息
        if (_updateService.lastError.isNotEmpty) {
          _updateStatus = _updateService.lastError;
        } else {
          _updateStatus = '当前已是最新版本。';
        }
      }
    } catch (e) {
      _hasUpdate = false;
      _updateStatus = '检查更新失败：$e';
    } finally {
      _isCheckingUpdate = false;
      notifyListeners();
    }

    return _hasUpdate;
  }
  
  // 下载更新
  Future<File?> downloadUpdate(String savePath) async {
    if (_latestVersionInfo == null) {
      return null;
    }
    
    _isDownloading = true;
    _downloadProgress = 0.0;
    _updateStatus = '开始下载更新...';
    _downloadCancelCompleter = Completer<bool>();
    notifyListeners();
    
    try {
      final downloadUrl = _latestVersionInfo!['downloadUrl'] as String;
      final file = await _updateService.downloadUpdate(
        downloadUrl, 
        savePath, 
        onProgress: (progress) {
          // 确保进度值在0-1之间
          _downloadProgress = progress.clamp(0.0, 1.0);
          _updateStatus = '正在下载更新...';
          notifyListeners();
        },
        onCancel: () {
          _downloadCancelCompleter?.complete(true);
        }
      );
      
      if (file != null) {
        _updateStatus = '更新包下载完成。';
        _downloadProgress = 1.0;
        notifyListeners();
        return file;
      } else {
        _updateStatus = _downloadCancelCompleter?.isCompleted == true ? '下载已取消。' : '下载更新包失败。';
        notifyListeners();
        return null;
      }
    } catch (e) {
      _updateStatus = e.toString() == '下载已取消' ? '下载已取消。' : '下载更新包失败：$e';
      notifyListeners();
      return null;
    } finally {
      _isDownloading = false;
      _downloadCancelCompleter = null;
      notifyListeners();
    }
  }
  
  // 取消下载
  void cancelDownload() {
    if (_downloadCancelCompleter != null && !_downloadCancelCompleter!.isCompleted) {
      _downloadCancelCompleter!.complete(true);
      _updateStatus = '正在取消下载...';
      notifyListeners();
    }
  }
  
  // 安装更新
  Future<bool> installUpdate(File updateFile, String appDirectory, String configFilePath) async {
    _isInstalling = true;
    _updateStatus = '开始安装更新...';
    notifyListeners();
    
    try {
      final success = await _updateService.installUpdate(
        updateFile.path,
        appDirectory,
        configFilePath,
      );
      
      if (success) {
        _updateStatus = '更新安装成功！';
      } else {
        _updateStatus = '更新安装失败。';
      }
      
      return success;
    } catch (e) {
      _updateStatus = '安装更新失败：$e';
      return false;
    } finally {
      _isInstalling = false;
      _isReinstalling = false;
      notifyListeners();
    }
  }
  
  // 重装最新版本（完整流程：检查更新 -> 下载 -> 安装）
  Future<bool> reinstallLatestVersion({
    required String appDirectory,
    required String configFilePath,
    Function(String)? onStatusUpdate,
  }) async {
    _isReinstalling = true;

    try {
      // 设置本地版本为1，确保能检测到更新
      final hasUpdate = await checkForUpdates(customLocalBuildNumber: 1);

      if (!hasUpdate || _latestVersionInfo == null) {
        _updateStatus = '未获取到最新版本信息。';
        onStatusUpdate?.call(_updateStatus);
        return false;
      }

      // 下载更新包
      _updateStatus = '正在下载最新版本...';
      onStatusUpdate?.call(_updateStatus);
      notifyListeners();

      final tempDir = Directory.systemTemp;
      final assetName = _latestVersionInfo!['assetName'] as String;
      final tempFilePath = '${tempDir.path}\\$assetName';

      final updateFile = await downloadUpdate(tempFilePath);
      if (updateFile == null) {
        _updateStatus = '下载更新包失败。';
        onStatusUpdate?.call(_updateStatus);
        return false;
      }

      // 安装更新
      _updateStatus = '正在安装更新...';
      onStatusUpdate?.call(_updateStatus);
      notifyListeners();

      final success = await installUpdate(updateFile, appDirectory, configFilePath);
      return success;
    } catch (e) {
      _updateStatus = '重装最新版本失败：$e';
      onStatusUpdate?.call(_updateStatus);
      return false;
    } finally {
      _isReinstalling = false;
      notifyListeners();
    }
  }
  
  // 设置自定义更新地址
  void setCustomUpdateUrl(String url) {
    _updateService.setCustomUpdateUrl(url);
  }
}