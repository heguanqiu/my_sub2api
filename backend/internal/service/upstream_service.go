package service

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type UpstreamService struct {
	repo        UpstreamRepository
	accountRepo AccountRepository
	groupRepo   GroupRepository
	encryptor   SecretEncryptor
	adapter     UpstreamAdminAdapter
}

func NewUpstreamService(repo UpstreamRepository, accountRepo AccountRepository, groupRepo GroupRepository, encryptor SecretEncryptor, adapter UpstreamAdminAdapter) *UpstreamService {
	return &UpstreamService{
		repo:        repo,
		accountRepo: accountRepo,
		groupRepo:   groupRepo,
		encryptor:   encryptor,
		adapter:     adapter,
	}
}

func (s *UpstreamService) List(ctx context.Context, params UpstreamListParams) ([]*Upstream, int64, error) {
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range items {
		attachUpstreamMetadata(item)
		s.attachSchedulerSnapshot(item)
	}
	return items, total, nil
}

func (s *UpstreamService) Get(ctx context.Context, id int64) (*Upstream, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	attachUpstreamMetadata(upstream)
	s.attachSchedulerSnapshot(upstream)
	s.maskSecretsForResponse(upstream)
	return upstream, nil
}

func (s *UpstreamService) Create(ctx context.Context, params UpstreamCreateParams) (*Upstream, error) {
	upstream := &Upstream{
		Name:                 strings.TrimSpace(params.Name),
		Type:                 strings.TrimSpace(params.Type),
		BaseURL:              strings.TrimSpace(params.BaseURL),
		Status:               strings.TrimSpace(params.Status),
		Priority:             params.Priority,
		Weight:               params.Weight,
		CostMultiplier:       params.CostMultiplier,
		TimeoutMS:            params.TimeoutMS,
		ConnectTimeoutMS:     params.ConnectTimeoutMS,
		RetryMax:             params.RetryMax,
		ProbeEnabled:         params.ProbeEnabled,
		ProbeModel:           strings.TrimSpace(params.ProbeModel),
		ProbeIntervalSeconds: params.ProbeIntervalSeconds,
		RoutingMode:          strings.TrimSpace(params.RoutingMode),
		Notes:                strings.TrimSpace(params.Notes),
		Metadata:             copyAnyMap(params.Metadata),
		LocalGroupIDs:        params.LocalGroupIDs,
	}
	normalizeUpstreamMetadata(upstream)
	if err := normalizeUpstream(upstream); err != nil {
		return nil, err
	}
	if _, err := s.validateRuntimeLocalGroupIDs(ctx, upstream.LocalGroupIDs); err != nil {
		return nil, err
	}
	var forwardCredential *UpstreamForwardCredential
	if params.ForwardCredential != nil {
		credential, err := s.buildForwardCredential(0, nil, params.ForwardCredential)
		if err != nil {
			return nil, err
		}
		forwardCredential = credential
	}
	var adminAuth *UpstreamAdminAuth
	if params.AdminAuth != nil {
		auth, err := s.buildAdminAuth(0, nil, params.AdminAuth)
		if err != nil {
			return nil, err
		}
		adminAuth = auth
	}
	if err := s.repo.Create(ctx, upstream); err != nil {
		return nil, err
	}
	if forwardCredential != nil {
		forwardCredential.UpstreamID = upstream.ID
		if err := s.repo.UpsertForwardCredential(ctx, forwardCredential); err != nil {
			return nil, err
		}
	}
	if adminAuth != nil {
		adminAuth.UpstreamID = upstream.ID
		if err := s.repo.UpsertAdminAuth(ctx, adminAuth); err != nil {
			return nil, err
		}
	}
	if err := s.ensureRuntimeAccount(ctx, upstream.ID); err != nil {
		return nil, err
	}
	return s.Get(ctx, upstream.ID)
}

