package service

import "testing"

func TestOpenAIAccountRateMultiplierForUsageUsesMappedRemoteGroupRate(t *testing.T) {
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
	if got != 3.0 {
		t.Fatalf("multiplier = %v, want 3.0", got)
	}
}

func TestOpenAIAccountRateMultiplierForUsageFallsBackForNormalAccount(t *testing.T) {
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
