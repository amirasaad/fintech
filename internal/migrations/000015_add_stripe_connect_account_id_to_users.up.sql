-- +migrate Up
-- Add stripe_connect_account_id column to users table
ALTER TABLE users
ADD COLUMN stripe_connect_account_id VARCHAR(50);

-- Create index for faster lookups
CREATE INDEX idx_users_stripe_connect_account_id ON users(stripe_connect_account_id);
