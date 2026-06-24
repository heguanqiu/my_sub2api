package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
)

const (
	upstreamRuntimeManagedExtraKey = "upstream_runtime_managed"
	upstreamRuntimeAccountVersion  = 1
)

func (s *UpstreamService) ensureRuntimeAccount(ctx context.Context, upstreamID int64) error {
	if s == nil || s.accountRepo == nil {
		return nil
	}
	upstream, err := s.repo.Get(ctx, upstreamID)
	if err != nil {
		return err
	}
	if err := s.decryptSecretsForUse(upstream); err != nil {
		return err
	}
	if err := s.decryptRemoteAPIKeySecretsForUse(upstream); err != nil {
		return err
	}
	return s.syncRuntimeAccount(ctx, upstream)
}

func (s *UpstreamService) syncRuntimeAccount(ctx context.Context, upstream *Upstream) error {
	if upstream == nil || s.accountRepo == nil {
		return nil
	}
	accounts, err := s.findRuntimeAccounts(ctx, upstream.ID)
	if err != nil {
		return err
	}

	existingByRemoteKeyID := make(map[string]*Account, len(accounts))
	for i := range accounts {
		account := accounts[i]
		remoteAPIKeyID := accountUpstreamRemoteAPIKeyID(&account)
		if remoteAPIKeyID == "" {
			continue
		}
		existingByRemoteKeyID[remoteAPIKeyID] = &accounts[i]
	}

	keptAccountIDs := make(map[int64]bool)
	for _, remoteKey := range upstream.RemoteAPIKeys {
		if remoteKey == nil || strings.TrimSpace(remoteKey.RemoteAPIKeyID) == "" {
			continue
		}
		remoteKey.LocalGroupIDs = uniquePositiveInt64sLocal(remoteKey.LocalGroupIDs)
		groupIDs, err := s.validateRuntimeLocalGroupIDs(ctx, remoteKey.LocalGroupIDs)
		if err != nil {
			return err
		}
		existing := existingByRemoteKeyID[strings.TrimSpace(remoteKey.RemoteAPIKeyID)]
		if existing == nil && !remoteAPIKeyHasRuntimeConfig(remoteKey) {
			continue
		}

		built := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, existing)
		if existing == nil {
			if err := s.accountRepo.Create(ctx, built); err != nil {
				return fmt.Errorf("create upstream runtime account: %w", err)
			}
		} else {
			built.ID = existing.ID
			if err := s.accountRepo.Update(ctx, built); err != nil {
				return fmt.Errorf("update upstream runtime account: %w", err)
			}
		}
		if err := s.accountRepo.BindGroups(ctx, built.ID, groupIDs); err != nil {
			return fmt.Errorf("bind upstream runtime account groups: %w", err)
		}
		keptAccountIDs[built.ID] = true
	}

	for i := range accounts {
		account := accounts[i]
		if keptAccountIDs[account.ID] {
			continue
		}
		if err := s.disableRuntimeAccountRecord(ctx, &account, "upstream runtime account no longer mapped to a synced API key"); err != nil {
			return err
		}
	}
	return nil
}

func (s *UpstreamService) findRuntimeAccount(ctx context.Context, upstreamID int64) (*Account, error) {
	accounts, err := s.findRuntimeAccounts(ctx, upstreamID)
	if err != nil || len(accounts) == 0 {
		return nil, err
	}
	return &accounts[0], nil
}

func (s *UpstreamService) findRuntimeAccounts(ctx context.Context, upstreamID int64) ([]Account, error) {
	if upstreamID <= 0 || s.accountRepo == nil {
		return nil, nil
	}
	accounts, err := s.accountRepo.FindByExtraField(ctx, "upstream_id", upstreamID)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(accounts))
	for i := range accounts {
		acc := accounts[i]
		if acc.Extra == nil || !extraBool(acc.Extra[upstreamRuntimeManagedExtraKey]) {
			continue
		}
		out = append(out, acc)
	}
	return out, nil
}

