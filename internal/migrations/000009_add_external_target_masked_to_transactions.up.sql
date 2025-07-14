-- Migration 000009: Add external_target_masked column to transactions table
-- This column stores a masked version of external target info (e.g., bank account, wallet address) for withdrawals, for security and audit purposes.
ALTER TABLE transactions
ADD COLUMN external_target_masked VARCHAR(128);
