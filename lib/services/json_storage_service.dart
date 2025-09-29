import 'dart:convert';
import 'dart:io';

import 'package:carrydock/utils/logger.dart';
import 'package:path/path.dart' as p;

/// 一个用于在应用程序可执行文件旁边读写JSON文件的服务。
///
/// 该服务将所有数据存储在 `data/app_data.json` 文件中，
/// 其中 `data` 目录与可执行文件位于同一级别。
/// JSON 存储服务（单例），负责读取/写入配置数据。
///
/// 稳健性增强：
/// - 单例：进程内只保留一个缓存，避免多实例缓存不一致。
/// - 写入加锁：使用同目录下的锁文件进行独占锁，防止并发写导致覆盖。
class JsonStorageService {
  // 单例实现，确保进程内共享一个实例与缓存。
  static final JsonStorageService _instance = JsonStorageService._internal();
  factory JsonStorageService() => _instance;
  JsonStorageService._internal();
  static const String _dataDirName = 'data';
  static const String _dbFileName = 'app_data.json';
  static const String _lockFileName = 'app_data.lock';

  File? _dbFile;
  Map<String, dynamic>? _cache;
  File? _lockFile;

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

  Future<File> get _databaseLockFile async {
    if (_lockFile != null) return _lockFile!;
    final db = await _databaseFile;
    final dir = p.dirname(db.path);
    _lockFile = File(p.join(dir, _lockFileName));
    if (!await _lockFile!.exists()) {
      await _lockFile!.create(recursive: true);
    }
    return _lockFile!;
  }

  Future<Map<String, dynamic>> _readData({bool forceReload = false}) async {
    // 当 forceReload 为 true 时，无视内存缓存，直接从磁盘读取，
    // 用于避免多实例缓存不一致导致的覆盖写入问题。
    if (_cache != null && !forceReload) return _cache!;

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

  /// 在独占文件锁下执行指定动作，避免并发写入互相覆盖。
  Future<T> _withExclusiveLock<T>(Future<T> Function() action) async {
    RandomAccessFile? raf;
    try {
      final lockFile = await _databaseLockFile;
      // 以写模式打开锁文件并加独占锁；同进程/跨进程写入会串行化。
      raf = await lockFile.open(mode: FileMode.write);
      await raf.lock(FileLock.exclusive);
      return await action();
    } finally {
      // 确保释放锁与关闭句柄，避免死锁。
      try {
        await raf?.unlock();
      } catch (_) {}
      await raf?.close();
    }
  }

  /// 从JSON文件中获取一个值。
  Future<T?> getValue<T>(String key) async {
    final data = await _readData();
    return data[key] as T?;
  }

  /// 向JSON文件中写入一个值（加锁并基于最新磁盘内容合并写入）。
  Future<void> setValue<T>(String key, T value) async {
    await _withExclusiveLock(() async {
      // 锁内再强制读盘，确保始终基于最新快照进行合并写入。
      final data = await _readData(forceReload: true);
      if (value == null) {
        data.remove(key);
      } else {
        data[key] = value;
      }
      await _writeData(data);
    });
  }

  /// 获取配置文件的绝对路径。
  Future<String> getFilePath() async {
    final file = await _databaseFile;
    return file.path;
  }
}