func (s *UpstreamService) Update(ctx context.Context, id int64, params UpstreamUpdateParams) (*Upstream, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if params.Name != nil {
		current.Name = strings.TrimSpace(*params.Name)
	}
	if params.Type != nil {
		current.Type = strings.TrimSpace(*params.Type)
	}
	if params.BaseURL != nil {
		current.BaseURL = strings.TrimSpace(*params.BaseURL)
	}
	if params.Status != nil {
		current.Status = strings.TrimSpace(*params.Status)
	}
	if params.Priority != nil {
		current.Priority = *params.Priority
	}
	if params.Weight != nil {
		current.Weight = *params.Weight
	}
	if params.CostMultiplier != nil {
		current.CostMultiplier = *params.CostMultiplier
	}
	if params.TimeoutMS != nil {
		current.TimeoutMS = *params.TimeoutMS
	}
	if params.ConnectTimeoutMS != nil {
		current.ConnectTimeoutMS = *params.ConnectTimeoutMS
	}
	if params.RetryMax != nil {
		current.RetryMax = *params.RetryMax
	}
	if params.ProbeEnabled != nil {
		current.ProbeEnabled = *params.ProbeEnabled
	}
	if params.ProbeModel != nil {
		current.ProbeModel = strings.TrimSpace(*params.ProbeModel)
	}
	if params.ProbeIntervalSeconds != nil {
		current.ProbeIntervalSeconds = *params.ProbeIntervalSeconds
	}
	if params.RoutingMode != nil {
		current.RoutingMode = strings.TrimSpace(*params.RoutingMode)
	}
	if params.Notes != nil {
		current.Notes = strings.TrimSpace(*params.Notes)
	}
	if params.Metadata != nil {
		current.Metadata = copyAnyMap(*params.Metadata)
	}
	if params.LocalGroupIDs != nil {
		current.LocalGroupIDs = *params.LocalGroupIDs
	}
	normalizeUpstreamMetadata(current)
	if err := normalizeUpstream(current); err != nil {
		return nil, err
	}
	if _, err := s.validateRuntimeLocalGroupIDs(ctx, current.LocalGroupIDs); err != nil {
		return nil, err
	}
	var forwardCredential *UpstreamForwardCredential
	if params.ForwardCredential != nil {
		credential, err := s.buildForwardCredential(id, current.ForwardCredential, params.ForwardCredential)
		if err != nil {
			return nil, err
		}
		forwardCredential = credential
	}
	var adminAuth *UpstreamAdminAuth
	if params.AdminAuth != nil {
		auth, err := s.buildAdminAuth(id, current.AdminAuth, params.AdminAuth)
		if err != nil {
			return nil, err
		}
		adminAuth = auth
	}
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}
	if forwardCredential != nil {
		if err := s.repo.UpsertForwardCredential(ctx, forwardCredential); err != nil {
			return nil, err
		}
	}
	if adminAuth != nil {
		if err := s.repo.UpsertAdminAuth(ctx, adminAuth); err != nil {
			return nil, err
		}
	}
	if err := s.ensureRuntimeAccount(ctx, id); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *UpstreamService) Delete(ctx context.Context, id int64) error {
	if err := s.disableRuntimeAccount(ctx, id, "upstream deleted"); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *UpstreamService) SyncRemoteResources(ctx context.Context, id int64) (*UpstreamSyncResult, error) {
	started := time.Now().UTC()
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.decryptSecretsForUse(upstream); err != nil {
		_ = s.recordFailedSync(ctx, id, started, err)
		return nil, err
	}

	session, err := s.adapter.Login(ctx, upstream)
	if err != nil {
		if upstream.AdminAuth != nil {
			upstream.AdminAuth.LastLoginError = err.Error()
			_ = s.persistPlainAdminAuth(ctx, upstream.AdminAuth)
		}
		_ = s.recordFailedSync(ctx, id, started, err)
		return nil, ErrUpstreamLoginFailed.WithCause(err)
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
			_ = s.recordFailedSync(ctx, id, started, err)
			return nil, err
		}
	}

	groups, err := s.adapter.ListGroups(ctx, upstream, session)
	if err != nil {
		_ = s.recordFailedSync(ctx, id, started, err)
		return nil, ErrUpstreamSyncFailed.WithCause(err)
	}
	keys, err := s.adapter.ListAPIKeys(ctx, upstream, session)
	if err != nil {
		keys = []*UpstreamRemoteAPIKey{}
	}

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
				_ = s.recordFailedSync(ctx, id, started, err)
				return nil, err
			}
			key.APIKey = encrypted
			key.APIKeyConfigured = true
		}
		if key.LastSyncedAt.IsZero() {
			key.LastSyncedAt = now
		}
	}

	finished := time.Now().UTC()
	run := &UpstreamSyncRun{
		UpstreamID:   id,
		Status:       "success",
		GroupsCount:  len(groups),
		APIKeysCount: len(keys),
		Message:      "sync completed",
		StartedAt:    started,
		FinishedAt:   &finished,
		RawResult: map[string]any{
			"groups_count":   len(groups),
			"api_keys_count": len(keys),
		},
	}
	if err != nil {
		run.Message = "sync completed without remote api keys: " + err.Error()
		run.RawResult["api_keys_error"] = err.Error()
	}
	if err := s.repo.ReplaceRemoteResources(ctx, id, groups, keys, run); err != nil {
		return nil, err
	}
	if err := s.ensureRuntimeAccount(ctx, id); err != nil {
		return nil, err
	}
	maskRemoteAPIKeySecretsForResponse(keys)
	return &UpstreamSyncResult{Run: run, Groups: groups, APIKeys: keys}, nil
}

