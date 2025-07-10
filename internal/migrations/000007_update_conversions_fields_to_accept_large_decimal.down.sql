-- Revert conversion fields to original decimal size in transactions table
ALTER TABLE transactions
ALTER COLUMN original_amount TYPE DECIMAL(20,8),
ALTER COLUMN conversion_rate TYPE DECIMAL(20,8);