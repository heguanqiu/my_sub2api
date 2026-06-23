package domain

import "time"

// Plugin 状态
const (
	PluginStatusDraft     = "draft"
	PluginStatusPublished = "published"
)

// Plugin 适用平台
const (
	PluginPlatformAll     = "all"
	PluginPlatformWindows = "windows"
	PluginPlatformMacOS   = "macos"
	PluginPlatformLinux   = "linux"
)

// Plugin 插件中心条目领域模型
type Plugin struct {
	ID            int64
	Name          string
	Description   string
	Version       string
	Category      string
	Platform      string
	IconKey       string
	FileKey       string
	FileName      string
	FileSize      int64
	DownloadCount int64
	Status        string
	SortWeight    int
	CreatedBy     *int64
	UpdatedBy     *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
