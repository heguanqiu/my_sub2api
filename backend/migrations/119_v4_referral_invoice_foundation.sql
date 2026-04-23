ALTER TABLE users
    ADD COLUMN IF NOT EXISTS invited_by_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS owner_sales_id BIGINT,
    ADD COLUMN IF NOT EXISTS first_paid_order_id BIGINT,
    ADD COLUMN IF NOT EXISTS first_paid_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_invited_by_user_id ON users(invited_by_user_id);
CREATE INDEX IF NOT EXISTS idx_users_owner_sales_id ON users(owner_sales_id);
CREATE INDEX IF NOT EXISTS idx_users_role_owner_sales_id ON users(role, owner_sales_id);

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS owner_sales_id_snapshot BIGINT,
    ADD COLUMN IF NOT EXISTS invited_by_user_id_snapshot BIGINT,
    ADD COLUMN IF NOT EXISTS invite_reward_status VARCHAR(30) NOT NULL DEFAULT 'not_applicable',
    ADD COLUMN IF NOT EXISTS invite_reward_ledger_id BIGINT,
    ADD COLUMN IF NOT EXISTS invoice_status VARCHAR(30) NOT NULL DEFAULT 'not_requested',
    ADD COLUMN IF NOT EXISTS invoice_request_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_payment_orders_owner_sales_id_snapshot ON payment_orders(owner_sales_id_snapshot);
CREATE INDEX IF NOT EXISTS idx_payment_orders_invited_by_user_id_snapshot ON payment_orders(invited_by_user_id_snapshot);
CREATE INDEX IF NOT EXISTS idx_payment_orders_invite_reward_status ON payment_orders(invite_reward_status);
CREATE INDEX IF NOT EXISTS idx_payment_orders_invoice_status ON payment_orders(invoice_status);
CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id_paid_at ON payment_orders(user_id, paid_at);

