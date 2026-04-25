package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/inviterewardledger"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/shopspring/decimal"
)

func orderInviteRewardStatusForUser(user *User, orderType string) string {
	if orderType != "balance" {
		return OrderInviteRewardStatusNotApplicable
	}
	if user == nil || user.InvitedByUserID == nil {
		return OrderInviteRewardStatusNotApplicable
	}
	return OrderInviteRewardStatusPendingEvaluation
}

func (s *PaymentService) maybeApplyInviteReward(ctx context.Context, orderID int64) {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil || order.OrderType != "balance" {
		return
	}
	if order.InviteRewardLedgerID != nil || order.InviteRewardStatus == OrderInviteRewardStatusGranted {
		return
	}
	if order.InvitedByUserIDSnapshot == nil {
		_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).SetInviteRewardStatus(OrderInviteRewardStatusNotApplicable).Save(ctx)
		return
	}
	if s.configService == nil {
		return
	}
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return
	}
	if !cfg.InviteRewardEnabled || cfg.InviteRewardRate <= 0 {
		_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).SetInviteRewardStatus(OrderInviteRewardStatusSkipped).Save(ctx)
		return
	}
	now := time.Now()
	recordedFirstPaid, err := s.recordFirstPaidOrderIfMissing(ctx, order.UserID, order.ID, now)
	if err != nil {
		return
	}
	if normalizeInviteRewardTriggerMode(cfg.InviteRewardTriggerMode) == InviteRewardTriggerFirstBalanceOrder && !recordedFirstPaid {
		_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).SetInviteRewardStatus(OrderInviteRewardStatusSkipped).Save(ctx)
		return
	}
	rewardAmount := calculateInviteRewardAmount(order.Amount, cfg.InviteRewardRate)
	if rewardAmount <= 0 {
		_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).SetInviteRewardStatus(OrderInviteRewardStatusSkipped).Save(ctx)
		return
	}
	ledger, err := s.entClient.InviteRewardLedger.Create().
		SetInviterUserID(*order.InvitedByUserIDSnapshot).
		SetInviteeUserID(order.UserID).
		SetTriggerOrderID(order.ID).
		SetRewardType(InviteRewardTypeBalance).
		SetRewardAmount(rewardAmount).
		SetStatus(InviteRewardStatusGranted).
		SetConfirmedAt(now).
		Save(ctx)
	if err != nil {
		_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).SetInviteRewardStatus(OrderInviteRewardStatusSkipped).Save(ctx)
		return
	}
	if err := s.userRepo.UpdateBalance(ctx, ledger.InviterUserID, ledger.RewardAmount); err != nil {
		_, _ = s.entClient.InviteRewardLedger.UpdateOneID(ledger.ID).
			SetStatus(InviteRewardStatusPending).
			ClearConfirmedAt().
			SetReason(err.Error()).
			Save(ctx)
		return
	}
	_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).
		SetInviteRewardStatus(OrderInviteRewardStatusGranted).
		SetInviteRewardLedgerID(ledger.ID).
		Save(ctx)
}

func (s *PaymentService) recordFirstPaidOrderIfMissing(ctx context.Context, userID, orderID int64, now time.Time) (bool, error) {
	updated, err := s.entClient.User.Update().
		Where(dbuser.IDEQ(userID), dbuser.FirstPaidOrderIDIsNil()).
		SetFirstPaidOrderID(orderID).
		SetFirstPaidAt(now).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return updated > 0, nil
}

func calculateInviteRewardAmount(orderAmount, rate float64) float64 {
	if orderAmount <= 0 || rate <= 0 {
		return 0
	}
	return decimal.NewFromFloat(orderAmount).
		Mul(decimal.NewFromFloat(rate)).
		Div(decimal.NewFromInt(100)).
		Round(8).
		InexactFloat64()
}

func (s *PaymentService) maybeReverseInviteReward(ctx context.Context, orderID int64) {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil || order.InviteRewardLedgerID == nil {
		return
	}
	ledger, err := s.entClient.InviteRewardLedger.Get(ctx, *order.InviteRewardLedgerID)
	if err != nil || ledger.Status == InviteRewardStatusReversed {
		return
	}
	if err := s.userRepo.DeductBalance(ctx, ledger.InviterUserID, ledger.RewardAmount); err != nil {
		return
	}
	now := time.Now()
	_, _ = s.entClient.InviteRewardLedger.UpdateOneID(ledger.ID).
		SetStatus(InviteRewardStatusReversed).
		SetReversedAt(now).
		Save(ctx)
	_, _ = s.entClient.PaymentOrder.UpdateOneID(orderID).
		SetInviteRewardStatus(OrderInviteRewardStatusReversed).
		Save(ctx)
}

func (s *PaymentService) ListInviteRewardLedger(ctx context.Context, page, pageSize int, status string) ([]*dbent.InviteRewardLedger, int, error) {
	q := s.entClient.InviteRewardLedger.Query()
	if status != "" {
		q = q.Where(inviterewardledger.StatusEQ(status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(dbent.Desc(inviterewardledger.FieldCreatedAt)).Offset(pageOffset(page, pageSize)).Limit(limitValue(pageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *PaymentService) GetInviteRewardLedgerByID(ctx context.Context, id int64) (*dbent.InviteRewardLedger, error) {
	return s.entClient.InviteRewardLedger.Get(ctx, id)
}
