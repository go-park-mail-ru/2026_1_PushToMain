package repository

import (
	"context"

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
