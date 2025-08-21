-- Remove the exclusion constraint
ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS uq_transactions_payment_id;

-- Drop the index
DROP INDEX IF EXISTS idx_transactions_payment_id_not_null;
