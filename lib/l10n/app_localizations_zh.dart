// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appName => '绿驿管家';

  @override
  String get appNameShort => '绿驿';

  @override
  String get loading => '加载中...';

  @override
  String get unknown => '未知';

  @override
  String get success => '成功';

  @override
  String get error => '错误';

  @override
  String get warning => '警告';

  @override
  String get cancel => '取消';

  @override
  String get save => '保存';

  @override
  String get delete => '删除';

  @override
  String get edit => '编辑';

  @override
  String get confirm => '确认';

  @override
  String get close => '关闭';

  @override
  String get ok => '确定';

  @override
  String get yes => '是';

  @override
  String get no => '否';

  @override
  String get retry => '重试';

  @override
  String get skip => '跳过';

  @override
  String get back => '返回';

  @override
  String get next => '下一步';

  @override
  String get finish => '完成';

  @override
  String get apply => '应用';

  @override
  String get reset => '重置';

  @override
  String get clear => '清除';

  @override
  String get search => '搜索';

  @override
  String get refresh => '刷新';

  @override
  String get open => '打开';

  @override
  String get browse => '浏览';

  @override
  String get select => '选择';

  @override
  String get add => '添加';

  @override
  String get remove => '移除';

  @override
  String get navHome => '主页';

  @override
  String get navArchive => '归档管理';

  @override
  String get navSettings => '设置';

  @override
  String get navDeveloperOptions => '开发者选项';

  @override
  String get navToggleTooltip => '折叠导航栏';

  @override
  String get navExpandTooltip => '展开导航栏';

  @override
  String get settingsTitle => '设置';

  @override
  String get homeTitle => '主页';

  @override
  String get languageChinese => '简体中文';

  @override
  String get languageEnglish => 'English';

  @override
  String get tileOpenFolder => '打开软件文件夹';

  @override
  String get tileChangeFolder => '修改软件文件夹';

  @override
  String get tileCreateBackup => '创建备份';

  @override
  String get tileAddToStartMenu => '添加到开始菜单';

  @override
  String get tileOpenArchiveFolder => '打开归档文件夹';

  @override
  String get tileLaunchOther => '启动其他程序';

  @override
  String get tileArchiveExists => '归档文件存在';

  @override
  String get tileArchiveNotExists => '归档文件不存在';

  @override
  String get tileBackupDetected => '检测到备份';

  @override
  String get tileBackupNotDetected => '未检测到备份';

  @override
  String get tileHintNoInstallDir => '提示';

  @override
  String get tileHintNoInstallDirMessage => '该软件未配置安装目录。';

  @override
  String tileHintCannotOpenDir(Object path) {
    return '找不到安装目录：$path';
  }

  @override
  String get tileError => '错误';

  @override
  String get tileErrorCannotOpenDirMessage => '无法打开安装目录，请稍后重试。';

  @override
  String get tileHintNoArchive => '提示';

  @override
  String get tileHintNoArchiveMessage => '该软件未配置归档文件。';

  @override
  String get tileHintArchiveNotExists => '归档文件不存在，已打开归档目录。';

  @override
  String get tileHintParentOpened => '归档文件不存在，已打开归档所在目录。';

  @override
  String get tileHintArchiveNotFound => '找不到归档文件所在位置。';

  @override
  String get tileErrorCannotOpenArchive => '无法打开归档目录，请稍后重试。';

  @override
  String get tileHintNoExecutable => '提示';

  @override
  String get tileHintNoExecutableMessage => '该软件未配置可执行文件路径。';

  @override
  String get tileHintInstallDirNotExists => '安装目录不存在，无法创建备份。';

  @override
  String get tileCreatingBackup => '正在创建备份...';

  @override
  String tileBackupCreated(Object name) {
    return '已创建备份：$name';
  }

  @override
  String get tileErrorCreateBackup => '创建备份失败，请稍后重试。';

  @override
  String get tileSuccess => '成功';

  @override
  String tileAddedToStartMenu(Object name) {
    return '已将 \"$name\" 添加到开始菜单。';
  }

  @override
  String tileErrorAddToStartMenu(Object error) {
    return '添加到开始菜单失败：$error';
  }

  @override
  String get tileCarryDock => '绿驿管家';

  @override
  String get tileCarryDockShortcutAdding => '开始菜单中不存在绿驿管家快捷方式，正在添加...';

  @override
  String get tileCarryDockShortcutAdded => '绿驿管家快捷方式已添加到开始菜单';

  @override
  String get homeUnknownFolder => '未知文件夹';

  @override
  String get homeUnknownArchiveFile => '未知归档文件';

  @override
  String get homeSoftwareDirDeleted => '软件目录已删除';

  @override
  String get homeRehost => '重新托管';

  @override
  String get homeBackupArchive => '备份归档';

  @override
  String get homeArchiveFile => '归档文件';

  @override
  String get homeChangeExecutable => '更改主程序';

  @override
  String get homeScanning => '正在扫描可执行程序...';

  @override
  String get homeNoExecutableFound => '未在安装目录中找到可执行程序';

  @override
  String get homeOnlyOneExecutableFound => '当前仅发现一个可执行程序，无法更改主程序';

  @override
  String tileNewPathHint(Object path) {
    return '新路径将是：$path\\[您输入的名称]';
  }

  @override
  String tileErrorMigrateFailed(Object error) {
    return '修改软件文件夹失败：$error';
  }

  @override
  String get tileChangeSoftwareFolder => '修改软件文件夹';

  @override
  String get tileCurrentPath => '当前路径：';

  @override
  String get tileNewFolderName => '新文件夹名称：';

  @override
  String get tileNewFolderPlaceholder => '输入新的文件夹名称';

  @override
  String get tileRenameConfirm => '确定';

  @override
  String get tileRenameCancel => '取消';

  @override
  String get tileHintSamePath => '提示';

  @override
  String get tileHintSamePathMessage => '新路径与当前路径相同，无需更新。';

  @override
  String get tileErrorFolderNotExist => '当前软件文件夹不存在，无法迁移。';

  @override
  String get tileErrorFolderExists => '新文件夹已存在，请选择其他名称。';

  @override
  String get tileMigrating => '正在迁移文件...';

  @override
  String get tileMigrateSuccess => '软件文件夹已成功迁移。';

  @override
  String get archiveTitle => '归档管理';

  @override
  String get archiveFiles => '归档文件';

  @override
  String get archiveNoArchiveFiles => '暂无归档文件';

  @override
  String get archiveBackupFiles => '备份文件';

  @override
  String get archiveNoBackupFiles => '暂无备份文件';

  @override
  String get archiveRefresh => '刷新';

  @override
  String get archiveOpenArchiveDir => '打开归档目录';

  @override
  String get archiveOpenBackupDir => '打开备份目录';

  @override
  String get archiveOperationFailed => '操作失败';

  @override
  String get archiveCannotOpenDirectory => '无法打开目录。';

  @override
  String get archiveCannotOpenExplorer => '无法打开资源管理器。';

  @override
  String get archiveInstallingFromArchive => '正在从归档安装...';

  @override
  String get archiveCompletingInstallation => '正在完成安装...';

  @override
  String get archiveCancellingOperation => '正在取消操作...';

  @override
  String get archiveDuplicateDetected => '检测到重复';

  @override
  String get archiveDuplicateMessage => '安装目录已存在：\n';

  @override
  String get archiveBackupAndRestore => '备份并还原';

  @override
  String get archiveRestoring => '正在还原...';

  @override
  String get archiveBackupRestoreFailed => '备份和还原失败。';

  @override
  String get archiveInstallFailed => '安装失败。';

  @override
  String get archiveDeleteArchive => '删除归档';

  @override
  String get archiveDeleteBackup => '删除备份';

  @override
  String get archiveDeleteArchiveHint => '确定要删除此归档吗？';

  @override
  String get archiveDeleteBackupHint => '确定要删除此备份吗？它可能已关联到已托管软件。';

  @override
  String get archiveDeletingArchive => '正在删除归档...';

  @override
  String get archiveDeletingBackup => '正在删除备份...';

  @override
  String get archiveDeleteFailed => '删除失败';

  @override
  String get archiveCannotDeleteFile => '无法删除文件。';

  @override
  String get archiveNoAvailableSoftware => '无可用软件';

  @override
  String get archiveNoAvailableSoftwareHint => '没有已托管的软件。请先添加软件。';

  @override
  String get archiveLinkArchive => '关联归档';

  @override
  String get archiveLinkBackup => '关联备份';

  @override
  String get archiveLink => '关联';

  @override
  String get archiveLinkSuccess => '关联成功';

  @override
  String get archiveLinkSuccessHint => '归档已关联到软件。';

  @override
  String get archiveManualLink => '手动关联';

  @override
  String get archiveRestore => '还原';

  @override
  String get archiveCreateBackup => '创建备份';

  @override
  String get archiveCreatingBackup => '正在创建备份...';

  @override
  String get archiveCreateSuccess => '成功';

  @override
  String get archiveBackupCreated => '已创建备份：';

  @override
  String get archiveCreateFailed => '创建失败';

  @override
  String get archiveCreateBackupFailed => '创建备份失败。';

  @override
  String get devTitle => '开发者选项';

  @override
  String get devHint => '这是一个隐藏菜单，用于调试和测试。';

  @override
  String get devIconTest => '图标测试';

  @override
  String get devIconTestHint => '测试从可执行文件提取图标';

  @override
  String get devExtractFailed => '提取失败';

  @override
  String get devExtractFailedMessage => '无法从文件中提取图标。';

  @override
  String get devFileName => '文件：';

  @override
  String get devMethod1 => '方法1：直接内存显示';

  @override
  String get devMethod1Desc => '直接从内存字节显示图标（32x32）';

  @override
  String get devMethod2 => '方法2：高质量显示';

  @override
  String get devMethod2Desc => '使用更高质量设置显示图标（64x64）';

  @override
  String get devMethod3 => '方法3：ImageProvider 集成';

  @override
  String get devMethod3Desc => '在组件中测试作为 ImageProvider 的图标';

  @override
  String get devButtonWithIcon => '按钮';

  @override
  String get devIconHint => '上面的图标是使用从可执行文件中提取的字节渲染的。';

  @override
  String get devClose => '关闭';

  @override
  String get devHide => '隐藏开发者选项';

  @override
  String get devHiddenFeature => '隐藏功能';

  @override
  String get devHiddenFeatureMessage => '这是一个用于未来开发的隐藏功能。';

  @override
  String get devHiddenButton => '隐藏';

  @override
  String get dialogSelectExecutable => '选择可执行文件';

  @override
  String get dialogCancel => '取消';

  @override
  String get dialogConfirm => '确认';

  @override
  String get dialogUpdateInfo => '更新信息';

  @override
  String get dialogNoUpdateInfo => '没有可用的更新信息。';

  @override
  String get dialogClose => '关闭';

  @override
  String dialogNewVersion(String version) {
    return '发现新版本：$version';
  }

  @override
  String get dialogUpdateContent => '更新内容';

  @override
  String get dialogRetry => '重试';

  @override
  String get dialogUpdateSuccess => '更新成功';

  @override
  String get dialogUpdateSuccessMessage => '应用程序已成功更新。';

  @override
  String get dialogDownloadAndUpdate => '下载并更新';

  @override
  String get dialogInvalidExtension => '扩展名格式无效。必须以\'.\'开头（例如\'.exe\'）';

  @override
  String get dialogExtensionExists => '此扩展名已存在。';

  @override
  String get dialogNoExtensionsSelected => '未选择扩展名';

  @override
  String get dialogSelectExtensions => '选择扩展名';

  @override
  String get dialogCommonExtensions => '常用扩展名';

  @override
  String get dialogCustomExtensions => '自定义扩展名';

  @override
  String get dialogExtensionPlaceholder => '.exe';

  @override
  String get dialogAdd => '添加';

  @override
  String get dialogSelected => '已选择';

  @override
  String get dialogSelectAtLeastOne => '请至少选择一个扩展名。';

  @override
  String get settingsError => '错误';

  @override
  String get settingsSuccess => '成功';

  @override
  String get settingsStorage => '存储';

  @override
  String get settingsInstallPath => '安装路径';

  @override
  String get settingsInstallPathPlaceholder => '输入安装路径';

  @override
  String get settingsSaveInstallPath => '保存路径';

  @override
  String get settingsInstallPathSaved => '安装路径已保存。';

  @override
  String get settingsArchivePath => '归档路径';

  @override
  String get settingsArchivePathPlaceholder => '输入归档路径';

  @override
  String get settingsSaveArchivePath => '保存路径';

  @override
  String get settingsArchivePathSaved => '归档路径已保存。';

  @override
  String get settingsConfigFilePath => '配置文件';

  @override
  String get settingsOpenFileLocation => '打开位置';

  @override
  String get settingsInvalidConfigPath => '配置文件路径无效。';

  @override
  String get settingsArchiveHandling => '归档处理';

  @override
  String get settingsRemoveNestedFolders => '删除嵌套文件夹';

  @override
  String get settingsRemoveNestedFoldersDesc => '解压归档时删除嵌套文件夹';

  @override
  String get settingsSaveArchiveHandling => '保存设置';

  @override
  String get settingsArchiveHandlingSaved => '归档处理设置已保存。';

  @override
  String get settingsTest7Zip => '测试 7-Zip';

  @override
  String get settingsTest7ZipDesc => '测试系统中是否能找到 7-Zip';

  @override
  String get settingsExecutableRecognition => '可执行文件识别';

  @override
  String get settingsMaxSearchDepth => '最大搜索深度';

  @override
  String get settingsInvalidSearchDepth => '搜索深度值无效。';

  @override
  String get settingsExecutableExtensions => '可执行文件扩展名';

  @override
  String get settingsInvalidExtensions => '扩展名列表无效。';

  @override
  String get settingsNoExtensionsSelected => '未选择扩展名';

  @override
  String get settingsSelectExtensions => '选择扩展名';

  @override
  String get settingsSaveExecutableSettings => '保存设置';

  @override
  String get settingsExecutableSettingsSaved => '可执行文件识别设置已保存。';

  @override
  String get settingsAppearance => '外观';

  @override
  String get settingsAppFont => '应用字体';

  @override
  String get settingsLanguage => '语言';

  @override
  String get settingsSaveLanguage => '保存语言';

  @override
  String get settingsLanguageSaved => '语言设置已保存。';

  @override
  String get translationByAI => '（AI翻译）';

  @override
  String get settingsLogSettings => '日志设置';

  @override
  String get settingsEnableFileLogging => '启用文件日志';

  @override
  String get settingsEnableFileLoggingDesc => '将日志保存到文件以便调试';

  @override
  String get settingsLogFilePath => '日志文件';

  @override
  String get settingsInvalidLogPath => '日志文件路径无效。';

  @override
  String get settingsLogSettingsSaved => '日志设置已保存。';

  @override
  String get settingsAbout => '关于';

  @override
  String get settingsVersion => '版本';

  @override
  String get settingsBuildTime => '构建时间';

  @override
  String get settingsCheckUpdate => '检查更新';

  @override
  String get settingsReinstallConfirm => '重新安装应用';

  @override
  String get settingsReinstallMessage => '这将下载并重新安装最新版本。确定吗？';

  @override
  String get settingsReinstallConfirmButton => '重新安装';

  @override
  String get settingsReinstallFailed => '重新安装失败';

  @override
  String settingsReinstallFailedMessage(String message) {
    return '重新安装失败：$message';
  }

  @override
  String get settingsReinstallLatest => '重新安装最新版本';

  @override
  String get settingsDevHint => '开发阶段';

  @override
  String get settingsDevHintContent => '如果遇到任何问题，请尝试删除配置文件并重启。';

  @override
  String get settingsUnsavedChanges => '未保存的更改';

  @override
  String get settingsUnsavedChangesMessage => '您有未保存的更改。是否在退出前保存？';

  @override
  String get settingsDiscardChanges => '放弃';

  @override
  String get settingsSaveAndExit => '保存并退出';

  @override
  String get settingsSaveAll => '全部保存';

  @override
  String get test7ZipTitle => '7-Zip 测试';

  @override
  String get test7ZipClose => '关闭';

  @override
  String get homeCopySuffix => ' - 副本';

  @override
  String get homeNewSoftware => '新软件';

  @override
  String homeDuplicateExisting(String name, String path) {
    return '已托管软件：$name（$path）';
  }

  @override
  String homeDuplicateInstallDir(String path) {
    return '安装目录已存在：$path';
  }

  @override
  String homeDuplicateArchive(String name) {
    return '归档文件已存在：$name';
  }

  @override
  String get homeDuplicateDetected => '检测到重复';

  @override
  String homeDuplicateHelp(String name) {
    return '软件名称「$name」与现有项目冲突。请选择如何继续：';
  }

  @override
  String get homeDuplicateRenameHint => '输入新名称以重命名并添加：';

  @override
  String get homeDuplicateCancel => '取消';

  @override
  String get homeDuplicateRenameAdd => '重命名并添加';

  @override
  String get homeDuplicateOverwrite => '覆盖';

  @override
  String get homeInstallDirNotExistNoArchive => '安装目录未配置';

  @override
  String get homeScanArchiveTitle => '扫描并归档';

  @override
  String homeScanArchiveHint(String path) {
    return '扫描「$path」中的所有子目录并将其归档为单独的软件？';
  }

  @override
  String homeScanArchiveBackupOn(String path) {
    return '归档将保存到：$path';
  }

  @override
  String get homeScanArchiveBackupOff => '将不会创建备份。';

  @override
  String get homeScanArchiveToggle => '为原始软件创建备份';

  @override
  String get homeScanArchiveToggleHint => '如果启用，将在归档前在备份文件夹中创建备份。';

  @override
  String get homeScanArchiveStart => '开始扫描';

  @override
  String get homeScanScanning => '正在扫描...';

  @override
  String homeScanProgress(String done, String total, String percent) {
    return '$done/$total（$percent%）';
  }

  @override
  String homeScanCurrent(String name) {
    return '当前：$name';
  }

  @override
  String get homeScanSummaryTitle => '扫描完成';

  @override
  String homeScanSummaryInstallDir(String path) {
    return '安装目录：$path';
  }

  @override
  String homeScanSummaryArchiveDir(String path) {
    return '归档目录：$path';
  }

  @override
  String homeScanSummaryTotal(String count) {
    return '共扫描：$count';
  }

  @override
  String homeScanSummaryArchived(String count) {
    return '成功归档：$count';
  }

  @override
  String homeScanSummarySkipped(String count) {
    return '已跳过（已存在）：$count';
  }

  @override
  String homeScanSummaryFailed(String count) {
    return '失败：$count';
  }

  @override
  String homeScanSummaryFailedItem(String name, String error) {
    return '- $name：$error';
  }

  @override
  String homeScanSummaryMore(String count) {
    return '还有 $count 个失败项...';
  }

  @override
  String homeAssocFound(String count) {
    return '发现 $count 个归档关联';
  }

  @override
  String get homeAssocHint => '以下安装可能存在对应的归档文件。请选择要关联的项目：';

  @override
  String get homeAssocSkipAll => '全部跳过';

  @override
  String get homeAssocApply => '应用关联';

  @override
  String get homeProcessingFiles => '正在处理文件，请稍候...';

  @override
  String get homeInstallingSoftware => '正在安装软件...';

  @override
  String get homeCancellingOperation => '正在取消操作...';

  @override
  String homeDeleteSoftware(String name) {
    return '删除 $name';
  }

  @override
  String homeDeleteUnmanagedHint(String name) {
    return '您确定要删除这个$name吗？此操作不可恢复。';
  }

  @override
  String get homeDeleteUnmanagedFolder => '未知文件夹';

  @override
  String get homeDeleteUnmanagedArchive => '未知归档文件';

  @override
  String get homeDeleteManagedHint => '您确定要删除此软件吗？此操作将从列表中移除该软件。';

  @override
  String get homeDeleteInstallDir => '同时删除安装目录';

  @override
  String get homeDeleteArchive => '同时删除归档文件';

  @override
  String get homeDeleting => '正在删除，请稍候...';

  @override
  String get homeManagedSoftware => '已托管软件';

  @override
  String get homeListView => '列表视图';

  @override
  String get homeGridView => '网格视图';

  @override
  String get homeReorder => '调整排序';

  @override
  String get homeOpenInstallDir => '打开安装目录';

  @override
  String get homeScanCurrentDir => '扫描当前目录软件';

  @override
  String get homeNoSoftware => '尚未添加任何软件。';

  @override
  String get homeDropToAdd => '松开以添加软件';

  @override
  String get homeDropHint => '支持 ZIP/TAR/RAR/7Z 压缩包或已配置的可执行文件';

  @override
  String get homeReorderTitle => '调整软件顺序';

  @override
  String get homeMoveUp => '向上移动';

  @override
  String get homeMoveDown => '向下移动';

  @override
  String get homeReorderHint => '通过点击右侧箭头可调整展示顺序，保存后将在主页生效。';

  @override
  String get homeSaveOrder => '保存顺序';

  @override
  String test7ZipError(String error) {
    return '测试过程中发生错误：\n$error';
  }

  @override
  String errorUnhandled(String error, String stackTrace) {
    return '发生了一个未处理的错误:\n\n$error\n\n堆栈跟踪:\n$stackTrace';
  }

  @override
  String get errorOk => '好的';
}
