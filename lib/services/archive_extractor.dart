import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:archive/archive_io.dart';
import 'package:path/path.dart' as p;

import '../utils/logger.dart';

/// 支持的归档格式枚举，便于后续扩展。
enum ArchiveFormat { zip, tar, tarGz, tarBz2, tarXz, gz }

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
}
