package service

import "testing"

func TestBuildForwardCredentialNormalizesAPIKeyAuthType(t *testing.T) {
	svc := &UpstreamService{}
	credential, err := svc.buildForwardCredential(7, nil, &UpstreamForwardCredentialInput{
		Name:     "default",
		AuthType: "api_key",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("buildForwardCredential returned error: %v", err)
	}
	if credential.AuthType != UpstreamForwardAuthOpenAI {
		t.Fatalf("auth type = %q, want %q", credential.AuthType, UpstreamForwardAuthOpenAI)
	}
}

func TestBuildForwardCredentialRejectsInvalidAuthType(t *testing.T) {
	svc := &UpstreamService{}
	_, err := svc.buildForwardCredential(7, nil, &UpstreamForwardCredentialInput{
		Name:     "default",
		AuthType: "basic",
		Enabled:  true,
	})
	if err == nil {
		t.Fatal("expected invalid auth type error")
	}
}

func TestBuildAdminAuthRejectsInvalidAuthMode(t *testing.T) {
	svc := &UpstreamService{}
	_, err := svc.buildAdminAuth(7, nil, &UpstreamAdminAuthInput{
		AuthMode: "basic",
	})
	if err == nil {
		t.Fatal("expected invalid admin auth mode error")
	}
}

func TestBuildRuntimeAccountFromUpstreamStoresLocalRemoteGroupMappingAndRates(t *testing.T) {
	upstream := &Upstream{
		ID:             9,
		Name:           "north",
		Type:           UpstreamTypeNewAPI,
		BaseURL:        "https://upstream.example.com/",
		Status:         UpstreamStatusActive,
		Priority:       100,
		Weight:         100,
		CostMultiplier: 1.1,
		RoutingMode:    UpstreamRoutingBalanced,
		RemoteGroups: []*UpstreamRemoteGroup{
			{RemoteGroupID: "vip", RateMultiplier: 2.5},
			{RemoteGroupID: "standard", RateMultiplier: 0.8},
			{RemoteGroupID: "ignored", RateMultiplier: 9},
		},
	}
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:   "5798",
		RemoteAPIKeyName: "pro",
		APIKey:           "sk-forward",
		RemoteGroupID:    "vip",
		Status:           "active",
		LocalGroupIDs:    []int64{10, 20},
	}

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil)
	mapping, ok := account.Credentials["upstream_group_mapping"].(map[string]any)
	if !ok {
		t.Fatalf("upstream_group_mapping type = %T", account.Credentials["upstream_group_mapping"])
	}
	if got := mapping["10"]; got != "vip" {
		t.Fatalf("mapping[10] = %v, want vip", got)
	}
	if got := mapping["20"]; got != "vip" {
		t.Fatalf("mapping[20] = %v, want vip", got)
	}
	if _, ok := mapping["30"]; ok {
		t.Fatal("mapping[30] should be ignored because local group is not bound")
	}

	rates, ok := account.Credentials["upstream_group_rate_multipliers"].(map[string]any)
	if !ok {
		t.Fatalf("upstream_group_rate_multipliers type = %T", account.Credentials["upstream_group_rate_multipliers"])
	}
	if got := rates["10"]; got != 2.5 {
		t.Fatalf("rate[10] = %v, want 2.5", got)
	}
	if got := rates["20"]; got != 2.5 {
		t.Fatalf("rate[20] = %v, want 2.5", got)
	}
	if _, ok := rates["30"]; ok {
		t.Fatal("rate[30] should be ignored because local group is not bound")
	}
	if got := account.Extra["upstream_remote_api_key_id"]; got != "5798" {
		t.Fatalf("remote key extra = %v, want 5798", got)
	}
	if got := account.Extra["upstream_remote_group_id"]; got != "vip" {
		t.Fatalf("remote group extra = %v, want vip", got)
	}
}

