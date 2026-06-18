package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes 注册无需认证的公开路由。
func RegisterPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	plugins := v1.Group("/plugins")
	{
		plugins.GET("", h.Plugin.List)
		plugins.GET("/:id/download", h.Plugin.Download)
	}
}
