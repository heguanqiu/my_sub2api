package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestSalesDashboardMonthFiltersOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentSalesRangeTestClient(t)

	userRepo := &salesDashboardRangeUserRepoStub{
		usersByID: map[int64]*User{},
	}
	svc := &PaymentService{
		entClient: client,
		userRepo:  userRepo,
	}

	sales, err := client.User.Create().
		SetEmail("range-sales@example.com").
		SetPasswordHash("hash").
		SetRole(RoleSales).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)
	userRepo.usersByID[sales.ID] = authUserEntityToService(sales)

	customer, err := client.User.Create().
		SetEmail("range-customer@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetOwnerSalesID(sales.ID).
		SetCreatedAt(time.Date(2026, 6, 2, 10, 0, 0, 0, time.Local)).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetEmail("range-july-customer@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetOwnerSalesID(sales.ID).
		SetCreatedAt(time.Date(2026, 7, 2, 10, 0, 0, 0, time.Local)).
		Save(ctx)
	require.NoError(t, err)

	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.Local)
	createSalesRangeOrder := func(outTradeNo string, amount float64, createdAt time.Time) {
		t.Helper()
		_, err := client.PaymentOrder.Create().
			SetUser(customer).
			SetUserEmail(customer.Email).
			SetUserName(customer.Email).
			SetAmount(amount).
			SetPayAmount(amount).
			SetFeeRate(0).
			SetRechargeCode(outTradeNo + "-code").
			SetPaymentType("alipay").
			SetPaymentTradeNo(outTradeNo + "-trade").
			SetOrderType("balance").
			SetOutTradeNo(outTradeNo).
			SetStatus(OrderStatusCompleted).
			SetExpiresAt(createdAt.Add(30 * time.Minute)).
			SetClientIP("127.0.0.1").
			SetSrcHost("app.example.com").
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	createSalesRangeOrder("range-june-a", 10, now)
	createSalesRangeOrder("range-june-b", 20, time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local))
	createSalesRangeOrder("range-july", 30, time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local))
	createSalesRangeOrder("range-may", 40, time.Date(2026, 5, 31, 23, 59, 59, 0, time.Local))

	month, err := svc.GetSalesDashboard(ctx, sales.ID, SalesDashboardParams{Month: "2026-06"})
	require.NoError(t, err)
	require.Equal(t, "2026-06", month.Month)
	require.Equal(t, "2026-06-01", month.StartDate)
	require.Equal(t, "2026-06-30", month.EndDate)
	require.Equal(t, 1, month.TotalCustomers)
	require.Equal(t, 2, month.TotalOrders)
	require.Equal(t, 30.0, month.TotalOrderAmount)
}

func TestSalesOrdersDateRangeFiltersOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentSalesRangeTestClient(t)

	userRepo := &salesDashboardRangeUserRepoStub{
		usersByID: map[int64]*User{},
	}
	svc := &PaymentService{
		entClient: client,
		userRepo:  userRepo,
	}

	sales, err := client.User.Create().
		SetEmail("orders-range-sales@example.com").
		SetPasswordHash("hash").
		SetRole(RoleSales).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)
	userRepo.usersByID[sales.ID] = authUserEntityToService(sales)

	customer, err := client.User.Create().
		SetEmail("orders-range-customer@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetOwnerSalesID(sales.ID).
		Save(ctx)
	require.NoError(t, err)

	createOrder := func(outTradeNo string, createdAt time.Time) {
		t.Helper()
		_, err := client.PaymentOrder.Create().
			SetUser(customer).
			SetUserEmail(customer.Email).
			SetUserName(customer.Email).
			SetAmount(10).
			SetPayAmount(10).
			SetFeeRate(0).
			SetRechargeCode(outTradeNo + "-code").
			SetPaymentType("alipay").
			SetPaymentTradeNo(outTradeNo + "-trade").
			SetOrderType("balance").
			SetOutTradeNo(outTradeNo).
			SetStatus(OrderStatusCompleted).
			SetExpiresAt(createdAt.Add(30 * time.Minute)).
			SetClientIP("127.0.0.1").
			SetSrcHost("app.example.com").
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	createOrder("orders-may", time.Date(2026, 5, 31, 23, 59, 59, 0, time.Local))
	createOrder("orders-june", time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local))
	createOrder("orders-july", time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local))

	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	orders, total, err := svc.ListSalesOrders(ctx, sales.ID, OrderListParams{Page: 1, PageSize: 20, StartTime: &start, EndTime: &end})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, orders, 1)
	require.Equal(t, "orders-june", orders[0].OutTradeNo)
}

func newPaymentSalesRangeTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:payment_sales_range?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type salesDashboardRangeUserRepoStub struct {
	usersByID map[int64]*User
}

func (s *salesDashboardRangeUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if user, ok := s.usersByID[id]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (s *salesDashboardRangeUserRepoStub) GetByIDIncludeDeleted(ctx context.Context, id int64) (*User, error) {
	return s.GetByID(ctx, id)
}

func (s *salesDashboardRangeUserRepoStub) Create(context.Context, *User) error {
	panic("unexpected Create call")
}

func (s *salesDashboardRangeUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}

func (s *salesDashboardRangeUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}

func (s *salesDashboardRangeUserRepoStub) Update(context.Context, *User) error {
	panic("unexpected Update call")
}

func (s *salesDashboardRangeUserRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *salesDashboardRangeUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	panic("unexpected GetUserAvatar call")
}

func (s *salesDashboardRangeUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}

func (s *salesDashboardRangeUserRepoStub) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected DeleteUserAvatar call")
}

func (s *salesDashboardRangeUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *salesDashboardRangeUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *salesDashboardRangeUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}

func (s *salesDashboardRangeUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}

func (s *salesDashboardRangeUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}

func (s *salesDashboardRangeUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *salesDashboardRangeUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}

func (s *salesDashboardRangeUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *salesDashboardRangeUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchSetConcurrency call")
}

func (s *salesDashboardRangeUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchAddConcurrency call")
}

func (s *salesDashboardRangeUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *salesDashboardRangeUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *salesDashboardRangeUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *salesDashboardRangeUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *salesDashboardRangeUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}

func (s *salesDashboardRangeUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}

func (s *salesDashboardRangeUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *salesDashboardRangeUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}

func (s *salesDashboardRangeUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}
