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

func TestSalesDashboardRangeFiltersOrders(t *testing.T) {
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
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
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

	createSalesRangeOrder("range-today", 10, now)
	createSalesRangeOrder("range-week", 20, now.AddDate(0, 0, -3))
	createSalesRangeOrder("range-month", 30, now.AddDate(0, 0, -20))
	createSalesRangeOrder("range-old", 40, now.AddDate(0, 0, -40))

	today, err := svc.GetSalesDashboard(ctx, sales.ID, SalesDashboardRangeToday)
	require.NoError(t, err)
	require.Equal(t, SalesDashboardRangeToday, today.Range)
	require.Equal(t, 1, today.TotalOrders)
	require.Equal(t, 10.0, today.TotalOrderAmount)

	week, err := svc.GetSalesDashboard(ctx, sales.ID, SalesDashboardRange7Days)
	require.NoError(t, err)
	require.Equal(t, SalesDashboardRange7Days, week.Range)
	require.Equal(t, 2, week.TotalOrders)
	require.Equal(t, 30.0, week.TotalOrderAmount)

	month, err := svc.GetSalesDashboard(ctx, sales.ID, SalesDashboardRange30Days)
	require.NoError(t, err)
	require.Equal(t, SalesDashboardRange30Days, month.Range)
	require.Equal(t, 3, month.TotalOrders)
	require.Equal(t, 60.0, month.TotalOrderAmount)
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
