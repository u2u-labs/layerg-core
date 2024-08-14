-- +migrate Up
DROP INDEX IF EXISTS purchase_user_id_purchase_time_transaction_id_idx;
DROP INDEX IF EXISTS subscription_user_id_purchase_time_transaction_id_idx;

CREATE INDEX IF NOT EXISTS purchase_time_user_id_transaction_id_idx
    ON purchase (purchase_time DESC, user_id DESC, transaction_id DESC);
CREATE INDEX IF NOT EXISTS subscription_time_user_id_transaction_id_idx
    ON subscription (purchase_time DESC, user_id DESC, original_transaction_id DESC);

-- +migrate Down
DROP INDEX IF EXISTS purchase_time_user_id_transaction_id_idx;
DROP INDEX IF EXISTS subscription_time_user_id_transaction_id_idx;

CREATE INDEX IF NOT EXISTS purchase_user_id_purchase_time_transaction_id_idx
    ON purchase (user_id, purchase_time DESC, transaction_id);
CREATE INDEX IF NOT EXISTS subscription_user_id_purchase_time_transaction_id_idx
    ON subscription (user_id, purchase_time DESC, original_transaction_id);
