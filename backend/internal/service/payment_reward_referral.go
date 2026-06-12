package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/inviterewardledger"
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
