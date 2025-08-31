-- First, remove the unique constraint
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS uidx_user_id;

-- Note: There's no way to perfectly rollback the account combination
-- as we've already lost the original account structure.
-- The best we can do is leave the combined accounts as they are.
-- If you need to restore the original accounts, you'll need to restore from a backup.