func (s *UpstreamService) TestAdminLogin(ctx context.Context, id int64) (*UpstreamLoginTestResult, error) {
	upstream, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.decryptSecretsForUse(upstream); err != nil {
		return nil, err
	}
	session, err := s.adapter.Login(ctx, upstream)
	if err != nil {
		return nil, ErrUpstreamLoginFailed.WithCause(err)
	}
	var tokenExpiresAt *time.Time
	hasToken := false
	if session != nil {
		tokenExpiresAt = session.TokenExpiresAt
		hasToken = strings.TrimSpace(session.AccessToken) != ""
	}
	if upstream.AdminAuth != nil && session != nil {
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
			return nil, err
		}
	}
	return &UpstreamLoginTestResult{
		Success:        true,
		HasToken:       hasToken,
		TokenExpiresAt: tokenExpiresAt,
		Message:        "login succeeded",
	}, nil
}

func (s *UpstreamService) ListRemoteGroups(ctx context.Context, id int64) ([]*UpstreamRemoteGroup, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListRemoteGroups(ctx, id)
}

func (s *UpstreamService) ListRemoteAPIKeys(ctx context.Context, id int64) ([]*UpstreamRemoteAPIKey, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}
	keys, err := s.repo.ListRemoteAPIKeys(ctx, id)
	if err != nil {
		return nil, err
	}
	maskRemoteAPIKeySecretsForResponse(keys)
	return keys, nil
}

