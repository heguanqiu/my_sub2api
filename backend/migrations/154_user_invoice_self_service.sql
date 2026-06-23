CREATE TABLE IF NOT EXISTS invoice_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_type VARCHAR(20) NOT NULL DEFAULT 'personal',
    name VARCHAR(255) NOT NULL,
    tax_no VARCHAR(64) NOT NULL DEFAULT '',
    address_phone TEXT NOT NULL DEFAULT '',
    bank_account TEXT NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_invoice_profiles_user_id ON invoice_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_invoice_profiles_user_default ON invoice_profiles(user_id, is_default);
CREATE INDEX IF NOT EXISTS idx_invoice_profiles_deleted_at ON invoice_profiles(deleted_at);

CREATE TABLE IF NOT EXISTS invoice_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id BIGINT NULL REFERENCES invoice_profiles(id) ON DELETE SET NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    amount DECIMAL(20,2) NOT NULL,
    paid_total DECIMAL(20,2) NOT NULL DEFAULT 0,
    invoiced_total DECIMAL(20,2) NOT NULL DEFAULT 0,
    reserved_total DECIMAL(20,2) NOT NULL DEFAULT 0,
    available_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    title_type VARCHAR(20) NOT NULL DEFAULT 'personal',
    title_name VARCHAR(255) NOT NULL,
    tax_no VARCHAR(64) NOT NULL DEFAULT '',
    address_phone TEXT NOT NULL DEFAULT '',
    bank_account TEXT NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL,
    content VARCHAR(255) NOT NULL,
    remark TEXT NOT NULL DEFAULT '',
    sdk_code INTEGER NULL,
    sdk_message TEXT NOT NULL DEFAULT '',
    sdk_response JSONB NOT NULL DEFAULT '{}'::jsonb,
    invoice_no VARCHAR(64) NOT NULL DEFAULT '',
    invoice_date VARCHAR(32) NOT NULL DEFAULT '',
    pdf_url TEXT NOT NULL DEFAULT '',
    ofd_url TEXT NOT NULL DEFAULT '',
    xml_url TEXT NOT NULL DEFAULT '',
    issued_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_id ON invoice_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_profile_id ON invoice_requests(profile_id);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_status ON invoice_requests(status);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_created_at ON invoice_requests(created_at);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_status ON invoice_requests(user_id, status);

INSERT INTO settings (key, value)
VALUES
    ('payment_invoice_enabled', 'false'),
    ('payment_invoice_min_amount', '0'),
    ('payment_invoice_sdk_base_url', 'https://api.fa-piao.com'),
    ('payment_invoice_sdk_app_key', ''),
    ('payment_invoice_sdk_app_secret', ''),
    ('payment_invoice_taxpayer_id', ''),
    ('payment_invoice_seller_name', ''),
    ('payment_invoice_seller_address_phone', ''),
    ('payment_invoice_seller_bank_account', ''),
    ('payment_invoice_username', ''),
    ('payment_invoice_password', ''),
    ('payment_invoice_account_type', '6'),
    ('payment_invoice_debug', 'false'),
    ('payment_invoice_default_content', '*信息技术服务*软件开发服务'),
    ('payment_invoice_tax_rate', '0.01'),
    ('payment_invoice_tax_code', '3040201010000000000'),
    ('payment_invoice_type_code', '82')
ON CONFLICT (key) DO NOTHING;
