-- Remove the local invoice system while preserving referral/invite reward tables.
-- The migration is idempotent so both fresh installs and upgraded instances can
-- pass through it safely.

ALTER TABLE IF EXISTS payment_orders
    DROP CONSTRAINT IF EXISTS fk_payment_orders_invoice_request_id;

DROP TABLE IF EXISTS invoice_documents;
DROP TABLE IF EXISTS invoice_requests;
DROP TABLE IF EXISTS invoice_profiles;

DROP INDEX IF EXISTS idx_payment_orders_invoice_status;

ALTER TABLE IF EXISTS payment_orders
    DROP COLUMN IF EXISTS invoice_request_id,
    DROP COLUMN IF EXISTS invoice_status;

DELETE FROM settings
WHERE key IN (
    'invoice_enabled',
    'invoice_provider',
    'invoice_baiwang_enabled',
    'invoice_baiwang_base_url',
    'invoice_baiwang_app_key',
    'invoice_baiwang_app_secret',
    'invoice_baiwang_taxpayer_id',
    'invoice_baiwang_seller_name',
    'invoice_baiwang_default_goods_name',
    'invoice_auto_retry_enabled',
    'invoice_retry_limit'
);
