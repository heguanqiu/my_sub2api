package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *OpenAIGatewayService) reportUpstreamRuntimeEvent(ctx context.Context, accountID int64, success bool, firstTokenMs *int, duration time.Duration, statusCode int, errorMessage string, ignored bool, reason string) {
	if s == nil || s.upstreamRepo == nil || accountID <= 0 {
		return
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return
	}
	upstreamID := accountUpstreamID(account)
	if upstreamID <= 0 {
		return
	}
	if strings.TrimSpace(reason) == "" {
		if ignored {
			reason = "client_disconnect"
		} else if success {
			reason = "success"
		} else {
			reason = "upstream_error"
		}
	}
	event := UpstreamRuntimeEvent{
		UpstreamID:   upstreamID,
		AccountID:    accountID,
		Success:      success,
		FirstTokenMs: firstTokenMs,
		DurationMs:   duration.Milliseconds(),
		StatusCode:   statusCode,
		ErrorMessage: strings.TrimSpace(errorMessage),
		Ignored:      ignored,
		Reason:       reason,
		ObservedAt:   time.Now().UTC(),
	}
	snapshot, err := s.upstreamRepo.RecordRuntimeEvent(ctx, event)
	if err != nil {
		logger.LegacyPrintf("service.openai_gateway", "[UpstreamRuntime] record event failed: upstream=%d account=%d err=%v", upstreamID, accountID, err)
		return
	}
	if snapshot != nil {
		s.refreshUpstreamRuntimeAccountFromSnapshot(ctx, account, snapshot)
	}
}

func (s *OpenAIGatewayService) refreshUpstreamRuntimeAccountFromSnapshot(ctx context.Context, account *Account, snapshot *UpstreamSchedulerSnapshot) {
	if s == nil || s.accountRepo == nil || account == nil || snapshot == nil {
		return
	}
	next := *account
	next.Credentials = copyAnyMap(account.Credentials)
	next.Extra = copyAnyMap(account.Extra)
	if next.Extra == nil {
		next.Extra = map[string]any{}
	}
	next.Extra["upstream_health_score"] = snapshot.HealthScore
	next.Extra["upstream_performance_score"] = snapshot.PerformanceScore
	next.Extra["upstream_capacity_score"] = snapshot.CapacityScore

	health := clampUpstreamScore01(defaultScore(snapshot.HealthScore))
	if health < 0.2 {
		next.Status = StatusError
		next.Schedulable = false
		next.ErrorMessage = "upstream health too low"
	} else if next.Status == StatusError && strings.Contains(strings.ToLower(next.ErrorMessage), "upstream health too low") {
		next.Status = StatusActive
		next.Schedulable = true
		next.ErrorMessage = ""
	}

	basePriority := parseAnyInt64(next.Extra["upstream_base_priority"])
	if basePriority <= 0 {
		basePriority = int64(next.Priority)
	}
	if basePriority <= 0 {
		basePriority = 100
	}
	priority := int(basePriority)
	if health < 0.9 {
		priority += int((0.9 - health) * 100)
	}
	next.Priority = clampInt(priority, 1, 10000)

	baseLoad := parseAnyInt64(next.Extra["upstream_base_load_factor"])
	if baseLoad <= 0 {
		baseLoad = int64(next.EffectiveLoadFactor())
	}
	if baseLoad <= 0 {
		baseLoad = 1
	}
	loadFactor := clampInt(int(float64(baseLoad)*health), 1, int(baseLoad))
	next.LoadFactor = &loadFactor

	if err := s.accountRepo.Update(ctx, &next); err != nil {
		logger.LegacyPrintf("service.openai_gateway", "[UpstreamRuntime] refresh account failed: upstream=%d account=%d err=%v", accountUpstreamID(account), account.ID, err)
	}
}

func accountUpstreamID(account *Account) int64 {
	if account == nil || account.Extra == nil {
		return 0
	}
	if !extraBool(account.Extra[upstreamRuntimeManagedExtraKey]) {
		return 0
	}
	raw := account.Extra["upstream_id"]
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		i, _ := v.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i
	default:
		return 0
	}
}
