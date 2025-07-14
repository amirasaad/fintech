DROP INDEX IF EXISTS idx_transactions_payment_id;
ALTER TABLE transactions DROP COLUMN payment_id;
