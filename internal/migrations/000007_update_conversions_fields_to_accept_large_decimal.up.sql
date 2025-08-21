-- Update conversion fields to accept larger decimals in transactions table
ALTER TABLE transactions
ALTER COLUMN original_amount TYPE DECIMAL(30,15),
ALTER COLUMN conversion_rate TYPE DECIMAL(30,15);
