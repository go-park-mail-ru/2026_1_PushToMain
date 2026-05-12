package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) CreateDraft(ctx context.Context, draft models.Draft) (*models.Draft, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrTransactionFailed
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO emails (sender_id, sender_email, header, body, is_draft)
		SELECT $1, u.email, $2, $3, true FROM users u WHERE u.id = $1
		RETURNING id, sender_email, created_at, updated_at
	`, draft.SenderID, draft.Header, draft.Body).Scan(
		&draft.ID, &draft.SenderEmail, &draft.CreatedAt, &draft.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, mapPgError(err)
	}

	if err = insertEmailRecipients(ctx, tx, draft.ID, draft.Recipients); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, ErrTransactionFailed
	}
	return &draft, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, userID int64, draft models.Draft) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrTransactionFailed
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE emails
		SET header = $1, body = $2, updated_at = NOW()
		WHERE id = $3 AND sender_id = $4 AND is_draft = true
	`, draft.Header, draft.Body, draft.ID, userID)
	if err != nil {
		return mapPgError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ErrQueryFail
	}
	if rows == 0 {
		return ErrDraftNotFound
	}

	if _, err = tx.ExecContext(ctx, `
		DELETE FROM email_recipients WHERE email_id = $1
	`, draft.ID); err != nil {
		return ErrQueryFail
	}
	if err = insertEmailRecipients(ctx, tx, draft.ID, draft.Recipients); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	return nil
}

func (r *Repository) GetDraftByID(ctx context.Context, draftID, userID int64) (*models.Draft, error) {
	const query = `
		SELECT
			e.id, e.sender_id, e.sender_email,
			COALESCE(e.header, ''), COALESCE(e.body, ''),
			e.created_at, e.updated_at,
			COALESCE((SELECT string_agg(er.recipient_email, ',') FROM email_recipients er WHERE er.email_id = e.id), '')
		FROM emails e
		WHERE e.id = $1 AND e.sender_id = $2 AND e.is_draft = true
	`
	var d models.Draft
	var recipients string
	err := r.db.QueryRowContext(ctx, query, draftID, userID).Scan(
		&d.ID, &d.SenderID, &d.SenderEmail,
		&d.Header, &d.Body, &d.CreatedAt, &d.UpdatedAt,
		&recipients,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDraftNotFound
	}
	if err != nil {
		return nil, ErrQueryFail
	}
	d.Recipients = parsePgTextArray(recipients)
	return &d, nil
}

func (r *Repository) GetDrafts(ctx context.Context, userID int64, limit, offset int) ([]models.Draft, error) {
	limit, offset = normPage(limit, offset)
	const query = `
		SELECT
			e.id, e.sender_id, e.sender_email,
			COALESCE(e.header, ''), COALESCE(e.body, ''),
			e.created_at, e.updated_at,
			COALESCE((SELECT string_agg(er.recipient_email, ',') FROM email_recipients er WHERE er.email_id = e.id), '')
		FROM emails e
		WHERE e.sender_id = $1 AND e.is_draft = true
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	out := make([]models.Draft, 0)
	for rows.Next() {
		var d models.Draft
		var recipients string
		if err := rows.Scan(
			&d.ID, &d.SenderID, &d.SenderEmail,
			&d.Header, &d.Body, &d.CreatedAt, &d.UpdatedAt,
			&recipients,
		); err != nil {
			return nil, ErrQueryFail
		}
		d.Recipients = parsePgTextArray(recipients)
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return out, nil
}

func (r *Repository) MarkDraftAsSentTx(ctx context.Context, tx *sql.Tx, draftID, userID int64) error {
	const query = `
		UPDATE emails
		SET is_draft = false, updated_at = NOW()
		WHERE id = $1 AND sender_id = $2 AND is_draft = true
	`
	res, err := tx.ExecContext(ctx, query, draftID, userID)
	if err != nil {
		return mapPgError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ErrQueryFail
	}
	if rows == 0 {
		return ErrDraftNotFound
	}
	return nil
}

func (r *Repository) DeleteDraftsBatch(ctx context.Context, userID int64, draftIDs []int64) error {
	if len(draftIDs) == 0 {
		return nil
	}
	parts := make([]string, len(draftIDs))
	args := make([]any, 0, len(draftIDs)+1)
	args = append(args, userID)
	for i, id := range draftIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		DELETE FROM emails
		WHERE sender_id = $1 AND is_draft = true AND id IN (%s)
	`, strings.Join(parts, ","))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}
