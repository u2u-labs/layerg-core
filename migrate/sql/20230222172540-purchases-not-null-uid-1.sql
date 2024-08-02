-- +migrate Up
-- This migration is split in two files due to the following CRDB limitation
-- https://stackoverflow.com/questions/68803747/encapsulating-a-drop-and-add-constraint-in-a-transaction
ALTER TABLE purchase
    DROP CONSTRAINT IF EXISTS purchase_user_id_fkey,
    DROP CONSTRAINT IF EXISTS fk_user_id_ref_users;
ALTER TABLE subscription
    DROP CONSTRAINT IF EXISTS subscription_user_id_fkey,
    DROP CONSTRAINT IF EXISTS fk_user_id_ref_users;

UPDATE purchase
    SET user_id = '00000000-0000-0000-0000-000000000000' WHERE user_id IS NULL;
UPDATE subscription
    SET user_id = '00000000-0000-0000-0000-000000000000' WHERE user_id IS NULL;

-- +migrate Down
-- This migration is split in two files due to the following CRDB limitation
-- https://stackoverflow.com/questions/68803747/encapsulating-a-drop-and-add-constraint-in-a-transaction
UPDATE purchase
    SET user_id = NULL WHERE user_id = '00000000-0000-0000-0000-000000000000';
UPDATE subscription
    SET user_id = NULL WHERE user_id = '00000000-0000-0000-0000-000000000000';

ALTER TABLE purchase
    ADD CONSTRAINT purchase_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE subscription
    ADD CONSTRAINT subscription_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;
