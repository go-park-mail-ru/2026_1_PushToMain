package repository

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

func (r *Repository) GetTrashEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
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
	    FROM (
	        SELECT DISTINCT ON (ue_inner.email_id)
	            ue_inner.*
	        FROM user_emails ue_inner
	        WHERE ue_inner.user_id = $1
	          AND ue_inner.is_deleted = true
	          AND ue_inner.is_draft = false
	        ORDER BY ue_inner.email_id, ue_inner.is_sender ASC
	    ) ue
	    JOIN emails e ON ue.email_id = e.id
	    ORDER BY ue.updated_at DESC
	    LIMIT $2 OFFSET $3
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}

func (r *Repository) GetTrashEmailsCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_deleted = true AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}

func (r *Repository) GetUnreadTrashCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_deleted = true AND is_read = false AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}

func (r *Repository) SetTrashedBatch(ctx context.Context, userID int64, emailIDs []int64, trashed bool) error {
	if len(emailIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrTransactionFailed
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	holders, idArgs := idsPlaceholders(emailIDs, 2)

	var updateQuery string
	if trashed {
		updateQuery = fmt.Sprintf(`
            UPDATE user_emails
            SET is_deleted = true,
                is_starred = false,
                updated_at = NOW()
            WHERE user_id = $1 AND email_id IN (%s)
        `, holders)
	} else {
		updateQuery = fmt.Sprintf(`
            UPDATE user_emails
            SET is_deleted = false,
                updated_at = NOW()
            WHERE user_id = $1 AND email_id IN (%s)
        `, holders)
	}

	args := append([]any{userID}, idArgs...)
	if _, err := tx.ExecContext(ctx, updateQuery, args...); err != nil {
		return ErrQueryFail
	}

	if trashed {
		holders2, idArgs2 := idsPlaceholders(emailIDs, 2)
		deleteQuery := fmt.Sprintf(`
            DELETE FROM folder_emails
            WHERE email_id IN (%s)
              AND folder_id IN (SELECT id FROM folders WHERE user_id = $1)
        `, holders2)
		args2 := append([]any{userID}, idArgs2...)
		if _, err := tx.ExecContext(ctx, deleteQuery, args2...); err != nil {
			return ErrQueryFail
		}
	}

	if err := tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	committed = true
	return nil
}
