package software

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"carrydock/internal/archive"
	"carrydock/internal/logger"
	"carrydock/internal/model"
	"carrydock/internal/settings"
	"carrydock/internal/winutil"
)

const (
	defaultArchiveFolderName = "~archives"
	softwareListFileName     = "software_list.json"
	softwareListLockFileName = "software_list.lock"
	backupSubfolderName      = "backup"
)

// ErrInstallPathNotConfigured 表示尚未在设置中配置绿色软件安装目录。
var ErrInstallPathNotConfigured = errors.New("尚未设置安装路径，请先前往设置界面进行配置")

// SourceType 表示添加软件时的源文件类型。
type SourceType int

const (
	SourceArchive SourceType = iota
	SourceExecutable
)

// AddResultType 表示添加或重新托管操作的结果类型。
type AddResultType int

const (
	AddSuccess AddResultType = iota
	AddCancelled
	AddNeedsSelection
	AddDuplicate
	AddError
)

// PendingAddition 在解压后发现多个可执行文件时携带待用户选择的上下文。
type PendingAddition struct {
	SoftwareName       string
	InstallPath        string
	ArchivePath        string
	BackupPath         string
	ExecutablePaths    []string
	PreferredSortOrder *int
	ExistingSoftwareID string
	// CreatedArchive 标记本次操作是否新复制生成了 ArchivePath，
	// 取消选择主程序时仅在为 true 时清理，避免误删归档目录中原本已有的归档文件。
	CreatedArchive bool
}

// DuplicateInfo 描述添加或还原时检测到的重复冲突信息。
type DuplicateInfo struct {
	SourceType              SourceType
	SourcePath              string
	InstallRootPath         string
	IntendedName            string
	TargetInstallPath       string
	TargetArchivePath       string
	InstallDirExists        bool
	ArchiveExists           bool
	ExistingManagedSoftware *model.Software
}

// AddResult 为添加/重新托管操作的统一返回结构。
type AddResult struct {
	Type            AddResultType
	PendingAddition *PendingAddition
	DuplicateInfo   *DuplicateInfo
	Err             error
}

// BatchArchiveFailure 记录批量扫描归档中的单个失败项。
type BatchArchiveFailure struct {
	Name  string
	Error string
}

// ArchiveAssociationSuggestion 表示扫描阶段发现的可关联旧归档候选项。
type ArchiveAssociationSuggestion struct {
	DisplayName string
	InstallPath string
	Candidates  []string
}

// ArchiveAssociationResolution 表示用户在统一关联弹窗中做出的决策。
type ArchiveAssociationResolution struct {
	InstallPath         string
	SelectedArchivePath string
}

// BatchArchiveSummary 为批量扫描与归档操作的结果摘要。
type BatchArchiveSummary struct {
	Total           int
	Archived        []string
	SkippedExisting []string
	Failures        []BatchArchiveFailure
	ArchiveDirPath  string
	InstallDirPath  string
	Suggestions     []ArchiveAssociationSuggestion
}

// Service 封装绿驿管家的核心软件托管、解压、备份、还原、排序与归档管理逻辑。
type Service struct {
	settings *settings.Service
}

// NewService 创建一个新的 Software Service。
func NewService(settingsSvc *settings.Service) *Service {
	return &Service{settings: settingsSvc}
}

// ResolveArchiveDirectoryPath 返回实际生效的归档根目录（若不存在会自动创建）。
func (s *Service) ResolveArchiveDirectoryPath() (string, error) {
	archivePath := s.settings.GetArchivePath()
	if archivePath == "" {
		installPath := s.settings.GetInstallPath()
		if installPath == "" {
			return "", ErrInstallPathNotConfigured
		}
		archivePath = filepath.Join(installPath, defaultArchiveFolderName)
	}
	if err := os.MkdirAll(archivePath, 0o755); err != nil {
		return "", err
	}
	return archivePath, nil
}

// ResolveBackupDirectoryPath 返回归档目录下的 backup 子目录（若不存在会自动创建）。
func (s *Service) ResolveBackupDirectoryPath() (string, error) {
	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return "", err
	}
	backupDir := filepath.Join(archDir, backupSubfolderName)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	return backupDir, nil
}

func (s *Service) softwareListPaths() (listPath, lockPath string, err error) {
	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(archDir, softwareListFileName),
		filepath.Join(archDir, softwareListLockFileName),
		nil
}

