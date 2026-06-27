package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func TestUpstreamServiceResetCostReportSchedulesNextLocalMidnight(t *testing.T) {
	if err := timezone.Init("Asia/Shanghai"); err != nil {
		t.Fatalf("timezone.Init() error = %v", err)
	}
	repo := &upstreamCostReportRepo{
		upstream: &Upstream{ID: 7, Metadata: map[string]any{}},
	}
	svc := NewUpstreamService(repo, nil, nil, nil, nil)
	before := timezone.Now()

	result, err := svc.ResetCostReport(context.Background(), 7)
	if err != nil {
		t.Fatalf("ResetCostReport() error = %v", err)
	}
	if result == nil || result.UpstreamID != 7 || result.ResetAt.IsZero() {
		t.Fatalf("result = %+v, want reset result", result)
	}
	raw := repo.updated.Metadata["cost_report_reset_at"]
	parsed, err := time.Parse(time.RFC3339, raw.(string))
	if err != nil {
		t.Fatalf("stored reset_at parse error = %v", err)
	}
	if !parsed.Equal(result.ResetAt) {
		t.Fatalf("stored reset_at = %v, result reset_at = %v", parsed, result.ResetAt)
	}
	local := parsed.In(timezone.Location())
	if local.Hour() != 0 || local.Minute() != 0 || local.Second() != 0 || local.Nanosecond() != 0 {
		t.Fatalf("stored reset_at local time = %v, want local midnight", local)
	}
	if !parsed.After(before) {
		t.Fatalf("stored reset_at = %v, want after click time %v", parsed, before)
	}
}

func TestUpstreamServiceCostReportUsesResetAtAsLowerBound(t *testing.T) {
	resetAt := time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)
	repo := &upstreamCostReportRepo{
		upstream: &Upstream{ID: 7, Metadata: map[string]any{"cost_report_reset_at": resetAt.Format(time.RFC3339)}},
		report:   &UpstreamCostReport{Dimension: "upstream"},
	}
	svc := NewUpstreamService(repo, nil, nil, nil, nil)

	start := resetAt.Add(-24 * time.Hour)
	end := resetAt.Add(time.Hour)
	report, err := svc.CostReport(context.Background(), 7, start, end, "upstream")
	if err != nil {
		t.Fatalf("CostReport() error = %v", err)
	}
	if !repo.costStart.Equal(resetAt) {
		t.Fatalf("cost start = %v, want %v", repo.costStart, resetAt)
	}
	if report.ResetAt == nil || !report.ResetAt.Equal(resetAt) {
		t.Fatalf("report reset_at = %v, want %v", report.ResetAt, resetAt)
	}
	if !report.Start.Equal(resetAt) {
		t.Fatalf("report start = %v, want %v", report.Start, resetAt)
	}
}

func TestUpstreamServiceCostReportDoesNotApplyFutureResetBeforeItArrives(t *testing.T) {
	resetAt := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	repo := &upstreamCostReportRepo{
		upstream: &Upstream{ID: 7, Metadata: map[string]any{"cost_report_reset_at": resetAt.Format(time.RFC3339)}},
		report:   &UpstreamCostReport{Dimension: "upstream"},
	}
	svc := NewUpstreamService(repo, nil, nil, nil, nil)

	start := resetAt.Add(-6 * time.Hour)
	end := resetAt.Add(-time.Hour)
	report, err := svc.CostReport(context.Background(), 7, start, end, "upstream")
	if err != nil {
		t.Fatalf("CostReport() error = %v", err)
	}
	if !repo.costStart.Equal(start) {
		t.Fatalf("cost start = %v, want original start %v", repo.costStart, start)
	}
	if report.ResetAt == nil || !report.ResetAt.Equal(resetAt) {
		t.Fatalf("report reset_at = %v, want %v", report.ResetAt, resetAt)
	}
	if !report.Start.Equal(start) {
		t.Fatalf("report start = %v, want original start %v", report.Start, start)
	}
}

type upstreamCostReportRepo struct {
	UpstreamRepository
	upstream  *Upstream
	updated   *Upstream
	report    *UpstreamCostReport
	costStart time.Time
}

func (r *upstreamCostReportRepo) Get(ctx context.Context, id int64) (*Upstream, error) {
	if r.upstream == nil || r.upstream.ID != id {
		return nil, ErrUpstreamNotFound
	}
	copy := *r.upstream
	copy.Metadata = copyAnyMap(r.upstream.Metadata)
	return &copy, nil
}

func (r *upstreamCostReportRepo) Update(ctx context.Context, upstream *Upstream) error {
	copy := *upstream
	copy.Metadata = copyAnyMap(upstream.Metadata)
	r.updated = &copy
	r.upstream = &copy
	return nil
}

func (r *upstreamCostReportRepo) GetCostReport(ctx context.Context, upstreamID int64, start, end time.Time, dimension string) (*UpstreamCostReport, error) {
	r.costStart = start
	if r.report != nil {
		copy := *r.report
		copy.Start = start
		copy.End = end
		copy.Dimension = dimension
		return &copy, nil
	}
	return &UpstreamCostReport{Start: start, End: end, Dimension: dimension}, nil
}
