-- +migrate Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS twitter_id VARCHAR(128) UNIQUE;

-- +migrate Down
ALTER TABLE users DROP COLUMN IF EXISTS twitter_id;
