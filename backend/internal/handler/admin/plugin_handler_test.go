package admin

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"
)

// --- fakes ---

type pluginStoreStub struct{}

func (pluginStoreStub) Upload(_ context.Context, key string, body io.Reader, _ string) (int64, error) {
	b, _ := io.ReadAll(body)
	return int64(len(b)), nil
}
func (pluginStoreStub) Download(_ context.Context, _ string) (io.ReadCloser, error) { return nil, nil }
func (pluginStoreStub) Delete(_ context.Context, _ string) error                   { return nil }
func (pluginStoreStub) PresignURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}
func (pluginStoreStub) HeadBucket(_ context.Context) error { return nil }

type pluginProviderStub struct{}

func (pluginProviderStub) Store(_ context.Context) (service.PluginObjectStore, error) {
	return pluginStoreStub{}, nil
}

type pluginRepoStub struct {
	created *service.Plugin
}

func (r *pluginRepoStub) Create(_ context.Context, p *service.Plugin) error {
	p.ID = 1
	r.created = p
	return nil
}
func (r *pluginRepoStub) GetByID(_ context.Context, id int64) (*service.Plugin, error) {
	return &service.Plugin{ID: id, Name: "P", Status: service.PluginStatusPublished}, nil
}
func (r *pluginRepoStub) Update(_ context.Context, _ *service.Plugin) error { return nil }
func (r *pluginRepoStub) Delete(_ context.Context, _ int64) error          { return nil }
func (r *pluginRepoStub) List(_ context.Context, _ pagination.PaginationParams, _ service.PluginListFilters) ([]service.Plugin, *pagination.PaginationResult, error) {
	return []service.Plugin{}, &pagination.PaginationResult{Total: 0}, nil
}
func (r *pluginRepoStub) ListPublished(_ context.Context, _ string) ([]service.Plugin, error) {
	return nil, nil
}
func (r *pluginRepoStub) IncrementDownloadCount(_ context.Context, _ int64) error { return nil }

func newPluginTestHandler() (*PluginHandler, *pluginRepoStub) {
	gin.SetMode(gin.TestMode)
	repo := &pluginRepoStub{}
	svc := service.NewPluginService(repo, pluginProviderStub{})
	return NewPluginHandler(svc), repo
}

func buildUploadRequest(t *testing.T, kind, filename string, size int) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	require.NoError(t, w.WriteField("kind", kind))
	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write(bytes.Repeat([]byte("a"), size))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestPluginHandler_Upload_RejectsOversizePackage(t *testing.T) {
	handler, _ := newPluginTestHandler()
	router := gin.New()
	router.POST("/admin/plugins/upload", handler.Upload)

	// 101MB
	req := buildUploadRequest(t, "package", "big.zip", (100<<20)+1)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "100MB")
}

func TestPluginHandler_Upload_RejectsBadExtension(t *testing.T) {
	handler, _ := newPluginTestHandler()
	router := gin.New()
	router.POST("/admin/plugins/upload", handler.Upload)

	req := buildUploadRequest(t, "package", "evil.exe", 16)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPluginHandler_Upload_AcceptsZip(t *testing.T) {
	handler, _ := newPluginTestHandler()
	router := gin.New()
	router.POST("/admin/plugins/upload", handler.Upload)

	req := buildUploadRequest(t, "package", "tool.zip", 16)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "plugins/files/")
}

func TestPluginHandler_Create_RequiresName(t *testing.T) {
	handler, _ := newPluginTestHandler()
	router := gin.New()
	router.POST("/admin/plugins", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/admin/plugins", strings.NewReader(`{"version":"v1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
