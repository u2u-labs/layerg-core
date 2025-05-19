-- +migrate Up
ALTER TABLE console_user
    ADD COLUMN mfa_secret BYTEA DEFAULT NULL,
    ADD COLUMN mfa_recovery_codes BYTEA DEFAULT NULL,
    ADD COLUMN mfa_required BOOLEAN DEFAULT FALSE;

-- +migrate Down
ALTER TABLE console_user
    DROP COLUMN mfa_secret,
    DROP COLUMN mfa_recovery_codes,
    DROP COLUMN mfa_required;