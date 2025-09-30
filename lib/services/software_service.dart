import 'dart:io';
import 'dart:convert';
import 'dart:math' as math;

import 'package:carrydock/models/software.dart';
import 'package:carrydock/services/archive_extractor.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/utils/error_handler.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:file_picker/file_picker.dart';
import 'package:archive/archive_io.dart';
import 'package:path/path.dart' as p;
import 'package:uuid/uuid.dart';

/// A data class to hold information for a pending software addition
/// when multiple executables are found.
class PendingSoftwareAddition {
  final String installPath;
  final String archivePath;
  final List<String> executablePaths;
  final int? preferredSortOrder;
  // 仅用于“重新托管”场景：指向需要更新的已存在软件ID。
  final String? existingSoftwareId;

  PendingSoftwareAddition({
    required this.installPath,
    required this.archivePath,
    required this.executablePaths,
    this.preferredSortOrder,
    this.existingSoftwareId,
  });
}

enum SoftwareSourceType { archive, executable }

class DuplicateSoftwareInfo {
  final SoftwareSourceType sourceType;
  final String sourcePath;
  final String installRootPath;
  final String intendedName;
  final String targetInstallPath;
  final String targetArchivePath;
  final bool installDirExists;
  final bool archiveExists;
  final Software? existingManagedSoftware;

  const DuplicateSoftwareInfo({
    required this.sourceType,
    required this.sourcePath,
    required this.installRootPath,
    required this.intendedName,
    required this.targetInstallPath,
    required this.targetArchivePath,
    required this.installDirExists,
    required this.archiveExists,
    required this.existingManagedSoftware,
  });
}

/// Enum to represent the result of the add software operation.
enum AddSoftwareResultType {
  /// The operation was successful (single executable found and processed).
  success,

  /// The user cancelled the file selection.
  cancelled,

  /// Multiple executables were found and user selection is required.
  needsSelection,

  /// A duplicate software has been detected and requires user decision.
  duplicate,

  /// An error occurred during the process.
  error,
}

/// A class to hold the result of the add software operation.
class AddSoftwareResult {
  final AddSoftwareResultType type;
  final PendingSoftwareAddition? pendingAddition;
  final DuplicateSoftwareInfo? duplicateInfo;

  AddSoftwareResult(this.type, {this.pendingAddition, this.duplicateInfo});
}

/// 批量归档失败项描述，用于在摘要中携带错误原因。
class BatchArchiveFailure {
  /// 被尝试归档的目录名（不含路径）。
  final String name;

  /// 失败原因（字符串化错误信息）。
  final String error;

  const BatchArchiveFailure({required this.name, required this.error});
}

/// 扫描阶段发现的“可关联归档”候选项。
class ArchiveAssociationSuggestion {
  /// 展示名称（通常为目录名）。
  final String displayName;

  /// 安装目录的绝对路径，用于定位软件条目。
  final String installPath;

  /// 可供关联的归档候选路径列表。
  final List<String> candidates;

  const ArchiveAssociationSuggestion({
    required this.displayName,
    required this.installPath,
    required this.candidates,
  });
}

/// 扫描完成后由用户做出的关联决策。
class ArchiveAssociationResolution {
  /// 安装目录路径（用于定位要更新的软件记录）。
  final String installPath;

  /// 选中的归档路径；若为 null 表示不关联旧归档。
  final String? selectedArchivePath;

  const ArchiveAssociationResolution({
    required this.installPath,
    this.selectedArchivePath,
  });
}

/// 批量归档结果摘要，包含计数与路径。
class BatchArchiveSummary {
  /// 扫描到的总项数（archived + skipped + failures）。
  final int total;

  /// 成功归档的目录名列表。
  final List<String> archived;

  /// 已存在同名 ZIP 而跳过的目录名列表。
  final List<String> skippedExisting;

  /// 失败列表（包含错误信息）。
  final List<BatchArchiveFailure> failures;

  /// 实际使用的归档目录路径。
  final String archiveDirPath;

  /// 实际扫描的安装根目录路径。
  final String installDirPath;

  /// 是否为覆盖模式（用于日志和提示）。
  final bool overwrite;

  /// 需要用户在扫描完成后统一处理的关联建议。
  final List<ArchiveAssociationSuggestion> suggestions;

  const BatchArchiveSummary({
    required this.total,
    required this.archived,
    required this.skippedExisting,
    required this.failures,
    required this.archiveDirPath,
    required this.installDirPath,
    required this.overwrite,
    this.suggestions = const [],
  });
}

/// 表示用户尚未配置安装路径的异常，用于提示用户前往设置界面。
class InstallPathNotConfiguredException implements Exception {
  /// 输出到日志的文本。
  final String message;

  /// 提示弹窗标题，默认统一为“需要配置安装路径”。
  final String hintTitle;

  /// 默认的提示内容，用户可在捕获时覆盖。
  final String defaultHintMessage;

  const InstallPathNotConfiguredException({
    this.message = '尚未设置安装路径，请先前往设置界面进行配置。',
    this.hintTitle = '需要配置安装路径',
    this.defaultHintMessage = '请先在设置中指定安装路径，之后即可继续该操作。',
  });

  /// 统一处理日志与提示弹窗展示。
  void notify(ErrorHandler? handler, {String? hintMessage}) {
    logger.w(message);
    handler?.showHint(hintTitle, hintMessage ?? defaultHintMessage);
  }

  @override
  String toString() => message;
}

class SoftwareService {
  static const String _defaultArchiveFolderName = '~archives';
  static const String _softwareListFileName = 'software_list.json';
  static const String _softwareListLockFileName = 'software_list.lock';
  static const String _backupSubfolder = 'backup';

  final SettingsService _settingsService = SettingsService();
  final Uuid _uuid = const Uuid();
  final ErrorHandler? errorHandler;

  SoftwareService({this.errorHandler});

  Future<String> _getPortableArchivesPath() async {
    final exePath = Platform.resolvedExecutable;
    final exeDir = p.dirname(exePath);
    return p.join(exeDir, _defaultArchiveFolderName);
  }

  Future<Directory> get _archivesDir async {
    var archivePath = await _settingsService.getArchivePath();
    if (archivePath == null || archivePath.isEmpty) {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        // 当安装路径和归档路径都未设置时，抛出异常强制用户配置
        throw const InstallPathNotConfiguredException(
          message: '尚未设置安装路径，无法确定归档目录。',
          defaultHintMessage: '请先在设置中指定安装路径，归档目录将默认创建在其下。',
        );
      }
      archivePath = p.join(installPath, _defaultArchiveFolderName);
    }

