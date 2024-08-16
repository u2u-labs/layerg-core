-- +migrate Up
ALTER TABLE users ADD telegram_id varchar(128) NULL;

-- +migrate Down
ALTER TABLE users DROP COLUMN telegram_id;
