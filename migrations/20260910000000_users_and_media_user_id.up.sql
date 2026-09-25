CREATE TABLE IF NOT EXISTS users
(
    id         VARCHAR(36) PRIMARY KEY,
    username   VARCHAR(64) UNIQUE NOT NULL,
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

ALTER TABLE media_binary ADD COLUMN IF NOT EXISTS user_id VARCHAR(36);
CREATE INDEX IF NOT EXISTS idx_media_user_id ON media_binary(user_id);
