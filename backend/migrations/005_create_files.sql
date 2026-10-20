-- Migration: 005_create_files.sql
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    uploader_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    s3_key VARCHAR(512) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_files_uploader_id ON files(uploader_id);
CREATE INDEX IF NOT EXISTS idx_files_room_id ON files(room_id);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at);

-- Foreign key linking messages.file_id -> files.id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_messages_files'
    ) THEN
        ALTER TABLE messages
        ADD CONSTRAINT fk_messages_files
        FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE SET NULL;
    END IF;
END $$;
