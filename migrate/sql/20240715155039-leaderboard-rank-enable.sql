-- +migrate Up
ALTER TABLE leaderboard ADD COLUMN IF NOT EXISTS enable_ranks boolean DEFAULT true;

-- +migrate Down
ALTER TABLE leaderboard DROP COLUMN IF EXISTS enable_ranks;
