-- +goose Down
-- +goose StatementBegin

-- Remove the index
DROP INDEX IF EXISTS idx_users_stripe_connect_account_id;

-- Remove the columns
ALTER TABLE users
    DROP COLUMN IF EXISTS stripe_connect_onboarding_completed,
    DROP COLUMN IF EXISTS stripe_connect_account_status;

-- +goose StatementEnd
