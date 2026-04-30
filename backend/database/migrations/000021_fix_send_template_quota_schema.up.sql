-- Align send/template/quota models with versioned SQL schema.

CREATE TABLE IF NOT EXISTS sent_emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    send_id VARCHAR(100) NOT NULL UNIQUE,
    account_id INTEGER NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    recipients TEXT,
    sent_at DATETIME NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'sent',
    size INTEGER DEFAULT 0,
    error TEXT,
    retry_count INTEGER DEFAULT 0,
    FOREIGN KEY (account_id) REFERENCES email_accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_sent_emails_deleted_at ON sent_emails(deleted_at);
CREATE INDEX IF NOT EXISTS idx_sent_emails_send_id ON sent_emails(send_id);
CREATE INDEX IF NOT EXISTS idx_sent_emails_account_id ON sent_emails(account_id);
CREATE INDEX IF NOT EXISTS idx_sent_emails_sent_at ON sent_emails(sent_at);

CREATE TABLE IF NOT EXISTS email_quotas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    user_id INTEGER NOT NULL UNIQUE,
    daily_limit INTEGER DEFAULT 1000,
    monthly_limit INTEGER DEFAULT 30000,
    attachment_size_limit INTEGER DEFAULT 26214400,
    daily_used INTEGER DEFAULT 0,
    monthly_used INTEGER DEFAULT 0,
    last_reset_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_blocked BOOLEAN DEFAULT false,
    block_reason VARCHAR(255),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_email_quotas_deleted_at ON email_quotas(deleted_at);
CREATE INDEX IF NOT EXISTS idx_email_quotas_user_id ON email_quotas(user_id);

ALTER TABLE email_templates ADD COLUMN description VARCHAR(500);
ALTER TABLE email_templates ADD COLUMN text_body TEXT;
ALTER TABLE email_templates ADD COLUMN html_body TEXT;
ALTER TABLE email_templates ADD COLUMN variables TEXT;
ALTER TABLE email_templates ADD COLUMN is_active BOOLEAN DEFAULT true;
ALTER TABLE email_templates ADD COLUMN is_default BOOLEAN DEFAULT false;
ALTER TABLE email_templates ADD COLUMN usage_count INTEGER DEFAULT 0;
ALTER TABLE email_templates ADD COLUMN last_used_at DATETIME;
ALTER TABLE email_templates ADD COLUMN deleted_at DATETIME;

UPDATE email_templates
SET text_body = COALESCE(text_body, body),
    html_body = COALESCE(html_body, body)
WHERE body IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_email_templates_is_active ON email_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_email_templates_is_default ON email_templates(is_default);
CREATE INDEX IF NOT EXISTS idx_email_templates_deleted_at ON email_templates(deleted_at);
