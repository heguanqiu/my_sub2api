package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type salesCustomerSummaryResponse struct {
	User                 *dto.User `json:"user"`
	TotalOrders          int       `json:"total_orders"`
	CompletedOrderAmount float64   `json:"completed_order_amount"`
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
	stats, err := h.paymentService.GetSalesDashboard(c.Request.Context(), subject.UserID, service.SalesDashboardParams{
		Month:    salesDashboardMonthQuery(c),
		Timezone: c.Query("timezone"),
	})
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
	startTime, endTime, err := service.ParseOrderListDateRange(c.Query("start_date"), c.Query("end_date"), c.Query("timezone"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items, total, err := h.paymentService.ListSalesOrders(c.Request.Context(), subject.UserID, service.OrderListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      c.Query("status"),
		PaymentType: c.Query("payment_type"),
		StartTime:   startTime,
		EndTime:     endTime,
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
	startTime, endTime, err := service.ParseOrderListDateRange(c.Query("start_date"), c.Query("end_date"), c.Query("timezone"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items, total, err := h.paymentService.GetSalesCustomerOrders(c.Request.Context(), subject.UserID, customerID, service.OrderListParams{
		Page:      page,
		PageSize:  pageSize,
		Status:    c.Query("status"),
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizePaymentOrdersForResponse(items), int64(total), page, pageSize)
}

func salesDashboardMonthQuery(c *gin.Context) string {
	if month := c.Query("month"); month != "" {
		return month
	}
	legacyRange := strings.TrimSpace(c.Query("range"))
	if len(legacyRange) == len("2006-01") {
		return legacyRange
	}
	return ""
}
