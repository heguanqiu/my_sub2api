package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type upstreamRepository struct {
	db *sql.DB
}

func NewUpstreamRepository(db *sql.DB) service.UpstreamRepository {
	return &upstreamRepository{db: db}
}

func (r *upstreamRepository) GetRoutingMode(ctx context.Context) (string, error) {
	const q = `SELECT value FROM settings WHERE key = $1`
	var mode string
	err := r.db.QueryRowContext(ctx, q, service.UpstreamRoutingModeKey).Scan(&mode)
	if err == sql.ErrNoRows {
		return service.UpstreamRoutingBalanced, nil
	}
	if err != nil {
		return "", err
	}
	return mode, nil
}

func (r *upstreamRepository) SetRoutingMode(ctx context.Context, mode string) error {
	const q = `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, q, service.UpstreamRoutingModeKey, mode)
	return err
}

func (r *upstreamRepository) Create(ctx context.Context, upstream *service.Upstream) error {
	const q = `
		INSERT INTO upstreams (
			name, type, base_url, status, priority, weight, cost_multiplier,
			timeout_ms, connect_timeout_ms, retry_max, probe_enabled, probe_model,
			probe_interval_seconds, routing_mode, notes, metadata
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::jsonb)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, q,
		upstream.Name, upstream.Type, upstream.BaseURL, upstream.Status, upstream.Priority,
		upstream.Weight, upstream.CostMultiplier, upstream.TimeoutMS, upstream.ConnectTimeoutMS,
		upstream.RetryMax, upstream.ProbeEnabled, upstream.ProbeModel,
		upstream.ProbeIntervalSeconds, upstream.RoutingMode, upstream.Notes, jsonObject(upstream.Metadata),
	).Scan(&upstream.ID, &upstream.CreatedAt, &upstream.UpdatedAt)
	return translatePersistenceError(err, nil, service.ErrUpstreamExists)
}

func (r *upstreamRepository) Update(ctx context.Context, upstream *service.Upstream) error {
	const q = `
		UPDATE upstreams SET
			name = $2,
			type = $3,
			base_url = $4,
			status = $5,
			priority = $6,
			weight = $7,
			cost_multiplier = $8,
			timeout_ms = $9,
			connect_timeout_ms = $10,
			retry_max = $11,
			probe_enabled = $12,
			probe_model = $13,
			probe_interval_seconds = $14,
			routing_mode = $15,
			notes = $16,
			metadata = $17::jsonb,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, q,
		upstream.ID, upstream.Name, upstream.Type, upstream.BaseURL, upstream.Status,
		upstream.Priority, upstream.Weight, upstream.CostMultiplier, upstream.TimeoutMS,
		upstream.ConnectTimeoutMS, upstream.RetryMax, upstream.ProbeEnabled,
		upstream.ProbeModel, upstream.ProbeIntervalSeconds, upstream.RoutingMode, upstream.Notes,
		jsonObject(upstream.Metadata),
	).Scan(&upstream.UpdatedAt)
	return translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
}

func (r *upstreamRepository) Delete(ctx context.Context, id int64) error {
	const q = `
		UPDATE upstreams
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil
	}
	return nil
}

