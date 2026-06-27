package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func (s *UpstreamService) HealthDashboard(ctx context.Context, id int64) (*UpstreamHealthDashboard, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUpstreamNotFound
	}
	dashboard, err := s.repo.GetHealthDashboard(ctx, id)
	if err != nil {
		return nil, err
	}
	if dashboard.SchedulerSnapshot.HealthScore < 0.5 {
		_ = s.repo.UpsertAlert(ctx, UpstreamAlert{
			UpstreamID: &id,
			AlertType:  "upstream_consecutive_failures",
			Severity:   "warning",
			Title:      "Upstream health degraded",
			Message:    dashboard.RecentErrorReason,
			Evidence: map[string]any{
				"health_score": dashboard.SchedulerSnapshot.HealthScore,
				"recent_error": dashboard.RecentErrorReason,
			},
		})
	} else {
		_ = s.repo.ResolveAlert(ctx, id, "upstream_consecutive_failures")
	}
	if dashboard.CircuitOpen {
		_ = s.repo.UpsertAlert(ctx, UpstreamAlert{
			UpstreamID: &id,
			AlertType:  "upstream_circuit_open",
			Severity:   "critical",
			Title:      "Upstream circuit breaker open",
			Message:    "Upstream is currently excluded from scheduling.",
		})
	} else {
		_ = s.repo.ResolveAlert(ctx, id, "upstream_circuit_open")
	}
	return dashboard, nil
}

func (s *UpstreamService) ListEvents(ctx context.Context, id int64, limit int, eventType string) ([]UpstreamEvent, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, id, limit, eventType)
}

func (s *UpstreamService) PreviewSyncRemoteResources(ctx context.Context, id int64) (*UpstreamSyncPreview, error) {
	started := time.Now().UTC()
	upstream, groups, keys, err := s.fetchRemoteResources(ctx, id)
	if err != nil {
		_ = s.recordFailedSync(ctx, id, started, err)
		return nil, err
	}
	currentGroups, err := s.repo.ListRemoteGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	currentKeys, err := s.repo.ListRemoteAPIKeys(ctx, id)
	if err != nil {
		return nil, err
	}
	diff := BuildUpstreamSyncDiff(currentGroups, currentKeys, groups, keys)
	preview := &UpstreamSyncPreview{
		UpstreamID:   id,
		PreviewToken: newPreviewToken(),
		Status:       "pending",
		Diff:         diff,
		Groups:       groups,
		APIKeys:      keys,
		CreatedAt:    time.Now().UTC(),
	}
	preview.ExpiresAt = preview.CreatedAt.Add(30 * time.Minute)
	if err := s.repo.CreateSyncPreview(ctx, preview); err != nil {
		return nil, err
	}
	preview.Groups = maskPreviewGroups(upstream, preview.Groups)
	maskRemoteAPIKeySecretsForResponse(preview.APIKeys)
	return preview, nil
}

func (s *UpstreamService) ApplySyncPreview(ctx context.Context, id int64, token string) (*UpstreamSyncResult, error) {
	preview, err := s.repo.GetSyncPreview(ctx, id, strings.TrimSpace(token))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if preview.Status != "pending" || now.After(preview.ExpiresAt) {
		return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "preview_token", "reason": "preview is not pending or has expired"})
	}
	finished := now
	run := &UpstreamSyncRun{
		UpstreamID:   id,
		Status:       "success",
		GroupsCount:  len(preview.Groups),
		APIKeysCount: len(preview.APIKeys),
		Message:      "sync preview applied",
		StartedAt:    preview.CreatedAt,
		FinishedAt:   &finished,
		RawResult: map[string]any{
			"preview_token": preview.PreviewToken,
			"diff":          preview.Diff,
		},
	}
	if err := s.repo.ReplaceRemoteResources(ctx, id, preview.Groups, preview.APIKeys, run); err != nil {
		return nil, err
	}
	if err := s.repo.MarkSyncPreviewApplied(ctx, id, preview.PreviewToken, finished); err != nil {
		return nil, err
	}
	if err := s.ensureRuntimeAccount(ctx, id); err != nil {
		return nil, err
	}
	if preview.Diff.CostMultiplierChangeCount > 0 {
		_ = s.repo.UpsertAlert(ctx, UpstreamAlert{
			UpstreamID: &id,
			AlertType:  "sync_rate_multiplier_changed",
			Severity:   "info",
			Title:      "Upstream group multiplier changed",
			Message:    fmt.Sprintf("%d group multiplier change(s) applied.", preview.Diff.CostMultiplierChangeCount),
			Evidence:   map[string]any{"diff": preview.Diff},
		})
	}
	maskRemoteAPIKeySecretsForResponse(preview.APIKeys)
	return &UpstreamSyncResult{Run: run, Groups: preview.Groups, APIKeys: preview.APIKeys}, nil
}

