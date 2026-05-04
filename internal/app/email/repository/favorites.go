package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

func (r *Repository) GetFavoriteEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at,
			ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
			ue.updated_at AS received_at,
			COALESCE((
				SELECT array_agg(ru.email ORDER BY ru.id)
				FROM user_emails rue
				JOIN users ru ON rue.user_id = ru.id
				WHERE rue.email_id = e.id AND rue.is_sender = false
			), '{}'::text[]) AS receivers_emails
		FROM user_emails ue
		JOIN emails e ON ue.email_id = e.id
		WHERE ue.user_id = $1
		  AND ue.is_starred = true
		  AND ue.is_deleted = false
		  AND ue.is_draft = false
		ORDER BY ue.updated_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}
