CREATE TABLE IF NOT EXISTS users (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS daily_entries (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    entry_date DATE NOT NULL,
    food_intake TEXT NOT NULL DEFAULT '',
    water_intake DECIMAL(5,2) NOT NULL DEFAULT 0.0,
    bed_time TIME NULL,
    wake_time TIME NULL,
    sleep_quality INTEGER NULL CHECK (sleep_quality BETWEEN 1 AND 5),
    alcohol_units DECIMAL(4,2) NOT NULL DEFAULT 0.0,
    exercise_type VARCHAR(255) NOT NULL DEFAULT '',
    exercise_duration INTEGER NOT NULL DEFAULT 0,
    exercise_intensity INTEGER NULL CHECK (exercise_intensity BETWEEN 1 AND 5),
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY unique_daily_entry_user_date (user_id, entry_date),
    KEY idx_daily_entries_date (entry_date),
    CONSTRAINT fk_daily_entries_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS goals (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    category VARCHAR(100) NOT NULL,
    target DECIMAL(10,2) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    KEY idx_goals_user_active (user_id, is_active),
    CONSTRAINT fk_goals_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;