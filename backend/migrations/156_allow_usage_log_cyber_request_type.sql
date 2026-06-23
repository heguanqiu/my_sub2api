-- Allow cyber policy usage rows to be written with request_type=4.
-- Existing migration 061 allowed only 0..3 before RequestTypeCyberBlocked was added.
ALTER TABLE usage_logs
    DROP CONSTRAINT IF EXISTS usage_logs_request_type_check;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_request_type_check
    CHECK (request_type IN (0, 1, 2, 3, 4)) NOT VALID;
