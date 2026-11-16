-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create daily_entries table
CREATE TABLE IF NOT EXISTS daily_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    food_intake TEXT DEFAULT '',
    water_intake DECIMAL(5,2) DEFAULT 0.0,
    bed_time TIME,
    wake_time TIME,
    sleep_quality INTEGER CHECK (sleep_quality >= 1 AND sleep_quality <= 5),
    alcohol_units DECIMAL(4,2) DEFAULT 0.0,
    exercise_type VARCHAR(255) DEFAULT '',
    exercise_duration INTEGER DEFAULT 0, -- in minutes
    exercise_intensity INTEGER CHECK (exercise_intensity >= 1 AND exercise_intensity <= 5),
    notes TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, entry_date)
);

-- Create goals table
CREATE TABLE IF NOT EXISTS goals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    target DECIMAL(10,2) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    description TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_daily_entries_user_date ON daily_entries(user_id, entry_date);
CREATE INDEX IF NOT EXISTS idx_daily_entries_date ON daily_entries(entry_date);
CREATE INDEX IF NOT EXISTS idx_goals_user_active ON goals(user_id, is_active);

-- Create function to update modified_at timestamp
CREATE OR REPLACE FUNCTION update_modified_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to automatically update modified_at
CREATE TRIGGER update_users_modified_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_modified_at_column();

CREATE TRIGGER update_daily_entries_modified_at BEFORE UPDATE ON daily_entries
    FOR EACH ROW EXECUTE FUNCTION update_modified_at_column();

CREATE TRIGGER update_goals_modified_at BEFORE UPDATE ON goals
    FOR EACH ROW EXECUTE FUNCTION update_modified_at_column();