func (s *UpstreamService) disableRuntimeAccountRecord(ctx context.Context, account *Account, reason string) error {
	if s == nil || s.accountRepo == nil || account == nil {
		return nil
	}
	account.Schedulable = false
	account.Status = StatusError
	account.ErrorMessage = strings.TrimSpace(reason)
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	account.Extra["upstream_status"] = UpstreamStatusDisabled
	return s.accountRepo.Update(ctx, account)
}

func remoteAPIKeyHasRuntimeConfig(remoteKey *UpstreamRemoteAPIKey) bool {
	return remoteKey != nil && len(uniquePositiveInt64sLocal(remoteKey.LocalGroupIDs)) > 0
}

func accountUpstreamRemoteAPIKeyID(account *Account) string {
	if account == nil || account.Extra == nil {
		return ""
	}
	return strings.TrimSpace(anyToString(account.Extra["upstream_remote_api_key_id"]))
}

func buildRuntimeAccountFromUpstream(upstream *Upstream, existing *Account) *Account {
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:      "default",
		RemoteAPIKeyName:    "default",
		Status:              UpstreamStatusActive,
		RemoteGroupID:       firstLegacyRuntimeRemoteGroupID(upstream),
		SyncedRemoteGroupID: firstLegacyRuntimeRemoteGroupID(upstream),
		LocalGroupIDs:       append([]int64(nil), upstream.LocalGroupIDs...),
	}
	if upstream != nil && upstream.ForwardCredential != nil {
		remoteKey.APIKey = upstream.ForwardCredential.APIKey
		remoteKey.APIKeyConfigured = strings.TrimSpace(remoteKey.APIKey) != ""
		remoteKey.MaskedKey = maskSecret(remoteKey.APIKey)
	}
	return buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, existing)
}

func firstLegacyRuntimeRemoteGroupID(upstream *Upstream) string {
	if upstream == nil || upstream.Metadata == nil {
		return ""
	}
	raw, ok := upstream.Metadata["local_group_remote_group_ids"].(map[string]any)
	if !ok {
		return ""
	}
	for _, localGroupID := range uniquePositiveInt64sLocal(upstream.LocalGroupIDs) {
		if v := strings.TrimSpace(anyToString(raw[strconv.FormatInt(localGroupID, 10)])); v != "" {
			return v
		}
	}
	return ""
}

func buildRuntimeAccountFromUpstreamAPIKey(upstream *Upstream, remoteKey *UpstreamRemoteAPIKey, existing *Account) *Account {
	attachUpstreamMetadata(upstream)
	account := &Account{}
	if existing != nil {
		*account = *existing
		account.Credentials = copyAnyMap(existing.Credentials)
		account.Extra = copyAnyMap(existing.Extra)
	}
	keyName := strings.TrimSpace(remoteKey.RemoteAPIKeyName)
	if keyName == "" {
		keyName = strings.TrimSpace(remoteKey.RemoteAPIKeyID)
	}
	account.Name = fmt.Sprintf("[Upstream] %s / %s", upstream.Name, keyName)
	account.Platform = PlatformOpenAI
	account.Type = AccountTypeAPIKey
	account.Status = StatusActive
	account.Schedulable = upstreamRemoteAPIKeyRuntimeSchedulable(upstream, remoteKey)
	account.ErrorMessage = upstreamRemoteAPIKeyRuntimeErrorMessage(upstream, remoteKey)
	if !account.Schedulable && account.ErrorMessage != "" {
		account.Status = StatusError
	}
	account.AutoPauseOnExpired = true
	account.Concurrency = upstreamRuntimeConcurrency(upstream)
	account.Priority = upstreamRuntimePriority(upstream)
	rateMultiplier := upstreamRuntimeRateMultiplier(upstream)
	account.RateMultiplier = &rateMultiplier
	loadFactor := upstreamRuntimeLoadFactor(upstream)
	account.LoadFactor = &loadFactor
	account.ExpiresAt = upstreamRuntimeExpiresAt(upstream)

	account.Credentials = upstreamRuntimeCredentials(upstream, remoteKey, account.Credentials)
	account.Extra = upstreamRuntimeExtra(upstream, remoteKey, account.Extra)
	return account
}

