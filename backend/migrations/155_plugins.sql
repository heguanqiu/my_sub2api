CREATE TABLE IF NOT EXISTS plugins (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version VARCHAR(50) NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT '',
    platform VARCHAR(50) NOT NULL DEFAULT 'all',
    icon_key VARCHAR(512) NOT NULL DEFAULT '',
    file_key VARCHAR(512) NOT NULL DEFAULT '',
    file_name VARCHAR(512) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    download_count BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    sort_weight INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT NULL,
    updated_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT plugins_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT plugins_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_plugins_status_sort
    ON plugins(status, sort_weight DESC, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_plugins_category
    ON plugins(category);
