import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:archive/archive_io.dart';
import 'package:path/path.dart' as p;

import '../utils/logger.dart';

/// 支持的归档格式枚举，便于后续扩展。
enum ArchiveFormat { zip, tar, tarGz, tarBz2, tarXz, gz, rar, sevenZip }

/// 负责将各类归档文件解压到指定目录，并处理编码兼容问题的工具类。
class ArchiveExtractor {
  ArchiveExtractor._();

  /// 文件选择器允许的扩展名集合。
  static const Set<String> filePickerExtensions = {
    'zip',
    'tar',
    'tgz',
    'tbz',
    'tbz2',
    'txz',
    'gz',
    'bz2',
    'xz',
    'rar',
    '7z',
  };

  /// 根据文件后缀推断归档类型，无法识别时返回 null。
  static ArchiveFormat? detectFormat(String path) {
    final lowerPath = path.toLowerCase();
    if (lowerPath.endsWith('.tar.gz') || lowerPath.endsWith('.tgz')) {
      return ArchiveFormat.tarGz;
    }
    if (lowerPath.endsWith('.tar.bz2') ||
        lowerPath.endsWith('.tbz') ||
        lowerPath.endsWith('.tbz2')) {
      return ArchiveFormat.tarBz2;
    }
    if (lowerPath.endsWith('.tar.xz') || lowerPath.endsWith('.txz')) {
      return ArchiveFormat.tarXz;
    }
    if (lowerPath.endsWith('.tar')) {
      return ArchiveFormat.tar;
    }
    if (lowerPath.endsWith('.zip')) {
      return ArchiveFormat.zip;
    }
    if (lowerPath.endsWith('.gz')) {
      return ArchiveFormat.gz;
    }
    if (lowerPath.endsWith('.rar')) {
      return ArchiveFormat.rar;
    }
    if (lowerPath.endsWith('.7z')) {
      return ArchiveFormat.sevenZip;
    }
    return null;
  }

  /// 解压入口，根据归档类型调用不同实现。
  static Future<void> extract({
    required File archiveFile,
    required Directory destination,
    required ArchiveFormat format,
  }) async {
    switch (format) {
      case ArchiveFormat.zip:
        await _extractZip(archiveFile, destination);
        break;
      case ArchiveFormat.tar:
      case ArchiveFormat.tarGz:
      case ArchiveFormat.tarBz2:
      case ArchiveFormat.tarXz:
        await _extractTarBasedArchive(archiveFile, destination, format);
        break;
      case ArchiveFormat.gz:
        await _extractGz(archiveFile, destination);
        break;
      case ArchiveFormat.rar:
        await _extractRar(archiveFile, destination);
        break;
      case ArchiveFormat.sevenZip:
        await _extractSevenZip(archiveFile, destination);
        break;
    }
  }

  static Future<void> _extractGz(
    File archiveFile,
    Directory destination,
  ) async {
    try {
      await destination.create(recursive: true);

      final decompressedFileName = p.basenameWithoutExtension(archiveFile.path);
      final targetPath = _safeJoin(destination.path, decompressedFileName);
      if (targetPath == null) {
        logger.w('检测到可能的路径穿越，条目已跳过: $decompressedFileName');
        return;
      }

      final outputStream = OutputFileStream(targetPath);
      try {
        final decompressedBytes = GZipDecoder().decodeBytes(
          await archiveFile.readAsBytes(),
        );
        outputStream.writeBytes(decompressedBytes);
      } finally {
        await outputStream.close();
      }
    } catch (e) {
      logger.e('解压 GZ 文件失败: ${archiveFile.path}', error: e);
      rethrow;
    }
  }

  static Future<void> _extractZip(
    File archiveFile,
    Directory destination,
  ) async {
    final inputStream = InputFileStream(archiveFile.path);
    try {
      final decoder = ZipDecoder();
      final archive = decoder.decodeStream(inputStream);

      await destination.create(recursive: true);

      for (var i = 0; i < archive.files.length; i++) {
        final entry = archive.files[i];
        final header = decoder.directory.fileHeaders[i];
        final resolvedName = _resolveZipEntryName(entry, header);
        if (resolvedName.isEmpty) {
          continue;
        }

        final targetPath = _safeJoin(destination.path, resolvedName);
        if (targetPath == null) {
          logger.w('检测到可能的路径穿越，条目已跳过: $resolvedName');
          continue;
        }

        if (entry.isFile) {
          await Directory(p.dirname(targetPath)).create(recursive: true);
          final outputStream = OutputFileStream(targetPath);
          try {
            entry.writeContent(outputStream);
          } finally {
            await outputStream.close();
          }
        } else {
          await Directory(targetPath).create(recursive: true);
        }
      }
    } finally {
      await inputStream.close();
    }
  }

