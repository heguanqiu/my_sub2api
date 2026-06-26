package service

import "testing"

func TestAccountRateMultiplierForUsageUsesMappedRemoteGroupRate(t *testing.T) {
	base := 1.2
	groupID := int64(30)
	account := &Account{
		ID:             10,
		RateMultiplier: &base,
		Extra: map[string]any{
			upstreamRuntimeManagedExtraKey: true,
			"upstream_id":                  int64(9),
		},
		Credentials: map[string]any{
			"upstream_group_rate_multipliers": map[string]any{
				"30": 2.5,
			},
		},
	}

	got := openAIAccountRateMultiplierForUsage(account, &groupID)
	if got != 2.5 {
		t.Fatalf("multiplier = %v, want 2.5", got)
	}
}

func TestAccountRateMultiplierForUsageUsesSyncedUpstreamGroupRate(t *testing.T) {
	localGroupID := int64(36)
	upstream := &Upstream{
		ID:      9,
		Name:    "north",
		Type:    UpstreamTypeSub2API,
		BaseURL: "https://upstream.example.com",
		Status:  UpstreamStatusActive,
		Weight:  100,
		RemoteGroups: []*UpstreamRemoteGroup{
			{RemoteGroupID: "vip", RateMultiplier: 3.25},
		},
	}
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:   "key-1",
		RemoteAPIKeyName: "VIP key",
		APIKey:           "sk-forward",
		RemoteGroupID:    "vip",
		Status:           UpstreamStatusActive,
		LocalGroupIDs:    []int64{localGroupID},
	}
	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil, PlatformOpenAI)

	got := accountRateMultiplierForUsage(account, &localGroupID)
	if got != 3.25 {
		t.Fatalf("multiplier = %v, want 3.25", got)
	}
}

func TestAccountRateMultiplierForUsageFallsBackForNormalAccount(t *testing.T) {
	base := 1.2
	groupID := int64(30)
	account := &Account{
		ID:             10,
		RateMultiplier: &base,
		Credentials: map[string]any{
			"upstream_group_rate_multipliers": map[string]any{"30": 2.5},
		},
	}

	got := openAIAccountRateMultiplierForUsage(account, &groupID)
	if got != 1.2 {
		t.Fatalf("multiplier = %v, want 1.2", got)
	}
}

func TestAccountRateMultiplierForUsageManagedFallsBackToOneWithoutMappedRemoteRate(t *testing.T) {
	base := 1.2
	groupID := int64(30)
	account := &Account{
		ID:             10,
		RateMultiplier: &base,
		Extra: map[string]any{
			upstreamRuntimeManagedExtraKey: true,
			"upstream_id":                  int64(9),
		},
	}

	got := accountRateMultiplierForUsage(account, &groupID)
	if got != 1 {
		t.Fatalf("multiplier = %v, want 1", got)
	}
}
