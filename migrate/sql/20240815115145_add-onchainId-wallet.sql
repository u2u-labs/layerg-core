-- +migrate Up
ALTER TABLE users ADD onchain_id varchar(128) NULL UNIQUE;

-- +migrate Down
ALTER TABLE users DROP COLUMN onchain_id;
