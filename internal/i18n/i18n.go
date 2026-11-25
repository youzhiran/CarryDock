package i18n

import (
	"fmt"
	"strings"
)

// Lang 表示受支持的界面语言代码。
type Lang string

const (
	LangZhCN Lang = "zh_CN"
	LangEnUS Lang = "en_US"
)

// Normalize 将语言代码规整为 zh_CN 或 en_US。
func Normalize(code string) Lang {
	c := strings.ToLower(strings.TrimSpace(code))
	if strings.HasPrefix(c, "en") {
		return LangEnUS
	}
	return LangZhCN
}

// Strings 保存应用所有界面文案，确保中英文完整覆盖。
type Strings struct {
	AppName      string
	AppNameShort string
	Loading      string
	Unknown      string
	Success      string
	Error        string
	Warning      string
	Cancel       string
	Save         string
	Delete       string
	Edit         string
	Confirm      string
	Close        string
	OK           string
	Yes          string
	No           string
	Retry        string
	Skip         string
	Finish       string
	Apply        string
	Search       string
	Refresh      string
	Open         string
	Browse       string
	Select       string
	Add          string
	Remove       string

	// Navigation
	NavHome             string
	NavArchive          string
	NavBackup           string
	NavSettings         string
	NavDeveloperOptions string
	NavToggleTooltip    string
	NavExpandTooltip    string

	// Languages
	LanguageChinese string
	LanguageEnglish string
	TranslationByAI string

	// Home
	HomeManagedSoftware        string
	HomeUnmanagedFolders       string
	HomeSearchPlaceholder      string
	HomeListView               string
	HomeGridView               string
	HomeReorder                string
	HomeOpenInstallDir         string
	HomeScanCurrentDir         string
	HomeNoSoftware             string
	HomeNoSoftwareHint         string
	HomeConfigureInstallFirst  string
	HomeGoToSettings           string
	HomeDropToAdd              string
	HomeDropHint               string
	HomeUnknownFolder          string
	HomeUnknownArchiveFile     string
	HomeSoftwareDirDeleted     string
	HomeRehost                 string
	HomeBackupArchive          string
	HomeArchiveFile            string
	HomeChangeExecutable       string
	HomeScanning               string
	HomeNoExecutableFound      string
	HomeOnlyOneExecutableFound string
	HomeCopySuffix             string
	HomeNewSoftware            string
	HomeProcessingFiles        string
	HomeInstallingSoftware     string
	HomeCancellingOperation    string
	HomeDeleting               string
	HomeRestoring              string
	HomeMoveUp                 string
	HomeMoveDown               string
	HomeReorderHint            string
	HomeReorderSave            string
	HomeMoreActions            string
	HomeLaunchFailedTitle      string
	HomeCannotLaunchTitle      string
	HomeCannotLaunchMissingDir string
	HomeCannotLaunchNoArchive  string
	HomeCannotLaunchNoExe      string
	HomeUnrecognizedDropTitle  string
	HomeUnrecognizedDropMsg    string
	HomeInstallDirNotConfig    string
	HomeInstallDirNotConfigMsg string
	HomeDirNotExistTitle       string
	HomeCountUnmanagedFmt      string
	FilterAll                  string
	FilterManaged              string
	FilterUnmanaged            string
	FilterLinked               string
	FilterUnlinked             string
	ClearFilter                string
	NoMatchingResults          string
	NoMatchingResultsHint      string
	HomeVersionTooltipFmt      string

	// Tile & Context Menu
	TileOpenFolder           string
	TileChangeFolder         string
	TileCreateBackup         string
	TileAddToStartMenu       string
	TileOpenArchiveFolder    string
	TileLaunchOther          string
	TileArchiveExists        string
	TileArchiveNotExists     string
	TileBackupDetected       string
	TileBackupNotDetected    string
	TileNoInstallDir         string
	TileNoArchive            string
	TileArchiveNotFound      string
	TileParentOpened         string
	TileNoExecutable         string
	TileInstallDirNotExists  string
	TileCreatingBackup       string
	TileBackupCreatedFmt     string
	TileCreateBackupFailed   string
	TileAddedToStartMenuFmt  string
	TileAddToStartMenuErrFmt string
	TileChangeFolderTitle    string
	TileCurrentPath          string
	TileNewFolderName        string
	TileNewFolderPlaceholder string
	TileSamePath             string
	TileFolderNotExist       string
	TileFolderExists         string
	TileMigrating            string
	TileMigrateSuccess       string
	TileMigrateFailedFmt     string

	// Duplicate dialog
	DupDetected           string
	DupHelpFmt            string
	DupExistingFmt        string
	DupInstallDirFmt      string
	DupArchiveFmt         string
	DupRenameHint         string
	DupRenameAdd          string
	DupOverwrite          string
	DupBackupOverwrite    string
	DupCurrentVersionFmt  string
	DupIncomingVersionFmt string
	DupVersionCompareFmt  string
	DupVersionSameSuffix  string
	DupUpdateTitle        string
	DupUpdateHelpFmt      string
	DupReplaceTitle       string
	DupReplaceHelpFmt     string
	DupIncomingFileFmt    string
	DupOldInstallDirFmt   string
	DupVersionOlderSuffix string
	DupKeepBothHint       string
	DupKeepBothAdd        string
	DupReplace            string
	DupBackupReplace      string

	// Delete dialog
	DeleteSoftwareFmt   string
	DeleteUnmanagedFmt  string
	DeleteManagedHint   string
	DeleteInstallDir    string
	DeleteArchive       string
	DeleteBackupHintMsg string

	// Scan & Assoc
	ScanArchiveTitle       string
	ScanArchiveHintFmt     string
	ScanArchiveBackupOnFmt string
	ScanArchiveBackupOff   string
	ScanArchiveToggle      string
	ScanArchiveToggleHint  string
	ScanArchiveStart       string
	ScanScanning           string
	ScanProgressFmt        string
	ScanCurrentFmt         string
	ScanSummaryTitle       string
	ScanSummaryInstallFmt  string
	ScanSummaryArchiveFmt  string
	ScanSummaryTotalFmt    string
	ScanSummaryArchivedFmt string
	ScanSummarySkippedFmt  string
	ScanSummaryFailedFmt   string
	ScanSummaryMoreFmt     string
	AssocFoundFmt          string
	AssocHint              string
	AssocSkipAll           string
	AssocApply             string
	AssocNoMatchKeepBackup string

	// Archive Manager
	ArchiveTitle             string
	BackupTitle              string
	ArchiveFiles             string
	ArchiveNoArchiveFiles    string
	ArchiveNoArchiveHint     string
	ArchiveBackupFiles       string
	ArchiveNoBackupFiles     string
	BackupNoBackupHint       string
	ArchiveOpenArchiveDir    string
	ArchiveOpenBackupDir     string
	ArchiveLinked            string
	ArchiveUnlinked          string
	ArchiveManualLink        string
	ArchiveRestore           string
	ArchiveCreateBackup      string
	ArchiveCreatingBackup    string
	ArchiveBackupCreatedFmt  string
	ArchiveDeleteArchive     string
	ArchiveDeleteBackup      string
	ArchiveDeleteArchiveHint string
	ArchiveDeleteBackupHint  string
	ArchiveNoAvailableSw     string
	ArchiveNoAvailableSwHint string
	ArchiveLinkArchive       string
	ArchiveLinkBackup        string
	ArchiveLinkAction        string
	ArchiveLinkSuccess       string
	ArchiveUnlinkAction      string
	ArchiveUnlinkSuccess     string
	ArchiveDuplicateMsg      string
	ArchiveBackupAndRestore  string
	ArchiveRevealInExplorer  string
	ArchiveSearchPlaceholder string
	BackupSearchPlaceholder  string

	// Select Executable Dialog
	DialogSelectExecutable string
	DialogNoExecutable     string

	// Settings
	SettingsTitle                string
	SettingsDevHint              string
	SettingsDevHintContent       string
	SettingsStorage              string
	SettingsInstallPath          string
	SettingsInstallPathHolder    string
	SettingsSaveInstallPath      string
	SettingsInstallPathSaved     string
	SettingsArchivePath          string
	SettingsArchivePathHolder    string
	SettingsSaveArchivePath      string
	SettingsArchivePathSaved     string
	SettingsConfigFilePath       string
	SettingsOpenFileLocation     string
	SettingsArchiveHandling      string
	SettingsRemoveNested         string
	SettingsRemoveNestedDesc     string
	SettingsSaveArchiveHandling  string
	SettingsArchiveHandlingSaved string
	SettingsTest7Zip             string
	SettingsTest7ZipDesc         string
	SettingsTest7ZipTitle        string
	SettingsExecRecognition      string
	SettingsMaxSearchDepth       string
	SettingsExecExtensions       string
	SettingsNoExtensions         string
	SettingsSelectExtensions     string
	SettingsSaveExecSettings     string
	SettingsExecSettingsSaved    string
	SettingsCommonExtensions     string
	SettingsCustomExtensions     string
	SettingsExtPlaceholder       string
	SettingsExtInvalid           string
	SettingsExtExists            string
	SettingsExtSelectAtLeastOne  string
	SettingsAppearance           string
	SettingsAppFont              string
	SettingsSystemDefaultFont    string
	SettingsPresetFonts          string
	SettingsSystemFonts          string
	SettingsFontUnavailable      string
	SettingsFontDefaultDescFmt   string
	SettingsFontDefaultDescPlain string
	SettingsFontMissingWarnFmt   string
	SettingsFontLoading          string
	SettingsThemeMode            string
	SettingsThemeModeDesc        string
	SettingsThemeModeSystem      string
	SettingsThemeModeLight       string
	SettingsThemeModeDark        string
	SettingsThemeColor           string
	SettingsThemeColorDesc       string
	SettingsThemeColorSystem     string
	SettingsThemeColorCustom     string
	SettingsColorBlue            string
	SettingsColorSky             string
	SettingsColorTeal            string
	SettingsColorGreen           string
	SettingsColorPurple          string
	SettingsColorPink            string
	SettingsColorOrange          string
	SettingsColorRed             string
	SettingsLanguage             string
	SettingsSaveLanguage         string
	SettingsLanguageSaved        string
	SettingsLogSettings          string
	SettingsEnableFileLogging    string
	SettingsEnableFileLoggingMsg string
	SettingsLogLevel             string
	SettingsLogLevelDesc         string
	SettingsLogLevelDebug        string
	SettingsLogLevelInfo         string
	SettingsLogLevelWarn         string
	SettingsLogLevelError        string
	SettingsLogLevelDebugHint    string
	SettingsLogLevelInfoHint     string
	SettingsLogLevelWarnHint     string
	SettingsLogLevelErrorHint    string
	SettingsLogFilePath          string
	SettingsLogSettingsSaved     string
	SettingsAbout                string
	SettingsVersion              string
	SettingsBuildTime            string
	SettingsWebsite              string
	SettingsGitHub               string
	SettingsCheckUpdate          string
	SettingsReinstallLatest      string
	SettingsReinstallConfirm     string
	SettingsReinstallMessage     string
	SettingsReinstallButton      string
	SettingsUnsavedTitle         string
	SettingsUnsavedMessage       string
	SettingsDiscardChanges       string
	SettingsSaveAndContinue      string
	SettingsSaveAll              string
	SettingsDevUnlocked          string
	SettingsUnsavedBadge         string

	// Update Dialog
	UpdateChecking      string
	UpdateUpToDate      string
	UpdateNewVersionFmt string
	UpdateContent       string
	UpdateDownloadBtn   string
	UpdateDownloading   string
	UpdateInstalling    string
	UpdateCancelled     string
	UpdateFailedFmt     string

	// Developer Options
	DevTitle            string
	DevHint             string
	DevIconTest         string
	DevIconTestHint     string
	DevExtractFailed    string
	DevExtractFailedMsg string
	DevFileName         string
	DevFileDesc         string
	DevFileVersion      string
	DevMethod1          string
	DevMethod1Desc      string
	DevMethod2          string
	DevMethod2Desc      string
	DevMethod3          string
	DevMethod3Desc      string
	DevButtonWithIcon   string
	DevIconHint         string
	DevHide             string
	DevHiddenFeature    string
	DevHiddenFeatureMsg string
	DevHiddenButton     string
}