func (r *upstreamRepository) Get(ctx context.Context, id int64) (*service.Upstream, error) {
	const q = `
		SELECT
			u.id, u.name, u.type, u.base_url, u.status, u.priority, u.weight,
			u.cost_multiplier, u.timeout_ms, u.connect_timeout_ms, u.retry_max,
			u.probe_enabled, u.probe_model, u.probe_interval_seconds, u.routing_mode,
			u.notes, u.last_synced_at, u.last_sync_status, u.last_sync_error,
			u.created_at, u.updated_at, u.deleted_at, u.metadata,
			COALESCE((SELECT COUNT(*) FROM upstream_remote_groups g WHERE g.upstream_id = u.id), 0),
			COALESCE((SELECT COUNT(*) FROM upstream_remote_api_keys k WHERE k.upstream_id = u.id), 0),
			COALESCE((SELECT h.health_score FROM upstream_health_snapshots h WHERE h.upstream_id = u.id ORDER BY h.computed_at DESC LIMIT 1), 1)
		FROM upstreams u
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	upstream, err := scanUpstream(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	credential, err := r.getForwardCredential(ctx, id)
	if err != nil {
		return nil, err
	}
	upstream.ForwardCredential = credential
	auth, err := r.getAdminAuth(ctx, id)
	if err != nil {
		return nil, err
	}
	upstream.AdminAuth = auth
	upstream.RemoteGroups, _ = r.ListRemoteGroups(ctx, id)
	upstream.RemoteAPIKeys, _ = r.ListRemoteAPIKeys(ctx, id)
	upstream.LatestSyncRun, _ = r.LatestSyncRun(ctx, id)
	return upstream, nil
}

func (r *upstreamRepository) List(ctx context.Context, params service.UpstreamListParams) ([]*service.Upstream, int64, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	where, args := buildUpstreamListWhere(params)
	countQuery := "SELECT COUNT(*) FROM upstreams u " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count upstreams: %w", err)
	}

	args = append(args, pageSize, (page-1)*pageSize)
	query := `
		SELECT
			u.id, u.name, u.type, u.base_url, u.status, u.priority, u.weight,
			u.cost_multiplier, u.timeout_ms, u.connect_timeout_ms, u.retry_max,
			u.probe_enabled, u.probe_model, u.probe_interval_seconds, u.routing_mode,
			u.notes, u.last_synced_at, u.last_sync_status, u.last_sync_error,
			u.created_at, u.updated_at, u.deleted_at, u.metadata,
			COALESCE((SELECT COUNT(*) FROM upstream_remote_groups g WHERE g.upstream_id = u.id), 0),
			COALESCE((SELECT COUNT(*) FROM upstream_remote_api_keys k WHERE k.upstream_id = u.id), 0),
			COALESCE((SELECT h.health_score FROM upstream_health_snapshots h WHERE h.upstream_id = u.id ORDER BY h.computed_at DESC LIMIT 1), 1)
		FROM upstreams u ` + where + `
		ORDER BY u.priority ASC, u.id DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list upstreams: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.Upstream, 0)
	for rows.Next() {
		item, err := scanUpstream(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func buildUpstreamListWhere(params service.UpstreamListParams) (string, []any) {
	clauses := []string{"u.deleted_at IS NULL"}
	args := make([]any, 0)
	if v := strings.TrimSpace(params.Type); v != "" {
		args = append(args, v)
		clauses = append(clauses, fmt.Sprintf("u.type = $%d", len(args)))
	}
	if v := strings.TrimSpace(params.Status); v != "" {
		args = append(args, v)
		clauses = append(clauses, fmt.Sprintf("u.status = $%d", len(args)))
	}
	if v := strings.TrimSpace(params.Search); v != "" {
		args = append(args, "%"+v+"%")
		clauses = append(clauses, fmt.Sprintf("(u.name ILIKE $%d OR u.base_url ILIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func (r *upstreamRepository) UpsertForwardCredential(ctx context.Context, credential *service.UpstreamForwardCredential) error {
	metadata := jsonObject(credential.Metadata)
	const q = `
		INSERT INTO upstream_forward_credentials (
			upstream_id, name, auth_type, api_key_encrypted, enabled, expires_at, metadata
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)
		ON CONFLICT (upstream_id, name) DO UPDATE SET
			auth_type = EXCLUDED.auth_type,
			api_key_encrypted = EXCLUDED.api_key_encrypted,
			enabled = EXCLUDED.enabled,
			expires_at = EXCLUDED.expires_at,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, q,
		credential.UpstreamID, credential.Name, credential.AuthType, credential.APIKey,
		credential.Enabled, credential.ExpiresAt, metadata,
	).Scan(&credential.ID, &credential.CreatedAt, &credential.UpdatedAt)
	return err
}

func (r *upstreamRepository) UpsertAdminAuth(ctx context.Context, auth *service.UpstreamAdminAuth) error {
	metadata := jsonObject(auth.Metadata)
	const q = `
		INSERT INTO upstream_admin_auth (
			upstream_id, auth_mode, login_url, username_encrypted, password_encrypted,
			access_token_encrypted, refresh_token_encrypted, token_expires_at,
			last_login_at, last_login_error, metadata
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)
		ON CONFLICT (upstream_id) DO UPDATE SET
			auth_mode = EXCLUDED.auth_mode,
			login_url = EXCLUDED.login_url,
			username_encrypted = EXCLUDED.username_encrypted,
			password_encrypted = EXCLUDED.password_encrypted,
			access_token_encrypted = EXCLUDED.access_token_encrypted,
			refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
			token_expires_at = EXCLUDED.token_expires_at,
			last_login_at = EXCLUDED.last_login_at,
			last_login_error = EXCLUDED.last_login_error,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, q,
		auth.UpstreamID, auth.AuthMode, auth.LoginURL, auth.Username, auth.Password,
		auth.AccessToken, auth.RefreshToken, auth.TokenExpiresAt, auth.LastLoginAt,
		auth.LastLoginError, metadata,
	).Scan(&auth.CreatedAt, &auth.UpdatedAt)
	return err
}

func (r *upstreamRepository) ReplaceRemoteResources(ctx context.Context, upstreamID int64, groups []*service.UpstreamRemoteGroup, keys []*service.UpstreamRemoteAPIKey, run *service.UpstreamSyncRun) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	if run == nil {
		run = &service.UpstreamSyncRun{}
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	if run.FinishedAt == nil {
		finished := now
		run.FinishedAt = &finished
	}
	run.UpstreamID = upstreamID
	run.GroupsCount = len(groups)
	run.APIKeysCount = len(keys)
	if strings.TrimSpace(run.Status) == "" {
		run.Status = "success"
	}

	raw := jsonObject(run.RawResult)
	err = tx.QueryRowContext(ctx, `
		INSERT INTO upstream_sync_runs (upstream_id, status, groups_count, api_keys_count, message, started_at, finished_at, raw_result)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)
		RETURNING id
	`, upstreamID, run.Status, run.GroupsCount, run.APIKeysCount, run.Message, run.StartedAt, run.FinishedAt, raw).Scan(&run.ID)
	if err != nil {
		return err
	}

	if run.Status != "failed" {
		if _, err := tx.ExecContext(ctx, "DELETE FROM upstream_remote_groups WHERE upstream_id = $1", upstreamID); err != nil {
			return err
		}
		for _, group := range groups {
			if group.LastSyncedAt.IsZero() {
				group.LastSyncedAt = now
			}
			err := tx.QueryRowContext(ctx, `
				INSERT INTO upstream_remote_groups (upstream_id, remote_group_id, remote_group_name, rate_multiplier, status, raw_snapshot, last_synced_at)
				VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7)
				RETURNING id, created_at, updated_at
			`, upstreamID, group.RemoteGroupID, group.RemoteGroupName, group.RateMultiplier, group.Status, jsonObject(group.RawSnapshot), group.LastSyncedAt).
				Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
			if err != nil {
				return err
			}
			group.UpstreamID = upstreamID
		}
		for _, key := range keys {
			if key == nil {
				continue
			}
			if key.LastSyncedAt.IsZero() {
				key.LastSyncedAt = now
			}
			if strings.TrimSpace(key.SyncedRemoteGroupID) == "" {
				key.SyncedRemoteGroupID = key.RemoteGroupID
			}
			if strings.TrimSpace(key.RemoteGroupID) == "" {
				key.RemoteGroupID = key.SyncedRemoteGroupID
			}
			err := tx.QueryRowContext(ctx, `
				INSERT INTO upstream_remote_api_keys (
					upstream_id, remote_api_key_id, remote_api_key_name, api_key_encrypted,
					masked_key, synced_remote_group_id, remote_group_id, local_group_ids,
					status, quota, used_quota, raw_snapshot, last_synced_at
				)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13)
				ON CONFLICT (upstream_id, remote_api_key_id) DO UPDATE SET
					remote_api_key_name = EXCLUDED.remote_api_key_name,
					api_key_encrypted = CASE
						WHEN btrim(EXCLUDED.api_key_encrypted) <> '' THEN EXCLUDED.api_key_encrypted
						ELSE upstream_remote_api_keys.api_key_encrypted
					END,
					masked_key = CASE
						WHEN btrim(EXCLUDED.masked_key) <> '' THEN EXCLUDED.masked_key
						ELSE upstream_remote_api_keys.masked_key
					END,
					synced_remote_group_id = EXCLUDED.synced_remote_group_id,
					remote_group_id = CASE
						WHEN btrim(upstream_remote_api_keys.remote_group_id) = '' THEN EXCLUDED.remote_group_id
						ELSE upstream_remote_api_keys.remote_group_id
					END,
					status = EXCLUDED.status,
					quota = EXCLUDED.quota,
					used_quota = EXCLUDED.used_quota,
					raw_snapshot = EXCLUDED.raw_snapshot,
					last_synced_at = EXCLUDED.last_synced_at,
					updated_at = NOW()
				RETURNING id, api_key_encrypted, synced_remote_group_id, remote_group_id, local_group_ids,
				          scheduling_enabled, created_at, updated_at
			`, upstreamID, key.RemoteAPIKeyID, key.RemoteAPIKeyName, key.APIKey,
				key.MaskedKey, key.SyncedRemoteGroupID, key.RemoteGroupID, pq.Array(uniquePositiveInt64sRepo(key.LocalGroupIDs)),
				key.Status, key.Quota, key.UsedQuota, jsonObject(key.RawSnapshot), key.LastSyncedAt).
				Scan(&key.ID, &key.APIKey, &key.SyncedRemoteGroupID, &key.RemoteGroupID, pq.Array(&key.LocalGroupIDs),
					&keySchedulingEnabledScanner{key: key}, &key.CreatedAt, &key.UpdatedAt)
			if err != nil {
				return err
			}
			key.UpstreamID = upstreamID
			key.APIKeyConfigured = strings.TrimSpace(key.APIKey) != ""
		}
		if keys != nil {
			syncedKeyIDs := make([]string, 0, len(keys))
			for _, key := range keys {
				if key == nil {
					continue
				}
				if id := strings.TrimSpace(key.RemoteAPIKeyID); id != "" {
					syncedKeyIDs = append(syncedKeyIDs, id)
				}
			}
			syncedKeyIDs = uniqueStrings(syncedKeyIDs)
			var err error
			if len(syncedKeyIDs) == 0 {
				_, err = tx.ExecContext(ctx, `
					DELETE FROM upstream_remote_api_keys
					WHERE upstream_id = $1
				`, upstreamID)
			} else {
				_, err = tx.ExecContext(ctx, `
					DELETE FROM upstream_remote_api_keys
					WHERE upstream_id = $1
					  AND remote_api_key_id <> ALL($2::text[])
				`, upstreamID, pq.Array(syncedKeyIDs))
			}
			if err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE upstreams
		SET last_synced_at = $2, last_sync_status = $3, last_sync_error = $4, updated_at = NOW()
		WHERE id = $1
	`, upstreamID, run.FinishedAt, run.Status, run.Message); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *upstreamRepository) ListRemoteGroups(ctx context.Context, upstreamID int64) ([]*service.UpstreamRemoteGroup, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, upstream_id, remote_group_id, remote_group_name, rate_multiplier, status, raw_snapshot, last_synced_at, created_at, updated_at
		FROM upstream_remote_groups
		WHERE upstream_id = $1
		ORDER BY remote_group_name ASC
	`, upstreamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.UpstreamRemoteGroup, 0)
	for rows.Next() {
		row, err := scanRemoteGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *upstreamRepository) ListRemoteAPIKeys(ctx context.Context, upstreamID int64) ([]*service.UpstreamRemoteAPIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, upstream_id, remote_api_key_id, remote_api_key_name,
		       api_key_encrypted, masked_key, synced_remote_group_id, remote_group_id,
		       local_group_ids, scheduling_enabled, status, quota, used_quota, raw_snapshot, last_synced_at, created_at, updated_at
		FROM upstream_remote_api_keys
		WHERE upstream_id = $1
		ORDER BY remote_api_key_name ASC
	`, upstreamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.UpstreamRemoteAPIKey, 0)
	for rows.Next() {
		row, err := scanRemoteAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *upstreamRepository) UpdateRemoteAPIKeyConfig(ctx context.Context, upstreamID int64, remoteAPIKeyID string, remoteGroupID string, localGroupIDs []int64, schedulingEnabled *bool, apiKeyEncrypted *string) (*service.UpstreamRemoteAPIKey, error) {
	const q = `
		UPDATE upstream_remote_api_keys
		SET remote_group_id = $3,
		    local_group_ids = $4,
		    scheduling_enabled = COALESCE($5::boolean, scheduling_enabled),
		    api_key_encrypted = CASE
		        WHEN $6::text IS NULL THEN api_key_encrypted
		        ELSE $6::text
		    END,
		    updated_at = NOW()
		WHERE upstream_id = $1 AND remote_api_key_id = $2
		RETURNING id, upstream_id, remote_api_key_id, remote_api_key_name,
		          api_key_encrypted, masked_key, synced_remote_group_id, remote_group_id,
		          local_group_ids, scheduling_enabled, status, quota, used_quota, raw_snapshot, last_synced_at, created_at, updated_at
	`
	key, err := scanRemoteAPIKey(r.db.QueryRowContext(ctx, q,
		upstreamID,
		strings.TrimSpace(remoteAPIKeyID),
		strings.TrimSpace(remoteGroupID),
		pq.Array(uniquePositiveInt64sRepo(localGroupIDs)),
		schedulingEnabled,
		apiKeyEncrypted,
	))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	return key, nil
}

func (r *upstreamRepository) ClearRemoteAPIKeyLocalConfig(ctx context.Context, upstreamID int64, remoteAPIKeyID string) error {
	const q = `
		UPDATE upstream_remote_api_keys
		SET api_key_encrypted = '',
		    masked_key = '',
		    local_group_ids = '{}',
		    updated_at = NOW()
		WHERE upstream_id = $1 AND remote_api_key_id = $2
	`
	result, err := r.db.ExecContext(ctx, q, upstreamID, strings.TrimSpace(remoteAPIKeyID))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUpstreamNotFound
	}
	return nil
}

func (r *upstreamRepository) RemoveLocalGroupMappings(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE upstream_remote_api_keys
		SET local_group_ids = array_remove(local_group_ids, $1::bigint),
		    updated_at = NOW()
		WHERE $1::bigint = ANY(local_group_ids)
	`, groupID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE upstreams
		SET metadata =
			jsonb_set(
				jsonb_set(
					COALESCE(metadata, '{}'::jsonb),
					'{local_group_ids}',
					COALESCE((
						SELECT jsonb_agg(value)
						FROM jsonb_array_elements(
							CASE
								WHEN jsonb_typeof(metadata->'local_group_ids') = 'array' THEN metadata->'local_group_ids'
								ELSE '[]'::jsonb
							END
						) AS elem(value)
						WHERE value <> to_jsonb($1::bigint)
					), '[]'::jsonb),
					true
				),
				'{local_group_remote_group_ids}',
				CASE
					WHEN jsonb_typeof(metadata->'local_group_remote_group_ids') = 'object' THEN metadata->'local_group_remote_group_ids'
					ELSE '{}'::jsonb
				END - $1::text,
				true
			),
		    updated_at = NOW()
		WHERE deleted_at IS NULL
		  AND (
		    CASE
		      WHEN jsonb_typeof(metadata->'local_group_ids') = 'array' THEN metadata->'local_group_ids'
		      ELSE '[]'::jsonb
		    END @> jsonb_build_array($1::bigint)
		    OR CASE
		      WHEN jsonb_typeof(metadata->'local_group_remote_group_ids') = 'object' THEN metadata->'local_group_remote_group_ids'
		      ELSE '{}'::jsonb
		    END ? $1::text
		  )
	`, groupID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *upstreamRepository) LatestSyncRun(ctx context.Context, upstreamID int64) (*service.UpstreamSyncRun, error) {
	const q = `
		SELECT id, upstream_id, status, groups_count, api_keys_count, message, started_at, finished_at, raw_result
		FROM upstream_sync_runs
		WHERE upstream_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`
	run, err := scanSyncRun(r.db.QueryRowContext(ctx, q, upstreamID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return run, err
}

func (r *upstreamRepository) RecordRuntimeEvent(ctx context.Context, event service.UpstreamRuntimeEvent) (*service.UpstreamSchedulerSnapshot, error) {
	if event.UpstreamID <= 0 {
		return nil, nil
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	if event.Confidence <= 0 {
		event.Confidence = 1
	}
	eventType := "runtime_success"
	if !event.Success {
		eventType = "runtime_error"
	}
	if event.Ignored {
		eventType = "runtime_ignored"
	}
	if strings.HasPrefix(event.Reason, "probe_") {
		if event.Ignored {
			eventType = "probe_ignored"
		} else if event.Success {
			eventType = "probe_success"
		} else {
			eventType = "probe_failure"
		}
	} else if event.StreamInterrupted {
		eventType = "stream_interrupted"
	}
	evidence := map[string]any{
		"account_id":         event.AccountID,
		"success":            event.Success,
		"ignored":            event.Ignored,
		"status_code":        event.StatusCode,
		"duration_ms":        event.DurationMs,
		"reason":             event.Reason,
		"error_message":      event.ErrorMessage,
		"remote_api_key_id":  event.RemoteAPIKeyID,
		"remote_group_id":    event.RemoteGroupID,
		"model":              event.Model,
		"stream_interrupted": event.StreamInterrupted,
		"retried":            event.Retried,
		"confidence":         event.Confidence,
	}
	if event.FirstTokenMs != nil {
		evidence["first_token_ms"] = *event.FirstTokenMs
	}
	if event.LocalGroupID != nil {
		evidence["local_group_id"] = *event.LocalGroupID
	}
	if event.UserID != nil {
		evidence["user_id"] = *event.UserID
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO upstream_events (
			upstream_id, event_type, reason, evidence, created_at,
			account_id, remote_api_key_id, remote_group_id, local_group_id,
			model, status_code, first_token_ms, duration_ms, user_id,
			stream_interrupted, retried, confidence
		)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`, event.UpstreamID, eventType, event.Reason, jsonObject(evidence), event.ObservedAt,
		upstreamNullInt64Value(event.AccountID), strings.TrimSpace(event.RemoteAPIKeyID), strings.TrimSpace(event.RemoteGroupID),
		event.LocalGroupID, strings.TrimSpace(event.Model), event.StatusCode, event.FirstTokenMs, event.DurationMs,
		event.UserID, event.StreamInterrupted, event.Retried, event.Confidence); err != nil {
		return nil, err
	}

	var out *service.UpstreamSchedulerSnapshot
	if !event.Ignored {
		snapshot, err := r.computeRuntimeSnapshot(ctx, tx, event)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO upstream_health_snapshots (
				upstream_id, window_seconds, success_count, error_count, timeout_count,
				ttft_p50_ms, ttft_p90_ms, ttft_p99_ms, health_score, performance_score,
				capacity_score, computed_at
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`, event.UpstreamID, snapshot.windowSeconds, snapshot.successCount, snapshot.errorCount,
			snapshot.timeoutCount, snapshot.ttftP50, snapshot.ttftP90, snapshot.ttftP99,
			snapshot.healthScore, snapshot.performanceScore, snapshot.capacityScore, event.ObservedAt); err != nil {
			return nil, err
		}
		out = &service.UpstreamSchedulerSnapshot{
			HealthScore:      snapshot.healthScore,
			PerformanceScore: snapshot.performanceScore,
			CostScore:        1,
			CapacityScore:    snapshot.capacityScore,
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

type upstreamRuntimeSnapshot struct {
	windowSeconds    int
	successCount     int
	errorCount       int
	timeoutCount     int
	ttftP50          *int
	ttftP90          *int
	ttftP99          *int
	healthScore      float64
	performanceScore float64
	capacityScore    float64
}

func (r *upstreamRepository) computeRuntimeSnapshot(ctx context.Context, tx *sql.Tx, event service.UpstreamRuntimeEvent) (*upstreamRuntimeSnapshot, error) {
	const windowSeconds = 300
	snapshot := &upstreamRuntimeSnapshot{
		windowSeconds:    windowSeconds,
		healthScore:      1,
		performanceScore: 1,
		capacityScore:    1,
	}
	if err := tx.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE event_type = 'runtime_success'),
			COUNT(*) FILTER (WHERE event_type = 'runtime_error'),
			COUNT(*) FILTER (
				WHERE event_type = 'runtime_error'
				  AND (
					COALESCE(evidence->>'reason', '') ILIKE '%timeout%'
					OR COALESCE(evidence->>'error_message', '') ILIKE '%timeout%'
				  )
			)
		FROM upstream_events
		WHERE upstream_id = $1 AND created_at >= $2
	`, event.UpstreamID, event.ObservedAt.Add(-windowSeconds*time.Second)).
		Scan(&snapshot.successCount, &snapshot.errorCount, &snapshot.timeoutCount); err != nil {
		return nil, err
	}

	var prevHealth, prevPerformance, prevCapacity sql.NullFloat64
	if err := tx.QueryRowContext(ctx, `
		SELECT health_score, performance_score, capacity_score
		FROM upstream_health_snapshots
		WHERE upstream_id = $1
		ORDER BY computed_at DESC
		LIMIT 1
	`, event.UpstreamID).Scan(&prevHealth, &prevPerformance, &prevCapacity); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if prevHealth.Valid {
		snapshot.healthScore = prevHealth.Float64
	}
	if prevPerformance.Valid {
		snapshot.performanceScore = prevPerformance.Float64
	}
	if prevCapacity.Valid {
		snapshot.capacityScore = prevCapacity.Float64
	}

	total := snapshot.successCount + snapshot.errorCount
	successRatio := 1.0
	if total > 0 {
		successRatio = float64(snapshot.successCount) / float64(total)
	}
	instantHealth := successRatio
	if !event.Success {
		instantHealth = 0
	}
	snapshot.healthScore = clampRepoScore(snapshot.healthScore*0.75 + instantHealth*0.25)

	instantPerformance := snapshot.performanceScore
	if event.Success {
		instantPerformance = firstTokenPerformanceScore(event.FirstTokenMs, event.DurationMs)
	}
	snapshot.performanceScore = clampRepoScore(snapshot.performanceScore*0.8 + instantPerformance*0.2)

	instantCapacity := 1.0
	if !event.Success {
		instantCapacity = 0.65
	}
	snapshot.capacityScore = clampRepoScore(snapshot.capacityScore*0.85 + instantCapacity*0.15)

	if event.FirstTokenMs != nil {
		v := *event.FirstTokenMs
		snapshot.ttftP50 = &v
		snapshot.ttftP90 = &v
		snapshot.ttftP99 = &v
	}
	return snapshot, nil
}

func (r *upstreamRepository) GetHealthDashboard(ctx context.Context, upstreamID int64) (*service.UpstreamHealthDashboard, error) {
	upstream, err := r.Get(ctx, upstreamID)
	if err != nil {
		return nil, err
	}
	dashboard := &service.UpstreamHealthDashboard{
		UpstreamID: upstreamID,
		Status:     upstream.Status,
		SchedulerSnapshot: service.UpstreamSchedulerSnapshot{
			HealthScore:      defaultRepoScore(upstream.LatestHealthScore),
			PerformanceScore: 1,
			CostScore:        costRepoScore(upstream.CostMultiplier),
			CapacityScore:    1,
		},
		Degraded:    upstream.Status == service.UpstreamStatusDegraded,
		CircuitOpen: upstream.Status == service.UpstreamStatusCircuitOpen,
		Recovering:  upstream.Status == service.UpstreamStatusHalfOpen,
	}
	var latestProbeTTFT sql.NullInt64
	var latestProbeAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			event_type,
			first_token_ms,
			created_at
		FROM upstream_events
		WHERE upstream_id = $1 AND event_type IN ('probe_success','probe_failure','probe_ignored')
		ORDER BY created_at DESC
		LIMIT 1
	`, upstreamID).Scan(&dashboard.LatestProbeStatus, &latestProbeTTFT, &latestProbeAt); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if latestProbeTTFT.Valid {
		v := int(latestProbeTTFT.Int64)
		dashboard.LatestProbeFirstTokenMS = &v
	}
	if latestProbeAt.Valid {
		v := latestProbeAt.Time
		dashboard.LatestProbeCheckedAt = &v
	}
	if strings.TrimSpace(dashboard.LatestProbeStatus) == "" {
		dashboard.LatestProbeStatus = "unknown"
	}
	var recentErrorAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
		SELECT reason, created_at
		FROM upstream_events
		WHERE upstream_id = $1
		  AND event_type IN ('runtime_error','probe_failure','stream_interrupted')
		ORDER BY created_at DESC
		LIMIT 1
	`, upstreamID).Scan(&dashboard.RecentErrorReason, &recentErrorAt); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if recentErrorAt.Valid {
		v := recentErrorAt.Time
		dashboard.RecentErrorAt = &v
	}
	windows := []int{3600, 21600, 86400}
	dashboard.Windows = make([]service.UpstreamHealthWindow, 0, len(windows))
	for _, seconds := range windows {
		win, err := r.healthWindow(ctx, upstreamID, seconds)
		if err != nil {
			return nil, err
		}
		dashboard.Windows = append(dashboard.Windows, *win)
	}
	if len(dashboard.Windows) > 0 {
		latest := dashboard.Windows[0]
		if latest.SuccessCount+latest.ErrorCount > 0 {
			dashboard.SchedulerSnapshot.HealthScore = latest.SuccessRate
		}
		if latest.TTFTP95MS != nil {
			dashboard.SchedulerSnapshot.PerformanceScore = firstTokenPerformanceScore(latest.TTFTP95MS, 0)
		}
	}
	dashboard.SchedulableAPIKeys = countSchedulableRemoteAPIKeys(upstream.RemoteGroups, upstream.RemoteAPIKeys)
	dashboard.ServableLocalGroups = countServableLocalGroups(upstream.RemoteGroups, upstream.RemoteAPIKeys)
	alerts, err := r.ListAlerts(ctx, upstreamID, true)
	if err != nil {
		return nil, err
	}
	dashboard.ActiveAlerts = alerts
	attribution, err := r.attributionSignals(ctx, upstreamID)
	if err != nil {
		return nil, err
	}
	dashboard.Attribution = attribution
	return dashboard, nil
}

func (r *upstreamRepository) healthWindow(ctx context.Context, upstreamID int64, seconds int) (*service.UpstreamHealthWindow, error) {
	window := &service.UpstreamHealthWindow{WindowSeconds: seconds}
	start := time.Now().UTC().Add(-time.Duration(seconds) * time.Second)
	var p50, p90, p95, p99 sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE event_type IN ('runtime_success','probe_success')),
			COUNT(*) FILTER (WHERE event_type IN ('runtime_error','probe_failure','stream_interrupted')),
			percentile_disc(0.50) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL),
			percentile_disc(0.90) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL),
			percentile_disc(0.95) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL),
			percentile_disc(0.99) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL)
		FROM upstream_events
		WHERE upstream_id = $1
		  AND created_at >= $2
		  AND event_type <> 'runtime_ignored'
		  AND event_type <> 'probe_ignored'
	`, upstreamID, start).Scan(&window.SuccessCount, &window.ErrorCount, &p50, &p90, &p95, &p99); err != nil {
		return nil, err
	}
	window.TTFTP50MS = upstreamNullableIntPtr(p50)
	window.TTFTP90MS = upstreamNullableIntPtr(p90)
	window.TTFTP95MS = upstreamNullableIntPtr(p95)
	window.TTFTP99MS = upstreamNullableIntPtr(p99)
	total := window.SuccessCount + window.ErrorCount
	if total > 0 {
		window.SuccessRate = roundRepoFloat(float64(window.SuccessCount) / float64(total))
	} else {
		window.SuccessRate = 1
	}
	return window, nil
}

func (r *upstreamRepository) attributionSignals(ctx context.Context, upstreamID int64) ([]service.UpstreamAttributionSignal, error) {
	start := time.Now().UTC().Add(-time.Hour)
	out := make([]service.UpstreamAttributionSignal, 0)
	for _, spec := range []struct {
		scope string
		expr  string
	}{
		{scope: "api_key", expr: "remote_api_key_id"},
		{scope: "remote_group", expr: "remote_group_id"},
		{scope: "local_group", expr: "COALESCE(local_group_id::text, '')"},
	} {
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, total_count, error_count
			FROM (
				SELECT `+spec.expr+` AS id,
				       COUNT(*) AS total_count,
				       COUNT(*) FILTER (WHERE event_type IN ('runtime_error','probe_failure','stream_interrupted')) AS error_count
				FROM upstream_events
				WHERE upstream_id = $1 AND created_at >= $2
				GROUP BY `+spec.expr+`
			) s
			WHERE id <> '' AND total_count >= 2 AND error_count > 0
			ORDER BY error_count DESC, total_count DESC
			LIMIT 5
		`, upstreamID, start)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var item service.UpstreamAttributionSignal
			item.Scope = spec.scope
			if err := rows.Scan(&item.ID, &item.TotalCount, &item.ErrorCount); err != nil {
				_ = rows.Close()
				return nil, err
			}
			item.ErrorRate = roundRepoFloat(float64(item.ErrorCount) / float64(item.TotalCount))
			item.Confidence = roundRepoFloat(math.Min(1, (float64(item.TotalCount)/10)*item.ErrorRate))
			switch spec.scope {
			case "api_key":
				item.Suggestion = "degrade_or_disable_api_key"
			case "remote_group":
				item.Suggestion = "degrade_remote_group"
			default:
				item.Suggestion = "check_local_group_routing"
			}
			out = append(out, item)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return out, nil
}

