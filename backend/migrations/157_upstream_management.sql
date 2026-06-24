-- Upstream management control-plane tables.
-- These tables intentionally stay separate from account pool tables: an
-- upstream is an external gateway/supplier node, not a self-owned account.

CREATE TABLE IF NOT EXISTS upstreams (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    type VARCHAR(40) NOT NULL DEFAULT 'sub2api',
    base_url TEXT NOT NULL,
    status VARCHAR(40) NOT NULL DEFAULT 'active',
    priority INTEGER NOT NULL DEFAULT 100,
    weight INTEGER NOT NULL DEFAULT 100,
    cost_multiplier NUMERIC(12, 6) NOT NULL DEFAULT 1,
    timeout_ms INTEGER NOT NULL DEFAULT 60000,
    connect_timeout_ms INTEGER NOT NULL DEFAULT 10000,
    retry_max INTEGER NOT NULL DEFAULT 1,
    probe_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    probe_model VARCHAR(200) NOT NULL DEFAULT '',
    probe_interval_seconds INTEGER NOT NULL DEFAULT 60,
    routing_mode VARCHAR(40) NOT NULL DEFAULT 'balanced',
    notes TEXT NOT NULL DEFAULT '',
    last_synced_at TIMESTAMPTZ NULL,
    last_sync_status VARCHAR(40) NOT NULL DEFAULT '',
    last_sync_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT upstreams_type_check CHECK (type IN ('sub2api','newapi','openai_compatible','custom')),
    CONSTRAINT upstreams_status_check CHECK (status IN ('active','degraded','half_open','circuit_open','disabled')),
    CONSTRAINT upstreams_routing_mode_check CHECK (routing_mode IN ('stability','balanced','cost','speed','manual')),
    CONSTRAINT upstreams_weight_check CHECK (weight >= 0),
    CONSTRAINT upstreams_cost_multiplier_check CHECK (cost_multiplier >= 0),
    CONSTRAINT upstreams_timeout_check CHECK (timeout_ms > 0 AND connect_timeout_ms > 0),
    CONSTRAINT upstreams_probe_interval_check CHECK (probe_interval_seconds > 0)
);

CREATE INDEX IF NOT EXISTS idx_upstreams_active_status
    ON upstreams (status, priority, weight)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_upstreams_deleted_at
    ON upstreams (deleted_at);

CREATE TABLE IF NOT EXISTS upstream_forward_credentials (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL DEFAULT 'default',
    auth_type VARCHAR(40) NOT NULL DEFAULT 'bearer',
    api_key_encrypted TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT upstream_forward_credentials_auth_type_check CHECK (auth_type IN ('bearer','openai','custom'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_forward_credentials_name
    ON upstream_forward_credentials (upstream_id, name);

CREATE TABLE IF NOT EXISTS upstream_admin_auth (
    upstream_id BIGINT PRIMARY KEY REFERENCES upstreams(id) ON DELETE CASCADE,
    auth_mode VARCHAR(40) NOT NULL DEFAULT 'password',
    login_url TEXT NOT NULL DEFAULT '',
    username_encrypted TEXT NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    access_token_encrypted TEXT NOT NULL DEFAULT '',
    refresh_token_encrypted TEXT NOT NULL DEFAULT '',
    token_expires_at TIMESTAMPTZ NULL,
    last_login_at TIMESTAMPTZ NULL,
    last_login_error TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT upstream_admin_auth_mode_check CHECK (auth_mode IN ('password','token','none'))
);

CREATE TABLE IF NOT EXISTS upstream_remote_groups (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    remote_group_id VARCHAR(200) NOT NULL,
    remote_group_name VARCHAR(200) NOT NULL,
    rate_multiplier NUMERIC(12, 6) NOT NULL DEFAULT 1,
    status VARCHAR(40) NOT NULL DEFAULT 'active',
    raw_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_remote_groups_unique
    ON upstream_remote_groups (upstream_id, remote_group_id);

CREATE INDEX IF NOT EXISTS idx_upstream_remote_groups_name
    ON upstream_remote_groups (upstream_id, remote_group_name);

CREATE TABLE IF NOT EXISTS upstream_remote_api_keys (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    remote_api_key_id VARCHAR(200) NOT NULL,
    remote_api_key_name VARCHAR(200) NOT NULL,
    masked_key VARCHAR(200) NOT NULL DEFAULT '',
    remote_group_id VARCHAR(200) NOT NULL DEFAULT '',
    status VARCHAR(40) NOT NULL DEFAULT 'active',
    quota NUMERIC(18, 6) NULL,
    used_quota NUMERIC(18, 6) NULL,
    raw_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_remote_api_keys_unique
    ON upstream_remote_api_keys (upstream_id, remote_api_key_id);

CREATE INDEX IF NOT EXISTS idx_upstream_remote_api_keys_group
    ON upstream_remote_api_keys (upstream_id, remote_group_id);

CREATE TABLE IF NOT EXISTS upstream_sync_runs (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    status VARCHAR(40) NOT NULL DEFAULT 'running',
    groups_count INTEGER NOT NULL DEFAULT 0,
    api_keys_count INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    raw_result JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT upstream_sync_runs_status_check CHECK (status IN ('running','success','failed','partial'))
);

CREATE INDEX IF NOT EXISTS idx_upstream_sync_runs_upstream_started
    ON upstream_sync_runs (upstream_id, started_at DESC);

CREATE TABLE IF NOT EXISTS upstream_probe_results (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    model VARCHAR(200) NOT NULL DEFAULT '',
    status VARCHAR(40) NOT NULL,
    ttft_ms INTEGER NULL,
    total_ms INTEGER NULL,
    error_code VARCHAR(120) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_probe_results_latest
    ON upstream_probe_results (upstream_id, checked_at DESC);

CREATE TABLE IF NOT EXISTS upstream_health_snapshots (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    window_seconds INTEGER NOT NULL,
    success_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    timeout_count INTEGER NOT NULL DEFAULT 0,
    ttft_p50_ms INTEGER NULL,
    ttft_p90_ms INTEGER NULL,
    ttft_p99_ms INTEGER NULL,
    health_score NUMERIC(8, 4) NOT NULL DEFAULT 1,
    performance_score NUMERIC(8, 4) NOT NULL DEFAULT 1,
    capacity_score NUMERIC(8, 4) NOT NULL DEFAULT 1,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_health_snapshots_latest
    ON upstream_health_snapshots (upstream_id, computed_at DESC);

CREATE TABLE IF NOT EXISTS upstream_events (
    id BIGSERIAL PRIMARY KEY,
    upstream_id BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    event_type VARCHAR(80) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_events_recent
    ON upstream_events (upstream_id, created_at DESC);
