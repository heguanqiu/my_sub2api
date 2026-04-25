package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/invitelink"
	"github.com/Wei-Shaw/sub2api/ent/inviterewardledger"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var ErrInviteLinkNotFound = infraerrors.NotFound("INVITE_LINK_NOT_FOUND", "invite link not found")

type registrationAffiliation struct {
	InvitedByUserID      *int64
	OwnerSalesID         *int64
	InvitationRedeemCode *RedeemCode
}

func canUseRegistrationInvitationRedeemCode(code *RedeemCode) bool {
	if code == nil || code.Type != RedeemTypeInvitation {
		return false
	}
	// 邀请码类型改为可重复使用，不再依赖 used/unused 状态。
	return true
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func deriveOwnerSalesIDForReferral(user *User) *int64 {
	if user == nil {
		return nil
	}
	if user.IsSales() {
		id := user.ID
		return &id
	}
	return cloneInt64Ptr(user.OwnerSalesID)
}

func (s *AuthService) ResolveRegistrationAffiliation(ctx context.Context, invitationCode string) (*registrationAffiliation, error) {
	return s.resolveRegistrationAffiliation(ctx, invitationCode)
}

func (s *AuthService) defaultSalesOwnerID(ctx context.Context) *int64 {
	if s == nil || s.settingService == nil {
		return nil
	}
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil || settings == nil || settings.DefaultSalesUserID <= 0 {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, settings.DefaultSalesUserID)
	if err != nil || user == nil || !user.IsSales() {
		return nil
	}
	id := user.ID
	return &id
}

func (s *AuthService) resolveRegistrationAffiliation(ctx context.Context, invitationCode string) (*registrationAffiliation, error) {
	invitationCode = strings.TrimSpace(invitationCode)
	if invitationCode == "" {
		return nil, nil
	}

	if link, err := s.findActiveInviteLinkByCode(ctx, invitationCode); err == nil && link != nil {
		inviter, getErr := s.userRepo.GetByID(ctx, link.CreatedByUserID)
		if getErr != nil {
			return nil, ErrInvitationCodeInvalid
		}
		return &registrationAffiliation{
			InvitedByUserID: cloneInt64Ptr(&inviter.ID),
			OwnerSalesID:    deriveOwnerSalesIDForReferral(inviter),
		}, nil
	}

	if s.redeemRepo == nil {
		return nil, ErrInvitationCodeInvalid
	}
	redeemCode, err := s.redeemRepo.GetByCode(ctx, invitationCode)
	if err != nil {
		return nil, ErrInvitationCodeInvalid
	}
	if !canUseRegistrationInvitationRedeemCode(redeemCode) {
		return nil, ErrInvitationCodeInvalid
	}
	return &registrationAffiliation{InvitationRedeemCode: redeemCode}, nil
}

func (s *AuthService) findActiveInviteLinkByCode(ctx context.Context, code string) (*dbent.InviteLink, error) {
	if s == nil || s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvitationCodeInvalid
	}
	link, err := s.entClient.InviteLink.Query().
		Where(
			invitelink.CodeEQ(code),
			invitelink.StatusEQ(InviteLinkStatusActive),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrInvitationCodeInvalid
		}
		return nil, err
	}
	return link, nil
}

