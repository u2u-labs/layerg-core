-- +migrate Up
ALTER TABLE subscription
    ADD COLUMN IF NOT EXISTS raw_response     JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS raw_notification JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS refund_time      TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00 UTC';

ALTER TABLE purchase
    ADD COLUMN IF NOT EXISTS refund_time TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00 UTC';

-- +migrate Down
ALTER TABLE subscription
    DROP COLUMN IF EXISTS raw_response,
    DROP COLUMN IF EXISTS raw_notification,
    DROP COLUMN IF EXISTS refund_time;

ALTER TABLE purchase
    DROP COLUMN IF EXISTS refund_time;
