package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type HTTPUpstreamAdminAdapter struct {
	client *http.Client
}

func NewHTTPUpstreamAdminAdapter() UpstreamAdminAdapter {
	return &HTTPUpstreamAdminAdapter{
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a *HTTPUpstreamAdminAdapter) Login(ctx context.Context, upstream *Upstream) (*UpstreamAdminSession, error) {
	if upstream == nil || upstream.AdminAuth == nil {
		return nil, ErrUpstreamLoginFailed.WithMetadata(map[string]string{"reason": "admin auth is not configured"})
	}
	auth := upstream.AdminAuth
	if auth.AuthMode == UpstreamAdminAuthNone {
		return &UpstreamAdminSession{}, nil
	}
	if token := strings.TrimSpace(auth.AccessToken); token != "" && auth.AuthMode != UpstreamAdminAuthPassword {
		return &UpstreamAdminSession{
			AccessToken:    token,
			RefreshToken:   strings.TrimSpace(auth.RefreshToken),
			TokenExpiresAt: auth.TokenExpiresAt,
			UserID:         anyToString(auth.Metadata["user_id"]),
		}, nil
	}
	if auth.AuthMode != UpstreamAdminAuthPassword {
		return nil, ErrUpstreamLoginFailed.WithMetadata(map[string]string{"reason": "access token is empty"})
	}

	username := strings.TrimSpace(auth.Username)
	password := strings.TrimSpace(auth.Password)
	if username == "" || password == "" {
		return nil, ErrUpstreamLoginFailed.WithMetadata(map[string]string{"reason": "username or password is empty"})
	}

	loginURL := strings.TrimSpace(auth.LoginURL)
	if loginURL == "" {
		loginURL = joinUpstreamURL(upstream.BaseURL, "/api/user/login")
	}

	body, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    username,
		"password": password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	jar, _ := cookiejar.New(nil)
	client := *a.client
	if jar != nil {
		client.Jar = jar
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ErrUpstreamLoginFailed.WithMetadata(map[string]string{"status": strconv.Itoa(resp.StatusCode)})
	}
	token := firstStringInJSON(raw, "token", "access_token", "accessToken", "session_token", "key")
	userID := firstStringInJSON(raw, "user_id", "userId", "uid", "id")
	if token == "" && loginLooksSuccessful(raw) {
		session := &UpstreamAdminSession{UserID: userID}
		a.enrichNewAPIUserSession(ctx, &client, upstream, session)
		if strings.TrimSpace(session.AccessToken) != "" {
			return session, nil
		}
	}
	if token == "" {
		return nil, ErrUpstreamLoginFailed.WithMetadata(map[string]string{"reason": "token not found in login response"})
	}
	return &UpstreamAdminSession{
		AccessToken:  token,
		RefreshToken: firstStringInJSON(raw, "refresh_token", "refreshToken"),
		UserID:       userID,
	}, nil
}

func (a *HTTPUpstreamAdminAdapter) enrichNewAPIUserSession(ctx context.Context, client *http.Client, upstream *Upstream, session *UpstreamAdminSession) {
	if client == nil || upstream == nil || session == nil {
		return
	}
	if strings.TrimSpace(session.UserID) == "" {
		if raw, err := a.getJSONWithClient(ctx, client, upstream, session, "/api/user/self"); err == nil {
			session.UserID = firstStringInJSON(raw, "user_id", "userId", "uid", "id")
		}
	}
	if strings.TrimSpace(session.AccessToken) == "" && strings.TrimSpace(session.UserID) != "" {
		if raw, err := a.getJSONWithClient(ctx, client, upstream, session, "/api/user/token"); err == nil {
			session.AccessToken = firstStringInJSON(raw, "token", "access_token", "accessToken", "key", "data")
		}
	}
}

func (a *HTTPUpstreamAdminAdapter) ListGroups(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) ([]*UpstreamRemoteGroup, error) {
	paths := []string{
		"/api/v1/groups/available",
		"/api/v1/groups/rates",
		"/api/pricing",
		"/api/v1/admin/groups/all?include_inactive=true",
		"/api/v1/admin/groups?page_size=1000",
		"/api/v1/group/",
		"/api/v1/group",
		"/api/v1/groups",
		"/api/v1/user/groups",
		"/api/v1/user/self/groups",
		"/api/user/self/groups",
		"/api/group",
		"/api/group/",
		"/api/groups",
		"/api/admin/groups",
		"/api/user/groups",
	}
	errs := make([]error, 0, len(paths))
	var emptyPaths []string
	for _, p := range paths {
		raw, err := a.getJSON(ctx, upstream, session, p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		groups := parseUpstreamGroups(raw)
		if len(groups) > 0 {
			return groups, nil
		}
		emptyPaths = append(emptyPaths, p)
	}
	if len(errs) > 0 {
		return nil, upstreamDiscoveryError("groups", errs, emptyPaths)
	}
	return []*UpstreamRemoteGroup{}, nil
}

func (a *HTTPUpstreamAdminAdapter) ListAPIKeys(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) ([]*UpstreamRemoteAPIKey, error) {
	paths := []string{
		"/api/v1/keys?page_size=1000",
		"/api/v1/api-keys?page_size=1000",
		"/api/v1/tokens?page_size=1000",
		"/api/v1/token?page_size=1000",
		"/api/v1/admin/api-keys?page_size=1000",
		"/api/v1/admin/tokens?page_size=1000",
		"/api/v1/user/keys?page_size=1000",
		"/api/v1/user/tokens?page_size=1000",
		"/api/v1/user/self/tokens?page_size=1000",
		"/api/token/?p=0&page_size=1000",
		"/api/token?p=0&page_size=1000",
		"/api/token",
		"/api/token/",
		"/api/tokens",
		"/api/key",
		"/api/keys",
		"/api/user/token",
		"/api/user/tokens",
		"/api/admin/tokens",
	}
	errs := make([]error, 0, len(paths))
	var emptyPaths []string
	for _, p := range paths {
		raw, err := a.getJSON(ctx, upstream, session, p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		keys := parseUpstreamAPIKeys(raw)
		if len(keys) > 0 {
			return keys, nil
		}
		emptyPaths = append(emptyPaths, p)
	}
	if len(errs) > 0 {
		return nil, upstreamDiscoveryError("api keys", errs, emptyPaths)
	}
	return []*UpstreamRemoteAPIKey{}, nil
}

func (a *HTTPUpstreamAdminAdapter) GetAccountBalance(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) (*UpstreamAccountBalanceResult, error) {
	raw, err := a.getJSON(ctx, upstream, session, "/api/user/self")
	if err != nil {
		return nil, err
	}
	result := parseNewAPIAccountBalance(raw)
	now := time.Now().UTC()
	result.UpstreamID = upstream.ID
	result.Source = "/api/user/self"
	result.CheckedAt = now
	result.HasBalance = result.Balance != nil || result.Quota != nil || result.UsedQuota != nil || result.RemainingQuota != nil
	if result.Message == "" {
		if result.HasBalance {
			result.Message = "account balance refreshed"
		} else {
			result.Message = "upstream /api/user/self did not include quota fields"
		}
	}
	return result, nil
}

func (a *HTTPUpstreamAdminAdapter) getJSON(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession, relPath string) ([]byte, error) {
	return a.getJSONWithClient(ctx, a.client, upstream, session, relPath)
}

func (a *HTTPUpstreamAdminAdapter) getJSONWithClient(ctx context.Context, client *http.Client, upstream *Upstream, session *UpstreamAdminSession, relPath string) ([]byte, error) {
	if client == nil {
		client = a.client
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinUpstreamURL(upstream.BaseURL, relPath), nil)
	if err != nil {
		return nil, err
	}
	if session != nil && strings.TrimSpace(session.AccessToken) != "" {
		token := strings.TrimSpace(session.AccessToken)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-API-Key", token)
		if userID := strings.TrimSpace(session.UserID); userID != "" {
			req.Header.Set("New-Api-User", userID)
		}
	}
	if session != nil && strings.TrimSpace(session.UserID) != "" && req.Header.Get("New-Api-User") == "" {
		req.Header.Set("New-Api-User", strings.TrimSpace(session.UserID))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream %s returned HTTP %d", relPath, resp.StatusCode)
	}
	return raw, nil
}

func upstreamDiscoveryError(kind string, errs []error, emptyPaths []string) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		msg := err.Error()
		duplicate := false
		for _, existing := range parts {
			if existing == msg {
				duplicate = true
				break
			}
		}
		if !duplicate {
			parts = append(parts, msg)
		}
	}
	if len(emptyPaths) > 0 {
		parts = append(parts, "empty or unrecognized response at "+strings.Join(emptyPaths, ", "))
	}
	if len(parts) == 0 {
		return fmt.Errorf("upstream %s discovery returned no usable data", kind)
	}
	return fmt.Errorf("upstream %s discovery failed: %s", kind, strings.Join(parts, "; "))
}

func joinUpstreamURL(baseURL, relPath string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(relPath, "/")
	}
	rel, err := url.Parse(relPath)
	if err == nil && rel.RawQuery != "" {
		u.RawQuery = rel.RawQuery
		relPath = rel.Path
	}
	u.Path = path.Join(u.Path, relPath)
	if strings.HasSuffix(relPath, "/") && !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String()
}

