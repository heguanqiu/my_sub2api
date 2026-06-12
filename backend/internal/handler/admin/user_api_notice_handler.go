package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UserAPINoticeHandler struct {
	noticeService *service.UserAPINoticeService
}

func NewUserAPINoticeHandler(noticeService *service.UserAPINoticeService) *UserAPINoticeHandler {
	return &UserAPINoticeHandler{noticeService: noticeService}
}

type CreateUserAPINoticeRequest struct {
	Message   string `json:"message" binding:"required"`
	ExpiresAt *int64 `json:"expires_at"`
}

func (h *UserAPINoticeHandler) Create(c *gin.Context) {
	userID, err := parsePositiveInt64Param(c, "id")
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req CreateUserAPINoticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt > 0 {
		t := time.Unix(*req.ExpiresAt, 0)
		expiresAt = &t
	}

	notice, err := h.noticeService.Create(c.Request.Context(), service.CreateUserAPINoticeInput{
		UserID:          userID,
		Message:         req.Message,
		CreatedByUserID: subject.UserID,
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, notice)
}

func (h *UserAPINoticeHandler) ListByUser(c *gin.Context) {
	userID, err := parsePositiveInt64Param(c, "id")
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	page, pageSize := response.ParsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	notices, result, err := h.noticeService.List(c.Request.Context(), service.UserAPINoticeFilter{
		UserID: userID,
		Status: status,
	}, pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "created_at",
		SortOrder: pagination.SortOrderDesc,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, notices, result.Total, result.Page, result.PageSize)
}

func (h *UserAPINoticeHandler) Cancel(c *gin.Context) {
	noticeID, err := parsePositiveInt64Param(c, "id")
	if err != nil {
		response.BadRequest(c, "Invalid notice ID")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	notice, err := h.noticeService.Cancel(c.Request.Context(), noticeID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, notice)
}

func parsePositiveInt64Param(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
