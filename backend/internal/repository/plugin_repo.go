package repository

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/plugin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type pluginRepository struct {
	client *dbent.Client
}

func NewPluginRepository(client *dbent.Client) service.PluginRepository {
	return &pluginRepository{client: client}
}

func (r *pluginRepository) Create(ctx context.Context, p *service.Plugin) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Plugin.Create().
		SetName(p.Name).
		SetDescription(p.Description).
		SetVersion(p.Version).
		SetCategory(p.Category).
		SetPlatform(p.Platform).
		SetIconKey(p.IconKey).
		SetFileKey(p.FileKey).
		SetFileName(p.FileName).
		SetFileSize(p.FileSize).
		SetStatus(p.Status).
		SetSortWeight(p.SortWeight)
	if p.CreatedBy != nil {
		builder.SetCreatedBy(*p.CreatedBy)
	}
	if p.UpdatedBy != nil {
		builder.SetUpdatedBy(*p.UpdatedBy)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	p.ID = created.ID
	p.DownloadCount = created.DownloadCount
	p.CreatedAt = created.CreatedAt
	p.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *pluginRepository) GetByID(ctx context.Context, id int64) (*service.Plugin, error) {
	m, err := r.client.Plugin.Query().Where(plugin.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrPluginNotFound, nil)
	}
	return pluginEntityToService(m), nil
}

func (r *pluginRepository) Update(ctx context.Context, p *service.Plugin) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Plugin.UpdateOneID(p.ID).
		SetName(p.Name).
		SetDescription(p.Description).
		SetVersion(p.Version).
		SetCategory(p.Category).
		SetPlatform(p.Platform).
		SetIconKey(p.IconKey).
		SetFileKey(p.FileKey).
		SetFileName(p.FileName).
		SetFileSize(p.FileSize).
		SetStatus(p.Status).
		SetSortWeight(p.SortWeight)
	if p.UpdatedBy != nil {
		builder.SetUpdatedBy(*p.UpdatedBy)
	} else {
		builder.ClearUpdatedBy()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrPluginNotFound, nil)
	}
	p.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *pluginRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.Plugin.Delete().Where(plugin.IDEQ(id)).Exec(ctx)
	return err
}

func (r *pluginRepository) IncrementDownloadCount(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	return client.Plugin.UpdateOneID(id).AddDownloadCount(1).Exec(ctx)
}

func (r *pluginRepository) ListPublished(ctx context.Context, category string) ([]service.Plugin, error) {
	q := r.client.Plugin.Query().Where(plugin.StatusEQ(service.PluginStatusPublished))
	if category != "" {
		q = q.Where(plugin.CategoryEQ(category))
	}
	items, err := q.
		Order(dbent.Desc(plugin.FieldSortWeight)).
		Order(dbent.Desc(plugin.FieldCreatedAt)).
		Order(dbent.Desc(plugin.FieldID)).
		Limit(500).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return pluginEntitiesToService(items), nil
}

func (r *pluginRepository) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters service.PluginListFilters,
) ([]service.Plugin, *pagination.PaginationResult, error) {
	q := r.client.Plugin.Query()
	if filters.Status != "" {
		q = q.Where(plugin.StatusEQ(filters.Status))
	}
	if filters.Category != "" {
		q = q.Where(plugin.CategoryEQ(filters.Category))
	}
	if filters.Search != "" {
		q = q.Where(plugin.Or(
			plugin.NameContainsFold(filters.Search),
			plugin.DescriptionContainsFold(filters.Search),
		))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	itemsQuery := q.Offset(params.Offset()).Limit(params.Limit())
	for _, order := range pluginListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}
	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pluginEntitiesToService(items), paginationResultFromTotal(int64(total), params), nil
}

func pluginListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := plugin.FieldCreatedAt
	switch sortBy {
	case "name":
		field = plugin.FieldName
	case "status":
		field = plugin.FieldStatus
	case "download_count":
		field = plugin.FieldDownloadCount
	case "sort_weight":
		field = plugin.FieldSortWeight
	case "id":
		field = plugin.FieldID
	case "", "created_at":
		field = plugin.FieldCreatedAt
	}
	if sortOrder == pagination.SortOrderAsc {
		return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(plugin.FieldID)}
	}
	return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(plugin.FieldID)}
}

func pluginEntityToService(m *dbent.Plugin) *service.Plugin {
	if m == nil {
		return nil
	}
	return &service.Plugin{
		ID:            m.ID,
		Name:          m.Name,
		Description:   m.Description,
		Version:       m.Version,
		Category:      m.Category,
		Platform:      m.Platform,
		IconKey:       m.IconKey,
		FileKey:       m.FileKey,
		FileName:      m.FileName,
		FileSize:      m.FileSize,
		DownloadCount: m.DownloadCount,
		Status:        m.Status,
		SortWeight:    m.SortWeight,
		CreatedBy:     m.CreatedBy,
		UpdatedBy:     m.UpdatedBy,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func pluginEntitiesToService(models []*dbent.Plugin) []service.Plugin {
	out := make([]service.Plugin, 0, len(models))
	for i := range models {
		if s := pluginEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