func (s *UpstreamService) UpdateRemoteAPIKeyConfig(ctx context.Context, id int64, remoteAPIKeyID string, remoteGroupID string, localGroupIDs []int64, apiKey *string) (*UpstreamRemoteAPIKey, error) {
	if strings.TrimSpace(remoteAPIKeyID) == "" {
		return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "remote_api_key_id"})
	}
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}
	remoteGroupID = strings.TrimSpace(remoteGroupID)
	if remoteGroupID != "" {
		groups, err := s.repo.ListRemoteGroups(ctx, id)
		if err != nil {
			return nil, err
		}
		found := false
		for _, group := range groups {
			if group != nil && strings.TrimSpace(group.RemoteGroupID) == remoteGroupID {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "remote_group_id", "reason": "remote group not found"})
		}
	}
	validLocalGroupIDs, err := s.validateRuntimeLocalGroupIDs(ctx, localGroupIDs)
	if err != nil {
		return nil, err
	}
	var encryptedAPIKey *string
	if apiKey != nil {
		plainAPIKey := strings.TrimSpace(*apiKey)
		if plainAPIKey != "" {
			encrypted, err := s.encryptSecret(plainAPIKey)
			if err != nil {
				return nil, err
			}
			encryptedAPIKey = &encrypted
		}
	}
	key, err := s.repo.UpdateRemoteAPIKeyConfig(ctx, id, remoteAPIKeyID, remoteGroupID, validLocalGroupIDs, encryptedAPIKey)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRuntimeAccount(ctx, id); err != nil {
		return nil, err
	}
	maskRemoteAPIKeySecretsForResponse([]*UpstreamRemoteAPIKey{key})
	return key, nil
}

func (s *UpstreamService) SchedulePreview(ctx context.Context, req UpstreamScheduleRequest) (*UpstreamScheduleDecision, error) {
	apiKeyCandidates := make([]UpstreamAPIKeyScheduleCandidate, 0)
	filteredAPIKeys := make([]UpstreamAPIKeyScheduleCandidate, 0)
	if len(req.Candidates) == 0 {
		items, _, err := s.repo.List(ctx, UpstreamListParams{Page: 1, PageSize: 100})
		if err != nil {
			return nil, err
		}
		req.Candidates = make([]UpstreamCandidate, 0, len(items))
		for _, upstream := range items {
			groups, _ := s.repo.ListRemoteGroups(ctx, upstream.ID)
			keys, _ := s.repo.ListRemoteAPIKeys(ctx, upstream.ID)
			keyCandidates, keyFiltered := scheduleAPIKeyCandidates(upstream, groups, keys, req.LocalGroupID, req.RemoteGroupID)
			apiKeyCandidates = append(apiKeyCandidates, keyCandidates...)
			filteredAPIKeys = append(filteredAPIKeys, keyFiltered...)
			candidate := UpstreamCandidate{
				ID:                upstream.ID,
				Name:              upstream.Name,
				Status:            upstream.Status,
				Priority:          upstream.Priority,
				Weight:            upstream.Weight,
				CostMultiplier:    upstream.CostMultiplier * remoteGroupMultiplier(groups, req.RemoteGroupID),
				SupportedModels:   upstreamSupportedModels(upstream),
				RemoteGroupIDs:    remoteGroupIDs(groups),
				CapacityAvailable: len(keyCandidates) > 0,
				HealthScore:       upstream.LatestHealthScore,
				PerformanceScore:  1,
				CapacityScore:     capacityScoreFromKeyCount(len(keyCandidates)),
			}
			req.Candidates = append(req.Candidates, candidate)
		}
	}
	decision := SelectUpstreamCandidate(req)
	decision.CandidateAPIKeys = apiKeyCandidates
	decision.FilteredAPIKeys = filteredAPIKeys
	if decision.SelectedID > 0 {
		for _, key := range apiKeyCandidates {
			if key.UpstreamID != decision.SelectedID {
				continue
			}
			decision.SelectedRemoteAPIKeyID = key.RemoteAPIKeyID
			decision.SelectedRemoteGroupID = key.RemoteGroupID
			break
		}
	}
	for i := range decision.CandidateScores {
		score := &decision.CandidateScores[i]
		for _, key := range apiKeyCandidates {
			if key.UpstreamID == score.UpstreamID {
				score.CandidateAPIKeys = append(score.CandidateAPIKeys, key)
			}
		}
	}
	return &decision, nil
}

