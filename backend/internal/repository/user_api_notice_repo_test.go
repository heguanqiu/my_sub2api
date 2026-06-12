package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserAPINoticeRepositoryConsumeReplaysRecentlyConsumedNotice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &UserAPINoticeRepository{db: db}
	now := time.Date(2026, 6, 13, 2, 45, 0, 0, time.UTC)
	consumedAt := now.Add(-30 * time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "message", "status", "created_by_user_id", "created_at", "updated_at",
		"expires_at", "consumed_at", "consumed_request_id", "cancelled_at", "cancelled_by_user_id",
	}).AddRow(
		int64(9), int64(42), "请先联系管理员", service.UserAPINoticeStatusConsumed,
		sql.NullInt64{Int64: 1, Valid: true}, now.Add(-time.Minute), consumedAt,
		sql.NullTime{}, sql.NullTime{Time: consumedAt, Valid: true}, sql.NullString{String: "req-first", Valid: true},
		sql.NullTime{}, sql.NullInt64{},
	)

	mock.ExpectQuery("WITH picked AS").WithArgs(
		int64(42),
		now,
		"req-retry",
		now.Add(-service.UserAPINoticeReplayWindow),
	).WillReturnRows(rows)

	notice, err := repo.ConsumeNextPending(context.Background(), 42, "req-retry", now)

	require.NoError(t, err)
	require.NotNil(t, notice)
	require.Equal(t, int64(9), notice.ID)
	require.Equal(t, service.UserAPINoticeStatusConsumed, notice.Status)
	require.Equal(t, "请先联系管理员", notice.Message)
	require.NoError(t, mock.ExpectationsWereMet())
}
