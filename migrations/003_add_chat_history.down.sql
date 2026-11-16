-- Remove chat_history table
DROP TRIGGER IF EXISTS update_chat_history_modified_at ON chat_history;
DROP TABLE IF EXISTS chat_history;