func inviteLinkEntityToService(m *dbent.InviteLink) *InviteLink {
	if m == nil {
		return nil
	}
	return &InviteLink{
		ID:              m.ID,
		Code:            m.Code,
		CreatedByUserID: m.CreatedByUserID,
		CreatorRole:     m.CreatorRole,
		OwnerSalesID:    m.OwnerSalesID,
		Status:          m.Status,
		Notes:           ptrStringValue(m.Notes),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func inviteRewardLedgerEntityToService(m *dbent.InviteRewardLedger) *InviteRewardLedger {
	if m == nil {
		return nil
	}
	return &InviteRewardLedger{
		ID:             m.ID,
		InviterUserID:  m.InviterUserID,
		InviteeUserID:  m.InviteeUserID,
		TriggerOrderID: m.TriggerOrderID,
		RewardType:     m.RewardType,
		RewardAmount:   m.RewardAmount,
		Status:         m.Status,
		Reason:         ptrStringValue(m.Reason),
		CreatedAt:      m.CreatedAt,
		ConfirmedAt:    m.ConfirmedAt,
		ReversedAt:     m.ReversedAt,
	}
}

func authUserEntityToService(u *dbent.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:               u.ID,
		Email:            u.Email,
		Username:         u.Username,
		Notes:            u.Notes,
		PasswordHash:     u.PasswordHash,
		Role:             u.Role,
		Balance:          u.Balance,
		Concurrency:      u.Concurrency,
		Status:           u.Status,
		SignupSource:     u.SignupSource,
		LastLoginAt:      u.LastLoginAt,
		LastActiveAt:     u.LastActiveAt,
		InvitedByUserID:  u.InvitedByUserID,
		OwnerSalesID:     u.OwnerSalesID,
		FirstPaidOrderID: u.FirstPaidOrderID,
		FirstPaidAt:      u.FirstPaidAt,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

func ptrStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nilIfEmptyTrimmed(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (s *AuthService) getLatestInviteLink(ctx context.Context, userID int64) (*InviteLink, error) {
	link, err := s.entClient.InviteLink.Query().
		Where(invitelink.CreatedByUserIDEQ(userID)).
		Order(dbent.Desc(invitelink.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrInviteLinkNotFound
		}
		return nil, err
	}
	return inviteLinkEntityToService(link), nil
}

func (s *AuthService) createInviteLink(ctx context.Context, link *InviteLink) error {
	if s == nil || s.entClient == nil || link == nil {
		return ErrServiceUnavailable
	}
	created, err := s.entClient.InviteLink.Create().
		SetCode(link.Code).
		SetCreatedByUserID(link.CreatedByUserID).
		SetCreatorRole(link.CreatorRole).
		SetNillableOwnerSalesID(link.OwnerSalesID).
		SetStatus(link.Status).
		SetNillableNotes(nilIfEmptyTrimmed(link.Notes)).
		Save(ctx)
	if err != nil {
		return err
	}
	*link = *inviteLinkEntityToService(created)
	return nil
}

func (s *AuthService) updateInviteLink(ctx context.Context, link *InviteLink) error {
	if s == nil || s.entClient == nil || link == nil {
		return ErrServiceUnavailable
	}
	updated, err := s.entClient.InviteLink.UpdateOneID(link.ID).
		SetStatus(link.Status).
		SetNillableOwnerSalesID(link.OwnerSalesID).
		SetNillableNotes(nilIfEmptyTrimmed(link.Notes)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	*link = *inviteLinkEntityToService(updated)
	return nil
}

func (s *AuthService) ensureMyInviteLink(ctx context.Context, userID int64) (*InviteLink, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	link, err := s.getLatestInviteLink(ctx, userID)
	if err != nil && !errors.Is(err, ErrInviteLinkNotFound) {
		return nil, err
	}
	if link != nil {
		return link, nil
	}
	code, err := randomHexString(12)
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}
	link = &InviteLink{
		Code:            code,
		CreatedByUserID: userID,
		CreatorRole:     user.Role,
		OwnerSalesID:    deriveOwnerSalesIDForReferral(user),
		Status:          InviteLinkStatusActive,
	}
	if err := s.createInviteLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *AuthService) GetMyInviteLink(ctx context.Context, userID int64) (*InviteLink, error) {
	return s.ensureMyInviteLink(ctx, userID)
}

func (s *AuthService) RegenerateMyInviteLink(ctx context.Context, userID int64) (*InviteLink, error) {
	current, err := s.ensureMyInviteLink(ctx, userID)
	if err != nil {
		return nil, err
	}
	current.Status = InviteLinkStatusRevoked
	if err := s.updateInviteLink(ctx, current); err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	code, err := randomHexString(12)
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}
	link := &InviteLink{
		Code:            code,
		CreatedByUserID: userID,
		CreatorRole:     user.Role,
		OwnerSalesID:    deriveOwnerSalesIDForReferral(user),
		Status:          InviteLinkStatusActive,
	}
	if err := s.createInviteLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *AuthService) UpdateMyInviteLinkStatus(ctx context.Context, userID int64, status string) (*InviteLink, error) {
	link, err := s.ensureMyInviteLink(ctx, userID)
	if err != nil {
		return nil, err
	}
	link.Status = status
	if err := s.updateInviteLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *AuthService) ListMyInvitees(ctx context.Context, userID int64, page, pageSize int, search string) ([]User, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: "created_at", SortOrder: "desc"}
	q := s.entClient.User.Query().Where(dbuser.InvitedByUserIDEQ(userID))
	search = strings.TrimSpace(search)
	if search != "" {
		q = q.Where(dbuser.Or(
			dbuser.EmailContainsFold(search),
			dbuser.UsernameContainsFold(search),
		))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := q.Order(dbent.Desc(dbuser.FieldCreatedAt)).Offset(params.Offset()).Limit(params.Limit()).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]User, 0, len(rows))
	for _, row := range rows {
		if user := authUserEntityToService(row); user != nil {
			out = append(out, *user)
		}
	}
	return out, &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.Limit(), Pages: paginationPages(total, params.Limit())}, nil
}

func (s *AuthService) ListMyInviteRewards(ctx context.Context, userID int64, page, pageSize int, status string) ([]InviteRewardLedger, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: "created_at", SortOrder: "desc"}
	q := s.entClient.InviteRewardLedger.Query().Where(inviterewardledger.InviterUserIDEQ(userID))
	status = strings.TrimSpace(status)
	if status != "" {
		q = q.Where(inviterewardledger.StatusEQ(status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := q.Order(dbent.Desc(inviterewardledger.FieldCreatedAt)).Offset(params.Offset()).Limit(params.Limit()).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]InviteRewardLedger, 0, len(rows))
	for _, row := range rows {
		if item := inviteRewardLedgerEntityToService(row); item != nil {
			out = append(out, *item)
		}
	}
	return out, &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.Limit(), Pages: paginationPages(total, params.Limit())}, nil
}

func paginationPages(total, pageSize int) int {
	if pageSize <= 0 {
		pageSize = 20
	}
	pages := total / pageSize
	if total%pageSize > 0 {
		pages++
	}
	if pages == 0 {
		return 0
	}
	return pages
}

func (s *AuthService) GetFrontendURL(ctx context.Context) string {
	if s == nil || s.settingService == nil {
		return ""
	}
	return strings.TrimSpace(s.settingService.GetFrontendURL(ctx))
}