func upstreamRuntimeCredentials(upstream *Upstream, remoteKey *UpstreamRemoteAPIKey, existing map[string]any) map[string]any {
	out := copyAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	out["api_key"] = strings.TrimSpace(remoteKey.APIKey)
	out["base_url"] = upstreamRuntimeOpenAIAPIBaseURL(upstream)
	out["pool_mode"] = true
	out["pool_mode_retry_count"] = upstreamRuntimeRetryCount(upstream)
	out["openai_capabilities"] = []string{
		string(OpenAIEndpointCapabilityChatCompletions),
		string(OpenAIEndpointCapabilityEmbeddings),
	}
	if upstream.ForwardCredential != nil {
		if upstream.ForwardCredential.Metadata != nil {
			copyKnownRuntimeCredentialMetadata(out, upstream.ForwardCredential.Metadata)
		}
	}
	out["upstream_remote_group_id"] = strings.TrimSpace(remoteKey.RemoteGroupID)
	out["upstream_group_mapping"] = upstreamRuntimeGroupMapping(remoteKey)
	out["upstream_group_rate_multipliers"] = upstreamRuntimeGroupRateMultipliers(upstream, remoteKey)
	out["upstream_remote_group_rate_multiplier"] = upstreamRemoteGroupRateMultiplier(upstream, remoteKey.RemoteGroupID)
	if mapping := upstreamRuntimeModelMapping(upstream); len(mapping) > 0 {
		out["model_mapping"] = mapping
	} else if _, ok := out["model_mapping"]; !ok {
		out["model_mapping"] = map[string]any{}
	}
	return out
}

func upstreamRuntimeOpenAIAPIBaseURL(upstream *Upstream) string {
	if upstream == nil {
		return ""
	}
	if upstream.Metadata != nil {
		for _, key := range []string{"openai_api_base_url", "api_base_url", "forward_base_url"} {
			if v := strings.TrimRight(strings.TrimSpace(anyToString(upstream.Metadata[key])), "/"); v != "" {
				return v
			}
		}
	}
	return inferOpenAIAPIBaseURLFromUpstreamBase(upstream.BaseURL)
}