  static Future<void> _extractTarBasedArchive(
    File archiveFile,
    Directory destination,
    ArchiveFormat format,
  ) async {
    final bytes = await archiveFile.readAsBytes();
    List<int> tarBytes;
    switch (format) {
      case ArchiveFormat.tar:
        tarBytes = bytes;
        break;
      case ArchiveFormat.tarGz:
        tarBytes = GZipDecoder().decodeBytes(bytes);
        break;
      case ArchiveFormat.tarBz2:
        tarBytes = BZip2Decoder().decodeBytes(bytes);
        break;
      case ArchiveFormat.tarXz:
        tarBytes = XZDecoder().decodeBytes(bytes);
        break;
      default:
        throw UnsupportedError('不支持的 tar 变体: $format');
    }

    final archive = TarDecoder().decodeBytes(tarBytes);
    await destination.create(recursive: true);

    for (final entry in archive.files) {
      final resolvedName = _normalizeArchivePath(entry.name);
      if (resolvedName.isEmpty) {
        continue;
      }
      final targetPath = _safeJoin(destination.path, resolvedName);
      if (targetPath == null) {
        logger.w('检测到可能的路径穿越，条目已跳过: $resolvedName');
        continue;
      }

      if (entry.isFile) {
        await Directory(p.dirname(targetPath)).create(recursive: true);
        final outputStream = OutputFileStream(targetPath);
        try {
          entry.writeContent(outputStream);
        } finally {
          await outputStream.close();
        }
      } else {
        await Directory(targetPath).create(recursive: true);
      }
    }
  }

  static String _resolveZipEntryName(ArchiveFile entry, ZipFileHeader header) {
    var name = entry.name;
    if (name.isEmpty) {
      return name;
    }

    final hasUtf8Flag = (header.generalPurposeBitFlag & 0x0800) != 0;
    if (!hasUtf8Flag) {
      final unicodeName = _readUnicodeExtraField(header.extraField);
      if (unicodeName != null && unicodeName.isNotEmpty) {
        name = unicodeName;
      } else {
        final decoded = _decodeUsingSystemEncoding(name);
        if (decoded != null && decoded.isNotEmpty) {
          name = decoded;
        }
      }
    }

    return _normalizeArchivePath(name);
  }

  static String _normalizeArchivePath(String rawName) {
    var normalized = rawName.replaceAll('\\', '/');
    normalized = normalized.replaceAll(RegExp(r'^\./+'), '');
    normalized = normalized.replaceAll(RegExp(r'/+'), '/');
    return normalized.trim();
  }

  static String? _safeJoin(String root, String relativePath) {
    final normalized = _normalizeArchivePath(relativePath);
    if (normalized.isEmpty) {
      return null;
    }

    final targetPath = p.normalize(p.join(root, normalized));
    final canonicalRoot = p.canonicalize(root);
    final canonicalTarget = p.canonicalize(targetPath);

    if (canonicalTarget == canonicalRoot) {
      return canonicalTarget;
    }

    if (!p.isWithin(canonicalRoot, canonicalTarget)) {
      return null;
    }
    return canonicalTarget;
  }

  static String? _readUnicodeExtraField(Uint8List? extraField) {
    if (extraField == null || extraField.length < 5) {
      return null;
    }

    var offset = 0;
    while (offset + 4 <= extraField.length) {
      final headerId = extraField[offset] | (extraField[offset + 1] << 8);
      final dataSize = extraField[offset + 2] | (extraField[offset + 3] << 8);
      offset += 4;
      if (offset + dataSize > extraField.length) {
        break;
      }
      if (headerId == 0x7075 && dataSize >= 5) {
        // Info-ZIP Unicode Path Extra Field
        final unicodeBytes = extraField.sublist(offset + 5, offset + dataSize);
        try {
          return utf8.decode(unicodeBytes);
        } catch (_) {
          return null;
        }
      }
      offset += dataSize;
    }

    return null;
  }

  static String? _decodeUsingSystemEncoding(String fallbackName) {
    if (fallbackName.isEmpty) {
      return null;
    }

    try {
      final encoding = systemEncoding;
      // 如果系统编码本身就是 UTF-8，则无需额外处理。
      if (encoding.name.toLowerCase() == 'utf-8') {
        return null;
      }
      final bytes = fallbackName.codeUnits.map((unit) => unit & 0xff).toList();
      return encoding.decode(bytes);
    } catch (_) {
      return null;
    }
  }

