package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PluginHandler handles public plugin center endpoints
type PluginHandler struct {
	pluginService *service.PluginService
}

// NewPluginHandler creates a new public plugin handler
func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{pluginService: pluginService}
}

// List GET /api/v1/plugins
func (h *PluginHandler) List(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))
	items, err := h.pluginService.ListPublished(c.Request.Context(), category)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.PublicPlugin, 0, len(items))
	for i := range items {
		p := items[i]
		iconURL := ""
		if p.IconKey != "" {
			// presign 失败不阻断列表，仅图标留空
			if u, perr := h.pluginService.PresignKey(c.Request.Context(), p.IconKey); perr == nil {
				iconURL = u
			}
		}
		out = append(out, dto.PublicPlugin{
			ID:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			Version:       p.Version,
			Category:      p.Category,
			Platform:      p.Platform,
			IconURL:       iconURL,
			FileName:      p.FileName,
			FileSize:      p.FileSize,
			DownloadCount: p.DownloadCount,
			DownloadURL:   "/api/v1/plugins/" + strconv.FormatInt(p.ID, 10) + "/download",
		})
	}
	response.Success(c, out)
}

// Download GET /api/v1/plugins/:id/download
func (h *PluginHandler) Download(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	download, err := h.pluginService.OpenDownload(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer download.Body.Close()

	contentType := strings.TrimSpace(download.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	if download.FileName != "" {
		c.Header("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(download.FileName, `"`, "")+`"`)
	}
	if download.ContentLength > 0 {
		c.Header("Content-Length", strconv.FormatInt(download.ContentLength, 10))
	} else if download.FileSize > 0 {
		c.Header("Content-Length", strconv.FormatInt(download.FileSize, 10))
	}
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, download.Body); err != nil {
		_ = c.Error(err)
	}
}
