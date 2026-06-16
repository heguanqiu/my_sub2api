package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newUserRoutesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		&handler.Handlers{
			User:             handler.NewUserHandler(nil, nil, nil, nil, nil, nil),
			APIKey:           handler.NewAPIKeyHandler(nil),
			Playground:       handler.NewPlaygroundHandler(nil, nil, nil),
			Usage:            handler.NewUsageHandler(nil, nil, nil, nil),
			Redeem:           handler.NewRedeemHandler(nil),
			Subscription:     handler.NewSubscriptionHandler(nil),
			Announcement:     handler.NewAnnouncementHandler(nil),
			ChannelMonitor:   handler.NewChannelMonitorUserHandler(nil, nil),
			Totp:             handler.NewTotpHandler(nil),
			AvailableChannel: handler.NewAvailableChannelHandler(nil, nil, nil),
		},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})
			c.Set(string(servermiddleware.ContextKeyUserRole), "sales")
			c.Next()
		}),
		nil,
	)

	return router
}

func TestUserRoutesReferralPathsAreRegistered(t *testing.T) {
	router := newUserRoutesTestRouter()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/referral/my-link"},
		{method: http.MethodPost, path: "/api/v1/referral/my-link/regenerate"},
		{method: http.MethodPost, path: "/api/v1/referral/my-link/disable"},
		{method: http.MethodPost, path: "/api/v1/referral/my-link/revoke"},
		{method: http.MethodGet, path: "/api/v1/referral/my-invitees"},
		{method: http.MethodGet, path: "/api/v1/referral/my-rewards"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit referral handler", tc.path)
	}
}
