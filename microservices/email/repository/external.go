package repository

import (
	"context"
	"database/sql"
)

func (r *Repository) InsertExternalEmail(
	ctx context.Context,
	tx *sql.Tx,
	senderEmail, header, body string,
) (int64, error) {
	const query = `
		INSERT INTO emails (sender_id, sender_email, header, body, is_draft)
		VALUES (NULL, $1, $2, $3, false)
		RETURNING id
	`
	var id int64
	err := tx.QueryRowContext(ctx, query, senderEmail, header, body).Scan(&id)
	if err != nil {
		return 0, mapPgError(err)
	}
	return id, nil
}
