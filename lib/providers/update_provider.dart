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
        _updateStatus = '当前已是最新版本。';
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
    notifyListeners();
    
    try {
      final downloadUrl = _latestVersionInfo!['downloadUrl'] as String;
      final file = await _updateService.downloadUpdate(downloadUrl, savePath);
      
      if (file != null) {
        _updateStatus = '更新包下载完成。';
        _downloadProgress = 1.0;
        return file;
      } else {
        _updateStatus = '下载更新包失败。';
        return null;
      }
    } catch (e) {
      _updateStatus = '下载更新包失败：$e';
      return null;
    } finally {
      _isDownloading = false;
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
  
  // 重装最新版本
  Future<bool> reinstallLatestVersion() async {
    _isReinstalling = true;
    
    try {
      // 设置本地版本为1，确保能检测到更新
      final hasUpdate = await checkForUpdates(customLocalBuildNumber: 1);
      
      if (hasUpdate && _latestVersionInfo != null) {
        // 发现新版本后，使用现有的更新流程
        _updateStatus = '开始下载最新版本...';
        notifyListeners();
        return true;
      } else {
        _updateStatus = '未获取到最新版本信息或已是最新版本。';
        return false;
      }
    } catch (e) {
      _updateStatus = '重装最新版本失败：$e';
      return false;
    } finally {
      notifyListeners();
    }
  }
  
  // 设置自定义更新地址
  void setCustomUpdateUrl(String url) {
    _updateService.setCustomUpdateUrl(url);
  }
}