func inferOpenAIAPIBaseURLFromUpstreamBase(raw string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	if openAIBaseURLHasVersionSuffix(trimmed) {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/responses") ||
		strings.HasSuffix(trimmed, "/chat/completions") ||
		strings.HasSuffix(trimmed, "/embeddings") ||
		strings.HasSuffix(trimmed, "/models") {
		return trimmed
	}
	return trimmed + "/v1"
}

func copyKnownRuntimeCredentialMetadata(out, metadata map[string]any) {
	for _, key := range []string{
		"model_mapping",
		"compact_model_mapping",
		"openai_capabilities",
		"openai_image_capabilities",
		"custom_error_codes_enabled",
		"custom_error_codes",
		"pool_mode_retry_status_codes",
		"websearch_enabled",
		"websearch_context_size",
	} {
		if value, ok := metadata[key]; ok {
			out[key] = value
		}
	}
}

func upstreamRuntimeExtra(upstream *Upstream, remoteKey *UpstreamRemoteAPIKey, existing map[string]any) map[string]any {
	out := copyAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	out[upstreamRuntimeManagedExtraKey] = true
	out["upstream_runtime_version"] = upstreamRuntimeAccountVersion
	out["upstream_id"] = upstream.ID
	out["upstream_type"] = upstream.Type
	out["upstream_name"] = upstream.Name
	out["upstream_status"] = upstream.Status
	out["upstream_routing_mode"] = upstream.RoutingMode
	out["upstream_weight"] = upstream.Weight
	out["upstream_cost_multiplier"] = upstream.CostMultiplier
	out["upstream_health_score"] = defaultScore(upstream.LatestHealthScore)
	out["upstream_base_priority"] = upstream.Priority
	out["upstream_base_load_factor"] = upstreamRuntimeConcurrency(upstream)
	out["upstream_remote_api_key_id"] = strings.TrimSpace(remoteKey.RemoteAPIKeyID)
	out["upstream_remote_api_key_name"] = strings.TrimSpace(remoteKey.RemoteAPIKeyName)
	out["upstream_remote_api_key_masked"] = strings.TrimSpace(remoteKey.MaskedKey)
	out["upstream_remote_api_key_status"] = strings.TrimSpace(remoteKey.Status)
	out["upstream_synced_remote_group_id"] = strings.TrimSpace(remoteKey.SyncedRemoteGroupID)
	out["upstream_remote_group_id"] = strings.TrimSpace(remoteKey.RemoteGroupID)
	out["upstream_local_group_ids"] = int64SliceToAny(remoteKey.LocalGroupIDs)
	out["hidden"] = true
	out["managed_by"] = "upstream_management"
	out["openai_passthrough"] = true
	out[openai_compat.ExtraKeyResponsesMode] = string(openai_compat.ResponsesSupportModeForceResponses)
	out[openai_compat.ExtraKeyResponsesSupported] = true
	out["openai_apikey_responses_websockets_v2_mode"] = "off"
	out["openai_compact_mode"] = OpenAICompactModeForceOn
	out["privacy_mode"] = PrivacyModeTrainingOff
	return out
}

func upstreamRuntimeGroupMapping(remoteKey *UpstreamRemoteAPIKey) map[string]any {
	mapping := map[string]any{}
	if remoteKey == nil {
		return mapping
	}
	remoteGroupID := strings.TrimSpace(remoteKey.RemoteGroupID)
	if remoteGroupID == "" {
		return mapping
	}
	localIDs := uniquePositiveInt64sLocal(remoteKey.LocalGroupIDs)
	for _, id := range localIDs {
		mapping[strconv.FormatInt(id, 10)] = remoteGroupID
	}
	return mapping
}

func upstreamRuntimeGroupRateMultipliers(upstream *Upstream, remoteKey *UpstreamRemoteAPIKey) map[string]any {
	rates := map[string]any{}
	if upstream == nil || remoteKey == nil {
		return rates
	}
	rate := upstreamRemoteGroupRateMultiplier(upstream, remoteKey.RemoteGroupID)
	for _, localID := range uniquePositiveInt64sLocal(remoteKey.LocalGroupIDs) {
		rates[strconv.FormatInt(localID, 10)] = rate
	}
	return rates
}

func upstreamRuntimeModelMapping(upstream *Upstream) map[string]any {
	models := upstreamSupportedModels(upstream)
	if len(models) == 0 {
		return nil
	}
	mapping := make(map[string]any, len(models))
	for _, model := range models {
		mapping[model] = model
	}
	return mapping
}

func upstreamSupportedModels(upstream *Upstream) []string {
	if upstream == nil || upstream.Metadata == nil {
		return nil
	}
	return parseStringList(upstream.Metadata["supported_models"])
}

func upstreamRemoteGroupRateMultiplier(upstream *Upstream, remoteGroupID string) float64 {
	remoteGroupID = strings.TrimSpace(remoteGroupID)
	if upstream == nil || remoteGroupID == "" {
		return 1
	}
	for _, group := range upstream.RemoteGroups {
		if group == nil || strings.TrimSpace(group.RemoteGroupID) == "" {
			continue
		}
		if strings.TrimSpace(group.RemoteGroupID) != remoteGroupID {
			continue
		}
		if group.RateMultiplier > 0 {
			return group.RateMultiplier
		}
		return 1
	}
	return 1
}

func upstreamRemoteAPIKeyRuntimeSchedulable(upstream *Upstream, remoteKey *UpstreamRemoteAPIKey) bool {
	if upstream == nil || upstream.Weight <= 0 {
		return false
	}
	switch upstream.Status {
	case UpstreamStatusDisabled, UpstreamStatusCircuitOpen:
		return false
	}
	if remoteKey == nil {
		return false
	}
	if !remoteAPIKeyStatusActive(remoteKey.Status) {
		return false
	}
	if strings.TrimSpace(remoteKey.APIKey) == "" {
		return false
	}
	if len(uniquePositiveInt64sLocal(remoteKey.LocalGroupIDs)) == 0 {
		return false
	}
	return true
}

func upstreamRemoteAPIKeyRuntimeErrorMessage(upstream *Upstream, remoteKey *UpstreamRemoteAPIKey) string {
	if upstream == nil {
		return "upstream not configured"
	}
	switch upstream.Status {
	case UpstreamStatusDisabled:
		return "upstream disabled"
	case UpstreamStatusCircuitOpen:
		return "upstream circuit open"
	}
	if upstream.Weight <= 0 {
		return "upstream weight is zero"
	}
	if remoteKey == nil {
		return "upstream API key missing"
	}
	if !remoteAPIKeyStatusActive(remoteKey.Status) {
		return "upstream API key inactive"
	}
	if strings.TrimSpace(remoteKey.APIKey) == "" {
		return "upstream API key secret missing"
	}
	if len(uniquePositiveInt64sLocal(remoteKey.LocalGroupIDs)) == 0 {
		return "upstream local groups missing"
	}
	return ""
}

func remoteAPIKeyStatusActive(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "", "active", "enabled", "enable", "1", "true":
		return true
	default:
		return false
	}
}

