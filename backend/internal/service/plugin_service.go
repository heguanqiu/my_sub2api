package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"

	"github.com/google/uuid"
)

const (
	pluginFilePrefix = "plugins/files/"
	pluginIconPrefix = "plugins/icons/"
	pluginURLExpiry  = 5 * time.Minute
)

type PluginService struct {
	repo          PluginRepository
	storeProvider PluginStoreProvider
	httpClient    *http.Client
}

func NewPluginService(repo PluginRepository, storeProvider PluginStoreProvider) *PluginService {
	return &PluginService{
		repo:          repo,
		storeProvider: storeProvider,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

// UploadResult 上传后返回给前端的元数据
type UploadResult struct {
	Key      string
	FileName string
	Size     int64
}

type PluginDownload struct {
	Body          io.ReadCloser
	FileName      string
	FileSize      int64
	ContentType   string
	ContentLength int64
}

// UploadObject 上传插件包或图标，返回 key/原名/大小
func (s *PluginService) UploadObject(ctx context.Context, kind, fileName, contentType string, body io.Reader) (*UploadResult, error) {
	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return nil, err
	}
	prefix := pluginFilePrefix
	if kind == "icon" {
		prefix = pluginIconPrefix
	}
	key := prefix + uuid.NewString() + "-" + path.Base(fileName)
	size, err := store.Upload(ctx, key, body, contentType)
	if err != nil {
		return nil, fmt.Errorf("upload object: %w", err)
	}
	return &UploadResult{Key: key, FileName: path.Base(fileName), Size: size}, nil
}

type CreatePluginInput struct {
	Name        string
	Description string
	Version     string
	Category    string
	Platform    string
	IconKey     string
	FileKey     string
	FileName    string
	FileSize    int64
	Status      string
	SortWeight  int
	ActorID     *int64
}

type UpdatePluginInput struct {
	Name        string
	Description string
	Version     string
	Category    string
	Platform    string
	IconKey     string
	FileKey     string
	FileName    string
	FileSize    int64
	Status      string
	SortWeight  int
	ActorID     *int64
}

func (s *PluginService) Create(ctx context.Context, in *CreatePluginInput) (*Plugin, error) {
	status := in.Status
	if status == "" {
		status = domain.PluginStatusDraft
	}
	platform := in.Platform
	if platform == "" {
		platform = domain.PluginPlatformAll
	}
	p := &Plugin{
		Name:        in.Name,
		Description: in.Description,
		Version:     in.Version,
		Category:    in.Category,
		Platform:    platform,
		IconKey:     in.IconKey,
		FileKey:     in.FileKey,
		FileName:    in.FileName,
		FileSize:    in.FileSize,
		Status:      status,
		SortWeight:  in.SortWeight,
		CreatedBy:   in.ActorID,
		UpdatedBy:   in.ActorID,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PluginService) Update(ctx context.Context, id int64, in *UpdatePluginInput) (*Plugin, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// 若替换了文件/图标，删除旧对象（best-effort）
	if in.FileKey != "" && in.FileKey != p.FileKey {
		s.deleteObjectBestEffort(ctx, p.FileKey)
	}
	if in.IconKey != "" && in.IconKey != p.IconKey {
		s.deleteObjectBestEffort(ctx, p.IconKey)
	}
	p.Name = in.Name
	p.Description = in.Description
	p.Version = in.Version
	p.Category = in.Category
	if in.Platform != "" {
		p.Platform = in.Platform
	}
	if in.IconKey != "" {
		p.IconKey = in.IconKey
	}
	if in.FileKey != "" {
		p.FileKey = in.FileKey
		p.FileName = in.FileName
		p.FileSize = in.FileSize
	}
	if in.Status != "" {
		p.Status = in.Status
	}
	p.SortWeight = in.SortWeight
	p.UpdatedBy = in.ActorID
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PluginService) Delete(ctx context.Context, id int64) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.deleteObjectBestEffort(ctx, p.FileKey)
	s.deleteObjectBestEffort(ctx, p.IconKey)
	return nil
}

func (s *PluginService) deleteObjectBestEffort(ctx context.Context, key string) {
	if key == "" {
		return
	}
	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return
	}
	_ = store.Delete(ctx, key)
}

func (s *PluginService) GetByID(ctx context.Context, id int64) (*Plugin, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PluginService) List(ctx context.Context, params pagination.PaginationParams, filters PluginListFilters) ([]Plugin, *pagination.PaginationResult, error) {
	return s.repo.List(ctx, params, filters)
}

func (s *PluginService) ListPublished(ctx context.Context, category string) ([]Plugin, error) {
	return s.repo.ListPublished(ctx, category)
}

// PresignKey 为某个对象 key 生成短时下载 URL
func (s *PluginService) PresignKey(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", nil
	}
	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return "", err
	}
	return store.PresignURL(ctx, key, pluginURLExpiry)
}

// PrepareDownload 计数 +1 并返回 presigned 下载 URL
func (s *PluginService) PrepareDownload(ctx context.Context, id int64) (string, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if p.Status != domain.PluginStatusPublished || p.FileKey == "" {
		return "", ErrPluginNotFound
	}
	if err := s.repo.IncrementDownloadCount(ctx, id); err != nil {
		return "", err
	}
	return s.PresignKey(ctx, p.FileKey)
}

// OpenDownload increments download count and opens the plugin package for server-side streaming.
func (s *PluginService) OpenDownload(ctx context.Context, id int64) (*PluginDownload, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.PluginStatusPublished || strings.TrimSpace(p.FileKey) == "" {
		return nil, ErrPluginNotFound
	}

	body, contentType, contentLength, err := s.openPluginFile(ctx, p)
	if err != nil {
		return nil, err
	}
	if err := s.repo.IncrementDownloadCount(ctx, id); err != nil {
		_ = body.Close()
		return nil, err
	}

	fileName := strings.TrimSpace(p.FileName)
	if fileName == "" {
		fileName = path.Base(p.FileKey)
	}
	return &PluginDownload{
		Body:          body,
		FileName:      fileName,
		FileSize:      p.FileSize,
		ContentType:   contentType,
		ContentLength: contentLength,
	}, nil
}

func (s *PluginService) openPluginFile(ctx context.Context, p *Plugin) (io.ReadCloser, string, int64, error) {
	fileKey := strings.TrimSpace(p.FileKey)
	if parsed, err := url.Parse(fileKey); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return s.openPluginFileURL(ctx, fileKey)
	}

	store, err := s.storeProvider.Store(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	body, err := store.Download(ctx, fileKey)
	if err != nil {
		return nil, "", 0, fmt.Errorf("download object: %w", err)
	}
	return body, "", p.FileSize, nil
}

func (s *PluginService) openPluginFileURL(ctx context.Context, rawURL string) (io.ReadCloser, string, int64, error) {
	normalized, err := urlvalidator.ValidateHTTPURL(rawURL, true, urlvalidator.ValidationOptions{
		AllowPrivate: false,
	})
	if err != nil {
		return nil, "", 0, err
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return nil, "", 0, err
	}
	if err := urlvalidator.ValidateResolvedIP(parsed.Hostname()); err != nil {
		return nil, "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return nil, "", 0, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("download remote plugin: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_ = resp.Body.Close()
		return nil, "", 0, fmt.Errorf("download remote plugin: status %d", resp.StatusCode)
	}
	return resp.Body, resp.Header.Get("Content-Type"), resp.ContentLength, nil
}
