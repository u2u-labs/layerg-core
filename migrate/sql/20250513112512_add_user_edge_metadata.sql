-- +migrate Up
ALTER TABLE user_edge
    ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';

-- +migrate Down
ALTER TABLE user_edge
    DROP COLUMN metadata;