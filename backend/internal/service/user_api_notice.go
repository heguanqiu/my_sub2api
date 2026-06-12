package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	UserAPINoticeStatusPending   = "pending"
	UserAPINoticeStatusConsumed  = "consumed"
	UserAPINoticeStatusCancelled = "cancelled"

	maxUserAPINoticeMessageRunes = 2000
	UserAPINoticeReplayWindow    = 2 * time.Minute
)

var (
	ErrUserAPINoticeNotFound         = infraerrors.NotFound("USER_API_NOTICE_NOT_FOUND", "user API notice not found")
	ErrUserAPINoticeMessageRequired  = infraerrors.BadRequest("USER_API_NOTICE_MESSAGE_REQUIRED", "message is required")
	ErrUserAPINoticeMessageTooLong   = infraerrors.BadRequest("USER_API_NOTICE_MESSAGE_TOO_LONG", "message is too long")
	ErrUserAPINoticeInvalidStatus    = infraerrors.BadRequest("USER_API_NOTICE_INVALID_STATUS", "invalid notice status")
	ErrUserAPINoticeInvalidExpiry    = infraerrors.BadRequest("USER_API_NOTICE_INVALID_EXPIRY", "expires_at must be in the future")
	ErrUserAPINoticeCannotBeCanceled = infraerrors.Conflict("USER_API_NOTICE_CANNOT_BE_CANCELED", "only pending notices can be canceled")
)

type UserAPINotice struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	Message           string     `json:"message"`
	Status            string     `json:"status"`
	CreatedByUserID   *int64     `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	ConsumedAt        *time.Time `json:"consumed_at,omitempty"`
	ConsumedRequestID *string    `json:"consumed_request_id,omitempty"`
	CancelledAt       *time.Time `json:"cancelled_at,omitempty"`
	CancelledByUserID *int64     `json:"cancelled_by_user_id,omitempty"`
}

type CreateUserAPINoticeInput struct {
	UserID          int64
	Message         string
	CreatedByUserID int64
	ExpiresAt       *time.Time
}

type UserAPINoticeFilter struct {
	UserID int64
	Status string
}

type UserAPINoticeRepository interface {
	Create(ctx context.Context, input CreateUserAPINoticeInput) (*UserAPINotice, error)
	List(ctx context.Context, filter UserAPINoticeFilter, params pagination.PaginationParams) ([]UserAPINotice, *pagination.PaginationResult, error)
	Cancel(ctx context.Context, id int64, operatorID int64) (*UserAPINotice, error)
	ConsumeNextPending(ctx context.Context, userID int64, requestID string, now time.Time) (*UserAPINotice, error)
}

type UserAPINoticeService struct {
	repo     UserAPINoticeRepository
	userRepo UserRepository
}

func NewUserAPINoticeService(repo UserAPINoticeRepository, userRepo UserRepository) *UserAPINoticeService {
	return &UserAPINoticeService{repo: repo, userRepo: userRepo}
}

func (s *UserAPINoticeService) Create(ctx context.Context, input CreateUserAPINoticeInput) (*UserAPINotice, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.InternalServer("USER_API_NOTICE_SERVICE_UNAVAILABLE", "user API notice service is unavailable")
	}
	if input.UserID <= 0 {
		return nil, ErrUserNotFound
	}
	if s.userRepo != nil {
		if _, err := s.userRepo.GetByID(ctx, input.UserID); err != nil {
			return nil, err
		}
	}

	message, err := normalizeUserAPINoticeMessage(input.Message)
	if err != nil {
		return nil, err
	}
	input.Message = message
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return nil, ErrUserAPINoticeInvalidExpiry
	}
	return s.repo.Create(ctx, input)
}

func (s *UserAPINoticeService) List(ctx context.Context, filter UserAPINoticeFilter, params pagination.PaginationParams) ([]UserAPINotice, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, infraerrors.InternalServer("USER_API_NOTICE_SERVICE_UNAVAILABLE", "user API notice service is unavailable")
	}
	if filter.UserID <= 0 {
		return nil, nil, ErrUserNotFound
	}
	if filter.Status != "" && !isValidUserAPINoticeStatus(filter.Status) {
		return nil, nil, ErrUserAPINoticeInvalidStatus
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	return s.repo.List(ctx, filter, params)
}

func (s *UserAPINoticeService) Cancel(ctx context.Context, id int64, operatorID int64) (*UserAPINotice, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.InternalServer("USER_API_NOTICE_SERVICE_UNAVAILABLE", "user API notice service is unavailable")
	}
	if id <= 0 {
		return nil, ErrUserAPINoticeNotFound
	}
	return s.repo.Cancel(ctx, id, operatorID)
}

func (s *UserAPINoticeService) ConsumeNextPending(ctx context.Context, userID int64, requestID string, now time.Time) (*UserAPINotice, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	return s.repo.ConsumeNextPending(ctx, userID, strings.TrimSpace(requestID), now)
}

func normalizeUserAPINoticeMessage(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", ErrUserAPINoticeMessageRequired
	}
	if utf8.RuneCountInString(message) > maxUserAPINoticeMessageRunes {
		return "", ErrUserAPINoticeMessageTooLong
	}
	return message, nil
}

func isValidUserAPINoticeStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case UserAPINoticeStatusPending, UserAPINoticeStatusConsumed, UserAPINoticeStatusCancelled:
		return true
	default:
		return false
	}
}