func firstStringInJSON(raw []byte, keys ...string) string {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ""
	}
	return firstStringInValue(decoded, keys...)
}

func firstStringInValue(v any, keys ...string) string {
	switch x := v.(type) {
	case map[string]any:
		for _, key := range keys {
			if val, ok := x[key]; ok {
				if s := anyToString(val); s != "" {
					return s
				}
			}
		}
		for _, val := range x {
			if s := firstStringInValue(val, keys...); s != "" {
				return s
			}
		}
	case []any:
		for _, val := range x {
			if s := firstStringInValue(val, keys...); s != "" {
				return s
			}
		}
	}
	return ""
}

func parseUpstreamGroups(raw []byte) []*UpstreamRemoteGroup {
	if mapped := parseMappedUpstreamGroups(raw); len(mapped) > 0 {
		return mapped
	}
	items := extractArrayPayload(raw)
	out := make([]*UpstreamRemoteGroup, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := firstObjectString(m, "id", "group_id", "name", "group")
		name := firstObjectString(m, "name", "group_name", "group", "id")
		if id == "" || name == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, &UpstreamRemoteGroup{
			RemoteGroupID:   id,
			RemoteGroupName: name,
			RateMultiplier:  firstObjectFloat(m, 1, "ratio", "rate", "rate_multiplier", "multiplier", "group_ratio"),
			Status:          defaultString(firstObjectString(m, "status"), "active"),
			RawSnapshot:     cloneObjectMap(m),
		})
	}
	return out
}