var zhStrings = Strings{
	AppName:      "绿驿管家",
	AppNameShort: "绿驿",
	Loading:      "加载中...",
	Unknown:      "未知",
	Success:      "成功",
	Error:        "错误",
	Warning:      "警告",
	Cancel:       "取消",
	Save:         "保存",
	Delete:       "删除",
	Edit:         "编辑",
	Confirm:      "确认",
	Close:        "关闭",
	OK:           "确定",
	Yes:          "是",
	No:           "否",
	Retry:        "重试",
	Skip:         "跳过",
	Finish:       "完成",
	Apply:        "应用",
	Search:       "搜索",
	Refresh:      "刷新",
	Open:         "打开",
	Browse:       "浏览",
	Select:       "选择",
	Add:          "添加",
	Remove:       "移除",

	NavHome:             "主页",
	NavArchive:          "归档管理",
	NavBackup:           "备份文件",
	NavSettings:         "设置",
	NavDeveloperOptions: "开发者选项",
	NavToggleTooltip:    "折叠导航栏",
	NavExpandTooltip:    "展开导航栏",

	LanguageChinese: "简体中文",
	LanguageEnglish: "English",
	TranslationByAI: "（AI翻译）",

	HomeManagedSoftware:        "已托管软件",
	HomeUnmanagedFolders:       "未托管文件夹",
	HomeSearchPlaceholder:      "搜索软件名称或路径...",
	HomeListView:               "列表视图",
	HomeGridView:               "网格视图",
	HomeReorder:                "调整排序",
	HomeOpenInstallDir:         "打开安装目录",
	HomeScanCurrentDir:         "扫描当前目录软件",
	HomeNoSoftware:             "尚未添加任何软件",
	HomeNoSoftwareHint:         "点击右上角「添加」按钮，或将压缩包 / 可执行文件直接拖入窗口。",
	HomeConfigureInstallFirst:  "尚未配置绿色软件安装目录",
	HomeGoToSettings:           "前往设置",
	HomeDropToAdd:              "松开以添加软件",
	HomeDropHint:               "支持 ZIP / TAR / GZ / BZ2 / XZ / RAR / 7Z 压缩包或已配置的可执行文件",
	HomeUnknownFolder:          "未知文件夹",
	HomeUnknownArchiveFile:     "未知归档文件",
	HomeSoftwareDirDeleted:     "软件目录已删除",
	HomeRehost:                 "重新托管",
	HomeBackupArchive:          "备份归档",
	HomeArchiveFile:            "归档文件",
	HomeChangeExecutable:       "更改主程序",
	HomeScanning:               "正在扫描可执行程序...",
	HomeNoExecutableFound:      "未在安装目录中找到可执行程序",
	HomeOnlyOneExecutableFound: "当前仅发现一个可执行程序",
	HomeCopySuffix:             " - 副本",
	HomeNewSoftware:            "新软件",
	HomeProcessingFiles:        "正在处理文件，请稍候...",
	HomeInstallingSoftware:     "正在安装软件...",
	HomeCancellingOperation:    "正在取消操作...",
	HomeDeleting:               "正在删除，请稍候...",
	HomeRestoring:              "正在从归档/备份恢复，请稍候...",
	HomeMoveUp:                 "上移",
	HomeMoveDown:               "下移",
	HomeReorderHint:            "拖动条目或使用上下按钮调整顺序，点击「完成」保存。",
	HomeReorderSave:            "保存顺序",
	HomeMoreActions:            "更多操作",
	HomeLaunchFailedTitle:      "启动失败",
	HomeCannotLaunchTitle:      "无法启动软件",
	HomeCannotLaunchMissingDir: "安装目录不存在。您可以尝试使用「重新托管」从归档恢复。",
	HomeCannotLaunchNoArchive:  "安装目录不存在，且未找到归档文件。",
	HomeCannotLaunchNoExe:      "请确认该软件已托管并且已配置主程序路径。",
	HomeUnrecognizedDropTitle:  "无法识别的文件",
	HomeUnrecognizedDropMsg:    "请拖放支持的压缩包（ZIP/TAR/GZ/BZ2/XZ/RAR/7Z）或已在设置中允许的可执行文件类型。",
	HomeInstallDirNotConfig:    "安装目录未配置",
	HomeInstallDirNotConfigMsg: "请先在设置页面配置绿色软件安装目录。",
	HomeDirNotExistTitle:       "目录不存在",
	HomeCountUnmanagedFmt:      "%d 个未托管",
	FilterAll:                  "全部",
	FilterManaged:              "已托管",
	FilterUnmanaged:            "未托管",
	FilterLinked:               "已关联",
	FilterUnlinked:             "未关联",
	ClearFilter:                "清除筛选",
	NoMatchingResults:          "没有匹配的项目",
	NoMatchingResultsHint:      "尝试更改搜索关键词或重置当前筛选条件。",
	HomeVersionTooltipFmt:      "软件原生版本：%s",

	TileOpenFolder:           "打开软件文件夹",
	TileChangeFolder:         "修改软件文件夹",
	TileCreateBackup:         "创建备份",
	TileAddToStartMenu:       "添加到开始菜单",
	TileOpenArchiveFolder:    "打开归档文件夹",
	TileLaunchOther:          "启动其他程序",
	TileArchiveExists:        "归档文件存在",
	TileArchiveNotExists:     "归档文件不存在",
	TileBackupDetected:       "检测到备份",
	TileBackupNotDetected:    "未检测到备份",
	TileNoInstallDir:         "该软件未配置安装目录。",
	TileNoArchive:            "该软件未配置归档文件。",
	TileArchiveNotFound:      "找不到归档文件所在位置。",
	TileParentOpened:         "归档文件不存在，已打开归档所在目录。",
	TileNoExecutable:         "该软件未配置可执行文件路径。",
	TileInstallDirNotExists:  "安装目录不存在，无法创建备份。",
	TileCreatingBackup:       "正在创建备份...",
	TileBackupCreatedFmt:     "已创建备份：%s",
	TileCreateBackupFailed:   "创建备份失败。",
	TileAddedToStartMenuFmt:  "已将 \"%s\" 添加到开始菜单。",
	TileAddToStartMenuErrFmt: "添加到开始菜单失败：%s",
	TileChangeFolderTitle:    "修改软件文件夹",
	TileCurrentPath:          "当前路径：",
	TileNewFolderName:        "新文件夹名称：",
	TileNewFolderPlaceholder: "输入新的文件夹名称",
	TileSamePath:             "新路径与当前路径相同，无需更新。",
	TileFolderNotExist:       "当前软件文件夹不存在，无法迁移。",
	TileFolderExists:         "新文件夹已存在，请选择其他名称。",
	TileMigrating:            "正在迁移文件...",
	TileMigrateSuccess:       "软件文件夹已成功迁移。",
	TileMigrateFailedFmt:     "修改软件文件夹失败：%s",

	DupDetected:           "检测到重复",
	DupHelpFmt:            "软件名称「%s」与现有项目冲突。请选择如何继续：",
	DupExistingFmt:        "已托管软件：%s（%s）",
	DupInstallDirFmt:      "安装目录已存在：%s",
	DupArchiveFmt:         "归档文件已存在：%s",
	DupRenameHint:         "输入新名称以重命名并添加：",
	DupRenameAdd:          "重命名并添加",
	DupOverwrite:          "覆盖",
	DupBackupOverwrite:    "备份并覆盖",
	DupCurrentVersionFmt:  "当前软件版本：%s",
	DupIncomingVersionFmt: "待导入文件版本：%s",
	DupVersionCompareFmt:  "版本对比：%s → %s",
	DupVersionSameSuffix:  "（版本相同）",
	DupUpdateTitle:        "发现新版本",
	DupUpdateHelpFmt:      "检测到「%s」的新版本，是否替换当前已安装的旧版本？",
	DupReplaceTitle:       "检测到已安装软件",
	DupReplaceHelpFmt:     "待导入文件与已托管软件「%s」属于同一软件，是否替换现有版本？",
	DupIncomingFileFmt:    "待导入文件：%s",
	DupOldInstallDirFmt:   "原安装目录：%s",
	DupVersionOlderSuffix: "（低于当前版本）",
	DupKeepBothHint:       "如需保留旧版本并作为独立软件添加，可修改名称：",
	DupKeepBothAdd:        "保留旧版并添加",
	DupReplace:            "替换旧版本",
	DupBackupReplace:      "备份并替换",

	DeleteSoftwareFmt:   "删除 %s",
	DeleteUnmanagedFmt:  "您确定要删除这个%s吗？此操作不可恢复。",
	DeleteManagedHint:   "您确定要删除此软件吗？此操作将从托管列表中移除该软件。",
	DeleteInstallDir:    "同时删除安装目录",
	DeleteArchive:       "同时删除归档文件",
	DeleteBackupHintMsg: "确定要删除此备份吗？它可能已关联到已托管软件。",

	ScanArchiveTitle:       "扫描并归档",
	ScanArchiveHintFmt:     "扫描「%s」中的所有子目录并将其纳入托管？",
	ScanArchiveBackupOnFmt: "新备份将保存到：%s",
	ScanArchiveBackupOff:   "将不会创建备份 ZIP。",
	ScanArchiveToggle:      "为未匹配旧归档的软件创建备份",
	ScanArchiveToggleHint:  "如果启用，将为每个没有对应归档的子目录创建带时间戳的 ZIP 备份。",
	ScanArchiveStart:       "开始扫描",
	ScanScanning:           "正在扫描...",
	ScanProgressFmt:        "%d / %d（%d%%）",
	ScanCurrentFmt:         "当前：%s",
	ScanSummaryTitle:       "扫描完成",
	ScanSummaryInstallFmt:  "安装目录：%s",
	ScanSummaryArchiveFmt:  "备份目录：%s",
	ScanSummaryTotalFmt:    "共处理：%d",
	ScanSummaryArchivedFmt: "成功处理：%d",
	ScanSummarySkippedFmt:  "已跳过：%d",
	ScanSummaryFailedFmt:   "失败：%d",
	ScanSummaryMoreFmt:     "还有 %d 个失败项...",
	AssocFoundFmt:          "发现 %d 个可关联归档",
	AssocHint:              "以下安装目录在归档目录中存在疑似对应的压缩包，请选择是否关联：",
	AssocSkipAll:           "全部跳过",
	AssocApply:             "应用关联",
	AssocNoMatchKeepBackup: "（不关联）",

	ArchiveTitle:             "归档管理",
	BackupTitle:              "备份文件",
	ArchiveFiles:             "归档文件",
	ArchiveNoArchiveFiles:    "暂无归档文件",
	ArchiveNoArchiveHint:     "添加或归档软件后，归档文件将显示在这里。",
	ArchiveBackupFiles:       "备份文件",
	ArchiveNoBackupFiles:     "暂无备份文件",
	BackupNoBackupHint:       "点击右上角「创建备份」为已托管软件创建备份。",
	ArchiveOpenArchiveDir:    "打开归档目录",
	ArchiveOpenBackupDir:     "打开备份目录",
	ArchiveLinked:            "已关联",
	ArchiveUnlinked:          "未关联",
	ArchiveManualLink:        "手动关联",
	ArchiveRestore:           "还原",
	ArchiveCreateBackup:      "创建备份",
	ArchiveCreatingBackup:    "正在创建备份...",
	ArchiveBackupCreatedFmt:  "已创建备份：%s",
	ArchiveDeleteArchive:     "删除归档",
	ArchiveDeleteBackup:      "删除备份",
	ArchiveDeleteArchiveHint: "确定要删除此归档文件吗？",
	ArchiveDeleteBackupHint:  "确定要删除此备份文件吗？相关软件的备份关联也会同步清除。",
	ArchiveNoAvailableSw:     "无可用软件",
	ArchiveNoAvailableSwHint: "当前没有已托管的软件，请先添加软件。",
	ArchiveLinkArchive:       "关联归档到软件",
	ArchiveLinkBackup:        "关联备份到软件",
	ArchiveLinkAction:        "关联",
	ArchiveLinkSuccess:       "归档已成功关联到软件。",
	ArchiveUnlinkAction:      "取消关联",
	ArchiveUnlinkSuccess:     "已取消关联。",
	ArchiveDuplicateMsg:      "目标安装目录或归档已存在，是否先备份现有目录后再覆盖还原？",
	ArchiveBackupAndRestore:  "备份并还原",
	ArchiveRevealInExplorer:  "在资源管理器中显示",
	ArchiveSearchPlaceholder: "搜索归档文件名...",
	BackupSearchPlaceholder:  "搜索备份文件名...",

	DialogSelectExecutable: "选择主程序可执行文件",
	DialogNoExecutable:     "未找到可执行文件",

	SettingsTitle:                "设置",
	SettingsDevHint:              "提示",
	SettingsDevHintContent:       "目前为开发版本，如遇异常请反馈。",
	SettingsStorage:              "存储",
	SettingsInstallPath:          "绿色软件安装目录",
	SettingsInstallPathHolder:    "输入或选择绿色软件安装目录",
	SettingsSaveInstallPath:      "保存安装路径",
	SettingsInstallPathSaved:     "安装路径已保存。",
	SettingsArchivePath:          "软件归档目录（可选，默认使用安装目录下的 ~archives）",
	SettingsArchivePathHolder:    "留空则默认使用安装目录下的 ~archives",
	SettingsSaveArchivePath:      "保存归档路径",
	SettingsArchivePathSaved:     "归档路径已保存。",
	SettingsConfigFilePath:       "配置文件位置",
	SettingsOpenFileLocation:     "打开位置",
	SettingsArchiveHandling:      "归档处理",
	SettingsRemoveNested:         "解压后移除多余嵌套目录",
	SettingsRemoveNestedDesc:     "如果压缩包内仅有一个顶层文件夹，自动将其内容上移",
	SettingsSaveArchiveHandling:  "保存归档处理设置",
	SettingsArchiveHandlingSaved: "归档处理设置已保存。",
	SettingsTest7Zip:             "检测 7-Zip 路径",
	SettingsTest7ZipDesc:         "检查系统中是否已安装 7-Zip（用于解压 RAR 与 7Z 文件）",
	SettingsTest7ZipTitle:        "7-Zip 路径检测结果",
	SettingsExecRecognition:      "可执行文件识别",
	SettingsMaxSearchDepth:       "最大搜索深度",
	SettingsExecExtensions:       "可执行文件扩展名",
	SettingsNoExtensions:         "未选择扩展名",
	SettingsSelectExtensions:     "编辑扩展名",
	SettingsSaveExecSettings:     "保存识别设置",
	SettingsExecSettingsSaved:    "可执行文件识别设置已保存。",
	SettingsCommonExtensions:     "常用扩展名",
	SettingsCustomExtensions:     "自定义扩展名",
	SettingsExtPlaceholder:       "例如 exe 或 .cmd",
	SettingsExtInvalid:           "请输入有效的扩展名。",
	SettingsExtExists:            "该扩展名已存在。",
	SettingsExtSelectAtLeastOne:  "请至少保留一个扩展名。",
	SettingsAppearance:           "外观",
	SettingsAppFont:              "应用字体",
	SettingsSystemDefaultFont:    "自动默认",
	SettingsPresetFonts:          "预设字体",
	SettingsSystemFonts:          "系统已安装字体",
	SettingsFontUnavailable:      "未安装",
	SettingsFontDefaultDescFmt:   "优先顺序：PingFang → 鸿蒙 → 微软雅黑（当前：%s）",
	SettingsFontDefaultDescPlain: "优先顺序：PingFang → 鸿蒙 → 微软雅黑",
	SettingsFontMissingWarnFmt:   "该字体在当前系统未安装，已自动回退至 %s",
	SettingsFontLoading:          "正在扫描系统字体...",
	SettingsThemeMode:            "主题模式",
	SettingsThemeModeDesc:        "切换浅色或深色外观，或跟随系统设置",
	SettingsThemeModeSystem:      "跟随系统",
	SettingsThemeModeLight:       "浅色",
	SettingsThemeModeDark:        "深色",
	SettingsThemeColor:           "主题颜色",
	SettingsThemeColorDesc:       "自定义界面强调色、按钮与高亮配色",
	SettingsThemeColorSystem:     "跟随系统",
	SettingsThemeColorCustom:     "自定义",
	SettingsColorBlue:            "经典蓝",
	SettingsColorSky:             "天空蓝",
	SettingsColorTeal:            "青碧色",
	SettingsColorGreen:           "翡翠绿",
	SettingsColorPurple:          "紫罗兰",
	SettingsColorPink:            "玫红色",
	SettingsColorOrange:          "琥珀橙",
	SettingsColorRed:             "绯红色",
	SettingsLanguage:             "界面语言",
	SettingsSaveLanguage:         "保存语言设置",
	SettingsLanguageSaved:        "语言设置已保存。",
	SettingsLogSettings:          "日志设置",
	SettingsEnableFileLogging:    "启用文件日志",
	SettingsEnableFileLoggingMsg: "将运行日志保存到文件以便排查问题",
	SettingsLogLevel:             "日志输出级别",
	SettingsLogLevelDesc:         "控制控制台与日志文件的最低输出策略",
	SettingsLogLevelDebug:        "调试 (DEBUG)",
	SettingsLogLevelInfo:         "常规 (INFO)",
	SettingsLogLevelWarn:         "警告 (WARN)",
	SettingsLogLevelError:        "仅错误 (ERROR)",
	SettingsLogLevelDebugHint:    "当前策略：输出全部日志（DEBUG / INFO / WARN / ERROR），包含列表刷新、元数据提取与残留备份匹配细节",
	SettingsLogLevelInfoHint:     "当前策略：输出常规业务及异常日志（INFO / WARN / ERROR），记录软件托管、解压、备份、关联与更新操作",
	SettingsLogLevelWarnHint:     "当前策略：仅输出警告与错误日志（WARN / ERROR），忽略常规操作记录",
	SettingsLogLevelErrorHint:    "当前策略：仅输出错误日志（ERROR），仅在操作失败或发生异常时记录",
	SettingsLogFilePath:          "日志文件位置",
	SettingsLogSettingsSaved:     "日志设置已保存。",
	SettingsAbout:                "关于",
	SettingsVersion:              "版本",
	SettingsBuildTime:            "构建时间",
	SettingsWebsite:              "官方网站",
	SettingsGitHub:               "GitHub 仓库",
	SettingsCheckUpdate:          "检查更新",
	SettingsReinstallLatest:      "重新安装最新版本",
	SettingsReinstallConfirm:     "重新安装应用",
	SettingsReinstallMessage:     "这将从 GitHub Releases 下载并重新安装最新版本（保留配置文件）。确定继续吗？",
	SettingsReinstallButton:      "重新安装",
	SettingsUnsavedTitle:         "存在未保存的更改",
	SettingsUnsavedMessage:       "设置页有尚未保存的更改，切换页面前要保存吗？",
	SettingsDiscardChanges:       "放弃更改",
	SettingsSaveAndContinue:      "保存并继续",
	SettingsSaveAll:              "全部保存",
	SettingsDevUnlocked:          "已开启开发者选项",
	SettingsUnsavedBadge:         "未保存",

	UpdateChecking:      "正在检查更新...",
	UpdateUpToDate:      "当前已是最新版本。",
	UpdateNewVersionFmt: "发现新版本：%s",
	UpdateContent:       "更新内容",
	UpdateDownloadBtn:   "下载并更新",
	UpdateDownloading:   "正在下载更新包...",
	UpdateInstalling:    "正在准备安装更新...",
	UpdateCancelled:     "下载已取消。",
	UpdateFailedFmt:     "更新失败：%s",

	DevTitle:            "开发者选项",
	DevHint:             "隐藏调试页面，用于测试可执行文件图标和元信息提取。",
	DevIconTest:         "图标提取测试",
	DevIconTestHint:     "选择一个 .exe / .lnk / .ico 文件，验证图标、文件说明和原生版本号提取效果。",
	DevExtractFailed:    "提取失败",
	DevExtractFailedMsg: "无法从所选文件中提取图标。",
	DevFileName:         "文件：",
	DevFileDesc:         "文件说明：",
	DevFileVersion:      "文件版本：",
	DevMethod1:          "32 × 32 图标预览",
	DevMethod1Desc:      "列表视图标准图标尺寸",
	DevMethod2:          "64 × 64 图标预览",
	DevMethod2Desc:      "网格视图大图标尺寸",
	DevMethod3:          "按钮组合预览",
	DevMethod3Desc:      "图标与原生按钮组件结合渲染效果",
	DevButtonWithIcon:   "带图标按钮",
	DevIconHint:         "以上图标由 Windows 原生 API / Shell 提取后经 MyGo GPU 渲染管线绘制。",
	DevHide:             "隐藏开发者选项",
	DevHiddenFeature:    "隐藏功能",
	DevHiddenFeatureMsg: "这是一个保留的调试彩蛋入口。",
	DevHiddenButton:     "彩蛋",
}

