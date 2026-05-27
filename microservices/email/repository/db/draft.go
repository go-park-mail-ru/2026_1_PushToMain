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
	defer func() {
		_ = tx.Rollback()
	}()

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
	defer func() {
		_ = tx.Rollback()
	}()

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
	var d models.Draft
	err := r.db.QueryRowContext(ctx, `
		SELECT id, sender_id, sender_email, header, body, created_at, updated_at
		FROM emails
		WHERE id = $1 AND sender_id = $2 AND is_draft = true
	`, draftID, userID).Scan(
		&d.ID,
		&d.SenderID,
		&d.SenderEmail,
		&d.Header,
		&d.Body,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDraftNotFound
	}
	if err != nil {
		return nil, ErrQueryFail
	}

	drafts := []models.Draft{d}
	if err := r.fillRecipients(ctx, drafts, []int64{d.ID}, map[int64]int{d.ID: 0}); err != nil {
		return nil, err
	}

	return &drafts[0], nil
}

func (r *Repository) GetDrafts(ctx context.Context, userID int64, limit, offset int) ([]models.Draft, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, sender_id, sender_email, header, body, created_at, updated_at
		FROM emails
		WHERE sender_id = $1 AND is_draft = true
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer func() {
		_ = rows.Close()
	}()

	var drafts []models.Draft
	var ids []int64
	idxByID := map[int64]int{}

	for rows.Next() {
		var d models.Draft
		if err := rows.Scan(&d.ID, &d.SenderID, &d.SenderEmail, &d.Header, &d.Body, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, ErrQueryFail
		}
		idxByID[d.ID] = len(drafts)
		drafts = append(drafts, d)
		ids = append(ids, d.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	if len(drafts) == 0 {
		return drafts, nil
	}

	if err := r.fillRecipients(ctx, drafts, ids, idxByID); err != nil {
		return nil, err
	}
	return drafts, nil
}

func (r *Repository) fillRecipients(ctx context.Context, drafts []models.Draft, ids []int64, idxByID map[int64]int) error {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(
			`SELECT email_id, recipient_email FROM email_recipients WHERE email_id IN (%s)`,
			strings.Join(placeholders, ","),
		),
		args...,
	)
	if err != nil {
		return ErrQueryFail
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var emailID int64
		var recipient string
		if err := rows.Scan(&emailID, &recipient); err != nil {
			return ErrQueryFail
		}
		idx := idxByID[emailID]
		drafts[idx].Recipients = append(drafts[idx].Recipients, recipient)
	}
	return rows.Err()
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
