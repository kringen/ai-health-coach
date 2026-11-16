-- Create chat_history table to persist AI evaluation/follow-up conversations per user per day
CREATE TABLE IF NOT EXISTS chat_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    messages JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, entry_date)
);

CREATE TRIGGER update_chat_history_modified_at BEFORE UPDATE ON chat_history
    FOR EACH ROW EXECUTE FUNCTION update_modified_at_column();
