DROP INDEX IF EXISTS idx_media_user_id;
ALTER TABLE media_binary DROP COLUMN IF EXISTS user_id;
DROP TABLE IF EXISTS users;
