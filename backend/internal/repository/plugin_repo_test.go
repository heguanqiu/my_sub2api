package repository

import (
	"context"
	"database/sql"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newPluginRepoSQLite(t *testing.T) *pluginRepository {
	t.Helper()

	db, err := sql.Open("sqlite", "file:plugin_repo?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return &pluginRepository{client: client}
}

func TestPluginRepository_CreateAndGetByID(t *testing.T) {
	repo := newPluginRepoSQLite(t)
	ctx := context.Background()

	actor := int64(7)
	p := &service.Plugin{
		Name:        "My Plugin",
		Description: "desc",
		Version:     "v1.0.0",
		Category:    "Claude Code",
		Platform:    service.PluginPlatformAll,
		FileKey:     "plugins/files/abc-pkg.zip",
		FileName:    "pkg.zip",
		FileSize:    1234,
		Status:      service.PluginStatusPublished,
		SortWeight:  5,
		CreatedBy:   &actor,
		UpdatedBy:   &actor,
	}
	require.NoError(t, repo.Create(ctx, p))
	require.NotZero(t, p.ID)

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "My Plugin", got.Name)
	require.Equal(t, "v1.0.0", got.Version)
	require.Equal(t, "Claude Code", got.Category)
	require.Equal(t, int64(1234), got.FileSize)
	require.Equal(t, service.PluginStatusPublished, got.Status)
	require.Equal(t, 5, got.SortWeight)
	require.Equal(t, int64(0), got.DownloadCount)
}

func TestPluginRepository_GetByID_NotFound(t *testing.T) {
	repo := newPluginRepoSQLite(t)
	_, err := repo.GetByID(context.Background(), 99999)
	require.ErrorIs(t, err, service.ErrPluginNotFound)
}

func TestPluginRepository_IncrementDownloadCount(t *testing.T) {
	repo := newPluginRepoSQLite(t)
	ctx := context.Background()

	p := &service.Plugin{Name: "P", Status: service.PluginStatusPublished, Platform: service.PluginPlatformAll}
	require.NoError(t, repo.Create(ctx, p))

	require.NoError(t, repo.IncrementDownloadCount(ctx, p.ID))
	require.NoError(t, repo.IncrementDownloadCount(ctx, p.ID))

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), got.DownloadCount)
}

func TestPluginRepository_ListPublished_FiltersDraft(t *testing.T) {
	repo := newPluginRepoSQLite(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &service.Plugin{Name: "pub-a", Status: service.PluginStatusPublished, Platform: service.PluginPlatformAll, Category: "cc", SortWeight: 1}))
	require.NoError(t, repo.Create(ctx, &service.Plugin{Name: "pub-b", Status: service.PluginStatusPublished, Platform: service.PluginPlatformAll, Category: "codex", SortWeight: 9}))
	require.NoError(t, repo.Create(ctx, &service.Plugin{Name: "draft-c", Status: service.PluginStatusDraft, Platform: service.PluginPlatformAll, Category: "cc"}))

	all, err := repo.ListPublished(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 2)
	// 排序：sort_weight desc → pub-b 在前
	require.Equal(t, "pub-b", all[0].Name)

	filtered, err := repo.ListPublished(ctx, "cc")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, "pub-a", filtered[0].Name)
}

func TestPluginRepository_List_SearchAndPaginate(t *testing.T) {
	repo := newPluginRepoSQLite(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &service.Plugin{Name: "Alpha tool", Description: "search me", Status: service.PluginStatusPublished, Platform: service.PluginPlatformAll}))
	require.NoError(t, repo.Create(ctx, &service.Plugin{Name: "Beta tool", Description: "nothing", Status: service.PluginStatusDraft, Platform: service.PluginPlatformAll}))

	params := pagination.PaginationParams{Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: "desc"}

	items, result, err := repo.List(ctx, params, service.PluginListFilters{Search: "search me"})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, items, 1)
	require.Equal(t, "Alpha tool", items[0].Name)

	items2, result2, err := repo.List(ctx, params, service.PluginListFilters{Status: service.PluginStatusDraft})
	require.NoError(t, err)
	require.Equal(t, int64(1), result2.Total)
	require.Len(t, items2, 1)
	require.Equal(t, "Beta tool", items2[0].Name)
}