func (s *UpstreamService) fetchRemoteResources(ctx context.Context, id int64) (*Upstream, []*UpstreamRemoteGroup, []*UpstreamRemoteAPIKey, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := s.decryptSecretsForUse(upstream); err != nil {
		return nil, nil, nil, err
	}
	session, err := s.adapter.Login(ctx, upstream)
	if err != nil {
		if upstream.AdminAuth != nil {
			upstream.AdminAuth.LastLoginError = err.Error()
			_ = s.persistPlainAdminAuth(ctx, upstream.AdminAuth)
		}
		return nil, nil, nil, ErrUpstreamLoginFailed.WithCause(err)
	}
	if upstream.AdminAuth != nil {
		now := time.Now().UTC()
		upstream.AdminAuth.LastLoginAt = &now
		upstream.AdminAuth.LastLoginError = ""
		if strings.TrimSpace(session.AccessToken) != "" {
			upstream.AdminAuth.AccessToken = session.AccessToken
		}
		if strings.TrimSpace(session.RefreshToken) != "" {
			upstream.AdminAuth.RefreshToken = session.RefreshToken
		}
		if session.TokenExpiresAt != nil {
			upstream.AdminAuth.TokenExpiresAt = session.TokenExpiresAt
		}
		if strings.TrimSpace(session.UserID) != "" {
			if upstream.AdminAuth.Metadata == nil {
				upstream.AdminAuth.Metadata = map[string]any{}
			}
			upstream.AdminAuth.Metadata["user_id"] = strings.TrimSpace(session.UserID)
		}
		if err := s.persistPlainAdminAuth(ctx, upstream.AdminAuth); err != nil {
			return nil, nil, nil, err
		}
	}
	groups, err := s.adapter.ListGroups(ctx, upstream, session)
	if err != nil {
		return nil, nil, nil, ErrUpstreamSyncFailed.WithCause(err)
	}
	keys, keyErr := s.adapter.ListAPIKeys(ctx, upstream, session)
	if keyErr != nil {
		keys = nil
	}
	if err := s.prepareRemoteResources(id, groups, keys); err != nil {
		return nil, nil, nil, err
	}
	return upstream, groups, keys, nil
}

func (s *UpstreamService) prepareRemoteResources(id int64, groups []*UpstreamRemoteGroup, keys []*UpstreamRemoteAPIKey) error {
	now := time.Now().UTC()
	for _, group := range groups {
		group.UpstreamID = id
		if strings.TrimSpace(group.Status) == "" {
			group.Status = UpstreamStatusActive
		}
		if group.RateMultiplier <= 0 {
			group.RateMultiplier = 1
		}
		if group.LastSyncedAt.IsZero() {
			group.LastSyncedAt = now
		}
	}
	for _, key := range keys {
		key.UpstreamID = id
		if strings.TrimSpace(key.Status) == "" {
			key.Status = UpstreamStatusActive
		}
		if strings.TrimSpace(key.SyncedRemoteGroupID) == "" {
			key.SyncedRemoteGroupID = strings.TrimSpace(key.RemoteGroupID)
		}
		if strings.TrimSpace(key.RemoteGroupID) == "" {
			key.RemoteGroupID = strings.TrimSpace(key.SyncedRemoteGroupID)
		}
		if strings.TrimSpace(key.APIKey) != "" {
			encrypted, err := s.encryptSecret(key.APIKey)
			if err != nil {
				return err
			}
			key.APIKey = encrypted
			key.APIKeyConfigured = true
		}
		if key.LastSyncedAt.IsZero() {
			key.LastSyncedAt = now
		}
	}
	return nil
}

