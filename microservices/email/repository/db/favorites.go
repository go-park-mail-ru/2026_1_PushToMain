package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) GetFavoritesStats(ctx context.Context, userID int64) (models.MailboxStats, error) {
	const query = `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE is_read = false)
		FROM user_emails
		WHERE user_id = $1 AND is_starred = true AND is_deleted = false
	`
	var s models.MailboxStats
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&s.Total, &s.Unread); err != nil {
		return models.MailboxStats{}, ErrQueryFail
	}
	return s, nil
}

func (r *Repository) StarEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	if len(emailIDs) == 0 {
		return nil
	}
	parts := make([]string, len(emailIDs))
	args := make([]any, 0, len(emailIDs)+1)
	args = append(args, userID)
	for i, id := range emailIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_starred = true, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (%s) AND is_spam = false
	`, strings.Join(parts, ","))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) UnstarEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	return r.setUserEmailFlagsBatch(ctx, userID, emailIDs, "is_starred", false)
}