func (r *upstreamRepository) ListEvents(ctx context.Context, upstreamID int64, limit int, eventType string) ([]service.UpstreamEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args := []any{upstreamID}
	where := "WHERE upstream_id = $1"
	if strings.TrimSpace(eventType) != "" {
		args = append(args, strings.TrimSpace(eventType))
		where += fmt.Sprintf(" AND event_type = $%d", len(args))
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, upstream_id, event_type, reason, account_id, remote_api_key_id,
		       remote_group_id, local_group_id, model, status_code, first_token_ms,
		       duration_ms, user_id, stream_interrupted, retried, confidence, evidence, created_at
		FROM upstream_events
		`+where+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.UpstreamEvent, 0, limit)
	for rows.Next() {
		item, err := scanUpstreamEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *upstreamRepository) CreateSyncPreview(ctx context.Context, preview *service.UpstreamSyncPreview) error {
	if preview == nil {
		return nil
	}
	if preview.CreatedAt.IsZero() {
		preview.CreatedAt = time.Now().UTC()
	}
	if preview.ExpiresAt.IsZero() {
		preview.ExpiresAt = preview.CreatedAt.Add(30 * time.Minute)
	}
	if strings.TrimSpace(preview.Status) == "" {
		preview.Status = "pending"
	}
	diffRaw, _ := json.Marshal(preview.Diff)
	groupsRaw, _ := json.Marshal(preview.Groups)
	keysRaw, _ := json.Marshal(preview.APIKeys)
	return r.db.QueryRowContext(ctx, `
		INSERT INTO upstream_sync_previews (
			upstream_id, preview_token, status, diff, groups_snapshot,
			api_keys_snapshot, created_at, expires_at
		)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7,$8)
		RETURNING id
	`, preview.UpstreamID, preview.PreviewToken, preview.Status, string(diffRaw), string(groupsRaw), string(keysRaw), preview.CreatedAt, preview.ExpiresAt).Scan(&preview.ID)
}

func (r *upstreamRepository) GetSyncPreview(ctx context.Context, upstreamID int64, token string) (*service.UpstreamSyncPreview, error) {
	preview := &service.UpstreamSyncPreview{}
	var rawDiff, rawGroups, rawKeys []byte
	var appliedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, upstream_id, preview_token, status, diff, groups_snapshot,
		       api_keys_snapshot, created_at, applied_at, expires_at
		FROM upstream_sync_previews
		WHERE upstream_id = $1 AND preview_token = $2
	`, upstreamID, strings.TrimSpace(token)).Scan(
		&preview.ID, &preview.UpstreamID, &preview.PreviewToken, &preview.Status,
		&rawDiff, &rawGroups, &rawKeys, &preview.CreatedAt, &appliedAt, &preview.ExpiresAt,
	)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	if appliedAt.Valid {
		preview.AppliedAt = &appliedAt.Time
	}
	_ = json.Unmarshal(rawDiff, &preview.Diff)
	_ = json.Unmarshal(rawGroups, &preview.Groups)
	_ = json.Unmarshal(rawKeys, &preview.APIKeys)
	return preview, nil
}

