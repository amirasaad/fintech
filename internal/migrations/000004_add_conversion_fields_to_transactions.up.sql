-- Add conversion fields to transactions table
ALTER TABLE transactions
ADD COLUMN original_amount DECIMAL(20,8) NULL,
ADD COLUMN original_currency VARCHAR(3) NULL,
ADD COLUMN conversion_rate DECIMAL(20,8) NULL;

-- Add index for conversion queries
CREATE INDEX idx_transactions_conversion ON transactions(original_currency, conversion_rate) WHERE original_currency IS NOT NULL;
