package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// Plugin 领域类型别名
type Plugin = domain.Plugin

// Plugin 状态常量（re-export domain）
const (
	PluginStatusDraft     = domain.PluginStatusDraft
	PluginStatusPublished = domain.PluginStatusPublished
)

// Plugin 平台常量（re-export domain）
const (
	PluginPlatformAll     = domain.PluginPlatformAll
	PluginPlatformWindows = domain.PluginPlatformWindows
	PluginPlatformMacOS   = domain.PluginPlatformMacOS
	PluginPlatformLinux   = domain.PluginPlatformLinux
)

// 领域错误
var (
	ErrPluginNotFound             = infraerrors.NotFound("PLUGIN_NOT_FOUND", "plugin not found")
	ErrPluginStorageNotConfigured = infraerrors.BadRequest("PLUGIN_STORAGE_NOT_CONFIGURED", "请先在备份设置中配置 S3 存储")
)

// PluginListFilters 管理列表过滤条件
type PluginListFilters struct {
	Status   string
	Category string
	Search   string
}

// PluginRepository 插件持久化接口
type PluginRepository interface {
	Create(ctx context.Context, p *Plugin) error
	GetByID(ctx context.Context, id int64) (*Plugin, error)
	Update(ctx context.Context, p *Plugin) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params pagination.PaginationParams, filters PluginListFilters) ([]Plugin, *pagination.PaginationResult, error)
	ListPublished(ctx context.Context, category string) ([]Plugin, error)
	IncrementDownloadCount(ctx context.Context, id int64) error
}