// LoadManagedSoftware 读取并按持久化 sortOrder 排序的已托管软件列表。
// 修复了原版忽略 sortOrder 直接按数组索引覆盖的问题。
func (s *Service) LoadManagedSoftware() ([]model.Software, error) {
	listPath, lockPath, err := s.softwareListPaths()
	if err != nil {
		return nil, err
	}
	var list []model.Software
	err = winutil.WithFileLock(lockPath, func() error {
		raw, readErr := os.ReadFile(listPath)
		if errors.Is(readErr, os.ErrNotExist) || len(strings.TrimSpace(string(raw))) == 0 {
			list = []model.Software{}
			return nil
		}
		if readErr != nil {
			return readErr
		}
		if unmarshalErr := json.Unmarshal(raw, &list); unmarshalErr != nil {
			return fmt.Errorf("解析 software_list.json 失败: %w", unmarshalErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].SortOrder != list[j].SortOrder {
			return list[i].SortOrder < list[j].SortOrder
		}
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	for i := range list {
		list[i].SortOrder = i
		list[i].Status = model.StatusManaged
	}
	return list, nil
}

func (s *Service) saveManagedSoftware(list []model.Software) error {
	managed := make([]model.Software, 0, len(list))
	for _, item := range list {
		if item.Status == model.StatusManaged {
			managed = append(managed, item)
		}
	}
	sort.SliceStable(managed, func(i, j int) bool {
		return managed[i].SortOrder < managed[j].SortOrder
	})
	for i := range managed {
		managed[i].SortOrder = i
	}

	listPath, lockPath, err := s.softwareListPaths()
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(managed, "", "  ")
	if err != nil {
		return err
	}

	return winutil.WithFileLock(lockPath, func() error {
		tmpPath := fmt.Sprintf("%s.tmp.%d", listPath, os.Getpid())
		if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, listPath); err != nil {
			_ = os.Remove(tmpPath)
			return os.WriteFile(listPath, encoded, 0o644)
		}
		return nil
	})
}

// GetSoftwareList 加载全部已托管软件、未托管安装目录以及未关联归档，并同步运行时状态。
func (s *Service) GetSoftwareList() ([]model.Software, error) {
	managed, err := s.LoadManagedSoftware()
	if err != nil {
		return nil, err
	}

	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return nil, err
	}
	backupDir, _ := s.ResolveBackupDirectoryPath()

	backupFiles := collectZipNames(backupDir)

	for i := range managed {
		sw := &managed[i]
		sw.ArchiveExists = sw.ArchivePath != "" && fileExists(sw.ArchivePath)
		sw.InstallExists = sw.InstallPath != "" && dirExists(sw.InstallPath)

		hasBackup := false
		if sw.BackupPath != "" && fileExists(sw.BackupPath) {
			hasBackup = true
		} else if sw.ArchivePath != "" && s.isBackupFilePath(sw.ArchivePath) && sw.ArchiveExists {
			hasBackup = true
		} else if len(backupFiles) > 0 {
			prefix := strings.ToLower(SanitizeSoftwareName(sw.Name) + "-")
			for _, bf := range backupFiles {
				if strings.HasPrefix(strings.ToLower(bf), prefix) {
					hasBackup = true
					break
				}
			}
		}
		sw.IsBackupArchive = hasBackup
	}

	// 扫描安装目录下未托管的一级子目录（使用完整路径比对归档目录，避免误跳过同名软件文件夹）。
	installPath := s.settings.GetInstallPath()
	var unknownInstalls []model.Software
	if installPath != "" && dirExists(installPath) {
		entries, readErr := os.ReadDir(installPath)
		if readErr == nil {
			for _, ent := range entries {
				if !ent.IsDir() {
					continue
				}
				name := ent.Name()
				fullPath := filepath.Join(installPath, name)
				if strings.EqualFold(name, defaultArchiveFolderName) ||
					strings.HasPrefix(name, ".carrydock_staging_") ||
					winutil.SamePath(fullPath, archDir) ||
					(backupDir != "" && winutil.SamePath(fullPath, backupDir)) {
					continue
				}
				isManaged := false
				for _, m := range managed {
					if winutil.SamePath(m.InstallPath, fullPath) {
						isManaged = true
						break
					}
				}
				if !isManaged {
					unknownInstalls = append(unknownInstalls, model.Software{
						ID:            fullPath,
						Name:          name,
						InstallPath:   fullPath,
						InstallExists: true,
						Status:        model.StatusUnknownInstall,
					})
				}
			}
		}
	}

	// 扫描归档根目录下的未关联归档文件。
	var unknownArchives []model.Software
	if dirExists(archDir) {
		entries, readErr := os.ReadDir(archDir)
		if readErr == nil {
			for _, ent := range entries {
				if ent.IsDir() {
					continue
				}
				lower := strings.ToLower(ent.Name())
				if lower == softwareListFileName || lower == softwareListLockFileName {
					continue
				}
				fullPath := filepath.Join(archDir, ent.Name())
				isManaged := false
				for _, m := range managed {
					if winutil.SamePath(m.ArchivePath, fullPath) {
						isManaged = true
						break
					}
				}
				if !isManaged {
					unknownArchives = append(unknownArchives, model.Software{
						ID:            fullPath,
						Name:          ent.Name(),
						ArchivePath:   fullPath,
						ArchiveExists: true,
						Status:        model.StatusUnknownArchive,
					})
				}
			}
		}
	}

	out := make([]model.Software, 0, len(managed)+len(unknownInstalls)+len(unknownArchives))
	out = append(out, managed...)
	out = append(out, unknownInstalls...)
	out = append(out, unknownArchives...)
	return out, nil
}

// ListArchivesForManager 列出归档根目录和 backup 子目录下的全部归档/备份文件。
func (s *Service) ListArchivesForManager(includeBackups bool) ([]model.Software, error) {
	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return nil, err
	}

	var result []model.Software
	if entries, err := os.ReadDir(archDir); err == nil {
		for _, ent := range entries {
			if ent.IsDir() {
				continue
			}
			lower := strings.ToLower(ent.Name())
			if lower == softwareListFileName || lower == softwareListLockFileName {
				continue
			}
			fullPath := filepath.Join(archDir, ent.Name())
			result = append(result, model.Software{
				ID:            fullPath,
				Name:          ent.Name(),
				ArchivePath:   fullPath,
				ArchiveExists: true,
				Status:        model.StatusUnknownArchive,
			})
		}
	}

	if includeBackups {
		if backupDir, err := s.ResolveBackupDirectoryPath(); err == nil {
			if entries, err := os.ReadDir(backupDir); err == nil {
				for _, ent := range entries {
					if ent.IsDir() {
						continue
					}
					fullPath := filepath.Join(backupDir, ent.Name())
					result = append(result, model.Software{
						ID:              fullPath,
						Name:            ent.Name(),
						ArchivePath:     fullPath,
						ArchiveExists:   true,
						Status:          model.StatusUnknownArchive,
						IsBackupArchive: true,
					})
				}
			}
		}
	}
	return result, nil
}

// LinkArchiveToSoftware 将指定归档或备份路径手动关联到指定已托管软件。
func (s *Service) LinkArchiveToSoftware(softwareID, archivePath string) error {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}
	idx := -1
	for i := range list {
		if list[i].ID == softwareID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("未找到指定软件 (ID: %s)", softwareID)
	}
	if s.isBackupFilePath(archivePath) {
		list[idx].BackupPath = archivePath
	} else {
		list[idx].ArchivePath = archivePath
	}
	return s.saveManagedSoftware(list)
}

// DeleteArchiveFile 删除归档或备份文件，并同步清除所有已托管软件中指向该文件的失效关联。
func (s *Service) DeleteArchiveFile(filePath string) error {
	if fileExists(filePath) {
		if err := os.Remove(filePath); err != nil {
			return err
		}
	}
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return nil
	}
	changed := false
	for i := range list {
		if list[i].BackupPath != "" && winutil.SamePath(list[i].BackupPath, filePath) {
			list[i].BackupPath = ""
			changed = true
		}
		if list[i].ArchivePath != "" && winutil.SamePath(list[i].ArchivePath, filePath) {
			list[i].ArchivePath = ""
			changed = true
		}
	}
	if changed {
		return s.saveManagedSoftware(list)
	}
	return nil
}

// CreateBackupForSoftware 为指定已托管软件创建带时间戳的 ZIP 备份，并更新其 BackupPath。
func (s *Service) CreateBackupForSoftware(sw model.Software) (string, error) {
	if sw.InstallPath == "" {
		return "", errors.New("该软件未配置安装目录，无法创建备份")
	}
	if !dirExists(sw.InstallPath) {
		return "", fmt.Errorf("安装目录不存在：%s", sw.InstallPath)
	}
	created, err := s.CreateBackupFromDirectory(sw.InstallPath, sw.Name)
	if err != nil {
		return "", err
	}
	// 同步记录最新备份路径
	if list, loadErr := s.LoadManagedSoftware(); loadErr == nil {
		for i := range list {
			if list[i].ID == sw.ID {
				list[i].BackupPath = created
				_ = s.saveManagedSoftware(list)
				break
			}
		}
	}
	return created, nil
}

