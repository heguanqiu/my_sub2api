package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakePluginStore struct {
	uploaded map[string][]byte
	deleted  []string
}

func newFakePluginStore() *fakePluginStore {
	return &fakePluginStore{uploaded: map[string][]byte{}}
}

func (f *fakePluginStore) Upload(_ context.Context, key string, body io.Reader, _ string) (int64, error) {
	b, _ := io.ReadAll(body)
	f.uploaded[key] = b
	return int64(len(b)), nil
}
func (f *fakePluginStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("download-body")), nil
}
func (f *fakePluginStore) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}
func (f *fakePluginStore) PresignURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.test/" + key, nil
}
func (f *fakePluginStore) HeadBucket(_ context.Context) error { return nil }

type fakePluginProvider struct{ store *fakePluginStore }

func (p *fakePluginProvider) Store(_ context.Context) (PluginObjectStore, error) {
	return p.store, nil
}

type fakePluginRepo struct {
	items     map[int64]*Plugin
	nextID    int64
	increment int
}

func newFakePluginRepo() *fakePluginRepo {
	return &fakePluginRepo{items: map[int64]*Plugin{}, nextID: 1}
}

func (r *fakePluginRepo) Create(_ context.Context, p *Plugin) error {
	p.ID = r.nextID
	r.nextID++
	cp := *p
	r.items[p.ID] = &cp
	return nil
}
func (r *fakePluginRepo) GetByID(_ context.Context, id int64) (*Plugin, error) {
	if p, ok := r.items[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, ErrPluginNotFound
}
func (r *fakePluginRepo) Update(_ context.Context, p *Plugin) error {
	if _, ok := r.items[p.ID]; !ok {
		return ErrPluginNotFound
	}
	cp := *p
	r.items[p.ID] = &cp
	return nil
}
func (r *fakePluginRepo) Delete(_ context.Context, id int64) error {
	delete(r.items, id)
	return nil
}
func (r *fakePluginRepo) List(_ context.Context, _ pagination.PaginationParams, _ PluginListFilters) ([]Plugin, *pagination.PaginationResult, error) {
	out := make([]Plugin, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, *p)
	}
	return out, &pagination.PaginationResult{Total: int64(len(out))}, nil
}
func (r *fakePluginRepo) ListPublished(_ context.Context, _ string) ([]Plugin, error) {
	out := make([]Plugin, 0)
	for _, p := range r.items {
		if p.Status == PluginStatusPublished {
			out = append(out, *p)
		}
	}
	return out, nil
}
func (r *fakePluginRepo) IncrementDownloadCount(_ context.Context, id int64) error {
	p, ok := r.items[id]
	if !ok {
		return ErrPluginNotFound
	}
	p.DownloadCount++
	r.increment++
	return nil
}

func newTestPluginService() (*PluginService, *fakePluginRepo, *fakePluginStore) {
	store := newFakePluginStore()
	repo := newFakePluginRepo()
	svc := NewPluginService(repo, &fakePluginProvider{store: store})
	return svc, repo, store
}

// --- tests ---

func TestPluginService_UploadObject_PrefixByKind(t *testing.T) {
	svc, _, store := newTestPluginService()
	ctx := context.Background()

	pkg, err := svc.UploadObject(ctx, "package", "tool.zip", "application/zip", strings.NewReader("data"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(pkg.Key, "plugins/files/"), "got %s", pkg.Key)
	require.Equal(t, "tool.zip", pkg.FileName)
	require.Equal(t, int64(4), pkg.Size)

	icon, err := svc.UploadObject(ctx, "icon", "logo.png", "image/png", strings.NewReader("img"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(icon.Key, "plugins/icons/"), "got %s", icon.Key)

	require.Len(t, store.uploaded, 2)
}

func TestPluginService_PrepareDownload_IncrementsAndPresigns(t *testing.T) {
	svc, repo, _ := newTestPluginService()
	ctx := context.Background()

	p, err := svc.Create(ctx, &CreatePluginInput{
		Name: "P", Status: PluginStatusPublished, FileKey: "plugins/files/x-tool.zip",
	})
	require.NoError(t, err)

	url, err := svc.PrepareDownload(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "https://example.test/plugins/files/x-tool.zip", url)
	require.Equal(t, 1, repo.increment)

	got, _ := repo.GetByID(ctx, p.ID)
	require.Equal(t, int64(1), got.DownloadCount)
}

func TestPluginService_PrepareDownload_DraftReturnsNotFound(t *testing.T) {
	svc, _, _ := newTestPluginService()
	ctx := context.Background()

	p, err := svc.Create(ctx, &CreatePluginInput{
		Name: "P", Status: PluginStatusDraft, FileKey: "plugins/files/x-tool.zip",
	})
	require.NoError(t, err)

	_, err = svc.PrepareDownload(ctx, p.ID)
	require.True(t, errors.Is(err, ErrPluginNotFound))
}

func TestPluginService_OpenDownload_IncrementsAndReturnsBody(t *testing.T) {
	svc, repo, _ := newTestPluginService()
	ctx := context.Background()

	p, err := svc.Create(ctx, &CreatePluginInput{
		Name: "P", Status: PluginStatusPublished, FileKey: "plugins/files/x-tool.zip", FileName: "tool.zip", FileSize: 13,
	})
	require.NoError(t, err)

	download, err := svc.OpenDownload(ctx, p.ID)
	require.NoError(t, err)
	defer download.Body.Close()

	body, err := io.ReadAll(download.Body)
	require.NoError(t, err)
	require.Equal(t, "download-body", string(body))
	require.Equal(t, "tool.zip", download.FileName)
	require.Equal(t, int64(13), download.FileSize)
	require.Equal(t, int64(13), download.ContentLength)
	require.Equal(t, 1, repo.increment)
}

func TestPluginService_Delete_RemovesObjects(t *testing.T) {
	svc, _, store := newTestPluginService()
	ctx := context.Background()

	p, err := svc.Create(ctx, &CreatePluginInput{
		Name: "P", Status: PluginStatusPublished,
		FileKey: "plugins/files/x-tool.zip", IconKey: "plugins/icons/y-logo.png",
	})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, p.ID))
	require.Contains(t, store.deleted, "plugins/files/x-tool.zip")
	require.Contains(t, store.deleted, "plugins/icons/y-logo.png")
}
