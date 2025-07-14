ALTER TABLE transactions ADD COLUMN payment_id VARCHAR(64);
CREATE INDEX idx_transactions_payment_id ON transactions(payment_id);
