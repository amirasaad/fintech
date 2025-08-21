-- Add deleted_at to users table
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

-- Add created_at and deleted_at to accounts table
ALTER TABLE accounts ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE accounts ADD COLUMN deleted_at TIMESTAMPTZ;

-- Add updated_at and deleted_at to transactions table
ALTER TABLE transactions ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE transactions ADD COLUMN deleted_at TIMESTAMPTZ;
