import 'package:carrydock/services/settings_service.dart';
import 'package:flutter/material.dart';

/// LanguageProvider
/// 管理应用语言状态
class LanguageProvider extends ChangeNotifier {
  final SettingsService _settingsService = SettingsService();

  // 使用简单的语言代码（不带国家代码）
  Locale _currentLocale = const Locale('zh');
  bool _isInitialized = false;

  /// 当前语言环境
  Locale get currentLocale => _currentLocale;

  /// 是否已初始化
  bool get isInitialized => _isInitialized;

  /// 初始化语言设置
  Future<void> initialize() async {
    if (_isInitialized) return;

    final languageCode = await _settingsService.getLanguage();
    _currentLocale = _parseLocale(languageCode);
    _isInitialized = true;
    notifyListeners();
  }

  /// 解析语言代码为 Locale
  /// 支持新旧格式：zh_CN -> zh, en_US -> en
  Locale _parseLocale(String? languageCode) {
    if (languageCode == null || languageCode.isEmpty) {
      return const Locale('zh');
    }

    // 提取语言代码部分（忽略国家代码）
    final parts = languageCode.split('_');
    final langCode = parts[0].toLowerCase();

    // 验证是否支持该语言
    if (langCode == 'zh' || langCode == 'en') {
      return Locale(langCode);
    }

    // 默认返回中文
    return const Locale('zh');
  }

  /// 设置语言
  Future<void> setLanguage(String languageCode) async {
    final newLocale = _parseLocale(languageCode);
    if (newLocale == _currentLocale) return;

    _currentLocale = newLocale;
    await _settingsService.saveLanguage(languageCode);
    notifyListeners();
  }

  /// 获取当前语言代码（仅语言部分）
  String getCurrentLanguageCode() {
    return _currentLocale.languageCode;
  }
}
