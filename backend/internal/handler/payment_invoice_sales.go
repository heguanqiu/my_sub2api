package handler

import (
	"strconv"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type invoiceProfileRequest struct {
	Title       string `json:"title"`
	TaxNo       string `json:"tax_no"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	BankName    string `json:"bank_name"`
	BankAccount string `json:"bank_account"`
	InvoiceType string `json:"invoice_type"`
	IsDefault   bool   `json:"is_default"`
}

type createInvoiceRequest struct {
	OrderID   int64 `json:"order_id" binding:"required"`
	ProfileID int64 `json:"profile_id" binding:"required"`
}

type salesCustomerSummaryResponse struct {
	User                 *dto.User `json:"user"`
	TotalOrders          int       `json:"total_orders"`
	CompletedOrderAmount float64   `json:"completed_order_amount"`
}

func sanitizeInvoiceRequest(item *dbent.InvoiceRequest) *dbent.InvoiceRequest {
	if item == nil {
		return nil
	}
	cloned := *item
	return &cloned
}

func sanitizeInvoiceProfile(item *dbent.InvoiceProfile) *dbent.InvoiceProfile {
	if item == nil {
		return nil
	}
	cloned := *item
	return &cloned
}

func sanitizeInvoiceDocument(item *dbent.InvoiceDocument) *dbent.InvoiceDocument {
	if item == nil {
		return nil
	}
	cloned := *item
	return &cloned
}

func (h *PaymentHandler) ListInvoiceProfiles(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	items, err := h.paymentService.ListInvoiceProfiles(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dbent.InvoiceProfile, 0, len(items))
	for _, item := range items {
		out = append(out, sanitizeInvoiceProfile(item))
	}
	response.Success(c, out)
}

func (h *PaymentHandler) CreateInvoiceProfile(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var req invoiceProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.paymentService.CreateInvoiceProfile(c.Request.Context(), subject.UserID, service.InvoiceProfileInput(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeInvoiceProfile(item))
}

func (h *PaymentHandler) UpdateInvoiceProfile(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid profile ID")
		return
	}
	var req invoiceProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.paymentService.UpdateInvoiceProfile(c.Request.Context(), subject.UserID, profileID, service.InvoiceProfileInput(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeInvoiceProfile(item))
}

func (h *PaymentHandler) DeleteInvoiceProfile(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid profile ID")
		return
	}
	if err := h.paymentService.DeleteInvoiceProfile(c.Request.Context(), subject.UserID, profileID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *PaymentHandler) SetDefaultInvoiceProfile(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid profile ID")
		return
	}
	item, err := h.paymentService.SetDefaultInvoiceProfile(c.Request.Context(), subject.UserID, profileID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeInvoiceProfile(item))
}

func (h *PaymentHandler) CreateInvoice(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var req createInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.paymentService.CreateInvoiceRequest(c.Request.Context(), subject.UserID, req.OrderID, req.ProfileID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sanitizeInvoiceRequest(item))
}

func (h *PaymentHandler) ListMyInvoices(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.ListUserInvoiceRequests(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dbent.InvoiceRequest, 0, len(items))
	for _, item := range items {
		out = append(out, sanitizeInvoiceRequest(item))
	}
	response.Paginated(c, out, int64(total), page, pageSize)
}

func (h *PaymentHandler) GetMyInvoice(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	invoiceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	item, err := h.paymentService.GetUserInvoiceRequest(c.Request.Context(), subject.UserID, invoiceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	_, order, profile, documents, err := h.paymentService.GetInvoiceRequestDetail(c.Request.Context(), item.ID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	outDocs := make([]*dbent.InvoiceDocument, 0, len(documents))
	for _, doc := range documents {
		outDocs = append(outDocs, sanitizeInvoiceDocument(doc))
	}
	response.Success(c, gin.H{
		"invoice":   sanitizeInvoiceRequest(item),
		"order":     sanitizePaymentOrderForResponse(order),
		"profile":   sanitizeInvoiceProfile(profile),
		"documents": outDocs,
	})
}

func requireSales(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := requireAuth(c)
	if !ok {
		return middleware2.AuthSubject{}, false
	}
	role, _ := middleware2.GetUserRoleFromContext(c)
	if role != service.RoleSales {
		response.Forbidden(c, "Sales role required")
		return middleware2.AuthSubject{}, false
	}
	return subject, true
}

func (h *PaymentHandler) GetSalesDashboard(c *gin.Context) {
	subject, ok := requireSales(c)
	if !ok {
		return
	}
	stats, err := h.paymentService.GetSalesDashboard(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *PaymentHandler) ListSalesCustomers(c *gin.Context) {
	subject, ok := requireSales(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.ListSalesCustomers(c.Request.Context(), subject.UserID, page, pageSize, c.Query("search"), c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]salesCustomerSummaryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, salesCustomerSummaryResponse{
			User:                 dto.UserFromServiceShallow(item.User),
			TotalOrders:          item.TotalOrders,
			CompletedOrderAmount: item.CompletedOrderAmount,
		})
	}
	response.Paginated(c, out, int64(total), page, pageSize)
}

func (h *PaymentHandler) GetSalesCustomer(c *gin.Context) {
	subject, ok := requireSales(c)
	if !ok {
		return
	}
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid customer ID")
		return
	}
	item, err := h.paymentService.GetSalesCustomer(c.Request.Context(), subject.UserID, customerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserFromServiceShallow(item))
}

func (h *PaymentHandler) GetSalesOrders(c *gin.Context) {
	subject, ok := requireSales(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.ListSalesOrders(c.Request.Context(), subject.UserID, service.OrderListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      c.Query("status"),
		PaymentType: c.Query("payment_type"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizePaymentOrdersForResponse(items), int64(total), page, pageSize)
}

func (h *PaymentHandler) GetSalesCustomerOrders(c *gin.Context) {
	subject, ok := requireSales(c)
	if !ok {
		return
	}
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid customer ID")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.GetSalesCustomerOrders(c.Request.Context(), subject.UserID, customerID, service.OrderListParams{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizePaymentOrdersForResponse(items), int64(total), page, pageSize)
}

func (h *PaymentHandler) GetSalesCustomerInvoices(c *gin.Context) {
	subject, ok := requireSales(c)
	if !ok {
		return
	}
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid customer ID")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.paymentService.GetSalesCustomerInvoices(c.Request.Context(), subject.UserID, customerID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dbent.InvoiceRequest, 0, len(items))
	for _, item := range items {
		out = append(out, sanitizeInvoiceRequest(item))
	}
	response.Paginated(c, out, int64(total), page, pageSize)
}
