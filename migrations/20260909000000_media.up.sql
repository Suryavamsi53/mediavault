CREATE TABLE IF NOT EXISTS media_binary
(
    id         VARCHAR(36) PRIMARY KEY,
    file_name  VARCHAR(255) NOT NULL,
    media_type VARCHAR(100) NOT NULL,
    file_size  BIGINT NOT NULL,
    data       BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