// CreateBackupFromDirectory 将指定目录打包为带时间戳的 ZIP 文件并存入 backup 子目录。
func (s *Service) CreateBackupFromDirectory(sourceDir, displayName string) (string, error) {
	backupDir, err := s.ResolveBackupDirectoryPath()
	if err != nil {
		return "", err
	}
	if displayName == "" {
		displayName = filepath.Base(sourceDir)
	}
	sanitized := SanitizeSoftwareName(displayName)
	ts := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s-%s.zip", sanitized, ts)
	targetPath := filepath.Join(backupDir, filename)

	// 若同一秒内重复创建，追加序号避免覆盖。
	for n := 1; fileExists(targetPath); n++ {
		targetPath = filepath.Join(backupDir, fmt.Sprintf("%s-%s_%d.zip", sanitized, ts, n))
	}

	if err := archive.CreateZipFromDirectory(sourceDir, targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
}

// AddSoftwareFromFile 根据文件类型自动识别（归档或可执行文件）并执行软件添加流程。
func (s *Service) AddSoftwareFromFile(filePath string) AddResult {
	installPath := s.settings.GetInstallPath()
	if installPath == "" {
		return AddResult{Type: AddError, Err: ErrInstallPathNotConfigured}
	}
	if !fileExists(filePath) {
		return AddResult{Type: AddError, Err: fmt.Errorf("提供的文件不存在: %s", filePath)}
	}

	if format := archive.DetectFormat(filePath); format != archive.FormatUnknown {
		return s.addFromArchiveFile(filePath, installPath, format, false, "")
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), ".")
	allowed := s.settings.GetExecutableExtensions()
	if slices.Contains(allowed, ext) {
		return s.addFromExecutableFile(filePath, installPath, false, "")
	}
	return AddResult{Type: AddError, Err: fmt.Errorf("不支持的文件类型: %s", filepath.Ext(filePath))}
}

// ResolveDuplicateAddition 根据用户的冲突解决决定（覆盖或重命名）继续添加软件。
func (s *Service) ResolveDuplicateAddition(info DuplicateInfo, overwrite bool, renameTo string) AddResult {
	renameTo = strings.TrimSpace(renameTo)
	if !overwrite && renameTo == "" {
		return AddResult{Type: AddError, Err: errors.New("未提供有效的重复处理指令")}
	}
	switch info.SourceType {
	case SourceArchive:
		if !fileExists(info.SourcePath) {
			return AddResult{Type: AddError, Err: errors.New("原始压缩包已不存在，请重新选择文件")}
		}
		format := archive.DetectFormat(info.SourcePath)
		if format == archive.FormatUnknown {
			return AddResult{Type: AddError, Err: fmt.Errorf("不支持的压缩格式: %s", filepath.Ext(info.SourcePath))}
		}
		return s.addFromArchiveFile(info.SourcePath, info.InstallRootPath, format, overwrite, renameTo)
	case SourceExecutable:
		if !fileExists(info.SourcePath) {
			return AddResult{Type: AddError, Err: errors.New("原始可执行文件已不存在，请重新选择文件")}
		}
		return s.addFromExecutableFile(info.SourcePath, info.InstallRootPath, overwrite, renameTo)
	default:
		return AddResult{Type: AddError, Err: errors.New("未知来源类型")}
	}
}

