-- +goose Up
-- +goose StatementBegin

-- Add Stripe Connect related columns to users table
ALTER TABLE users
    ADD COLUMN stripe_connect_onboarding_completed BOOLEAN DEFAULT FALSE,
    ADD COLUMN stripe_connect_account_status VARCHAR(50);

-- +goose StatementEnd
