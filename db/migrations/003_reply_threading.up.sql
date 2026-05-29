ALTER TABLE emails
    ADD COLUMN parent_email_id BIGINT NULL REFERENCES emails(id) ON DELETE SET NULL;

CREATE INDEX idx_emails_parent_email_id
    ON emails(parent_email_id)
    WHERE parent_email_id IS NOT NULL;