func BuildUpstreamSyncDiff(currentGroups []*UpstreamRemoteGroup, currentKeys []*UpstreamRemoteAPIKey, nextGroups []*UpstreamRemoteGroup, nextKeys []*UpstreamRemoteAPIKey) UpstreamSyncDiff {
	diff := UpstreamSyncDiff{}
	curGroups := map[string]*UpstreamRemoteGroup{}
	nextGroupMap := map[string]*UpstreamRemoteGroup{}
	for _, group := range currentGroups {
		if group != nil && strings.TrimSpace(group.RemoteGroupID) != "" {
			curGroups[group.RemoteGroupID] = group
		}
	}
	for _, group := range nextGroups {
		if group != nil && strings.TrimSpace(group.RemoteGroupID) != "" {
			nextGroupMap[group.RemoteGroupID] = group
		}
	}
	for id, group := range nextGroupMap {
		if curGroups[id] == nil {
			diff.AddedGroups = append(diff.AddedGroups, syncDiffItem("group", id, group.RemoteGroupName, nil, groupMap(group), "new upstream group", nil))
			continue
		}
		before := curGroups[id]
		changed := map[string]any{}
		beforeMap := map[string]any{}
		if before.RemoteGroupName != group.RemoteGroupName {
			beforeMap["name"] = before.RemoteGroupName
			changed["name"] = group.RemoteGroupName
		}
		if before.RateMultiplier != group.RateMultiplier {
			beforeMap["rate_multiplier"] = before.RateMultiplier
			changed["rate_multiplier"] = group.RateMultiplier
			diff.CostMultiplierChangeCount++
		}
		if len(changed) > 0 {
			diff.ChangedGroups = append(diff.ChangedGroups, syncDiffItem("group", id, group.RemoteGroupName, beforeMap, changed, "group metadata changed", nil))
		}
	}
	for id, group := range curGroups {
		if nextGroupMap[id] == nil {
			diff.RemovedGroups = append(diff.RemovedGroups, syncDiffItem("group", id, group.RemoteGroupName, groupMap(group), nil, "group missing from upstream", nil))
		}
	}

	if nextKeys != nil {
		curKeys := map[string]*UpstreamRemoteAPIKey{}
		nextKeyMap := map[string]*UpstreamRemoteAPIKey{}
		for _, key := range currentKeys {
			if key != nil && strings.TrimSpace(key.RemoteAPIKeyID) != "" {
				curKeys[key.RemoteAPIKeyID] = key
			}
		}
		for _, key := range nextKeys {
			if key != nil && strings.TrimSpace(key.RemoteAPIKeyID) != "" {
				nextKeyMap[key.RemoteAPIKeyID] = key
			}
		}
		affected := map[int64]struct{}{}
		for id, key := range nextKeyMap {
			before := curKeys[id]
			if before == nil {
				diff.AddedAPIKeys = append(diff.AddedAPIKeys, syncDiffItem("api_key", id, key.RemoteAPIKeyName, nil, apiKeyMap(key), "new upstream API key", key.LocalGroupIDs))
				continue
			}
			beforeMap := map[string]any{}
			changed := map[string]any{}
			if before.RemoteAPIKeyName != key.RemoteAPIKeyName {
				beforeMap["name"] = before.RemoteAPIKeyName
				changed["name"] = key.RemoteAPIKeyName
			}
			if before.SyncedRemoteGroupID != key.SyncedRemoteGroupID {
				beforeMap["synced_remote_group_id"] = before.SyncedRemoteGroupID
				changed["synced_remote_group_id"] = key.SyncedRemoteGroupID
				for _, gid := range before.LocalGroupIDs {
					affected[gid] = struct{}{}
				}
			}
			if before.Status != key.Status {
				beforeMap["status"] = before.Status
				changed["status"] = key.Status
			}
			if len(changed) > 0 {
				diff.ChangedAPIKeys = append(diff.ChangedAPIKeys, syncDiffItem("api_key", id, key.RemoteAPIKeyName, beforeMap, changed, "API key metadata changed", before.LocalGroupIDs))
			}
		}
		for id, key := range curKeys {
			if nextKeyMap[id] == nil {
				diff.RemovedAPIKeys = append(diff.RemovedAPIKeys, syncDiffItem("api_key", id, key.RemoteAPIKeyName, apiKeyMap(key), nil, "API key missing from upstream", key.LocalGroupIDs))
				diff.UnschedulableAPIKeyIDs = append(diff.UnschedulableAPIKeyIDs, id)
				for _, gid := range key.LocalGroupIDs {
					affected[gid] = struct{}{}
				}
			}
		}
		for _, key := range nextKeys {
			if key == nil {
				continue
			}
			if strings.TrimSpace(key.RemoteGroupID) == "" || nextGroupMap[strings.TrimSpace(key.RemoteGroupID)] == nil || len(uniquePositiveInt64sLocal(key.LocalGroupIDs)) == 0 {
				diff.UnschedulableAPIKeyIDs = append(diff.UnschedulableAPIKeyIDs, strings.TrimSpace(key.RemoteAPIKeyID))
			}
		}
		for id := range affected {
			if id > 0 {
				diff.AffectedLocalGroupIDs = append(diff.AffectedLocalGroupIDs, id)
			}
		}
		sort.Slice(diff.AffectedLocalGroupIDs, func(i, j int) bool { return diff.AffectedLocalGroupIDs[i] < diff.AffectedLocalGroupIDs[j] })
		diff.UnschedulableAPIKeyIDs = uniqueStringsService(diff.UnschedulableAPIKeyIDs)
	}
	return diff
}

