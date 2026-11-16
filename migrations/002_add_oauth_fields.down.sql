-- Remove Google OAuth2 fields from users table
DROP INDEX IF EXISTS idx_users_google_id;
ALTER TABLE users DROP COLUMN IF EXISTS picture;
ALTER TABLE users DROP COLUMN IF EXISTS google_id;