func parseUpstreamAPIKeys(raw []byte) []*UpstreamRemoteAPIKey {
	items := extractArrayPayload(raw)
	out := make([]*UpstreamRemoteAPIKey, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := firstObjectString(m, "id", "key_id", "token_id", "name", "key")
		name := defaultString(firstObjectString(m, "name", "key_name", "token_name", "id"), id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		apiKey := firstObjectString(m, "key", "token", "api_key")
		remoteGroupID := firstObjectStringDeep(m, []string{"group_id"}, []string{"group"}, []string{"group_name"}, []string{"Group"}, []string{"group", "id"}, []string{"group", "name"})
		out = append(out, &UpstreamRemoteAPIKey{
			RemoteAPIKeyID:      id,
			RemoteAPIKeyName:    name,
			APIKey:              "",
			APIKeyConfigured:    false,
			MaskedKey:           maskSecret(apiKey),
			SyncedRemoteGroupID: remoteGroupID,
			RemoteGroupID:       remoteGroupID,
			Status:              normalizeRemoteKeyStatus(firstObjectString(m, "status")),
			Quota:               optionalObjectFloat(m, "quota", "remain_quota", "balance", "total_granted"),
			UsedQuota:           optionalObjectFloat(m, "used_quota", "quota_used", "used_amount", "used", "total_used"),
			RawSnapshot:         sanitizeRawSnapshot(m),
		})
	}
	return out
}

func parseNewAPIAccountBalance(raw []byte) *UpstreamAccountBalanceResult {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return &UpstreamAccountBalanceResult{Message: "invalid /api/user/self response"}
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return &UpstreamAccountBalanceResult{Message: "invalid /api/user/self response"}
	}
	payload := root
	if data, ok := root["data"].(map[string]any); ok {
		payload = data
	}

	quota := optionalObjectFloat(payload, "quota")
	usedQuota := optionalObjectFloat(payload, "used_quota")
	result := &UpstreamAccountBalanceResult{
		Quota:     quota,
		UsedQuota: usedQuota,
	}
	if quota != nil {
		// new-api stores current usable account balance in user.quota.
		result.Balance = quota
		result.RemainingQuota = quota
	}
	return result
}

func parseMappedUpstreamGroups(raw []byte) []*UpstreamRemoteGroup {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}
	ratioMap := firstMapAtPaths(root, []string{"group_ratio"}, []string{"data", "group_ratio"})
	groups := firstMapAtPaths(root, []string{"usable_group"}, []string{"data", "usable_group"}, []string{"data"})
	if len(groups) == 0 || len(ratioMap) == 0 && len(groups) == 1 && groups["items"] != nil {
		return nil
	}
	out := make([]*UpstreamRemoteGroup, 0, len(groups))
	for id, value := range groups {
		if id == "" || isEnvelopeKey(id) {
			continue
		}
		name := id
		status := "active"
		rawGroup := map[string]any{"id": id, "name": id}
		if m, ok := value.(map[string]any); ok {
			if s := firstObjectString(m, "name", "group_name", "desc", "description"); s != "" {
				name = s
			}
			status = defaultString(firstObjectString(m, "status"), status)
			for k, v := range m {
				rawGroup[k] = v
			}
		} else if s := anyToString(value); s != "" {
			rawGroup["desc"] = s
		}
		rate := 1.0
		if v := numericAny(ratioMap[id]); v != nil {
			rate = *v
		} else if m, ok := value.(map[string]any); ok {
			rate = firstObjectFloat(m, rate, "ratio", "rate", "rate_multiplier", "multiplier", "group_ratio")
		}
		rawGroup["rate_multiplier"] = rate
		out = append(out, &UpstreamRemoteGroup{
			RemoteGroupID:   id,
			RemoteGroupName: name,
			RateMultiplier:  rate,
			Status:          status,
			RawSnapshot:     rawGroup,
		})
	}
	return out
}