func (r *upstreamRepository) MarkSyncPreviewApplied(ctx context.Context, upstreamID int64, token string, appliedAt time.Time) error {
	if appliedAt.IsZero() {
		appliedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE upstream_sync_previews
		SET status = 'applied', applied_at = $3
		WHERE upstream_id = $1 AND preview_token = $2
	`, upstreamID, strings.TrimSpace(token), appliedAt)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUpstreamNotFound
	}
	return nil
}

func (r *upstreamRepository) ListAlerts(ctx context.Context, upstreamID int64, activeOnly bool) ([]service.UpstreamAlert, error) {
	args := []any{upstreamID}
	where := "WHERE (upstream_id = $1 OR upstream_id IS NULL)"
	if activeOnly {
		where += " AND status = 'active'"
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, upstream_id, alert_type, severity, status, title, message, evidence, created_at, resolved_at
		FROM upstream_alerts
		`+where+`
		ORDER BY created_at DESC
		LIMIT 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.UpstreamAlert, 0)
	for rows.Next() {
		item, err := scanUpstreamAlert(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *upstreamRepository) UpsertAlert(ctx context.Context, alert service.UpstreamAlert) error {
	if strings.TrimSpace(alert.Severity) == "" {
		alert.Severity = "warning"
	}
	if strings.TrimSpace(alert.Status) == "" {
		alert.Status = "active"
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
		UPDATE upstream_alerts
		SET severity = $3, title = $4, message = $5, evidence = $6::jsonb, created_at = $7
		WHERE COALESCE(upstream_id, 0) = COALESCE($1::bigint, 0)
		  AND alert_type = $2
		  AND status = 'active'
	`, alert.UpstreamID, alert.AlertType, alert.Severity, alert.Title, alert.Message, jsonObject(alert.Evidence), alert.CreatedAt)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO upstream_alerts (upstream_id, alert_type, severity, status, title, message, evidence, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8)
		`, alert.UpstreamID, alert.AlertType, alert.Severity, alert.Status, alert.Title, alert.Message, jsonObject(alert.Evidence), alert.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *upstreamRepository) ResolveAlert(ctx context.Context, upstreamID int64, alertType string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE upstream_alerts
		SET status = 'resolved', resolved_at = NOW()
		WHERE upstream_id = $1 AND alert_type = $2 AND status = 'active'
	`, upstreamID, strings.TrimSpace(alertType))
	return err
}

