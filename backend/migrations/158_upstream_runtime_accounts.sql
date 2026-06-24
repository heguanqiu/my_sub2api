-- Runtime wiring for upstream management.
-- Metadata keeps routing/binding preferences in the upstream domain while a
-- hidden managed account lets the existing gateway, billing, and usage log
-- paths keep using accounts.id.

ALTER TABLE upstreams
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_upstreams_metadata_gin
    ON upstreams USING GIN (metadata);

CREATE INDEX IF NOT EXISTS idx_accounts_upstream_runtime
    ON accounts ((extra->>'upstream_runtime_managed'), (extra->>'upstream_id'))
    WHERE deleted_at IS NULL;
