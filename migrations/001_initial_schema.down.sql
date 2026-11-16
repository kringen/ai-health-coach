-- Drop triggers
DROP TRIGGER IF EXISTS update_goals_modified_at ON goals;
DROP TRIGGER IF EXISTS update_daily_entries_modified_at ON daily_entries;
DROP TRIGGER IF EXISTS update_users_modified_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_modified_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_goals_user_active;
DROP INDEX IF EXISTS idx_daily_entries_date;
DROP INDEX IF EXISTS idx_daily_entries_user_date;

-- Drop tables (in reverse order due to foreign key constraints)
DROP TABLE IF EXISTS goals;
DROP TABLE IF EXISTS daily_entries;
DROP TABLE IF EXISTS users;
