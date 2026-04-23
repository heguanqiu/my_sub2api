package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestSalesCustomerResponsesUseFrontendJSONShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client, rawDB := newPaymentInvoiceSalesHandlerTestClient(t)
	userRepo := repository.NewUserRepository(client, rawDB)
	paymentSvc := service.NewPaymentService(client, nil, nil, nil, nil, nil, userRepo, nil)
	handler := NewPaymentHandler(paymentSvc, nil, nil)

	sales, err := client.User.Create().
		SetEmail("sales@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(context.Background())
	require.NoError(t, err)

	customer, err := client.User.Create().
		SetEmail("customer@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetOwnerSalesID(sales.ID).
		Save(context.Background())
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: sales.ID})
		c.Set(string(middleware2.ContextKeyUserRole), service.RoleSales)
		c.Next()
	})
	router.GET("/api/v1/sales/customers", handler.ListSalesCustomers)
	router.GET("/api/v1/sales/customers/:id", handler.GetSalesCustomer)

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/sales/customers?page=1&page_size=20", nil)
	router.ServeHTTP(recList, reqList)
	require.Equal(t, http.StatusOK, recList.Code)

	var listResp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recList.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data.Items, 1)
	userPayload, ok := listResp.Data.Items[0]["user"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, userPayload, "id")
	require.Contains(t, userPayload, "email")
	require.NotContains(t, userPayload, "ID")
	require.NotContains(t, userPayload, "Email")

	recDetail := httptest.NewRecorder()
	reqDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sales/customers/%d", customer.ID), nil)
	router.ServeHTTP(recDetail, reqDetail)
	require.Equal(t, http.StatusOK, recDetail.Code)

	var detailResp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recDetail.Body.Bytes(), &detailResp))
	require.Equal(t, float64(customer.ID), detailResp.Data["id"])
	require.Equal(t, customer.Email, detailResp.Data["email"])
	require.NotContains(t, detailResp.Data, "ID")
	require.NotContains(t, detailResp.Data, "Email")
}

func newPaymentInvoiceSalesHandlerTestClient(t *testing.T) (*dbent.Client, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:payment_invoice_sales_handler?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client, db
}
