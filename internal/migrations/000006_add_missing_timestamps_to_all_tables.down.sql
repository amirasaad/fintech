-- Drop deleted_at from users table
ALTER TABLE users DROP COLUMN deleted_at;

-- Drop created_at and deleted_at from accounts table
ALTER TABLE accounts DROP COLUMN created_at;
ALTER TABLE accounts DROP COLUMN deleted_at;

-- Drop updated_at and deleted_at from transactions table
ALTER TABLE transactions DROP COLUMN updated_at;
ALTER TABLE transactions DROP COLUMN deleted_at;
