ALTER TABLE users
    DROP INDEX unique_users_google_id,
    DROP COLUMN picture,
    DROP COLUMN google_id;