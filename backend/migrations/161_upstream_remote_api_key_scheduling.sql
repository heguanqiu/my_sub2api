ALTER TABLE upstream_remote_api_keys
    ADD COLUMN IF NOT EXISTS scheduling_enabled BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_upstream_remote_api_keys_scheduling
    ON upstream_remote_api_keys (upstream_id, scheduling_enabled)
    WHERE scheduling_enabled = TRUE;

COMMENT ON COLUMN upstream_remote_api_keys.scheduling_enabled IS 'Local switch that controls whether the synced upstream API key can enter runtime scheduling.';
