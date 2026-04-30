DROP INDEX IF EXISTS idx_mailbox_jobs_completed_at;
DROP INDEX IF EXISTS idx_mailbox_jobs_started_at;
DROP INDEX IF EXISTS idx_mailbox_jobs_status;
DROP INDEX IF EXISTS idx_mailbox_jobs_operation;
DROP INDEX IF EXISTS idx_mailbox_jobs_user_id;
DROP INDEX IF EXISTS idx_mailbox_jobs_job_id;
DROP INDEX IF EXISTS idx_mailbox_jobs_deleted_at;

DROP TABLE IF EXISTS mailbox_jobs;
