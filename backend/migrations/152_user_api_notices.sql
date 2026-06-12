CREATE TABLE IF NOT EXISTS user_api_notices (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NULL,
    consumed_at TIMESTAMPTZ NULL,
    consumed_request_id VARCHAR(128) NULL,
    cancelled_at TIMESTAMPTZ NULL,
    cancelled_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT user_api_notices_status_check CHECK (status IN ('pending', 'consumed', 'cancelled')),
    CONSTRAINT user_api_notices_message_not_blank CHECK (length(btrim(message)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_user_api_notices_pending_user
    ON user_api_notices(user_id, created_at, id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_user_api_notices_user_created
    ON user_api_notices(user_id, created_at DESC, id DESC);