func scheduleAPIKeyCandidates(upstream *Upstream, groups []*UpstreamRemoteGroup, keys []*UpstreamRemoteAPIKey, localGroupID int64, remoteGroupID string) ([]UpstreamAPIKeyScheduleCandidate, []UpstreamAPIKeyScheduleCandidate) {
	groupSet := map[string]bool{}
	for _, group := range groups {
		if group == nil || strings.TrimSpace(group.RemoteGroupID) == "" {
			continue
		}
		groupSet[strings.TrimSpace(group.RemoteGroupID)] = remoteAPIKeyStatusActive(group.Status)
	}
	eligible := make([]UpstreamAPIKeyScheduleCandidate, 0, len(keys))
	filtered := make([]UpstreamAPIKeyScheduleCandidate, 0)
	for _, key := range keys {
		if key == nil {
			continue
		}
		item := UpstreamAPIKeyScheduleCandidate{
			UpstreamID:       upstream.ID,
			UpstreamName:     upstream.Name,
			RemoteAPIKeyID:   strings.TrimSpace(key.RemoteAPIKeyID),
			RemoteAPIKeyName: strings.TrimSpace(key.RemoteAPIKeyName),
			RemoteGroupID:    strings.TrimSpace(key.RemoteGroupID),
			LocalGroupIDs:    uniquePositiveInt64sLocal(key.LocalGroupIDs),
			Status:           strings.TrimSpace(key.Status),
		}
		reason := ""
		if !remoteAPIKeyStatusActive(key.Status) {
			reason = "api key inactive"
		} else if strings.TrimSpace(key.APIKey) == "" && !key.APIKeyConfigured {
			reason = "api key secret missing"
		} else if item.RemoteGroupID != "" && !groupSet[item.RemoteGroupID] {
			reason = "remote group inactive or missing"
		} else if strings.TrimSpace(remoteGroupID) != "" && !strings.EqualFold(strings.TrimSpace(remoteGroupID), item.RemoteGroupID) {
			reason = "remote group mismatch"
		} else if localGroupID > 0 && !containsInt64(item.LocalGroupIDs, localGroupID) {
			reason = "local group not bound"
		} else if len(item.LocalGroupIDs) == 0 {
			reason = "local groups missing"
		}
		if reason != "" {
			item.Schedulable = false
			item.FilterReason = reason
			filtered = append(filtered, item)
			continue
		}
		item.Schedulable = true
		eligible = append(eligible, item)
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].RemoteGroupID != eligible[j].RemoteGroupID {
			return eligible[i].RemoteGroupID < eligible[j].RemoteGroupID
		}
		return eligible[i].RemoteAPIKeyID < eligible[j].RemoteAPIKeyID
	})
	return eligible, filtered
}

func capacityScoreFromKeyCount(count int) float64 {
	switch {
	case count <= 0:
		return 0
	case count >= 5:
		return 1
	default:
		return 0.5 + float64(count)*0.1
	}
}

func (s *UpstreamService) buildForwardCredential(upstreamID int64, existing *UpstreamForwardCredential, input *UpstreamForwardCredentialInput) (*UpstreamForwardCredential, error) {
	credential := &UpstreamForwardCredential{
		UpstreamID: upstreamID,
		Name:       "default",
		AuthType:   UpstreamForwardAuthBearer,
		Enabled:    true,
		Metadata:   map[string]any{},
	}
	if existing != nil {
		*credential = *existing
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		credential.Name = name
	}
	if authType := normalizeUpstreamForwardAuthType(input.AuthType); authType != "" {
		credential.AuthType = authType
	}
	if !isValidUpstreamForwardAuthType(credential.AuthType) {
		return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "forward_credential.auth_type"})
	}
	credential.Enabled = input.Enabled
	credential.ExpiresAt = input.ExpiresAt
	if input.Metadata != nil {
		credential.Metadata = input.Metadata
	}
	if input.APIKey != nil {
		encrypted, err := s.encryptSecret(strings.TrimSpace(*input.APIKey))
		if err != nil {
			return nil, err
		}
		credential.APIKey = encrypted
	}
	if credential.Metadata == nil {
		credential.Metadata = map[string]any{}
	}
	return credential, nil
}

