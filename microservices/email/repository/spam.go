package repository

import (
	"context"
	"database/sql"
)

func (r *Repository) GetSpamEmailIDs(ctx context.Context, userID int64, limit int, offset int) ([]int64, error) {
	const query = `
		SELECT id
		FROM user_emails
		WHERE user_id = $1
			AND is_spam = true
			AND is_deleted = false
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, ErrQueryFail
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}

	return ids, nil
}

func (r *Repository) InsertSpamSender(ctx context.Context, tx *sql.Tx, userID int64, senderID int64) error {
	const query = `
		INSERT INTO spam_senders
		(user_id, sender_id)
		VALUES ($1, $2)
	`

	if _, err := tx.ExecContext(ctx, query, userID, senderID); err != nil {
		return ErrQueryFail
	}

	return nil
}

func (r *Repository) DeleteSpamSender(ctx context.Context, tx *sql.Tx, userID int64, senderID int64) error {
	const query = `
		DELETE FROM spam_senders
		WHERE user_id = $1 AND sender_id = $2
	`

	if _, err := tx.ExecContext(ctx, query, userID, senderID); err != nil {
		return ErrQueryFail
	}

	return nil
}
