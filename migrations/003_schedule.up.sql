CREATE TABLE IF NOT EXISTS schedule_files (
    id          SERIAL PRIMARY KEY,
    filename    TEXT NOT NULL,
    file_type   VARCHAR(20) NOT NULL,
    file_size   BIGINT,
    downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(filename, file_type)
);

CREATE INDEX idx_schedule_files_type ON schedule_files(file_type);
CREATE INDEX idx_schedule_files_downloaded ON schedule_files(downloaded_at);