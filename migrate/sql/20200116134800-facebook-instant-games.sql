-- +migrate Up
ALTER TABLE users
    ADD COLUMN facebook_instant_game_id VARCHAR(128) UNIQUE;

-- +migrate Down
ALTER TABLE users
    DROP COLUMN IF EXISTS facebook_instant_game_id;
