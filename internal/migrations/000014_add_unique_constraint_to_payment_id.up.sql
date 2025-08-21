-- Add unique constraint to payment_id
-- First, clean up any duplicate payment_ids by keeping only the latest transaction
WITH duplicates AS (
    SELECT payment_id, MAX(updated_at) as latest_update
    FROM transactions
    WHERE payment_id IS NOT NULL
    GROUP BY payment_id
    HAVING COUNT(*) > 1
)
UPDATE transactions t1
SET payment_id = t1.payment_id || '_' || t1.id::text
FROM duplicates d
WHERE t1.payment_id = d.payment_id
  AND t1.updated_at < d.latest_update;

-- Add a unique constraint that only enforces uniqueness for non-null values
-- This is done by creating a function-based unique index
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_payment_id_not_null
    ON transactions (payment_id)
    WHERE payment_id IS NOT NULL;

-- Create a constraint that uses the index
ALTER TABLE transactions
    ADD CONSTRAINT uq_transactions_payment_id
    EXCLUDE (payment_id WITH =)
    WHERE (payment_id IS NOT NULL);