func TestBuildRuntimeAccountFromUpstreamAPIKeyStoresLocalGroupsInExtra(t *testing.T) {
	upstream := &Upstream{
		ID:             9,
		Name:           "north",
		Type:           UpstreamTypeSub2API,
		BaseURL:        "https://upstream.example.com/",
		Status:         UpstreamStatusActive,
		Priority:       100,
		Weight:         100,
		CostMultiplier: 1,
		RoutingMode:    UpstreamRoutingBalanced,
		RemoteGroups: []*UpstreamRemoteGroup{
			{RemoteGroupID: "10", RateMultiplier: 0.08},
			{RemoteGroupID: "30", RateMultiplier: 0.065},
		},
	}
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:      "5798",
		RemoteAPIKeyName:    "pro",
		APIKey:              "sk-forward",
		SyncedRemoteGroupID: "30",
		RemoteGroupID:       "10",
		Status:              "active",
		LocalGroupIDs:       []int64{36, 38},
	}

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil)

	mapping, ok := account.Credentials["upstream_group_mapping"].(map[string]any)
	if !ok {
		t.Fatalf("upstream_group_mapping type = %T", account.Credentials["upstream_group_mapping"])
	}
	if got := mapping["36"]; got != "10" {
		t.Fatalf("mapping[36] = %v, want 10", got)
	}
	if got := mapping["38"]; got != "10" {
		t.Fatalf("mapping[38] = %v, want 10", got)
	}

	localGroupIDs, ok := account.Extra["upstream_local_group_ids"].([]any)
	if !ok {
		t.Fatalf("upstream_local_group_ids type = %T", account.Extra["upstream_local_group_ids"])
	}
	if len(localGroupIDs) != 2 || localGroupIDs[0] != int64(36) || localGroupIDs[1] != int64(38) {
		t.Fatalf("upstream_local_group_ids = %#v, want [36 38]", localGroupIDs)
	}

	rates, ok := account.Credentials["upstream_group_rate_multipliers"].(map[string]any)
	if !ok {
		t.Fatalf("upstream_group_rate_multipliers type = %T", account.Credentials["upstream_group_rate_multipliers"])
	}
	if got := rates["36"]; got != 0.08 {
		t.Fatalf("rate[36] = %v, want 0.08", got)
	}
	if got := rates["38"]; got != 0.08 {
		t.Fatalf("rate[38] = %v, want 0.08", got)
	}
	if got := account.Extra["upstream_synced_remote_group_id"]; got != "30" {
		t.Fatalf("synced remote group extra = %v, want 30", got)
	}
	if got := account.Extra["upstream_remote_group_id"]; got != "10" {
		t.Fatalf("remote group extra = %v, want 10", got)
	}
}

func TestBuildRuntimeAccountFromUpstreamAPIKeyDoesNotRequireRemoteGroup(t *testing.T) {
	upstream := &Upstream{
		ID:             9,
		Name:           "north",
		Type:           UpstreamTypeSub2API,
		BaseURL:        "https://upstream.example.com/",
		Status:         UpstreamStatusActive,
		Priority:       100,
		Weight:         100,
		CostMultiplier: 1,
		RoutingMode:    UpstreamRoutingBalanced,
		Metadata:       map[string]any{"supported_models": []any{"gpt-4o", "gpt-4.1-*"}},
	}
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:   "5798",
		RemoteAPIKeyName: "pro",
		APIKey:           "sk-forward",
		Status:           "active",
		LocalGroupIDs:    []int64{36},
	}

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil)

	if !account.Schedulable {
		t.Fatalf("account should be schedulable, error = %q", account.ErrorMessage)
	}
	mapping, ok := account.Credentials["model_mapping"].(map[string]any)
	if !ok {
		t.Fatalf("model_mapping type = %T", account.Credentials["model_mapping"])
	}
	if got := mapping["gpt-4o"]; got != "gpt-4o" {
		t.Fatalf("model_mapping[gpt-4o] = %v, want gpt-4o", got)
	}
	if got := mapping["gpt-4.1-*"]; got != "gpt-4.1-*" {
		t.Fatalf("model_mapping[gpt-4.1-*] = %v, want gpt-4.1-*", got)
	}
}

func TestUpstreamRuntimeOpenAIAPIBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		upstream *Upstream
		want     string
	}{
		{
			name:     "site root defaults to openai compatible v1 base",
			upstream: &Upstream{BaseURL: "https://upstream.example.com/"},
			want:     "https://upstream.example.com/v1",
		},
		{
			name:     "already versioned base is preserved",
			upstream: &Upstream{BaseURL: "https://upstream.example.com/v1"},
			want:     "https://upstream.example.com/v1",
		},
		{
			name: "explicit api base from metadata wins",
			upstream: &Upstream{
				BaseURL: "https://panel.example.com",
				Metadata: map[string]any{
					"openai_api_base_url": "https://gateway.example.com/openai/v1/",
				},
			},
			want: "https://gateway.example.com/openai/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upstreamRuntimeOpenAIAPIBaseURL(tt.upstream); got != tt.want {
				t.Fatalf("base url = %q, want %q", got, tt.want)
			}
		})
	}
}
