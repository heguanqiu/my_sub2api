package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type changeInviterRequest struct {
	NewInvitedByUserID *int64 `json:"new_invited_by_user_id"`
}

type migrateSalesOwnerRequest struct {
	TargetSalesUserID int64 `json:"target_sales_user_id" binding:"required"`
}

func (h *UserHandler) GetReferralTree(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	tree, err := h.adminService.GetReferralTree(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tree)
}

func (h *UserHandler) ChangeInviter(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	var req changeInviterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.adminService.ChangeInviter(c.Request.Context(), userID, req.NewInvitedByUserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UserHandler) RecomputeSalesOwner(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	result, err := h.adminService.RecomputeSalesOwner(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UserHandler) PreviewSalesOwnerMigration(c *gin.Context) {
	h.handleSalesOwnerMigration(c, true)
}

func (h *UserHandler) MigrateSalesOwner(c *gin.Context) {
	h.handleSalesOwnerMigration(c, false)
}

func (h *UserHandler) handleSalesOwnerMigration(c *gin.Context, preview bool) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	var req migrateSalesOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var result *service.ReferralMutationResult
	if preview {
		result, err = h.adminService.PreviewSalesOwnerMigration(c.Request.Context(), userID, req.TargetSalesUserID)
	} else {
		result, err = h.adminService.MigrateSalesOwner(c.Request.Context(), userID, req.TargetSalesUserID)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
