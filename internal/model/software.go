package model

// SoftwareStatus 表示软件条目的托管状态。
type SoftwareStatus int

const (
	StatusManaged SoftwareStatus = iota
	StatusUnknownInstall
	StatusUnknownArchive
)

// Software 表示一个绿色软件或未托管目录/归档条目。
// JSON 字段与旧版 Flutter CarryDock 的 software_list.json 保持完全兼容。
type Software struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	InstallPath    string `json:"installPath"`
	ExecutablePath string `json:"executablePath"`
	ArchivePath    string `json:"archivePath"`
	BackupPath     string `json:"backupPath"`
	IconPath       string `json:"iconPath,omitempty"`
	SortOrder      int    `json:"sortOrder"`

	// 以下为运行时状态字段，不持久化到 software_list.json。
	ArchiveExists   bool           `json:"-"`
	InstallExists   bool           `json:"-"`
	Status          SoftwareStatus `json:"-"`
	IsBackupArchive bool           `json:"-"`
}
