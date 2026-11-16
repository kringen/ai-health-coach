ALTER TABLE users
    ADD COLUMN google_id VARCHAR(255) NULL,
    ADD COLUMN picture TEXT NULL,
    ADD UNIQUE KEY unique_users_google_id (google_id);