func (s *Service) addFromArchiveFile(
	archivePath, installPath string,
	format archive.Format,
	allowOverride bool,
	customName string,
) AddResult {
	isFromBackup := s.isBackupFilePath(archivePath)
	fallbackName := archive.StripArchiveExtension(archivePath)
	if customName == "" && isFromBackup {
		fallbackName = stripBackupTimestamp(fallbackName)
	}
	nameInput := customName
	if nameInput == "" {
		nameInput = fallbackName
	}
	softwareName := SanitizeSoftwareName(nameInput)

	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return AddResult{Type: AddError, Err: err}
	}

	destDir := filepath.Join(installPath, softwareName)
	var targetArchivePath string
	var backupPath string
	var sameSourceAsDest bool

	if isFromBackup {
		// 从 backup 恢复时保留原备份文件位置，不额外复制一份损坏的副本到归档根目录。
		targetArchivePath = ""
		backupPath = archivePath
		sameSourceAsDest = true
	} else {
		archiveFileName := buildArchiveFileName(softwareName, filepath.Base(archivePath))
		targetArchivePath = filepath.Join(archDir, archiveFileName)
		sameSourceAsDest = winutil.SamePath(archivePath, targetArchivePath)
	}

	installExists := dirExists(destDir)
	archiveExists := !isFromBackup && !sameSourceAsDest && fileExists(targetArchivePath)
	existingManaged, _ := s.findManagedByPaths(destDir, targetArchivePath)

	if !allowOverride && (installExists || archiveExists || existingManaged != nil) {
		conflictArchive := targetArchivePath
		if conflictArchive == "" {
			conflictArchive = backupPath
		}
		return AddResult{
			Type: AddDuplicate,
			DuplicateInfo: &DuplicateInfo{
				SourceType:              SourceArchive,
				SourcePath:              archivePath,
				InstallRootPath:         installPath,
				IntendedName:            softwareName,
				TargetInstallPath:       destDir,
				TargetArchivePath:       conflictArchive,
				InstallDirExists:        installExists,
				ArchiveExists:           archiveExists,
				ExistingManagedSoftware: existingManaged,
			},
		}
	}

	// 先解压到临时 staging 目录，验证确实包含可执行文件后再替换目标目录，
	// 防止覆盖模式在解压失败时毁掉旧软件。
	stagingDir, err := os.MkdirTemp(installPath, ".carrydock_staging_")
	if err != nil {
		return AddResult{Type: AddError, Err: fmt.Errorf("创建临时解压目录失败: %w", err)}
	}
	cleanupStaging := true
	defer func() {
		if cleanupStaging {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	logger.Info("开始解压归档 (%s): %s -> %s", format.String(), archivePath, stagingDir)
	if err := archive.Extract(archivePath, stagingDir, format); err != nil {
		return AddResult{Type: AddError, Err: err}
	}

	if s.settings.GetRemoveNestedFoldersEnabled() {
		if flattened := FlattenRedundantTopDirectories(stagingDir); flattened {
			logger.Info("已自动展开嵌套目录: %s", stagingDir)
		}
	}

	executables := s.FindExecutablesInDirectory(stagingDir)
	if len(executables) == 0 {
		return AddResult{Type: AddError, Err: errors.New("在压缩包中未找到可执行文件")}
	}

	// 验证通过后执行覆盖准备与归档复制。
	var preferredOrder *int
	if allowOverride {
		overrideArchive := targetArchivePath
		if sameSourceAsDest {
			overrideArchive = ""
		}
		preferredOrder, err = s.prepareForOverride(destDir, overrideArchive)
		if err != nil {
			return AddResult{Type: AddError, Err: err}
		}
	}

	createdArchive := false
	if !isFromBackup && !sameSourceAsDest && targetArchivePath != "" {
		if err := copyFile(archivePath, targetArchivePath); err != nil {
			return AddResult{Type: AddError, Err: fmt.Errorf("归档源压缩包失败: %w", err)}
		}
		createdArchive = true
	}

	// 将 staging 目录原子移动到目标安装目录。
	if err := os.Rename(stagingDir, destDir); err != nil {
		if createdArchive {
			_ = os.Remove(targetArchivePath)
		}
		return AddResult{Type: AddError, Err: fmt.Errorf("移动解压目录失败: %w", err)}
	}
	cleanupStaging = false

	// 将找到的可执行路径重映射到最终 destDir。
	mappedExecs := make([]string, 0, len(executables))
	for _, ex := range executables {
		if rel, relErr := filepath.Rel(stagingDir, ex); relErr == nil {
			mappedExecs = append(mappedExecs, filepath.Join(destDir, rel))
		}
	}
	if len(mappedExecs) == 0 {
		mappedExecs = s.FindExecutablesInDirectory(destDir)
	}

	finalArchivePath := targetArchivePath
	if isFromBackup && existingManaged != nil && existingManaged.ArchivePath != "" && fileExists(existingManaged.ArchivePath) {
		finalArchivePath = existingManaged.ArchivePath
	}

	if len(mappedExecs) == 1 {
		if err := s.CompleteSoftwareAddition(PendingAddition{
			SoftwareName:       softwareName,
			InstallPath:        destDir,
			ArchivePath:        finalArchivePath,
			BackupPath:         backupPath,
			PreferredSortOrder: preferredOrder,
			CreatedArchive:     createdArchive,
		}, mappedExecs[0]); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		return AddResult{Type: AddSuccess}
	}

	return AddResult{
		Type: AddNeedsSelection,
		PendingAddition: &PendingAddition{
			SoftwareName:       softwareName,
			InstallPath:        destDir,
			ArchivePath:        finalArchivePath,
			BackupPath:         backupPath,
			ExecutablePaths:    mappedExecs,
			PreferredSortOrder: preferredOrder,
			CreatedArchive:     createdArchive,
		},
	}
}

func (s *Service) addFromExecutableFile(
	exePath, installPath string,
	allowOverride bool,
	customName string,
) AddResult {
	base := filepath.Base(exePath)
	fallback := strings.TrimSuffix(base, filepath.Ext(base))
	if fallback == "" {
		fallback = base
	}
	nameInput := customName
	if nameInput == "" {
		nameInput = fallback
	}
	softwareName := SanitizeSoftwareName(nameInput)

	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return AddResult{Type: AddError, Err: err}
	}

	archiveFileName := buildArchiveFileName(softwareName, base)
	targetArchivePath := filepath.Join(archDir, archiveFileName)
	destDir := filepath.Join(installPath, softwareName)
	sameSourceAsDest := winutil.SamePath(exePath, targetArchivePath)

	installExists := dirExists(destDir)
	archiveExists := !sameSourceAsDest && fileExists(targetArchivePath)
	existingManaged, _ := s.findManagedByPaths(destDir, targetArchivePath)

	if !allowOverride && (installExists || archiveExists || existingManaged != nil) {
		return AddResult{
			Type: AddDuplicate,
			DuplicateInfo: &DuplicateInfo{
				SourceType:              SourceExecutable,
				SourcePath:              exePath,
				InstallRootPath:         installPath,
				IntendedName:            softwareName,
				TargetInstallPath:       destDir,
				TargetArchivePath:       targetArchivePath,
				InstallDirExists:        installExists,
				ArchiveExists:           archiveExists,
				ExistingManagedSoftware: existingManaged,
			},
		}
	}

	var preferredOrder *int
	if allowOverride {
		overrideArchive := targetArchivePath
		if sameSourceAsDest {
			overrideArchive = ""
		}
		preferredOrder, err = s.prepareForOverride(destDir, overrideArchive)
		if err != nil {
			return AddResult{Type: AddError, Err: err}
		}
	}

	createdArchive := false
	if !sameSourceAsDest {
		if err := copyFile(exePath, targetArchivePath); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		createdArchive = true
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		if createdArchive {
			_ = os.Remove(targetArchivePath)
		}
		return AddResult{Type: AddError, Err: err}
	}
	destExe := filepath.Join(destDir, base)
	if err := copyFile(exePath, destExe); err != nil {
		_ = os.RemoveAll(destDir)
		if createdArchive {
			_ = os.Remove(targetArchivePath)
		}
		return AddResult{Type: AddError, Err: err}
	}

	if err := s.CompleteSoftwareAddition(PendingAddition{
		SoftwareName:       softwareName,
		InstallPath:        destDir,
		ArchivePath:        targetArchivePath,
		PreferredSortOrder: preferredOrder,
		CreatedArchive:     createdArchive,
	}, destExe); err != nil {
		return AddResult{Type: AddError, Err: err}
	}
	return AddResult{Type: AddSuccess}
}

// CompleteSoftwareAddition 将用户选定主程序后的待添加软件写入 software_list.json。
func (s *Service) CompleteSoftwareAddition(pending PendingAddition, selectedExe string) error {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}

	// 去重：若同安装路径已存在记录，先移除避免重复条目。
	filtered := make([]model.Software, 0, len(list))
	for _, item := range list {
		if winutil.SamePath(item.InstallPath, pending.InstallPath) {
			continue
		}
		filtered = append(filtered, item)
	}
	list = filtered

	order := nextSortOrder(list)
	if pending.PreferredSortOrder != nil {
		order = *pending.PreferredSortOrder
		for i := range list {
			if list[i].SortOrder >= order {
				list[i].SortOrder++
			}
		}
	}

	name := pending.SoftwareName
	if name == "" {
		name = filepath.Base(pending.InstallPath)
	}
	backupPath := pending.BackupPath
	if backupPath == "" && s.isBackupFilePath(pending.ArchivePath) {
		backupPath = pending.ArchivePath
	}

	list = append(list, model.Software{
		ID:             newUUID(),
		Name:           name,
		InstallPath:    pending.InstallPath,
		ExecutablePath: selectedExe,
		ArchivePath:    pending.ArchivePath,
		BackupPath:     backupPath,
		Status:         model.StatusManaged,
		SortOrder:      order,
	})
	return s.saveManagedSoftware(list)
}

