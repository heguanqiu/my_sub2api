package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type inviteLinkResponse struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	CreatorRole  string `json:"creator_role"`
	OwnerSalesID *int64 `json:"owner_sales_id,omitempty"`
}

type inviteRewardResponse struct {
	ID             int64   `json:"id"`
	InviterUserID  int64   `json:"inviter_user_id"`
	InviteeUserID  int64   `json:"invitee_user_id"`
	TriggerOrderID int64   `json:"trigger_order_id"`
	RewardType     string  `json:"reward_type"`
	RewardAmount   float64 `json:"reward_amount"`
	Status         string  `json:"status"`
	Reason         string  `json:"reason,omitempty"`
	CreatedAt      string  `json:"created_at"`
	ConfirmedAt    *string `json:"confirmed_at,omitempty"`
	ReversedAt     *string `json:"reversed_at,omitempty"`
}

func (h *UserHandler) inviteLinkURL(c *gin.Context, code string) string {
	base := ""
	if h.authService != nil {
		base = h.authService.GetFrontendURL(c.Request.Context())
	}
	if base == "" {
		base = "http://" + c.Request.Host
	}
	return base + "/register?invitation_code=" + code
}

func inviteLinkToResponse(c *gin.Context, h *UserHandler, link *service.InviteLink) *inviteLinkResponse {
	if link == nil {
		return nil
	}
	return &inviteLinkResponse{
		ID:           link.ID,
		Code:         link.Code,
		URL:          h.inviteLinkURL(c, link.Code),
		Status:       link.Status,
		CreatorRole:  link.CreatorRole,
		OwnerSalesID: link.OwnerSalesID,
	}
}

func rewardToResponse(item service.InviteRewardLedger) inviteRewardResponse {
	resp := inviteRewardResponse{
		ID:             item.ID,
		InviterUserID:  item.InviterUserID,
		InviteeUserID:  item.InviteeUserID,
		TriggerOrderID: item.TriggerOrderID,
		RewardType:     item.RewardType,
		RewardAmount:   item.RewardAmount,
		Status:         item.Status,
		Reason:         item.Reason,
		CreatedAt:      item.CreatedAt.Format(timeLayoutRFC3339),
	}
	if item.ConfirmedAt != nil {
		value := item.ConfirmedAt.Format(timeLayoutRFC3339)
		resp.ConfirmedAt = &value
	}
	if item.ReversedAt != nil {
		value := item.ReversedAt.Format(timeLayoutRFC3339)
		resp.ReversedAt = &value
	}
	return resp
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"

func (h *UserHandler) GetMyInviteLink(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.authService == nil {
		response.InternalError(c, "Auth service not configured")
		return
	}
	link, err := h.authService.GetMyInviteLink(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inviteLinkToResponse(c, h, link))
}

func (h *UserHandler) RegenerateMyInviteLink(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.authService == nil {
		response.InternalError(c, "Auth service not configured")
		return
	}
	link, err := h.authService.RegenerateMyInviteLink(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inviteLinkToResponse(c, h, link))
}

func (h *UserHandler) DisableMyInviteLink(c *gin.Context) {
	h.updateInviteLinkStatus(c, service.InviteLinkStatusDisabled)
}

func (h *UserHandler) RevokeMyInviteLink(c *gin.Context) {
	h.updateInviteLinkStatus(c, service.InviteLinkStatusRevoked)
}

func (h *UserHandler) updateInviteLinkStatus(c *gin.Context, status string) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.authService == nil {
		response.InternalError(c, "Auth service not configured")
		return
	}
	link, err := h.authService.UpdateMyInviteLinkStatus(c.Request.Context(), subject.UserID, status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inviteLinkToResponse(c, h, link))
}

func (h *UserHandler) GetMyInvitees(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	if h.authService == nil {
		response.InternalError(c, "Auth service not configured")
		return
	}
	items, result, err := h.authService.ListMyInvitees(c.Request.Context(), subject.UserID, page, pageSize, c.Query("search"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dto.User, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, dto.UserFromServiceShallow(&item))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *UserHandler) GetMyRewards(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	if h.authService == nil {
		response.InternalError(c, "Auth service not configured")
		return
	}
	items, result, err := h.authService.ListMyInviteRewards(c.Request.Context(), subject.UserID, page, pageSize, c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]inviteRewardResponse, 0, len(items))
	for _, item := range items {
		out = append(out, rewardToResponse(item))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}
