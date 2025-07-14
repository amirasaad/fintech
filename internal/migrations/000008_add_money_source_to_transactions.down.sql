-- Drop money_source field transactions table
ALTER TABLE transactions DROP COLUMN IF EXISTS money_source;