// CancelPendingAddition 安全清理待添加软件产生的临时文件；不会删除原本已存在的归档文件。
func (s *Service) CancelPendingAddition(pending PendingAddition) {
	if pending.InstallPath != "" {
		_ = os.RemoveAll(pending.InstallPath)
	}
	if pending.CreatedArchive && pending.ArchivePath != "" {
		_ = os.Remove(pending.ArchivePath)
	}
}

// RehostSoftware 当安装目录丢失而归档或备份仍存在时，从归档/备份重新解压恢复。
func (s *Service) RehostSoftware(sw model.Software) AddResult {
	if sw.InstallPath == "" {
		return AddResult{Type: AddError, Err: errors.New("安装路径未知，无法重新托管")}
	}
	sourceArchive := s.resolveRehostSource(sw)
	if sourceArchive == "" {
		return AddResult{Type: AddError, Err: errors.New("归档与备份文件均不存在，无法重新托管")}
	}

	format := archive.DetectFormat(sourceArchive)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(sourceArchive)), ".")
	allowed := s.settings.GetExecutableExtensions()

	parentDir := filepath.Dir(sw.InstallPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return AddResult{Type: AddError, Err: err}
	}

	stagingDir, err := os.MkdirTemp(parentDir, ".carrydock_staging_")
	if err != nil {
		return AddResult{Type: AddError, Err: err}
	}
	cleanupStaging := true
	defer func() {
		if cleanupStaging {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	if format != archive.FormatUnknown {
		if err := archive.Extract(sourceArchive, stagingDir, format); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		if s.settings.GetRemoveNestedFoldersEnabled() {
			FlattenRedundantTopDirectories(stagingDir)
		}
		execs := s.FindExecutablesInDirectory(stagingDir)
		if len(execs) == 0 {
			return AddResult{Type: AddError, Err: errors.New("未能在归档内容中发现可执行程序")}
		}
		if dirExists(sw.InstallPath) {
			_ = os.RemoveAll(sw.InstallPath)
		}
		if err := os.Rename(stagingDir, sw.InstallPath); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		cleanupStaging = false

		mapped := make([]string, 0, len(execs))
		for _, ex := range execs {
			if rel, relErr := filepath.Rel(stagingDir, ex); relErr == nil {
				mapped = append(mapped, filepath.Join(sw.InstallPath, rel))
			}
		}
		if len(mapped) == 1 {
			if err := s.CompleteSoftwareRehost(sw.ID, sw.InstallPath, sourceArchive, mapped[0]); err != nil {
				return AddResult{Type: AddError, Err: err}
			}
			return AddResult{Type: AddSuccess}
		}
		return AddResult{
			Type: AddNeedsSelection,
			PendingAddition: &PendingAddition{
				SoftwareName:       sw.Name,
				InstallPath:        sw.InstallPath,
				ArchivePath:        sourceArchive,
				ExecutablePaths:    mapped,
				ExistingSoftwareID: sw.ID,
				CreatedArchive:     false,
			},
		}
	} else if slices.Contains(allowed, ext) {
		destExe := filepath.Join(stagingDir, filepath.Base(sourceArchive))
		if err := copyFile(sourceArchive, destExe); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		if dirExists(sw.InstallPath) {
			_ = os.RemoveAll(sw.InstallPath)
		}
		if err := os.Rename(stagingDir, sw.InstallPath); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		cleanupStaging = false
		finalExe := filepath.Join(sw.InstallPath, filepath.Base(sourceArchive))
		if err := s.CompleteSoftwareRehost(sw.ID, sw.InstallPath, sourceArchive, finalExe); err != nil {
			return AddResult{Type: AddError, Err: err}
		}
		return AddResult{Type: AddSuccess}
	}
	return AddResult{Type: AddError, Err: fmt.Errorf("不支持的归档格式: %s", filepath.Ext(sourceArchive))}
}

func (s *Service) resolveRehostSource(sw model.Software) string {
	if sw.ArchivePath != "" && fileExists(sw.ArchivePath) {
		return sw.ArchivePath
	}
	if sw.BackupPath != "" && fileExists(sw.BackupPath) {
		return sw.BackupPath
	}
	// 在归档目录按文件名或名称前缀回退查找
	if archDir, err := s.ResolveArchiveDirectoryPath(); err == nil {
		expected := ""
		if sw.ArchivePath != "" {
			expected = filepath.Base(sw.ArchivePath)
		}
		var candidates []string
		if entries, err := os.ReadDir(archDir); err == nil {
			for _, ent := range entries {
				if ent.IsDir() {
					continue
				}
				if expected != "" && strings.EqualFold(ent.Name(), expected) {
					return filepath.Join(archDir, ent.Name())
				}
				if expected == "" && strings.HasPrefix(strings.ToLower(ent.Name()), strings.ToLower(sw.Name)) {
					candidates = append(candidates, filepath.Join(archDir, ent.Name()))
				}
			}
		}
		if len(candidates) == 1 {
			return candidates[0]
		}
	}
	// 最后回退查找最新的匹配备份文件
	if backupDir, err := s.ResolveBackupDirectoryPath(); err == nil {
		prefix := strings.ToLower(SanitizeSoftwareName(sw.Name) + "-")
		var matches []string
		if entries, err := os.ReadDir(backupDir); err == nil {
			for _, ent := range entries {
				if !ent.IsDir() && strings.HasPrefix(strings.ToLower(ent.Name()), prefix) {
					matches = append(matches, filepath.Join(backupDir, ent.Name()))
				}
			}
		}
		if len(matches) > 0 {
			sort.Strings(matches)
			return matches[len(matches)-1]
		}
	}
	return ""
}

// CompleteSoftwareRehost 在用户选定主程序后完成重新托管记录更新。
func (s *Service) CompleteSoftwareRehost(existingID, installPath, sourceArchive, selectedExe string) error {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}
	idx := -1
	for i := range list {
		if list[i].ID == existingID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("找不到需要更新的软件记录 (ID: %s)", existingID)
	}
	list[idx].InstallPath = installPath
	list[idx].ExecutablePath = selectedExe
	if s.isBackupFilePath(sourceArchive) {
		list[idx].BackupPath = sourceArchive
	} else if sourceArchive != "" {
		list[idx].ArchivePath = sourceArchive
	}
	return s.saveManagedSoftware(list)
}

