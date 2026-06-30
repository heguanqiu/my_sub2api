package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseUpstreamGroupsFromSub2APIPaginatedResponse(t *testing.T) {
	raw := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"items": [
				{"id": 36, "name": "default", "rate_multiplier": 1.25, "status": "active"},
				{"id": 38, "name": "vip", "rate_multiplier": "2"}
			],
			"total": 2
		}
	}`)

	groups := parseUpstreamGroups(raw)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if groups[0].RemoteGroupID != "36" || groups[0].RemoteGroupName != "default" || groups[0].RateMultiplier != 1.25 {
		t.Fatalf("groups[0] = %#v", groups[0])
	}
	if groups[1].RemoteGroupID != "38" || groups[1].RateMultiplier != 2 {
		t.Fatalf("groups[1] = %#v", groups[1])
	}
}

func TestParseUpstreamGroupsFromNewAPIUsableGroupsAndRatios(t *testing.T) {
	raw := []byte(`{
		"success": true,
		"data": {
			"default": {"ratio": 1, "desc": "Default"},
			"vip": {"ratio": 2.5, "desc": "VIP"}
		}
	}`)

	groups := parseUpstreamGroups(raw)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	byID := map[string]*UpstreamRemoteGroup{}
	for _, group := range groups {
		byID[group.RemoteGroupID] = group
	}
	if byID["default"].RateMultiplier != 1 || byID["default"].RemoteGroupName != "Default" {
		t.Fatalf("default group = %#v", byID["default"])
	}
	if byID["vip"].RateMultiplier != 2.5 || byID["vip"].RemoteGroupName != "VIP" {
		t.Fatalf("vip group = %#v", byID["vip"])
	}
}

func TestParseUpstreamGroupsFromNewAPIPricingResponse(t *testing.T) {
	raw := []byte(`{
		"success": true,
		"group_ratio": {"default": 1, "vip": 1.8},
		"usable_group": {"default": "Default", "vip": "VIP"}
	}`)

	groups := parseUpstreamGroups(raw)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	byID := map[string]*UpstreamRemoteGroup{}
	for _, group := range groups {
		byID[group.RemoteGroupID] = group
	}
	if byID["vip"].RateMultiplier != 1.8 {
		t.Fatalf("vip rate = %v, want 1.8", byID["vip"].RateMultiplier)
	}
}

func TestParseUpstreamAPIKeysFromNewAPITokenResponse(t *testing.T) {
	raw := []byte(`{
		"success": true,
		"data": {
			"items": [
				{"id": 7, "name": "main", "key": "sk-abcdef123456", "group": "vip", "status": 1, "remain_quota": 100, "used_quota": 20}
			],
			"total": 1
		}
	}`)

	keys := parseUpstreamAPIKeys(raw)
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	key := keys[0]
	if key.RemoteAPIKeyID != "7" || key.RemoteAPIKeyName != "main" || key.RemoteGroupID != "vip" {
		t.Fatalf("key = %#v", key)
	}
	if key.Status != "active" {
		t.Fatalf("status = %q, want active", key.Status)
	}
	if key.Quota == nil || *key.Quota != 100 || key.UsedQuota == nil || *key.UsedQuota != 20 {
		t.Fatalf("quota = %v used = %v", key.Quota, key.UsedQuota)
	}
	if key.APIKey != "" || key.APIKeyConfigured {
		t.Fatalf("synced remote API key should not be stored as a forwarding secret: key=%q configured=%v", key.APIKey, key.APIKeyConfigured)
	}
	if key.MaskedKey == "" {
		t.Fatal("masked key should be kept for display")
	}
	if key.RawSnapshot["key"] == "sk-abcdef123456" {
		t.Fatal("raw snapshot should not contain the full api key")
	}
}

func TestHTTPUpstreamAdminAdapterNewAPILoginFetchesAccessTokenWithSession(t *testing.T) {
	var sawCookie bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42,"username":"up"}}`))
		case "/api/user/token":
			if _, err := r.Cookie("session"); err == nil {
				sawCookie = true
			}
			if got := r.Header.Get("New-Api-User"); got != "42" {
				t.Fatalf("New-Api-User = %q, want 42", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":"access-token"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewHTTPUpstreamAdminAdapter()
	session, err := adapter.Login(context.Background(), &Upstream{
		BaseURL: server.URL,
		AdminAuth: &UpstreamAdminAuth{
			AuthMode: UpstreamAdminAuthPassword,
			Username: "up",
			Password: "password",
			LoginURL: server.URL + "/api/user/login",
		},
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if session.AccessToken != "access-token" || session.UserID != "42" {
		t.Fatalf("session = %#v", session)
	}
	if !sawCookie {
		t.Fatal("expected /api/user/token to receive login cookie")
	}
}

func TestHTTPUpstreamAdminAdapterTokenAuthRefreshesAccessToken(t *testing.T) {
	var sawRefresh bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/refresh" {
			http.NotFound(w, r)
			return
		}
		sawRefresh = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}}`))
	}))
	defer server.Close()

	adapter := NewHTTPUpstreamAdminAdapter()
	expired := time.Now().Add(-time.Minute).UTC()
	session, err := adapter.Login(context.Background(), &Upstream{
		BaseURL: server.URL,
		AdminAuth: &UpstreamAdminAuth{
			AuthMode:       UpstreamAdminAuthToken,
			AccessToken:    "old-access",
			RefreshToken:   "old-refresh",
			TokenExpiresAt: &expired,
		},
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if !sawRefresh {
		t.Fatal("expected refresh endpoint to be called")
	}
	if session.AccessToken != "new-access" || session.RefreshToken != "new-refresh" {
		t.Fatalf("session = %#v", session)
	}
	if session.TokenExpiresAt == nil || time.Until(*session.TokenExpiresAt) < 50*time.Minute {
		t.Fatalf("token expires at = %v, want refreshed expiry", session.TokenExpiresAt)
	}
}

func TestHTTPUpstreamAdminAdapterListGroupsUsesNewAPIHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self/groups" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("New-Api-User"); got != "42" {
			t.Fatalf("New-Api-User = %q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"vip":{"ratio":2,"desc":"VIP"}}}`))
	}))
	defer server.Close()

	adapter := NewHTTPUpstreamAdminAdapter()
	groups, err := adapter.ListGroups(context.Background(), &Upstream{BaseURL: server.URL}, &UpstreamAdminSession{
		AccessToken: "access-token",
		UserID:      "42",
	})
	if err != nil {
		t.Fatalf("ListGroups returned error: %v", err)
	}
	if len(groups) != 1 || groups[0].RemoteGroupID != "vip" {
		t.Fatalf("groups = %#v", groups)
	}
}

