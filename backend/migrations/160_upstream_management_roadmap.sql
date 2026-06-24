-- Upstream management roadmap additions: observability, sync preview,
-- governance policy, alerts, and cost reconciliation support.

ALTER TABLE upstream_events
    ADD COLUMN IF NOT EXISTS account_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS remote_api_key_id VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS remote_group_id VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS local_group_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS model VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status_code INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS first_token_ms INTEGER NULL,
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS user_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS stream_interrupted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS retried BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS confidence NUMERIC(8,4) NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_upstream_events_api_key_recent
    ON upstream_events (upstream_id, remote_api_key_id, created_at DESC)
    WHERE remote_api_key_id <> '';

CREATE INDEX IF NOT EXISTS idx_upstream_events_group_recent
    ON upstream_events (upstream_id, remote_group_id, created_at DESC)
    WHERE remote_group_id <> '';

CREATE INDEX IF NOT EXISTS idx_upstream_events_local_group_recent
    ON upstream_events (local_group_id, created_at DESC)
    WHERE local_group_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS upstream_sync_previews (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    preview_token VARCHAR(80) NOT NULL UNIQUE,
    status VARCHAR(40) NOT NULL DEFAULT 'pending',
    diff JSONB NOT NULL DEFAULT '{}'::jsonb,
    groups_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    api_keys_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 minutes'),
    CONSTRAINT upstream_sync_previews_status_check CHECK (status IN ('pending','applied','expired','failed'))
);

CREATE INDEX IF NOT EXISTS idx_upstream_sync_previews_latest
    ON upstream_sync_previews (upstream_id, created_at DESC);

CREATE TABLE IF NOT EXISTS upstream_alerts (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    alert_type VARCHAR(80) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ NULL,
    CONSTRAINT upstream_alerts_severity_check CHECK (severity IN ('info','warning','critical')),
    CONSTRAINT upstream_alerts_status_check CHECK (status IN ('active','resolved'))
);

CREATE INDEX IF NOT EXISTS idx_upstream_alerts_recent
    ON upstream_alerts (upstream_id, status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_alerts_active_dedup
    ON upstream_alerts (COALESCE(upstream_id, 0), alert_type)
    WHERE status = 'active';
