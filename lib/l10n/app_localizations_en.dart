// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appName => 'CarryDock';

  @override
  String get appNameShort => 'CarryDock';

  @override
  String get loading => 'Loading...';

  @override
  String get unknown => 'Unknown';

  @override
  String get success => 'Success';

  @override
  String get error => 'Error';

  @override
  String get warning => 'Warning';

  @override
  String get cancel => 'Cancel';

  @override
  String get save => 'Save';

  @override
  String get delete => 'Delete';

  @override
  String get edit => 'Edit';

  @override
  String get confirm => 'Confirm';

  @override
  String get close => 'Close';

  @override
  String get ok => 'OK';

  @override
  String get yes => 'Yes';

  @override
  String get no => 'No';

  @override
  String get retry => 'Retry';

  @override
  String get skip => 'Skip';

  @override
  String get back => 'Back';

  @override
  String get next => 'Next';

  @override
  String get finish => 'Finish';

  @override
  String get apply => 'Apply';

  @override
  String get reset => 'Reset';

  @override
  String get clear => 'Clear';

  @override
  String get search => 'Search';

  @override
  String get refresh => 'Refresh';

  @override
  String get open => 'Open';

  @override
  String get browse => 'Browse';

  @override
  String get select => 'Select';

  @override
  String get add => 'Add';

  @override
  String get remove => 'Remove';

  @override
  String get navHome => 'Home';

  @override
  String get navArchive => 'Archive Manager';

  @override
  String get navSettings => 'Settings';

  @override
  String get navDeveloperOptions => 'Developer Options';

  @override
  String get navToggleTooltip => 'Collapse navigation pane';

  @override
  String get navExpandTooltip => 'Expand navigation pane';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get homeTitle => 'Home';

  @override
  String get languageChinese => '简体中文';

  @override
  String get languageEnglish => 'English';

  @override
  String get tileOpenFolder => 'Open Software Folder';

  @override
  String get tileChangeFolder => 'Change Software Folder';

  @override
  String get tileCreateBackup => 'Create Backup';

  @override
  String get tileAddToStartMenu => 'Add to Start Menu';

  @override
  String get tileOpenArchiveFolder => 'Open Archive Folder';

  @override
  String get tileLaunchOther => 'Launch Other Program';

  @override
  String get tileArchiveExists => 'Archive file exists';

  @override
  String get tileArchiveNotExists => 'Archive file does not exist';

  @override
  String get tileBackupDetected => 'Backup detected';

  @override
  String get tileBackupNotDetected => 'No backup detected';

  @override
  String get tileHintNoInstallDir => 'Hint';

  @override
  String get tileHintNoInstallDirMessage =>
      'This software has no installation directory configured.';

  @override
  String tileHintCannotOpenDir(Object path) {
    return 'Cannot find installation directory: $path';
  }

  @override
  String get tileError => 'Error';

  @override
  String get tileErrorCannotOpenDirMessage =>
      'Cannot open installation directory, please try again later.';

  @override
  String get tileHintNoArchive => 'Hint';

  @override
  String get tileHintNoArchiveMessage =>
      'This software has no archive file configured.';

  @override
  String get tileHintArchiveNotExists =>
      'Archive file does not exist, opened archive directory.';

  @override
  String get tileHintParentOpened =>
      'Archive file does not exist, opened archive parent directory.';

  @override
  String get tileHintArchiveNotFound => 'Cannot find archive file location.';

  @override
  String get tileErrorCannotOpenArchive =>
      'Cannot open archive directory, please try again later.';

  @override
  String get tileHintNoExecutable => 'Hint';

  @override
  String get tileHintNoExecutableMessage =>
      'This software has no executable file path configured.';

  @override
  String get tileHintInstallDirNotExists =>
      'Installation directory does not exist, cannot create backup.';

  @override
  String get tileCreatingBackup => 'Creating backup...';

  @override
  String tileBackupCreated(Object name) {
    return 'Backup created: $name';
  }

  @override
  String get tileErrorCreateBackup =>
      'Failed to create backup, please try again later.';

  @override
  String get tileSuccess => 'Success';

  @override
  String tileAddedToStartMenu(Object name) {
    return 'Added \"$name\" to Start Menu.';
  }

  @override
  String tileErrorAddToStartMenu(Object error) {
    return 'Failed to add to Start Menu: $error';
  }

  @override
  String get tileCarryDock => 'CarryDock';

  @override
  String get tileCarryDockShortcutAdding =>
      'CarryDock shortcut does not exist in Start Menu, adding...';

  @override
  String get tileCarryDockShortcutAdded =>
      'CarryDock shortcut added to Start Menu';

  @override
  String get homeUnknownFolder => 'Unknown Folder';

  @override
  String get homeUnknownArchiveFile => 'Unknown Archive File';

  @override
  String get homeSoftwareDirDeleted => 'Software directory deleted';

  @override
  String get homeRehost => 'Rehost';

  @override
  String get homeBackupArchive => 'Backup Archive';

  @override
  String get homeArchiveFile => 'Archive File';

  @override
  String get homeChangeExecutable => 'Change Main Program';

  @override
  String get homeScanning => 'Scanning for executable programs...';

  @override
  String get homeNoExecutableFound =>
      'No executable programs found in installation directory';

  @override
  String get homeOnlyOneExecutableFound =>
      'Only one executable program found, cannot change main program';

  @override
  String tileNewPathHint(Object path) {
    return 'New path will be: $path\\[your input name]';
  }

  @override
  String tileErrorMigrateFailed(Object error) {
    return 'Failed to change software folder: $error';
  }

  @override
  String get tileChangeSoftwareFolder => 'Change Software Folder';

  @override
  String get tileCurrentPath => 'Current path:';

  @override
  String get tileNewFolderName => 'New folder name:';

  @override
  String get tileNewFolderPlaceholder => 'Enter new folder name';

  @override
  String get tileRenameConfirm => 'Confirm';

  @override
  String get tileRenameCancel => 'Cancel';

  @override
  String get tileHintSamePath => 'Hint';

  @override
  String get tileHintSamePathMessage =>
      'New path is the same as current path, no update needed.';

  @override
  String get tileErrorFolderNotExist =>
      'Current software folder does not exist, cannot migrate.';

  @override
  String get tileErrorFolderExists =>
      'New folder already exists, please choose another name.';

  @override
  String get tileMigrating => 'Migrating files...';

  @override
  String get tileMigrateSuccess => 'Software folder migrated successfully.';

  @override
  String get archiveTitle => 'Archive Manager';

  @override
  String get archiveFiles => 'Archives';

  @override
  String get archiveNoArchiveFiles => 'No archive files';

  @override
  String get archiveBackupFiles => 'Backups';

  @override
  String get archiveNoBackupFiles => 'No backup files';

  @override
  String get archiveRefresh => 'Refresh';

  @override
  String get archiveOpenArchiveDir => 'Open Archive Directory';

  @override
  String get archiveOpenBackupDir => 'Open Backup Directory';

  @override
  String get archiveOperationFailed => 'Operation Failed';

  @override
  String get archiveCannotOpenDirectory => 'Cannot open directory.';

  @override
  String get archiveCannotOpenExplorer => 'Cannot open Explorer.';

  @override
  String get archiveInstallingFromArchive => 'Installing from archive...';

  @override
  String get archiveCompletingInstallation => 'Completing installation...';

  @override
  String get archiveCancellingOperation => 'Cancelling operation...';

  @override
  String get archiveDuplicateDetected => 'Duplicate Detected';

  @override
  String get archiveDuplicateMessage =>
      'Installation directory already exists:\n';

  @override
  String get archiveBackupAndRestore => 'Backup and Restore';

  @override
  String get archiveRestoring => 'Restoring...';

  @override
  String get archiveBackupRestoreFailed => 'Backup and restore failed.';

  @override
  String get archiveInstallFailed => 'Installation failed.';

  @override
  String get archiveDeleteArchive => 'Delete Archive';

  @override
  String get archiveDeleteBackup => 'Delete Backup';

  @override
  String get archiveDeleteArchiveHint =>
      'Are you sure you want to delete this archive?';

  @override
  String get archiveDeleteBackupHint =>
      'Are you sure you want to delete this backup? It may be linked to managed software.';

  @override
  String get archiveDeletingArchive => 'Deleting archive...';

  @override
  String get archiveDeletingBackup => 'Deleting backup...';

  @override
  String get archiveDeleteFailed => 'Deletion Failed';

  @override
  String get archiveCannotDeleteFile => 'Cannot delete file.';

  @override
  String get archiveNoAvailableSoftware => 'No Available Software';

  @override
  String get archiveNoAvailableSoftwareHint =>
      'No managed software available. Please add software first.';

  @override
  String get archiveLinkArchive => 'Link Archive';

  @override
  String get archiveLinkBackup => 'Link Backup';

  @override
  String get archiveLink => 'Link';

  @override
  String get archiveLinkSuccess => 'Link Successful';

  @override
  String get archiveLinkSuccessHint => 'Archive has been linked to software.';

  @override
  String get archiveManualLink => 'Manual Link';

  @override
  String get archiveRestore => 'Restore';

  @override
  String get archiveCreateBackup => 'Create Backup';

  @override
  String get archiveCreatingBackup => 'Creating backup...';

  @override
  String get archiveCreateSuccess => 'Success';

  @override
  String get archiveBackupCreated => 'Backup created: ';

  @override
  String get archiveCreateFailed => 'Creation Failed';

  @override
  String get archiveCreateBackupFailed => 'Failed to create backup.';

  @override
  String get devTitle => 'Developer Options';

  @override
  String get devHint =>
      'This is a hidden menu for debugging and testing purposes.';

  @override
  String get devIconTest => 'Icon Test';

  @override
  String get devIconTestHint => 'Test icon extraction from executable files';

  @override
  String get devExtractFailed => 'Extraction Failed';

  @override
  String get devExtractFailedMessage => 'Failed to extract icon from file.';

  @override
  String get devFileName => 'File: ';

  @override
  String get devMethod1 => 'Method 1: Direct Memory Display';

  @override
  String get devMethod1Desc =>
      'Display icon directly from memory bytes (32x32)';

  @override
  String get devMethod2 => 'Method 2: High Quality Display';

  @override
  String get devMethod2Desc =>
      'Display icon with higher quality settings (64x64)';

  @override
  String get devMethod3 => 'Method 3: ImageProvider Integration';

  @override
  String get devMethod3Desc => 'Test icon as ImageProvider in widgets';

  @override
  String get devButtonWithIcon => 'Button';

  @override
  String get devIconHint =>
      'The icon above is rendered using the extracted bytes from the executable file.';

  @override
  String get devClose => 'Close';

  @override
  String get devHide => 'Hide Developer Options';

  @override
  String get devHiddenFeature => 'Hidden Feature';

  @override
  String get devHiddenFeatureMessage =>
      'This is a hidden feature for future development.';

  @override
  String get devHiddenButton => 'Hidden';

  @override
  String get dialogSelectExecutable => 'Select Executable';

  @override
  String get dialogCancel => 'Cancel';

  @override
  String get dialogConfirm => 'Confirm';

  @override
  String get dialogUpdateInfo => 'Update Info';

  @override
  String get dialogNoUpdateInfo => 'No update information available.';

  @override
  String get dialogClose => 'Close';

  @override
  String dialogNewVersion(String version) {
    return 'New Version Available: $version';
  }

  @override
  String get dialogUpdateContent => 'Update content';

  @override
  String get dialogRetry => 'Retry';

  @override
  String get dialogUpdateSuccess => 'Update Successful';

  @override
  String get dialogUpdateSuccessMessage =>
      'Application has been updated successfully.';

  @override
  String get dialogDownloadAndUpdate => 'Download and Update';

  @override
  String get dialogInvalidExtension =>
      'Invalid extension format. Must start with \'.\' (e.g. \'.exe\')';

  @override
  String get dialogExtensionExists => 'This extension already exists.';

  @override
  String get dialogNoExtensionsSelected => 'No extensions selected';

  @override
  String get dialogSelectExtensions => 'Select Extensions';

  @override
  String get dialogCommonExtensions => 'Common Extensions';

  @override
  String get dialogCustomExtensions => 'Custom Extensions';

  @override
  String get dialogExtensionPlaceholder => '.exe';

  @override
  String get dialogAdd => 'Add';

  @override
  String get dialogSelected => 'Selected';

  @override
  String get dialogSelectAtLeastOne => 'Please select at least one extension.';

  @override
  String get settingsError => 'Error';

  @override
  String get settingsSuccess => 'Success';

  @override
  String get settingsStorage => 'Storage';

  @override
  String get settingsInstallPath => 'Installation Path';

  @override
  String get settingsInstallPathPlaceholder => 'Enter installation path';

  @override
  String get settingsSaveInstallPath => 'Save Path';

  @override
  String get settingsInstallPathSaved => 'Installation path saved.';

  @override
  String get settingsArchivePath => 'Archive Path';

  @override
  String get settingsArchivePathPlaceholder => 'Enter archive path';

  @override
  String get settingsSaveArchivePath => 'Save Path';

  @override
  String get settingsArchivePathSaved => 'Archive path saved.';

  @override
  String get settingsConfigFilePath => 'Configuration File';

  @override
  String get settingsOpenFileLocation => 'Open Location';

  @override
  String get settingsInvalidConfigPath => 'Invalid configuration file path.';

  @override
  String get settingsArchiveHandling => 'Archive Handling';

  @override
  String get settingsRemoveNestedFolders => 'Remove Nested Folders';

  @override
  String get settingsRemoveNestedFoldersDesc =>
      'Remove nested folders when extracting archives';

  @override
  String get settingsSaveArchiveHandling => 'Save Settings';

  @override
  String get settingsArchiveHandlingSaved => 'Archive handling settings saved.';

  @override
  String get settingsTest7Zip => 'Test 7-Zip';

  @override
  String get settingsTest7ZipDesc => 'Test if 7-Zip can be found in the system';

  @override
  String get settingsExecutableRecognition => 'Executable Recognition';

  @override
  String get settingsMaxSearchDepth => 'Max Search Depth';

  @override
  String get settingsInvalidSearchDepth => 'Invalid search depth value.';

  @override
  String get settingsExecutableExtensions => 'Executable Extensions';

  @override
  String get settingsInvalidExtensions => 'Invalid extensions list.';

  @override
  String get settingsNoExtensionsSelected => 'No extensions selected';

  @override
  String get settingsSelectExtensions => 'Select Extensions';

  @override
  String get settingsSaveExecutableSettings => 'Save Settings';

  @override
  String get settingsExecutableSettingsSaved =>
      'Executable recognition settings saved.';

  @override
  String get settingsAppearance => 'Appearance';

  @override
  String get settingsAppFont => 'Application Font';

  @override
  String get settingsLanguage => 'Language';

  @override
  String get settingsSaveLanguage => 'Save Language';

  @override
  String get settingsLanguageSaved => 'Language setting saved.';

  @override
  String get translationByAI => '(AI translated)';

  @override
  String get settingsLogSettings => 'Log Settings';

  @override
  String get settingsEnableFileLogging => 'Enable File Logging';

  @override
  String get settingsEnableFileLoggingDesc =>
      'Save logs to a file for debugging';

  @override
  String get settingsLogFilePath => 'Log File';

  @override
  String get settingsInvalidLogPath => 'Invalid log file path.';

  @override
  String get settingsLogSettingsSaved => 'Log settings saved.';

  @override
  String get settingsAbout => 'About';

  @override
  String get settingsVersion => 'Version';

  @override
  String get settingsBuildTime => 'Build Time';

  @override
  String get settingsCheckUpdate => 'Check Update';

  @override
  String get settingsReinstallConfirm => 'Reinstall Application';

  @override
  String get settingsReinstallMessage =>
      'This will download and reinstall the latest version. Are you sure?';

  @override
  String get settingsReinstallConfirmButton => 'Reinstall';

  @override
  String get settingsReinstallFailed => 'Reinstall Failed';

  @override
  String settingsReinstallFailedMessage(String message) {
    return 'Failed to reinstall: $message';
  }

  @override
  String get settingsReinstallLatest => 'Reinstall Latest Version';

  @override
  String get settingsDevHint => 'Development Stage';

  @override
  String get settingsDevHintContent =>
      'If you encounter any issues, try deleting the configuration file and restart.';

  @override
  String get settingsUnsavedChanges => 'Unsaved Changes';

  @override
  String get settingsUnsavedChangesMessage =>
      'You have unsaved changes. Do you want to save them before exiting?';

  @override
  String get settingsDiscardChanges => 'Discard';

  @override
  String get settingsSaveAndExit => 'Save and Exit';

  @override
  String get settingsSaveAll => 'Save All';

  @override
  String get test7ZipTitle => '7-Zip Test';

  @override
  String get test7ZipClose => 'Close';

  @override
  String get homeCopySuffix => ' - Copy';

  @override
  String get homeNewSoftware => 'New Software ';

  @override
  String homeDuplicateExisting(String name, String path) {
    return 'Managed software: $name ($path)';
  }

  @override
  String homeDuplicateInstallDir(String path) {
    return 'Installation directory already exists: $path';
  }

  @override
  String homeDuplicateArchive(String name) {
    return 'Archive file already exists: $name';
  }

  @override
  String get homeDuplicateDetected => 'Duplicate Detected';

  @override
  String homeDuplicateHelp(String name) {
    return 'The software name \'$name\' conflicts with existing items. Please choose how to proceed:';
  }

  @override
  String get homeDuplicateRenameHint => 'Enter a new name to rename and add:';

  @override
  String get homeDuplicateCancel => 'Cancel';

  @override
  String get homeDuplicateRenameAdd => 'Rename and Add';

  @override
  String get homeDuplicateOverwrite => 'Overwrite';

  @override
  String get homeInstallDirNotExistNoArchive =>
      'Install Directory Not Configured';

  @override
  String get homeScanArchiveTitle => 'Scan and Archive';

  @override
  String homeScanArchiveHint(String path) {
    return 'Scan all subdirectories in \'$path\' and archive them as individual software?';
  }

  @override
  String homeScanArchiveBackupOn(String path) {
    return 'Archives will be saved to: $path';
  }

  @override
  String get homeScanArchiveBackupOff => 'Backup will not be created.';

  @override
  String get homeScanArchiveToggle => 'Create backup for original software';

  @override
  String get homeScanArchiveToggleHint =>
      'If enabled, a backup will be created in the backup folder before archiving.';

  @override
  String get homeScanArchiveStart => 'Start Scan';

  @override
  String get homeScanScanning => 'Scanning...';

  @override
  String homeScanProgress(String done, String total, String percent) {
    return '$done / $total ($percent%)';
  }

  @override
  String homeScanCurrent(String name) {
    return 'Current: $name';
  }

  @override
  String get homeScanSummaryTitle => 'Scan Complete';

  @override
  String homeScanSummaryInstallDir(String path) {
    return 'Installation directory: $path';
  }

  @override
  String homeScanSummaryArchiveDir(String path) {
    return 'Archive directory: $path';
  }

  @override
  String homeScanSummaryTotal(String count) {
    return 'Total scanned: $count';
  }

  @override
  String homeScanSummaryArchived(String count) {
    return 'Successfully archived: $count';
  }

  @override
  String homeScanSummarySkipped(String count) {
    return 'Skipped (already exists): $count';
  }

  @override
  String homeScanSummaryFailed(String count) {
    return 'Failed: $count';
  }

  @override
  String homeScanSummaryFailedItem(String name, String error) {
    return '- $name: $error';
  }

  @override
  String homeScanSummaryMore(String count) {
    return 'And $count more failures...';
  }

  @override
  String homeAssocFound(String count) {
    return 'Found $count Archive Associations';
  }

  @override
  String get homeAssocHint =>
      'The following installations may have corresponding archives. Please select which ones to associate:';

  @override
  String get homeAssocSkipAll => 'Skip All';

  @override
  String get homeAssocApply => 'Apply Associations';

  @override
  String get homeProcessingFiles => 'Processing files, please wait...';

  @override
  String get homeInstallingSoftware => 'Installing software...';

  @override
  String get homeCancellingOperation => 'Cancelling operation...';

  @override
  String homeDeleteSoftware(String name) {
    return 'Delete $name';
  }

  @override
  String homeDeleteUnmanagedHint(String name) {
    return 'Are you sure you want to delete this $name? This action cannot be undone.';
  }

  @override
  String get homeDeleteUnmanagedFolder => 'unknown folder';

  @override
  String get homeDeleteUnmanagedArchive => 'unknown archive file';

  @override
  String get homeDeleteManagedHint =>
      'Are you sure you want to delete this software? This will remove it from the list.';

  @override
  String get homeDeleteInstallDir => 'Also delete installation directory';

  @override
  String get homeDeleteArchive => 'Also delete archive file';

  @override
  String get homeDeleting => 'Deleting, please wait...';

  @override
  String get homeManagedSoftware => 'Managed Software';

  @override
  String get homeListView => 'List View';

  @override
  String get homeGridView => 'Grid View';

  @override
  String get homeReorder => 'Reorder';

  @override
  String get homeOpenInstallDir => 'Open Installation Directory';

  @override
  String get homeScanCurrentDir => 'Scan Current Directory for Software';

  @override
  String get homeNoSoftware => 'No software added yet.';

  @override
  String get homeDropToAdd => 'Release to add software';

  @override
  String get homeDropHint =>
      'Supports ZIP/TAR/RAR/7Z archives or configured executable files';

  @override
  String get homeReorderTitle => 'Adjust Software Order';

  @override
  String get homeMoveUp => 'Move Up';

  @override
  String get homeMoveDown => 'Move Down';

  @override
  String get homeReorderHint =>
      'Click the arrows on the right to adjust the display order. Changes will take effect on the home page after saving.';

  @override
  String get homeSaveOrder => 'Save Order';

  @override
  String test7ZipError(String error) {
    return 'An error occurred during testing:\n$error';
  }

  @override
  String errorUnhandled(String error, String stackTrace) {
    return 'An unhandled error occurred:\n\n$error\n\nStack trace:\n$stackTrace';
  }

  @override
  String get errorOk => 'OK';
}