// RenameSoftwareFolder 安全重命名/迁移软件安装文件夹，优先使用原子 os.Rename，并同步更新主程序路径。
func (s *Service) RenameSoftwareFolder(sw model.Software, newFolderName string) error {
	newFolderName = strings.TrimSpace(newFolderName)
	if newFolderName == "" {
		return errors.New("文件夹名称不能为空")
	}
	if strings.ContainsAny(newFolderName, `\/:*?"<>|`) || newFolderName == "." || newFolderName == ".." {
		return errors.New("文件夹名称包含非法字符或路径分隔符")
	}
	if isWindowsReservedName(newFolderName) {
		return fmt.Errorf("文件夹名称不能为系统保留名: %s", newFolderName)
	}

	oldInstallPath := sw.InstallPath
	if oldInstallPath == "" || !dirExists(oldInstallPath) {
		return errors.New("当前软件文件夹不存在，无法迁移")
	}
	parentDir := filepath.Dir(oldInstallPath)
	newInstallPath := filepath.Join(parentDir, newFolderName)

	if winutil.SamePath(oldInstallPath, newInstallPath) {
		return errors.New("新路径与当前路径相同")
	}
	if _, err := os.Stat(newInstallPath); err == nil {
		return errors.New("目标文件夹已存在，请选择其他名称")
	}

	// 同卷优先原子重命名；失败时回退到复制+删除。
	if err := os.Rename(oldInstallPath, newInstallPath); err != nil {
		if copyErr := copyDirectory(oldInstallPath, newInstallPath); copyErr != nil {
			_ = os.RemoveAll(newInstallPath)
			return fmt.Errorf("迁移文件夹失败: %w", copyErr)
		}
		if rmErr := os.RemoveAll(oldInstallPath); rmErr != nil {
			logger.Warn("删除旧目录失败 (%s): %v", oldInstallPath, rmErr)
		}
	}

	newExecPath := sw.ExecutablePath
	if sw.ExecutablePath != "" {
		if rel, err := filepath.Rel(oldInstallPath, sw.ExecutablePath); err == nil && !strings.HasPrefix(rel, "..") {
			newExecPath = filepath.Join(newInstallPath, rel)
		}
	}

	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].ID == sw.ID {
			if strings.EqualFold(list[i].Name, filepath.Base(oldInstallPath)) {
				list[i].Name = newFolderName
			}
			list[i].InstallPath = newInstallPath
			list[i].ExecutablePath = newExecPath
			return s.saveManagedSoftware(list)
		}
	}
	return fmt.Errorf("未找到指定软件记录 (ID: %s)", sw.ID)
}

// UpdateSoftwareExecutable 更新已托管软件的主程序可执行文件路径。
func (s *Service) UpdateSoftwareExecutable(softwareID, newExePath string) error {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].ID == softwareID {
			list[i].ExecutablePath = newExePath
			return s.saveManagedSoftware(list)
		}
	}
	return fmt.Errorf("未找到 ID 为 %s 的软件", softwareID)
}

// UpdateManagedSoftwareOrder 按给定 ID 顺序更新已托管软件排序。
func (s *Service) UpdateManagedSoftwareOrder(orderedIDs []string) error {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}
	orderMap := make(map[string]int, len(orderedIDs))
	for i, id := range orderedIDs {
		orderMap[id] = i
	}
	next := len(orderedIDs)
	for i := range list {
		if pos, ok := orderMap[list[i].ID]; ok {
			list[i].SortOrder = pos
		} else {
			list[i].SortOrder = next
			next++
		}
	}
	return s.saveManagedSoftware(list)
}

