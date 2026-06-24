ALTER TABLE upstream_remote_api_keys
    ADD COLUMN IF NOT EXISTS api_key_encrypted TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS synced_remote_group_id VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS local_group_ids BIGINT[] NOT NULL DEFAULT '{}';

UPDATE upstream_remote_api_keys
SET synced_remote_group_id = remote_group_id
WHERE synced_remote_group_id = '';

CREATE INDEX IF NOT EXISTS idx_upstream_remote_api_keys_local_groups
    ON upstream_remote_api_keys USING GIN (local_group_ids);