func normalizeUpstreamForwardAuthType(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "":
		return ""
	case "api_key", "apikey", "openai_api_key":
		return UpstreamForwardAuthOpenAI
	default:
		return strings.TrimSpace(strings.ToLower(raw))
	}
}

func isValidUpstreamForwardAuthType(authType string) bool {
	switch authType {
	case UpstreamForwardAuthBearer, UpstreamForwardAuthOpenAI, UpstreamForwardAuthCustom:
		return true
	default:
		return false
	}
}

func (s *UpstreamService) buildAdminAuth(upstreamID int64, existing *UpstreamAdminAuth, input *UpstreamAdminAuthInput) (*UpstreamAdminAuth, error) {
	auth := &UpstreamAdminAuth{
		UpstreamID: upstreamID,
		AuthMode:   UpstreamAdminAuthPassword,
		Metadata:   map[string]any{},
	}
	if existing != nil {
		*auth = *existing
	}
	if mode := normalizeUpstreamAdminAuthMode(input.AuthMode); mode != "" {
		auth.AuthMode = mode
	}
	if !isValidUpstreamAdminAuthMode(auth.AuthMode) {
		return nil, ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "admin_auth.auth_mode"})
	}
	auth.LoginURL = strings.TrimSpace(input.LoginURL)
	auth.TokenExpiresAt = input.TokenExpiresAt
	if input.Metadata != nil {
		auth.Metadata = input.Metadata
	}
	var err error
	if input.Username != nil {
		auth.Username, err = s.encryptSecret(strings.TrimSpace(*input.Username))
		if err != nil {
			return nil, err
		}
	}
	if input.Password != nil {
		auth.Password, err = s.encryptSecret(strings.TrimSpace(*input.Password))
		if err != nil {
			return nil, err
		}
	}
	if input.AccessToken != nil {
		auth.AccessToken, err = s.encryptSecret(strings.TrimSpace(*input.AccessToken))
		if err != nil {
			return nil, err
		}
	}
	if input.RefreshToken != nil {
		auth.RefreshToken, err = s.encryptSecret(strings.TrimSpace(*input.RefreshToken))
		if err != nil {
			return nil, err
		}
	}
	if auth.Metadata == nil {
		auth.Metadata = map[string]any{}
	}
	return auth, nil
}

