package service

import (
	"context"
	"testing"
)

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

func TestSyncRuntimeAccountSkipsDeletedLocalGroupMappings(t *testing.T) {
	accountRepo := &upstreamRuntimeAccountRepo{}
	groupRepo := &upstreamRuntimeGroupRepo{
		groups: map[int64]*Group{
			36: {ID: 36, Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	svc := NewUpstreamService(&upstreamRuntimeRepo{}, accountRepo, groupRepo, nil, nil)
	upstream := &Upstream{
		ID:          7,
		Name:        "north",
		Type:        UpstreamTypeSub2API,
		BaseURL:     "https://upstream.example.com",
		Status:      UpstreamStatusActive,
		RoutingMode: UpstreamRoutingBalanced,
		RemoteAPIKeys: []*UpstreamRemoteAPIKey{
			{
				RemoteAPIKeyID:   "key-1",
				RemoteAPIKeyName: "pro",
				APIKey:           "sk-forward",
				Status:           UpstreamStatusActive,
				LocalGroupIDs:    []int64{36, 404},
			},
		},
	}

	err := svc.syncRuntimeAccount(context.Background(), upstream)
	if err != nil {
		t.Fatalf("syncRuntimeAccount() error = %v", err)
	}
	if len(accountRepo.created) != 1 {
		t.Fatalf("created accounts = %d, want 1", len(accountRepo.created))
	}
	if len(accountRepo.boundGroups) != 1 || len(accountRepo.boundGroups[0]) != 1 || accountRepo.boundGroups[0][0] != 36 {
		t.Fatalf("bound groups = %#v, want [[36]]", accountRepo.boundGroups)
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

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil, PlatformOpenAI)
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

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil, PlatformOpenAI)

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

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil, PlatformOpenAI)

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

func TestBuildRuntimeAccountFromUpstreamAPIKeyDisabledScheduling(t *testing.T) {
	upstream := &Upstream{
		ID:          9,
		Name:        "north",
		Type:        UpstreamTypeSub2API,
		BaseURL:     "https://upstream.example.com/",
		Status:      UpstreamStatusActive,
		Priority:    100,
		Weight:      100,
		RoutingMode: UpstreamRoutingBalanced,
	}
	disabled := false
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:    "5798",
		RemoteAPIKeyName:  "pro",
		APIKey:            "sk-forward",
		Status:            "active",
		LocalGroupIDs:     []int64{36},
		SchedulingEnabled: &disabled,
	}

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil, PlatformOpenAI)

	if account.Schedulable {
		t.Fatal("account should not be schedulable when the remote key scheduling switch is disabled")
	}
	if account.ErrorMessage != "upstream API key scheduling disabled" {
		t.Fatalf("error message = %q", account.ErrorMessage)
	}
	if got := account.Extra["upstream_remote_api_key_scheduling_enabled"]; got != false {
		t.Fatalf("scheduling extra = %v, want false", got)
	}
}

func TestUpstreamRuntimeConcurrencyPrefersSyncedAccountConcurrency(t *testing.T) {
	upstream := &Upstream{
		Weight:   100,
		Metadata: map[string]any{"account_concurrency": 8, "concurrency": 3},
	}
	if got := upstreamRuntimeConcurrency(upstream); got != 8 {
		t.Fatalf("concurrency = %d, want 8", got)
	}
}

func TestBuildRuntimeAccountFromUpstreamAPIKeyAnthropicPassthrough(t *testing.T) {
	upstream := &Upstream{
		ID:          9,
		Name:        "north",
		Type:        UpstreamTypeNewAPI,
		BaseURL:     "https://upstream.example.com/",
		Status:      UpstreamStatusActive,
		Priority:    100,
		Weight:      100,
		RoutingMode: UpstreamRoutingBalanced,
	}
	remoteKey := &UpstreamRemoteAPIKey{
		RemoteAPIKeyID:   "5798",
		RemoteAPIKeyName: "pro",
		APIKey:           "sk-forward",
		Status:           "active",
		LocalGroupIDs:    []int64{36},
	}

	account := buildRuntimeAccountFromUpstreamAPIKey(upstream, remoteKey, nil, PlatformAnthropic)

	if account.Platform != PlatformAnthropic {
		t.Fatalf("platform = %q, want %q", account.Platform, PlatformAnthropic)
	}
	if got := account.Credentials["base_url"]; got != "https://upstream.example.com" {
		t.Fatalf("base_url = %v, want upstream root", got)
	}
	if got := account.Extra["anthropic_passthrough"]; got != true {
		t.Fatalf("anthropic_passthrough = %v, want true", got)
	}
	if _, ok := account.Extra["openai_passthrough"]; ok {
		t.Fatal("openai_passthrough should not be set for anthropic runtime account")
	}
	if _, ok := account.Credentials["openai_capabilities"]; ok {
		t.Fatal("openai_capabilities should not be set for anthropic runtime account")
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

func TestUpstreamRuntimeAnthropicAPIBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		upstream *Upstream
		want     string
	}{
		{
			name:     "site root is preserved",
			upstream: &Upstream{BaseURL: "https://upstream.example.com/"},
			want:     "https://upstream.example.com",
		},
		{
			name:     "messages endpoint trims to root",
			upstream: &Upstream{BaseURL: "https://upstream.example.com/v1/messages"},
			want:     "https://upstream.example.com",
		},
		{
			name: "explicit anthropic api base from metadata wins",
			upstream: &Upstream{
				BaseURL: "https://panel.example.com",
				Metadata: map[string]any{
					"anthropic_api_base_url": "https://gateway.example.com/anthropic/",
				},
			},
			want: "https://gateway.example.com/anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upstreamRuntimeAnthropicAPIBaseURL(tt.upstream); got != tt.want {
				t.Fatalf("base url = %q, want %q", got, tt.want)
			}
		})
	}
}

type upstreamRuntimeRepo struct {
	UpstreamRepository
}

func (r *upstreamRuntimeRepo) GetRoutingMode(context.Context) (string, error) {
	return UpstreamRoutingBalanced, nil
}

type upstreamRuntimeGroupRepo struct {
	GroupRepository
	groups map[int64]*Group
}

func (r *upstreamRuntimeGroupRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	group := r.groups[id]
	if group == nil {
		return nil, ErrGroupNotFound
	}
	cp := *group
	return &cp, nil
}

type upstreamRuntimeAccountRepo struct {
	AccountRepository
	created     []*Account
	updated     []*Account
	boundGroups [][]int64
	nextID      int64
}

func (r *upstreamRuntimeAccountRepo) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}

func (r *upstreamRuntimeAccountRepo) Create(_ context.Context, account *Account) error {
	r.nextID++
	cp := *account
	cp.ID = r.nextID
	account.ID = cp.ID
	r.created = append(r.created, &cp)
	return nil
}

func (r *upstreamRuntimeAccountRepo) Update(_ context.Context, account *Account) error {
	cp := *account
	r.updated = append(r.updated, &cp)
	return nil
}

func (r *upstreamRuntimeAccountRepo) BindGroups(_ context.Context, _ int64, groupIDs []int64) error {
	r.boundGroups = append(r.boundGroups, append([]int64(nil), groupIDs...))
	return nil
}
