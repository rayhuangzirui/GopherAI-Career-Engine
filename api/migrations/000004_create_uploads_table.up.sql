CREATE TABLE IF NOT EXISTS uploads (
    id BIGINT AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    file_kind VARCHAR(20) NOT NULL,
    storage_key VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    CONSTRAINT fk_uploads_user_id
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    INDEX idx_uploads_user_id (user_id),
    INDEX idx_uploads_file_kind (file_kind),
    INDEX idx_uploads_created_at (created_at),
    UNIQUE KEY uk_uploads_storage_key (storage_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;