import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('zh'),
  ];

  /// No description provided for @appName.
  ///
  /// In en, this message translates to:
  /// **'CarryDock'**
  String get appName;

  /// No description provided for @appNameShort.
  ///
  /// In en, this message translates to:
  /// **'CarryDock'**
  String get appNameShort;

  /// No description provided for @loading.
  ///
  /// In en, this message translates to:
  /// **'Loading...'**
  String get loading;

  /// No description provided for @unknown.
  ///
  /// In en, this message translates to:
  /// **'Unknown'**
  String get unknown;

  /// No description provided for @success.
  ///
  /// In en, this message translates to:
  /// **'Success'**
  String get success;

  /// No description provided for @error.
  ///
  /// In en, this message translates to:
  /// **'Error'**
  String get error;

  /// No description provided for @warning.
  ///
  /// In en, this message translates to:
  /// **'Warning'**
  String get warning;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @delete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get delete;

  /// No description provided for @edit.
  ///
  /// In en, this message translates to:
  /// **'Edit'**
  String get edit;

  /// No description provided for @confirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get confirm;

  /// No description provided for @close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// No description provided for @ok.
  ///
  /// In en, this message translates to:
  /// **'OK'**
  String get ok;

  /// No description provided for @yes.
  ///
  /// In en, this message translates to:
  /// **'Yes'**
  String get yes;

  /// No description provided for @no.
  ///
  /// In en, this message translates to:
  /// **'No'**
  String get no;

  /// No description provided for @retry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get retry;

  /// No description provided for @skip.
  ///
  /// In en, this message translates to:
  /// **'Skip'**
  String get skip;

  /// No description provided for @back.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get back;

  /// No description provided for @next.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get next;

  /// No description provided for @finish.
  ///
  /// In en, this message translates to:
  /// **'Finish'**
  String get finish;

  /// No description provided for @apply.
  ///
  /// In en, this message translates to:
  /// **'Apply'**
  String get apply;

  /// No description provided for @reset.
  ///
  /// In en, this message translates to:
  /// **'Reset'**
  String get reset;

  /// No description provided for @clear.
  ///
  /// In en, this message translates to:
  /// **'Clear'**
  String get clear;

  /// No description provided for @search.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get search;

  /// No description provided for @refresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get refresh;

  /// No description provided for @open.
  ///
  /// In en, this message translates to:
  /// **'Open'**
  String get open;

  /// No description provided for @browse.
  ///
  /// In en, this message translates to:
  /// **'Browse'**
  String get browse;

  /// No description provided for @select.
  ///
  /// In en, this message translates to:
  /// **'Select'**
  String get select;

  /// No description provided for @add.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get add;

  /// No description provided for @remove.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get remove;

  /// No description provided for @navHome.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get navHome;

  /// No description provided for @navArchive.
  ///
  /// In en, this message translates to:
  /// **'Archive Manager'**
  String get navArchive;

  /// No description provided for @navSettings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get navSettings;

  /// No description provided for @navDeveloperOptions.
  ///
  /// In en, this message translates to:
  /// **'Developer Options'**
  String get navDeveloperOptions;

  /// No description provided for @navToggleTooltip.
  ///
  /// In en, this message translates to:
  /// **'Collapse navigation pane'**
  String get navToggleTooltip;

  /// No description provided for @navExpandTooltip.
  ///
  /// In en, this message translates to:
  /// **'Expand navigation pane'**
  String get navExpandTooltip;

  /// No description provided for @settingsTitle.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTitle;

  /// No description provided for @homeTitle.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get homeTitle;

  /// No description provided for @languageChinese.
  ///
  /// In en, this message translates to:
  /// **'简体中文'**
  String get languageChinese;

  /// No description provided for @languageEnglish.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get languageEnglish;

  /// No description provided for @tileOpenFolder.
  ///
  /// In en, this message translates to:
  /// **'Open Software Folder'**
  String get tileOpenFolder;

  /// No description provided for @tileChangeFolder.
  ///
  /// In en, this message translates to:
  /// **'Change Software Folder'**
  String get tileChangeFolder;

  /// No description provided for @tileCreateBackup.
  ///
  /// In en, this message translates to:
  /// **'Create Backup'**
  String get tileCreateBackup;

  /// No description provided for @tileAddToStartMenu.
  ///
  /// In en, this message translates to:
  /// **'Add to Start Menu'**
  String get tileAddToStartMenu;

  /// No description provided for @tileOpenArchiveFolder.
  ///
  /// In en, this message translates to:
  /// **'Open Archive Folder'**
  String get tileOpenArchiveFolder;

  /// No description provided for @tileLaunchOther.
  ///
  /// In en, this message translates to:
  /// **'Launch Other Program'**
  String get tileLaunchOther;

  /// No description provided for @tileArchiveExists.
  ///
  /// In en, this message translates to:
  /// **'Archive file exists'**
  String get tileArchiveExists;

  /// No description provided for @tileArchiveNotExists.
  ///
  /// In en, this message translates to:
  /// **'Archive file does not exist'**
  String get tileArchiveNotExists;

  /// No description provided for @tileBackupDetected.
  ///
  /// In en, this message translates to:
  /// **'Backup detected'**
  String get tileBackupDetected;

  /// No description provided for @tileBackupNotDetected.
  ///
  /// In en, this message translates to:
  /// **'No backup detected'**
  String get tileBackupNotDetected;

  /// No description provided for @tileHintNoInstallDir.
  ///
  /// In en, this message translates to:
  /// **'Hint'**
  String get tileHintNoInstallDir;

  /// No description provided for @tileHintNoInstallDirMessage.
  ///
  /// In en, this message translates to:
  /// **'This software has no installation directory configured.'**
  String get tileHintNoInstallDirMessage;

  /// No description provided for @tileHintCannotOpenDir.
  ///
  /// In en, this message translates to:
  /// **'Cannot find installation directory: {path}'**
  String tileHintCannotOpenDir(Object path);

  /// No description provided for @tileError.
  ///
  /// In en, this message translates to:
  /// **'Error'**
  String get tileError;

  /// No description provided for @tileErrorCannotOpenDirMessage.
  ///
  /// In en, this message translates to:
  /// **'Cannot open installation directory, please try again later.'**
  String get tileErrorCannotOpenDirMessage;

  /// No description provided for @tileHintNoArchive.
  ///
  /// In en, this message translates to:
  /// **'Hint'**
  String get tileHintNoArchive;

  /// No description provided for @tileHintNoArchiveMessage.
  ///
  /// In en, this message translates to:
  /// **'This software has no archive file configured.'**
  String get tileHintNoArchiveMessage;

  /// No description provided for @tileHintArchiveNotExists.
  ///
  /// In en, this message translates to:
  /// **'Archive file does not exist, opened archive directory.'**
  String get tileHintArchiveNotExists;

  /// No description provided for @tileHintParentOpened.
  ///
  /// In en, this message translates to:
  /// **'Archive file does not exist, opened archive parent directory.'**
  String get tileHintParentOpened;

  /// No description provided for @tileHintArchiveNotFound.
  ///
  /// In en, this message translates to:
  /// **'Cannot find archive file location.'**
  String get tileHintArchiveNotFound;

  /// No description provided for @tileErrorCannotOpenArchive.
  ///
  /// In en, this message translates to:
  /// **'Cannot open archive directory, please try again later.'**
  String get tileErrorCannotOpenArchive;

  /// No description provided for @tileHintNoExecutable.
  ///
  /// In en, this message translates to:
  /// **'Hint'**
  String get tileHintNoExecutable;

  /// No description provided for @tileHintNoExecutableMessage.
  ///
  /// In en, this message translates to:
  /// **'This software has no executable file path configured.'**
  String get tileHintNoExecutableMessage;

  /// No description provided for @tileHintInstallDirNotExists.
  ///
  /// In en, this message translates to:
  /// **'Installation directory does not exist, cannot create backup.'**
  String get tileHintInstallDirNotExists;

  /// No description provided for @tileCreatingBackup.
  ///
  /// In en, this message translates to:
  /// **'Creating backup...'**
  String get tileCreatingBackup;

  /// No description provided for @tileBackupCreated.
  ///
  /// In en, this message translates to:
  /// **'Backup created: {name}'**
  String tileBackupCreated(Object name);

  /// No description provided for @tileErrorCreateBackup.
  ///
  /// In en, this message translates to:
  /// **'Failed to create backup, please try again later.'**
  String get tileErrorCreateBackup;

  /// No description provided for @tileSuccess.
  ///
  /// In en, this message translates to:
  /// **'Success'**
  String get tileSuccess;

  /// No description provided for @tileAddedToStartMenu.
  ///
  /// In en, this message translates to:
  /// **'Added \"{name}\" to Start Menu.'**
  String tileAddedToStartMenu(Object name);

  /// No description provided for @tileErrorAddToStartMenu.
  ///
  /// In en, this message translates to:
  /// **'Failed to add to Start Menu: {error}'**
  String tileErrorAddToStartMenu(Object error);

  /// No description provided for @tileCarryDock.
  ///
  /// In en, this message translates to:
  /// **'CarryDock'**
  String get tileCarryDock;

  /// No description provided for @tileCarryDockShortcutAdding.
  ///
  /// In en, this message translates to:
  /// **'CarryDock shortcut does not exist in Start Menu, adding...'**
  String get tileCarryDockShortcutAdding;

  /// No description provided for @tileCarryDockShortcutAdded.
  ///
  /// In en, this message translates to:
  /// **'CarryDock shortcut added to Start Menu'**
  String get tileCarryDockShortcutAdded;

  /// No description provided for @homeUnknownFolder.
  ///
  /// In en, this message translates to:
  /// **'Unknown Folder'**
  String get homeUnknownFolder;

  /// No description provided for @homeUnknownArchiveFile.
  ///
  /// In en, this message translates to:
  /// **'Unknown Archive File'**
  String get homeUnknownArchiveFile;

  /// No description provided for @homeSoftwareDirDeleted.
  ///
  /// In en, this message translates to:
  /// **'Software directory deleted'**
  String get homeSoftwareDirDeleted;

  /// No description provided for @homeRehost.
  ///
  /// In en, this message translates to:
  /// **'Rehost'**
  String get homeRehost;

  /// No description provided for @homeBackupArchive.
  ///
  /// In en, this message translates to:
  /// **'Backup Archive'**
  String get homeBackupArchive;

  /// No description provided for @homeArchiveFile.
  ///
  /// In en, this message translates to:
  /// **'Archive File'**
  String get homeArchiveFile;

  /// No description provided for @homeChangeExecutable.
  ///
  /// In en, this message translates to:
  /// **'Change Main Program'**
  String get homeChangeExecutable;

  /// No description provided for @homeScanning.
  ///
  /// In en, this message translates to:
  /// **'Scanning for executable programs...'**
  String get homeScanning;

  /// No description provided for @homeNoExecutableFound.
  ///
  /// In en, this message translates to:
  /// **'No executable programs found in installation directory'**
  String get homeNoExecutableFound;

  /// No description provided for @homeOnlyOneExecutableFound.
  ///
  /// In en, this message translates to:
  /// **'Only one executable program found, cannot change main program'**
  String get homeOnlyOneExecutableFound;

  /// No description provided for @tileNewPathHint.
  ///
  /// In en, this message translates to:
  /// **'New path will be: {path}\\[your input name]'**
  String tileNewPathHint(Object path);

  /// No description provided for @tileErrorMigrateFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to change software folder: {error}'**
  String tileErrorMigrateFailed(Object error);

  /// No description provided for @tileChangeSoftwareFolder.
  ///
  /// In en, this message translates to:
  /// **'Change Software Folder'**
  String get tileChangeSoftwareFolder;

  /// No description provided for @tileCurrentPath.
  ///
  /// In en, this message translates to:
  /// **'Current path:'**
  String get tileCurrentPath;

  /// No description provided for @tileNewFolderName.
  ///
  /// In en, this message translates to:
  /// **'New folder name:'**
  String get tileNewFolderName;

  /// No description provided for @tileNewFolderPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Enter new folder name'**
  String get tileNewFolderPlaceholder;

  /// No description provided for @tileRenameConfirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get tileRenameConfirm;

  /// No description provided for @tileRenameCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get tileRenameCancel;

  /// No description provided for @tileHintSamePath.
  ///
  /// In en, this message translates to:
  /// **'Hint'**
  String get tileHintSamePath;

  /// No description provided for @tileHintSamePathMessage.
  ///
  /// In en, this message translates to:
  /// **'New path is the same as current path, no update needed.'**
  String get tileHintSamePathMessage;

  /// No description provided for @tileErrorFolderNotExist.
  ///
  /// In en, this message translates to:
  /// **'Current software folder does not exist, cannot migrate.'**
  String get tileErrorFolderNotExist;

  /// No description provided for @tileErrorFolderExists.
  ///
  /// In en, this message translates to:
  /// **'New folder already exists, please choose another name.'**
  String get tileErrorFolderExists;

  /// No description provided for @tileMigrating.
  ///
  /// In en, this message translates to:
  /// **'Migrating files...'**
  String get tileMigrating;

  /// No description provided for @tileMigrateSuccess.
  ///
  /// In en, this message translates to:
  /// **'Software folder migrated successfully.'**
  String get tileMigrateSuccess;

  /// No description provided for @archiveTitle.
  ///
  /// In en, this message translates to:
  /// **'Archive Manager'**
  String get archiveTitle;

  /// No description provided for @archiveFiles.
  ///
  /// In en, this message translates to:
  /// **'Archives'**
  String get archiveFiles;

  /// No description provided for @archiveNoArchiveFiles.
  ///
  /// In en, this message translates to:
  /// **'No archive files'**
  String get archiveNoArchiveFiles;

  /// No description provided for @archiveBackupFiles.
  ///
  /// In en, this message translates to:
  /// **'Backups'**
  String get archiveBackupFiles;

  /// No description provided for @archiveNoBackupFiles.
  ///
  /// In en, this message translates to:
  /// **'No backup files'**
  String get archiveNoBackupFiles;

  /// No description provided for @archiveRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get archiveRefresh;

  /// No description provided for @archiveOpenArchiveDir.
  ///
  /// In en, this message translates to:
  /// **'Open Archive Directory'**
  String get archiveOpenArchiveDir;

  /// No description provided for @archiveOpenBackupDir.
  ///
  /// In en, this message translates to:
  /// **'Open Backup Directory'**
  String get archiveOpenBackupDir;

  /// No description provided for @archiveOperationFailed.
  ///
  /// In en, this message translates to:
  /// **'Operation Failed'**
  String get archiveOperationFailed;

  /// No description provided for @archiveCannotOpenDirectory.
  ///
  /// In en, this message translates to:
  /// **'Cannot open directory.'**
  String get archiveCannotOpenDirectory;

  /// No description provided for @archiveCannotOpenExplorer.
  ///
  /// In en, this message translates to:
  /// **'Cannot open Explorer.'**
  String get archiveCannotOpenExplorer;

  /// No description provided for @archiveInstallingFromArchive.
  ///
  /// In en, this message translates to:
  /// **'Installing from archive...'**
  String get archiveInstallingFromArchive;

  /// No description provided for @archiveCompletingInstallation.
  ///
  /// In en, this message translates to:
  /// **'Completing installation...'**
  String get archiveCompletingInstallation;

  /// No description provided for @archiveCancellingOperation.
  ///
  /// In en, this message translates to:
  /// **'Cancelling operation...'**
  String get archiveCancellingOperation;

  /// No description provided for @archiveDuplicateDetected.
  ///
  /// In en, this message translates to:
  /// **'Duplicate Detected'**
  String get archiveDuplicateDetected;

  /// No description provided for @archiveDuplicateMessage.
  ///
  /// In en, this message translates to:
  /// **'Installation directory already exists:\n'**
  String get archiveDuplicateMessage;

  /// No description provided for @archiveBackupAndRestore.
  ///
  /// In en, this message translates to:
  /// **'Backup and Restore'**
  String get archiveBackupAndRestore;

  /// No description provided for @archiveRestoring.
  ///
  /// In en, this message translates to:
  /// **'Restoring...'**
  String get archiveRestoring;

  /// No description provided for @archiveBackupRestoreFailed.
  ///
  /// In en, this message translates to:
  /// **'Backup and restore failed.'**
  String get archiveBackupRestoreFailed;

  /// No description provided for @archiveInstallFailed.
  ///
  /// In en, this message translates to:
  /// **'Installation failed.'**
  String get archiveInstallFailed;

  /// No description provided for @archiveDeleteArchive.
  ///
  /// In en, this message translates to:
  /// **'Delete Archive'**
  String get archiveDeleteArchive;

  /// No description provided for @archiveDeleteBackup.
  ///
  /// In en, this message translates to:
  /// **'Delete Backup'**
  String get archiveDeleteBackup;

  /// No description provided for @archiveDeleteArchiveHint.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to delete this archive?'**
  String get archiveDeleteArchiveHint;

  /// No description provided for @archiveDeleteBackupHint.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to delete this backup? It may be linked to managed software.'**
  String get archiveDeleteBackupHint;

  /// No description provided for @archiveDeletingArchive.
  ///
  /// In en, this message translates to:
  /// **'Deleting archive...'**
  String get archiveDeletingArchive;

  /// No description provided for @archiveDeletingBackup.
  ///
  /// In en, this message translates to:
  /// **'Deleting backup...'**
  String get archiveDeletingBackup;

  /// No description provided for @archiveDeleteFailed.
  ///
  /// In en, this message translates to:
  /// **'Deletion Failed'**
  String get archiveDeleteFailed;

  /// No description provided for @archiveCannotDeleteFile.
  ///
  /// In en, this message translates to:
  /// **'Cannot delete file.'**
  String get archiveCannotDeleteFile;

  /// No description provided for @archiveNoAvailableSoftware.
  ///
  /// In en, this message translates to:
  /// **'No Available Software'**
  String get archiveNoAvailableSoftware;

  /// No description provided for @archiveNoAvailableSoftwareHint.
  ///
  /// In en, this message translates to:
  /// **'No managed software available. Please add software first.'**
  String get archiveNoAvailableSoftwareHint;

  /// No description provided for @archiveLinkArchive.
  ///
  /// In en, this message translates to:
  /// **'Link Archive'**
  String get archiveLinkArchive;

  /// No description provided for @archiveLinkBackup.
  ///
  /// In en, this message translates to:
  /// **'Link Backup'**
  String get archiveLinkBackup;

  /// No description provided for @archiveLink.
  ///
  /// In en, this message translates to:
  /// **'Link'**
  String get archiveLink;

  /// No description provided for @archiveLinkSuccess.
  ///
  /// In en, this message translates to:
  /// **'Link Successful'**
  String get archiveLinkSuccess;

  /// No description provided for @archiveLinkSuccessHint.
  ///
  /// In en, this message translates to:
  /// **'Archive has been linked to software.'**
  String get archiveLinkSuccessHint;

  /// No description provided for @archiveManualLink.
  ///
  /// In en, this message translates to:
  /// **'Manual Link'**
  String get archiveManualLink;

  /// No description provided for @archiveRestore.
  ///
  /// In en, this message translates to:
  /// **'Restore'**
  String get archiveRestore;

  /// No description provided for @archiveCreateBackup.
  ///
  /// In en, this message translates to:
  /// **'Create Backup'**
  String get archiveCreateBackup;

  /// No description provided for @archiveCreatingBackup.
  ///
  /// In en, this message translates to:
  /// **'Creating backup...'**
  String get archiveCreatingBackup;

  /// No description provided for @archiveCreateSuccess.
  ///
  /// In en, this message translates to:
  /// **'Success'**
  String get archiveCreateSuccess;

  /// No description provided for @archiveBackupCreated.
  ///
  /// In en, this message translates to:
  /// **'Backup created: '**
  String get archiveBackupCreated;

  /// No description provided for @archiveCreateFailed.
  ///
  /// In en, this message translates to:
  /// **'Creation Failed'**
  String get archiveCreateFailed;

  /// No description provided for @archiveCreateBackupFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to create backup.'**
  String get archiveCreateBackupFailed;

  /// No description provided for @devTitle.
  ///
  /// In en, this message translates to:
  /// **'Developer Options'**
  String get devTitle;

  /// No description provided for @devHint.
  ///
  /// In en, this message translates to:
  /// **'This is a hidden menu for debugging and testing purposes.'**
  String get devHint;

  /// No description provided for @devIconTest.
  ///
  /// In en, this message translates to:
  /// **'Icon Test'**
  String get devIconTest;

  /// No description provided for @devIconTestHint.
  ///
  /// In en, this message translates to:
  /// **'Test icon extraction from executable files'**
  String get devIconTestHint;

  /// No description provided for @devExtractFailed.
  ///
  /// In en, this message translates to:
  /// **'Extraction Failed'**
  String get devExtractFailed;

  /// No description provided for @devExtractFailedMessage.
  ///
  /// In en, this message translates to:
  /// **'Failed to extract icon from file.'**
  String get devExtractFailedMessage;

  /// No description provided for @devFileName.
  ///
  /// In en, this message translates to:
  /// **'File: '**
  String get devFileName;

  /// No description provided for @devMethod1.
  ///
  /// In en, this message translates to:
  /// **'Method 1: Direct Memory Display'**
  String get devMethod1;

  /// No description provided for @devMethod1Desc.
  ///
  /// In en, this message translates to:
  /// **'Display icon directly from memory bytes (32x32)'**
  String get devMethod1Desc;

  /// No description provided for @devMethod2.
  ///
  /// In en, this message translates to:
  /// **'Method 2: High Quality Display'**
  String get devMethod2;

  /// No description provided for @devMethod2Desc.
  ///
  /// In en, this message translates to:
  /// **'Display icon with higher quality settings (64x64)'**
  String get devMethod2Desc;

  /// No description provided for @devMethod3.
  ///
  /// In en, this message translates to:
  /// **'Method 3: ImageProvider Integration'**
  String get devMethod3;

  /// No description provided for @devMethod3Desc.
  ///
  /// In en, this message translates to:
  /// **'Test icon as ImageProvider in widgets'**
  String get devMethod3Desc;

  /// No description provided for @devButtonWithIcon.
  ///
  /// In en, this message translates to:
  /// **'Button'**
  String get devButtonWithIcon;

  /// No description provided for @devIconHint.
  ///
  /// In en, this message translates to:
  /// **'The icon above is rendered using the extracted bytes from the executable file.'**
  String get devIconHint;

  /// No description provided for @devClose.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get devClose;

  /// No description provided for @devHide.
  ///
  /// In en, this message translates to:
  /// **'Hide Developer Options'**
  String get devHide;

  /// No description provided for @devHiddenFeature.
  ///
  /// In en, this message translates to:
  /// **'Hidden Feature'**
  String get devHiddenFeature;

  /// No description provided for @devHiddenFeatureMessage.
  ///
  /// In en, this message translates to:
  /// **'This is a hidden feature for future development.'**
  String get devHiddenFeatureMessage;

  /// No description provided for @devHiddenButton.
  ///
  /// In en, this message translates to:
  /// **'Hidden'**
  String get devHiddenButton;

  /// No description provided for @dialogSelectExecutable.
  ///
  /// In en, this message translates to:
  /// **'Select Executable'**
  String get dialogSelectExecutable;

  /// No description provided for @dialogCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get dialogCancel;

  /// No description provided for @dialogConfirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get dialogConfirm;

  /// No description provided for @dialogUpdateInfo.
  ///
  /// In en, this message translates to:
  /// **'Update Info'**
  String get dialogUpdateInfo;

  /// No description provided for @dialogNoUpdateInfo.
  ///
  /// In en, this message translates to:
  /// **'No update information available.'**
  String get dialogNoUpdateInfo;

  /// No description provided for @dialogClose.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get dialogClose;

  /// No description provided for @dialogNewVersion.
  ///
  /// In en, this message translates to:
  /// **'New Version Available: {version}'**
  String dialogNewVersion(String version);

  /// No description provided for @dialogUpdateContent.
  ///
  /// In en, this message translates to:
  /// **'Update content'**
  String get dialogUpdateContent;

  /// No description provided for @dialogRetry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get dialogRetry;

  /// No description provided for @dialogUpdateSuccess.
  ///
  /// In en, this message translates to:
  /// **'Update Successful'**
  String get dialogUpdateSuccess;

  /// No description provided for @dialogUpdateSuccessMessage.
  ///
  /// In en, this message translates to:
  /// **'Application has been updated successfully.'**
  String get dialogUpdateSuccessMessage;

  /// No description provided for @dialogDownloadAndUpdate.
  ///
  /// In en, this message translates to:
  /// **'Download and Update'**
  String get dialogDownloadAndUpdate;

  /// No description provided for @dialogInvalidExtension.
  ///
  /// In en, this message translates to:
  /// **'Invalid extension format. Must start with \'.\' (e.g. \'.exe\')'**
  String get dialogInvalidExtension;

  /// No description provided for @dialogExtensionExists.
  ///
  /// In en, this message translates to:
  /// **'This extension already exists.'**
  String get dialogExtensionExists;

  /// No description provided for @dialogNoExtensionsSelected.
  ///
  /// In en, this message translates to:
  /// **'No extensions selected'**
  String get dialogNoExtensionsSelected;

  /// No description provided for @dialogSelectExtensions.
  ///
  /// In en, this message translates to:
  /// **'Select Extensions'**
  String get dialogSelectExtensions;

  /// No description provided for @dialogCommonExtensions.
  ///
  /// In en, this message translates to:
  /// **'Common Extensions'**
  String get dialogCommonExtensions;

  /// No description provided for @dialogCustomExtensions.
  ///
  /// In en, this message translates to:
  /// **'Custom Extensions'**
  String get dialogCustomExtensions;

  /// No description provided for @dialogExtensionPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'.exe'**
  String get dialogExtensionPlaceholder;

  /// No description provided for @dialogAdd.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get dialogAdd;

  /// No description provided for @dialogSelected.
  ///
  /// In en, this message translates to:
  /// **'Selected'**
  String get dialogSelected;

  /// No description provided for @dialogSelectAtLeastOne.
  ///
  /// In en, this message translates to:
  /// **'Please select at least one extension.'**
  String get dialogSelectAtLeastOne;

  /// No description provided for @settingsError.
  ///
  /// In en, this message translates to:
  /// **'Error'**
  String get settingsError;

  /// No description provided for @settingsSuccess.
  ///
  /// In en, this message translates to:
  /// **'Success'**
  String get settingsSuccess;

  /// No description provided for @settingsStorage.
  ///
  /// In en, this message translates to:
  /// **'Storage'**
  String get settingsStorage;

  /// No description provided for @settingsInstallPath.
  ///
  /// In en, this message translates to:
  /// **'Installation Path'**
  String get settingsInstallPath;

  /// No description provided for @settingsInstallPathPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Enter installation path'**
  String get settingsInstallPathPlaceholder;

  /// No description provided for @settingsSaveInstallPath.
  ///
  /// In en, this message translates to:
  /// **'Save Path'**
  String get settingsSaveInstallPath;

  /// No description provided for @settingsInstallPathSaved.
  ///
  /// In en, this message translates to:
  /// **'Installation path saved.'**
  String get settingsInstallPathSaved;

  /// No description provided for @settingsArchivePath.
  ///
  /// In en, this message translates to:
  /// **'Archive Path'**
  String get settingsArchivePath;

  /// No description provided for @settingsArchivePathPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Enter archive path'**
  String get settingsArchivePathPlaceholder;

  /// No description provided for @settingsSaveArchivePath.
  ///
  /// In en, this message translates to:
  /// **'Save Path'**
  String get settingsSaveArchivePath;

  /// No description provided for @settingsArchivePathSaved.
  ///
  /// In en, this message translates to:
  /// **'Archive path saved.'**
  String get settingsArchivePathSaved;

  /// No description provided for @settingsConfigFilePath.
  ///
  /// In en, this message translates to:
  /// **'Configuration File'**
  String get settingsConfigFilePath;

  /// No description provided for @settingsOpenFileLocation.
  ///
  /// In en, this message translates to:
  /// **'Open Location'**
  String get settingsOpenFileLocation;

  /// No description provided for @settingsInvalidConfigPath.
  ///
  /// In en, this message translates to:
  /// **'Invalid configuration file path.'**
  String get settingsInvalidConfigPath;

  /// No description provided for @settingsArchiveHandling.
  ///
  /// In en, this message translates to:
  /// **'Archive Handling'**
  String get settingsArchiveHandling;

  /// No description provided for @settingsRemoveNestedFolders.
  ///
  /// In en, this message translates to:
  /// **'Remove Nested Folders'**
  String get settingsRemoveNestedFolders;

  /// No description provided for @settingsRemoveNestedFoldersDesc.
  ///
  /// In en, this message translates to:
  /// **'Remove nested folders when extracting archives'**
  String get settingsRemoveNestedFoldersDesc;

  /// No description provided for @settingsSaveArchiveHandling.
  ///
  /// In en, this message translates to:
  /// **'Save Settings'**
  String get settingsSaveArchiveHandling;

  /// No description provided for @settingsArchiveHandlingSaved.
  ///
  /// In en, this message translates to:
  /// **'Archive handling settings saved.'**
  String get settingsArchiveHandlingSaved;

  /// No description provided for @settingsTest7Zip.
  ///
  /// In en, this message translates to:
  /// **'Test 7-Zip'**
  String get settingsTest7Zip;

  /// No description provided for @settingsTest7ZipDesc.
  ///
  /// In en, this message translates to:
  /// **'Test if 7-Zip can be found in the system'**
  String get settingsTest7ZipDesc;

  /// No description provided for @settingsExecutableRecognition.
  ///
  /// In en, this message translates to:
  /// **'Executable Recognition'**
  String get settingsExecutableRecognition;

  /// No description provided for @settingsMaxSearchDepth.
  ///
  /// In en, this message translates to:
  /// **'Max Search Depth'**
  String get settingsMaxSearchDepth;

  /// No description provided for @settingsInvalidSearchDepth.
  ///
  /// In en, this message translates to:
  /// **'Invalid search depth value.'**
  String get settingsInvalidSearchDepth;

  /// No description provided for @settingsExecutableExtensions.
  ///
  /// In en, this message translates to:
  /// **'Executable Extensions'**
  String get settingsExecutableExtensions;

  /// No description provided for @settingsInvalidExtensions.
  ///
  /// In en, this message translates to:
  /// **'Invalid extensions list.'**
  String get settingsInvalidExtensions;

  /// No description provided for @settingsNoExtensionsSelected.
  ///
  /// In en, this message translates to:
  /// **'No extensions selected'**
  String get settingsNoExtensionsSelected;

  /// No description provided for @settingsSelectExtensions.
  ///
  /// In en, this message translates to:
  /// **'Select Extensions'**
  String get settingsSelectExtensions;

  /// No description provided for @settingsSaveExecutableSettings.
  ///
  /// In en, this message translates to:
  /// **'Save Settings'**
  String get settingsSaveExecutableSettings;

  /// No description provided for @settingsExecutableSettingsSaved.
  ///
  /// In en, this message translates to:
  /// **'Executable recognition settings saved.'**
  String get settingsExecutableSettingsSaved;

  /// No description provided for @settingsAppearance.
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get settingsAppearance;

  /// No description provided for @settingsAppFont.
  ///
  /// In en, this message translates to:
  /// **'Application Font'**
  String get settingsAppFont;

  /// No description provided for @settingsLanguage.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get settingsLanguage;

  /// No description provided for @settingsSaveLanguage.
  ///
  /// In en, this message translates to:
  /// **'Save Language'**
  String get settingsSaveLanguage;

  /// No description provided for @settingsLanguageSaved.
  ///
  /// In en, this message translates to:
  /// **'Language setting saved.'**
  String get settingsLanguageSaved;

  /// No description provided for @translationByAI.
  ///
  /// In en, this message translates to:
  /// **'(AI translated)'**
  String get translationByAI;

  /// No description provided for @settingsLogSettings.
  ///
  /// In en, this message translates to:
  /// **'Log Settings'**
  String get settingsLogSettings;

  /// No description provided for @settingsEnableFileLogging.
  ///
  /// In en, this message translates to:
  /// **'Enable File Logging'**
  String get settingsEnableFileLogging;

  /// No description provided for @settingsEnableFileLoggingDesc.
  ///
  /// In en, this message translates to:
  /// **'Save logs to a file for debugging'**
  String get settingsEnableFileLoggingDesc;

  /// No description provided for @settingsLogFilePath.
  ///
  /// In en, this message translates to:
  /// **'Log File'**
  String get settingsLogFilePath;

  /// No description provided for @settingsInvalidLogPath.
  ///
  /// In en, this message translates to:
  /// **'Invalid log file path.'**
  String get settingsInvalidLogPath;

  /// No description provided for @settingsLogSettingsSaved.
  ///
  /// In en, this message translates to:
  /// **'Log settings saved.'**
  String get settingsLogSettingsSaved;

  /// No description provided for @settingsAbout.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get settingsAbout;

  /// No description provided for @settingsVersion.
  ///
  /// In en, this message translates to:
  /// **'Version'**
  String get settingsVersion;

  /// No description provided for @settingsBuildTime.
  ///
  /// In en, this message translates to:
  /// **'Build Time'**
  String get settingsBuildTime;

  /// No description provided for @settingsCheckUpdate.
  ///
  /// In en, this message translates to:
  /// **'Check Update'**
  String get settingsCheckUpdate;

  /// No description provided for @settingsReinstallConfirm.
  ///
  /// In en, this message translates to:
  /// **'Reinstall Application'**
  String get settingsReinstallConfirm;

  /// No description provided for @settingsReinstallMessage.
  ///
  /// In en, this message translates to:
  /// **'This will download and reinstall the latest version. Are you sure?'**
  String get settingsReinstallMessage;

  /// No description provided for @settingsReinstallConfirmButton.
  ///
  /// In en, this message translates to:
  /// **'Reinstall'**
  String get settingsReinstallConfirmButton;

  /// No description provided for @settingsReinstallFailed.
  ///
  /// In en, this message translates to:
  /// **'Reinstall Failed'**
  String get settingsReinstallFailed;

  /// No description provided for @settingsReinstallFailedMessage.
  ///
  /// In en, this message translates to:
  /// **'Failed to reinstall: {message}'**
  String settingsReinstallFailedMessage(String message);

  /// No description provided for @settingsReinstallLatest.
  ///
  /// In en, this message translates to:
  /// **'Reinstall Latest Version'**
  String get settingsReinstallLatest;

  /// No description provided for @settingsDevHint.
  ///
  /// In en, this message translates to:
  /// **'Development Stage'**
  String get settingsDevHint;

  /// No description provided for @settingsDevHintContent.
  ///
  /// In en, this message translates to:
  /// **'If you encounter any issues, try deleting the configuration file and restart.'**
  String get settingsDevHintContent;

  /// No description provided for @settingsUnsavedChanges.
  ///
  /// In en, this message translates to:
  /// **'Unsaved Changes'**
  String get settingsUnsavedChanges;

  /// No description provided for @settingsUnsavedChangesMessage.
  ///
  /// In en, this message translates to:
  /// **'You have unsaved changes. Do you want to save them before exiting?'**
  String get settingsUnsavedChangesMessage;

  /// No description provided for @settingsDiscardChanges.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get settingsDiscardChanges;

  /// No description provided for @settingsSaveAndExit.
  ///
  /// In en, this message translates to:
  /// **'Save and Exit'**
  String get settingsSaveAndExit;

  /// No description provided for @settingsSaveAll.
  ///
  /// In en, this message translates to:
  /// **'Save All'**
  String get settingsSaveAll;

  /// No description provided for @test7ZipTitle.
  ///
  /// In en, this message translates to:
  /// **'7-Zip Test'**
  String get test7ZipTitle;

  /// No description provided for @test7ZipClose.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get test7ZipClose;

  /// No description provided for @homeCopySuffix.
  ///
  /// In en, this message translates to:
  /// **' - Copy'**
  String get homeCopySuffix;

  /// No description provided for @homeNewSoftware.
  ///
  /// In en, this message translates to:
  /// **'New Software '**
  String get homeNewSoftware;

  /// No description provided for @homeDuplicateExisting.
  ///
  /// In en, this message translates to:
  /// **'Managed software: {name} ({path})'**
  String homeDuplicateExisting(String name, String path);

  /// No description provided for @homeDuplicateInstallDir.
  ///
  /// In en, this message translates to:
  /// **'Installation directory already exists: {path}'**
  String homeDuplicateInstallDir(String path);

  /// No description provided for @homeDuplicateArchive.
  ///
  /// In en, this message translates to:
  /// **'Archive file already exists: {name}'**
  String homeDuplicateArchive(String name);

  /// No description provided for @homeDuplicateDetected.
  ///
  /// In en, this message translates to:
  /// **'Duplicate Detected'**
  String get homeDuplicateDetected;

  /// No description provided for @homeDuplicateHelp.
  ///
  /// In en, this message translates to:
  /// **'The software name \'{name}\' conflicts with existing items. Please choose how to proceed:'**
  String homeDuplicateHelp(String name);

  /// No description provided for @homeDuplicateRenameHint.
  ///
  /// In en, this message translates to:
  /// **'Enter a new name to rename and add:'**
  String get homeDuplicateRenameHint;

  /// No description provided for @homeDuplicateCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get homeDuplicateCancel;

  /// No description provided for @homeDuplicateRenameAdd.
  ///
  /// In en, this message translates to:
  /// **'Rename and Add'**
  String get homeDuplicateRenameAdd;

  /// No description provided for @homeDuplicateOverwrite.
  ///
  /// In en, this message translates to:
  /// **'Overwrite'**
  String get homeDuplicateOverwrite;

  /// No description provided for @homeInstallDirNotExistNoArchive.
  ///
  /// In en, this message translates to:
  /// **'Install Directory Not Configured'**
  String get homeInstallDirNotExistNoArchive;

  /// No description provided for @homeScanArchiveTitle.
  ///
  /// In en, this message translates to:
  /// **'Scan and Archive'**
  String get homeScanArchiveTitle;

  /// No description provided for @homeScanArchiveHint.
  ///
  /// In en, this message translates to:
  /// **'Scan all subdirectories in \'{path}\' and archive them as individual software?'**
  String homeScanArchiveHint(String path);

  /// No description provided for @homeScanArchiveBackupOn.
  ///
  /// In en, this message translates to:
  /// **'Archives will be saved to: {path}'**
  String homeScanArchiveBackupOn(String path);

  /// No description provided for @homeScanArchiveBackupOff.
  ///
  /// In en, this message translates to:
  /// **'Backup will not be created.'**
  String get homeScanArchiveBackupOff;

  /// No description provided for @homeScanArchiveToggle.
  ///
  /// In en, this message translates to:
  /// **'Create backup for original software'**
  String get homeScanArchiveToggle;

  /// No description provided for @homeScanArchiveToggleHint.
  ///
  /// In en, this message translates to:
  /// **'If enabled, a backup will be created in the backup folder before archiving.'**
  String get homeScanArchiveToggleHint;

  /// No description provided for @homeScanArchiveStart.
  ///
  /// In en, this message translates to:
  /// **'Start Scan'**
  String get homeScanArchiveStart;

  /// No description provided for @homeScanScanning.
  ///
  /// In en, this message translates to:
  /// **'Scanning...'**
  String get homeScanScanning;

  /// No description provided for @homeScanProgress.
  ///
  /// In en, this message translates to:
  /// **'{done} / {total} ({percent}%)'**
  String homeScanProgress(String done, String total, String percent);

  /// No description provided for @homeScanCurrent.
  ///
  /// In en, this message translates to:
  /// **'Current: {name}'**
  String homeScanCurrent(String name);

  /// No description provided for @homeScanSummaryTitle.
  ///
  /// In en, this message translates to:
  /// **'Scan Complete'**
  String get homeScanSummaryTitle;

  /// No description provided for @homeScanSummaryInstallDir.
  ///
  /// In en, this message translates to:
  /// **'Installation directory: {path}'**
  String homeScanSummaryInstallDir(String path);

  /// No description provided for @homeScanSummaryArchiveDir.
  ///
  /// In en, this message translates to:
  /// **'Archive directory: {path}'**
  String homeScanSummaryArchiveDir(String path);

  /// No description provided for @homeScanSummaryTotal.
  ///
  /// In en, this message translates to:
  /// **'Total scanned: {count}'**
  String homeScanSummaryTotal(String count);

  /// No description provided for @homeScanSummaryArchived.
  ///
  /// In en, this message translates to:
  /// **'Successfully archived: {count}'**
  String homeScanSummaryArchived(String count);

  /// No description provided for @homeScanSummarySkipped.
  ///
  /// In en, this message translates to:
  /// **'Skipped (already exists): {count}'**
  String homeScanSummarySkipped(String count);

  /// No description provided for @homeScanSummaryFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed: {count}'**
  String homeScanSummaryFailed(String count);

  /// No description provided for @homeScanSummaryFailedItem.
  ///
  /// In en, this message translates to:
  /// **'- {name}: {error}'**
  String homeScanSummaryFailedItem(String name, String error);

  /// No description provided for @homeScanSummaryMore.
  ///
  /// In en, this message translates to:
  /// **'And {count} more failures...'**
  String homeScanSummaryMore(String count);

  /// No description provided for @homeAssocFound.
  ///
  /// In en, this message translates to:
  /// **'Found {count} Archive Associations'**
  String homeAssocFound(String count);

  /// No description provided for @homeAssocHint.
  ///
  /// In en, this message translates to:
  /// **'The following installations may have corresponding archives. Please select which ones to associate:'**
  String get homeAssocHint;

  /// No description provided for @homeAssocSkipAll.
  ///
  /// In en, this message translates to:
  /// **'Skip All'**
  String get homeAssocSkipAll;

  /// No description provided for @homeAssocApply.
  ///
  /// In en, this message translates to:
  /// **'Apply Associations'**
  String get homeAssocApply;

  /// No description provided for @homeProcessingFiles.
  ///
  /// In en, this message translates to:
  /// **'Processing files, please wait...'**
  String get homeProcessingFiles;

  /// No description provided for @homeInstallingSoftware.
  ///
  /// In en, this message translates to:
  /// **'Installing software...'**
  String get homeInstallingSoftware;

  /// No description provided for @homeCancellingOperation.
  ///
  /// In en, this message translates to:
  /// **'Cancelling operation...'**
  String get homeCancellingOperation;

  /// No description provided for @homeDeleteSoftware.
  ///
  /// In en, this message translates to:
  /// **'Delete {name}'**
  String homeDeleteSoftware(String name);

  /// No description provided for @homeDeleteUnmanagedHint.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to delete this {name}? This action cannot be undone.'**
  String homeDeleteUnmanagedHint(String name);

  /// No description provided for @homeDeleteUnmanagedFolder.
  ///
  /// In en, this message translates to:
  /// **'unknown folder'**
  String get homeDeleteUnmanagedFolder;

  /// No description provided for @homeDeleteUnmanagedArchive.
  ///
  /// In en, this message translates to:
  /// **'unknown archive file'**
  String get homeDeleteUnmanagedArchive;

  /// No description provided for @homeDeleteManagedHint.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to delete this software? This will remove it from the list.'**
  String get homeDeleteManagedHint;

  /// No description provided for @homeDeleteInstallDir.
  ///
  /// In en, this message translates to:
  /// **'Also delete installation directory'**
  String get homeDeleteInstallDir;

  /// No description provided for @homeDeleteArchive.
  ///
  /// In en, this message translates to:
  /// **'Also delete archive file'**
  String get homeDeleteArchive;

  /// No description provided for @homeDeleting.
  ///
  /// In en, this message translates to:
  /// **'Deleting, please wait...'**
  String get homeDeleting;

  /// No description provided for @homeManagedSoftware.
  ///
  /// In en, this message translates to:
  /// **'Managed Software'**
  String get homeManagedSoftware;

  /// No description provided for @homeListView.
  ///
  /// In en, this message translates to:
  /// **'List View'**
  String get homeListView;

  /// No description provided for @homeGridView.
  ///
  /// In en, this message translates to:
  /// **'Grid View'**
  String get homeGridView;

  /// No description provided for @homeReorder.
  ///
  /// In en, this message translates to:
  /// **'Reorder'**
  String get homeReorder;

  /// No description provided for @homeOpenInstallDir.
  ///
  /// In en, this message translates to:
  /// **'Open Installation Directory'**
  String get homeOpenInstallDir;

  /// No description provided for @homeScanCurrentDir.
  ///
  /// In en, this message translates to:
  /// **'Scan Current Directory for Software'**
  String get homeScanCurrentDir;

  /// No description provided for @homeNoSoftware.
  ///
  /// In en, this message translates to:
  /// **'No software added yet.'**
  String get homeNoSoftware;

  /// No description provided for @homeDropToAdd.
  ///
  /// In en, this message translates to:
  /// **'Release to add software'**
  String get homeDropToAdd;

  /// No description provided for @homeDropHint.
  ///
  /// In en, this message translates to:
  /// **'Supports ZIP/TAR/RAR/7Z archives or configured executable files'**
  String get homeDropHint;

  /// No description provided for @homeReorderTitle.
  ///
  /// In en, this message translates to:
  /// **'Adjust Software Order'**
  String get homeReorderTitle;

  /// No description provided for @homeMoveUp.
  ///
  /// In en, this message translates to:
  /// **'Move Up'**
  String get homeMoveUp;

  /// No description provided for @homeMoveDown.
  ///
  /// In en, this message translates to:
  /// **'Move Down'**
  String get homeMoveDown;

  /// No description provided for @homeReorderHint.
  ///
  /// In en, this message translates to:
  /// **'Click the arrows on the right to adjust the display order. Changes will take effect on the home page after saving.'**
  String get homeReorderHint;

  /// No description provided for @homeSaveOrder.
  ///
  /// In en, this message translates to:
  /// **'Save Order'**
  String get homeSaveOrder;

  /// No description provided for @test7ZipError.
  ///
  /// In en, this message translates to:
  /// **'An error occurred during testing:\n{error}'**
  String test7ZipError(String error);

  /// No description provided for @errorUnhandled.
  ///
  /// In en, this message translates to:
  /// **'An unhandled error occurred:\n\n{error}\n\nStack trace:\n{stackTrace}'**
  String errorUnhandled(String error, String stackTrace);

  /// No description provided for @errorOk.
  ///
  /// In en, this message translates to:
  /// **'OK'**
  String get errorOk;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
