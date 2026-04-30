CREATE TABLE IF NOT EXISTS mailbox_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    job_id VARCHAR(100) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    operation VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    account_ids TEXT NOT NULL,
    processed_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_deleted_at ON mailbox_jobs(deleted_at);
CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_job_id ON mailbox_jobs(job_id);
CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_user_id ON mailbox_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_operation ON mailbox_jobs(operation);
CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_status ON mailbox_jobs(status);
CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_started_at ON mailbox_jobs(started_at);
CREATE INDEX IF NOT EXISTS idx_mailbox_jobs_completed_at ON mailbox_jobs(completed_at);