func (s *UpstreamService) UpdateGovernancePolicy(ctx context.Context, id int64, policy UpstreamGovernancePolicy) (*UpstreamGovernancePolicy, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	normalized := normalizeGovernancePolicy(policy)
	if upstream.Metadata == nil {
		upstream.Metadata = map[string]any{}
	}
	upstream.Metadata["governance_policy"] = governancePolicyMap(normalized)
	params := UpstreamUpdateParams{Metadata: &upstream.Metadata}
	if _, err := s.Update(ctx, id, params); err != nil {
		return nil, err
	}
	return &normalized, nil
}

func (s *UpstreamService) GovernancePolicy(ctx context.Context, id int64) (*UpstreamGovernancePolicy, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	policy := governancePolicyFromMetadata(upstream.Metadata)
	return &policy, nil
}

func (s *UpstreamService) ListAlerts(ctx context.Context, id int64, activeOnly bool) ([]UpstreamAlert, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListAlerts(ctx, id, activeOnly)
}

func (s *UpstreamService) ResolveAlert(ctx context.Context, id int64, alertType string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.ResolveAlert(ctx, id, alertType)
}

func (s *UpstreamService) CostReport(ctx context.Context, id int64, start, end time.Time, dimension string) (*UpstreamCostReport, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	resetAt := upstreamCostReportResetAt(upstream)
	if resetAt != nil && end.After(*resetAt) && start.Before(*resetAt) {
		start = *resetAt
	}
	report, err := s.repo.GetCostReport(ctx, id, start, end, dimension)
	if err != nil {
		return nil, err
	}
	report.ResetAt = resetAt
	if resetAt != nil && report.End.After(*resetAt) && report.Start.Before(*resetAt) {
		report.Start = *resetAt
	}
	return report, nil
}