CREATE TABLE IF NOT EXISTS invite_links (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(128) NOT NULL UNIQUE,
    created_by_user_id BIGINT NOT NULL,
    creator_role VARCHAR(20) NOT NULL,
    owner_sales_id BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invite_links_created_by_user_id ON invite_links(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_invite_links_owner_sales_id ON invite_links(owner_sales_id);
CREATE INDEX IF NOT EXISTS idx_invite_links_status ON invite_links(status);

CREATE TABLE IF NOT EXISTS invite_reward_ledger (
    id BIGSERIAL PRIMARY KEY,
    inviter_user_id BIGINT NOT NULL,
    invitee_user_id BIGINT NOT NULL,
    trigger_order_id BIGINT NOT NULL,
    reward_type VARCHAR(20) NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL,
    status VARCHAR(30) NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    reversed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_invite_reward_ledger_inviter_user_id ON invite_reward_ledger(inviter_user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_invite_reward_ledger_invitee_user_id ON invite_reward_ledger(invitee_user_id);
CREATE INDEX IF NOT EXISTS idx_invite_reward_ledger_trigger_order_id ON invite_reward_ledger(trigger_order_id);
CREATE INDEX IF NOT EXISTS idx_invite_reward_ledger_status ON invite_reward_ledger(status);

CREATE TABLE IF NOT EXISTS invoice_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    tax_no VARCHAR(100),
    email VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    bank_name VARCHAR(255),
    bank_account VARCHAR(255),
    invoice_type VARCHAR(30) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_profiles_user_id ON invoice_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_invoice_profiles_user_id_is_default ON invoice_profiles(user_id, is_default);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_invoice_profiles_user_default ON invoice_profiles(user_id) WHERE is_default = TRUE;

CREATE TABLE IF NOT EXISTS invoice_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    profile_id BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'requested',
    provider VARCHAR(50) NOT NULL DEFAULT 'baiwang',
    provider_request_id VARCHAR(255),
    provider_invoice_id VARCHAR(255),
    fail_reason TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    issued_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_id ON invoice_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_order_id ON invoice_requests(order_id);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_status ON invoice_requests(status);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_provider ON invoice_requests(provider);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_invoice_requests_order_id_active
    ON invoice_requests(order_id)
    WHERE status IN ('requested', 'processing');

CREATE TABLE IF NOT EXISTS invoice_documents (
    id BIGSERIAL PRIMARY KEY,
    invoice_request_id BIGINT NOT NULL,
    invoice_no VARCHAR(100),
    invoice_code VARCHAR(100),
    file_url TEXT,
    file_type VARCHAR(20),
    raw_payload_summary JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_documents_invoice_request_id ON invoice_documents(invoice_request_id);
CREATE INDEX IF NOT EXISTS idx_invoice_documents_invoice_no ON invoice_documents(invoice_no);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_invited_by_user_id') THEN
        ALTER TABLE users
            ADD CONSTRAINT fk_users_invited_by_user_id
            FOREIGN KEY (invited_by_user_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_owner_sales_id') THEN
        ALTER TABLE users
            ADD CONSTRAINT fk_users_owner_sales_id
            FOREIGN KEY (owner_sales_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_first_paid_order_id') THEN
        ALTER TABLE users
            ADD CONSTRAINT fk_users_first_paid_order_id
            FOREIGN KEY (first_paid_order_id) REFERENCES payment_orders(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_payment_orders_owner_sales_id_snapshot') THEN
        ALTER TABLE payment_orders
            ADD CONSTRAINT fk_payment_orders_owner_sales_id_snapshot
            FOREIGN KEY (owner_sales_id_snapshot) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_payment_orders_invited_by_user_id_snapshot') THEN
        ALTER TABLE payment_orders
            ADD CONSTRAINT fk_payment_orders_invited_by_user_id_snapshot
            FOREIGN KEY (invited_by_user_id_snapshot) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_payment_orders_invite_reward_ledger_id') THEN
        ALTER TABLE payment_orders
            ADD CONSTRAINT fk_payment_orders_invite_reward_ledger_id
            FOREIGN KEY (invite_reward_ledger_id) REFERENCES invite_reward_ledger(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_payment_orders_invoice_request_id') THEN
        ALTER TABLE payment_orders
            ADD CONSTRAINT fk_payment_orders_invoice_request_id
            FOREIGN KEY (invoice_request_id) REFERENCES invoice_requests(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_links_created_by_user_id') THEN
        ALTER TABLE invite_links
            ADD CONSTRAINT fk_invite_links_created_by_user_id
            FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_links_owner_sales_id') THEN
        ALTER TABLE invite_links
            ADD CONSTRAINT fk_invite_links_owner_sales_id
            FOREIGN KEY (owner_sales_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_reward_ledger_inviter_user_id') THEN
        ALTER TABLE invite_reward_ledger
            ADD CONSTRAINT fk_invite_reward_ledger_inviter_user_id
            FOREIGN KEY (inviter_user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_reward_ledger_invitee_user_id') THEN
        ALTER TABLE invite_reward_ledger
            ADD CONSTRAINT fk_invite_reward_ledger_invitee_user_id
            FOREIGN KEY (invitee_user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invite_reward_ledger_trigger_order_id') THEN
        ALTER TABLE invite_reward_ledger
            ADD CONSTRAINT fk_invite_reward_ledger_trigger_order_id
            FOREIGN KEY (trigger_order_id) REFERENCES payment_orders(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invoice_profiles_user_id') THEN
        ALTER TABLE invoice_profiles
            ADD CONSTRAINT fk_invoice_profiles_user_id
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invoice_requests_user_id') THEN
        ALTER TABLE invoice_requests
            ADD CONSTRAINT fk_invoice_requests_user_id
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invoice_requests_order_id') THEN
        ALTER TABLE invoice_requests
            ADD CONSTRAINT fk_invoice_requests_order_id
            FOREIGN KEY (order_id) REFERENCES payment_orders(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invoice_requests_profile_id') THEN
        ALTER TABLE invoice_requests
            ADD CONSTRAINT fk_invoice_requests_profile_id
            FOREIGN KEY (profile_id) REFERENCES invoice_profiles(id) ON DELETE RESTRICT;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_invoice_documents_invoice_request_id') THEN
        ALTER TABLE invoice_documents
            ADD CONSTRAINT fk_invoice_documents_invoice_request_id
            FOREIGN KEY (invoice_request_id) REFERENCES invoice_requests(id) ON DELETE CASCADE;
    END IF;
END $$;