func (r *upstreamRepository) GetCostReport(ctx context.Context, upstreamID int64, start, end time.Time, dimension string) (*service.UpstreamCostReport, error) {
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start.IsZero() || !start.Before(end) {
		start = end.Add(-24 * time.Hour)
	}
	dimension = normalizeCostDimension(dimension)
	selectExpr, groupExpr, orderExpr := upstreamCostDimensionSQL(dimension)
	args := []any{upstreamID, start, end}
	costMultiplierExpr := upstreamCostMultiplierSQL()
	query := `
		SELECT ` + selectExpr + `,
		       COUNT(*) AS request_count,
		       COALESCE(SUM(ul.actual_cost), 0) AS local_billed_cost,
		       COALESCE(SUM(COALESCE(ul.account_stats_cost, ul.total_cost) * ` + costMultiplierExpr + `), 0) AS upstream_cost,
		       COALESCE(AVG(` + costMultiplierExpr + `), 1) AS avg_multiplier
		FROM usage_logs ul
		JOIN accounts a ON a.id = ul.account_id
		JOIN upstreams u ON (a.extra->>'upstream_id')::bigint = u.id
		LEFT JOIN upstream_remote_api_keys urak
		  ON urak.upstream_id = u.id
		 AND urak.remote_api_key_id = COALESCE(a.extra->>'upstream_remote_api_key_id', '')
		LEFT JOIN upstream_remote_groups urg
		  ON urg.upstream_id = u.id
		 AND urg.remote_group_id = ` + upstreamCostRemoteGroupSQL() + `
		WHERE (a.extra->>'upstream_runtime_managed') = 'true'
		  AND u.id = $1
		  AND ul.created_at >= $2 AND ul.created_at < $3
		` + groupExpr + `
		ORDER BY ` + orderExpr + `
		LIMIT 200
	`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	report := &service.UpstreamCostReport{
		Start:     start,
		End:       end,
		Dimension: dimension,
		Items:     make([]service.UpstreamCostDimension, 0),
	}
	for rows.Next() {
		item, err := scanCostDimension(rows, upstreamID, dimension)
		if err != nil {
			return nil, err
		}
		report.Items = append(report.Items, item)
		report.Totals.UpstreamID = upstreamID
		report.Totals.UpstreamName = item.UpstreamName
		report.Totals.RequestCount += item.RequestCount
		report.Totals.LocalBilledCost += item.LocalBilledCost
		report.Totals.UpstreamCost += item.UpstreamCost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	report.Totals.LocalBilledCost = roundRepoFloat(report.Totals.LocalBilledCost)
	report.Totals.UpstreamCost = roundRepoFloat(report.Totals.UpstreamCost)
	report.Totals.CostDelta = roundRepoFloat(report.Totals.UpstreamCost - report.Totals.LocalBilledCost)
	report.Totals.GrossProfit = roundRepoFloat(report.Totals.LocalBilledCost - report.Totals.UpstreamCost)
	return report, nil
}

func firstTokenPerformanceScore(firstTokenMs *int, durationMs int64) float64 {
	ms := 0
	if firstTokenMs != nil {
		ms = *firstTokenMs
	} else if durationMs > 0 {
		ms = int(durationMs)
	}
	if ms <= 0 {
		return 1
	}
	switch {
	case ms <= 1000:
		return 1
	case ms >= 12000:
		return 0.2
	default:
		return 1 - (float64(ms-1000) / 11000 * 0.8)
	}
}

func defaultRepoScore(v float64) float64 {
	if v == 0 {
		return 1
	}
	return clampRepoScore(v)
}

func costRepoScore(multiplier float64) float64 {
	if multiplier <= 0 {
		return 1
	}
	return clampRepoScore(1 / multiplier)
}

func clampRepoScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func roundRepoFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10000) / 10000
}

