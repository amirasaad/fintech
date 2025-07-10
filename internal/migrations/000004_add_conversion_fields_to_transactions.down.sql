-- Remove conversion fields from transactions table
DROP INDEX IF EXISTS idx_transactions_conversion;

ALTER TABLE transactions
DROP COLUMN IF EXISTS original_amount,
DROP COLUMN IF EXISTS original_currency,
DROP COLUMN IF EXISTS conversion_rate; 