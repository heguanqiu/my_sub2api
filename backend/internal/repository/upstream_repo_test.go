package repository

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestReplaceRemoteResourcesDeletesStaleRemoteAPIKeys(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := &upstreamRepository{db: db}
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)

	expectReplaceRemoteResourcesStart(mock)
	mock.ExpectQuery("INSERT INTO upstream_remote_api_keys").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "api_key_encrypted", "synced_remote_group_id", "remote_group_id", "local_group_ids", "created_at", "updated_at",
		}).AddRow(int64(10), "", "vip", "vip", "{100}", now, now))
	mock.ExpectExec(`DELETE FROM upstream_remote_api_keys\s+WHERE upstream_id = \$1\s+AND remote_api_key_id <> ALL\(\$2::text\[\]\)`).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectReplaceRemoteResourcesFinish(mock)

	err = repo.ReplaceRemoteResources(context.Background(), 42, []*service.UpstreamRemoteGroup{}, []*service.UpstreamRemoteAPIKey{
		{
			RemoteAPIKeyID:      "key-live",
			RemoteAPIKeyName:    "Live",
			SyncedRemoteGroupID: "vip",
			RemoteGroupID:       "vip",
			Status:              service.UpstreamStatusActive,
			LocalGroupIDs:       []int64{100},
		},
	}, &service.UpstreamSyncRun{Status: "success", Message: "sync completed"})
	if err != nil {
		t.Fatalf("ReplaceRemoteResources() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestReplaceRemoteResourcesPreservesRemoteAPIKeysWhenListUnknown(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := &upstreamRepository{db: db}

	expectReplaceRemoteResourcesStart(mock)
	expectReplaceRemoteResourcesFinish(mock)

	err = repo.ReplaceRemoteResources(context.Background(), 42, []*service.UpstreamRemoteGroup{}, nil, &service.UpstreamSyncRun{Status: "success", Message: "sync completed without remote api keys"})
	if err != nil {
		t.Fatalf("ReplaceRemoteResources() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestReplaceRemoteResourcesDeletesAllRemoteAPIKeysWhenUpstreamReturnsEmptyList(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := &upstreamRepository{db: db}

	expectReplaceRemoteResourcesStart(mock)
	mock.ExpectExec(`DELETE FROM upstream_remote_api_keys\s+WHERE upstream_id = \$1\s*$`).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 5))
	expectReplaceRemoteResourcesFinish(mock)

	err = repo.ReplaceRemoteResources(context.Background(), 42, []*service.UpstreamRemoteGroup{}, []*service.UpstreamRemoteAPIKey{}, &service.UpstreamSyncRun{Status: "success", Message: "sync completed"})
	if err != nil {
		t.Fatalf("ReplaceRemoteResources() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetCostReportJoinsRemoteGroupRateMultiplier(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := &upstreamRepository{db: db}
	start := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery(`(?s)SELECT .*urg\.rate_multiplier.*FROM usage_logs.*LEFT JOIN upstream_remote_api_keys.*LEFT JOIN upstream_remote_groups`).
		WithArgs(int64(7), start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"upstream_id", "upstream_name", "request_count", "local_billed_cost", "upstream_cost", "avg_multiplier",
		}).AddRow(int64(7), "north", int64(2), 1.25, 3.75, 3.0))

	report, err := repo.GetCostReport(context.Background(), 7, start, end, "upstream")
	if err != nil {
		t.Fatalf("GetCostReport() error = %v", err)
	}
	if report.Totals.UpstreamCost != 3.75 {
		t.Fatalf("upstream cost = %v, want 3.75", report.Totals.UpstreamCost)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func expectReplaceRemoteResourcesStart(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO upstream_sync_runs").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectExec("DELETE FROM upstream_remote_groups").
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 0))
}

func expectReplaceRemoteResourcesFinish(mock sqlmock.Sqlmock) {
	mock.ExpectExec("UPDATE upstreams").
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
}
