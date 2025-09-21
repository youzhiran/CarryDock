import 'dart:io';
import 'dart:math' as math;

import 'package:carrydock/models/software.dart';
import 'package:carrydock/services/archive_extractor.dart';
import 'package:carrydock/services/json_storage_service.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/utils/error_handler.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:file_picker/file_picker.dart';
import 'package:path/path.dart' as p;
import 'package:uuid/uuid.dart';

/// A data class to hold information for a pending software addition
/// when multiple executables are found.
class PendingSoftwareAddition {
  final String installPath;
  final String archivePath;
  final List<String> executablePaths;
  final int? preferredSortOrder;

  PendingSoftwareAddition({
    required this.installPath,
    required this.archivePath,
    required this.executablePaths,
    this.preferredSortOrder,
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
  static const String _softwareListKey = 'software_list';

  final SettingsService _settingsService = SettingsService();
  final JsonStorageService _storageService = JsonStorageService();
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

  Future<List<Software>> getSoftwareList() async {
    logger.i('开始获取软件列表...');
    try {
      final List<Software> managedSoftware = await _loadManagedSoftware();

      for (final software in managedSoftware) {
        final archiveFile = File(software.archivePath);
        software.archiveExists = await archiveFile.exists();
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
        final entities = await archivesDir.list().toList();
        for (final entity in entities) {
          if (entity is File) {
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

  Future<void> _saveSoftwareList(List<Software> softwareList) async {
    final managedSoftware =
        softwareList.where((s) => s.status == SoftwareStatus.managed).toList()
          ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
    for (var i = 0; i < managedSoftware.length; i++) {
      managedSoftware[i].sortOrder = i;
    }
    try {
      final jsonList = managedSoftware.map((s) => s.toJson()).toList();
      await _storageService.setValue(_softwareListKey, jsonList);
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

    try {
      final baseName = p.basenameWithoutExtension(archiveFile.path);
      final fallbackName = baseName.isEmpty
          ? p.basename(archiveFile.path)
          : baseName;
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
      if (allowOverride) {
        preferredSortOrder = await _prepareForOverride(
          installPath: destinationDir.path,
          archivePath: archivedPath,
        );
      }

      final storedArchiveFile = await archiveFile.copy(archivedPath);
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
        archivePath: archivedPath,
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
      if (allowOverride) {
        preferredSortOrder = await _prepareForOverride(
          installPath: destinationDir.path,
          archivePath: archivedPath,
        );
      }

      await executableFile.copy(archivedPath);
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
        archivePath: archivedPath,
      );
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
      final newSoftware = Software(
        id: _uuid.v4(),
        name: softwareName,
        installPath: installPath,
        executablePath: selectedExecutablePath,
        archivePath: archivePath,
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
      final jsonList = await _storageService.getValue<List<dynamic>>(
        _softwareListKey,
      );
      if (jsonList == null) {
        return [];
      }
      final softwareList = jsonList
          .map((json) => Software.fromJson(json))
          .toList();
      for (var i = 0; i < softwareList.length; i++) {
        softwareList[i].sortOrder = i;
      }
      return softwareList;
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
      if (software.status == SoftwareStatus.unknownInstall) {
        final installDir = Directory(software.installPath);
        if (await installDir.exists()) {
          await installDir.delete(recursive: true);
        }
        return;
      }
      if (software.status == SoftwareStatus.unknownArchive) {
        final archiveFile = File(software.archivePath);
        if (await archiveFile.exists()) {
          await archiveFile.delete();
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

      final softwareList = await _loadManagedSoftware();

      softwareList.removeWhere((s) => s.id == software.id);

      await _saveSoftwareList(softwareList);
    } catch (e, s) {
      logger.e('删除软件 "${software.name}" 失败', error: e, stackTrace: s);
      errorHandler?.handleError(e, s);
      rethrow;
    }
  }
}
