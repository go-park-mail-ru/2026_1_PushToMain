package repository

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

func (r *Repository) UnspamEmailsForReceiver(ctx context.Context, userID int64, emailIDs []int64) error {
    if len(emailIDs) == 0 {
        return nil
    }
    holders, idArgs := idsPlaceholders(emailIDs, 2)
    query := fmt.Sprintf(`
        UPDATE user_emails
        SET is_spam = false, updated_at = NOW()
        WHERE user_id = $1
        	AND is_sender = false
         	AND email_id IN (%s)
    `, holders)
    args := append([]any{userID}, idArgs...)
    if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
        return ErrQueryFail
    }
    return nil
}

func (r *Repository) RemoveSpamSendersByReceiverEmails(ctx context.Context, userID int64, emailIDs []int64) error {
    if len(emailIDs) == 0 {
        return nil
    }

    holders, idArgs := idsPlaceholders(emailIDs, 2)
    selectQuery := fmt.Sprintf(`
        SELECT DISTINCT e.sender_id
        FROM emails e
        JOIN user_emails ue ON ue.email_id = e.id
        WHERE ue.user_id = $1
        	AND ue.is_sender = false
          	AND e.id IN (%s)
    `, holders)

    args := append([]any{userID}, idArgs...)
    rows, err := r.db.QueryContext(ctx, selectQuery, args...)
    if err != nil {
        return ErrQueryFail
    }
    defer rows.Close()

    var senderIDs []int64
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err != nil {
            return ErrQueryFail
        }
        senderIDs = append(senderIDs, id)
    }
    if err := rows.Err(); err != nil {
        return ErrQueryFail
    }
    if len(senderIDs) == 0 {
        return nil
    }

    sHolders, sIDArgs := idsPlaceholders(senderIDs, 2)
    deleteQuery := fmt.Sprintf(`
        DELETE FROM spam_senders
        WHERE user_id = $1 AND sender_id IN (%s)
    `, sHolders)

    deleteArgs := append([]any{userID}, sIDArgs...)
    if _, err := r.db.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
        return ErrQueryFail
    }

    return nil
}

func (r *Repository) GetSpamEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at,
			ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
			ue.created_at AS received_at,
			COALESCE((
				SELECT array_agg(ru.email ORDER BY ru.id)
				FROM user_emails rue
				JOIN users ru ON rue.user_id = ru.id
				WHERE rue.email_id = e.id AND rue.is_sender = false
			), '{}'::text[]) AS receivers_emails
		FROM user_emails ue
		JOIN emails e ON ue.email_id = e.id
		WHERE ue.user_id = $1
		  AND ue.is_sender = false
		  AND ue.is_spam = true
		  AND ue.is_deleted = false
		  AND ue.is_draft = false
		ORDER BY ue.created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}

func (r *Repository) GetSpamEmailsCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_spam = true AND is_deleted = false AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}

func (r *Repository) GetUnreadSpamCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_spam = true AND is_read = false
		  AND is_deleted = false AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}
