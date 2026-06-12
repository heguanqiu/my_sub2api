package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeUserAPINoticeConsumer struct {
	notice *service.UserAPINotice
	err    error
	calls  int
	userID int64
}

func (f *fakeUserAPINoticeConsumer) ConsumeNextPending(_ context.Context, userID int64, _ string, _ time.Time) (*service.UserAPINotice, error) {
	f.calls++
	f.userID = userID
	return f.notice, f.err
}

func TestUserAPINoticeGateConsumesAndAborts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	consumer := &fakeUserAPINoticeConsumer{
		notice: &service.UserAPINotice{ID: 1, UserID: 42, Message: "请先联系管理员"},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 42})
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, "req-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(UserAPINoticeGate(consumer, func(c *gin.Context, status int, message string) {
		require.True(t, service.HasOpsClientBusinessLimited(c))
		reason, ok := c.Get(service.OpsClientBusinessLimitedReasonKey)
		require.True(t, ok)
		require.Equal(t, service.OpsClientBusinessLimitedReasonAdminNotice, reason)
		AdminNoticeAnthropicErrorWriter(c, status, message)
	}))
	router.POST("/v1/messages", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "请先联系管理员")
	require.Contains(t, w.Body.String(), "admin_notice")
	require.Equal(t, 1, consumer.calls)
	require.Equal(t, int64(42), consumer.userID)
}

func TestUserAPINoticeGatePassesThroughWithoutNotice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	consumer := &fakeUserAPINoticeConsumer{}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 42})
		c.Next()
	})
	router.Use(UserAPINoticeGate(consumer, AdminNoticeAnthropicErrorWriter))
	router.POST("/v1/messages", func(c *gin.Context) {
		require.False(t, service.HasOpsClientBusinessLimited(c))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, consumer.calls)
}