// DeleteSoftware 删除软件记录，并按选项删除安装目录或归档文件。
func (s *Service) DeleteSoftware(sw model.Software, deleteInstallDir, deleteArchive bool) error {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}

	if deleteInstallDir && sw.InstallPath != "" && dirExists(sw.InstallPath) {
		if err := os.RemoveAll(sw.InstallPath); err != nil {
			return fmt.Errorf("删除安装目录失败: %w", err)
		}
	}
	if deleteArchive && sw.ArchivePath != "" && fileExists(sw.ArchivePath) {
		if err := os.Remove(sw.ArchivePath); err != nil {
			return fmt.Errorf("删除归档文件失败: %w", err)
		}
	}

	// 若该条目在托管列表中存在，移除并保存。
	filtered := make([]model.Software, 0, len(list))
	removed := false
	for _, item := range list {
		if item.ID == sw.ID {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if removed {
		return s.saveManagedSoftware(filtered)
	}
	return nil
}

// ArchiveInstallSubdirectories 批量扫描安装目录一级子目录，纳管并可选创建备份，同时收集旧归档关联建议。
func (s *Service) ArchiveInstallSubdirectories(
	createBackup bool,
	onProgress func(done, total int, currentName string),
) (*BatchArchiveSummary, error) {
	installPath := s.settings.GetInstallPath()
	if installPath == "" {
		return nil, ErrInstallPathNotConfigured
	}
	if !dirExists(installPath) {
		return nil, fmt.Errorf("安装目录不存在：%s", installPath)
	}
	archDir, err := s.ResolveArchiveDirectoryPath()
	if err != nil {
		return nil, err
	}
	backupDir, err := s.ResolveBackupDirectoryPath()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(installPath)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		fullPath := filepath.Join(installPath, ent.Name())
		if strings.EqualFold(ent.Name(), defaultArchiveFolderName) ||
			strings.HasPrefix(ent.Name(), ".carrydock_staging_") ||
			winutil.SamePath(fullPath, archDir) ||
			winutil.SamePath(fullPath, backupDir) {
			continue
		}
		dirs = append(dirs, fullPath)
	}

	managed, err := s.LoadManagedSoftware()
	if err != nil {
		return nil, err
	}
	nextOrder := nextSortOrder(managed)

	// 预先列出归档根目录下的全部有效归档文件，避免循环内重复扫盘。
	var rootArchiveFiles []string
	if archEntries, err := os.ReadDir(archDir); err == nil {
		for _, ent := range archEntries {
			if ent.IsDir() {
				continue
			}
			full := filepath.Join(archDir, ent.Name())
			if archive.DetectFormat(full) != archive.FormatUnknown {
				rootArchiveFiles = append(rootArchiveFiles, full)
			}
		}
	}

	summary := &BatchArchiveSummary{
		ArchiveDirPath: backupDir,
		InstallDirPath: installPath,
	}
	total := len(dirs)
	if onProgress != nil {
		onProgress(0, total, "")
	}

	for i, dirPath := range dirs {
		baseName := filepath.Base(dirPath)
		if onProgress != nil {
			onProgress(i, total, baseName)
		}

		sanitized := SanitizeSoftwareName(baseName)
		sanitizedLower := strings.ToLower(sanitized)

		// 查找当前目录在归档根目录下的疑似匹配归档（未被其他已托管软件占用）
		var candidates []string
		for _, af := range rootArchiveFiles {
			bn := strings.ToLower(archive.StripArchiveExtension(af))
			if bn == sanitizedLower || strings.HasPrefix(bn, sanitizedLower+"-") || strings.HasPrefix(bn, sanitizedLower+"_") {
				occupied := false
				for _, m := range managed {
					if m.ArchivePath != "" && winutil.SamePath(m.ArchivePath, af) && !winutil.SamePath(m.InstallPath, dirPath) {
						occupied = true
						break
					}
				}
				if !occupied {
					candidates = append(candidates, af)
				}
			}
		}

		existingIdx := -1
		for idx := range managed {
			if winutil.SamePath(managed[idx].InstallPath, dirPath) {
				existingIdx = idx
				break
			}
		}

		// 若已关联有效归档，无需再次提示关联
		hasValidArchive := existingIdx >= 0 && managed[existingIdx].ArchivePath != "" && fileExists(managed[existingIdx].ArchivePath)

		var createdBackup string
		if len(candidates) > 0 && !hasValidArchive {
			summary.Suggestions = append(summary.Suggestions, ArchiveAssociationSuggestion{
				DisplayName: baseName,
				InstallPath: dirPath,
				Candidates:  candidates,
			})
		} else if createBackup {
			bp, bErr := s.CreateBackupFromDirectory(dirPath, baseName)
			if bErr != nil {
				logger.Error("批量创建备份失败 (%s): %v", dirPath, bErr)
				summary.Failures = append(summary.Failures, BatchArchiveFailure{Name: baseName, Error: bErr.Error()})
				if onProgress != nil {
					onProgress(i+1, total, baseName)
				}
				continue
			}
			createdBackup = bp
		}

		if existingIdx >= 0 {
			if managed[existingIdx].ExecutablePath == "" {
				execs := s.FindExecutablesInDirectory(dirPath)
				if len(execs) > 0 {
					managed[existingIdx].ExecutablePath = pickShallowestExecutable(execs, dirPath)
				}
			}
			if createdBackup != "" {
				managed[existingIdx].BackupPath = createdBackup
			}
		} else {
			execs := s.FindExecutablesInDirectory(dirPath)
			exe := ""
			if len(execs) > 0 {
				exe = pickShallowestExecutable(execs, dirPath)
			}
			managed = append(managed, model.Software{
				ID:             newUUID(),
				Name:           sanitized,
				InstallPath:    dirPath,
				ExecutablePath: exe,
				BackupPath:     createdBackup,
				Status:         model.StatusManaged,
				SortOrder:      nextOrder,
			})
			nextOrder++
		}

		summary.Archived = append(summary.Archived, baseName)
		if onProgress != nil {
			onProgress(i+1, total, baseName)
		}
	}

	if err := s.saveManagedSoftware(managed); err != nil {
		return nil, err
	}
	summary.Total = len(summary.Archived) + len(summary.SkippedExisting) + len(summary.Failures)
	return summary, nil
}

// ApplyArchiveAssociations 应用用户在扫描后的统一关联决策。
func (s *Service) ApplyArchiveAssociations(resolutions []ArchiveAssociationResolution, createBackupForUnselected bool) error {
	if len(resolutions) == 0 {
		return nil
	}
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return err
	}
	changed := false
	for _, r := range resolutions {
		idx := -1
		for i := range list {
			if winutil.SamePath(list[i].InstallPath, r.InstallPath) {
				idx = i
				break
			}
		}
		if idx < 0 {
			if !dirExists(r.InstallPath) {
				continue
			}
			execs := s.FindExecutablesInDirectory(r.InstallPath)
			exe := ""
			if len(execs) > 0 {
				exe = pickShallowestExecutable(execs, r.InstallPath)
			}
			sw := model.Software{
				ID:             newUUID(),
				Name:           SanitizeSoftwareName(filepath.Base(r.InstallPath)),
				InstallPath:    r.InstallPath,
				ExecutablePath: exe,
				Status:         model.StatusManaged,
				SortOrder:      nextSortOrder(list),
			}
			if r.SelectedArchivePath != "" {
				sw.ArchivePath = r.SelectedArchivePath
			} else if createBackupForUnselected {
				if bp, err := s.CreateBackupFromDirectory(r.InstallPath, sw.Name); err == nil {
					sw.BackupPath = bp
				}
			}
			list = append(list, sw)
			changed = true
			continue
		}

		if r.SelectedArchivePath != "" {
			list[idx].ArchivePath = r.SelectedArchivePath
			changed = true
		} else if createBackupForUnselected && (list[idx].BackupPath == "" || !fileExists(list[idx].BackupPath)) {
			if dirExists(list[idx].InstallPath) {
				if bp, err := s.CreateBackupFromDirectory(list[idx].InstallPath, list[idx].Name); err == nil {
					list[idx].BackupPath = bp
					changed = true
				}
			}
		}

		if list[idx].ExecutablePath == "" && dirExists(list[idx].InstallPath) {
			execs := s.FindExecutablesInDirectory(list[idx].InstallPath)
			if len(execs) > 0 {
				list[idx].ExecutablePath = pickShallowestExecutable(execs, list[idx].InstallPath)
				changed = true
			}
		}
	}
	if changed {
		return s.saveManagedSoftware(list)
	}
	return nil
}

// FindExecutablesInDirectory 在给定目录内按配置深度和扩展名扫描全部可执行文件。
func (s *Service) FindExecutablesInDirectory(root string) []string {
	maxDepth := s.settings.GetExecutableSearchMaxDepth()
	exts := s.settings.GetExecutableExtensions()
	allowed := make(map[string]struct{}, len(exts))
	for _, e := range exts {
		allowed[strings.ToLower(e)] = struct{}{}
	}
	seen := make(map[string]struct{})
	var results []string

	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, ent := range entries {
			full := filepath.Join(dir, ent.Name())
			if ent.IsDir() {
				if depth < maxDepth {
					walk(full, depth+1)
				}
				continue
			}
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(ent.Name())), ".")
			if _, ok := allowed[ext]; ok {
				if _, dup := seen[full]; !dup {
					seen[full] = struct{}{}
					results = append(results, full)
				}
			}
		}
	}
	walk(root, 0)
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i]) < strings.ToLower(results[j])
	})
	return results
}