    final archivesDir = Directory(archivePath);
    if (!await archivesDir.exists()) {
      await archivesDir.create(recursive: true);
    }
    return archivesDir;
  }

  /// 对外暴露归档目录的实际路径，便于 UI 层展示确认信息。
  Future<String> resolveArchiveDirectoryPath() async {
    final dir = await _archivesDir;
    return dir.path;
  }

  /// 归档目录下的“backup”子目录。
  Future<Directory> get _backupDir async {
    final dir = await _archivesDir;
    final backup = Directory(p.join(dir.path, _backupSubfolder));
    if (!await backup.exists()) {
      await backup.create(recursive: true);
    }
    return backup;
  }

  /// 获取位于归档目录中的软件列表文件路径。
  Future<File> get _softwareListFile async {
    final dir = await _archivesDir;
    return File(p.join(dir.path, _softwareListFileName));
  }

  /// 获取软件列表锁文件路径。
  Future<File> get _softwareListLockFile async {
    final dir = await _archivesDir;
    final file = File(p.join(dir.path, _softwareListLockFileName));
    if (!await file.exists()) {
      await file.create(recursive: true);
    }
    return file;
  }

  /// 在软件列表文件上加独占锁，序列化并发读写。
  Future<T> _withSoftwareListLock<T>(Future<T> Function() action) async {
    RandomAccessFile? raf;
    try {
      final lockFile = await _softwareListLockFile;
      raf = await lockFile.open(mode: FileMode.write);
      await raf.lock(FileLock.exclusive);
      return await action();
    } finally {
      try {
        await raf?.unlock();
      } catch (_) {}
      await raf?.close();
    }
  }

  Future<List<Software>> getSoftwareList() async {
    logger.i('开始获取软件列表...');
    try {
      final List<Software> managedSoftware = await _loadManagedSoftware();

      // 同步运行时状态：归档是否存在、安装目录是否存在，但不改变托管状态
      for (final software in managedSoftware) {
        final archiveFile = File(software.archivePath);
        software.archiveExists = await archiveFile.exists();

        if (software.installPath.isNotEmpty) {
          final installDir = Directory(software.installPath);
          final installExists = await installDir.exists();
          software.installExists = installExists;
        } else {
          software.installExists = false;
        }
      }

      final installPath = await _settingsService.getInstallPath();
      final List<Software> unknownInstalls = [];
      if (installPath != null &&
          installPath.isNotEmpty &&
          await Directory(installPath).exists()) {
        final installDir = Directory(installPath);
        final entities = await installDir.list().toList();
        for (final entity in entities) {
          if (entity is Directory) {
            final archivePath = await _settingsService.getArchivePath();
            final defaultPortableArchivesPath =
                await _getPortableArchivesPath();
            final reservedArchiveDirNames = {
              _defaultArchiveFolderName,
              if (archivePath != null) p.basename(archivePath),
              p.basename(defaultPortableArchivesPath),
            };
            if (reservedArchiveDirNames.contains(p.basename(entity.path))) {
              continue;
            }
            final isManaged = managedSoftware.any(
              (s) => p.equals(s.installPath, entity.path),
            );
            if (!isManaged) {
              unknownInstalls.add(
                Software(
                  id: entity.path,
                  name: p.basename(entity.path),
                  installPath: entity.path,
                  status: SoftwareStatus.unknownInstall,
                ),
              );
            }
          }
        }
      }


      final archivesDir = await _archivesDir;
      final List<Software> unknownArchives = [];
      if (await archivesDir.exists()) {
        // 扫描归档根目录下的文件（不包含 backup 子目录，避免列表“多一个”）
        final entities = await archivesDir.list().toList();
        for (final entity in entities) {
          if (entity is File) {
            final base = p.basename(entity.path).toLowerCase();
            if (base == _softwareListFileName.toLowerCase() ||
                base == _softwareListLockFileName.toLowerCase()) {
              continue;
            }
            final isManaged = managedSoftware.any(
              (s) => p.equals(s.archivePath, entity.path),
            );
            if (!isManaged) {
              unknownArchives.add(
                Software(
                  id: entity.path,
                  name: p.basename(entity.path),
                  archivePath: entity.path,
                  archiveExists: true,
                  status: SoftwareStatus.unknownArchive,
                ),
              );
            }
          }
        }
      }

      // 标注“已托管软件”是否存在备份归档：
      // 规则：archivePath 位于 backup 子目录 或 backup 子目录存在以“sanitizedName-”开头的 zip
      try {
        final backupDir = await _backupDir;
        final backupExists = await backupDir.exists();
        final backupFiles = <String>[];
        if (backupExists) {
          await for (final entity in backupDir.list(recursive: false)) {
            if (entity is File && p.extension(entity.path).toLowerCase() == '.zip') {
              backupFiles.add(p.basename(entity.path));
            }
          }
        }
        for (final s in managedSoftware) {
          final isArchiveInBackup = s.archivePath.isNotEmpty &&
              p.basename(p.dirname(s.archivePath)) == _backupSubfolder;
          var hasBackupByName = false;
          if (backupFiles.isNotEmpty) {
            final sanitizedName = _sanitizeSoftwareName(s.name);
            final prefix = '$sanitizedName-';
            hasBackupByName = backupFiles.any((f) => f.startsWith(prefix));
          }
          s.isBackupArchive = isArchiveInBackup || hasBackupByName;
        }
      } catch (e, s) {
        logger.w('标注备份归档状态失败', error: e, stackTrace: s);
      }

      return [...managedSoftware, ...unknownInstalls, ...unknownArchives];
    } on InstallPathNotConfiguredException catch (e) {
      e.notify(errorHandler, hintMessage: '请先在设置中指定安装路径，之后即可加载软件列表。');
      return [];
    } catch (e, s) {
      logger.e('获取软件列表失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  /// 提供给“归档管理”页面使用的扫描方法：
  /// 返回归档根目录与 backup 子目录下的文件列表（不包含内部维护文件）。
  Future<List<Software>> listArchivesForManager({bool includeBackups = true}) async {
    final archives = <Software>[];
    final dir = await _archivesDir;
    if (await dir.exists()) {
      await for (final entity in dir.list(recursive: false, followLinks: false)) {
        if (entity is File) {
          final base = p.basename(entity.path).toLowerCase();
          if (base == _softwareListFileName.toLowerCase() ||
              base == _softwareListLockFileName.toLowerCase()) {
            continue;
          }
          archives.add(
            Software(
              id: entity.path,
              name: p.basename(entity.path),
              archivePath: entity.path,
              archiveExists: true,
              status: SoftwareStatus.unknownArchive,
            ),
          );
        }
      }
    }

    if (includeBackups) {
      final backup = await _backupDir;
      if (await backup.exists()) {
        await for (final entity in backup.list(recursive: false, followLinks: false)) {
          if (entity is File) {
            archives.add(
              Software(
                id: entity.path,
                name: p.basename(entity.path),
                archivePath: entity.path,
                archiveExists: true,
                status: SoftwareStatus.unknownArchive,
                isBackupArchive: true,
              ),
            );
          }
        }
      }
    }

    return archives;
  }

  /// 将指定归档（或备份）路径手动关联到某个已托管软件。
  Future<void> linkArchiveToSoftware({
    required String softwareId,
    required String archivePath,
  }) async {
    try {
      final list = await _loadManagedSoftware();
      final idx = list.indexWhere((s) => s.id == softwareId);
      if (idx < 0) {
        throw Exception('未找到指定软件');
      }
      final existing = list[idx];
      final isBackup = _isBackupFilePath(archivePath);
      list[idx] = Software(
        id: existing.id,
        name: existing.name,
        installPath: existing.installPath,
        executablePath: existing.executablePath,
        archivePath: isBackup ? existing.archivePath : archivePath,
        backupPath: isBackup ? archivePath : existing.backupPath,
        iconPath: existing.iconPath,
        archiveExists: true,
        installExists: existing.installExists,
        status: existing.status,
        sortOrder: existing.sortOrder,
      );
      await _saveSoftwareList(list);
    } catch (e, s) {
      logger.e('手动关联归档失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  /// 清除所有软件中指向指定备份路径的关联（将 backupPath 置空）。
  Future<void> clearBackupAssociationsForPath(String backupPath) async {
    try {
      final list = await _loadManagedSoftware();
      var changed = false;
      for (var i = 0; i < list.length; i++) {
        final s = list[i];
        if (s.backupPath.isNotEmpty && p.equals(s.backupPath, backupPath)) {
          list[i] = Software(
            id: s.id,
            name: s.name,
            installPath: s.installPath,
            executablePath: s.executablePath,
            archivePath: s.archivePath,
            backupPath: '',
            iconPath: s.iconPath,
            archiveExists: s.archiveExists,
            installExists: s.installExists,
            status: s.status,
            sortOrder: s.sortOrder,
          );
          changed = true;
        }
      }
      if (changed) {
        await _saveSoftwareList(list);
      }
    } catch (e, s) {
      logger.e('清除备份关联失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  /// 扫描安装目录下的一级子目录并批量压缩为 ZIP 文件存入归档目录。
  ///
  /// 约定与处理规则：
  /// - 仅处理安装根目录下的直接子文件夹；
  /// - 跳过保留目录（如 `~archives` 以及配置的归档目录名等）；
  /// - 若目标 ZIP 已存在且 `overwrite=false`，则跳过；
  /// - 使用 `archive` 库的 `ZipFileEncoder` 进行压缩，包含顶层目录名；
  /// - 返回压缩结果摘要，供 UI 展示。
  Future<BatchArchiveSummary> archiveInstallSubdirectories({
    bool overwrite = false,
    void Function(int done, int total)? onProgress,
    void Function(String currentName)? onCurrentItem,
    bool manageRecognized = true,
    bool createBackup = true,
  }) async {
    logger.i('开始扫描安装目录并批量归档...');
    final processed = <String>[];
    final skipped = <String>[];
    final failed = <BatchArchiveFailure>[];
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        throw const InstallPathNotConfiguredException();
      }

      final installDir = Directory(installPath);
      if (!await installDir.exists()) {
        throw Exception('安装目录不存在：$installPath');
      }

      final backupDir = await _backupDir;

      // 需要跳过的保留目录名集合
      final archivePath = await _settingsService.getArchivePath();
      final defaultPortableArchivesPath = await _getPortableArchivesPath();
      final reservedArchiveDirNames = <String>{
        _defaultArchiveFolderName,
        if (archivePath != null && archivePath.isNotEmpty)
          p.basename(archivePath),
        p.basename(defaultPortableArchivesPath),
        _backupSubfolder,
      };

      final entities = await installDir.list(followLinks: false).toList();
      final dirEntities = <Directory>[];
      for (final entity in entities) {
        if (entity is Directory) {
          dirEntities.add(entity);
        }
      }

      final total = dirEntities.length;
      onProgress?.call(0, total);

      // 预加载已托管列表，便于在扫描过程中维护 JSON
      final managedList = await _loadManagedSoftware();
      int nextOrder = managedList.isEmpty
          ? 0
          : managedList.map((s) => s.sortOrder).reduce(math.max) + 1;

      final suggestions = <ArchiveAssociationSuggestion>[];
      for (var i = 0; i < dirEntities.length; i++) {
        final entity = dirEntities[i];
        final baseName = p.basename(entity.path);
        onCurrentItem?.call(baseName);
        if (reservedArchiveDirNames.contains(baseName)) {
          logger.d('跳过保留目录: $baseName');
          onProgress?.call(i + 1, total);
          continue;
        }

        try {
          String? createdPath;
          // 先收集关联候选，统一在扫描完成后由 UI 决定是否关联
          final sanitized = _sanitizeSoftwareName(baseName);
          final candidates = <String>[];
          final archivesDir = await _archivesDir;
          if (await archivesDir.exists()) {
            await for (final e in archivesDir.list(recursive: false, followLinks: false)) {
              if (e is File) {
                final fmt = ArchiveExtractor.detectFormat(e.path);
                if (fmt == null) continue;
                // 仅归档根目录下的文件参与关联，备份目录（backup）不参与
                final parentBase = p.basename(p.dirname(e.path));
                if (parentBase == _backupSubfolder) continue;
                final bn = p.basenameWithoutExtension(e.path);
                if (bn == sanitized || bn.startsWith('$sanitized-')) {
                  candidates.add(e.path);
                }
              }
            }
          }
          // 过滤掉已被托管软件占用的归档
          candidates.removeWhere((path) =>
              managedList.any((s) => s.archivePath.isNotEmpty && p.equals(s.archivePath, path)));
          if (candidates.isNotEmpty) {
            suggestions.add(ArchiveAssociationSuggestion(
              displayName: baseName,
              installPath: entity.path,
              candidates: candidates,
            ));
          } else if (createBackup) {
            // 若没有候选且允许创建备份，立即创建
            createdPath = await createBackupFromDirectory(
              sourceDir: entity,
              displayName: baseName,
            );
          }
          processed.add(baseName);
          if (manageRecognized) {
            // 查找是否已有同安装目录的软件
            final idx = managedList.indexWhere(
              (s) => p.equals(s.installPath, entity.path),
            );
            if (idx >= 0) {
              final existing = managedList[idx];
              var newExecutable = existing.executablePath;
              if (newExecutable.isEmpty) {
                final execs = await findExecutablesInDirectory(entity);
                if (execs.isNotEmpty) {
                  newExecutable = _pickShallowestExecutablePath(execs, entity);
                }
              }
              managedList[idx] = Software(
                id: existing.id,
                name: existing.name,
                installPath: existing.installPath,
                executablePath: newExecutable,
                archivePath: createdPath ?? existing.archivePath,
                iconPath: existing.iconPath,
                archiveExists: createdPath != null ? true : existing.archiveExists,
                installExists: existing.installExists,
                status: existing.status,
                sortOrder: existing.sortOrder,
              );
            } else {
              final execs = await findExecutablesInDirectory(entity);
              String exe = '';
              if (execs.isNotEmpty) {
                exe = _pickShallowestExecutablePath(execs, entity);
              }
              final software = Software(
                id: _uuid.v4(),
                name: _sanitizeSoftwareName(baseName),
                installPath: entity.path,
                executablePath: exe,
                archivePath: createdPath ?? '',
                status: SoftwareStatus.managed,
                sortOrder: nextOrder++,
              );
              managedList.add(software);
            }
          }
          if (createBackup && createdPath != null) {
            logger.i('已归档(backup): ${entity.path} -> $createdPath');
          } else {
            logger.i('已识别(未归档): ${entity.path}');
          }
        } catch (e, s) {
          logger.e('归档失败: ${entity.path}', error: e, stackTrace: s);
          failed.add(BatchArchiveFailure(name: baseName, error: e.toString()));
        }
        onProgress?.call(i + 1, total);
      }

      if (manageRecognized) {
        await _saveSoftwareList(managedList);
      }

      return BatchArchiveSummary(
        total: processed.length + skipped.length + failed.length,
        archived: processed,
        skippedExisting: skipped,
        failures: failed,
        archiveDirPath: backupDir.path,
        installDirPath: installDir.path,
        overwrite: overwrite,
        suggestions: suggestions,
      );
    } on InstallPathNotConfiguredException catch (e) {
      e.notify(errorHandler, hintMessage: '请先在设置中指定安装路径，之后即可执行扫描归档。');
      rethrow;
    } catch (e, s) {
      logger.e('批量归档过程中发生错误', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  /// 为指定的软件创建一次“备份归档”，压缩其安装目录到归档目录的 backup 子目录。
  /// 返回创建的 ZIP 文件路径。
  Future<String> createBackupForSoftware(Software software) async {
    if (software.installPath.isEmpty) {
      throw Exception('该软件未配置安装目录，无法创建备份。');
    }
    final dir = Directory(software.installPath);
    if (!await dir.exists()) {
      throw Exception('安装目录不存在：${software.installPath}');
    }
    return createBackupFromDirectory(
      sourceDir: dir,
      displayName: software.name,
    );
  }

  /// 将任意目录压缩为 ZIP 到 backup 子目录，文件名包含日期时间。
  Future<String> createBackupFromDirectory({
    required Directory sourceDir,
    String? displayName,
  }) async {
    final backupDir = await _backupDir;
    final baseName = displayName ?? p.basename(sourceDir.path);
    final sanitized = _sanitizeSoftwareName(baseName);
    final timestamp = _formatNowForFilename();
    final filename = '$sanitized-$timestamp.zip';
    final targetPath = p.join(backupDir.path, filename);

    final encoder = ZipFileEncoder();
    encoder.create(targetPath);
    // 必须等待目录添加完成后再关闭，否则在 Windows 上可能出现文件被关闭后仍写入导致的“拒绝访问”错误。
    await encoder.addDirectory(sourceDir, includeDirName: true);
    encoder.close();
    return targetPath;
  }

  /// 以 yyyyMMdd_HHmmss 生成时间片段，用于文件名。
  String _formatNowForFilename() {
    final now = DateTime.now();
    String two(int n) => n.toString().padLeft(2, '0');
    final y = now.year.toString().padLeft(4, '0');
    final m = two(now.month);
    final d = two(now.day);
    final hh = two(now.hour);
    final mm = two(now.minute);
    final ss = two(now.second);
    return '$y$m${d}_$hh$mm$ss';
  }

  bool _isBackupFilePath(String path) {
    try {
      return p.basename(p.dirname(path)) == _backupSubfolder;
    } catch (_) {
      return false;
    }
  }

  /// 去除形如 `-YYYYMMDD_HHMMSS` 的结尾时间后缀。
  String _stripBackupTimestampFromName(String name) {
    final reg = RegExp(r'-\d{8}_\d{6}$');
    return name.replaceFirst(reg, '');
  }

  /// 应用用户对“直接关联归档”的决策：
  /// - 对于有 selectedArchivePath 的项，直接更新对应软件的 archivePath；
  /// - 对于未选择关联且允许创建备份的项，为其创建备份并更新；
  /// - 若软件主程序为空，自动选择相对目录更浅的可执行文件。
  Future<void> applyArchiveAssociations({
    required List<ArchiveAssociationResolution> resolutions,
    required bool createBackupForUnselected,
  }) async {
    if (resolutions.isEmpty) return;
    try {
      final managedList = await _loadManagedSoftware();
      var updatedAny = false;
      for (final r in resolutions) {
        final idx = managedList.indexWhere((s) => p.equals(s.installPath, r.installPath));
        if (idx < 0) {
          // 兜底：若未找到，尝试新增记录
          final dir = Directory(r.installPath);
          if (!await dir.exists()) continue;
          final execs = await findExecutablesInDirectory(dir);
          String exe = '';
          if (execs.isNotEmpty) {
            exe = _pickShallowestExecutablePath(execs, dir);
          }
          String archivePath = '';
          if (r.selectedArchivePath != null && r.selectedArchivePath!.isNotEmpty) {
            archivePath = r.selectedArchivePath!;
          } else if (createBackupForUnselected) {
            archivePath = await createBackupFromDirectory(
              sourceDir: dir,
              displayName: p.basename(r.installPath),
            );
          }
          managedList.add(Software(
            id: _uuid.v4(),
            name: _sanitizeSoftwareName(p.basename(r.installPath)),
            installPath: r.installPath,
            executablePath: exe,
            archivePath: archivePath,
            status: SoftwareStatus.managed,
            sortOrder: _nextSortOrder(managedList),
          ));
          updatedAny = true;
          continue;
        }

        final existing = managedList[idx];
        String archivePath = existing.archivePath;
        if (r.selectedArchivePath != null && r.selectedArchivePath!.isNotEmpty) {
          archivePath = r.selectedArchivePath!;
        } else if (createBackupForUnselected && (archivePath.isEmpty || !await File(archivePath).exists())) {
          final dir = Directory(existing.installPath);
          if (await dir.exists()) {
            archivePath = await createBackupFromDirectory(
              sourceDir: dir,
              displayName: existing.name,
            );
          }
        }

        var exe = existing.executablePath;
        if (exe.isEmpty) {
          final dir = Directory(existing.installPath);
          if (await dir.exists()) {
            final execs = await findExecutablesInDirectory(dir);
            if (execs.isNotEmpty) {
              exe = _pickShallowestExecutablePath(execs, dir);
            }
          }
        }

        managedList[idx] = Software(
          id: existing.id,
          name: existing.name,
          installPath: existing.installPath,
          executablePath: exe,
          archivePath: archivePath,
          iconPath: existing.iconPath,
          archiveExists: archivePath.isNotEmpty,
          installExists: existing.installExists,
          status: existing.status,
          sortOrder: existing.sortOrder,
        );
        updatedAny = true;
      }

      if (updatedAny) {
        await _saveSoftwareList(managedList);
      }
    } catch (e, s) {
      logger.e('应用归档关联决策失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  /// 在给定的可执行文件列表中，选择相对 root 目录层级最浅的路径。
  /// 若存在并列，取字典序较小者，保证稳定性。
  String _pickShallowestExecutablePath(List<String> executables, Directory root) {
    if (executables.isEmpty) return '';
    String best = executables.first;
    int bestDepth = _relativeDepth(best, root.path);
    for (var i = 1; i < executables.length; i++) {
      final path = executables[i];
      final depth = _relativeDepth(path, root.path);
      if (depth < bestDepth || (depth == bestDepth && path.compareTo(best) < 0)) {
        best = path;
        bestDepth = depth;
      }
    }
    return best;
  }

  int _relativeDepth(String path, String root) {
    try {
      final rel = p.relative(path, from: root);
      final parts = p.split(rel);
      return parts.length;
    } catch (_) {
      // 回退：用分隔符计数
      final normalized = p.normalize(path).replaceAll('\\\\', '/');
      return '/'.allMatches(normalized).length;
    }
  }

  Future<void> _saveSoftwareList(List<Software> softwareList) async {
    final managedSoftware = softwareList
        .where((s) => s.status == SoftwareStatus.managed)
        .toList()
      ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
    for (var i = 0; i < managedSoftware.length; i++) {
      managedSoftware[i].sortOrder = i;
    }
    try {
      final file = await _softwareListFile;
      final data = managedSoftware.map((s) => s.toJson()).toList();
      await _withSoftwareListLock(() async {
        const encoder = JsonEncoder.withIndent('  ');
        await file.writeAsString(encoder.convert(data));
      });
    } catch (e, s) {
      logger.e('保存软件列表失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  int _nextSortOrder(List<Software> softwareList) {
    final managedSoftware = softwareList
        .where((s) => s.status == SoftwareStatus.managed)
        .toList();
    if (managedSoftware.isEmpty) {
      return 0;
    }
    final maxOrder = managedSoftware
        .map((software) => software.sortOrder)
        .reduce(math.max);
    return maxOrder + 1;
  }

  /// 自动识别文件类型并完成软件添加。
  Future<AddSoftwareResult> addSoftware() async {
    logger.i('开始自动识别添加新软件...');
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        throw const InstallPathNotConfiguredException();
      }

      final executableExtensions = await _settingsService
          .getExecutableExtensions();
      final normalizedExecutableExtensions = executableExtensions
          .map((ext) => ext.toLowerCase())
          .toSet();
      final pickerExtensions = <String>{
        ...ArchiveExtractor.filePickerExtensions,
        ...normalizedExecutableExtensions,
      };

      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: pickerExtensions.toList(),
      );
      if (result == null) {
        logger.i('用户取消选择文件，操作中止');
        return AddSoftwareResult(AddSoftwareResultType.cancelled);
      }

      final selectedPath = result.files.single.path!;
      final selectedFile = File(selectedPath);
      if (!await selectedFile.exists()) {
        throw Exception('选择的文件不存在。');
      }

      final archiveFormat = ArchiveExtractor.detectFormat(selectedPath);
      if (archiveFormat != null) {
        return await _addSoftwareFromArchiveFile(
          archiveFile: selectedFile,
          installPath: installPath,
          format: archiveFormat,
        );
      }

      final extension = p.extension(selectedPath).toLowerCase();
      final normalizedExtension = extension.startsWith('.')
          ? extension.substring(1)
          : extension;

      if (normalizedExecutableExtensions.contains(normalizedExtension)) {
        return await _addSoftwareFromExecutableFile(
          executableFile: selectedFile,
          installPath: installPath,
        );
      }

      throw Exception('不支持的文件类型: ${p.extension(selectedPath)}');
    } on InstallPathNotConfiguredException catch (e) {
      e.notify(errorHandler, hintMessage: '请先在设置中指定安装路径，之后即可添加新软件。');
      return AddSoftwareResult(AddSoftwareResultType.error);
    } catch (e, s) {
      logger.e('自动识别添加软件失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      return AddSoftwareResult(AddSoftwareResultType.error);
    }
  }

  /// 基于指定文件路径添加软件，常用于拖拽文件场景。
  Future<AddSoftwareResult> addSoftwareFromFile(String filePath) async {
    logger.i('通过外部文件添加软件: $filePath');
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        throw const InstallPathNotConfiguredException();
      }

      final targetFile = File(filePath);
      if (!await targetFile.exists()) {
        throw Exception('提供的文件不存在: $filePath');
      }

      final archiveFormat = ArchiveExtractor.detectFormat(filePath);
      if (archiveFormat != null) {
        return await _addSoftwareFromArchiveFile(
          archiveFile: targetFile,
          installPath: installPath,
          format: archiveFormat,
        );
      }

      final extension = p.extension(filePath).toLowerCase();
      final normalizedExtension = extension.startsWith('.')
          ? extension.substring(1)
          : extension;

      final executableExtensions = await _settingsService
          .getExecutableExtensions();
      final normalizedExecutableExtensions = executableExtensions
          .map((ext) => ext.toLowerCase())
          .toSet();

      if (normalizedExecutableExtensions.contains(normalizedExtension)) {
        return await _addSoftwareFromExecutableFile(
          executableFile: targetFile,
          installPath: installPath,
        );
      }

      throw Exception('不支持的文件类型: ${p.extension(filePath)}');
    } on InstallPathNotConfiguredException catch (e) {
      e.notify(errorHandler, hintMessage: '请先在设置中指定安装路径，之后即可添加新软件。');
      return AddSoftwareResult(AddSoftwareResultType.error);
    } catch (e, s) {
      logger.e('通过文件添加软件失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      return AddSoftwareResult(AddSoftwareResultType.error);
    }
  }

  /// 从压缩包开始添加一个新软件的流程。
  ///
  /// 返回一个 [AddSoftwareResult] 对象，指明操作的结果。
  Future<AddSoftwareResult> addSoftwareFromArchive() async {
    logger.i('开始从压缩包添加新软件...');
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        throw const InstallPathNotConfiguredException();
      }

      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: ArchiveExtractor.filePickerExtensions.toList(),
      );
      if (result == null) {
        logger.i('用户取消选择文件，操作中止');
        return AddSoftwareResult(AddSoftwareResultType.cancelled);
      }

      final archiveFile = File(result.files.single.path!);
      if (!await archiveFile.exists()) {
        throw Exception('选择的压缩包不存在。');
      }

      final archiveFormat = ArchiveExtractor.detectFormat(archiveFile.path);
      if (archiveFormat == null) {
        throw Exception('不支持的压缩格式: ${p.extension(archiveFile.path)}');
      }

      return await _addSoftwareFromArchiveFile(
        archiveFile: archiveFile,
        installPath: installPath,
        format: archiveFormat,
      );
    } on InstallPathNotConfiguredException catch (e) {
      e.notify(errorHandler, hintMessage: '请先在设置中指定安装路径，之后即可从压缩包添加软件。');
      return AddSoftwareResult(AddSoftwareResultType.error);
    } catch (e, s) {
      logger.e('添加软件失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      return AddSoftwareResult(AddSoftwareResultType.error);
    }
  }

  /// 从单个可执行文件开始添加一个新软件的流程。
  Future<AddSoftwareResult> addSoftwareFromExecutable() async {
    logger.i('开始从可执行文件添加新软件...');
    try {
      final installPath = await _settingsService.getInstallPath();
      if (installPath == null || installPath.isEmpty) {
        throw const InstallPathNotConfiguredException();
      }

      final allowedExtensions = await _settingsService
          .getExecutableExtensions();
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: allowedExtensions,
      );
      if (result == null) {
        logger.i('用户取消选择可执行文件，操作中止');
        return AddSoftwareResult(AddSoftwareResultType.cancelled);
      }

      final executableFile = File(result.files.single.path!);
      if (!await executableFile.exists()) {
        throw Exception('选择的可执行文件不存在。');
      }

      return await _addSoftwareFromExecutableFile(
        executableFile: executableFile,
        installPath: installPath,
      );
    } on InstallPathNotConfiguredException catch (e) {
      e.notify(errorHandler, hintMessage: '请先在设置中指定安装路径，之后即可从可执行文件添加软件。');
      return AddSoftwareResult(AddSoftwareResultType.error);
    } catch (e, s) {
      logger.e('从可执行文件添加软件失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      return AddSoftwareResult(AddSoftwareResultType.error);
    }
  }

  Future<AddSoftwareResult> _addSoftwareFromArchiveFile({
    required File archiveFile,
    required String installPath,
    required ArchiveFormat format,
    bool allowOverride = false,
    String? customSoftwareName,
  }) async {
    String? archivedPath;
    Directory? destinationDir;
    bool _sameSourceAsDest = false;

    try {
      final baseName = p.basenameWithoutExtension(archiveFile.path);
      String fallbackName = baseName.isEmpty
          ? p.basename(archiveFile.path)
          : baseName;
      // 备份文件：移除结尾的时间后缀
      if (customSoftwareName == null && _isBackupFilePath(archiveFile.path)) {
        fallbackName = _stripBackupTimestampFromName(fallbackName);
      }
      final softwareName = _sanitizeSoftwareName(
        customSoftwareName ?? fallbackName,
      );

      final archivesDir = await _archivesDir;
      final archiveFileName = _buildArchiveFileName(
        sanitizedName: softwareName,
        sourceFileName: p.basename(archiveFile.path),
      );
      archivedPath = p.join(archivesDir.path, archiveFileName);
      destinationDir = Directory(p.join(installPath, softwareName));

      final archiveDestFile = File(archivedPath);
      final bool installExists = await destinationDir.exists();
      final bool archiveExists = await archiveDestFile.exists();
      final existingManagedSoftware = await _findManagedSoftwareByPaths(
        destinationDir.path,
        archivedPath,
      );

      if (!allowOverride &&
          (installExists || archiveExists || existingManagedSoftware != null)) {
        return AddSoftwareResult(
          AddSoftwareResultType.duplicate,
          duplicateInfo: DuplicateSoftwareInfo(
            sourceType: SoftwareSourceType.archive,
            sourcePath: archiveFile.path,
            installRootPath: installPath,
            intendedName: softwareName,
            targetInstallPath: destinationDir.path,
            targetArchivePath: archivedPath,
            installDirExists: installExists,
            archiveExists: archiveExists,
            existingManagedSoftware: existingManagedSoftware,
          ),
        );
      }

      int? preferredSortOrder;
      _sameSourceAsDest = p.equals(archiveFile.path, archivedPath);
      if (allowOverride) {
        preferredSortOrder = await _prepareForOverride(
          installPath: destinationDir.path,
          // 当源文件即目标文件时，避免在覆盖准备阶段删除源文件
          archivePath: _sameSourceAsDest ? '' : archivedPath,
        );
      }

      final File storedArchiveFile;
      if (_sameSourceAsDest) {
        storedArchiveFile = archiveFile;
      } else {
        storedArchiveFile = await archiveFile.copy(archivedPath);
      }
      await destinationDir.create(recursive: true);

      logger.i(
        '开始解压归档文件(${format.name}): ${storedArchiveFile.path} -> ${destinationDir.path}',
      );
      await ArchiveExtractor.extract(
        archiveFile: storedArchiveFile,
        destination: destinationDir,
        format: format,
      );

      final removeNestedFolders = await _settingsService
          .getRemoveNestedFoldersEnabled();
      if (removeNestedFolders) {
        final flattened = await _flattenRedundantTopDirectory(destinationDir);
        if (flattened) {
          logger.i('已自动展开嵌套目录: ${destinationDir.path}');
        }
      }

      final executables = await findExecutablesInDirectory(destinationDir);

      if (executables.isEmpty) {
        throw Exception('在压缩包中未找到可执行文件。');
      }

      if (executables.length == 1) {
      await completeSoftwareAddition(
        installPath: destinationDir.path,
        archivePath: archivedPath,
        selectedExecutablePath: executables.first,
        preferredSortOrder: preferredSortOrder,
      );
        return AddSoftwareResult(AddSoftwareResultType.success);
      }

      return AddSoftwareResult(
        AddSoftwareResultType.needsSelection,
        pendingAddition: PendingSoftwareAddition(
          installPath: destinationDir.path,
          archivePath: archivedPath,
          executablePaths: executables,
          preferredSortOrder: preferredSortOrder,
        ),
      );
    } catch (e) {
      await cleanupTemporaryFiles(
        installPath: destinationDir?.path,
        archivePath: _sameSourceAsDest ? null : archivedPath,
      );
      rethrow;
    }
  }

  Future<AddSoftwareResult> _addSoftwareFromExecutableFile({
    required File executableFile,
    required String installPath,
    bool allowOverride = false,
    String? customSoftwareName,
  }) async {
    String? archivedPath;
    Directory? destinationDir;
    bool _sameSourceAsDest = false;

    try {
      final baseName = p.basenameWithoutExtension(executableFile.path);
      final fallbackName = baseName.isEmpty
          ? p.basename(executableFile.path)
          : baseName;
      final softwareName = _sanitizeSoftwareName(
        customSoftwareName ?? fallbackName,
      );

      final archivesDir = await _archivesDir;
      final archiveFileName = _buildArchiveFileName(
        sanitizedName: softwareName,
        sourceFileName: p.basename(executableFile.path),
      );
      archivedPath = p.join(archivesDir.path, archiveFileName);
      destinationDir = Directory(p.join(installPath, softwareName));

      final archiveDestFile = File(archivedPath);
      final bool installExists = await destinationDir.exists();
      final bool archiveExists = await archiveDestFile.exists();
      final existingManagedSoftware = await _findManagedSoftwareByPaths(
        destinationDir.path,
        archivedPath,
      );

      if (!allowOverride &&
          (installExists || archiveExists || existingManagedSoftware != null)) {
        return AddSoftwareResult(
          AddSoftwareResultType.duplicate,
          duplicateInfo: DuplicateSoftwareInfo(
            sourceType: SoftwareSourceType.executable,
            sourcePath: executableFile.path,
            installRootPath: installPath,
            intendedName: softwareName,
            targetInstallPath: destinationDir.path,
            targetArchivePath: archivedPath,
            installDirExists: installExists,
            archiveExists: archiveExists,
            existingManagedSoftware: existingManagedSoftware,
          ),
        );
      }

      int? preferredSortOrder;
      _sameSourceAsDest = p.equals(executableFile.path, archivedPath);
      if (allowOverride) {
        preferredSortOrder = await _prepareForOverride(
          installPath: destinationDir.path,
          archivePath: _sameSourceAsDest ? '' : archivedPath,
        );
      }

      if (!_sameSourceAsDest) {
        await executableFile.copy(archivedPath);
      }
      await destinationDir.create(recursive: true);

      final executableDestinationPath = p.join(
        destinationDir.path,
        p.basename(executableFile.path),
      );
      await executableFile.copy(executableDestinationPath);

      await completeSoftwareAddition(
        installPath: destinationDir.path,
        archivePath: archivedPath,
        selectedExecutablePath: executableDestinationPath,
        preferredSortOrder: preferredSortOrder,
      );

      return AddSoftwareResult(AddSoftwareResultType.success);
    } catch (e) {
      await cleanupTemporaryFiles(
        installPath: destinationDir?.path,
        archivePath: _sameSourceAsDest ? null : archivedPath,
      );
      rethrow;
    }
  }

  /// 重新托管：当安装目录丢失但归档仍存在时，根据归档重新解压并恢复托管状态。
  ///
  /// - 若找到多个可执行程序，返回 needsSelection 并携带可选路径与上下文；
  /// - 若找到一个可执行程序，直接更新现有记录并返回 success；
  /// - 若归档缺失或无可执行程序，返回 error。
  Future<AddSoftwareResult> rehostSoftware(Software software) async {
    try {
      // 确定归档文件：优先使用记录路径；若不存在，尝试在归档目录中按文件名或软件名匹配；仍未找到则直接报错。
      File? archiveFile;
      if (software.archivePath.isNotEmpty) {
        final recorded = File(software.archivePath);
        if (await recorded.exists()) {
          archiveFile = recorded;
        }
      }

      if (archiveFile == null) {
        final archivesDir = await _archivesDir;
        if (await archivesDir.exists()) {
          final expectedName = software.archivePath.isNotEmpty
              ? p.basename(software.archivePath)
              : '';
          final candidates = <File>[];
          await for (final entity in archivesDir.list()) {
            if (entity is File) {
              final base = p.basename(entity.path);
              if (expectedName.isNotEmpty && base == expectedName) {
                candidates.add(entity);
                break;
              }
              // 回退：按软件名前缀尝试匹配
              if (expectedName.isEmpty && base.startsWith(software.name)) {
                candidates.add(entity);
              }
            }
          }
          if (candidates.length == 1) {
            archiveFile = candidates.first;
          }
        }
      }

      if (archiveFile == null) {
        throw Exception('归档文件不存在，无法重新托管。');
      }

      // 识别归档或可执行文件
      final format = ArchiveExtractor.detectFormat(archiveFile.path);
      final ext = p.extension(archiveFile.path).toLowerCase();
      final normalizedExt = ext.startsWith('.') ? ext.substring(1) : ext;
      final allowedExecExts = (await _settingsService.getExecutableExtensions())
          .map((e) => e.toLowerCase())
          .toSet();

      if (software.installPath.isEmpty) {
        throw Exception('安装路径未知，无法重新托管。');
      }
      final destinationDir = Directory(software.installPath);
      await destinationDir.create(recursive: true);

      if (format != null) {
        // 归档：解压 + 可选扁平化 + 扫描可执行文件
        logger.i('开始重新托管(归档): ${archiveFile.path} -> ${destinationDir.path}');
        await ArchiveExtractor.extract(
          archiveFile: archiveFile,
          destination: destinationDir,
          format: format,
        );

        final removeNested = await _settingsService.getRemoveNestedFoldersEnabled();
        if (removeNested) {
          final flattened = await _flattenRedundantTopDirectory(destinationDir);
          if (flattened) {
            logger.i('已自动展开嵌套目录: ${destinationDir.path}');
          }
        }

        final executables = await findExecutablesInDirectory(destinationDir);
        if (executables.isEmpty) {
          throw Exception('未能在归档内容中发现可执行程序。');
        }

        if (executables.length == 1) {
          await completeSoftwareRehost(
            existingId: software.id,
            installPath: destinationDir.path,
            archivePath: archiveFile.path,
            selectedExecutablePath: executables.first,
          );
          return AddSoftwareResult(AddSoftwareResultType.success);
        }

        return AddSoftwareResult(
          AddSoftwareResultType.needsSelection,
          pendingAddition: PendingSoftwareAddition(
            installPath: destinationDir.path,
            archivePath: archiveFile.path,
            executablePaths: executables,
            preferredSortOrder: null,
            existingSoftwareId: software.id,
          ),
        );
      } else if (allowedExecExts.contains(normalizedExt)) {
        // 可执行文件：直接复制到安装目录并作为主程序
        logger.i('开始重新托管(可执行文件): ${archiveFile.path} -> ${destinationDir.path}');
        final destPath = p.join(destinationDir.path, p.basename(archiveFile.path));
        await archiveFile.copy(destPath);
        await completeSoftwareRehost(
          existingId: software.id,
          installPath: destinationDir.path,
          archivePath: archiveFile.path,
          selectedExecutablePath: destPath,
        );
        return AddSoftwareResult(AddSoftwareResultType.success);
      } else {
        throw Exception('不支持的归档格式：${p.extension(archiveFile.path)}');
      }
    } catch (e, s) {
      logger.e('重新托管失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      return AddSoftwareResult(AddSoftwareResultType.error);
    }
  }

  /// 在用户选择主程序后，完成重新托管：更新现有记录为 managed 并保存。
  Future<void> completeSoftwareRehost({
    required String existingId,
    required String installPath,
    required String archivePath,
    required String selectedExecutablePath,
  }) async {
    logger.i('完成重新托管: $installPath');
    try {
      final softwareList = await _loadManagedSoftware();
      final idx = softwareList.indexWhere((s) => s.id == existingId);
      if (idx < 0) {
        throw Exception('找不到需要更新的软件记录 (ID: $existingId)');
      }
      final existing = softwareList[idx];
      final isBackup = _isBackupFilePath(archivePath);
      softwareList[idx] = Software(
        id: existing.id,
        name: existing.name,
        installPath: installPath,
        executablePath: selectedExecutablePath,
        archivePath: archivePath,
        backupPath: isBackup ? archivePath : existing.backupPath,
        iconPath: existing.iconPath,
        archiveExists: true,
        installExists: true,
        status: SoftwareStatus.managed,
        sortOrder: existing.sortOrder,
      );
      await _saveSoftwareList(softwareList);
    } catch (e, s) {
      logger.e('完成重新托管失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }
  /// 在用户选择主程序后，完成软件的添加。
  Future<void> completeSoftwareAddition({
    required String installPath,
    required String archivePath,
    required String selectedExecutablePath,
    int? preferredSortOrder,
  }) async {
    logger.i('完成软件添加: $installPath');
    try {
      final softwareName = p.basename(installPath);
      final softwareList = await getSoftwareList();
      if (preferredSortOrder != null) {
        for (final software in softwareList) {
          if (software.status == SoftwareStatus.managed &&
              software.sortOrder >= preferredSortOrder) {
            software.sortOrder += 1;
          }
        }
      }
      final isBackup = _isBackupFilePath(archivePath);
      final newSoftware = Software(
        id: _uuid.v4(),
        name: softwareName,
        installPath: installPath,
        executablePath: selectedExecutablePath,
        archivePath: archivePath,
        backupPath: isBackup ? archivePath : '',
        status: SoftwareStatus.managed,
        sortOrder: preferredSortOrder ?? _nextSortOrder(softwareList),
      );

      softwareList.add(newSoftware);
      await _saveSoftwareList(softwareList);
      logger.i('新软件 "$softwareName" 添加成功');
    } catch (e, s) {
      logger.e('完成软件添加失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  Future<AddSoftwareResult> resolveDuplicateAddition({
    required DuplicateSoftwareInfo info,
    bool overwriteExisting = false,
    String? renameTo,
  }) async {
    final trimmedRename = renameTo?.trim();

    if (!overwriteExisting &&
        (trimmedRename == null || trimmedRename.isEmpty)) {
      throw ArgumentError('未提供有效的重复处理指令');
    }

    switch (info.sourceType) {
      case SoftwareSourceType.archive:
        final archiveFile = File(info.sourcePath);
        if (!await archiveFile.exists()) {
          throw Exception('原始压缩包已不存在，请重新选择文件。');
        }
        final archiveFormat = ArchiveExtractor.detectFormat(archiveFile.path);
        if (archiveFormat == null) {
          throw Exception('不支持的压缩格式: ${p.extension(archiveFile.path)}');
        }
        return _addSoftwareFromArchiveFile(
          archiveFile: archiveFile,
          installPath: info.installRootPath,
          format: archiveFormat,
          allowOverride: overwriteExisting,
          customSoftwareName: trimmedRename,
        );
      case SoftwareSourceType.executable:
        final executableFile = File(info.sourcePath);
        if (!await executableFile.exists()) {
          throw Exception('原始可执行文件已不存在，请重新选择文件。');
        }
        return _addSoftwareFromExecutableFile(
          executableFile: executableFile,
          installPath: info.installRootPath,
          allowOverride: overwriteExisting,
          customSoftwareName: trimmedRename,
        );
    }
  }

  /// 根据当前设置在给定目录中查找可执行文件列表。
  Future<List<String>> findExecutablesInDirectory(Directory root) async {
    final maxDepth = await _settingsService.getExecutableSearchMaxDepth();
    final extensions = await _settingsService.getExecutableExtensions();
    final normalizedExtensions = extensions
        .map((ext) => ext.toLowerCase())
        .toSet();
    final results = <String>{};

    await _scanForExecutables(
      directory: root,
      currentDepth: 0,
      maxDepth: math.max(0, maxDepth),
      allowedExtensions: normalizedExtensions,
      results: results,
    );

    final sorted = results.toList()
      ..sort((a, b) => a.toLowerCase().compareTo(b.toLowerCase()));
    return sorted;
  }

  Future<void> _scanForExecutables({
    required Directory directory,
    required int currentDepth,
    required int maxDepth,
    required Set<String> allowedExtensions,
    required Set<String> results,
  }) async {
    try {
      await for (final entity in directory.list(
        recursive: false,
        followLinks: false,
      )) {
        if (entity is File) {
          final extension = p.extension(entity.path).toLowerCase();
          final normalizedExtension = extension.startsWith('.')
              ? extension.substring(1)
              : extension;
          if (allowedExtensions.contains(normalizedExtension)) {
            results.add(entity.path);
          }
        } else if (entity is Directory) {
          if (currentDepth >= maxDepth) continue;
          await _scanForExecutables(
            directory: entity,
            currentDepth: currentDepth + 1,
            maxDepth: maxDepth,
            allowedExtensions: allowedExtensions,
            results: results,
          );
        }
      }
    } catch (e, s) {
      logger.w('扫描目录 ${directory.path} 时出错', error: e, stackTrace: s);
    }
  }

  /// 若压缩包解压后仅包含单一子目录，则将子目录内容上移一级以避免多余嵌套。
  Future<bool> _flattenRedundantTopDirectory(Directory root) async {
    if (!await root.exists()) {
      return false;
    }

    final entries = await root.list(followLinks: false).toList();
    if (entries.isEmpty) {
      return false;
    }

    final retainedEntries = <FileSystemEntity>[];
    for (final entity in entries) {
      final name = p.basename(entity.path);
      final isDirectory = entity is Directory;
      if (_isIgnorableArchiveArtifact(name, isDirectory)) {
        try {
          await entity.delete(recursive: true);
        } catch (e, s) {
          logger.w('删除压缩包元数据失败: ${entity.path}', error: e, stackTrace: s);
        }
        continue;
      }
      retainedEntries.add(entity);
    }

    if (retainedEntries.isEmpty) {
      return false;
    }

    Directory? nestedDir;
    for (final entity in retainedEntries) {
      if (entity is Directory) {
        if (nestedDir != null) {
          return false;
        }
        nestedDir = entity;
        continue;
      }
      return false;
    }

    if (nestedDir == null) {
      return false;
    }

    final nestedName = p.basename(nestedDir.path);
    if (nestedName.isEmpty ||
        nestedName == '.' ||
        nestedName == '..' ||
        nestedName.startsWith('.')) {
      return false;
    }

    final nestedEntries = await nestedDir.list(followLinks: false).toList();
    if (nestedEntries.isEmpty) {
      await nestedDir.delete(recursive: true);
      return true;
    }

    for (final entity in nestedEntries) {
      final targetPath = p.join(root.path, p.basename(entity.path));
      final targetType = await FileSystemEntity.type(targetPath);
      if (targetType != FileSystemEntityType.notFound) {
        logger.w('展开嵌套目录时发现命名冲突，已跳过: $targetPath');
        return false;
      }
    }

    for (final entity in nestedEntries) {
      final targetPath = p.join(root.path, p.basename(entity.path));
      try {
        await entity.rename(targetPath);
      } catch (e, s) {
        logger.w(
          '展开嵌套目录失败: ${entity.path} -> $targetPath',
          error: e,
          stackTrace: s,
        );
        return false;
      }
    }

    try {
      await nestedDir.delete(recursive: true);
    } catch (e, s) {
      logger.w('删除空目录失败: ${nestedDir.path}', error: e, stackTrace: s);
    }

    return true;
  }

  /// 判断压缩包解压产物是否属于可忽略的元数据文件或目录。
  bool _isIgnorableArchiveArtifact(String name, bool isDirectory) {
    final lower = name.toLowerCase();
    if (lower == '__macosx' && isDirectory) {
      return true;
    }
    if (lower == '.ds_store' ||
        lower == 'thumbs.db' ||
        lower == 'desktop.ini') {
      return true;
    }
    if (lower.startsWith('._')) {
      return true;
    }
    return false;
  }

  Future<List<Software>> _loadManagedSoftware() async {
    try {
      final file = await _softwareListFile;
      if (await file.exists()) {
        return await _withSoftwareListLock(() async {
          final content = await file.readAsString();
          if (content.trim().isNotEmpty) {
            final decoded = json.decode(content);
            if (decoded is List) {
              final softwareList = decoded
                  .map((e) => Software.fromJson(Map<String, dynamic>.from(e)))
                  .toList();
              for (var i = 0; i < softwareList.length; i++) {
                softwareList[i].sortOrder = i;
              }
              return softwareList;
            }
          }
          return <Software>[];
        });
      }
      return [];
    } catch (e, s) {
      logger.e('读取托管软件列表失败', error: e, stackTrace: s);
      rethrow;
    }
  }

  Future<Software?> _findManagedSoftwareByPaths(
    String installPath,
    String archivePath,
  ) async {
    final managedSoftware = await _loadManagedSoftware();
    for (final software in managedSoftware) {
      final matchesInstall =
          installPath.isNotEmpty && p.equals(software.installPath, installPath);
      final matchesArchive =
          archivePath.isNotEmpty && p.equals(software.archivePath, archivePath);
      if (matchesInstall || matchesArchive) {
        return software;
      }
    }
    return null;
  }

  Future<int?> _prepareForOverride({
    required String installPath,
    required String archivePath,
  }) async {
    final managedSoftware = await _loadManagedSoftware();
    var removed = false;
    final retained = <Software>[];
    int? removedSortOrder;
    for (final software in managedSoftware) {
      final matchesInstall =
          installPath.isNotEmpty && p.equals(software.installPath, installPath);
      final matchesArchive =
          archivePath.isNotEmpty && p.equals(software.archivePath, archivePath);
      if (matchesInstall || matchesArchive) {
        removed = true;
        removedSortOrder = software.sortOrder;
        continue;
      }
      retained.add(software);
    }

    if (removed) {
      await _saveSoftwareList(retained);
    }

    final existingDir = Directory(installPath);
    if (await existingDir.exists()) {
      await existingDir.delete(recursive: true);
    }

    if (archivePath.isNotEmpty) {
      final existingArchive = File(archivePath);
      if (await existingArchive.exists()) {
        await existingArchive.delete();
      }
    }

    return removedSortOrder;
  }

  String _sanitizeSoftwareName(String name) {
    final sanitized = name.trim().replaceAll(RegExp(r'[\\/:*?"<>|]'), '_');
    if (sanitized.isEmpty) {
      return '软件_${DateTime.now().millisecondsSinceEpoch}';
    }
    return sanitized;
  }

  String _buildArchiveFileName({
    required String sanitizedName,
    required String sourceFileName,
  }) {
    final extension = p.extension(sourceFileName);
    if (extension.isEmpty) {
      return sanitizedName;
    }

    final lowerName = sanitizedName.toLowerCase();
    final lowerExt = extension.toLowerCase();
    if (lowerName.endsWith(lowerExt)) {
      return sanitizedName;
    }
    return '$sanitizedName$extension';
  }

  /// 清理添加失败后残留的临时文件。
  Future<void> cleanupTemporaryFiles({
    String? installPath,
    String? archivePath,
  }) async {
    logger.d('正在清理临时文件...');
    try {
      if (archivePath != null) {
        final file = File(archivePath);
        if (await file.exists()) {
          await file.delete();
          logger.d('已删除临时归档: $archivePath');
        }
      }
      if (installPath != null) {
        final dir = Directory(installPath);
        if (await dir.exists()) {
          await dir.delete(recursive: true);
          logger.d('已删除临时安装目录: $installPath');
        }
      }
    } catch (e, s) {
      logger.w('清理临时文件时出错', error: e, stackTrace: s);
    }
  }

  /// 更新一个已托管软件的主可执行文件路径。
  Future<void> updateSoftwareExecutable(
    String softwareId,
    String newExecutablePath,
  ) async {
    logger.i('开始更新软件 (ID: $softwareId) 的主程序为: $newExecutablePath');
    try {
      final softwareList = await getSoftwareList();
      final softwareToUpdate = softwareList.cast<Software?>().firstWhere(
        (s) => s?.id == softwareId,
        orElse: () => null,
      );

      if (softwareToUpdate == null) {
        throw Exception('未找到 ID 为 $softwareId 的软件');
      }

      softwareToUpdate.executablePath = newExecutablePath;
      await _saveSoftwareList(softwareList);
      logger.i('软件 (ID: $softwareId) 的主程序更新成功');
    } catch (e, s) {
      logger.e('更新软件主程序失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  /// 根据用户提供的顺序更新已托管软件的排序索引。
  Future<void> updateManagedSoftwareOrder(List<String> orderedIds) async {
    logger.i('更新用户自定义排序: ${orderedIds.join(', ')}');
    try {
      final managedSoftware = await _loadManagedSoftware();

      final idOrderMap = <String, int>{};
      for (var index = 0; index < orderedIds.length; index++) {
        idOrderMap[orderedIds[index]] = index;
      }

      var nextOrder = orderedIds.length;
      for (final software in managedSoftware) {
        final newOrder = idOrderMap[software.id];
        if (newOrder != null) {
          software.sortOrder = newOrder;
        } else {
          software.sortOrder = nextOrder++;
        }
      }

      await _saveSoftwareList(managedSoftware);
    } catch (e, s) {
      logger.e('更新软件排序失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }

  Future<void> deleteSoftware(
    Software software, {
    bool deleteInstallDir = false,
    bool deleteArchive = false,
  }) async {
    logger.i('开始删除软件: ${software.name} (ID: ${software.id})');
    try {
      // 先加载已托管列表，用于判断当前条目是否来源于持久化记录
      final managedList = await _loadManagedSoftware();
      final existsPersisted = managedList.any((s) => s.id == software.id);

      if (software.status == SoftwareStatus.unknownInstall) {
        if (deleteInstallDir) {
          final installDir = Directory(software.installPath);
          if (await installDir.exists()) {
            await installDir.delete(recursive: true);
          }
        }
        // 若该未知条目实际上源自持久化记录（安装目录丢失导致变为未知），则从列表中移除记录
        if (existsPersisted) {
          managedList.removeWhere((s) => s.id == software.id);
          await _saveSoftwareList(managedList);
        }
        return;
      }
      if (software.status == SoftwareStatus.unknownArchive) {
        if (deleteArchive) {
          final archiveFile = File(software.archivePath);
          if (await archiveFile.exists()) {
            await archiveFile.delete();
          }
        }
        if (existsPersisted) {
          managedList.removeWhere((s) => s.id == software.id);
          await _saveSoftwareList(managedList);
        }
        return;
      }

      if (deleteInstallDir) {
        final installDir = Directory(software.installPath);
        if (await installDir.exists()) {
          await installDir.delete(recursive: true);
        }
      }

      if (deleteArchive) {
        final archiveFile = File(software.archivePath);
        if (await archiveFile.exists()) {
          await archiveFile.delete();
        }
      }

      managedList.removeWhere((s) => s.id == software.id);
      await _saveSoftwareList(managedList);
    } catch (e, s) {
      logger.e('删除软件 "${software.name}" 失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }
}
