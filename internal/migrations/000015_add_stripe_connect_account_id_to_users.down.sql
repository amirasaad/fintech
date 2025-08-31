-- +migrate Down
-- Drop the index first
DROP INDEX IF EXISTS idx_users_stripe_connect_account_id;

-- Then drop the column
ALTER TABLE users
DROP COLUMN IF EXISTS stripe_connect_account_id;