func TestHTTPUpstreamAdminAdapterGetAccountBalanceUsesSub2APIProfileEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":42,"balance":12.34}}`))
	}))
	defer server.Close()

	adapter := NewHTTPUpstreamAdminAdapter()
	result, err := adapter.GetAccountBalance(context.Background(), &Upstream{
		ID:      1,
		Type:    UpstreamTypeSub2API,
		BaseURL: server.URL,
	}, &UpstreamAdminSession{AccessToken: "access-token"})
	if err != nil {
		t.Fatalf("GetAccountBalance returned error: %v", err)
	}
	if result.Source != "/api/v1/auth/me" {
		t.Fatalf("source = %q, want /api/v1/auth/me", result.Source)
	}
	if result.Balance == nil || *result.Balance != 12.34 || !result.HasBalance {
		t.Fatalf("result = %#v", result)
	}
}

func TestHTTPUpstreamAdminAdapterGetAccountBalanceKeepsNewAPIPathFirst(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("New-Api-User"); got != "42" {
			t.Fatalf("New-Api-User = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"quota":99,"used_quota":10}}`))
	}))
	defer server.Close()

	adapter := NewHTTPUpstreamAdminAdapter()
	result, err := adapter.GetAccountBalance(context.Background(), &Upstream{
		ID:      1,
		Type:    UpstreamTypeNewAPI,
		BaseURL: server.URL,
	}, &UpstreamAdminSession{AccessToken: "access-token", UserID: "42"})
	if err != nil {
		t.Fatalf("GetAccountBalance returned error: %v", err)
	}
	if result.Source != "/api/user/self" {
		t.Fatalf("source = %q, want /api/user/self", result.Source)
	}
	if result.Balance == nil || *result.Balance != 99 || result.RemainingQuota == nil || *result.RemainingQuota != 99 {
		t.Fatalf("result = %#v", result)
	}
	if result.UsedQuota == nil || *result.UsedQuota != 10 {
		t.Fatalf("used quota = %v, want 10", result.UsedQuota)
	}
}

func TestParseUpstreamAccountBalanceExtractsConcurrency(t *testing.T) {
	result := parseUpstreamAccountBalance([]byte(`{"data":{"quota":99,"used_quota":10,"concurrency":8,"current_concurrency":2}}`))
	if result.Concurrency == nil || *result.Concurrency != 8 {
		t.Fatalf("concurrency = %v, want 8", result.Concurrency)
	}
	if result.ConcurrencyUsed == nil || *result.ConcurrencyUsed != 2 {
		t.Fatalf("concurrency used = %v, want 2", result.ConcurrencyUsed)
	}
}

func TestTokenExpiresAtFromJWT(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp)))
	got := tokenExpiresAtFromJWT("header." + payload + ".sig")
	if got == nil || got.Unix() != exp {
		t.Fatalf("expires at = %v, want %d", got, exp)
	}
}

func TestJoinUpstreamURLPreservesQueryAndTrailingSlash(t *testing.T) {
	got := joinUpstreamURL("https://example.com/root/", "/api/token/?p=0&page_size=1000")
	if !strings.HasPrefix(got, "https://example.com/root/api/token/?") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "p=0") || !strings.Contains(got, "page_size=1000") {
		t.Fatalf("query missing from %q", got)
	}
}