func upstreamRuntimeConcurrency(upstream *Upstream) int {
	if upstream == nil {
		return 1
	}
	if n := metadataInt(upstream.Metadata, "concurrency"); n > 0 {
		return n
	}
	if n := metadataInt(upstream.Metadata, "max_concurrency"); n > 0 {
		return n
	}
	if upstream.Weight > 0 {
		return clampInt(int(math.Round(float64(upstream.Weight)/10)), 1, 128)
	}
	return 1
}

func upstreamRuntimeLoadFactor(upstream *Upstream) int {
	base := upstreamRuntimeConcurrency(upstream)
	health := clampUpstreamScore01(defaultScore(upstream.LatestHealthScore))
	if health <= 0 {
		return 1
	}
	return clampInt(int(math.Round(float64(base)*health)), 1, base)
}

func upstreamRuntimePriority(upstream *Upstream) int {
	if upstream == nil {
		return 100
	}
	priority := upstream.Priority
	if priority <= 0 {
		priority = 100
	}
	health := clampUpstreamScore01(defaultScore(upstream.LatestHealthScore))
	if health < 0.9 {
		priority += int(math.Round((0.9 - health) * 100))
	}
	switch upstream.Status {
	case UpstreamStatusDegraded:
		priority += 25
	case UpstreamStatusHalfOpen:
		priority += 50
	}
	return clampInt(priority, 1, 10000)
}

func upstreamRuntimeRateMultiplier(upstream *Upstream) float64 {
	if upstream == nil || upstream.CostMultiplier < 0 {
		return 1
	}
	return upstream.CostMultiplier
}

func upstreamRuntimeRetryCount(upstream *Upstream) int {
	if upstream == nil {
		return defaultPoolModeRetryCount
	}
	if upstream.RetryMax <= 0 {
		return defaultPoolModeRetryCount
	}
	return clampInt(upstream.RetryMax, 0, maxPoolModeRetryCount)
}

