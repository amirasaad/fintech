-- First, create a temporary table to store the combined balances
-- Using a more robust approach to handle potential integer overflow
CREATE TEMP TABLE combined_balances AS
WITH account_sums AS (
    SELECT
        user_id,
        currency,
        SUM(CAST(balance AS NUMERIC)) as total_balance_numeric,
        MAX(created_at) as latest_created_at,
        MAX(updated_at) as latest_updated_at
    FROM
        accounts
    GROUP BY
        user_id, currency
)
SELECT
    user_id,
    CASE
        WHEN total_balance_numeric > 9223372036854775807 THEN 9223372036854775807 -- Max bigint
        WHEN total_balance_numeric < -9223372036854775808 THEN -9223372036854775808 -- Min bigint
        ELSE total_balance_numeric::BIGINT
    END as total_balance,
    latest_created_at,
    latest_updated_at,
    currency
FROM
    account_sums;

-- Delete all existing accounts
TRUNCATE TABLE accounts CASCADE;

-- Ensure the uuid-ossp extension is available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Insert one account per user with the combined balance
INSERT INTO accounts (id, user_id, balance, currency, created_at, updated_at)
SELECT
    uuid_generate_v4(),
    user_id,
    total_balance,
    currency,
    latest_created_at,
    latest_updated_at
FROM
    combined_balances;

-- Drop the temporary table
DROP TABLE combined_balances;

-- Finally, add a unique constraint to ensure one account per user and currency
ALTER TABLE accounts
ADD CONSTRAINT uidx_user_currency UNIQUE (user_id, currency);
