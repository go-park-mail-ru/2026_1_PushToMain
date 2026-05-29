DROP INDEX IF EXISTS idx_emails_parent_email_id;
ALTER TABLE emails DROP COLUMN IF EXISTS parent_email_id;
