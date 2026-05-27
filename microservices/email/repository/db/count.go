package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) GetInboxStats(ctx context.Context, userID int64) (models.MailboxStats, error) {
	const query = `
        SELECT
            COUNT(*),
            COUNT(*) FILTER (WHERE is_read = false)
        FROM user_emails
        WHERE user_id = $1 AND is_deleted = false AND is_spam = false AND is_inbox = true AND is_sender = false
    `
	var s models.MailboxStats
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&s.Total, &s.Unread); err != nil {
		return models.MailboxStats{}, ErrQueryFail
	}
	return s, nil
}

func (r *Repository) GetReceivedStats(ctx context.Context, userID int64) (models.MailboxStats, error) {
	const query = `
        SELECT
            COUNT(*),
            COUNT(*) FILTER (WHERE is_read = false)
        FROM user_emails
        WHERE user_id = $1 AND is_deleted = false AND is_spam = false AND is_sender = false
    `
	var s models.MailboxStats
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&s.Total, &s.Unread); err != nil {
		return models.MailboxStats{}, ErrQueryFail
	}
	return s, nil
}

func (r *Repository) GetSpamStats(ctx context.Context, userID int64) (models.MailboxStats, error) {
	const query = `
        SELECT
            COUNT(*),
            COUNT(*) FILTER (WHERE is_read = false)
        FROM user_emails
        WHERE user_id = $1 AND is_spam = true AND is_deleted = false AND is_sender = false
    `
	var s models.MailboxStats
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&s.Total, &s.Unread); err != nil {
		return models.MailboxStats{}, ErrQueryFail
	}
	return s, nil
}

func (r *Repository) GetTrashStats(ctx context.Context, userID int64) (models.MailboxStats, error) {
	const query = `
        SELECT
            COUNT(*),
            COUNT(*) FILTER (WHERE is_read = false)
        FROM user_emails
        WHERE user_id = $1 AND is_deleted = true
    `
	var s models.MailboxStats
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&s.Total, &s.Unread); err != nil {
		return models.MailboxStats{}, ErrQueryFail
	}
	return s, nil
}

func (r *Repository) CountSentEmails(ctx context.Context, userID int64) (int, error) {
	const query = `
        SELECT COUNT(*)
        FROM emails
        WHERE sender_id = $1 AND is_draft = false 
    `
	var n int
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&n); err != nil {
		return 0, ErrQueryFail
	}
	return n, nil
}

func (r *Repository) CountDrafts(ctx context.Context, userID int64) (int, error) {
	const query = `
        SELECT COUNT(*)
        FROM emails
        WHERE sender_id = $1 AND is_draft = true
    `
	var n int
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&n); err != nil {
		return 0, ErrQueryFail
	}
	return n, nil
}
