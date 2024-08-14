-- +migrate Up
ALTER TABLE users
    ADD COLUMN apple_id VARCHAR(128) UNIQUE;

-- +migrate Down
ALTER TABLE users
    DROP COLUMN IF EXISTS apple_id;
