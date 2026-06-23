package admin

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	maxPluginFileSize = 512 << 20 // 512MB
	maxPluginIconSize = 2 << 20   // 2MB
)

var (
	allowedPluginExts = map[string]bool{".zip": true, ".vsix": true, ".gz": true, ".tgz": true}
	allowedIconExts   = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".webp": true}
)

func validatePluginUploadFile(kind, filename string, size int64) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if kind == "package" {
		if size > maxPluginFileSize {
			return "插件包不能超过 512MB"
		}
		if !allowedPluginExts[ext] {
			return "不支持的插件包格式"
		}
		return ""
	}
	if size > maxPluginIconSize {
		return "图标不能超过 2MB"
	}
	if !allowedIconExts[ext] {
		return "不支持的图标格式"
	}
	return ""
}

// PluginHandler handles admin plugin management
type PluginHandler struct {
	pluginService *service.PluginService
}

// NewPluginHandler creates a new admin plugin handler
func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{pluginService: pluginService}
}

type createPluginRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Platform    string `json:"platform" binding:"omitempty,oneof=all windows macos linux"`
	IconKey     string `json:"icon_key"`
	FileKey     string `json:"file_key"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	Status      string `json:"status" binding:"omitempty,oneof=draft published"`
	SortWeight  int    `json:"sort_weight"`
}

type updatePluginRequest = createPluginRequest

func pluginActorID(c *gin.Context) *int64 {
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		id := subject.UserID
		return &id
	}
	return nil
}

// List GET /api/v1/admin/plugins
func (h *PluginHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	items, result, err := h.pluginService.List(c.Request.Context(), params, service.PluginListFilters{
		Status:   strings.TrimSpace(c.Query("status")),
		Category: strings.TrimSpace(c.Query("category")),
		Search:   search,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminPlugin, 0, len(items))
	for i := range items {
		out = append(out, *dto.AdminPluginFromService(&items[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

// GetByID GET /api/v1/admin/plugins/:id
func (h *PluginHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	p, err := h.pluginService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminPluginFromService(p))
}

// Create POST /api/v1/admin/plugins
func (h *PluginHandler) Create(c *gin.Context) {
	var req createPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.pluginService.Create(c.Request.Context(), &service.CreatePluginInput{
		Name: req.Name, Description: req.Description, Version: req.Version,
		Category: req.Category, Platform: req.Platform, IconKey: req.IconKey,
		FileKey: req.FileKey, FileName: req.FileName, FileSize: req.FileSize,
		Status: req.Status, SortWeight: req.SortWeight, ActorID: pluginActorID(c),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.AdminPluginFromService(p))
}

// Update PUT /api/v1/admin/plugins/:id
func (h *PluginHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	var req updatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.pluginService.Update(c.Request.Context(), id, &service.UpdatePluginInput{
		Name: req.Name, Description: req.Description, Version: req.Version,
		Category: req.Category, Platform: req.Platform, IconKey: req.IconKey,
		FileKey: req.FileKey, FileName: req.FileName, FileSize: req.FileSize,
		Status: req.Status, SortWeight: req.SortWeight, ActorID: pluginActorID(c),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminPluginFromService(p))
}

// Delete DELETE /api/v1/admin/plugins/:id
func (h *PluginHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid plugin ID")
		return
	}
	if err := h.pluginService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// Upload POST /api/v1/admin/plugins/upload  (multipart: kind, file)
func (h *PluginHandler) Upload(c *gin.Context) {
	kind := c.PostForm("kind")
	if kind != "package" && kind != "icon" {
		response.BadRequest(c, "kind must be package or icon")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	if msg := validatePluginUploadFile(kind, fileHeader.Filename, fileHeader.Size); msg != "" {
		response.BadRequest(c, msg)
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c, "open uploaded file failed")
		return
	}
	defer f.Close()

	res, err := h.pluginService.UploadObject(
		c.Request.Context(), kind, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), f)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"key": res.Key, "file_name": res.FileName, "size": res.Size})
}