func (s *UpstreamService) ResetCostReport(ctx context.Context, id int64) (*UpstreamCostReportResetResult, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	resetAt := nextUpstreamCostReportResetAt(timezone.Now())
	metadata := copyAnyMap(upstream.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["cost_report_reset_at"] = resetAt.UTC().Format(time.RFC3339)
	upstream.Metadata = metadata
	if err := s.repo.Update(ctx, upstream); err != nil {
		return nil, err
	}
	return &UpstreamCostReportResetResult{
		UpstreamID: id,
		ResetAt:    resetAt.UTC(),
	}, nil
}

func nextUpstreamCostReportResetAt(now time.Time) time.Time {
	if now.IsZero() {
		now = timezone.Now()
	}
	return timezone.StartOfDay(now).AddDate(0, 0, 1)
}

func upstreamCostReportResetAt(upstream *Upstream) *time.Time {
	if upstream == nil || upstream.Metadata == nil {
		return nil
	}
	raw := strings.TrimSpace(anyToString(upstream.Metadata["cost_report_reset_at"]))
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func normalizeGovernancePolicy(policy UpstreamGovernancePolicy) UpstreamGovernancePolicy {
	if policy.ConsecutiveFailuresToCircuitOpen <= 0 {
		policy.ConsecutiveFailuresToCircuitOpen = 5
	}
	if policy.FirstTokenDegradeThresholdMS <= 0 {
		policy.FirstTokenDegradeThresholdMS = 8000
	}
	if policy.ErrorRateDegradeThreshold <= 0 || policy.ErrorRateDegradeThreshold > 1 {
		policy.ErrorRateDegradeThreshold = 0.25
	}
	if policy.RecoveryProbeIntervalSeconds <= 0 {
		policy.RecoveryProbeIntervalSeconds = 60
	}
	if policy.RecoverySuccessesRequired <= 0 {
		policy.RecoverySuccessesRequired = 3
	}
	if policy.ProbeFailureWeight <= 0 {
		policy.ProbeFailureWeight = 1
	}
	if policy.RuntimeFailureWeight <= 0 {
		policy.RuntimeFailureWeight = 2
	}
	policy.IgnoredStatusCodes = uniqueInts(policy.IgnoredStatusCodes)
	policy.ImmediateCircuitStatusCodes = uniqueInts(policy.ImmediateCircuitStatusCodes)
	return policy
}

func governancePolicyFromMetadata(metadata map[string]any) UpstreamGovernancePolicy {
	var policy UpstreamGovernancePolicy
	if raw, ok := metadata["governance_policy"].(map[string]any); ok {
		policy.ConsecutiveFailuresToCircuitOpen = int(parseAnyInt64(raw["consecutive_failures_to_circuit_open"]))
		policy.FirstTokenDegradeThresholdMS = int(parseAnyInt64(raw["first_token_degrade_threshold_ms"]))
		policy.ErrorRateDegradeThreshold = parseFloatLike(raw["error_rate_degrade_threshold"])
		policy.RecoveryProbeIntervalSeconds = int(parseAnyInt64(raw["recovery_probe_interval_seconds"]))
		policy.RecoverySuccessesRequired = int(parseAnyInt64(raw["recovery_successes_required"]))
		policy.IgnoredStatusCodes = parseIntSlice(raw["ignored_status_codes"])
		policy.ImmediateCircuitStatusCodes = parseIntSlice(raw["immediate_circuit_status_codes"])
		policy.ProbeFailureWeight = parseFloatLike(raw["probe_failure_weight"])
		policy.RuntimeFailureWeight = parseFloatLike(raw["runtime_failure_weight"])
		if v, ok := raw["alert_enabled"].(bool); ok {
			policy.AlertEnabled = v
		}
	}
	if !policy.AlertEnabled {
		policy.AlertEnabled = true
	}
	return normalizeGovernancePolicy(policy)
}

func governancePolicyMap(policy UpstreamGovernancePolicy) map[string]any {
	return map[string]any{
		"consecutive_failures_to_circuit_open": policy.ConsecutiveFailuresToCircuitOpen,
		"first_token_degrade_threshold_ms":     policy.FirstTokenDegradeThresholdMS,
		"error_rate_degrade_threshold":         policy.ErrorRateDegradeThreshold,
		"recovery_probe_interval_seconds":      policy.RecoveryProbeIntervalSeconds,
		"recovery_successes_required":          policy.RecoverySuccessesRequired,
		"ignored_status_codes":                 policy.IgnoredStatusCodes,
		"immediate_circuit_status_codes":       policy.ImmediateCircuitStatusCodes,
		"probe_failure_weight":                 policy.ProbeFailureWeight,
		"runtime_failure_weight":               policy.RuntimeFailureWeight,
		"alert_enabled":                        policy.AlertEnabled,
	}
}

func parseIntSlice(raw any) []int {
	switch v := raw.(type) {
	case []int:
		return uniqueInts(v)
	case []any:
		out := make([]int, 0, len(v))
		for _, item := range v {
			n := int(parseAnyInt64(item))
			if n > 0 {
				out = append(out, n)
			}
		}
		return uniqueInts(out)
	case string:
		parts := strings.Split(v, ",")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			n, _ := strconv.Atoi(strings.TrimSpace(part))
			if n > 0 {
				out = append(out, n)
			}
		}
		return uniqueInts(out)
	default:
		n := int(parseAnyInt64(raw))
		if n > 0 {
			return []int{n}
		}
	}
	return nil
}

func uniqueInts(items []int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(items))
	for _, item := range items {
		if item <= 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Ints(out)
	return out
}

func groupMap(group *UpstreamRemoteGroup) map[string]any {
	if group == nil {
		return nil
	}
	return map[string]any{
		"name":            group.RemoteGroupName,
		"rate_multiplier": group.RateMultiplier,
		"status":          group.Status,
	}
}

func apiKeyMap(key *UpstreamRemoteAPIKey) map[string]any {
	if key == nil {
		return nil
	}
	return map[string]any{
		"name":                   key.RemoteAPIKeyName,
		"status":                 key.Status,
		"synced_remote_group_id": key.SyncedRemoteGroupID,
		"remote_group_id":        key.RemoteGroupID,
		"local_group_ids":        key.LocalGroupIDs,
	}
}

func syncDiffItem(kind, id, name string, before, after map[string]any, impact string, localGroups []int64) UpstreamSyncDiffItem {
	return UpstreamSyncDiffItem{
		Kind:        kind,
		ID:          strings.TrimSpace(id),
		Name:        strings.TrimSpace(name),
		Before:      before,
		After:       after,
		Impact:      impact,
		LocalGroups: uniquePositiveInt64sLocal(localGroups),
	}
}

func newPreviewToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

func uniqueStringsService(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func maskPreviewGroups(_ *Upstream, groups []*UpstreamRemoteGroup) []*UpstreamRemoteGroup {
	return groups
}
