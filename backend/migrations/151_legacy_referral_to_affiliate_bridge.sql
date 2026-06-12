-- Bridge legacy local referral data into upstream Affiliate tables.
--
-- Legacy local referral rewards were credited directly to inviter balances via
-- invite_reward_ledger. Upstream Affiliate accrues quota first and lets users
-- transfer it later. For historical rows we therefore backfill relationship and
-- audit visibility, but intentionally keep aff_quota unchanged to avoid paying
-- the same reward twice.

-- 1) Enable upstream Affiliate when the legacy invite reward feature was enabled.
INSERT INTO settings (key, value, updated_at)
SELECT 'affiliate_enabled',
       CASE WHEN value = 'true' THEN 'true' ELSE 'false' END,
       NOW()
FROM settings
WHERE key = 'INVITE_REWARD_ENABLED'
ON CONFLICT (key) DO UPDATE
SET value = CASE
        WHEN settings.value = 'true' THEN 'true'
        WHEN EXCLUDED.value = 'true' THEN 'true'
        ELSE settings.value
    END,
    updated_at = NOW();

-- 2) Carry the legacy reward percentage into the upstream Affiliate setting
-- only when Affiliate does not already have a positive configured rate.
INSERT INTO settings (key, value, updated_at)
SELECT 'affiliate_rebate_rate', value, NOW()
FROM settings
WHERE key = 'INVITE_REWARD_RATE'
  AND value ~ '^-?[0-9]+([.][0-9]+)?$'
  AND value::numeric > 0
ON CONFLICT (key) DO UPDATE
SET value = CASE
        WHEN settings.value ~ '^-?[0-9]+([.][0-9]+)?$'
         AND settings.value::numeric > 0 THEN settings.value
        ELSE EXCLUDED.value
    END,
    updated_at = NOW();

INSERT INTO settings (key, value, updated_at)
VALUES
    ('affiliate_rebate_freeze_hours', '0', NOW()),
    ('affiliate_rebate_duration_days', '0', NOW()),
    ('affiliate_rebate_per_invitee_cap', '0.00000000', NOW())
ON CONFLICT (key) DO NOTHING;

-- 3) Ensure official Affiliate profiles exist for all users that participate in
-- legacy referral relationships or rewards. Deterministic LEGACY{id} codes are
-- only for rows that never generated an upstream code before.
INSERT INTO user_affiliates (
    user_id,
    aff_code,
    inviter_id,
    aff_count,
    aff_quota,
    aff_history_quota,
    aff_frozen_quota,
    created_at,
    updated_at
)
SELECT u.id,
       'LEGACY' || u.id::text,
       NULL,
       0,
       0,
       0,
       0,
       COALESCE(u.created_at, NOW()),
       NOW()
FROM users u
WHERE (
        u.invited_by_user_id IS NOT NULL
        OR EXISTS (SELECT 1 FROM users child WHERE child.invited_by_user_id = u.id)
        OR EXISTS (SELECT 1 FROM invite_reward_ledger irl WHERE irl.inviter_user_id = u.id OR irl.invitee_user_id = u.id)
    )
  AND NOT EXISTS (SELECT 1 FROM user_affiliates ua WHERE ua.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM user_affiliates ua WHERE ua.aff_code = 'LEGACY' || u.id::text)
ON CONFLICT (user_id) DO NOTHING;

-- 4) Backfill inviter bindings from the legacy users.invited_by_user_id column.
-- Do not overwrite a different non-null upstream binding if one already exists.
UPDATE user_affiliates ua
SET inviter_id = u.invited_by_user_id,
    created_at = LEAST(ua.created_at, COALESCE(u.created_at, ua.created_at)),
    updated_at = NOW()
FROM users u
WHERE ua.user_id = u.id
  AND u.invited_by_user_id IS NOT NULL
  AND u.invited_by_user_id <> u.id
  AND EXISTS (SELECT 1 FROM user_affiliates inviter WHERE inviter.user_id = u.invited_by_user_id)
  AND (ua.inviter_id IS NULL OR ua.inviter_id = u.invited_by_user_id)
  AND ua.inviter_id IS DISTINCT FROM u.invited_by_user_id;

-- 5) Recompute invite counts from official bindings.
WITH counts AS (
    SELECT inviter.user_id,
           COUNT(invitee.user_id)::integer AS invitee_count
    FROM user_affiliates inviter
    LEFT JOIN user_affiliates invitee ON invitee.inviter_id = inviter.user_id
    GROUP BY inviter.user_id
)
UPDATE user_affiliates ua
SET aff_count = counts.invitee_count,
    updated_at = NOW()
FROM counts
WHERE ua.user_id = counts.user_id
  AND ua.aff_count IS DISTINCT FROM counts.invitee_count;

-- 6) Backfill legacy granted rewards as upstream accrual ledger rows for audit
-- and reporting. Do not add them to aff_quota because they were already credited
-- directly to inviter balances by the legacy system.
INSERT INTO user_affiliate_ledger (
    user_id,
    action,
    amount,
    source_user_id,
    source_order_id,
    created_at,
    updated_at
)
SELECT irl.inviter_user_id,
       'accrue',
       irl.reward_amount,
       irl.invitee_user_id,
       irl.trigger_order_id,
       COALESCE(irl.confirmed_at, irl.created_at, NOW()),
       NOW()
FROM invite_reward_ledger irl
WHERE irl.status = 'granted'
  AND irl.reward_amount > 0
  AND EXISTS (SELECT 1 FROM user_affiliates ua WHERE ua.user_id = irl.inviter_user_id)
  AND EXISTS (SELECT 1 FROM user_affiliates ua WHERE ua.user_id = irl.invitee_user_id)
  AND NOT EXISTS (
      SELECT 1
      FROM user_affiliate_ledger ual
      WHERE ual.action = 'accrue'
        AND ual.source_order_id = irl.trigger_order_id
        AND ual.user_id = irl.inviter_user_id
        AND ual.source_user_id = irl.invitee_user_id
  );

-- 7) Make upstream historical quota reflect all known accrual ledger rows while
-- keeping available quota untouched.
WITH totals AS (
    SELECT user_id,
           COALESCE(SUM(amount), 0) AS accrued_total
    FROM user_affiliate_ledger
    WHERE action = 'accrue'
    GROUP BY user_id
)
UPDATE user_affiliates ua
SET aff_history_quota = GREATEST(ua.aff_history_quota, totals.accrued_total),
    updated_at = NOW()
FROM totals
WHERE ua.user_id = totals.user_id
  AND ua.aff_history_quota < totals.accrued_total;
