package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestAdminSalesViewsUseTargetSalesAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	client, rawDB := newAdminPaymentSalesHandlerTestClient(t)
	userRepo := repository.NewUserRepository(client, rawDB)
	paymentSvc := service.NewPaymentService(client, nil, nil, nil, nil, nil, userRepo, nil)
	handler := NewPaymentHandler(paymentSvc, nil)

	salesA, err := client.User.Create().
		SetEmail("sales-a@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	salesB, err := client.User.Create().
		SetEmail("sales-b@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	customer, err := client.User.Create().
		SetEmail("customer@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetOwnerSalesID(salesB.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUser(customer).
		SetUserEmail(customer.Email).
		SetUserName(customer.Email).
		SetAmount(100).
		SetPayAmount(100).
		SetFeeRate(0).
		SetRechargeCode("admin-sales-order-code").
		SetPaymentType("alipay").
		SetPaymentTradeNo("admin-sales-order-trade").
		SetOrderType("balance").
		SetOutTradeNo("admin-sales-order").
		SetStatus(service.OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(30 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		SetOwnerSalesIDSnapshot(salesA.ID).
		Save(ctx)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/v1/admin/payment/sales/:sales_id/dashboard", handler.GetSalesDashboard)
	router.GET("/api/v1/admin/payment/sales/:sales_id/customers", handler.ListSalesCustomers)
	router.GET("/api/v1/admin/payment/sales/:sales_id/orders", handler.GetSalesOrders)

	recDashboard := httptest.NewRecorder()
	reqDashboard := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/payment/sales/%d/dashboard?month=%s", salesB.ID, time.Now().Format("2006-01")), nil)
	router.ServeHTTP(recDashboard, reqDashboard)
	require.Equal(t, http.StatusOK, recDashboard.Code)

	var dashboardResp struct {
		Data struct {
			TotalCustomers   int     `json:"total_customers"`
			TotalOrders      int     `json:"total_orders"`
			CompletedOrders  int     `json:"completed_orders"`
			TotalOrderAmount float64 `json:"total_order_amount"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recDashboard.Body.Bytes(), &dashboardResp))
	require.Equal(t, 1, dashboardResp.Data.TotalCustomers)
	require.Equal(t, 1, dashboardResp.Data.TotalOrders)
	require.Equal(t, 1, dashboardResp.Data.CompletedOrders)
	require.Equal(t, 100.0, dashboardResp.Data.TotalOrderAmount)

	recCustomers := httptest.NewRecorder()
	reqCustomers := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/payment/sales/%d/customers?page=1&page_size=20", salesB.ID), nil)
	router.ServeHTTP(recCustomers, reqCustomers)
	require.Equal(t, http.StatusOK, recCustomers.Code)

	var customersResp struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recCustomers.Body.Bytes(), &customersResp))
	require.Equal(t, 1, customersResp.Data.Total)
	require.Len(t, customersResp.Data.Items, 1)
	userPayload, ok := customersResp.Data.Items[0]["user"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, customer.Email, userPayload["email"])
	require.NotContains(t, userPayload, "Email")

	recOldSalesOrders := httptest.NewRecorder()
	reqOldSalesOrders := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/payment/sales/%d/orders?page=1&page_size=20", salesA.ID), nil)
	router.ServeHTTP(recOldSalesOrders, reqOldSalesOrders)
	require.Equal(t, http.StatusOK, recOldSalesOrders.Code)

	var oldSalesOrdersResp struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recOldSalesOrders.Body.Bytes(), &oldSalesOrdersResp))
	require.Equal(t, 0, oldSalesOrdersResp.Data.Total)
	require.Len(t, oldSalesOrdersResp.Data.Items, 0)
}

func TestAdminSalesViewsRejectNonSalesAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	client, rawDB := newAdminPaymentSalesHandlerTestClient(t)
	userRepo := repository.NewUserRepository(client, rawDB)
	paymentSvc := service.NewPaymentService(client, nil, nil, nil, nil, nil, userRepo, nil)
	handler := NewPaymentHandler(paymentSvc, nil)

	user, err := client.User.Create().
		SetEmail("user@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/v1/admin/payment/sales/:sales_id/dashboard", handler.GetSalesDashboard)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/payment/sales/%d/dashboard", user.ID), nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func newAdminPaymentSalesHandlerTestClient(t *testing.T) (*dbent.Client, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:admin_payment_sales_handler_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return client, db
}
