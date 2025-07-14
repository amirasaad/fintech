-- Add money_source field transactions table
ALTER TABLE transactions ADD COLUMN money_source VARCHAR(64) NOT NULL DEFAULT 'Internal';
