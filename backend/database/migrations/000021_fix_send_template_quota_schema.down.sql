DROP INDEX IF EXISTS idx_email_templates_is_default;
DROP INDEX IF EXISTS idx_email_templates_is_active;
DROP INDEX IF EXISTS idx_email_templates_deleted_at;

ALTER TABLE email_templates DROP COLUMN deleted_at;
ALTER TABLE email_templates DROP COLUMN last_used_at;
ALTER TABLE email_templates DROP COLUMN usage_count;
ALTER TABLE email_templates DROP COLUMN is_default;
ALTER TABLE email_templates DROP COLUMN is_active;
ALTER TABLE email_templates DROP COLUMN variables;
ALTER TABLE email_templates DROP COLUMN html_body;
ALTER TABLE email_templates DROP COLUMN text_body;
ALTER TABLE email_templates DROP COLUMN description;

DROP TABLE IF EXISTS email_quotas;
DROP TABLE IF EXISTS sent_emails;