// FlattenRedundantTopDirectories 循环展开多余的单一顶层嵌套文件夹。
// 修复了原版仅展开一层以及子目录与自身同名（如 foo/foo/app.exe）时命名冲突的 bug。
func FlattenRedundantTopDirectories(root string) bool {
	flattenedAny := false
	for range 8 {
		if !flattenOnce(root) {
			break
		}
		flattenedAny = true
	}
	return flattenedAny
}

func flattenOnce(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) == 0 {
		return false
	}

	var retained []os.DirEntry
	for _, ent := range entries {
		if isIgnorableArchiveArtifact(ent.Name(), ent.IsDir()) {
			_ = os.RemoveAll(filepath.Join(root, ent.Name()))
			continue
		}
		retained = append(retained, ent)
	}
	if len(retained) != 1 || !retained[0].IsDir() {
		return false
	}

	nestedName := retained[0].Name()
	if nestedName == "" || nestedName == "." || nestedName == ".." || strings.HasPrefix(nestedName, ".") {
		return false
	}
	nestedPath := filepath.Join(root, nestedName)
	nestedEntries, err := os.ReadDir(nestedPath)
	if err != nil {
		return false
	}
	if len(nestedEntries) == 0 {
		_ = os.RemoveAll(nestedPath)
		return true
	}

	// 先将 nestedPath 重命名为唯一的临时名，彻底消除 foo/foo 同名冲突！
	tmpNested := filepath.Join(root, fmt.Sprintf(".carrydock_flatten_%d", time.Now().UnixNano()))
	if err := os.Rename(nestedPath, tmpNested); err != nil {
		return false
	}

	for _, child := range nestedEntries {
		src := filepath.Join(tmpNested, child.Name())
		dst := filepath.Join(root, child.Name())
		if err := os.Rename(src, dst); err != nil {
			logger.Warn("展开嵌套目录移动失败 (%s -> %s): %v", src, dst, err)
			return false
		}
	}
	_ = os.RemoveAll(tmpNested)
	return true
}

func isIgnorableArchiveArtifact(name string, isDir bool) bool {
	lower := strings.ToLower(name)
	if isDir && lower == "__macosx" {
		return true
	}
	if lower == ".ds_store" || lower == "thumbs.db" || lower == "desktop.ini" {
		return true
	}
	return strings.HasPrefix(lower, "._")
}

func (s *Service) findManagedByPaths(installPath, archivePath string) (*model.Software, error) {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return nil, err
	}
	for i := range list {
		matchInstall := installPath != "" && winutil.SamePath(list[i].InstallPath, installPath)
		matchArchive := archivePath != "" && winutil.SamePath(list[i].ArchivePath, archivePath)
		if matchInstall || matchArchive {
			cp := list[i]
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *Service) prepareForOverride(installPath, archivePath string) (*int, error) {
	list, err := s.LoadManagedSoftware()
	if err != nil {
		return nil, err
	}
	var removedOrder *int
	retained := make([]model.Software, 0, len(list))
	removed := false
	for _, item := range list {
		matchInstall := installPath != "" && winutil.SamePath(item.InstallPath, installPath)
		matchArchive := archivePath != "" && winutil.SamePath(item.ArchivePath, archivePath)
		if matchInstall || matchArchive {
			order := item.SortOrder
			removedOrder = &order
			removed = true
			continue
		}
		retained = append(retained, item)
	}
	if removed {
		if err := s.saveManagedSoftware(retained); err != nil {
			return nil, err
		}
	}
	if installPath != "" && dirExists(installPath) {
		if err := os.RemoveAll(installPath); err != nil {
			return nil, fmt.Errorf("清理旧安装目录失败: %w", err)
		}
	}
	if archivePath != "" && fileExists(archivePath) {
		if err := os.Remove(archivePath); err != nil {
			return nil, fmt.Errorf("清理旧归档文件失败: %w", err)
		}
	}
	return removedOrder, nil
}

func (s *Service) isBackupFilePath(p string) bool {
	if p == "" {
		return false
	}
	return strings.EqualFold(filepath.Base(filepath.Dir(filepath.Clean(p))), backupSubfolderName)
}

var (
	invalidNameCharsRe = regexp.MustCompile(`[\\/:*?"<>|]`)
	backupTimestampRe  = regexp.MustCompile(`-\d{8}_\d{6}(?:_\d+)?$`)
)

// SanitizeSoftwareName 清洗软件名，去除 Windows 非法字符、首尾点/空格，并规避系统保留设备名。
func SanitizeSoftwareName(name string) string {
	s := strings.TrimSpace(name)
	s = invalidNameCharsRe.ReplaceAllString(s, "_")
	s = strings.TrimRight(s, ". ")
	if s == "" || s == "." || s == ".." {
		return fmt.Sprintf("软件_%d", time.Now().UnixMilli())
	}
	if isWindowsReservedName(s) {
		s = "_" + s
	}
	return s
}

func isWindowsReservedName(name string) bool {
	upper := strings.ToUpper(strings.TrimSuffix(strings.TrimSpace(name), filepath.Ext(name)))
	switch upper {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func stripBackupTimestamp(name string) string {
	return backupTimestampRe.ReplaceAllString(name, "")
}

func buildArchiveFileName(sanitizedName, sourceFileName string) string {
	ext := archive.CompoundExtension(sourceFileName)
	if ext == "" {
		return sanitizedName
	}
	if strings.HasSuffix(strings.ToLower(sanitizedName), strings.ToLower(ext)) {
		return sanitizedName
	}
	return sanitizedName + ext
}

func pickShallowestExecutable(executables []string, root string) string {
	if len(executables) == 0 {
		return ""
	}
	best := executables[0]
	bestDepth := relativeDepth(best, root)
	for _, cand := range executables[1:] {
		d := relativeDepth(cand, root)
		if d < bestDepth || (d == bestDepth && strings.ToLower(cand) < strings.ToLower(best)) {
			best = cand
			bestDepth = d
		}
	}
	return best
}

func relativeDepth(path, root string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return strings.Count(filepath.ToSlash(path), "/")
	}
	return strings.Count(filepath.ToSlash(rel), "/") + 1
}

func nextSortOrder(list []model.Software) int {
	maxOrder := -1
	for _, s := range list {
		if s.Status == model.StatusManaged && s.SortOrder > maxOrder {
			maxOrder = s.SortOrder
		}
	}
	return maxOrder + 1
}

func collectZipNames(dir string) []string {
	if dir == "" || !dirExists(dir) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, ent := range entries {
		if !ent.IsDir() && strings.EqualFold(filepath.Ext(ent.Name()), ".zip") {
			out = append(out, ent.Name())
		}
	}
	return out
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func copyDirectory(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
