import 'package:carrydock/services/json_storage_service.dart';

class SettingsService {
  final JsonStorageService _storageService = JsonStorageService();

  static const String _installPathKey = 'install_path';
  static const String _archivePathKey = 'archive_path';
  static const String _fontFamilyKey = 'font_family';
  static const String _maxExecutableSearchDepthKey =
      'executable_search_max_depth';
  static const String _executableExtensionsKey = 'executable_extensions';
  static const String _developerOptionsEnabledKey = 'developer_options_enabled';
  static const String _removeNestedFoldersKey = 'remove_nested_folders';

  static const int defaultExecutableSearchMaxDepth = 3;
  static const List<String> defaultExecutableExtensions = ['exe', 'bat'];
  static const bool defaultRemoveNestedFoldersEnabled = true;

  Future<void> saveInstallPath(String path) async {
    await _storageService.setValue(_installPathKey, path);
  }

  Future<String?> getInstallPath() async {
    return await _storageService.getValue<String>(_installPathKey);
  }

  Future<void> saveArchivePath(String path) async {
    await _storageService.setValue(_archivePathKey, path);
  }

  Future<String?> getArchivePath() async {
    return await _storageService.getValue<String>(_archivePathKey);
  }

  Future<void> saveFontFamily(String fontFamily) async {
    await _storageService.setValue(_fontFamilyKey, fontFamily);
  }

  Future<String?> getFontFamily() async {
    return await _storageService.getValue<String>(_fontFamilyKey);
  }

  Future<void> saveExecutableSearchMaxDepth(int depth) async {
    final normalizedDepth = depth < 0 ? defaultExecutableSearchMaxDepth : depth;
    await _storageService.setValue(
      _maxExecutableSearchDepthKey,
      normalizedDepth,
    );
  }

  Future<int> getExecutableSearchMaxDepth() async {
    final stored = await _storageService.getValue<int>(
      _maxExecutableSearchDepthKey,
    );
    if (stored == null || stored < 0) {
      return defaultExecutableSearchMaxDepth;
    }
    return stored;
  }

  Future<void> saveExecutableExtensions(List<String> extensions) async {
    final normalized = _normalizeExtensions(extensions);
    final value = normalized.isEmpty
        ? List<String>.from(defaultExecutableExtensions)
        : normalized;
    await _storageService.setValue(_executableExtensionsKey, value);
  }

  Future<List<String>> getExecutableExtensions() async {
    final stored = await _storageService.getValue<List<dynamic>>(
      _executableExtensionsKey,
    );
    final storedStringList = stored?.map((e) => e.toString()).toList();
    final normalized = storedStringList == null
        ? List<String>.from(defaultExecutableExtensions)
        : _normalizeExtensions(storedStringList);
    return normalized.isEmpty
        ? List<String>.from(defaultExecutableExtensions)
        : normalized;
  }

  Future<void> saveDeveloperOptionsEnabled(bool enabled) async {
    await _storageService.setValue(_developerOptionsEnabledKey, enabled);
  }

  Future<bool> getDeveloperOptionsEnabled() async {
    return await _storageService.getValue<bool>(_developerOptionsEnabledKey) ??
        false;
  }

  Future<void> saveRemoveNestedFoldersEnabled(bool enabled) async {
    await _storageService.setValue(_removeNestedFoldersKey, enabled);
  }

  Future<bool> getRemoveNestedFoldersEnabled() async {
    return await _storageService.getValue<bool>(_removeNestedFoldersKey) ??
        defaultRemoveNestedFoldersEnabled;
  }

  /// 获取配置文件的绝对路径。
  Future<String> getConfigFilePath() async {
    return _storageService.getFilePath();
  }

  List<String> _normalizeExtensions(Iterable<String>? extensions) {
    final normalized = <String>{};
    if (extensions != null) {
      for (final ext in extensions) {
        final trimmed = ext.trim().toLowerCase();
        if (trimmed.isEmpty) continue;
        final cleaned = trimmed.startsWith('.')
            ? trimmed.substring(1)
            : trimmed;
        if (cleaned.isEmpty) continue;
        normalized.add(cleaned);
      }
    }
    return normalized.toList();
  }
}
