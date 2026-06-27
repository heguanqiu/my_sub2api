package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type upstreamRuntimeFeedbackRepo struct {
	UpstreamRepository
	lastEvent UpstreamRuntimeEvent
	snapshot  *UpstreamSchedulerSnapshot
}

func (r *upstreamRuntimeFeedbackRepo) RecordRuntimeEvent(ctx context.Context, event UpstreamRuntimeEvent) (*UpstreamSchedulerSnapshot, error) {
	r.lastEvent = event
	return r.snapshot, nil
}

type upstreamRuntimeFeedbackAccountRepo struct {
	AccountRepository
	account Account
	updated *Account
}

func (r *upstreamRuntimeFeedbackAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	account := r.account
	return &account, nil
}

func (r *upstreamRuntimeFeedbackAccountRepo) Update(ctx context.Context, account *Account) error {
	copy := *account
	r.updated = &copy
	return nil
}

func TestReportUpstreamRuntimeEventRefreshesHiddenAccountSchedulingFields(t *testing.T) {
	load := 20
	account := Account{
		ID:          42,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 20,
		Priority:    100,
		LoadFactor:  &load,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra: map[string]any{
			upstreamRuntimeManagedExtraKey: true,
			"upstream_id":                  int64(7),
			"upstream_routing_mode":        UpstreamRoutingStability,
			"upstream_base_priority":       int64(100),
			"upstream_base_load_factor":    int64(20),
		},
		AccountGroups:      []AccountGroup{{AccountID: 42, GroupID: 3}},
		GroupIDs:           []int64{3},
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		AutoPauseOnExpired: true,
	}
	accountRepo := &upstreamRuntimeFeedbackAccountRepo{account: account}
	upstreamRepo := &upstreamRuntimeFeedbackRepo{
		snapshot: &UpstreamSchedulerSnapshot{
			HealthScore:      0.5,
			PerformanceScore: 0.4,
			CapacityScore:    0.8,
		},
	}
	svc := NewOpenAIGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		upstreamRepo,
	)

	ttft := 3200
	svc.ReportOpenAIAccountScheduleEvent(context.Background(), account.ID, false, &ttft, 4*time.Second, 502, "bad gateway", false, "forward_error")

	if upstreamRepo.lastEvent.UpstreamID != 7 {
		t.Fatalf("upstream id = %d, want 7", upstreamRepo.lastEvent.UpstreamID)
	}
	if upstreamRepo.lastEvent.FirstTokenMs == nil || *upstreamRepo.lastEvent.FirstTokenMs != ttft {
		t.Fatalf("first token event = %v, want %d", upstreamRepo.lastEvent.FirstTokenMs, ttft)
	}
	if accountRepo.updated == nil {
		t.Fatal("expected hidden runtime account to be refreshed")
	}
	if got, want := accountRepo.updated.Priority, 140; got != want {
		t.Fatalf("priority = %d, want %d", got, want)
	}
	if accountRepo.updated.LoadFactor == nil || *accountRepo.updated.LoadFactor != 10 {
		t.Fatalf("load factor = %v, want 10", accountRepo.updated.LoadFactor)
	}
	if got := accountRepo.updated.Extra["upstream_health_score"]; got != 0.5 {
		t.Fatalf("health extra = %v, want 0.5", got)
	}
}

func TestUpstreamRuntimeSchedulingFieldsFollowRoutingMode(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		basePriority int
		baseLoad     int
		cost         float64
		health       float64
		wantPriority int
		wantLoad     int
	}{
		{
			name:         "balanced keeps base load but adjusts unhealthy priority",
			mode:         UpstreamRoutingBalanced,
			basePriority: 100,
			baseLoad:     20,
			cost:         1,
			health:       0.5,
			wantPriority: 140,
			wantLoad:     20,
		},
		{
			name:         "stability shrinks load when health drops",
			mode:         UpstreamRoutingStability,
			basePriority: 100,
			baseLoad:     20,
			cost:         1,
			health:       0.5,
			wantPriority: 140,
			wantLoad:     10,
		},
		{
			name:         "cost uses cost score and minimum probe load",
			mode:         UpstreamRoutingCost,
			basePriority: 100,
			baseLoad:     20,
			cost:         2,
			health:       1,
			wantPriority: 50,
			wantLoad:     1,
		},
		{
			name:         "manual keeps configured priority",
			mode:         UpstreamRoutingManual,
			basePriority: 100,
			baseLoad:     20,
			cost:         1,
			health:       0.5,
			wantPriority: 100,
			wantLoad:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upstreamRuntimePriorityForMode(tt.mode, tt.basePriority, tt.cost, tt.health, UpstreamStatusActive); got != tt.wantPriority {
				t.Fatalf("priority = %d, want %d", got, tt.wantPriority)
			}
			if got := upstreamRuntimeLoadFactorForMode(tt.mode, tt.baseLoad, tt.health); got != tt.wantLoad {
				t.Fatalf("load factor = %d, want %d", got, tt.wantLoad)
			}
		})
	}
}