var enStrings = Strings{
	AppName:      "CarryDock",
	AppNameShort: "CarryDock",
	Loading:      "Loading...",
	Unknown:      "Unknown",
	Success:      "Success",
	Error:        "Error",
	Warning:      "Warning",
	Cancel:       "Cancel",
	Save:         "Save",
	Delete:       "Delete",
	Edit:         "Edit",
	Confirm:      "Confirm",
	Close:        "Close",
	OK:           "OK",
	Yes:          "Yes",
	No:           "No",
	Retry:        "Retry",
	Skip:         "Skip",
	Finish:       "Finish",
	Apply:        "Apply",
	Search:       "Search",
	Refresh:      "Refresh",
	Open:         "Open",
	Browse:       "Browse",
	Select:       "Select",
	Add:          "Add",
	Remove:       "Remove",

	NavHome:             "Home",
	NavArchive:          "Archive Manager",
	NavBackup:           "Backups",
	NavSettings:         "Settings",
	NavDeveloperOptions: "Developer Options",
	NavToggleTooltip:    "Collapse sidebar",
	NavExpandTooltip:    "Expand sidebar",

	LanguageChinese: "简体中文",
	LanguageEnglish: "English",
	TranslationByAI: "",

	HomeManagedSoftware:        "Managed Software",
	HomeUnmanagedFolders:       "Unmanaged Folders",
	HomeSearchPlaceholder:      "Search software name or path...",
	HomeListView:               "List View",
	HomeGridView:               "Grid View",
	HomeReorder:                "Reorder",
	HomeOpenInstallDir:         "Open Install Dir",
	HomeScanCurrentDir:         "Scan Directory for Software",
	HomeNoSoftware:             "No software added yet",
	HomeNoSoftwareHint:         "Click Add above, or drag & drop an archive or executable into this window.",
	HomeConfigureInstallFirst:  "Installation directory is not configured",
	HomeGoToSettings:           "Open Settings",
	HomeDropToAdd:              "Drop to add software",
	HomeDropHint:               "Supports ZIP / TAR / GZ / BZ2 / XZ / RAR / 7Z archives or configured executables",
	HomeUnknownFolder:          "Unknown Folder",
	HomeUnknownArchiveFile:     "Unknown Archive",
	HomeSoftwareDirDeleted:     "Install directory missing",
	HomeRehost:                 "Rehost",
	HomeBackupArchive:          "Backup Archive",
	HomeArchiveFile:            "Archive File",
	HomeChangeExecutable:       "Change Main Program",
	HomeScanning:               "Scanning executables...",
	HomeNoExecutableFound:      "No executable found in installation directory",
	HomeOnlyOneExecutableFound: "Only one executable found",
	HomeCopySuffix:             " - Copy",
	HomeNewSoftware:            "New Software ",
	HomeProcessingFiles:        "Processing files, please wait...",
	HomeInstallingSoftware:     "Installing software...",
	HomeCancellingOperation:    "Cancelling operation...",
	HomeDeleting:               "Deleting, please wait...",
	HomeRestoring:              "Restoring from archive/backup...",
	HomeMoveUp:                 "Move Up",
	HomeMoveDown:               "Move Down",
	HomeReorderHint:            "Drag rows or use the up/down arrows to reorder, then click Finish to save.",
	HomeReorderSave:            "Save Order",
	HomeMoreActions:            "More Actions",
	HomeLaunchFailedTitle:      "Launch Failed",
	HomeCannotLaunchTitle:      "Cannot Launch Software",
	HomeCannotLaunchMissingDir: "Installation directory does not exist. Try Rehost to restore from archive.",
	HomeCannotLaunchNoArchive:  "Installation directory does not exist and no archive was found.",
	HomeCannotLaunchNoExe:      "Please ensure the software is managed and has a main executable configured.",
	HomeUnrecognizedDropTitle:  "Unrecognized File",
	HomeUnrecognizedDropMsg:    "Please drop a supported archive (ZIP/TAR/GZ/BZ2/XZ/RAR/7Z) or a configured executable file.",
	HomeInstallDirNotConfig:    "Install Directory Not Configured",
	HomeInstallDirNotConfigMsg: "Please configure the portable software installation directory in Settings first.",
	HomeDirNotExistTitle:       "Directory Does Not Exist",
	HomeCountUnmanagedFmt:      "%d Unmanaged",
	FilterAll:                  "All",
	FilterManaged:              "Managed",
	FilterUnmanaged:            "Unmanaged",
	FilterLinked:               "Linked",
	FilterUnlinked:             "Unlinked",
	ClearFilter:                "Clear Filter",
	NoMatchingResults:          "No Matching Items",
	NoMatchingResultsHint:      "Try adjusting your search query or clearing active filters.",
	HomeVersionTooltipFmt:      "Native version: %s",

	TileOpenFolder:           "Open Software Folder",
	TileChangeFolder:         "Rename Software Folder",
	TileCreateBackup:         "Create Backup",
	TileAddToStartMenu:       "Add to Start Menu",
	TileOpenArchiveFolder:    "Open Archive Folder",
	TileLaunchOther:          "Launch Other Program",
	TileArchiveExists:        "Archive exists",
	TileArchiveNotExists:     "Archive missing",
	TileBackupDetected:       "Backup detected",
	TileBackupNotDetected:    "No backup detected",
	TileNoInstallDir:         "This software has no installation directory configured.",
	TileNoArchive:            "This software has no archive file configured.",
	TileArchiveNotFound:      "Cannot find archive file location.",
	TileParentOpened:         "Archive file does not exist; opened its parent folder.",
	TileNoExecutable:         "This software has no executable file configured.",
	TileInstallDirNotExists:  "Installation directory does not exist; cannot create backup.",
	TileCreatingBackup:       "Creating backup...",
	TileBackupCreatedFmt:     "Backup created: %s",
	TileCreateBackupFailed:   "Failed to create backup.",
	TileAddedToStartMenuFmt:  "Added \"%s\" to Start Menu.",
	TileAddToStartMenuErrFmt: "Failed to add to Start Menu: %s",
	TileChangeFolderTitle:    "Rename Software Folder",
	TileCurrentPath:          "Current path:",
	TileNewFolderName:        "New folder name:",
	TileNewFolderPlaceholder: "Enter new folder name",
	TileSamePath:             "New path is the same as the current path.",
	TileFolderNotExist:       "Current software folder does not exist.",
	TileFolderExists:         "Target folder already exists; choose another name.",
	TileMigrating:            "Migrating folder...",
	TileMigrateSuccess:       "Software folder renamed successfully.",
	TileMigrateFailedFmt:     "Failed to rename software folder: %s",

	DupDetected:           "Duplicate Detected",
	DupHelpFmt:            "Software name '%s' conflicts with existing items. Choose how to proceed:",
	DupExistingFmt:        "Managed software: %s (%s)",
	DupInstallDirFmt:      "Installation directory exists: %s",
	DupArchiveFmt:         "Archive file exists: %s",
	DupRenameHint:         "Enter a new name to rename and add:",
	DupRenameAdd:          "Rename & Add",
	DupOverwrite:          "Overwrite",
	DupBackupOverwrite:    "Backup & Overwrite",
	DupCurrentVersionFmt:  "Current version: %s",
	DupIncomingVersionFmt: "Incoming version: %s",
	DupVersionCompareFmt:  "Version change: %s → %s",
	DupVersionSameSuffix:  " (same version)",
	DupUpdateTitle:        "New Version Detected",
	DupUpdateHelpFmt:      "A newer version of '%s' was detected. Replace the currently installed version?",
	DupReplaceTitle:       "Existing Software Detected",
	DupReplaceHelpFmt:     "The incoming file matches managed software '%s'. Replace the existing version?",
	DupIncomingFileFmt:    "Incoming file: %s",
	DupOldInstallDirFmt:   "Current install directory: %s",
	DupVersionOlderSuffix: " (older than current)",
	DupKeepBothHint:       "To keep the existing version and add separately, edit the name below:",
	DupKeepBothAdd:        "Keep Both & Add",
	DupReplace:            "Replace",
	DupBackupReplace:      "Backup & Replace",

	DeleteSoftwareFmt:   "Delete %s",
	DeleteUnmanagedFmt:  "Are you sure you want to delete this %s? This cannot be undone.",
	DeleteManagedHint:   "Are you sure you want to remove this software from the managed list?",
	DeleteInstallDir:    "Also delete installation directory",
	DeleteArchive:       "Also delete archive file",
	DeleteBackupHintMsg: "Are you sure you want to delete this backup? Its associations will be cleared.",

	ScanArchiveTitle:       "Scan & Archive",
	ScanArchiveHintFmt:     "Scan subdirectories in '%s' and register them as managed software?",
	ScanArchiveBackupOnFmt: "New backups will be saved to: %s",
	ScanArchiveBackupOff:   "No backup ZIP will be created.",
	ScanArchiveToggle:      "Create backups for unmatched software",
	ScanArchiveToggleHint:  "When enabled, creates a timestamped ZIP backup for directories with no matching archive.",
	ScanArchiveStart:       "Start Scan",
	ScanScanning:           "Scanning...",
	ScanProgressFmt:        "%d / %d (%d%%)",
	ScanCurrentFmt:         "Current: %s",
	ScanSummaryTitle:       "Scan Complete",
	ScanSummaryInstallFmt:  "Install directory: %s",
	ScanSummaryArchiveFmt:  "Backup directory: %s",
	ScanSummaryTotalFmt:    "Total processed: %d",
	ScanSummaryArchivedFmt: "Succeeded: %d",
	ScanSummarySkippedFmt:  "Skipped: %d",
	ScanSummaryFailedFmt:   "Failed: %d",
	ScanSummaryMoreFmt:     "And %d more failures...",
	AssocFoundFmt:          "Found %d Archive Associations",
	AssocHint:              "The following folders have matching archive candidates. Choose which ones to associate:",
	AssocSkipAll:           "Skip All",
	AssocApply:             "Apply Associations",
	AssocNoMatchKeepBackup: "(Do not associate)",

	ArchiveTitle:             "Archive Manager",
	BackupTitle:              "Backups",
	ArchiveFiles:             "Archives",
	ArchiveNoArchiveFiles:    "No archive files",
	ArchiveNoArchiveHint:     "Archived software packages will appear here.",
	ArchiveBackupFiles:       "Backups",
	ArchiveNoBackupFiles:     "No backup files",
	BackupNoBackupHint:       "Click Create Backup above to back up managed software.",
	ArchiveOpenArchiveDir:    "Open Archive Dir",
	ArchiveOpenBackupDir:     "Open Backup Dir",
	ArchiveLinked:            "Linked",
	ArchiveUnlinked:          "Unlinked",
	ArchiveManualLink:        "Manual Link",
	ArchiveRestore:           "Restore",
	ArchiveCreateBackup:      "Create Backup",
	ArchiveCreatingBackup:    "Creating backup...",
	ArchiveBackupCreatedFmt:  "Backup created: %s",
	ArchiveDeleteArchive:     "Delete Archive",
	ArchiveDeleteBackup:      "Delete Backup",
	ArchiveDeleteArchiveHint: "Are you sure you want to delete this archive file?",
	ArchiveDeleteBackupHint:  "Are you sure you want to delete this backup file? Linked references will be cleared.",
	ArchiveNoAvailableSw:     "No Available Software",
	ArchiveNoAvailableSwHint: "No managed software available. Please add software first.",
	ArchiveLinkArchive:       "Link Archive to Software",
	ArchiveLinkBackup:        "Link Backup to Software",
	ArchiveLinkAction:        "Link",
	ArchiveLinkSuccess:       "Archive has been linked to software.",
	ArchiveUnlinkAction:      "Unlink",
	ArchiveUnlinkSuccess:     "Unlinked successfully.",
	ArchiveDuplicateMsg:      "Target install folder or archive already exists. Back up existing folder and restore?",
	ArchiveBackupAndRestore:  "Backup & Restore",
	ArchiveRevealInExplorer:  "Show in Explorer",
	ArchiveSearchPlaceholder: "Search archive files...",
	BackupSearchPlaceholder:  "Search backup files...",

	DialogSelectExecutable: "Select Main Executable",
	DialogNoExecutable:     "No executables found",

	SettingsTitle:                "Settings",
	SettingsDevHint:              "Notice",
	SettingsDevHintContent:       "This is currently a development version. Please report any issues you encounter.",
	SettingsStorage:              "Storage",
	SettingsInstallPath:          "Installation Directory",
	SettingsInstallPathHolder:    "Enter or browse portable software root directory",
	SettingsSaveInstallPath:      "Save Install Path",
	SettingsInstallPathSaved:     "Installation path saved.",
	SettingsArchivePath:          "Archive Directory (optional; defaults to ~archives inside install dir)",
	SettingsArchivePathHolder:    "Leave blank to use ~archives inside install directory",
	SettingsSaveArchivePath:      "Save Archive Path",
	SettingsArchivePathSaved:     "Archive path saved.",
	SettingsConfigFilePath:       "Configuration File",
	SettingsOpenFileLocation:     "Open Location",
	SettingsArchiveHandling:      "Archive Handling",
	SettingsRemoveNested:         "Remove redundant nested directories",
	SettingsRemoveNestedDesc:     "Flatten single top-level folder automatically after extraction",
	SettingsSaveArchiveHandling:  "Save Archive Settings",
	SettingsArchiveHandlingSaved: "Archive handling settings saved.",
	SettingsTest7Zip:             "Test 7-Zip Detection",
	SettingsTest7ZipDesc:         "Check if 7-Zip is installed for RAR and 7Z extraction",
	SettingsTest7ZipTitle:        "7-Zip Path Detection",
	SettingsExecRecognition:      "Executable Recognition",
	SettingsMaxSearchDepth:       "Max Search Depth",
	SettingsExecExtensions:       "Executable Extensions",
	SettingsNoExtensions:         "No extensions selected",
	SettingsSelectExtensions:     "Edit Extensions",
	SettingsSaveExecSettings:     "Save Recognition Settings",
	SettingsExecSettingsSaved:    "Executable recognition settings saved.",
	SettingsCommonExtensions:     "Common Extensions",
	SettingsCustomExtensions:     "Custom Extension",
	SettingsExtPlaceholder:       "e.g. exe or .cmd",
	SettingsExtInvalid:           "Please enter a valid extension.",
	SettingsExtExists:            "This extension already exists.",
	SettingsExtSelectAtLeastOne:  "Please keep at least one extension.",
	SettingsAppearance:           "Appearance",
	SettingsAppFont:              "Application Font",
	SettingsSystemDefaultFont:    "Auto Default",
	SettingsPresetFonts:          "Preset Fonts",
	SettingsSystemFonts:          "System Fonts",
	SettingsFontUnavailable:      "Unavailable",
	SettingsFontDefaultDescFmt:   "Priority: PingFang → HarmonyOS → YaHei (Active: %s)",
	SettingsFontDefaultDescPlain: "Priority: PingFang → HarmonyOS → YaHei",
	SettingsFontMissingWarnFmt:   "Font not installed on this system; falling back to %s",
	SettingsFontLoading:          "Loading system fonts...",
	SettingsThemeMode:            "Theme Mode",
	SettingsThemeModeDesc:        "Switch between light and dark appearance, or follow system",
	SettingsThemeModeSystem:      "System",
	SettingsThemeModeLight:       "Light",
	SettingsThemeModeDark:        "Dark",
	SettingsThemeColor:           "Theme Color",
	SettingsThemeColorDesc:       "Customize accent color for buttons and highlights",
	SettingsThemeColorSystem:     "System",
	SettingsThemeColorCustom:     "Custom",
	SettingsColorBlue:            "Blue",
	SettingsColorSky:             "Sky",
	SettingsColorTeal:            "Teal",
	SettingsColorGreen:           "Green",
	SettingsColorPurple:          "Purple",
	SettingsColorPink:            "Pink",
	SettingsColorOrange:          "Orange",
	SettingsColorRed:             "Red",
	SettingsLanguage:             "Language",
	SettingsSaveLanguage:         "Save Language",
	SettingsLanguageSaved:        "Language saved.",
	SettingsLogSettings:          "Logging",
	SettingsEnableFileLogging:    "Enable File Logging",
	SettingsEnableFileLoggingMsg: "Write logs to disk for troubleshooting",
	SettingsLogLevel:             "Log Output Level",
	SettingsLogLevelDesc:         "Control the minimum severity strategy for console and file logs",
	SettingsLogLevelDebug:        "Debug (DEBUG)",
	SettingsLogLevelInfo:         "Standard (INFO)",
	SettingsLogLevelWarn:         "Warning (WARN)",
	SettingsLogLevelError:        "Error Only (ERROR)",
	SettingsLogLevelDebugHint:    "Current strategy: Output all logs (DEBUG / INFO / WARN / ERROR), including list refresh, metadata extraction, and backup matching details",
	SettingsLogLevelInfoHint:     "Current strategy: Output standard and error logs (INFO / WARN / ERROR) for software hosting, extraction, backup, linking, and updates",
	SettingsLogLevelWarnHint:     "Current strategy: Output only warnings and errors (WARN / ERROR), omitting routine operations",
	SettingsLogLevelErrorHint:    "Current strategy: Output only errors (ERROR) when operations fail",
	SettingsLogFilePath:          "Log File",
	SettingsLogSettingsSaved:     "Log settings saved.",
	SettingsAbout:                "About",
	SettingsVersion:              "Version",
	SettingsBuildTime:            "Build Time",
	SettingsWebsite:              "Official Website",
	SettingsGitHub:               "GitHub Repository",
	SettingsCheckUpdate:          "Check for Updates",
	SettingsReinstallLatest:      "Reinstall Latest Version",
	SettingsReinstallConfirm:     "Reinstall Application",
	SettingsReinstallMessage:     "This will download and reinstall the latest GitHub release while preserving your configuration. Continue?",
	SettingsReinstallButton:      "Reinstall",
	SettingsUnsavedTitle:         "Unsaved Changes",
	SettingsUnsavedMessage:       "You have unsaved changes in Settings. Save before leaving?",
	SettingsDiscardChanges:       "Discard",
	SettingsSaveAndContinue:      "Save & Continue",
	SettingsSaveAll:              "Save All",
	SettingsDevUnlocked:          "Developer Options enabled",
	SettingsUnsavedBadge:         "Unsaved",

	UpdateChecking:      "Checking for updates...",
	UpdateUpToDate:      "You are already on the latest version.",
	UpdateNewVersionFmt: "New Version Available: %s",
	UpdateContent:       "Release Notes",
	UpdateDownloadBtn:   "Download & Update",
	UpdateDownloading:   "Downloading update...",
	UpdateInstalling:    "Preparing to install update...",
	UpdateCancelled:     "Download cancelled.",
	UpdateFailedFmt:     "Update failed: %s",

	DevTitle:            "Developer Options",
	DevHint:             "Hidden debug screen for testing executable icon and metadata extraction.",
	DevIconTest:         "Icon Extraction Test",
	DevIconTestHint:     "Pick an .exe, .lnk, or .ico file to inspect icon, file description, and native version extraction.",
	DevExtractFailed:    "Extraction Failed",
	DevExtractFailedMsg: "Could not extract an icon from the selected file.",
	DevFileName:         "File: ",
	DevFileDesc:         "Description: ",
	DevFileVersion:      "Version: ",
	DevMethod1:          "32 × 32 Preview",
	DevMethod1Desc:      "Standard list view icon size",
	DevMethod2:          "64 × 64 Preview",
	DevMethod2Desc:      "Grid view large icon size",
	DevMethod3:          "Button Preview",
	DevMethod3Desc:      "Icon rendered inside native buttons",
	DevButtonWithIcon:   "Button",
	DevIconHint:         "Icons above are extracted via Windows native API / Shell and drawn on GPU.",
	DevHide:             "Hide Developer Options",
	DevHiddenFeature:    "Hidden Feature",
	DevHiddenFeatureMsg: "A placeholder easter egg entry.",
	DevHiddenButton:     "Hidden",
}

// For 返回指定语言对应的文案集合。
func For(lang Lang) Strings {
	if lang == LangEnUS {
		return enStrings
	}
	return zhStrings
}

// Fmt 是一个便捷格式化函数。
func Fmt(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
