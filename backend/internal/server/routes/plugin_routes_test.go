package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPublicPluginRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterPublicRoutes(v1, &handler.Handlers{
		Plugin: handler.NewPluginHandler(nil),
	})

	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/plugins")
	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/plugins/:id/download")
}

func TestAdminPluginRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")

	registerPluginRoutes(admin, &handler.Handlers{
		Admin: &handler.AdminHandlers{
			Plugin: adminhandler.NewPluginHandler(nil),
		},
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/admin/plugins"},
		{method: http.MethodPost, path: "/api/v1/admin/plugins"},
		{method: http.MethodPost, path: "/api/v1/admin/plugins/upload"},
		{method: http.MethodGet, path: "/api/v1/admin/plugins/:id"},
		{method: http.MethodPut, path: "/api/v1/admin/plugins/:id"},
		{method: http.MethodDelete, path: "/api/v1/admin/plugins/:id"},
	} {
		requireRouteRegistered(t, router, tc.method, tc.path)
	}
}

func requireRouteRegistered(t *testing.T, router *gin.Engine, method, path string) {
	t.Helper()
	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			return
		}
	}
	require.Failf(t, "route not registered", "%s %s", method, path)
}
