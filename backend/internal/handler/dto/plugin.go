package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AdminPlugin 管理面 DTO
type AdminPlugin struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Version       string    `json:"version"`
	Category      string    `json:"category"`
	Platform      string    `json:"platform"`
	IconKey       string    `json:"icon_key"`
	FileKey       string    `json:"file_key"`
	FileName      string    `json:"file_name"`
	FileSize      int64     `json:"file_size"`
	DownloadCount int64     `json:"download_count"`
	Status        string    `json:"status"`
	SortWeight    int       `json:"sort_weight"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func AdminPluginFromService(p *service.Plugin) *AdminPlugin {
	if p == nil {
		return nil
	}
	return &AdminPlugin{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		Version:       p.Version,
		Category:      p.Category,
		Platform:      p.Platform,
		IconKey:       p.IconKey,
		FileKey:       p.FileKey,
		FileName:      p.FileName,
		FileSize:      p.FileSize,
		DownloadCount: p.DownloadCount,
		Status:        p.Status,
		SortWeight:    p.SortWeight,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// PublicPlugin 公开面 DTO
type PublicPlugin struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	Category      string `json:"category"`
	Platform      string `json:"platform"`
	IconURL       string `json:"icon_url"`
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	DownloadCount int64  `json:"download_count"`
	DownloadURL   string `json:"download_url"`
}
