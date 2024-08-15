-- +migrate Up
ALTER TABLE wallet_ledger ADD onchain_id varchar(128) NULL UNIQUE;
ALTER TABLE wallet_ledger ADD wallet_signature varchar(128) NULL UNIQUE;

-- +migrate Down
ALTER TABLE wallet_ledger DROP COLUMN telegram_id;
ALTER TABLE wallet_ledger DROP COLUMN wallet_signature;