func countSchedulableRemoteAPIKeys(groups []*service.UpstreamRemoteGroup, keys []*service.UpstreamRemoteAPIKey) int {
	groupSet := activeRemoteGroupSet(groups)
	count := 0
	for _, key := range keys {
		if key == nil {
			continue
		}
		if key.SchedulingEnabled != nil && !*key.SchedulingEnabled {
			continue
		}
		if !remoteAPIKeyActiveRepo(key.Status) {
			continue
		}
		if strings.TrimSpace(key.APIKey) == "" && strings.TrimSpace(key.MaskedKey) == "" {
			continue
		}
		if !groupSet[strings.TrimSpace(key.RemoteGroupID)] {
			continue
		}
		if len(uniquePositiveInt64sRepo(key.LocalGroupIDs)) == 0 {
			continue
		}
		count++
	}
	return count
}

func countServableLocalGroups(groups []*service.UpstreamRemoteGroup, keys []*service.UpstreamRemoteAPIKey) int {
	groupSet := activeRemoteGroupSet(groups)
	localGroups := map[int64]struct{}{}
	for _, key := range keys {
		if key == nil || key.SchedulingEnabled != nil && !*key.SchedulingEnabled || !remoteAPIKeyActiveRepo(key.Status) || !groupSet[strings.TrimSpace(key.RemoteGroupID)] {
			continue
		}
		for _, id := range uniquePositiveInt64sRepo(key.LocalGroupIDs) {
			localGroups[id] = struct{}{}
		}
	}
	return len(localGroups)
}