func normalizeUpstreamAdminAuthMode(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

func isValidUpstreamAdminAuthMode(mode string) bool {
	switch mode {
	case UpstreamAdminAuthPassword, UpstreamAdminAuthToken, UpstreamAdminAuthNone:
		return true
	default:
		return false
	}
}

func (s *UpstreamService) persistPlainAdminAuth(ctx context.Context, auth *UpstreamAdminAuth) error {
	if auth == nil {
		return nil
	}
	encrypted := *auth
	var err error
	if encrypted.Username, err = s.encryptSecret(auth.Username); err != nil {
		return err
	}
	if encrypted.Password, err = s.encryptSecret(auth.Password); err != nil {
		return err
	}
	if encrypted.AccessToken, err = s.encryptSecret(auth.AccessToken); err != nil {
		return err
	}
	if encrypted.RefreshToken, err = s.encryptSecret(auth.RefreshToken); err != nil {
		return err
	}
	return s.repo.UpsertAdminAuth(ctx, &encrypted)
}

func (s *UpstreamService) decryptSecretsForUse(upstream *Upstream) error {
	if upstream == nil {
		return ErrUpstreamNotFound
	}
	var err error
	if upstream.ForwardCredential != nil {
		if upstream.ForwardCredential.APIKey, err = s.decryptSecret(upstream.ForwardCredential.APIKey); err != nil {
			upstream.ForwardCredential.DecryptFailed = true
			return fmt.Errorf("decrypt forwarding credential: %w", err)
		}
	}
	if upstream.AdminAuth != nil {
		auth := upstream.AdminAuth
		if auth.Username, err = s.decryptSecret(auth.Username); err != nil {
			auth.SecretDecryptFailed = true
			return fmt.Errorf("decrypt admin username: %w", err)
		}
		if auth.Password, err = s.decryptSecret(auth.Password); err != nil {
			auth.SecretDecryptFailed = true
			return fmt.Errorf("decrypt admin password: %w", err)
		}
		if auth.AccessToken, err = s.decryptSecret(auth.AccessToken); err != nil {
			auth.SecretDecryptFailed = true
			return fmt.Errorf("decrypt admin access token: %w", err)
		}
		if auth.RefreshToken, err = s.decryptSecret(auth.RefreshToken); err != nil {
			auth.SecretDecryptFailed = true
			return fmt.Errorf("decrypt admin refresh token: %w", err)
		}
	}
	return nil
}

func (s *UpstreamService) decryptRemoteAPIKeySecretsForUse(upstream *Upstream) error {
	if upstream == nil {
		return ErrUpstreamNotFound
	}
	for _, key := range upstream.RemoteAPIKeys {
		if key == nil {
			continue
		}
		plain, err := s.decryptSecret(key.APIKey)
		if err != nil {
			return fmt.Errorf("decrypt upstream API key %s: %w", key.RemoteAPIKeyID, err)
		}
		key.APIKey = plain
		key.APIKeyConfigured = strings.TrimSpace(plain) != ""
	}
	return nil
}

func (s *UpstreamService) maskSecretsForResponse(upstream *Upstream) {
	if upstream == nil {
		return
	}
	if upstream.ForwardCredential != nil {
		plain, err := s.decryptSecret(upstream.ForwardCredential.APIKey)
		if err != nil {
			upstream.ForwardCredential.DecryptFailed = true
			upstream.DecryptFailed = true
		}
		upstream.ForwardCredential.APIKeyMasked = maskSecret(plain)
		upstream.ForwardCredential.APIKey = ""
	}
	if upstream.AdminAuth != nil {
		auth := upstream.AdminAuth
		username, usernameErr := s.decryptSecret(auth.Username)
		password, passwordErr := s.decryptSecret(auth.Password)
		accessToken, accessErr := s.decryptSecret(auth.AccessToken)
		refreshToken, refreshErr := s.decryptSecret(auth.RefreshToken)
		if usernameErr != nil || passwordErr != nil || accessErr != nil || refreshErr != nil {
			auth.SecretDecryptFailed = true
			upstream.DecryptFailed = true
		}
		auth.UsernameMasked = maskSecret(username)
		auth.PasswordConfigured = strings.TrimSpace(password) != ""
		auth.AccessTokenMasked = maskSecret(accessToken)
		auth.RefreshTokenMasked = maskSecret(refreshToken)
		auth.Username = ""
		auth.Password = ""
		auth.AccessToken = ""
		auth.RefreshToken = ""
	}
	maskRemoteAPIKeySecretsForResponse(upstream.RemoteAPIKeys)
}

func maskRemoteAPIKeySecretsForResponse(keys []*UpstreamRemoteAPIKey) {
	for _, key := range keys {
		if key == nil {
			continue
		}
		key.APIKeyConfigured = strings.TrimSpace(key.APIKey) != ""
		key.APIKey = ""
	}
}

func (s *UpstreamService) encryptSecret(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	return s.encryptor.Encrypt(plain)
}

func (s *UpstreamService) decryptSecret(cipher string) (string, error) {
	if strings.TrimSpace(cipher) == "" {
		return "", nil
	}
	return s.encryptor.Decrypt(cipher)
}

func (s *UpstreamService) recordFailedSync(ctx context.Context, upstreamID int64, started time.Time, cause error) error {
	finished := time.Now().UTC()
	run := &UpstreamSyncRun{
		UpstreamID:  upstreamID,
		Status:      "failed",
		Message:     cause.Error(),
		StartedAt:   started,
		FinishedAt:  &finished,
		RawResult:   map[string]any{"error": cause.Error()},
		GroupsCount: 0,
	}
	return s.repo.ReplaceRemoteResources(ctx, upstreamID, nil, nil, run)
}

func (s *UpstreamService) disableRuntimeAccount(ctx context.Context, upstreamID int64, reason string) error {
	if s == nil || s.accountRepo == nil {
		return nil
	}
	account, err := s.findRuntimeAccount(ctx, upstreamID)
	if err != nil || account == nil {
		return err
	}
	account.Schedulable = false
	account.Status = StatusError
	account.ErrorMessage = reason
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	account.Extra["upstream_status"] = UpstreamStatusDisabled
	return s.accountRepo.Update(ctx, account)
}

func (s *UpstreamService) attachSchedulerSnapshot(upstream *Upstream) {
	if upstream == nil {
		return
	}
	upstream.SchedulerSnapshot = &UpstreamSchedulerSnapshot{
		HealthScore:      defaultScore(upstream.LatestHealthScore),
		PerformanceScore: 1,
		CostScore:        costScore(upstream.CostMultiplier),
		CapacityScore:    1,
	}
}

func normalizeUpstream(upstream *Upstream) error {
	if upstream == nil {
		return ErrUpstreamInvalidInput
	}
	if strings.TrimSpace(upstream.Name) == "" {
		return ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "name"})
	}
	if strings.TrimSpace(upstream.BaseURL) == "" {
		return ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "base_url"})
	}
	parsed, err := url.Parse(upstream.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "base_url"})
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "base_url"})
	}
	switch upstream.Type {
	case "", UpstreamTypeSub2API:
		upstream.Type = UpstreamTypeSub2API
	case UpstreamTypeNewAPI, UpstreamTypeOpenAICompatible, UpstreamTypeCustom:
	default:
		return ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "type"})
	}
	switch upstream.Status {
	case "":
		upstream.Status = UpstreamStatusActive
	case UpstreamStatusActive, UpstreamStatusDegraded, UpstreamStatusHalfOpen, UpstreamStatusCircuitOpen, UpstreamStatusDisabled:
	default:
		return ErrUpstreamInvalidInput.WithMetadata(map[string]string{"field": "status"})
	}
	upstream.RoutingMode = normalizeUpstreamRoutingMode(upstream.RoutingMode)
	if upstream.Priority <= 0 {
		upstream.Priority = 100
	}
	if upstream.Weight <= 0 {
		upstream.Weight = 100
	}
	if upstream.CostMultiplier <= 0 {
		upstream.CostMultiplier = 1
	}
	if upstream.TimeoutMS <= 0 {
		upstream.TimeoutMS = 60000
	}
	if upstream.ConnectTimeoutMS <= 0 {
		upstream.ConnectTimeoutMS = 10000
	}
	if upstream.RetryMax < 0 {
		upstream.RetryMax = 0
	}
	if upstream.ProbeIntervalSeconds <= 0 {
		upstream.ProbeIntervalSeconds = 60
	}
	return nil
}

func remoteGroupIDs(groups []*UpstreamRemoteGroup) []string {
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		if group == nil || strings.TrimSpace(group.RemoteGroupID) == "" {
			continue
		}
		out = append(out, group.RemoteGroupID)
	}
	return out
}

func remoteGroupMultiplier(groups []*UpstreamRemoteGroup, remoteGroupID string) float64 {
	if strings.TrimSpace(remoteGroupID) == "" {
		return 1
	}
	for _, group := range groups {
		if group == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(group.RemoteGroupID), strings.TrimSpace(remoteGroupID)) {
			if group.RateMultiplier > 0 {
				return group.RateMultiplier
			}
			return 1
		}
	}
	return 1
}
