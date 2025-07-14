-- Migration 000009: Remove external_target_masked column from transactions table
-- This column stored a masked version of external target info for withdrawals.
ALTER TABLE transactions
DROP COLUMN IF EXISTS external_target_masked;
