package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminRiskControlRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterAdminRoutes(
		v1,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{},
		},
		servermiddleware.AdminAuthMiddleware(func(c *gin.Context) {
			c.AbortWithStatus(http.StatusTeapot)
		}),
	)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/admin/risk-control/config"},
		{method: http.MethodPut, path: "/api/v1/admin/risk-control/config", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/admin/risk-control/status"},
		{method: http.MethodPost, path: "/api/v1/admin/risk-control/api-keys/test", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/admin/risk-control/logs"},
		{method: http.MethodPost, path: "/api/v1/admin/risk-control/users/1/unban"},
		{method: http.MethodDelete, path: "/api/v1/admin/risk-control/hashes", body: `{}`},
		{method: http.MethodDelete, path: "/api/v1/admin/risk-control/hashes/all"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusTeapot, w.Code, "path=%s should hit admin auth middleware", tt.path)
	}
}
