package admin

import (
	"strconv"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func sanitizeAdminInvoiceRequest(item *dbent.InvoiceRequest) *dbent.InvoiceRequest {
	if item == nil {
		return nil
	}
	cloned := *item
	return &cloned
}

func sanitizeAdminInvoiceProfile(item *dbent.InvoiceProfile) *dbent.InvoiceProfile {
	if item == nil {
		return nil
	}
	cloned := *item
	return &cloned
}

func sanitizeAdminInvoiceDocument(item *dbent.InvoiceDocument) *dbent.InvoiceDocument {
	if item == nil {
		return nil
	}
	cloned := *item
	return &cloned
}

func (h *PaymentHandler) ListInvoices(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.ListAdminInvoiceRequests(c.Request.Context(), page, pageSize, c.Query("status"), c.Query("keyword"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dbent.InvoiceRequest, 0, len(items))
	for _, item := range items {
		out = append(out, sanitizeAdminInvoiceRequest(item))
	}
	response.Paginated(c, out, int64(total), page, pageSize)
}

func (h *PaymentHandler) GetInvoiceDetail(c *gin.Context) {
	invoiceID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, order, profile, documents, err := h.paymentService.GetInvoiceRequestDetail(c.Request.Context(), invoiceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	outDocs := make([]*dbent.InvoiceDocument, 0, len(documents))
	for _, doc := range documents {
		outDocs = append(outDocs, sanitizeAdminInvoiceDocument(doc))
	}
	response.Success(c, gin.H{
		"invoice":   sanitizeAdminInvoiceRequest(item),
		"order":     sanitizeAdminPaymentOrderForResponse(order),
		"profile":   sanitizeAdminInvoiceProfile(profile),
		"documents": outDocs,
	})
}

func (h *PaymentHandler) RetryInvoice(c *gin.Context) {
	invoiceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	item, err := h.paymentService.RetryInvoiceRequest(c.Request.Context(), invoiceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeAdminInvoiceRequest(item))
}

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
