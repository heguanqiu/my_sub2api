package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UserAPINoticeConsumer interface {
	ConsumeNextPending(ctx context.Context, userID int64, requestID string, now time.Time) (*service.UserAPINotice, error)
}

func UserAPINoticeGate(noticeConsumer UserAPINoticeConsumer, writeError GatewayErrorWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if noticeConsumer == nil {
			c.Next()
			return
		}
		subject, ok := GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			c.Next()
			return
		}

		notice, err := noticeConsumer.ConsumeNextPending(c.Request.Context(), subject.UserID, noticeRequestID(c), time.Now())
		if err != nil {
			slog.Warn("user api notice check failed", "user_id", subject.UserID, "error", err)
			c.Next()
			return
		}
		if notice == nil {
			c.Next()
			return
		}

		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAdminNotice)
		writeError(c, http.StatusForbidden, notice.Message)
		c.Abort()
	}
}

func AdminNoticeAnthropicErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    "user_notice",
			"code":    "admin_notice",
			"message": message,
		},
	})
}

func AdminNoticeGoogleErrorWriter(c *gin.Context, status int, message string) {
	GoogleErrorWriter(c, status, message)
}

func noticeRequestID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if v, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if v := strings.TrimSpace(c.GetHeader("X-Client-Request-ID")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-ID")); v != "" {
		return v
	}
	return ""
}
