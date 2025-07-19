ALTER TABLE transactions ADD COLUMN target_currency VARCHAR(8) NULL;
ALTER TABLE transactions ADD COLUMN conversion_info TEXT NULL;