func upstreamRuntimeExpiresAt(upstream *Upstream) *time.Time {
	return nil
}

func (s *UpstreamService) validateRuntimeLocalGroupIDs(ctx context.Context, groupIDs []int64) ([]int64, error) {
	groupIDs = uniquePositiveInt64sLocal(groupIDs)
	if len(groupIDs) == 0 || s.groupRepo == nil {
		return groupIDs, nil
	}
	out := make([]int64, 0, len(groupIDs))
	for _, id := range groupIDs {
		group, err := s.groupRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if group.Platform != PlatformOpenAI {
			return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "local_group_ids", "reason": "group platform must be openai"})
		}
		if group.RequireOAuthOnly {
			return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "local_group_ids", "reason": "group requires oauth only"})
		}
		out = append(out, id)
	}
	return out, nil
}

func normalizeUpstreamMetadata(upstream *Upstream) {
	if upstream.Metadata == nil {
		upstream.Metadata = map[string]any{}
	}
	if upstream.LocalGroupIDs == nil {
		upstream.LocalGroupIDs = parseUpstreamInt64Slice(upstream.Metadata["local_group_ids"])
	} else {
		upstream.LocalGroupIDs = uniquePositiveInt64sLocal(upstream.LocalGroupIDs)
	}
	upstream.Metadata["local_group_ids"] = int64SliceToAny(upstream.LocalGroupIDs)
	upstream.Metadata["supported_models"] = stringSliceToAny(parseStringList(upstream.Metadata["supported_models"]))
}

func attachUpstreamMetadata(upstream *Upstream) {
	if upstream == nil {
		return
	}
	normalizeUpstreamMetadata(upstream)
}

func copyAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func parseUpstreamInt64Slice(raw any) []int64 {
	switch v := raw.(type) {
	case []int64:
		return uniquePositiveInt64sLocal(v)
	case []int:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			out = append(out, int64(item))
		}
		return uniquePositiveInt64sLocal(out)
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			if parsed := parseAnyInt64(item); parsed > 0 {
				out = append(out, parsed)
			}
		}
		return uniquePositiveInt64sLocal(out)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		var arr []any
		if strings.HasPrefix(trimmed, "[") && json.Unmarshal([]byte(trimmed), &arr) == nil {
			return parseUpstreamInt64Slice(arr)
		}
		parts := strings.Split(trimmed, ",")
		out := make([]int64, 0, len(parts))
		for _, part := range parts {
			if parsed := parseAnyInt64(part); parsed > 0 {
				out = append(out, parsed)
			}
		}
		return uniquePositiveInt64sLocal(out)
	default:
		if parsed := parseAnyInt64(raw); parsed > 0 {
			return []int64{parsed}
		}
	}
	return nil
}

func parseStringList(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return uniqueNonEmptyStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(anyToString(item)); s != "" {
				out = append(out, s)
			}
		}
		return uniqueNonEmptyStrings(out)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		var arr []any
		if strings.HasPrefix(trimmed, "[") && json.Unmarshal([]byte(trimmed), &arr) == nil {
			return parseStringList(arr)
		}
		parts := strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r' || r == '\t'
		})
		return uniqueNonEmptyStrings(parts)
	default:
		if s := strings.TrimSpace(anyToString(raw)); s != "" {
			return []string{s}
		}
	}
	return nil
}

func uniqueNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func stringSliceToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range uniqueNonEmptyStrings(in) {
		out = append(out, item)
	}
	return out
}

func parseAnyInt64(raw any) int64 {
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

func int64SliceToAny(in []int64) []any {
	out := make([]any, 0, len(in))
	for _, id := range uniquePositiveInt64sLocal(in) {
		out = append(out, id)
	}
	return out
}

func uniquePositiveInt64sLocal(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func metadataInt(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	return int(parseAnyInt64(metadata[key]))
}

func extraBool(raw any) bool {
	v, ok := raw.(bool)
	return ok && v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