func activeRemoteGroupSet(groups []*service.UpstreamRemoteGroup) map[string]bool {
	out := map[string]bool{}
	for _, group := range groups {
		if group == nil {
			continue
		}
		id := strings.TrimSpace(group.RemoteGroupID)
		if id == "" {
			continue
		}
		out[id] = remoteAPIKeyActiveRepo(group.Status)
	}
	return out
}

func remoteAPIKeyActiveRepo(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "", "active", "enabled", "enable", "1", "true":
		return true
	default:
		return false
	}
}

func upstreamNullableIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func upstreamNullableInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func upstreamNullableTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func upstreamNullInt64Value(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
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
	return out
}

func (r *upstreamRepository) getForwardCredential(ctx context.Context, upstreamID int64) (*service.UpstreamForwardCredential, error) {
	const q = `
		SELECT id, upstream_id, name, auth_type, api_key_encrypted, enabled, expires_at, metadata, created_at, updated_at
		FROM upstream_forward_credentials
		WHERE upstream_id = $1
		ORDER BY id ASC
		LIMIT 1
	`
	credential, err := scanForwardCredential(r.db.QueryRowContext(ctx, q, upstreamID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return credential, err
}

func (r *upstreamRepository) getAdminAuth(ctx context.Context, upstreamID int64) (*service.UpstreamAdminAuth, error) {
	const q = `
		SELECT upstream_id, auth_mode, login_url, username_encrypted, password_encrypted,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       last_login_at, last_login_error, metadata, created_at, updated_at
		FROM upstream_admin_auth
		WHERE upstream_id = $1
	`
	auth, err := scanAdminAuth(r.db.QueryRowContext(ctx, q, upstreamID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return auth, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUpstream(row scanner) (*service.Upstream, error) {
	u := &service.Upstream{}
	var lastSynced, deletedAt sql.NullTime
	var rawMetadata []byte
	err := row.Scan(
		&u.ID, &u.Name, &u.Type, &u.BaseURL, &u.Status, &u.Priority, &u.Weight,
		&u.CostMultiplier, &u.TimeoutMS, &u.ConnectTimeoutMS, &u.RetryMax,
		&u.ProbeEnabled, &u.ProbeModel, &u.ProbeIntervalSeconds, &u.RoutingMode,
		&u.Notes, &lastSynced, &u.LastSyncStatus, &u.LastSyncError,
		&u.CreatedAt, &u.UpdatedAt, &deletedAt, &rawMetadata,
		&u.GroupsCount, &u.APIKeysCount, &u.LatestHealthScore,
	)
	if err != nil {
		return nil, err
	}
	if lastSynced.Valid {
		u.LastSyncedAt = &lastSynced.Time
	}
	if deletedAt.Valid {
		u.DeletedAt = &deletedAt.Time
	}
	u.Metadata = decodeJSONMap(rawMetadata)
	return u, nil
}

func scanForwardCredential(row scanner) (*service.UpstreamForwardCredential, error) {
	c := &service.UpstreamForwardCredential{}
	var expiresAt sql.NullTime
	var raw []byte
	err := row.Scan(&c.ID, &c.UpstreamID, &c.Name, &c.AuthType, &c.APIKey, &c.Enabled, &expiresAt, &raw, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		c.ExpiresAt = &expiresAt.Time
	}
	c.Metadata = decodeJSONMap(raw)
	return c, nil
}

func scanAdminAuth(row scanner) (*service.UpstreamAdminAuth, error) {
	a := &service.UpstreamAdminAuth{}
	var tokenExpiresAt, lastLoginAt sql.NullTime
	var raw []byte
	err := row.Scan(
		&a.UpstreamID, &a.AuthMode, &a.LoginURL, &a.Username, &a.Password,
		&a.AccessToken, &a.RefreshToken, &tokenExpiresAt, &lastLoginAt,
		&a.LastLoginError, &raw, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tokenExpiresAt.Valid {
		a.TokenExpiresAt = &tokenExpiresAt.Time
	}
	if lastLoginAt.Valid {
		a.LastLoginAt = &lastLoginAt.Time
	}
	a.Metadata = decodeJSONMap(raw)
	return a, nil
}

func scanRemoteGroup(row scanner) (*service.UpstreamRemoteGroup, error) {
	g := &service.UpstreamRemoteGroup{}
	var raw []byte
	err := row.Scan(&g.ID, &g.UpstreamID, &g.RemoteGroupID, &g.RemoteGroupName, &g.RateMultiplier, &g.Status, &raw, &g.LastSyncedAt, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.RawSnapshot = decodeJSONMap(raw)
	return g, nil
}

func scanRemoteAPIKey(row scanner) (*service.UpstreamRemoteAPIKey, error) {
	k := &service.UpstreamRemoteAPIKey{}
	var quota, usedQuota sql.NullFloat64
	var schedulingEnabled sql.NullBool
	var raw []byte
	err := row.Scan(
		&k.ID, &k.UpstreamID, &k.RemoteAPIKeyID, &k.RemoteAPIKeyName,
		&k.APIKey, &k.MaskedKey, &k.SyncedRemoteGroupID, &k.RemoteGroupID,
		pq.Array(&k.LocalGroupIDs), &schedulingEnabled, &k.Status, &quota, &usedQuota,
		&raw, &k.LastSyncedAt, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	k.APIKeyConfigured = strings.TrimSpace(k.APIKey) != ""
	if schedulingEnabled.Valid {
		k.SchedulingEnabled = boolPtrRepo(schedulingEnabled.Bool)
	}
	if quota.Valid {
		v := quota.Float64
		k.Quota = &v
	}
	if usedQuota.Valid {
		v := usedQuota.Float64
		k.UsedQuota = &v
	}
	k.RawSnapshot = decodeJSONMap(raw)
	return k, nil
}

type keySchedulingEnabledScanner struct {
	key *service.UpstreamRemoteAPIKey
}

func (s *keySchedulingEnabledScanner) Scan(value any) error {
	if s == nil || s.key == nil {
		return nil
	}
	switch v := value.(type) {
	case bool:
		s.key.SchedulingEnabled = boolPtrRepo(v)
	case nil:
		s.key.SchedulingEnabled = boolPtrRepo(true)
	default:
		return fmt.Errorf("scan scheduling_enabled: unsupported type %T", value)
	}
	return nil
}

func boolPtrRepo(v bool) *bool {
	return &v
}

func scanSyncRun(row scanner) (*service.UpstreamSyncRun, error) {
	run := &service.UpstreamSyncRun{}
	var finishedAt sql.NullTime
	var raw []byte
	err := row.Scan(&run.ID, &run.UpstreamID, &run.Status, &run.GroupsCount, &run.APIKeysCount, &run.Message, &run.StartedAt, &finishedAt, &raw)
	if err != nil {
		return nil, err
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	run.RawResult = decodeJSONMap(raw)
	return run, nil
}

func scanUpstreamEvent(row scanner) (service.UpstreamEvent, error) {
	var item service.UpstreamEvent
	var accountID, localGroupID, userID, firstTokenMS sql.NullInt64
	var rawEvidence []byte
	err := row.Scan(
		&item.ID, &item.UpstreamID, &item.EventType, &item.Reason, &accountID,
		&item.RemoteAPIKeyID, &item.RemoteGroupID, &localGroupID, &item.Model,
		&item.StatusCode, &firstTokenMS, &item.DurationMS, &userID,
		&item.StreamInterrupted, &item.Retried, &item.Confidence, &rawEvidence,
		&item.CreatedAt,
	)
	if err != nil {
		return item, err
	}
	item.AccountID = upstreamNullableInt64Ptr(accountID)
	item.LocalGroupID = upstreamNullableInt64Ptr(localGroupID)
	item.UserID = upstreamNullableInt64Ptr(userID)
	item.FirstTokenMS = upstreamNullableIntPtr(firstTokenMS)
	item.Evidence = decodeJSONMap(rawEvidence)
	return item, nil
}

func scanUpstreamAlert(row scanner) (service.UpstreamAlert, error) {
	var item service.UpstreamAlert
	var upstreamID sql.NullInt64
	var rawEvidence []byte
	var resolvedAt sql.NullTime
	err := row.Scan(
		&item.ID, &upstreamID, &item.AlertType, &item.Severity, &item.Status,
		&item.Title, &item.Message, &rawEvidence, &item.CreatedAt, &resolvedAt,
	)
	if err != nil {
		return item, err
	}
	item.UpstreamID = upstreamNullableInt64Ptr(upstreamID)
	item.ResolvedAt = upstreamNullableTimePtr(resolvedAt)
	item.Evidence = decodeJSONMap(rawEvidence)
	return item, nil
}

func normalizeCostDimension(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "remote_group", "api_key", "local_group", "user", "model":
		return strings.TrimSpace(strings.ToLower(raw))
	default:
		return "upstream"
	}
}

func upstreamCostDimensionSQL(dimension string) (selectExpr, groupExpr, orderExpr string) {
	base := "u.id AS upstream_id, u.name AS upstream_name"
	switch normalizeCostDimension(dimension) {
	case "remote_group":
		remoteGroupExpr := upstreamCostRemoteGroupSQL()
		return base + ", " + remoteGroupExpr + " AS remote_group_id",
			"GROUP BY u.id, u.name, " + remoteGroupExpr,
			"upstream_cost DESC"
	case "api_key":
		return base + ", COALESCE(a.extra->>'upstream_remote_api_key_id', '') AS remote_api_key_id",
			"GROUP BY u.id, u.name, COALESCE(a.extra->>'upstream_remote_api_key_id', '')",
			"upstream_cost DESC"
	case "local_group":
		return base + ", ul.group_id AS local_group_id",
			"GROUP BY u.id, u.name, ul.group_id",
			"upstream_cost DESC"
	case "user":
		return base + ", ul.user_id AS user_id",
			"GROUP BY u.id, u.name, ul.user_id",
			"upstream_cost DESC"
	case "model":
		return base + ", COALESCE(ul.upstream_model, ul.model, '') AS model",
			"GROUP BY u.id, u.name, COALESCE(ul.upstream_model, ul.model, '')",
			"upstream_cost DESC"
	default:
		return base, "GROUP BY u.id, u.name", "upstream_cost DESC"
	}
}

func upstreamCostRemoteGroupSQL() string {
	return "COALESCE(NULLIF(urak.remote_group_id, ''), NULLIF(urak.synced_remote_group_id, ''), NULLIF(a.extra->>'upstream_remote_group_id', ''), '')"
}

func upstreamCostMultiplierSQL() string {
	return `CASE
			WHEN ul.account_rate_multiplier IS NOT NULL AND ul.account_rate_multiplier > 0 THEN ul.account_rate_multiplier
			WHEN urg.rate_multiplier IS NOT NULL AND urg.rate_multiplier > 0 THEN urg.rate_multiplier
			ELSE 1
		END`
}

func scanCostDimension(row scanner, upstreamID int64, dimension string) (service.UpstreamCostDimension, error) {
	item := service.UpstreamCostDimension{UpstreamID: upstreamID}
	var localGroupID, userID sql.NullInt64
	switch normalizeCostDimension(dimension) {
	case "remote_group":
		err := row.Scan(&item.UpstreamID, &item.UpstreamName, &item.RemoteGroupID, &item.RequestCount, &item.LocalBilledCost, &item.UpstreamCost, &item.AvgMultiplier)
		finishCostDimension(&item)
		return item, err
	case "api_key":
		err := row.Scan(&item.UpstreamID, &item.UpstreamName, &item.RemoteAPIKeyID, &item.RequestCount, &item.LocalBilledCost, &item.UpstreamCost, &item.AvgMultiplier)
		finishCostDimension(&item)
		return item, err
	case "local_group":
		err := row.Scan(&item.UpstreamID, &item.UpstreamName, &localGroupID, &item.RequestCount, &item.LocalBilledCost, &item.UpstreamCost, &item.AvgMultiplier)
		item.LocalGroupID = upstreamNullableInt64Ptr(localGroupID)
		finishCostDimension(&item)
		return item, err
	case "user":
		err := row.Scan(&item.UpstreamID, &item.UpstreamName, &userID, &item.RequestCount, &item.LocalBilledCost, &item.UpstreamCost, &item.AvgMultiplier)
		item.UserID = upstreamNullableInt64Ptr(userID)
		finishCostDimension(&item)
		return item, err
	case "model":
		err := row.Scan(&item.UpstreamID, &item.UpstreamName, &item.Model, &item.RequestCount, &item.LocalBilledCost, &item.UpstreamCost, &item.AvgMultiplier)
		finishCostDimension(&item)
		return item, err
	default:
		err := row.Scan(&item.UpstreamID, &item.UpstreamName, &item.RequestCount, &item.LocalBilledCost, &item.UpstreamCost, &item.AvgMultiplier)
		finishCostDimension(&item)
		return item, err
	}
}

func finishCostDimension(item *service.UpstreamCostDimension) {
	if item == nil {
		return
	}
	item.LocalBilledCost = roundRepoFloat(item.LocalBilledCost)
	item.UpstreamCost = roundRepoFloat(item.UpstreamCost)
	item.CostDelta = roundRepoFloat(item.UpstreamCost - item.LocalBilledCost)
	item.GrossProfit = roundRepoFloat(item.LocalBilledCost - item.UpstreamCost)
	item.AvgMultiplier = roundRepoFloat(item.AvgMultiplier)
}

func jsonObject(v map[string]any) string {
	if v == nil {
		v = map[string]any{}
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func uniquePositiveInt64sRepo(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
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

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}
