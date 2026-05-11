package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) GetDraftIDs(ctx context.Context, userID int64, limit, offset int) ([]int64, error) {
	const query = `
		SELECT id
		FROM emails
		WHERE sender_id = $1 AND is_draft = true
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

func (r *Repository) GetDraftByID(ctx context.Context, draftID int64) (*models.Draft, error) {
	const queryDraftBase = `
		SELECT id, sender_id, sender_email, COALESCE(header, ''), COALESCE(body, ''), created_at, updated_at
		FROM emails
		WHERE id = $1
	`

	const queryDraftRecipients = `
		SELECT recipient_email
		FROM email_recipients
		WHERE email_id = $1
	`

	var draft models.Draft
	err := r.db.QueryRowContext(ctx, queryDraftBase, draftID).
		Scan(
			&draft.ID,
			&draft.SenderID,
			&draft.SenderEmail,
			&draft.Header,
			&draft.Body,
			&draft.CreatedAt,
			&draft.UpdatedAt,
		)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDraftNotFound
	}
	if err != nil {
		return nil, ErrQueryFail
	}

	rows, err := r.db.QueryContext(ctx, queryDraftRecipients, draftID)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	recipients := make([]string, 0)
	for rows.Next() {
		var recipient string
		if err := rows.Scan(&recipient); err != nil {
			return nil, ErrQueryFail
		}
		recipients = append(recipients, recipient)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}

	draft.Recipients = recipients
	return &draft, nil
}

func (r *Repository) InsertDraft(ctx context.Context, tx *sql.Tx, userID int64, draft models.Draft) (int64, error) {
	const query = `
		INSERT INTO emails (sender_id, sender_email, header, body, is_draft)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id
	`
	var id int64
	err := tx.QueryRowContext(ctx, query, userID, draft.SenderEmail, draft.Header, draft.Body).Scan(&id)
	if err != nil {
		return 0, ErrQueryFail
	}
	return id, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, draft models.Draft) error {}

func (r *Repository) MarkDraftSent(ctx context.Context, tx *sql.Tx, draftID, userID int64) error {}

func (r *Repository) DeleteDraftsBatch(ctx context.Context, userID int64, draftIDs []int64) error {}
