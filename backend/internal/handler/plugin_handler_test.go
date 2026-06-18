package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"
)

// --- fakes ---

type pubPluginStore struct{}

func (pubPluginStore) Upload(_ context.Context, _ string, _ io.Reader, _ string) (int64, error) {
	return 0, nil
}
func (pubPluginStore) Download(_ context.Context, _ string) (io.ReadCloser, error) { return nil, nil }
func (pubPluginStore) Delete(_ context.Context, _ string) error                    { return nil }
func (pubPluginStore) PresignURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}
func (pubPluginStore) HeadBucket(_ context.Context) error { return nil }

type pubPluginProvider struct{}

func (pubPluginProvider) Store(_ context.Context) (service.PluginObjectStore, error) {
	return pubPluginStore{}, nil
}

type pubPluginRepo struct {
	incremented int
	plugins     []service.Plugin
}

func (r *pubPluginRepo) Create(_ context.Context, _ *service.Plugin) error { return nil }
func (r *pubPluginRepo) GetByID(_ context.Context, id int64) (*service.Plugin, error) {
	for i := range r.plugins {
		if r.plugins[i].ID == id {
			p := r.plugins[i]
			return &p, nil
		}
	}
	return nil, service.ErrPluginNotFound
}
func (r *pubPluginRepo) Update(_ context.Context, _ *service.Plugin) error { return nil }
func (r *pubPluginRepo) Delete(_ context.Context, _ int64) error          { return nil }
func (r *pubPluginRepo) List(_ context.Context, _ pagination.PaginationParams, _ service.PluginListFilters) ([]service.Plugin, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{}, nil
}
func (r *pubPluginRepo) ListPublished(_ context.Context, _ string) ([]service.Plugin, error) {
	out := make([]service.Plugin, 0)
	for i := range r.plugins {
		if r.plugins[i].Status == service.PluginStatusPublished {
			out = append(out, r.plugins[i])
		}
	}
	return out, nil
}
func (r *pubPluginRepo) IncrementDownloadCount(_ context.Context, _ int64) error {
	r.incremented++
	return nil
}

func newPublicPluginHandler(plugins []service.Plugin) (*PluginHandler, *pubPluginRepo) {
	gin.SetMode(gin.TestMode)
	repo := &pubPluginRepo{plugins: plugins}
	svc := service.NewPluginService(repo, pubPluginProvider{})
	return NewPluginHandler(svc), repo
}

func TestPublicPluginHandler_List_OnlyPublished(t *testing.T) {
	handler, _ := newPublicPluginHandler([]service.Plugin{
		{ID: 1, Name: "pub", Status: service.PluginStatusPublished, FileName: "a.zip"},
		{ID: 2, Name: "draft", Status: service.PluginStatusDraft},
	})
	router := gin.New()
	router.GET("/api/v1/plugins", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "pub")
	require.NotContains(t, body, "draft")
	require.Contains(t, body, "/api/v1/plugins/1/download")
}

func TestPublicPluginHandler_Download_Redirects(t *testing.T) {
	handler, repo := newPublicPluginHandler([]service.Plugin{
		{ID: 1, Name: "pub", Status: service.PluginStatusPublished, FileKey: "plugins/files/x-a.zip"},
	})
	router := gin.New()
	router.GET("/api/v1/plugins/:id/download", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/1/download", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "https://example.test/plugins/files/x-a.zip", rec.Header().Get("Location"))
	require.Equal(t, 1, repo.incremented)
}

func TestPublicPluginHandler_Download_NotFound(t *testing.T) {
	handler, _ := newPublicPluginHandler([]service.Plugin{
		{ID: 1, Name: "draft", Status: service.PluginStatusDraft, FileKey: "plugins/files/x-a.zip"},
	})
	router := gin.New()
	router.GET("/api/v1/plugins/:id/download", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/1/download", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