func firstMapAtPaths(root map[string]any, paths ...[]string) map[string]any {
	for _, p := range paths {
		if m := mapAtPath(root, p...); len(m) > 0 {
			return m
		}
	}
	return nil
}

func mapAtPath(v any, keys ...string) map[string]any {
	current := v
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	m, _ := current.(map[string]any)
	return m
}

func isEnvelopeKey(key string) bool {
	switch key {
	case "success", "message", "code", "data", "items", "total", "page", "page_size", "pages":
		return true
	default:
		return false
	}
}

func extractArrayPayload(raw []byte) []any {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	if dict, ok := decoded.(map[string]any); ok {
		if data, ok := dict["data"].(map[string]any); ok {
			if groups, ok := data["groups"].(map[string]any); ok {
				items := make([]any, 0, len(groups))
				for id, value := range groups {
					item := map[string]any{"id": id, "name": id}
					if v, ok := value.(map[string]any); ok {
						for k, nested := range v {
							item[k] = nested
						}
					} else if s := anyToString(value); s != "" {
						item["name"] = s
					}
					items = append(items, item)
				}
				return items
			}
		}
	}
	switch v := decoded.(type) {
	case []any:
		return v
	case map[string]any:
		for _, key := range []string{"data", "items", "list", "rows", "records", "groups", "tokens", "keys", "api_keys", "apiKeys"} {
			if arr, ok := v[key].([]any); ok {
				return arr
			}
			if nested, ok := v[key].(map[string]any); ok {
				for _, nestedKey := range []string{"items", "list", "data", "rows", "records"} {
					if arr, ok := nested[nestedKey].([]any); ok {
						return arr
					}
				}
			}
		}
	}
	return nil
}

func firstObjectStringDeep(m map[string]any, paths ...[]string) string {
	for _, p := range paths {
		if s := valueAtPathString(m, p...); s != "" {
			return s
		}
	}
	return ""
}

func valueAtPathString(v any, keys ...string) string {
	current := v
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	return anyToString(current)
}

func firstObjectString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := anyToString(m[key]); s != "" {
			return s
		}
	}
	return ""
}

func firstObjectFloat(m map[string]any, fallback float64, keys ...string) float64 {
	if v := optionalObjectFloat(m, keys...); v != nil {
		return *v
	}
	return fallback
}

func optionalObjectFloat(m map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if v := numericAny(m[key]); v != nil {
			return v
		}
	}
	return nil
}

func numericAny(v any) *float64 {
	switch x := v.(type) {
	case float64:
		return &x
	case float32:
		f := float64(x)
		return &f
	case int:
		f := float64(x)
		return &f
	case int64:
		f := float64(x)
		return &f
	case json.Number:
		if parsed, err := x.Float64(); err == nil {
			return &parsed
		}
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil {
			return &parsed
		}
	}
	return nil
}

func anyToString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case json.Number:
		return x.String()
	default:
		return ""
	}
}

func cloneObjectMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sanitizeRawSnapshot(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if isSensitiveSnapshotKey(k) {
			if s := anyToString(v); s != "" {
				out[k] = maskSecret(s)
			} else {
				out[k] = "***"
			}
			continue
		}
		out[k] = sanitizeSnapshotValue(v)
	}
	return out
}

func sanitizeSnapshotValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return sanitizeRawSnapshot(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = sanitizeSnapshotValue(item)
		}
		return out
	default:
		return v
	}
}

func isSensitiveSnapshotKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "key", "token", "api_key", "apikey", "access_token", "accesstoken", "refresh_token", "refreshtoken", "password", "secret", "authorization":
		return true
	default:
		return false
	}
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func normalizeRemoteKeyStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", "1", "true":
		return "active"
	case "2", "0", "false":
		return "inactive"
	default:
		return strings.TrimSpace(status)
	}
}

func loginLooksSuccessful(raw []byte) bool {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return false
	}
	m, ok := decoded.(map[string]any)
	if !ok {
		return false
	}
	success, ok := m["success"].(bool)
	if ok && success {
		return true
	}
	code := anyToString(m["code"])
	return code == "0" || strings.EqualFold(code, "true")
}

func maskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "***" + secret[len(secret)-4:]
}

func maskedSecretLike(secret string) bool {
	secret = strings.TrimSpace(secret)
	return secret == "" || strings.Contains(secret, "*")
}
