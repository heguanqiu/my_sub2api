package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type UserAPINoticeRepository struct {
	db *sql.DB
}

func NewUserAPINoticeRepository(db *sql.DB) service.UserAPINoticeRepository {
	return &UserAPINoticeRepository{db: db}
}

func (r *UserAPINoticeRepository) Create(ctx context.Context, input service.CreateUserAPINoticeInput) (*service.UserAPINotice, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO user_api_notices (user_id, message, created_by_user_id, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, message, status, created_by_user_id, created_at, updated_at,
          expires_at, consumed_at, consumed_request_id, cancelled_at, cancelled_by_user_id`,
		input.UserID,
		input.Message,
		nullableInt64(input.CreatedByUserID),
		input.ExpiresAt,
	)
	return scanUserAPINotice(row)
}

func (r *UserAPINoticeRepository) List(ctx context.Context, filter service.UserAPINoticeFilter, params pagination.PaginationParams) ([]service.UserAPINotice, *pagination.PaginationResult, error) {
	where := []string{"user_id = $1"}
	args := []any{filter.UserID}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_api_notices WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx, `
SELECT id, user_id, message, status, created_by_user_id, created_at, updated_at,
       expires_at, consumed_at, consumed_request_id, cancelled_at, cancelled_by_user_id
FROM user_api_notices
WHERE `+whereSQL+`
ORDER BY created_at DESC, id DESC
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	notices := make([]service.UserAPINotice, 0)
	for rows.Next() {
		notice, scanErr := scanUserAPINotice(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		notices = append(notices, *notice)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return notices, paginationResultFromTotal(total, params), nil
}

func (r *UserAPINoticeRepository) Cancel(ctx context.Context, id int64, operatorID int64) (*service.UserAPINotice, error) {
	row := r.db.QueryRowContext(ctx, `
UPDATE user_api_notices
SET status = 'cancelled',
    cancelled_at = NOW(),
    cancelled_by_user_id = $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'pending'
RETURNING id, user_id, message, status, created_by_user_id, created_at, updated_at,
          expires_at, consumed_at, consumed_request_id, cancelled_at, cancelled_by_user_id`,
		id,
		nullableInt64(operatorID),
	)
	notice, err := scanUserAPINotice(row)
	if err == nil {
		return notice, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var status string
	if statusErr := r.db.QueryRowContext(ctx, "SELECT status FROM user_api_notices WHERE id = $1", id).Scan(&status); statusErr != nil {
		if errors.Is(statusErr, sql.ErrNoRows) {
			return nil, service.ErrUserAPINoticeNotFound
		}
		return nil, statusErr
	}
	return nil, service.ErrUserAPINoticeCannotBeCanceled
}

func (r *UserAPINoticeRepository) ConsumeNextPending(ctx context.Context, userID int64, requestID string, now time.Time) (*service.UserAPINotice, error) {
	replaySince := now.Add(-service.UserAPINoticeReplayWindow)
	row := r.db.QueryRowContext(ctx, `
WITH picked AS (
    SELECT id
    FROM user_api_notices
    WHERE user_id = $1
      AND status = 'pending'
      AND (expires_at IS NULL OR expires_at > $2)
    ORDER BY created_at ASC, id ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
),
consumed AS (
UPDATE user_api_notices n
SET status = 'consumed',
    consumed_at = $2,
    consumed_request_id = NULLIF($3, ''),
    updated_at = $2
FROM picked
WHERE n.id = picked.id
RETURNING n.id, n.user_id, n.message, n.status, n.created_by_user_id, n.created_at, n.updated_at,
          n.expires_at, n.consumed_at, n.consumed_request_id, n.cancelled_at, n.cancelled_by_user_id
),
replay AS (
    SELECT id, user_id, message, status, created_by_user_id, created_at, updated_at,
           expires_at, consumed_at, consumed_request_id, cancelled_at, cancelled_by_user_id
    FROM user_api_notices
    WHERE user_id = $1
      AND status = 'consumed'
      AND consumed_at IS NOT NULL
      AND consumed_at >= $4
      AND (expires_at IS NULL OR expires_at > $2)
      AND NOT EXISTS (SELECT 1 FROM consumed)
    ORDER BY consumed_at DESC, id DESC
    LIMIT 1
)
SELECT id, user_id, message, status, created_by_user_id, created_at, updated_at,
       expires_at, consumed_at, consumed_request_id, cancelled_at, cancelled_by_user_id
FROM consumed
UNION ALL
SELECT id, user_id, message, status, created_by_user_id, created_at, updated_at,
       expires_at, consumed_at, consumed_request_id, cancelled_at, cancelled_by_user_id
FROM replay
LIMIT 1`,
		userID,
		now,
		requestID,
		replaySince,
	)
	notice, err := scanUserAPINotice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return notice, err
}

type userAPINoticeScanner interface {
	Scan(dest ...any) error
}

func scanUserAPINotice(scanner userAPINoticeScanner) (*service.UserAPINotice, error) {
	var notice service.UserAPINotice
	var createdBy sql.NullInt64
	var expiresAt sql.NullTime
	var consumedAt sql.NullTime
	var consumedRequestID sql.NullString
	var cancelledAt sql.NullTime
	var cancelledBy sql.NullInt64

	if err := scanner.Scan(
		&notice.ID,
		&notice.UserID,
		&notice.Message,
		&notice.Status,
		&createdBy,
		&notice.CreatedAt,
		&notice.UpdatedAt,
		&expiresAt,
		&consumedAt,
		&consumedRequestID,
		&cancelledAt,
		&cancelledBy,
	); err != nil {
		return nil, err
	}

	notice.CreatedByUserID = ptrFromNullInt64(createdBy)
	notice.ExpiresAt = ptrFromNullTime(expiresAt)
	notice.ConsumedAt = ptrFromNullTime(consumedAt)
	if consumedRequestID.Valid {
		v := consumedRequestID.String
		notice.ConsumedRequestID = &v
	}
	notice.CancelledAt = ptrFromNullTime(cancelledAt)
	notice.CancelledByUserID = ptrFromNullInt64(cancelledBy)
	return &notice, nil
}

func nullableInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v > 0}
}

func ptrFromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

func ptrFromNullTime(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	out := v.Time
	return &out
}
