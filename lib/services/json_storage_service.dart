import 'dart:convert';
import 'dart:io';

import 'package:carrydock/utils/logger.dart';
import 'package:path/path.dart' as p;

/// 一个用于在应用程序可执行文件旁边读写JSON文件的服务。
///
/// 该服务将所有数据存储在 `data/app_data.json` 文件中，
/// 其中 `data` 目录与可执行文件位于同一级别。
class JsonStorageService {
  static const String _dataDirName = 'data';
  static const String _dbFileName = 'app_data.json';

  File? _dbFile;
  Map<String, dynamic>? _cache;

  Future<File> get _databaseFile async {
    if (_dbFile != null) return _dbFile!;

    final exePath = Platform.resolvedExecutable;
    final exeDir = p.dirname(exePath);
    final dataDir = Directory(p.join(exeDir, _dataDirName));

    if (!await dataDir.exists()) {
      await dataDir.create(recursive: true);
    }

    _dbFile = File(p.join(dataDir.path, _dbFileName));
    return _dbFile!;
  }

  Future<Map<String, dynamic>> _readData() async {
    if (_cache != null) return _cache!;

    try {
      final file = await _databaseFile;
      if (await file.exists()) {
        final content = await file.readAsString();
        if (content.isNotEmpty) {
          _cache = json.decode(content) as Map<String, dynamic>;
        } else {
          _cache = {};
        }
      } else {
        _cache = {};
      }
    } catch (e, s) {
      logger.e('读取数据文件失败', error: e, stackTrace: s);
      _cache = {};
    }
    return _cache!;
  }

  Future<void> _writeData(Map<String, dynamic> data) async {
    _cache = data;
    try {
      final file = await _databaseFile;
      // 使用 `jsonEncodeWithIndent` 来格式化输出，提高可读性。
      const encoder = JsonEncoder.withIndent('  ');
      await file.writeAsString(encoder.convert(data));
    } catch (e, s) {
      logger.e('写入数据文件失败', error: e, stackTrace: s);
      rethrow;
    }
  }

  /// 从JSON文件中获取一个值。
  Future<T?> getValue<T>(String key) async {
    final data = await _readData();
    return data[key] as T?;
  }

  /// 向JSON文件中写入一个值。
  Future<void> setValue<T>(String key, T value) async {
    final data = await _readData();
    if (value == null) {
      data.remove(key);
    } else {
      data[key] = value;
    }
    await _writeData(data);
  }

  /// 获取配置文件的绝对路径。
  Future<String> getFilePath() async {
    final file = await _databaseFile;
    return file.path;
  }
}
