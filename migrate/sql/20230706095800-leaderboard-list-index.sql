-- +migrate Up
CREATE INDEX IF NOT EXISTS leaderboard_create_time_id_idx
    ON leaderboard (create_time ASC, id ASC);

-- +migrate Down
DROP INDEX IF EXISTS leaderboard_create_time_id_idx;
