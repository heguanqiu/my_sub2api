package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) ListInviteRewards(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.ListInviteRewardLedger(c.Request.Context(), page, pageSize, c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, int64(total), page, pageSize)
}

func (h *PaymentHandler) GetInviteReward(c *gin.Context) {
	rewardID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid reward ID")
		return
	}
	item, err := h.paymentService.GetInviteRewardLedgerByID(c.Request.Context(), rewardID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
