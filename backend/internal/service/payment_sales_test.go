//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestSalesViewsUseCurrentCustomerOwnership(t *testing.T) {
	ctx := context.Background()
	client, _ := newPaymentSalesTestClient(t)

	userRepo := &salesViewUserRepoStub{
		usersByID: map[int64]*User{},
	}
	svc := &PaymentService{
		entClient: client,
		userRepo:  userRepo,
	}

	salesA, err := client.User.Create().
		SetEmail("sales-a@example.com").
		SetPasswordHash("hash").
		SetRole(RoleSales).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	salesB, err := client.User.Create().
		SetEmail("sales-b@example.com").
		SetPasswordHash("hash").
		SetRole(RoleSales).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	userRepo.usersByID[salesA.ID] = authUserEntityToService(salesA)
	userRepo.usersByID[salesB.ID] = authUserEntityToService(salesB)

	customer, err := client.User.Create().
		SetEmail("customer@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
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
		SetRechargeCode("sales-order-completed-code").
		SetPaymentType("alipay").
		SetPaymentTradeNo("sales-order-completed-trade").
		SetOrderType("balance").
		SetOutTradeNo("sales-order-completed").
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(30 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		SetOwnerSalesIDSnapshot(salesA.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUser(customer).
		SetUserEmail(customer.Email).
		SetUserName(customer.Email).
		SetAmount(50).
		SetPayAmount(50).
		SetFeeRate(0).
		SetRechargeCode("sales-order-pending-code").
		SetPaymentType("wxpay").
		SetPaymentTradeNo("sales-order-pending-trade").
		SetOrderType("balance").
		SetOutTradeNo("sales-order-pending").
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(30 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		SetOwnerSalesIDSnapshot(salesA.ID).
		Save(ctx)
	require.NoError(t, err)

	dashboard, err := svc.GetSalesDashboard(ctx, salesB.ID, SalesDashboardParams{})
	require.NoError(t, err)
	require.Equal(t, 1, dashboard.TotalCustomers)
	require.Equal(t, 2, dashboard.TotalOrders)
	require.Equal(t, 1, dashboard.CompletedOrders)
	require.Equal(t, 100.0, dashboard.TotalOrderAmount)

	customerRows, total, err := svc.ListSalesCustomers(ctx, salesB.ID, 1, 20, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, customerRows, 1)
	require.Equal(t, customer.ID, customerRows[0].User.ID)
	require.Equal(t, 2, customerRows[0].TotalOrders)
	require.Equal(t, 100.0, customerRows[0].CompletedOrderAmount)

	orders, orderTotal, err := svc.ListSalesOrders(ctx, salesB.ID, OrderListParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 2, orderTotal)
	require.Len(t, orders, 2)

	customerOrders, customerOrderTotal, err := svc.GetSalesCustomerOrders(ctx, salesB.ID, customer.ID, OrderListParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 2, customerOrderTotal)
	require.Len(t, customerOrders, 2)

	oldSalesOrders, oldSalesTotal, err := svc.ListSalesOrders(ctx, salesA.ID, OrderListParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 0, oldSalesTotal)
	require.Len(t, oldSalesOrders, 0)
}

func newPaymentSalesTestClient(t *testing.T) (*dbent.Client, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:payment_sales?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return client, db
}

type salesViewUserRepoStub struct {
	userRepoStub
	usersByID map[int64]*User
}

func (s *salesViewUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if user, ok := s.usersByID[id]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}
