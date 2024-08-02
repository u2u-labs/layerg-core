-- +migrate Up
-- This migration is split in two files due to the following CRDB limitation
-- https://stackoverflow.com/questions/68803747/encapsulating-a-drop-and-add-constraint-in-a-transaction
ALTER TABLE purchase
    ALTER COLUMN user_id SET NOT NULL,
    ALTER COLUMN user_id SET DEFAULT '00000000-0000-0000-0000-000000000000';
ALTER TABLE subscription
    ALTER COLUMN user_id SET NOT NULL,
    ALTER COLUMN user_id SET DEFAULT '00000000-0000-0000-0000-000000000000';

ALTER TABLE purchase
    ADD CONSTRAINT purchase_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET DEFAULT;
ALTER TABLE subscription
    ADD CONSTRAINT subscription_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET DEFAULT;

-- +migrate Down
-- This migration is split in two files due to the following CRDB limitation
-- https://stackoverflow.com/questions/68803747/encapsulating-a-drop-and-add-constraint-in-a-transaction
ALTER TABLE purchase
    DROP CONSTRAINT IF EXISTS purchase_user_id_fkey,
    DROP CONSTRAINT IF EXISTS fk_user_id_ref_users;
ALTER TABLE subscription
    DROP CONSTRAINT IF EXISTS subscription_user_id_fkey,
    DROP CONSTRAINT IF EXISTS fk_user_id_ref_users;

ALTER TABLE purchase
    ALTER COLUMN user_id DROP DEFAULT,
    ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE subscription
    ALTER COLUMN user_id DROP DEFAULT,
    ALTER COLUMN user_id DROP NOT NULL;