  /// 使用系统7z命令解压RAR文件
  static Future<void> _extractRar(
    File archiveFile,
    Directory destination,
  ) async {
    await _extractWith7z(archiveFile, destination, 'rar');
  }

  /// 使用系统7z命令解压7Z文件
  static Future<void> _extractSevenZip(
    File archiveFile,
    Directory destination,
  ) async {
    await _extractWith7z(archiveFile, destination, '7z');
  }

  /// 使用系统7z命令解压文件的通用方法
  static Future<void> _extractWith7z(
    File archiveFile,
    Directory destination,
    String format,
  ) async {
    try {
      await destination.create(recursive: true);

      // 自动寻找7z.exe路径
      final sevenZipPath = await find7ZipPath();
      if (sevenZipPath == null) {
        throw UnsupportedError(
          '无法解压${format.toUpperCase()}文件，未找到7-Zip安装。\n'
          '请先安装7-Zip软件（https://www.7-zip.org/）。'
        );
      }

      logger.i('找到7z.exe路径: $sevenZipPath');

      // 构建7z命令：7z x <archiveFile> -o<destination> -y
      final result = await Process.run(
        sevenZipPath,
        [
          'x',
          archiveFile.path,
          '-o${destination.path}',
          '-y', // 自动确认所有提示
        ],
        runInShell: false,
      );

      if (result.exitCode != 0) {
        logger.e('7z解压失败: ${result.stderr}');
        throw Exception('7z解压失败: ${result.stderr}');
      }

      logger.i('7z解压成功: ${archiveFile.path} -> ${destination.path}');
    } catch (e) {
      logger.e('解压${format.toUpperCase()}文件失败: ${archiveFile.path}', error: e);
      rethrow;
    }
  }

  /// 自动寻找7z.exe的路径
  static Future<String?> find7ZipPath() async {
    // 常见的7-Zip安装路径
    final commonPaths = [
      r'C:\Program Files\7-Zip\7z.exe',
      r'C:\Program Files (x86)\7-Zip\7z.exe',
      r'C:\Apps\7-Zip\7z.exe',
      r'D:\Program Files\7-Zip\7z.exe',
      r'D:\Program Files (x86)\7-Zip\7z.exe',
    ];

    // 检查常见路径
    for (final path in commonPaths) {
      if (await File(path).exists()) {
        return path;
      }
    }

    // 检查系统PATH中的7z命令
    try {
      final result = await Process.run(
        'where',
        ['7z'],
        runInShell: true,
      );
      if (result.exitCode == 0 && result.stdout.toString().trim().isNotEmpty) {
        final paths = result.stdout.toString().trim().split('\r\n');
        for (final path in paths) {
          if (await File(path).exists()) {
            return path;
          }
        }
      }
    } catch (e) {
      logger.d('通过where命令查找7z失败: $e');
    }

    // 检查注册表中的安装路径（Windows系统）
    try {
      // 尝试从注册表获取7-Zip安装路径
      final result = await Process.run(
        'reg',
        [
          'query',
          r'HKLM\SOFTWARE\7-Zip',
          '/v',
          'Path',
        ],
        runInShell: true,
      );
      if (result.exitCode == 0) {
        final output = result.stdout.toString();
        final match = RegExp(r'Path\s+REG_SZ\s+([^\r\n]+)').firstMatch(output);
        if (match != null) {
          final path = match.group(1)!.trim();
          final exePath = path.endsWith('\\') ? '${path}7z.exe' : '$path\\7z.exe';
          if (await File(exePath).exists()) {
            return exePath;
          }
        }
      }

      // 检查32位注册表项
      final result32 = await Process.run(
        'reg',
        [
          'query',
          r'HKLM\SOFTWARE\WOW6432Node\7-Zip',
          '/v',
          'Path',
        ],
        runInShell: true,
      );
      if (result32.exitCode == 0) {
        final output = result32.stdout.toString();
        final match = RegExp(r'Path\s+REG_SZ\s+([^\r\n]+)').firstMatch(output);
        if (match != null) {
          final path = match.group(1)!.trim();
          final exePath = path.endsWith('\\') ? '${path}7z.exe' : '${path}\\7z.exe';
          if (await File(exePath).exists()) {
            return exePath;
          }
        }
      }
    } catch (e) {
      logger.d('通过注册表查找7z失败: $e');
    }

    logger.d('未找到7z.exe');
    return null;
  }
}
