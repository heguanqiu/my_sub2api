package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"

	"github.com/stretchr/testify/require"
)

type inviteRewardUserRepoStub struct {
	UserRepository
	client *dbent.Client
}

func (r *inviteRewardUserRepoStub) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	user, err := r.client.User.Get(ctx, id)
	if err != nil {
		return err
	}
	return r.client.User.UpdateOneID(id).SetBalance(user.Balance + amount).Exec(ctx)
}

func TestMaybeApplyInviteReward_FirstBalanceOrderModeRewardsOnlyOnce(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	inviterID, inviteeID := createInviteRewardTestUsers(t, ctx, client)

	settingRepo := &paymentConfigSettingRepoStub{values: map[string]string{
		SettingInviteRewardEnabled: "true",
		SettingInviteRewardRate:    "50.00",
		SettingInviteRewardTrigger: InviteRewardTriggerFirstBalanceOrder,
	}}
	svc := &PaymentService{
		entClient:     client,
		userRepo:      &inviteRewardUserRepoStub{client: client},
		configService: &PaymentConfigService{entClient: client, settingRepo: settingRepo},
	}

	order1 := createInviteRewardTestOrder(t, ctx, client, inviteeID, inviterID, 20)
	svc.maybeApplyInviteReward(ctx, order1.ID)

	inviter, err := client.User.Get(ctx, inviterID)
	require.NoError(t, err)
	require.InDelta(t, 10.0, inviter.Balance, 0.0000001)

	order1Reloaded, err := client.PaymentOrder.Get(ctx, order1.ID)
	require.NoError(t, err)
	require.Equal(t, OrderInviteRewardStatusGranted, order1Reloaded.InviteRewardStatus)
	require.NotNil(t, order1Reloaded.InviteRewardLedgerID)

	ledgerCount, err := client.InviteRewardLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, ledgerCount)

	invitee, err := client.User.Get(ctx, inviteeID)
	require.NoError(t, err)
	require.NotNil(t, invitee.FirstPaidOrderID)
	require.Equal(t, order1.ID, *invitee.FirstPaidOrderID)

	order2 := createInviteRewardTestOrder(t, ctx, client, inviteeID, inviterID, 30)
	svc.maybeApplyInviteReward(ctx, order2.ID)

	inviter, err = client.User.Get(ctx, inviterID)
	require.NoError(t, err)
	require.InDelta(t, 10.0, inviter.Balance, 0.0000001)

	order2Reloaded, err := client.PaymentOrder.Get(ctx, order2.ID)
	require.NoError(t, err)
	require.Equal(t, OrderInviteRewardStatusSkipped, order2Reloaded.InviteRewardStatus)

	ledgerCount, err = client.InviteRewardLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, ledgerCount)
}

func TestMaybeApplyInviteReward_EveryBalanceOrderModeRewardsEachOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	inviterID, inviteeID := createInviteRewardTestUsers(t, ctx, client)

	settingRepo := &paymentConfigSettingRepoStub{values: map[string]string{
		SettingInviteRewardEnabled: "true",
		SettingInviteRewardRate:    "10.00",
		SettingInviteRewardTrigger: InviteRewardTriggerEveryBalanceOrder,
	}}
	svc := &PaymentService{
		entClient:     client,
		userRepo:      &inviteRewardUserRepoStub{client: client},
		configService: &PaymentConfigService{entClient: client, settingRepo: settingRepo},
	}

	order1 := createInviteRewardTestOrder(t, ctx, client, inviteeID, inviterID, 20)
	order2 := createInviteRewardTestOrder(t, ctx, client, inviteeID, inviterID, 30)

	svc.maybeApplyInviteReward(ctx, order1.ID)
	svc.maybeApplyInviteReward(ctx, order2.ID)

	inviter, err := client.User.Get(ctx, inviterID)
	require.NoError(t, err)
	require.InDelta(t, 5.0, inviter.Balance, 0.0000001)

	ledgerCount, err := client.InviteRewardLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, ledgerCount)

	order1Reloaded, err := client.PaymentOrder.Get(ctx, order1.ID)
	require.NoError(t, err)
	require.Equal(t, OrderInviteRewardStatusGranted, order1Reloaded.InviteRewardStatus)

	order2Reloaded, err := client.PaymentOrder.Get(ctx, order2.ID)
	require.NoError(t, err)
	require.Equal(t, OrderInviteRewardStatusGranted, order2Reloaded.InviteRewardStatus)

	invitee, err := client.User.Get(ctx, inviteeID)
	require.NoError(t, err)
	require.NotNil(t, invitee.FirstPaidOrderID)
	require.Equal(t, order1.ID, *invitee.FirstPaidOrderID)
	require.NotNil(t, invitee.FirstPaidAt)
}

func createInviteRewardTestUsers(t *testing.T, ctx context.Context, client *dbent.Client) (int64, int64) {
	t.Helper()

	inviter, err := client.User.Create().
		SetEmail("inviter@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetUsername("inviter").
		Save(ctx)
	require.NoError(t, err)

	invitee, err := client.User.Create().
		SetEmail("invitee@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetUsername("invitee").
		Save(ctx)
	require.NoError(t, err)

	return inviter.ID, invitee.ID
}

func createInviteRewardTestOrder(t *testing.T, ctx context.Context, client *dbent.Client, userID, inviterID int64, amount float64) *dbent.PaymentOrder {
	t.Helper()

	order, err := client.PaymentOrder.Create().
		SetUserID(userID).
		SetUserEmail("invitee@example.com").
		SetUserName("invitee").
		SetAmount(amount).
		SetPayAmount(amount).
		SetFeeRate(0).
		SetRechargeCode("REWARD-TEST").
		SetOutTradeNo(generateOutTradeNo()).
		SetPaymentType("alipay").
		SetPaymentTradeNo("trade-test").
		SetOrderType("balance").
		SetStatus(OrderStatusCompleted).
		SetInviteRewardStatus(OrderInviteRewardStatusPendingEvaluation).
		SetInvitedByUserIDSnapshot(inviterID).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetCompletedAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.com").
		Save(ctx)
	require.NoError(t, err)
	return order
}
